//go:build integration

package integration

import (
	"bytes"
	"context"
	"image/png"
	"testing"

	"github.com/gotenberg/litho/pkg/models"
	"github.com/gotenberg/litho/pkg/pdf"
)

// These helpers read the output with litho, in process, so integration tests
// assert real output properties (page count, dimensions, text, tagging, PDF/A
// conformance) rather than only checking the %PDF header. litho's PDF/A
// validator is scored against veraPDF in litho's own test suite.

// openPDF opens path with litho and closes it when the test ends.
func openPDF(t *testing.T, path string) *pdf.Reader {
	t.Helper()

	r, err := pdf.OpenFile(context.Background(), path)
	if err != nil {
		t.Fatalf("opening %q: %v", path, err)
	}

	t.Cleanup(func() { r.Close() })

	return r
}

// pdfPageCount returns the number of pages in the PDF.
func pdfPageCount(t *testing.T, path string) int {
	t.Helper()

	n, err := openPDF(t, path).PageCount(context.Background())
	if err != nil {
		t.Fatalf("counting the pages of %q: %v", path, err)
	}

	return n
}

// pdfPageSize returns the width and height of the first page in points.
func pdfPageSize(t *testing.T, path string) (width, height float64) {
	t.Helper()

	page, err := openPDF(t, path).Page(context.Background(), 1)
	if err != nil {
		t.Fatalf("reading the first page of %q: %v", path, err)
	}

	return page.Width, page.Height
}

// pdfText extracts the text content of the PDF.
func pdfText(t *testing.T, path string) string {
	t.Helper()

	text, err := openPDF(t, path).ExtractAllText(context.Background())
	if err != nil {
		t.Fatalf("extracting the text of %q: %v", path, err)
	}

	return text
}

// pdfTagged reports whether the PDF declares itself a Tagged PDF.
func pdfTagged(t *testing.T, path string) bool {
	t.Helper()

	tagged, err := openPDF(t, path).IsTagged(context.Background())
	if err != nil {
		t.Fatalf("reading the tagging of %q: %v", path, err)
	}

	return tagged
}

// pdfRenderFirstPage renders the first page at a low resolution and returns it
// as PNG bytes, to compare what two PDFs draw.
func pdfRenderFirstPage(t *testing.T, path string) []byte {
	t.Helper()

	img, err := openPDF(t, path).RenderPage(context.Background(), 1, pdf.WithRenderDPI(20))
	if err != nil {
		t.Fatalf("rendering the first page of %q: %v", path, err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding the render of %q: %v", path, err)
	}

	return buf.Bytes()
}

// pdfaViolations validates the PDF against a PDF/A profile and returns the
// violations, none when it conforms.
func pdfaViolations(t *testing.T, path string, profile models.PDFAProfile) []pdf.Violation {
	t.Helper()

	violations, err := openPDF(t, path).ValidatePDFAReport(context.Background(), profile)
	if err != nil {
		t.Fatalf("validating %q against %s: %v", path, profile, err)
	}

	return violations
}

// assertPDFACompliant fails the test if the PDF does not conform to profile.
func assertPDFACompliant(t *testing.T, path string, profile models.PDFAProfile) {
	t.Helper()

	if violations := pdfaViolations(t, path, profile); len(violations) > 0 {
		t.Fatalf("PDF %q is not %s compliant: %v", path, profile, violations)
	}
}
