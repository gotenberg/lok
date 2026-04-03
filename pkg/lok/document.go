package lok

import (
	"fmt"

	"github.com/gotenberg/lok/pkg/lok/internal/cgo"
)

// Document wraps a LibreOfficeKit document handle for format conversion and
// manipulation. Close the document when no longer needed to release resources.
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

	d.internal.Destroy()
	d.closed = true
}

// SaveAs exports the document to the given path in the specified format.
// Returns [ErrDocumentDestroyed] if the document has been closed, or
// [ErrSaveFailed] if the export fails.
func (d *Document) SaveAs(path, format, filterOptions string) error {
	if d.closed {
		return ErrDocumentDestroyed
	}

	err := d.internal.SaveAs(path, format, filterOptions)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSaveFailed, err)
	}

	return nil
}

// Type returns the [DocumentType] of the loaded document.
func (d *Document) Type() DocumentType {
	return DocumentType(d.internal.GetType())
}

// SetLandscape sets the page orientation to landscape for text and spreadsheet
// documents. For [TextDocument] and [DrawingDocument], it sends a UNO command
// to set A4 landscape dimensions (297x210mm) and the IsLandscape flag. For
// [SpreadsheetDocument], the same approach is used. For [PresentationDocument],
// this is a no-op since slide dimensions are controlled differently.
func (d *Document) SetLandscape(landscape bool) error {
	if d.closed {
		return ErrDocumentDestroyed
	}

	if !landscape {
		return nil
	}

	docType := d.Type()

	// Presentations control slide size differently; landscape is the default.
	if docType == PresentationDocument {
		return nil
	}

	// A4 dimensions in 1/100mm, swapped for landscape orientation.
	// Both IsLandscape and swapped Width/Height are required for Writer.
	args := `{"IsLandscape":{"type":"boolean","value":"true"},"Width":{"type":"long","value":"29700"},"Height":{"type":"long","value":"21000"}}`

	d.internal.PostUnoCommand(".uno:AttributePageSize", args, false)

	return nil
}

// PostUnoCommand sends a UNO command to the document. Returns
// [ErrDocumentDestroyed] if the document has been closed.
func (d *Document) PostUnoCommand(command, arguments string) error {
	if d.closed {
		return ErrDocumentDestroyed
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
// EXPERIMENTAL: this method may change or be removed in future versions.
func (d *Document) ExportPDFViaUnoCommand(outputPath, filterOptions string) error {
	if d.closed {
		return ErrDocumentDestroyed
	}

	// Set the printer to landscape orientation via the printer descriptor.
	// PaperOrientation 1 = landscape in the com.sun.star.view.PaperOrientation enum.
	d.internal.PostUnoCommand(".uno:Printer",
		`{"PaperOrientation":{"type":"long","value":"1"}}`, false)

	// Build the ExportDirectToPDF arguments with the output URL and filter data.
	args := `{"URL":{"type":"string","value":"` + outputPath + `"}`
	if filterOptions != "" {
		args += `,"FilterData":{"type":"string","value":"` + filterOptions + `"}`
	}
	args += `}`

	d.internal.PostUnoCommand(".uno:ExportDirectToPDF", args, false)

	return nil
}

// IsClosed reports whether the document has been destroyed.
func (d *Document) IsClosed() bool {
	return d.closed
}
