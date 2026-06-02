package lok

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/gotenberg/lok/pkg/lok/internal/cgo"
)

// Document wraps a LibreOfficeKit document handle for format conversion and
// manipulation. Close the document when no longer needed to release resources.
//
// A Document is not safe for concurrent use by multiple goroutines. Its CGO
// calls are serialized against the parent [Office] through the same mutex, so a
// Document call and an Office call never run concurrently against the shared
// LibreOfficeKit instance. Serializing calls on a single Document is the
// caller's responsibility.
type Document struct {
	internal *cgo.Document
	office   *Office
	closed   bool
}

// Close destroys the document handle and releases resources.
// Close is idempotent: calling it more than once has no effect.
func (d *Document) Close() {
	if d.closed {
		return
	}

	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	// Re-check under the lock so concurrent Close calls cannot double-free.
	if d.closed {
		return
	}

	// Destroying the office invalidates its documents, so skip the C call when
	// the office is already gone to avoid a use-after-free.
	if !d.office.closed {
		d.internal.Destroy()
	}

	d.closed = true
}

// SaveAs exports the document to the given path in the specified format.
// Returns [ErrDocumentDestroyed] if the document has been closed,
// [ErrOfficeDestroyed] if the office has been closed, or [ErrSaveFailed] if the
// export fails.
func (d *Document) SaveAs(path, format, filterOptions string) error {
	if d.closed {
		return ErrDocumentDestroyed
	}

	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	if d.office.closed {
		return ErrOfficeDestroyed
	}

	err := d.internal.SaveAs(path, format, filterOptions)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSaveFailed, err)
	}

	return nil
}

// Type returns the [DocumentType] of the loaded document. Returns an invalid
// type (one where [DocumentType.IsValid] reports false) if the document or
// office has been closed.
func (d *Document) Type() DocumentType {
	if d.closed {
		return DocumentType(-1)
	}

	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	if d.office.closed {
		return DocumentType(-1)
	}

	return DocumentType(d.internal.GetType())
}

// SetLandscape sets the page orientation to landscape for non-presentation
// documents. It sends .uno:AttributePageSize with A4 landscape dimensions
// (297x210mm) and the IsLandscape flag. [PresentationDocument] controls slide
// size differently, so this is a no-op for it. Returns [ErrDocumentDestroyed]
// or [ErrOfficeDestroyed] if the document or office has been closed.
//
// The dispatch is fire-and-forget: a nil return means the command was posted,
// not that LibreOffice applied it successfully.
func (d *Document) SetLandscape(landscape bool) error {
	if d.closed {
		return ErrDocumentDestroyed
	}

	if !landscape {
		return nil
	}

	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	if d.office.closed {
		return ErrOfficeDestroyed
	}

	// Presentations control slide size differently; landscape is the default.
	if DocumentType(d.internal.GetType()) == PresentationDocument {
		return nil
	}

	// A4 dimensions in 1/100mm, swapped for landscape orientation.
	// Both IsLandscape and swapped Width/Height are required for Writer.
	// TODO: A4 is hardcoded. When the caller sets a non-A4 PaperFormat or a
	// custom PaperSize, this forces A4 landscape and conflicts with the printer
	// descriptor. Derive the dimensions from the requested paper size.
	args := `{"IsLandscape":{"type":"boolean","value":"true"},"Width":{"type":"long","value":"29700"},"Height":{"type":"long","value":"21000"}}`

	d.internal.PostUnoCommand(".uno:AttributePageSize", args, false)

	return nil
}

// PostUnoCommand sends a UNO command to the document. Returns
// [ErrDocumentDestroyed] or [ErrOfficeDestroyed] if the document or office has
// been closed.
//
// The dispatch is fire-and-forget: LibreOfficeKit's postUnoCommand has no
// return value and lok does not wait for the completion callback, so a nil
// return means the command was posted, not that it ran or succeeded.
func (d *Document) PostUnoCommand(command, arguments string) error {
	if d.closed {
		return ErrDocumentDestroyed
	}

	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	if d.office.closed {
		return ErrOfficeDestroyed
	}

	d.internal.PostUnoCommand(command, arguments, false)

	return nil
}

// ExportPDFViaUnoCommand exports the document to PDF using the
// .uno:ExportDirectToPDF dispatch command instead of the saveAs API. This goes
// through the print path, which respects printer descriptor properties such as
// paper orientation. This is useful as a fallback for [SpreadsheetDocument]
// landscape export where saveAs does not honor the page orientation set via
// .uno:AttributePageSize.
//
// The dispatch is fire-and-forget: lok does not wait for the export to finish,
// so a nil return does not guarantee the file was written. Prefer
// [Document.SaveAs] unless the print path is specifically required.
//
// EXPERIMENTAL: this method may change or be removed in future versions.
func (d *Document) ExportPDFViaUnoCommand(outputPath, filterOptions string) error {
	if d.closed {
		return ErrDocumentDestroyed
	}

	args, err := buildExportPDFArgs(outputPath, filterOptions)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSaveFailed, err)
	}

	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	if d.office.closed {
		return ErrOfficeDestroyed
	}

	// Set the printer to landscape orientation via the printer descriptor.
	// PaperOrientation 1 = landscape in the com.sun.star.view.PaperOrientation enum.
	d.internal.PostUnoCommand(".uno:Printer",
		`{"PaperOrientation":{"type":"long","value":1}}`, false)

	d.internal.PostUnoCommand(".uno:ExportDirectToPDF", args, false)

	return nil
}

// buildExportPDFArgs builds the JSON arguments for .uno:ExportDirectToPDF.
// The output path is encoded as an absolute file URL, and filterOptions (a UNO
// property map as produced by [BuildFilterOptions]) is wrapped as a
// com.sun.star.beans.PropertyValue sequence under FilterData. Returns an error
// if the path cannot be resolved or filterOptions is not valid JSON.
func buildExportPDFArgs(outputPath, filterOptions string) (string, error) {
	abs, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("resolving output path: %w", err)
	}

	// .uno:ExportDirectToPDF expects a file URL; unlike saveAs, the dispatch
	// path does not convert a plain filesystem path internally.
	fileURL := (&url.URL{Scheme: "file", Path: abs}).String()

	args := map[string]any{
		"URL": map[string]any{"type": "string", "value": fileURL},
	}

	if filterOptions != "" {
		if !json.Valid([]byte(filterOptions)) {
			return "", fmt.Errorf("filter options is not valid JSON: %q", filterOptions)
		}

		args["FilterData"] = map[string]any{
			"type":  "[]com.sun.star.beans.PropertyValue",
			"value": json.RawMessage(filterOptions),
		}
	}

	data, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("marshaling export arguments: %w", err)
	}

	return string(data), nil
}

// IsClosed reports whether the document has been destroyed.
func (d *Document) IsClosed() bool {
	return d.closed
}
