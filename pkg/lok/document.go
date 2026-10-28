package lok

import (
	"fmt"

	"github.com/gotenberg/lok/pkg/lok/internal/cgo"
)

// Document wraps a LibreOfficeKit document handle for format conversion and
// manipulation. Close the document when no longer needed to release resources.
//
// Every Document call runs under the parent [Office] mutex, so Document and
// Office calls never run concurrently against the shared LibreOfficeKit
// instance, and a concurrent [Document.Close] makes other calls return
// [ErrDocumentDestroyed] rather than use a freed handle. A sequence of calls
// (load, save, close) is not atomic: serializing it is the caller's
// responsibility.
type Document struct {
	internal *cgo.Document
	office   *Office
	closed   bool
}

// Close destroys the document handle and releases resources.
// Close is idempotent: calling it more than once has no effect.
func (d *Document) Close() {
	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	// closed is read under the lock so a Close racing another call can never
	// free the handle that call is about to use.
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
	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	if d.closed {
		return ErrDocumentDestroyed
	}

	if d.office.closed {
		return ErrOfficeDestroyed
	}

	err := d.internal.SaveAs(path, format, filterOptions)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveFailed, err)
	}

	return nil
}

// Type returns the [DocumentType] of the loaded document. Returns an invalid
// type (one where [DocumentType.IsValid] reports false) if the document or
// office has been closed.
func (d *Document) Type() DocumentType {
	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	if d.closed {
		return DocumentType(-1)
	}

	if d.office.closed {
		return DocumentType(-1)
	}

	return DocumentType(d.internal.GetType())
}

// IsClosed reports whether the document has been destroyed.
func (d *Document) IsClosed() bool {
	d.office.mu.Lock()
	defer d.office.mu.Unlock()

	return d.closed
}
