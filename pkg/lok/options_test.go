package lok

import (
	"encoding/json"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"ExportFormFields", opts.ExportFormFields, false},
		{"ReduceImageResolution", opts.ReduceImageResolution, true},
		{"ExportBookmarks", opts.ExportBookmarks, true},
		{"Quality", opts.Quality, 90},
		{"MaxImageResolution", opts.MaxImageResolution, 300},
		{"Zoom", opts.Zoom, 100},
		{"UseTransitionEffects", opts.UseTransitionEffects, true},
		{"OpenBookmarkLevels", opts.OpenBookmarkLevels, -1},
		{"Landscape", opts.Landscape, false},
		{"Password", opts.Password, ""},
		{"PageRanges", opts.PageRanges, ""},
		{"PaperFormat", int(opts.PaperFormat), -1},
		{"MacroExecutionMode", opts.MacroExecutionMode, 0},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestBuildFilterOptions_Empty(t *testing.T) {
	result := BuildFilterOptions(DefaultOptions())
	if result != "" {
		t.Fatalf("expected empty string for defaults, got: %s", result)
	}
}

func TestBuildFilterOptions_SingleBool(t *testing.T) {
	opts := DefaultOptions()
	opts.ExportNotes = true

	result := BuildFilterOptions(opts)
	props := parseFilterJSON(t, result)

	assertProp(t, props, "ExportNotes", "boolean", true)

	if len(props) != 1 {
		t.Fatalf("expected 1 property, got %d: %s", len(props), result)
	}
}

func TestBuildFilterOptions_PageRanges(t *testing.T) {
	opts := DefaultOptions()
	opts.PageRanges = "1-3,5"

	result := BuildFilterOptions(opts)
	props := parseFilterJSON(t, result)

	assertProp(t, props, "PageRange", "string", "1-3,5")
}

func TestBuildFilterOptions_TrickyNames(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Options)
		unoName  string
		unoType  string
		unoValue any
	}{
		{
			name:     "ExportBookmarksToPdfDestination",
			mutate:   func(o *Options) { o.ExportBookmarksToPdfDestination = true },
			unoName:  "ExportBookmarksToPDFDestination",
			unoType:  "boolean",
			unoValue: true,
		},
		{
			name:     "ConvertOooTargetToPdfTarget",
			mutate:   func(o *Options) { o.ConvertOooTargetToPdfTarget = true },
			unoName:  "ConvertOOoTargetToPDFTarget",
			unoType:  "boolean",
			unoValue: true,
		},
		{
			name:     "SkipEmptyPages",
			mutate:   func(o *Options) { o.SkipEmptyPages = true },
			unoName:  "IsSkipEmptyPages",
			unoType:  "boolean",
			unoValue: true,
		},
		{
			name:     "AddOriginalDocumentAsStream",
			mutate:   func(o *Options) { o.AddOriginalDocumentAsStream = true },
			unoName:  "IsAddStream",
			unoType:  "boolean",
			unoValue: true,
		},
		{
			name:     "LosslessImageCompression",
			mutate:   func(o *Options) { o.LosslessImageCompression = true },
			unoName:  "UseLosslessCompression",
			unoType:  "boolean",
			unoValue: true,
		},
		{
			name:     "PageRanges",
			mutate:   func(o *Options) { o.PageRanges = "2-4" },
			unoName:  "PageRange",
			unoType:  "string",
			unoValue: "2-4",
		},
		{
			name:     "NativeWatermarkText",
			mutate:   func(o *Options) { o.NativeWatermarkText = "DRAFT" },
			unoName:  "TiledWatermark",
			unoType:  "string",
			unoValue: "DRAFT",
		},
		{
			name:     "PDFVersion",
			mutate:   func(o *Options) { o.PDFVersion = 2 },
			unoName:  "SelectPdfVersion",
			unoType:  "long",
			unoValue: float64(2),
		},
		{
			name:     "PDFUniversalAccess",
			mutate:   func(o *Options) { o.PDFUniversalAccess = true },
			unoName:  "PDFUACompliance",
			unoType:  "boolean",
			unoValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultOptions()
			tt.mutate(&opts)

			result := BuildFilterOptions(opts)
			props := parseFilterJSON(t, result)

			assertProp(t, props, tt.unoName, tt.unoType, tt.unoValue)
		})
	}
}

func TestBuildFilterOptions_Quality(t *testing.T) {
	opts := DefaultOptions()
	opts.Quality = 50

	result := BuildFilterOptions(opts)
	props := parseFilterJSON(t, result)

	assertProp(t, props, "Quality", "long", float64(50))
}

func TestBuildFilterOptions_PDFVersion(t *testing.T) {
	opts := DefaultOptions()
	opts.PDFVersion = 1

	result := BuildFilterOptions(opts)
	props := parseFilterJSON(t, result)

	assertProp(t, props, "SelectPdfVersion", "long", float64(1))
}

func TestBuildFilterOptions_Watermark(t *testing.T) {
	opts := DefaultOptions()
	opts.NativeWatermarkText = "CONFIDENTIAL"

	result := BuildFilterOptions(opts)
	props := parseFilterJSON(t, result)

	assertProp(t, props, "TiledWatermark", "string", "CONFIDENTIAL")
}

func TestBuildFilterOptions_AllOptions(t *testing.T) {
	opts := Options{
		PageRanges:                      "1-5",
		ExportFormFields:                true,
		AllowDuplicateFieldNames:        true,
		ExportBookmarks:                 false,
		ExportBookmarksToPdfDestination: true,
		ExportPlaceholders:              true,
		ExportNotes:                     true,
		ExportNotesPages:                true,
		ExportOnlyNotesPages:            true,
		ExportNotesInMargin:             true,
		ConvertOooTargetToPdfTarget:     true,
		ExportLinksRelativeFsys:         true,
		ExportHiddenSlides:              true,
		SkipEmptyPages:                  true,
		AddOriginalDocumentAsStream:     true,
		SinglePageSheets:                true,
		LosslessImageCompression:        true,
		Quality:                         50,
		ReduceImageResolution:           false,
		MaxImageResolution:              150,
		PDFVersion:                      2,
		PDFUniversalAccess:              true,
		NativeWatermarkText:             "DRAFT",
		InitialView:                     1,
		InitialPage:                     3,
		Magnification:                   2,
		Zoom:                            75,
		PageLayout:                      1,
		FirstPageOnLeft:                 true,
		ResizeWindowToInitialPage:       true,
		CenterWindow:                    true,
		OpenInFullScreenMode:            true,
		DisplayPDFDocumentTitle:         true,
		HideViewerMenubar:               true,
		HideViewerToolbar:               true,
		HideViewerWindowControls:        true,
		UseTransitionEffects:            false,
		OpenBookmarkLevels:              3,
	}

	result := BuildFilterOptions(opts)
	props := parseFilterJSON(t, result)

	// Verify a representative set of properties.
	assertProp(t, props, "PageRange", "string", "1-5")
	assertProp(t, props, "ExportFormFields", "boolean", true)
	assertProp(t, props, "ReduceImageResolution", "boolean", false)
	assertProp(t, props, "ExportBookmarks", "boolean", false)
	assertProp(t, props, "IsSkipEmptyPages", "boolean", true)
	assertProp(t, props, "UseLosslessCompression", "boolean", true)
	assertProp(t, props, "IsAddStream", "boolean", true)
	assertProp(t, props, "SelectPdfVersion", "long", float64(2))
	assertProp(t, props, "PDFUACompliance", "boolean", true)
	assertProp(t, props, "TiledWatermark", "string", "DRAFT")
	assertProp(t, props, "UseTransitionEffects", "boolean", false)
	assertProp(t, props, "OpenBookmarkLevels", "long", float64(3))
	assertProp(t, props, "Quality", "long", float64(50))
	assertProp(t, props, "Zoom", "long", float64(75))

	// Landscape, Password, UpdateIndexes must not appear.
	for _, excluded := range []string{"Landscape", "Password", "UpdateIndexes"} {
		if _, ok := props[excluded]; ok {
			t.Errorf("excluded field %q should not be in filter options", excluded)
		}
	}
}

func TestBuildFilterOptions_OmitsDefaults(t *testing.T) {
	opts := DefaultOptions()
	// Set a single non-default value.
	opts.ExportNotes = true

	result := BuildFilterOptions(opts)
	props := parseFilterJSON(t, result)

	// Default values must not appear.
	defaultKeys := []string{
		"ExportFormFields", "ExportBookmarks", "Quality",
		"MaxImageResolution", "Zoom", "UseTransitionEffects",
		"OpenBookmarkLevels",
	}

	for _, key := range defaultKeys {
		if _, ok := props[key]; ok {
			t.Errorf("default field %q should not be in filter options", key)
		}
	}

	// Only the changed field should be present.
	if len(props) != 1 {
		t.Fatalf("expected 1 property, got %d: %s", len(props), result)
	}
}

// parseFilterJSON unmarshals the filter options JSON string into a map.
func parseFilterJSON(t *testing.T, s string) map[string]map[string]any {
	t.Helper()

	if s == "" {
		t.Fatal("expected non-empty filter options JSON")
	}

	var props map[string]map[string]any
	if err := json.Unmarshal([]byte(s), &props); err != nil {
		t.Fatalf("failed to parse filter JSON: %v\nraw: %s", err, s)
	}

	return props
}

// assertProp checks that a UNO property has the expected type and value.
func assertProp(t *testing.T, props map[string]map[string]any, name, typ string, value any) {
	t.Helper()

	prop, ok := props[name]
	if !ok {
		t.Fatalf("missing property %q", name)
	}

	if prop["type"] != typ {
		t.Errorf("%s: type = %v, want %v", name, prop["type"], typ)
	}

	if prop["value"] != value {
		t.Errorf("%s: value = %v (%T), want %v (%T)", name, prop["value"], prop["value"], value, value)
	}
}

func TestResolveGeometry_None(t *testing.T) {
	if _, needed := resolveGeometry(DefaultOptions()); needed {
		t.Fatal("expected no geometry for default options")
	}
}

func TestResolveGeometry_Landscape(t *testing.T) {
	opts := DefaultOptions()
	opts.Landscape = true

	g, needed := resolveGeometry(opts)
	if !needed || !g.landscape {
		t.Fatalf("expected landscape geometry, got %+v needed=%v", g, needed)
	}

	if g.width != 0 || g.height != 0 {
		t.Errorf("expected no explicit size, got %dx%d", g.width, g.height)
	}
}

func TestResolveGeometry_PaperFormat(t *testing.T) {
	opts := DefaultOptions()
	opts.PaperFormat = PaperFormatA4

	g, needed := resolveGeometry(opts)
	if !needed {
		t.Fatal("expected geometry for A4")
	}

	if g.width != 21000 || g.height != 29700 {
		t.Errorf("A4 portrait = %dx%d, want 21000x29700", g.width, g.height)
	}

	if g.landscape {
		t.Error("expected portrait orientation")
	}
}

func TestResolveGeometry_PaperFormatLandscape(t *testing.T) {
	opts := DefaultOptions()
	opts.PaperFormat = PaperFormatA4
	opts.Landscape = true

	g, _ := resolveGeometry(opts)
	if g.width != 29700 || g.height != 21000 {
		t.Errorf("A4 landscape = %dx%d, want 29700x21000", g.width, g.height)
	}
}

func TestResolveGeometry_CustomSize(t *testing.T) {
	opts := DefaultOptions()
	opts.PaperFormat = PaperFormatUser
	opts.PaperWidth = 10000
	opts.PaperHeight = 20000

	g, needed := resolveGeometry(opts)
	if !needed || g.width != 10000 || g.height != 20000 {
		t.Errorf("custom = %dx%d needed=%v, want 10000x20000", g.width, g.height, needed)
	}
}

func TestPaperFormatDimensions(t *testing.T) {
	if w, h := paperFormatDimensions(PaperFormatLetter); w != 21590 || h != 27940 {
		t.Errorf("Letter = %dx%d, want 21590x27940", w, h)
	}

	if w, h := paperFormatDimensions(PaperFormatUser); w != 0 || h != 0 {
		t.Errorf("User = %dx%d, want 0x0", w, h)
	}
}

func TestBuildLoadOptions_Empty(t *testing.T) {
	result := BuildLoadOptions(DefaultOptions())
	if result != "" {
		t.Fatalf("expected empty string for defaults, got: %s", result)
	}
}

func TestBuildLoadOptions_Password(t *testing.T) {
	opts := DefaultOptions()
	opts.Password = "secret"

	result := BuildLoadOptions(opts)
	if result != "Password=secret" {
		t.Fatalf("expected Password=secret, got: %s", result)
	}
}

func TestBuildLoadOptions_MacroExecutionMode(t *testing.T) {
	opts := DefaultOptions()
	opts.MacroExecutionMode = 7

	result := BuildLoadOptions(opts)
	if result != "MacroExecutionMode=7" {
		t.Fatalf("expected MacroExecutionMode=7, got: %s", result)
	}
}

func TestBuildLoadOptions_Combined(t *testing.T) {
	opts := DefaultOptions()
	opts.Password = "secret"
	opts.MacroExecutionMode = 7

	result := BuildLoadOptions(opts)
	if result != "Password=secret,MacroExecutionMode=7" {
		t.Fatalf("expected Password=secret,MacroExecutionMode=7, got: %s", result)
	}
}
