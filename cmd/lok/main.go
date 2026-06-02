// lok converts documents to PDF using LibreOfficeKit.
//
// Single-shot mode (default):
//
//	lok --input-path input.docx --output-path output.pdf
//
// Long-running mode:
//
//	lok --long-running --libreoffice-program-path /usr/lib/libreoffice/program
//
// In long-running mode, lok reads JSON requests from stdin (one per line)
// and writes JSON responses to stdout, reusing a single LibreOffice instance.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/gotenberg/lok/pkg/lok"
)

const (
	exitSuccess = 0
	exitError   = 1
)

func main() {
	ensureAsyncPreemptOff()

	opts := defineFlags()
	flag.Parse()

	opts.lokOpts.PaperFormat = lok.PaperFormat(opts.paperFormat)

	if opts.longRunning {
		os.Exit(runLongRunning(opts))
	}

	os.Exit(runOnce(opts))
}

// ensureAsyncPreemptOff re-executes the process with GODEBUG=asyncpreemptoff=1
// when it is not already set. LibreOffice installs signal handlers without
// SA_ONSTACK, and Go's async preemption (SIGURG) can then crash the runtime.
// GODEBUG must be set before the runtime starts, so the only reliable fix from
// within the binary is to re-exec. This must run before lok.Init.
func ensureAsyncPreemptOff() {
	const want = "asyncpreemptoff=1"

	for _, entry := range strings.Split(os.Getenv("GODEBUG"), ",") {
		if entry == want {
			return
		}
	}

	exe, err := os.Executable()
	if err != nil {
		// Best effort: warn and continue rather than abort.
		fmt.Fprintf(os.Stderr, "warning: could not enable %s: %v\n", want, err)
		return
	}

	godebug := want
	if existing := os.Getenv("GODEBUG"); existing != "" {
		godebug = existing + "," + want
	}

	// Replace any existing GODEBUG entry rather than appending a duplicate,
	// otherwise the new value may be ignored and the process re-execs forever.
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GODEBUG=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "GODEBUG="+godebug)

	// Re-exec this same binary with its own arguments to set GODEBUG before the
	// runtime starts. The executable and arguments are this process's own, not
	// external input.
	if err := syscall.Exec(exe, os.Args, env); err != nil { //nolint:gosec // re-exec of self with own args
		fmt.Fprintf(os.Stderr, "warning: could not enable %s: %v\n", want, err)
	}
}

// cliOptions holds all parsed flag values.
type cliOptions struct {
	// Required paths.
	inputPath   string
	outputPath  string
	programPath string

	// Long-running mode.
	longRunning bool

	// Raw paper format flag (-1 = unset).
	paperFormat int

	// Build lok.Options from these.
	lokOpts lok.Options
}

func defineFlags() *cliOptions {
	opts := &cliOptions{}

	// Required.
	flag.StringVar(&opts.inputPath, "input-path", "", "Path to the input document")
	flag.StringVar(&opts.outputPath, "output-path", "", "Path for the output PDF")
	flag.StringVar(&opts.programPath, "libreoffice-program-path", "/usr/lib/libreoffice/program", "LibreOffice program directory")

	// Mode.
	flag.BoolVar(&opts.longRunning, "long-running", false, "Read JSON requests from stdin, write responses to stdout")

	// Defaults from lok.DefaultOptions().
	defaults := lok.DefaultOptions()
	o := &opts.lokOpts

	// Document load options.
	flag.StringVar(&o.Password, "password", "", "Document open password")
	flag.IntVar(&o.MacroExecutionMode, "macro-execution-mode", 0, "Macro execution mode (0=never, 7=always)")

	// Printer descriptor options.
	flag.BoolVar(&o.Landscape, "landscape", false, "Set landscape orientation")
	flag.IntVar(&opts.paperFormat, "paper-format", -1, "Paper format (0=A3, 1=A4, 2=A5, 3=B4, 4=B5, 5=Letter, 6=Legal, 7=Tabloid, 8=User)")
	flag.IntVar(&o.PaperWidth, "paper-width", 0, "Custom paper width in 1/100mm (requires --paper-format 8)")
	flag.IntVar(&o.PaperHeight, "paper-height", 0, "Custom paper height in 1/100mm (requires --paper-format 8)")

	// Document commands.
	flag.BoolVar(&o.UpdateIndexes, "update-indexes", false, "Rebuild TOC and indexes before export")

	// Page content.
	flag.StringVar(&o.PageRanges, "page-ranges", defaults.PageRanges, "Page ranges to export (e.g., \"1-3,5\")")
	flag.BoolVar(&o.SkipEmptyPages, "skip-empty-pages", defaults.SkipEmptyPages, "Omit empty pages")
	flag.BoolVar(&o.SinglePageSheets, "single-page-sheets", defaults.SinglePageSheets, "Print each sheet on a single page")

	// Image handling.
	flag.BoolVar(&o.LosslessImageCompression, "lossless-image-compression", defaults.LosslessImageCompression, "Use lossless image compression")
	flag.IntVar(&o.Quality, "quality", defaults.Quality, "JPEG compression quality (1-100)")
	flag.BoolVar(&o.ReduceImageResolution, "reduce-image-resolution", defaults.ReduceImageResolution, "Downscale images")
	flag.IntVar(&o.MaxImageResolution, "max-image-resolution", defaults.MaxImageResolution, "Target DPI for image downscaling")

	// Form fields.
	flag.BoolVar(&o.ExportFormFields, "export-form-fields", defaults.ExportFormFields, "Preserve PDF form fields")
	flag.BoolVar(&o.AllowDuplicateFieldNames, "allow-duplicate-field-names", defaults.AllowDuplicateFieldNames, "Allow duplicate form field names")

	// Bookmarks and links.
	flag.BoolVar(&o.ExportBookmarks, "export-bookmarks", defaults.ExportBookmarks, "Include bookmarks")
	flag.BoolVar(&o.ExportBookmarksToPdfDestination, "export-bookmarks-to-pdf-destination", defaults.ExportBookmarksToPdfDestination, "Export bookmarks as named destinations")
	flag.BoolVar(&o.ConvertOooTargetToPdfTarget, "convert-ooo-target-to-pdf-target", defaults.ConvertOooTargetToPdfTarget, "Convert internal links to PDF targets")
	flag.BoolVar(&o.ExportLinksRelativeFsys, "export-links-relative-fsys", defaults.ExportLinksRelativeFsys, "Export filesystem links as relative")

	// Notes and annotations.
	flag.BoolVar(&o.ExportNotes, "export-notes", defaults.ExportNotes, "Include notes")
	flag.BoolVar(&o.ExportNotesPages, "export-notes-pages", defaults.ExportNotesPages, "Export notes pages")
	flag.BoolVar(&o.ExportOnlyNotesPages, "export-only-notes-pages", defaults.ExportOnlyNotesPages, "Export only notes pages")
	flag.BoolVar(&o.ExportNotesInMargin, "export-notes-in-margin", defaults.ExportNotesInMargin, "Place notes in margin")

	// Presentation-specific.
	flag.BoolVar(&o.ExportHiddenSlides, "export-hidden-slides", defaults.ExportHiddenSlides, "Include hidden slides")
	flag.BoolVar(&o.UseTransitionEffects, "use-transition-effects", defaults.UseTransitionEffects, "Enable slide transitions")

	// Placeholders and streams.
	flag.BoolVar(&o.ExportPlaceholders, "export-placeholders", defaults.ExportPlaceholders, "Export placeholder fields")
	flag.BoolVar(&o.AddOriginalDocumentAsStream, "add-original-document-as-stream", defaults.AddOriginalDocumentAsStream, "Embed source document")

	// PDF standards.
	flag.IntVar(&o.PDFVersion, "pdf-version", defaults.PDFVersion, "PDF version (0=standard, 1=PDF/A-1b, 2=PDF/A-2b, 3=PDF/A-3b)")
	flag.BoolVar(&o.PDFUniversalAccess, "pdf-universal-access", defaults.PDFUniversalAccess, "Enable PDF/UA compliance")

	// Watermark.
	flag.StringVar(&o.NativeWatermarkText, "native-watermark-text", defaults.NativeWatermarkText, "Tiled watermark text")

	// Viewer preferences.
	flag.IntVar(&o.InitialView, "initial-view", defaults.InitialView, "Initial view mode (0=default, 1=bookmarks, 2=thumbnails)")
	flag.IntVar(&o.InitialPage, "initial-page", defaults.InitialPage, "Page displayed on open")
	flag.IntVar(&o.Magnification, "magnification", defaults.Magnification, "Default magnification mode")
	flag.IntVar(&o.Zoom, "zoom", defaults.Zoom, "Default zoom percentage")
	flag.IntVar(&o.PageLayout, "page-layout", defaults.PageLayout, "Page layout (0=default, 1=single, 2=continuous)")
	flag.BoolVar(&o.FirstPageOnLeft, "first-page-on-left", defaults.FirstPageOnLeft, "First page on left in two-page layout")
	flag.BoolVar(&o.ResizeWindowToInitialPage, "resize-window-to-initial-page", defaults.ResizeWindowToInitialPage, "Resize viewer to first page")
	flag.BoolVar(&o.CenterWindow, "center-window", defaults.CenterWindow, "Center viewer window")
	flag.BoolVar(&o.OpenInFullScreenMode, "open-in-full-screen-mode", defaults.OpenInFullScreenMode, "Open in full-screen mode")
	flag.BoolVar(&o.DisplayPDFDocumentTitle, "display-pdf-document-title", defaults.DisplayPDFDocumentTitle, "Show document title in title bar")
	flag.BoolVar(&o.HideViewerMenubar, "hide-viewer-menubar", defaults.HideViewerMenubar, "Hide menu bar")
	flag.BoolVar(&o.HideViewerToolbar, "hide-viewer-toolbar", defaults.HideViewerToolbar, "Hide toolbar")
	flag.BoolVar(&o.HideViewerWindowControls, "hide-viewer-window-controls", defaults.HideViewerWindowControls, "Hide window controls")
	flag.IntVar(&o.OpenBookmarkLevels, "open-bookmark-levels", defaults.OpenBookmarkLevels, "Bookmark levels shown (-1=all)")

	return opts
}

func runOnce(opts *cliOptions) int {
	if opts.inputPath == "" || opts.outputPath == "" {
		fmt.Fprintln(os.Stderr, "error: --input-path and --output-path are required")
		flag.Usage()
		return exitError
	}

	office, err := lok.Init(opts.programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitError
	}

	// Skip office.Close() to avoid LibreOffice signal handler conflict with
	// Go's runtime on process exit.

	err = lok.Convert(office, opts.inputPath, opts.outputPath, opts.lokOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitError
	}

	office.TrimMemory(0)

	return exitSuccess
}

// longRunningRequest is the JSON schema for stdin requests in long-running mode.
type longRunningRequest struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`

	// All Options fields are optional overrides.
	Password                        *string `json:"password,omitempty"` //nolint:gosec // document open password, not a credential
	MacroExecutionMode              *int    `json:"macroExecutionMode,omitempty"`
	Landscape                       *bool   `json:"landscape,omitempty"`
	PaperFormat                     *int    `json:"paperFormat,omitempty"`
	PaperWidth                      *int    `json:"paperWidth,omitempty"`
	PaperHeight                     *int    `json:"paperHeight,omitempty"`
	UpdateIndexes                   *bool   `json:"updateIndexes,omitempty"`
	PageRanges                      *string `json:"pageRanges,omitempty"`
	Quality                         *int    `json:"quality,omitempty"`
	LosslessImageCompression        *bool   `json:"losslessImageCompression,omitempty"`
	ReduceImageResolution           *bool   `json:"reduceImageResolution,omitempty"`
	MaxImageResolution              *int    `json:"maxImageResolution,omitempty"`
	ExportFormFields                *bool   `json:"exportFormFields,omitempty"`
	AllowDuplicateFieldNames        *bool   `json:"allowDuplicateFieldNames,omitempty"`
	ExportBookmarks                 *bool   `json:"exportBookmarks,omitempty"`
	ExportBookmarksToPdfDestination *bool   `json:"exportBookmarksToPdfDestination,omitempty"`
	ExportPlaceholders              *bool   `json:"exportPlaceholders,omitempty"`
	ExportNotes                     *bool   `json:"exportNotes,omitempty"`
	ExportNotesPages                *bool   `json:"exportNotesPages,omitempty"`
	ExportOnlyNotesPages            *bool   `json:"exportOnlyNotesPages,omitempty"`
	ExportNotesInMargin             *bool   `json:"exportNotesInMargin,omitempty"`
	ConvertOooTargetToPdfTarget     *bool   `json:"convertOooTargetToPdfTarget,omitempty"`
	ExportLinksRelativeFsys         *bool   `json:"exportLinksRelativeFsys,omitempty"`
	ExportHiddenSlides              *bool   `json:"exportHiddenSlides,omitempty"`
	SkipEmptyPages                  *bool   `json:"skipEmptyPages,omitempty"`
	AddOriginalDocumentAsStream     *bool   `json:"addOriginalDocumentAsStream,omitempty"`
	SinglePageSheets                *bool   `json:"singlePageSheets,omitempty"`
	PDFVersion                      *int    `json:"pdfVersion,omitempty"`
	PDFUniversalAccess              *bool   `json:"pdfUniversalAccess,omitempty"`
	NativeWatermarkText             *string `json:"nativeWatermarkText,omitempty"`
	UseTransitionEffects            *bool   `json:"useTransitionEffects,omitempty"`
	OpenBookmarkLevels              *int    `json:"openBookmarkLevels,omitempty"`
	InitialView                     *int    `json:"initialView,omitempty"`
	InitialPage                     *int    `json:"initialPage,omitempty"`
	Magnification                   *int    `json:"magnification,omitempty"`
	Zoom                            *int    `json:"zoom,omitempty"`
	PageLayout                      *int    `json:"pageLayout,omitempty"`
	FirstPageOnLeft                 *bool   `json:"firstPageOnLeft,omitempty"`
	ResizeWindowToInitialPage       *bool   `json:"resizeWindowToInitialPage,omitempty"`
	CenterWindow                    *bool   `json:"centerWindow,omitempty"`
	OpenInFullScreenMode            *bool   `json:"openInFullScreenMode,omitempty"`
	DisplayPDFDocumentTitle         *bool   `json:"displayPDFDocumentTitle,omitempty"`
	HideViewerMenubar               *bool   `json:"hideViewerMenubar,omitempty"`
	HideViewerToolbar               *bool   `json:"hideViewerToolbar,omitempty"`
	HideViewerWindowControls        *bool   `json:"hideViewerWindowControls,omitempty"`
}

// longRunningResponse is the JSON schema for stdout responses.
type longRunningResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func runLongRunning(opts *cliOptions) int {
	office, err := lok.Init(opts.programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitError
	}

	enc := json.NewEncoder(os.Stdout)

	// Read line-delimited requests with a growable reader rather than a
	// bufio.Scanner, so an oversized request cannot terminate the server.
	reader := bufio.NewReaderSize(os.Stdin, 1024*1024)

	for {
		line, readErr := reader.ReadBytes('\n')

		if len(bytes.TrimSpace(line)) > 0 {
			// An Encode error means stdout is broken; nothing actionable, so
			// drop it and keep serving requests.
			_ = enc.Encode(processRequest(office, line))
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return exitSuccess
			}

			fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", readErr)
			return exitError
		}
	}
}

// processRequest converts a single long-running request. It recovers from Go
// panics so a single malformed or failing request cannot bring down the server.
// A native LibreOffice crash (a segfault inside CGO) still terminates the
// process and cannot be recovered here.
func processRequest(office *lok.Office, line []byte) (resp longRunningResponse) {
	defer func() {
		if r := recover(); r != nil {
			resp = longRunningResponse{Error: fmt.Sprintf("panic during conversion: %v", r)}
		}
	}()

	var req longRunningRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return longRunningResponse{Error: fmt.Sprintf("invalid JSON: %v", err)}
	}

	if req.InputPath == "" || req.OutputPath == "" {
		return longRunningResponse{Error: "inputPath and outputPath are required"}
	}

	err := lok.Convert(office, req.InputPath, req.OutputPath, buildOptsFromRequest(req))

	// Trim caches after each conversion attempt, matching single-shot mode.
	office.TrimMemory(0)

	if err != nil {
		return longRunningResponse{Error: err.Error()}
	}

	return longRunningResponse{Success: true}
}

func buildOptsFromRequest(req longRunningRequest) lok.Options {
	opts := lok.DefaultOptions()

	// Helper to apply optional overrides.
	setBool := func(dst *bool, src *bool) {
		if src != nil {
			*dst = *src
		}
	}
	setInt := func(dst *int, src *int) {
		if src != nil {
			*dst = *src
		}
	}
	setString := func(dst *string, src *string) {
		if src != nil {
			*dst = *src
		}
	}

	// Load options.
	setString(&opts.Password, req.Password)
	setInt(&opts.MacroExecutionMode, req.MacroExecutionMode)

	// Printer descriptor.
	setBool(&opts.Landscape, req.Landscape)
	if req.PaperFormat != nil {
		opts.PaperFormat = lok.PaperFormat(*req.PaperFormat)
	}
	setInt(&opts.PaperWidth, req.PaperWidth)
	setInt(&opts.PaperHeight, req.PaperHeight)

	// Document commands.
	setBool(&opts.UpdateIndexes, req.UpdateIndexes)

	// Filter options.
	setString(&opts.PageRanges, req.PageRanges)
	setInt(&opts.Quality, req.Quality)
	setBool(&opts.LosslessImageCompression, req.LosslessImageCompression)
	setBool(&opts.ReduceImageResolution, req.ReduceImageResolution)
	setInt(&opts.MaxImageResolution, req.MaxImageResolution)
	setBool(&opts.ExportFormFields, req.ExportFormFields)
	setBool(&opts.AllowDuplicateFieldNames, req.AllowDuplicateFieldNames)
	setBool(&opts.ExportBookmarks, req.ExportBookmarks)
	setBool(&opts.ExportBookmarksToPdfDestination, req.ExportBookmarksToPdfDestination)
	setBool(&opts.ExportPlaceholders, req.ExportPlaceholders)
	setBool(&opts.ExportNotes, req.ExportNotes)
	setBool(&opts.ExportNotesPages, req.ExportNotesPages)
	setBool(&opts.ExportOnlyNotesPages, req.ExportOnlyNotesPages)
	setBool(&opts.ExportNotesInMargin, req.ExportNotesInMargin)
	setBool(&opts.ConvertOooTargetToPdfTarget, req.ConvertOooTargetToPdfTarget)
	setBool(&opts.ExportLinksRelativeFsys, req.ExportLinksRelativeFsys)
	setBool(&opts.ExportHiddenSlides, req.ExportHiddenSlides)
	setBool(&opts.SkipEmptyPages, req.SkipEmptyPages)
	setBool(&opts.AddOriginalDocumentAsStream, req.AddOriginalDocumentAsStream)
	setBool(&opts.SinglePageSheets, req.SinglePageSheets)
	setInt(&opts.PDFVersion, req.PDFVersion)
	setBool(&opts.PDFUniversalAccess, req.PDFUniversalAccess)
	setString(&opts.NativeWatermarkText, req.NativeWatermarkText)
	setBool(&opts.UseTransitionEffects, req.UseTransitionEffects)
	setInt(&opts.OpenBookmarkLevels, req.OpenBookmarkLevels)
	setInt(&opts.InitialView, req.InitialView)
	setInt(&opts.InitialPage, req.InitialPage)
	setInt(&opts.Magnification, req.Magnification)
	setInt(&opts.Zoom, req.Zoom)
	setInt(&opts.PageLayout, req.PageLayout)
	setBool(&opts.FirstPageOnLeft, req.FirstPageOnLeft)
	setBool(&opts.ResizeWindowToInitialPage, req.ResizeWindowToInitialPage)
	setBool(&opts.CenterWindow, req.CenterWindow)
	setBool(&opts.OpenInFullScreenMode, req.OpenInFullScreenMode)
	setBool(&opts.DisplayPDFDocumentTitle, req.DisplayPDFDocumentTitle)
	setBool(&opts.HideViewerMenubar, req.HideViewerMenubar)
	setBool(&opts.HideViewerToolbar, req.HideViewerToolbar)
	setBool(&opts.HideViewerWindowControls, req.HideViewerWindowControls)

	return opts
}
