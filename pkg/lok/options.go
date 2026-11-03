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
	// PaperFormatA3 selects the A3 paper size.
	PaperFormatA3 PaperFormat = 0
	// PaperFormatA4 selects the A4 paper size.
	PaperFormatA4 PaperFormat = 1
	// PaperFormatA5 selects the A5 paper size.
	PaperFormatA5 PaperFormat = 2
	// PaperFormatB4 selects the B4 paper size.
	PaperFormatB4 PaperFormat = 3
	// PaperFormatB5 selects the B5 paper size.
	PaperFormatB5 PaperFormat = 4
	// PaperFormatLetter selects the US Letter paper size.
	PaperFormatLetter PaperFormat = 5
	// PaperFormatLegal selects the US Legal paper size.
	PaperFormatLegal PaperFormat = 6
	// PaperFormatTabloid selects the Tabloid paper size.
	PaperFormatTabloid PaperFormat = 7
	// PaperFormatUser selects a custom size set by PaperWidth and PaperHeight.
	PaperFormatUser PaperFormat = 8
)

// Options configures PDF export behavior for LibreOffice document conversion.
//
// Most fields map directly to UNO filter properties passed to saveAs via
// [BuildFilterOptions]. Several fields are handled separately by the
// conversion layer:
//   - Landscape, PaperFormat, PaperWidth, PaperHeight, UpdateIndexes,
//     DisableCalcHeader, ShowTrackedChanges: applied to the document by a
//     Basic macro before export. See [Convert].
//   - Password: answers the document-password request on load.
//   - EnableMacros, FilterOptions: passed as document load options.
type Options struct {
	// Landscape sets the page orientation to landscape before export.
	// Not a filter option.
	Landscape bool

	// PageRanges limits export to the specified pages (e.g., "1-3,5").
	// Maps to UNO property "PageRange".
	PageRanges string

	// Password opens an encrypted document. Answers LibreOffice's password
	// request rather than being passed as a load option. Not a filter option.
	Password string //nolint:gosec // document open password, not a credential

	// UpdateIndexes refreshes table of contents and indexes before export.
	// Not a filter option.
	UpdateIndexes bool

	// EnableMacros runs the document's macros, including the ones bound to
	// document events such as load, without asking. Defaults to false: macros
	// never run. Only enable it for trusted documents. Passed as document load
	// options. Not a filter option.
	EnableMacros bool

	// FilterOptions sets the LibreOffice import filter options, passed
	// through as the FilterOptions load property. Empty keeps the filter's
	// defaults, except for a .csv input, where [Convert] uses
	// [DefaultCSVFilterOptions]: LibreOfficeKit gives the CSV filter no
	// separator otherwise, and every line lands in one cell. The value is
	// filter specific, and other text filters read it differently (the Writer
	// text filter takes an encoding name), so only set it for the format of the
	// input. Not a filter option of the PDF export.
	FilterOptions string

	// DisableCalcHeader turns off the page header of every spreadsheet page
	// style. Calc prints the sheet name there by default, and a plain-text
	// import such as CSV names its only sheet after the input file, so the
	// header leaks the file name. Not a filter option.
	DisableCalcHeader bool

	// ShowTrackedChanges exports tracked changes (insertions and deletions) as
	// marked up. Defaults to false: the document is exported as if every
	// change were accepted. Not a filter option.
	ShowTrackedChanges bool

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

	// PDFVersion selects the PDF version (0=standard, 1=PDF/A-1b, 2=PDF/A-2b,
	// 3=PDF/A-3b, 4=PDF/A-4). Maps to UNO property "SelectPdfVersion".
	PDFVersion int

	// PDFUniversalAccess enables PDF/UA (Universal Accessibility) compliance.
	// PDF/UA requires a tagged PDF, so it also turns TaggedPdf on.
	// Maps to UNO property "PDFUACompliance".
	PDFUniversalAccess bool

	// TaggedPdf writes the document structure as PDF tags, for accessibility
	// tools. Maps to UNO property "UseTaggedPDF".
	TaggedPdf bool

	// EmbedStandardFonts embeds the 14 standard PDF fonts instead of
	// referencing them, as PDF/A and PDF/UA require. Maps to UNO property
	// "EmbedStandardFonts".
	EmbedStandardFonts bool

	// NativeWatermarkText draws one watermark with the given text across the
	// middle of each page. Maps to UNO property "Watermark".
	NativeWatermarkText string

	// NativeWatermarkColor sets the watermark color as a 0xRRGGBB integer.
	// Nil keeps LibreOffice's default color, and 0 is black. Maps to UNO
	// property "WatermarkColor".
	NativeWatermarkColor *int

	// NativeWatermarkFontHeight sets the watermark font height in points.
	// 0 lets LibreOffice fit the text to the page, so lok never passes it:
	// LibreOffice would draw an explicit 0 at its 12 pt default. Maps to UNO
	// property "WatermarkFontHeight".
	NativeWatermarkFontHeight int

	// NativeWatermarkRotateAngle sets the watermark rotation in tenths of a
	// degree, counterclockwise. Nil keeps LibreOffice's default, which runs
	// the text along the longer side of the page, and 0 draws it horizontal.
	// Maps to UNO property "WatermarkRotateAngle".
	NativeWatermarkRotateAngle *int

	// NativeWatermarkFontName sets the watermark font. Empty keeps
	// LibreOffice's default font. Maps to UNO property "WatermarkFontName".
	NativeWatermarkFontName string

	// NativeTiledWatermarkText repeats the given text as a watermark tiled
	// over each page. Maps to UNO property "TiledWatermark".
	NativeTiledWatermarkText string

	// InitialView sets the initial view mode (0=default, 1=bookmarks, 2=thumbnails).
	InitialView int

	// InitialPage sets the page displayed when the PDF is opened.
	InitialPage int

	// Magnification sets the default magnification mode.
	Magnification int

	// Zoom sets the default zoom percentage.
	Zoom int

	// PageLayout sets the page layout the viewer opens with: 0 = default,
	// 1 = SinglePage, 2 = OneColumn (continuous), 3 = TwoColumnRight (facing
	// pages, odd pages on the right); other values = default.
	PageLayout int

	// FirstPageOnLeft has no effect: LibreOffice resets it, so a facing
	// layout always opens on TwoColumnRight.
	FirstPageOnLeft bool

	// ResizeWindowToInitialPage resizes the viewer window to the first page.
	ResizeWindowToInitialPage bool

	// CenterWindow centers the viewer window on screen.
	CenterWindow bool

	// OpenInFullScreenMode opens the PDF in full-screen mode.
	OpenInFullScreenMode bool

	// DisplayPDFDocumentTitle shows the document title in the viewer title bar.
	// Defaults to true, matching LibreOffice. LibreOffice writes
	// /DisplayDocTitle true whatever the value when the PDF is PDF/UA.
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

// DefaultCSVFilterOptions is the import filter token [Convert] applies to a
// .csv input when [Options.FilterOptions] is empty.
//
// The token has ten comma-separated positions, as documented in the CSV filter
// options of the LibreOffice developer guide
// (https://wiki.documentfoundation.org/Documentation/DevGuide/Spreadsheet_Documents#Filter_Options_for_the_CSV_Filter):
//
//  1. Field separators as decimal ASCII codes joined by "/": 44 (comma), 59
//     (semicolon) and 9 (tab).
//  2. Text delimiter: 34 (").
//  3. Character set: 76 (UTF-8).
//  4. First line to import: 1.
//  5. Cell formats per column: empty, so every column is detected.
//  6. Language identifier: 0 (system default).
//  7. Quoted field as text: false, so "1" in quotes is still a number.
//  8. Detect special numbers: true, so dates and similar are recognized.
//  9. Save cell contents as shown: true.
//  10. Remove spaces: false, so leading and trailing spaces are kept.
const DefaultCSVFilterOptions = "44/59/9,34,76,1,,0,false,true,true,false"

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
		// LibreOffice's export filter reads DisplayPDFDocumentTitle with true
		// as its default, so only false has to reach it.
		DisplayPDFDocumentTitle: true,
		OpenBookmarkLevels:      -1,
		PaperFormat:             -1,
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
// Fields matching [DefaultOptions] are omitted, as are nil pointer fields.
// Landscape, Password, and UpdateIndexes are not filter options and are
// always excluded.
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

	// addOptLong sends a set pointer even when it points to the zero value:
	// LibreOffice tells an unset property from 0.
	addOptLong := func(name string, v *int) {
		if v != nil {
			props[name] = filterProp{Type: "long", Value: *v}
		}
	}

	addString := func(name string, val, def string) {
		if val != def {
			props[name] = filterProp{Type: "string", Value: val}
		}
	}

	// String properties.
	addString("PageRange", opts.PageRanges, defaults.PageRanges)
	addString("Watermark", opts.NativeWatermarkText, defaults.NativeWatermarkText)
	addString("WatermarkFontName", opts.NativeWatermarkFontName, defaults.NativeWatermarkFontName)
	addString("TiledWatermark", opts.NativeTiledWatermarkText, defaults.NativeTiledWatermarkText)

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
	addBool("UseTaggedPDF", opts.TaggedPdf || opts.PDFUniversalAccess, defaults.TaggedPdf)
	addBool("EmbedStandardFonts", opts.EmbedStandardFonts, defaults.EmbedStandardFonts)
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
	addLong("WatermarkFontHeight", opts.NativeWatermarkFontHeight, defaults.NativeWatermarkFontHeight)
	addOptLong("WatermarkColor", opts.NativeWatermarkColor)
	addOptLong("WatermarkRotateAngle", opts.NativeWatermarkRotateAngle)

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

// resolveGeometry derives the page geometry from [Options]. A zero geometry
// leaves the page styles untouched. When a paper format and landscape are both
// set, the dimensions are oriented to landscape.
func resolveGeometry(opts Options) geometry {
	g := geometry{landscape: opts.Landscape}

	if opts.PaperFormat == PaperFormatUser {
		g.width, g.height = opts.PaperWidth, opts.PaperHeight
	} else if opts.PaperFormat >= 0 {
		g.width, g.height = paperFormatDimensions(opts.PaperFormat)
	}

	if g.landscape && g.width > 0 && g.height > 0 && g.width < g.height {
		g.width, g.height = g.height, g.width
	}

	return g
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
//
// It sets Batch=true unless a password is set. Without it, LibreOfficeKit
// waits on import dialogs a headless process cannot answer, such as Calc's
// text import dialog for a CSV, and the load never returns.
//
// Format: comma-separated key=value pairs. The password is not a load
// option: LibreOfficeKit ignores it there, so [Convert] answers the
// document-password request instead.
//
// [Options.FilterOptions] comes last. LibreOfficeKit removes the keys it knows
// (Language, Timezone, Batch, MacroSecurityLevel, ClientVisibleArea and
// EnableMacrosExecution) wherever they sit, and hands the rest to the import
// filter as its FilterOptions property, commas included.
func BuildLoadOptions(opts Options) string {
	var parts []string

	// Batch also makes the load silent, which skips the document-password
	// request. Encrypted documents are ODF or OOXML, whose import shows no
	// dialog, so they load without it.
	if opts.Password == "" {
		parts = append(parts, "Batch=true")
	}

	// LibreOfficeKit reads EnableMacrosExecution and MacroSecurityLevel, and
	// resets the security level to 1 (medium: ask first) on every load. Level 0
	// runs macros without asking, since nothing can answer a prompt.
	if opts.EnableMacros {
		parts = append(parts, "EnableMacrosExecution=true", "MacroSecurityLevel=0")
	}

	if opts.FilterOptions != "" {
		parts = append(parts, opts.FilterOptions)
	}

	return strings.Join(parts, ",")
}
