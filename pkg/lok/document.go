package lok

import (
	"fmt"

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

// IsClosed reports whether the document has been destroyed.
func (d *Document) IsClosed() bool {
	return d.closed
}
