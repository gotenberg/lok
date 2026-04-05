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
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/gotenberg/lok/pkg/lok"
)

const (
	exitSuccess = 0
	exitError   = 1
)

func main() {
	opts := defineFlags()
	flag.Parse()

	opts.lokOpts.PaperFormat = lok.PaperFormat(opts.paperFormat)

	if opts.longRunning {
		os.Exit(runLongRunning(opts))
	}

	os.Exit(runOnce(opts))
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
	flag.IntVar(&o.PDFVersion, "pdf-version", defaults.PDFVersion, "PDF version (0=1.7, 1=A-1b, 2=A-2b, 3=A-3b)")
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
	Password                        *string `json:"password,omitempty"`
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
	scanner := bufio.NewScanner(os.Stdin)

	// 10 MiB max line size for large JSON requests.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req longRunningRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(longRunningResponse{Error: fmt.Sprintf("invalid JSON: %v", err)})
			continue
		}

		if req.InputPath == "" || req.OutputPath == "" {
			_ = enc.Encode(longRunningResponse{Error: "inputPath and outputPath are required"})
			continue
		}

		lokOpts := buildOptsFromRequest(req)

		if err := lok.Convert(office, req.InputPath, req.OutputPath, lokOpts); err != nil {
			_ = enc.Encode(longRunningResponse{Error: err.Error()})
		} else {
			_ = enc.Encode(longRunningResponse{Success: true})
		}

		office.TrimMemory(0)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		return exitError
	}

	return exitSuccess
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
