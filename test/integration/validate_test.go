//go:build integration

package integration

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// These helpers shell out to poppler-utils (pdfinfo, pdftotext) and veraPDF,
// provided by the test image. They let integration tests assert real output
// properties (page count, dimensions, text, PDF/A conformance) rather than only
// checking the %PDF header.

// pdfInfo runs pdfinfo and returns its key/value fields.
func pdfInfo(t *testing.T, path string) map[string]string {
	t.Helper()

	out, err := exec.Command("pdfinfo", path).Output()
	if err != nil {
		t.Fatalf("pdfinfo %q failed: %v", path, err)
	}

	fields := make(map[string]string)

	for line := range strings.SplitSeq(string(out), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return fields
}

// pdfPageCount returns the number of pages in the PDF.
func pdfPageCount(t *testing.T, path string) int {
	t.Helper()

	pages := pdfInfo(t, path)["Pages"]

	n, err := strconv.Atoi(pages)
	if err != nil {
		t.Fatalf("parsing page count %q: %v", pages, err)
	}

	return n
}

var pageSizeRe = regexp.MustCompile(`([0-9.]+) x ([0-9.]+)`)

// pdfPageSize returns the width and height of the first page in points.
func pdfPageSize(t *testing.T, path string) (width, height float64) {
	t.Helper()

	size := pdfInfo(t, path)["Page size"]

	m := pageSizeRe.FindStringSubmatch(size)
	if m == nil {
		t.Fatalf("could not parse page size %q", size)
	}

	width, _ = strconv.ParseFloat(m[1], 64)
	height, _ = strconv.ParseFloat(m[2], 64)

	return width, height
}

// pdfText extracts the text content of the PDF.
func pdfText(t *testing.T, path string) string {
	t.Helper()

	out, err := exec.Command("pdftotext", path, "-").Output()
	if err != nil {
		t.Fatalf("pdftotext %q failed: %v", path, err)
	}

	return string(out)
}

// verapdfReport runs veraPDF for the given flavour and returns its
// machine-readable report. veraPDF exits non-zero for non-compliant files, so
// the exit code is ignored and the report is parsed instead.
func verapdfReport(t *testing.T, path, flavour string) string {
	t.Helper()

	out, _ := exec.Command("verapdf", "--flavour", flavour, path).CombinedOutput()

	return string(out)
}

// assertPDFACompliant fails the test if the PDF does not conform to the given
// veraPDF flavour (for example "1b", "2b", "3b", or "ua1").
func assertPDFACompliant(t *testing.T, path, flavour string) {
	t.Helper()

	report := verapdfReport(t, path, flavour)

	switch {
	case strings.Contains(report, `isCompliant="false"`):
		t.Fatalf("PDF %q is not %s compliant:\n%s", path, flavour, report)
	case strings.Contains(report, `isCompliant="true"`):
		return
	default:
		t.Fatalf("could not determine %s conformance of %q:\n%s", flavour, path, report)
	}
}
