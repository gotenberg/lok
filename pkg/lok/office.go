package lok

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/gotenberg/lok/pkg/lok/internal/cgo"
	"github.com/gotenberg/lok/pkg/lok/internal/profile"
)

// Office wraps a LibreOfficeKit instance for document conversion.
//
// All CGO calls are serialized with an internal mutex, including calls made
// through [Document] handles obtained from this Office. Individual calls are
// therefore safe across goroutines. A full conversion workflow (load,
// configure, save, close) is not atomic, so the caller must hold its own
// higher-level lock to serialize an entire sequence, for example concurrent
// calls to [Convert] or [Lifecycle.Convert] on the same Office.
type Office struct {
	mu             sync.Mutex
	internal       *cgo.Office
	closed         bool
	managedProfile string
}

// geometryEnv is the environment variable the geometry macro reads.
const geometryEnv = "LOK_GEOM"

// Init loads LibreOffice from the given program directory and returns an
// [Office] ready for document operations. The programPath must point to the
// LibreOffice program directory (e.g., "/usr/lib/libreoffice/program").
//
// Init creates a private user profile and installs the geometry macro into it.
// The profile is removed by [Office.Close].
func Init(programPath string) (*Office, error) {
	if programPath == "" {
		return nil, fmt.Errorf("%w: program path must not be empty", ErrInitFailed)
	}

	profileDir, err := os.MkdirTemp("", "lok-profile-")
	if err != nil {
		return nil, fmt.Errorf("%w: creating profile: %s", ErrInitFailed, err)
	}

	internal, err := initWithProfile(programPath, profileDir)
	if err != nil {
		_ = os.RemoveAll(profileDir)
		return nil, err
	}

	return &Office{internal: internal, managedProfile: profileDir}, nil
}

// InitWithUserProfile loads LibreOffice with a custom user profile directory.
// The profilePath isolates LibreOffice settings and caches per instance.
// The geometry macro is installed into the profile. The profile is not removed
// by [Office.Close].
func InitWithUserProfile(programPath, profilePath string) (*Office, error) {
	if programPath == "" {
		return nil, fmt.Errorf("%w: program path must not be empty", ErrInitFailed)
	}

	if profilePath == "" {
		return nil, fmt.Errorf("%w: profile path must not be empty", ErrInitFailed)
	}

	internal, err := initWithProfile(programPath, profilePath)
	if err != nil {
		return nil, err
	}

	return &Office{internal: internal}, nil
}

// initWithProfile installs the geometry macro into profileDir and initializes
// LibreOffice against it.
func initWithProfile(programPath, profileDir string) (*cgo.Office, error) {
	if err := profile.Install(programPath, profileDir); err != nil {
		return nil, fmt.Errorf("%w: installing macro: %s", ErrInitFailed, err)
	}

	internal, err := cgo.InitWithUserProfile(programPath, fileURL(profileDir))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInitFailed, err)
	}

	return internal, nil
}

// fileURL converts a filesystem path to a file URL for LibreOfficeKit.
func fileURL(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	return (&url.URL{Scheme: "file", Path: abs}).String()
}

// Close destroys the LibreOfficeKit instance and releases resources.
// Close is idempotent: calling it more than once has no effect.
//
// Calling Close is optional and often unnecessary. LibreOffice's destroy() has
// been observed to crash the Go runtime during process shutdown because it
// installs signal handlers without SA_ONSTACK. For a short-lived process,
// prefer letting process exit reclaim resources. If Close is required, run with
// GODEBUG=asyncpreemptoff=1.
func (o *Office) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return
	}

	o.internal.Destroy()
	o.closed = true

	if o.managedProfile != "" {
		_ = os.RemoveAll(o.managedProfile)
	}
}

// applyGeometry runs the geometry macro to set page orientation and size on the
// currently loaded document. landscape sets the orientation. width and height
// are in 1/100 mm; 0 keeps the document's current size. A page-geometry change
// is not reachable through a UNO dispatch in headless LibreOfficeKit, so a Basic
// macro applies it through the UNO page styles.
func (o *Office) applyGeometry(landscape bool, width, height int) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return ErrOfficeDestroyed
	}

	land := 0
	if landscape {
		land = 1
	}

	// The macro reads its parameters from the environment. Set them only for
	// the duration of the call; this is safe because Office calls are
	// serialized and the macro runs synchronously.
	prev, had := os.LookupEnv(geometryEnv)
	_ = os.Setenv(geometryEnv, fmt.Sprintf("%d,%d,%d", land, width, height))

	err := o.internal.RunMacro(profile.MacroURL)

	if had {
		_ = os.Setenv(geometryEnv, prev)
	} else {
		_ = os.Unsetenv(geometryEnv)
	}

	if err != nil {
		return fmt.Errorf("%w: applying geometry: %s", ErrSaveFailed, err)
	}

	return nil
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
// closed or trimMemory is unavailable. Use [Office.HasTrimMemory] to detect
// availability.
func (o *Office) TrimMemory(target int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return
	}

	o.internal.TrimMemory(target)
}

// HasTrimMemory reports whether the loaded LibreOffice supports the trimMemory
// API (LibreOffice 7.6+). When it returns false, [Office.TrimMemory] and the
// [Lifecycle] trim schedule are no-ops, and memory is only reclaimed on process
// exit. Returns false if the office is closed.
func (o *Office) HasTrimMemory() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return false
	}

	return o.internal.HasTrimMemory()
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
