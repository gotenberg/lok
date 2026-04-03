package lok

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
)

// Lifecycle manages a LibreOfficeKit [Office] instance with automatic memory
// trimming after conversions. LibreOffice leaks approximately 0.5 MiB per
// conversion; [Lifecycle] mitigates this by calling [Office.TrimMemory] on a
// configurable schedule.
//
// TrimMemory target values:
//   - 0: gentle trim, releases per-document caches.
//   - 2000: aggressive trim, joins threads and releases VCL caches.
//
// trimMemory is available since LibreOffice 7.6. On older versions it is a
// no-op.
type Lifecycle struct {
	office          *Office
	programPath     string
	profilePath     string
	conversionCount atomic.Int64
	trimInterval    int
	gentleTrimEvery bool
}

// LifecycleConfig configures a [Lifecycle] instance.
type LifecycleConfig struct {
	// ProgramPath is the LibreOffice program directory
	// (e.g., "/usr/lib/libreoffice/program"). Required.
	ProgramPath string

	// ProfilePath is an optional user profile directory. When set,
	// LibreOffice settings and caches are isolated to this directory.
	ProfilePath string

	// TrimInterval controls how often an aggressive trim (target 2000) is
	// performed, expressed as a number of conversions. Defaults to 10.
	TrimInterval int

	// GentleTrimEvery enables a gentle trim (target 0) after every conversion.
	// Defaults to true.
	GentleTrimEvery *bool
}

// NewLifecycle initializes a LibreOfficeKit [Office] and returns a [Lifecycle]
// that manages its memory. Close the Lifecycle when done.
func NewLifecycle(cfg LifecycleConfig) (*Lifecycle, error) {
	if cfg.ProgramPath == "" {
		return nil, fmt.Errorf("%w: program path must not be empty", ErrInitFailed)
	}

	trimInterval := cfg.TrimInterval
	if trimInterval <= 0 {
		trimInterval = 10
	}

	gentleTrimEvery := true
	if cfg.GentleTrimEvery != nil {
		gentleTrimEvery = *cfg.GentleTrimEvery
	}

	var office *Office
	var err error

	if cfg.ProfilePath != "" {
		office, err = InitWithUserProfile(cfg.ProgramPath, cfg.ProfilePath)
	} else {
		office, err = Init(cfg.ProgramPath)
	}

	if err != nil {
		return nil, err
	}

	return &Lifecycle{
		office:          office,
		programPath:     cfg.ProgramPath,
		profilePath:     cfg.ProfilePath,
		trimInterval:    trimInterval,
		gentleTrimEvery: gentleTrimEvery,
	}, nil
}

// Office returns the underlying [Office] instance.
func (lc *Lifecycle) Office() *Office {
	return lc.office
}

// Convert runs a document conversion and performs memory trimming afterward.
// See [Convert] for the conversion pipeline details.
func (lc *Lifecycle) Convert(inputPath, outputPath string, opts Options) error {
	err := Convert(lc.office, inputPath, outputPath, opts)
	if err != nil {
		return err
	}

	lc.afterConversion()

	return nil
}

// afterConversion increments the conversion counter and performs memory
// trimming based on the configured schedule.
func (lc *Lifecycle) afterConversion() {
	count := lc.conversionCount.Add(1)

	if lc.gentleTrimEvery {
		lc.office.TrimMemory(0)
	}

	if count%int64(lc.trimInterval) == 0 {
		lc.office.TrimMemory(2000)
	}
}

// ConversionCount returns the total number of successful conversions.
func (lc *Lifecycle) ConversionCount() int64 {
	return lc.conversionCount.Load()
}

// Close destroys the underlying [Office] instance and releases resources.
func (lc *Lifecycle) Close() {
	lc.office.Close()
}

// RSS returns the resident set size of the current process in bytes.
// On Linux, it reads /proc/self/statm. On other platforms it returns 0 with
// no error.
func (lc *Lifecycle) RSS() (int64, error) {
	if runtime.GOOS != "linux" {
		return 0, nil
	}

	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, fmt.Errorf("reading /proc/self/statm: %w", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, fmt.Errorf("unexpected /proc/self/statm format: %s", string(data))
	}

	// Second field is RSS in pages.
	pages, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing RSS pages: %w", err)
	}

	pageSize := int64(os.Getpagesize())

	return pages * pageSize, nil
}
