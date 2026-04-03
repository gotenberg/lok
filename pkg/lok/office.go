package lok

import (
	"fmt"
	"sync"

	"github.com/gotenberg/lok/pkg/lok/internal/cgo"
)

// Office wraps a LibreOfficeKit instance for document conversion.
//
// All CGO calls are serialized with an internal mutex. For a full conversion
// workflow (load, configure, save, close), the caller must hold its own
// higher-level lock to ensure atomicity of the entire sequence.
type Office struct {
	mu       sync.Mutex
	internal *cgo.Office
	closed   bool
}

// Init loads LibreOffice from the given program directory and returns an
// [Office] ready for document operations. The programPath must point to the
// LibreOffice program directory (e.g., "/usr/lib/libreoffice/program").
func Init(programPath string) (*Office, error) {
	if programPath == "" {
		return nil, fmt.Errorf("%w: program path must not be empty", ErrInitFailed)
	}

	internal, err := cgo.Init(programPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInitFailed, err)
	}

	return &Office{internal: internal}, nil
}

// InitWithUserProfile loads LibreOffice with a custom user profile directory.
// The profilePath isolates LibreOffice settings and caches per instance.
func InitWithUserProfile(programPath, profilePath string) (*Office, error) {
	if programPath == "" {
		return nil, fmt.Errorf("%w: program path must not be empty", ErrInitFailed)
	}

	internal, err := cgo.InitWithUserProfile(programPath, profilePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInitFailed, err)
	}

	return &Office{internal: internal}, nil
}

// Close destroys the LibreOfficeKit instance and releases resources.
// Close is idempotent: calling it more than once has no effect.
func (o *Office) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return
	}

	o.internal.Destroy()
	o.closed = true
}

// LoadDocument opens a document at the given file path. The returned
// [Document] must be closed by the caller when no longer needed.
func (o *Office) LoadDocument(path string) (*Document, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return nil, ErrOfficeDestroyed
	}

	doc, err := o.internal.LoadDocument(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrLoadFailed, err)
	}

	return &Document{internal: doc, office: o}, nil
}

// LoadDocumentWithOptions opens a document with additional load options.
// The returned [Document] must be closed by the caller when no longer needed.
func (o *Office) LoadDocumentWithOptions(path, options string) (*Document, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return nil, ErrOfficeDestroyed
	}

	doc, err := o.internal.LoadDocumentWithOptions(path, options)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrLoadFailed, err)
	}

	return &Document{internal: doc, office: o}, nil
}

// TrimMemory asks LibreOffice to release cached memory. The target parameter
// controls aggressiveness: 0 for gentle (per-document caches), 2000 for
// aggressive (join threads, release VCL caches). No-op if the office is
// closed or trimMemory is unavailable.
func (o *Office) TrimMemory(target int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return
	}

	o.internal.TrimMemory(target)
}

// GetVersionInfo returns LibreOffice version information as a JSON string.
func (o *Office) GetVersionInfo() (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return "", ErrOfficeDestroyed
	}

	return o.internal.GetVersionInfo(), nil
}

// GetFilterTypes returns the available document filter types as a JSON string.
func (o *Office) GetFilterTypes() (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return "", ErrOfficeDestroyed
	}

	return o.internal.GetFilterTypes(), nil
}

// GetError retrieves the last error message from LibreOffice. Returns an
// empty string if the office is closed or no error is available.
func (o *Office) GetError() string {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return ""
	}

	return o.internal.GetError()
}

// IsClosed reports whether the office instance has been destroyed.
func (o *Office) IsClosed() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.closed
}
