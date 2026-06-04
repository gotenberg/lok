//go:build integration

package integration

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

// convertFixture runs a full [lok.Convert] of input into a temp PDF and returns
// its path, failing the test if conversion or the %PDF check fails.
func convertFixture(t *testing.T, input string, opts lok.Options) string {
	t.Helper()

	out := filepath.Join(t.TempDir(), "out.pdf")

	if err := lok.Convert(sharedOffice, input, out, opts); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	assertValidPDF(t, out)

	return out
}

func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

// TestBehavior_PageRanges validates that PageRanges limits the exported pages.
// document.docx is five pages.
func TestBehavior_PageRanges(t *testing.T) {
	input := testdataPath(t, "document.docx")

	cases := []struct {
		ranges string
		want   int
	}{
		{"", 5},
		{"1", 1},
		{"1-3", 3},
		{"2-5", 4},
	}

	for _, c := range cases {
		opts := lok.DefaultOptions()
		opts.PageRanges = c.ranges

		out := convertFixture(t, input, opts)

		if got := pdfPageCount(t, out); got != c.want {
			t.Errorf("PageRanges=%q: got %d pages, want %d", c.ranges, got, c.want)
		}
	}
}

// TestBehavior_DefaultPageSize validates that a default conversion of
// document.docx yields its native US Letter portrait geometry (612x792 pts).
func TestBehavior_DefaultPageSize(t *testing.T) {
	input := testdataPath(t, "document.docx")

	out := convertFixture(t, input, lok.DefaultOptions())
	w, h := pdfPageSize(t, out)

	if !approxEqual(w, 612, 2) || !approxEqual(h, 792, 2) {
		t.Errorf("default page size = %.1f x %.1f pts, want ~612 x 792 (Letter portrait)", w, h)
	}

	if w >= h {
		t.Errorf("expected portrait (width < height), got %.1f x %.1f", w, h)
	}
}

// TestBehavior_PDFAConformance validates that PDFVersion produces a PDF that
// veraPDF certifies as conformant with the corresponding PDF/A flavour.
//
// PDF/A-1b (PDFVersion 1) is covered separately: LibreOffice 25.2 emits a
// CreationDate that does not match the XMP xmp:CreateDate, which veraPDF rejects.
func TestBehavior_PDFAConformance(t *testing.T) {
	input := testdataPath(t, "document.docx")

	cases := []struct {
		version int
		flavour string
	}{
		{2, "2b"},
		{3, "3b"},
	}

	for _, c := range cases {
		opts := lok.DefaultOptions()
		opts.PDFVersion = c.version

		out := convertFixture(t, input, opts)
		assertPDFACompliant(t, out, c.flavour)
	}
}

// TestBehavior_PDFA1b_KnownCreationDateMismatch pins a known limitation: the
// LibreOffice 25.2 PDF/A-1b export fails veraPDF clause 6.7.3 because the
// document information CreationDate and the XMP xmp:CreateDate differ. If a
// future LibreOffice fixes this, switch PDF/A-1b into the conformance table.
func TestBehavior_PDFA1b_KnownCreationDateMismatch(t *testing.T) {
	input := testdataPath(t, "document.docx")

	opts := lok.DefaultOptions()
	opts.PDFVersion = 1

	out := convertFixture(t, input, opts)
	report := verapdfReport(t, out, "1b")

	if !strings.Contains(report, `isCompliant="false"`) {
		t.Fatalf("PDF/A-1b is now compliant; move it into TestBehavior_PDFAConformance:\n%s", report)
	}

	if !strings.Contains(report, `clause="6.7.3"`) {
		t.Errorf("expected the known clause 6.7.3 (CreationDate) failure, got:\n%s", report)
	}
}

// TestBehavior_DefaultIsNotPDFA confirms veraPDF discriminates: a default export
// (PDFVersion 0) is not PDF/A-2b conformant.
func TestBehavior_DefaultIsNotPDFA(t *testing.T) {
	input := testdataPath(t, "document.docx")

	out := convertFixture(t, input, lok.DefaultOptions())

	report := verapdfReport(t, out, "2b")
	if !strings.Contains(report, `isCompliant="false"`) {
		t.Errorf("expected default export to be non-PDF/A-2b, report:\n%s", report)
	}
}

// TestBehavior_HasTrimMemory validates that the trimMemory API is available on
// the test image (LibreOffice 25.2, built with -DLOK_HAS_TRIM_MEMORY).
func TestBehavior_HasTrimMemory(t *testing.T) {
	if !sharedOffice.HasTrimMemory() {
		t.Fatal("expected trimMemory support on LibreOffice 25.2; check -DLOK_HAS_TRIM_MEMORY in the build")
	}
}

// TestBehavior_Landscape validates that landscape orientation rotates the page
// so width exceeds height. document.docx is US Letter, so landscape is 792x612.
func TestBehavior_Landscape(t *testing.T) {
	input := testdataPath(t, "document.docx")

	opts := lok.DefaultOptions()
	opts.Landscape = true
	w, h := pdfPageSize(t, convertFixture(t, input, opts))

	if w <= h {
		t.Fatalf("landscape: expected width > height, got %.0f x %.0f", w, h)
	}

	if !approxEqual(w, 792, 2) || !approxEqual(h, 612, 2) {
		t.Errorf("landscape page size = %.1f x %.1f, want ~792 x 612", w, h)
	}
}

// TestBehavior_PaperFormat validates that PaperFormat overrides the document's
// native size: an A4 request yields A4 (595x842 pts) instead of Letter.
func TestBehavior_PaperFormat(t *testing.T) {
	input := testdataPath(t, "document.docx")

	opts := lok.DefaultOptions()
	opts.PaperFormat = lok.PaperFormatA4
	w, h := pdfPageSize(t, convertFixture(t, input, opts))

	if !approxEqual(w, 595, 2) || !approxEqual(h, 842, 2) {
		t.Errorf("A4 page size = %.1f x %.1f, want ~595 x 842", w, h)
	}
}

// TestBehavior_PaperFormatLandscape validates a paper format combined with
// landscape orientation: A4 landscape is 842x595 pts.
func TestBehavior_PaperFormatLandscape(t *testing.T) {
	input := testdataPath(t, "document.docx")

	opts := lok.DefaultOptions()
	opts.PaperFormat = lok.PaperFormatA4
	opts.Landscape = true
	w, h := pdfPageSize(t, convertFixture(t, input, opts))

	if !approxEqual(w, 842, 2) || !approxEqual(h, 595, 2) {
		t.Errorf("A4 landscape size = %.1f x %.1f, want ~842 x 595", w, h)
	}
}

// TestBehavior_CustomPaperSize validates a custom (user) paper size in 1/100 mm:
// 100x200 mm is 283x567 pts.
func TestBehavior_CustomPaperSize(t *testing.T) {
	input := testdataPath(t, "document.docx")

	opts := lok.DefaultOptions()
	opts.PaperFormat = lok.PaperFormatUser
	opts.PaperWidth = 10000  // 100 mm
	opts.PaperHeight = 20000 // 200 mm
	w, h := pdfPageSize(t, convertFixture(t, input, opts))

	if !approxEqual(w, 283, 2) || !approxEqual(h, 567, 2) {
		t.Errorf("custom page size = %.1f x %.1f, want ~283 x 567", w, h)
	}
}
