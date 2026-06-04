package lok

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PaperFormat represents a paper size from the com.sun.star.view.PaperFormat
// enum.
type PaperFormat int

const (
	PaperFormatA3      PaperFormat = 0
	PaperFormatA4      PaperFormat = 1
	PaperFormatA5      PaperFormat = 2
	PaperFormatB4      PaperFormat = 3
	PaperFormatB5      PaperFormat = 4
	PaperFormatLetter  PaperFormat = 5
	PaperFormatLegal   PaperFormat = 6
	PaperFormatTabloid PaperFormat = 7
	PaperFormatUser    PaperFormat = 8
)

// Options configures PDF export behavior for LibreOffice document conversion.
//
// Most fields map directly to UNO filter properties passed to saveAs via
// [BuildFilterOptions]. Several fields are handled separately by the
// conversion layer:
//   - Landscape, PaperFormat, PaperWidth, PaperHeight: applied to the page
//     styles by a Basic macro before export. See [Convert].
//   - Password, MacroExecutionMode: passed as document load options.
//   - UpdateIndexes: applied via .uno:UpdateAllIndexes before export.
type Options struct {
	// Landscape sets the page orientation to landscape before export.
	// Not a filter option.
	Landscape bool

	// PageRanges limits export to the specified pages (e.g., "1-3,5").
	// Maps to UNO property "PageRange".
	PageRanges string

	// Password is used to open password-protected documents.
	// Not a filter option.
	Password string //nolint:gosec // document open password, not a credential

	// UpdateIndexes refreshes table of contents and indexes before export.
	// Not a filter option.
	UpdateIndexes bool

	// MacroExecutionMode controls macro execution when loading the document.
	// 0 = never (default), 7 = always. Passed as a document load option.
	// Not a filter option.
	MacroExecutionMode int

	// PaperFormat sets the paper format applied to the page styles before
	// export. Use [PaperFormatUser] with PaperWidth/PaperHeight for custom
	// sizes. -1 means unset (default). Not a filter option.
	PaperFormat PaperFormat

	// PaperWidth sets the custom paper width in 1/100mm. Only used when
	// PaperFormat is [PaperFormatUser]. Not a filter option.
	PaperWidth int

	// PaperHeight sets the custom paper height in 1/100mm. Only used when
	// PaperFormat is [PaperFormatUser]. Not a filter option.
	PaperHeight int

	// ExportFormFields preserves PDF form fields in the output.
	// Defaults to false, matching LibreOffice.
	ExportFormFields bool

	// AllowDuplicateFieldNames allows duplicate form field names.
	AllowDuplicateFieldNames bool

	// ExportBookmarks includes bookmarks in the PDF.
	ExportBookmarks bool

	// ExportBookmarksToPdfDestination exports bookmarks as named destinations.
	// Maps to UNO property "ExportBookmarksToPDFDestination".
	ExportBookmarksToPdfDestination bool

	// ExportPlaceholders exports placeholder fields.
	ExportPlaceholders bool

	// ExportNotes includes comments/notes in the PDF.
	ExportNotes bool

	// ExportNotesPages exports notes pages (Impress).
	ExportNotesPages bool

	// ExportOnlyNotesPages exports only the notes pages.
	ExportOnlyNotesPages bool

	// ExportNotesInMargin places notes in the page margin.
	ExportNotesInMargin bool

	// ConvertOooTargetToPdfTarget converts internal links to PDF targets.
	// Maps to UNO property "ConvertOOoTargetToPDFTarget".
	ConvertOooTargetToPdfTarget bool

	// ExportLinksRelativeFsys exports filesystem links as relative.
	ExportLinksRelativeFsys bool

	// ExportHiddenSlides includes hidden slides in the PDF (Impress).
	ExportHiddenSlides bool

	// SkipEmptyPages omits empty pages from the output.
	// Maps to UNO property "IsSkipEmptyPages".
	SkipEmptyPages bool

	// AddOriginalDocumentAsStream embeds the source document in the PDF.
	// Maps to UNO property "IsAddStream".
	AddOriginalDocumentAsStream bool

	// SinglePageSheets prints each spreadsheet sheet on a single page.
	SinglePageSheets bool

	// LosslessImageCompression uses lossless compression for images.
	// Maps to UNO property "UseLosslessCompression".
	LosslessImageCompression bool

	// Quality sets the JPEG compression quality (1-100).
	Quality int

	// ReduceImageResolution downscales images to MaxImageResolution.
	// Defaults to true, matching LibreOffice.
	ReduceImageResolution bool

	// MaxImageResolution sets the target DPI when ReduceImageResolution is true.
	MaxImageResolution int

	// PDFVersion selects the PDF version (0=standard, 1=PDF/A-1b, 2=PDF/A-2b, 3=PDF/A-3b).
	// Maps to UNO property "SelectPdfVersion".
	PDFVersion int

	// PDFUniversalAccess enables PDF/UA (Universal Accessibility) compliance.
	// Maps to UNO property "PDFUACompliance".
	PDFUniversalAccess bool

	// NativeWatermarkText adds a tiled watermark with the given text.
	// Maps to UNO property "TiledWatermark".
	NativeWatermarkText string

	// InitialView sets the initial view mode (0=default, 1=bookmarks, 2=thumbnails).
	InitialView int

	// InitialPage sets the page displayed when the PDF is opened.
	InitialPage int

	// Magnification sets the default magnification mode.
	Magnification int

	// Zoom sets the default zoom percentage.
	Zoom int

	// PageLayout sets the page layout (0=default, 1=single, 2=continuous, 3=two-left, 4=two-right).
	PageLayout int

	// FirstPageOnLeft places the first page on the left in two-page layout.
	FirstPageOnLeft bool

	// ResizeWindowToInitialPage resizes the viewer window to the first page.
	ResizeWindowToInitialPage bool

	// CenterWindow centers the viewer window on screen.
	CenterWindow bool

	// OpenInFullScreenMode opens the PDF in full-screen mode.
	OpenInFullScreenMode bool

	// DisplayPDFDocumentTitle shows the document title in the viewer title bar.
	DisplayPDFDocumentTitle bool

	// HideViewerMenubar hides the menu bar in the PDF viewer.
	HideViewerMenubar bool

	// HideViewerToolbar hides the toolbar in the PDF viewer.
	HideViewerToolbar bool

	// HideViewerWindowControls hides window controls in the PDF viewer.
	HideViewerWindowControls bool

	// UseTransitionEffects enables slide transition effects in the PDF.
	UseTransitionEffects bool

	// OpenBookmarkLevels sets how many bookmark levels are shown (-1=all).
	OpenBookmarkLevels int
}

// DefaultOptions returns [Options] with default values matching LibreOffice's
// built-in PDF export defaults.
func DefaultOptions() Options {
	return Options{
		ExportBookmarks:       true,
		Quality:               90,
		MaxImageResolution:    300,
		ReduceImageResolution: true,
		Zoom:                  100,
		UseTransitionEffects:  true,
		OpenBookmarkLevels:    -1,
		PaperFormat:           -1,
	}
}

// filterProp represents a single UNO property value in the filter options JSON.
type filterProp struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// BuildFilterOptions serializes non-default [Options] fields into a JSON string
// suitable for LibreOffice's saveAs filter options parameter.
//
// Fields matching [DefaultOptions] are omitted. Landscape, Password, and
// UpdateIndexes are not filter options and are always excluded.
//
// Returns an empty string when all fields match defaults.
func BuildFilterOptions(opts Options) string {
	defaults := DefaultOptions()
	props := make(map[string]filterProp)

	addBool := func(name string, val, def bool) {
		if val != def {
			props[name] = filterProp{Type: "boolean", Value: val}
		}
	}

	addLong := func(name string, val, def int) {
		if val != def {
			props[name] = filterProp{Type: "long", Value: val}
		}
	}

	addString := func(name string, val, def string) {
		if val != def {
			props[name] = filterProp{Type: "string", Value: val}
		}
	}

	// String properties.
	addString("PageRange", opts.PageRanges, defaults.PageRanges)
	addString("TiledWatermark", opts.NativeWatermarkText, defaults.NativeWatermarkText)

	// Bool properties.
	addBool("ExportFormFields", opts.ExportFormFields, defaults.ExportFormFields)
	addBool("AllowDuplicateFieldNames", opts.AllowDuplicateFieldNames, defaults.AllowDuplicateFieldNames)
	addBool("ExportBookmarks", opts.ExportBookmarks, defaults.ExportBookmarks)
	addBool("ExportBookmarksToPDFDestination", opts.ExportBookmarksToPdfDestination, defaults.ExportBookmarksToPdfDestination)
	addBool("ExportPlaceholders", opts.ExportPlaceholders, defaults.ExportPlaceholders)
	addBool("ExportNotes", opts.ExportNotes, defaults.ExportNotes)
	addBool("ExportNotesPages", opts.ExportNotesPages, defaults.ExportNotesPages)
	addBool("ExportOnlyNotesPages", opts.ExportOnlyNotesPages, defaults.ExportOnlyNotesPages)
	addBool("ExportNotesInMargin", opts.ExportNotesInMargin, defaults.ExportNotesInMargin)
	addBool("ConvertOOoTargetToPDFTarget", opts.ConvertOooTargetToPdfTarget, defaults.ConvertOooTargetToPdfTarget)
	addBool("ExportLinksRelativeFsys", opts.ExportLinksRelativeFsys, defaults.ExportLinksRelativeFsys)
	addBool("ExportHiddenSlides", opts.ExportHiddenSlides, defaults.ExportHiddenSlides)
	addBool("IsSkipEmptyPages", opts.SkipEmptyPages, defaults.SkipEmptyPages)
	addBool("IsAddStream", opts.AddOriginalDocumentAsStream, defaults.AddOriginalDocumentAsStream)
	addBool("SinglePageSheets", opts.SinglePageSheets, defaults.SinglePageSheets)
	addBool("UseLosslessCompression", opts.LosslessImageCompression, defaults.LosslessImageCompression)
	addBool("ReduceImageResolution", opts.ReduceImageResolution, defaults.ReduceImageResolution)
	addBool("PDFUACompliance", opts.PDFUniversalAccess, defaults.PDFUniversalAccess)
	addBool("FirstPageOnLeft", opts.FirstPageOnLeft, defaults.FirstPageOnLeft)
	addBool("ResizeWindowToInitialPage", opts.ResizeWindowToInitialPage, defaults.ResizeWindowToInitialPage)
	addBool("CenterWindow", opts.CenterWindow, defaults.CenterWindow)
	addBool("OpenInFullScreenMode", opts.OpenInFullScreenMode, defaults.OpenInFullScreenMode)
	addBool("DisplayPDFDocumentTitle", opts.DisplayPDFDocumentTitle, defaults.DisplayPDFDocumentTitle)
	addBool("HideViewerMenubar", opts.HideViewerMenubar, defaults.HideViewerMenubar)
	addBool("HideViewerToolbar", opts.HideViewerToolbar, defaults.HideViewerToolbar)
	addBool("HideViewerWindowControls", opts.HideViewerWindowControls, defaults.HideViewerWindowControls)
	addBool("UseTransitionEffects", opts.UseTransitionEffects, defaults.UseTransitionEffects)

	// Long properties.
	addLong("Quality", opts.Quality, defaults.Quality)
	addLong("MaxImageResolution", opts.MaxImageResolution, defaults.MaxImageResolution)
	addLong("SelectPdfVersion", opts.PDFVersion, defaults.PDFVersion)
	addLong("InitialView", opts.InitialView, defaults.InitialView)
	addLong("InitialPage", opts.InitialPage, defaults.InitialPage)
	addLong("Magnification", opts.Magnification, defaults.Magnification)
	addLong("Zoom", opts.Zoom, defaults.Zoom)
	addLong("PageLayout", opts.PageLayout, defaults.PageLayout)
	addLong("OpenBookmarkLevels", opts.OpenBookmarkLevels, defaults.OpenBookmarkLevels)

	if len(props) == 0 {
		return ""
	}

	data, err := json.Marshal(props)
	if err != nil {
		// All values are primitive types, so Marshal cannot fail in practice.
		panic(fmt.Sprintf("lok: failed to marshal filter options: %v", err))
	}

	return string(data)
}

// geometry describes the page orientation and size to apply before export.
// width and height are in 1/100 mm; 0 means keep the document's current size.
type geometry struct {
	landscape bool
	width     int
	height    int
}

// resolveGeometry derives the page geometry from [Options]. needed reports
// whether any geometry change was requested. When a paper format and landscape
// are both set, the dimensions are oriented to landscape.
func resolveGeometry(opts Options) (g geometry, needed bool) {
	g.landscape = opts.Landscape

	if opts.PaperFormat == PaperFormatUser {
		g.width, g.height = opts.PaperWidth, opts.PaperHeight
	} else if opts.PaperFormat >= 0 {
		g.width, g.height = paperFormatDimensions(opts.PaperFormat)
	}

	if g.landscape && g.width > 0 && g.height > 0 && g.width < g.height {
		g.width, g.height = g.height, g.width
	}

	needed = g.landscape || (g.width > 0 && g.height > 0)

	return g, needed
}

// paperFormatDimensions returns the width and height of a paper format in
// 1/100 mm (portrait). It returns 0, 0 for [PaperFormatUser] or an unknown
// format.
func paperFormatDimensions(f PaperFormat) (width, height int) {
	switch f {
	case PaperFormatA3:
		return 29700, 42000
	case PaperFormatA4:
		return 21000, 29700
	case PaperFormatA5:
		return 14800, 21000
	case PaperFormatB4:
		return 25000, 35300
	case PaperFormatB5:
		return 17600, 25000
	case PaperFormatLetter:
		return 21590, 27940
	case PaperFormatLegal:
		return 21590, 35560
	case PaperFormatTabloid:
		return 27940, 43180
	case PaperFormatUser:
		return 0, 0
	default:
		return 0, 0
	}
}

// BuildLoadOptions builds a document load options string from [Options].
// Returns an empty string if no load options are set.
//
// Format: comma-separated key=value pairs (e.g., "Password=secret").
func BuildLoadOptions(opts Options) string {
	var parts []string

	if opts.Password != "" {
		parts = append(parts, "Password="+opts.Password)
	}

	if opts.MacroExecutionMode > 0 {
		parts = append(parts, fmt.Sprintf("MacroExecutionMode=%d", opts.MacroExecutionMode))
	}

	if len(parts) == 0 {
		return ""
	}

	var result strings.Builder
	for i, p := range parts {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(p)
	}

	return result.String()
}
