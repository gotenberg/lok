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

// TestBehavior_Landscape_NotAppliedBySaveAs pins a known limitation: landscape
// is requested via .uno:AttributePageSize, a page-style change that headless
// LibreOfficeKit does not apply to the saveAs output. The page geometry is
// unchanged. If a future LibreOffice honors it, SetLandscape/Convert and this
// test must be updated.
func TestBehavior_Landscape_NotAppliedBySaveAs(t *testing.T) {
	input := testdataPath(t, "document.docx")

	pw, ph := pdfPageSize(t, convertFixture(t, input, lok.DefaultOptions()))

	opts := lok.DefaultOptions()
	opts.Landscape = true
	lw, lh := pdfPageSize(t, convertFixture(t, input, opts))

	if lw != pw || lh != ph {
		t.Fatalf("landscape changed geometry from %.0fx%.0f to %.0fx%.0f; it may now be honored, update the library", pw, ph, lw, lh)
	}
}

// TestBehavior_PaperFormat_NotAppliedBySaveAs pins a known limitation: the
// printer-descriptor paper format set via .uno:Printer does not affect the
// saveAs output, so an A4 request still yields the document's native Letter size.
func TestBehavior_PaperFormat_NotAppliedBySaveAs(t *testing.T) {
	input := testdataPath(t, "document.docx")

	opts := lok.DefaultOptions()
	opts.PaperFormat = lok.PaperFormatA4
	w, h := pdfPageSize(t, convertFixture(t, input, opts))

	// A4 would be ~595x842 pts; the output is unchanged Letter (612x792).
	if !approxEqual(w, 612, 2) || !approxEqual(h, 792, 2) {
		t.Fatalf("paper format now affects saveAs output (%.0fx%.0f); update the library", w, h)
	}
}

// TestBehavior_CustomPaperSize_NotAppliedBySaveAs pins the same limitation for a
// custom (user) paper size, encoded as a com.sun.star.awt.Size on .uno:Printer.
func TestBehavior_CustomPaperSize_NotAppliedBySaveAs(t *testing.T) {
	input := testdataPath(t, "document.docx")

	opts := lok.DefaultOptions()
	opts.PaperFormat = lok.PaperFormatUser
	opts.PaperWidth = 10000  // 100mm
	opts.PaperHeight = 20000 // 200mm
	w, h := pdfPageSize(t, convertFixture(t, input, opts))

	// 100x200mm would be ~283x567 pts; the output is unchanged Letter.
	if !approxEqual(w, 612, 2) || !approxEqual(h, 792, 2) {
		t.Fatalf("custom paper size now affects saveAs output (%.0fx%.0f); update the library", w, h)
	}
}
