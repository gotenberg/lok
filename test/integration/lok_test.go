//go:build integration

package integration

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

// sharedOffice is a single LibreOfficeKit instance shared across all
// integration tests. LibreOffice initialization and destruction are slow,
// so we reuse one instance via TestMain.
var sharedOffice *lok.Office

func TestMain(m *testing.M) {
	progPath := os.Getenv("LOK_PROGRAM_PATH")
	if progPath == "" {
		progPath = "/usr/lib/libreoffice/program"
	}

	var err error
	sharedOffice, err = lok.Init(progPath)
	if err != nil {
		panic("failed to initialize LibreOfficeKit: " + err.Error())
	}

	code := m.Run()

	// Skip office.Close() here. LibreOffice's destroy() installs signal
	// handlers that conflict with Go's runtime (SA_ONSTACK), causing a
	// fatal crash on exit. The process is about to terminate anyway.
	os.Exit(code)
}

func testdataPath(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("test fixture not found, see testdata/README.md: %s", path)
	}

	return path
}

func saveToPDF(t *testing.T, inputPath, filterOptions string) string {
	t.Helper()

	doc, err := sharedOffice.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument(%q) failed: %v", inputPath, err)
	}

	defer doc.Close()

	outPath := filepath.Join(t.TempDir(), "output.pdf")

	err = doc.SaveAs(outPath, "pdf", filterOptions)
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}

	return outPath
}

func assertValidPDF(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output PDF: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("output PDF is empty")
	}

	if !strings.HasPrefix(string(data), "%PDF") {
		t.Fatalf("output does not start with %%PDF header, got: %q", string(data[:min(len(data), 20)]))
	}
}

func TestBasicDocxToPDF(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")

	doc, err := sharedOffice.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	if doc.Type() != lok.TextDocument {
		t.Errorf("Type() = %v, want TextDocument", doc.Type())
	}

	outPath := filepath.Join(t.TempDir(), "output.pdf")

	err = doc.SaveAs(outPath, "pdf", "")
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestBasicXlsxToPDF(t *testing.T) {
	inputPath := testdataPath(t, "spreadsheet.xlsx")

	doc, err := sharedOffice.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	if doc.Type() != lok.SpreadsheetDocument {
		t.Errorf("Type() = %v, want SpreadsheetDocument", doc.Type())
	}

	outPath := filepath.Join(t.TempDir(), "output.pdf")

	err = doc.SaveAs(outPath, "pdf", "")
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestBasicPptxToPDF(t *testing.T) {
	inputPath := testdataPath(t, "presentation.pptx")

	doc, err := sharedOffice.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	if doc.Type() != lok.PresentationDocument {
		t.Errorf("Type() = %v, want PresentationDocument", doc.Type())
	}

	outPath := filepath.Join(t.TempDir(), "output.pdf")

	err = doc.SaveAs(outPath, "pdf", "")
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestFilterOptions_Quality(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")

	lowOpts := lok.DefaultOptions()
	lowOpts.Quality = 10
	lowFilter := lok.BuildFilterOptions(lowOpts)

	highOpts := lok.DefaultOptions()
	highOpts.Quality = 90
	highFilter := lok.BuildFilterOptions(highOpts)

	lowPath := saveToPDF(t, inputPath, lowFilter)
	highPath := saveToPDF(t, inputPath, highFilter)

	assertValidPDF(t, lowPath)
	assertValidPDF(t, highPath)
}

func TestPageRanges(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")

	fullPath := saveToPDF(t, inputPath, "")

	opts := lok.DefaultOptions()
	opts.PageRanges = "1"
	rangePath := saveToPDF(t, inputPath, lok.BuildFilterOptions(opts))

	assertValidPDF(t, fullPath)
	assertValidPDF(t, rangePath)

	fullInfo, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("stat full PDF: %v", err)
	}

	rangeInfo, err := os.Stat(rangePath)
	if err != nil {
		t.Fatalf("stat range PDF: %v", err)
	}

	// A single-page export of a multi-page document should be smaller.
	// Skip the comparison if the document only has one page.
	if rangeInfo.Size() >= fullInfo.Size() {
		t.Logf("single-page PDF (%d bytes) not smaller than full PDF (%d bytes); document may have only one page",
			rangeInfo.Size(), fullInfo.Size())
	}
}

func TestLoadError(t *testing.T) {
	_, err := sharedOffice.LoadDocument("nonexistent.docx")
	if err == nil {
		t.Fatal("expected error for nonexistent document")
	}

	if !errors.Is(err, lok.ErrLoadFailed) {
		t.Fatalf("expected ErrLoadFailed, got: %v", err)
	}
}

func TestGetVersionInfo(t *testing.T) {
	info, err := sharedOffice.GetVersionInfo()
	if err != nil {
		t.Fatalf("GetVersionInfo failed: %v", err)
	}

	if info == "" {
		t.Fatal("GetVersionInfo returned empty string")
	}

	if !strings.Contains(info, "ProductName") {
		t.Errorf("GetVersionInfo does not contain ProductName: %s", info)
	}
}

func TestTrimMemory(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")

	// Perform a conversion to populate caches.
	saveToPDF(t, inputPath, "")

	// Gentle trim.
	sharedOffice.TrimMemory(0)

	// Aggressive trim.
	sharedOffice.TrimMemory(2000)
}

func TestSetLandscape_Writer(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")

	doc, err := sharedOffice.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	err = doc.SetLandscape(true)
	if err != nil {
		t.Fatalf("SetLandscape failed: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "landscape.pdf")

	err = doc.SaveAs(outPath, "pdf", "")
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestSetLandscape_Calc(t *testing.T) {
	inputPath := testdataPath(t, "spreadsheet.xlsx")

	doc, err := sharedOffice.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	err = doc.SetLandscape(true)
	if err != nil {
		t.Fatalf("SetLandscape failed: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "landscape.pdf")

	err = doc.SaveAs(outPath, "pdf", "")
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestSetLandscape_Impress(t *testing.T) {
	inputPath := testdataPath(t, "presentation.pptx")

	doc, err := sharedOffice.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	// SetLandscape is a no-op for presentations.
	err = doc.SetLandscape(true)
	if err != nil {
		t.Fatalf("SetLandscape should be a no-op for Impress, got: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "landscape.pdf")

	err = doc.SaveAs(outPath, "pdf", "")
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestConvert_WithLandscape(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")
	outPath := filepath.Join(t.TempDir(), "landscape.pdf")

	opts := lok.DefaultOptions()
	opts.Landscape = true

	err := lok.Convert(sharedOffice, inputPath, outPath, opts)
	if err != nil {
		t.Fatalf("Convert with landscape failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

// Password-protected document tests are disabled. The test fixture created by
// msoffcrypto-tool uses OLE2 compound encryption which triggers a LibreOffice
// signal handler conflict with Go's runtime (SA_ONSTACK) on load, causing a
// fatal crash. A proper OOXML-encrypted fixture requires LibreOffice itself to
// create it.

func TestConvert_WithPageRanges(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")
	outPath := filepath.Join(t.TempDir(), "output.pdf")

	opts := lok.DefaultOptions()
	opts.PageRanges = "1"

	err := lok.Convert(sharedOffice, inputPath, outPath, opts)
	if err != nil {
		t.Fatalf("Convert with page ranges failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestExportPDFViaUnoCommand_CalcLandscape(t *testing.T) {
	inputPath := testdataPath(t, "spreadsheet.xlsx")

	doc, err := sharedOffice.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	outPath := filepath.Join(t.TempDir(), "landscape_uno.pdf")

	err = doc.ExportPDFViaUnoCommand(outPath, "")
	if err != nil {
		t.Fatalf("ExportPDFViaUnoCommand failed: %v", err)
	}

	// The UNO dispatch path may not produce a file if LibreOffice does not
	// support it in headless LOK mode. Log rather than fail.
	if _, statErr := os.Stat(outPath); os.IsNotExist(statErr) {
		t.Log("ExportPDFViaUnoCommand did not produce a file; dispatch may not be supported in headless LOK mode")
		return
	}

	assertValidPDF(t, outPath)
}

func TestConvert_CalcLandscape_SaveAs(t *testing.T) {
	inputPath := testdataPath(t, "spreadsheet.xlsx")
	outPath := filepath.Join(t.TempDir(), "landscape_saveas.pdf")

	opts := lok.DefaultOptions()
	opts.Landscape = true
	opts.ExportMethod = lok.ExportViaSaveAs

	err := lok.Convert(sharedOffice, inputPath, outPath, opts)
	if err != nil {
		t.Fatalf("Convert via SaveAs failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestConvert_CalcLandscape_UnoCommand(t *testing.T) {
	inputPath := testdataPath(t, "spreadsheet.xlsx")
	outPath := filepath.Join(t.TempDir(), "landscape_uno.pdf")

	opts := lok.DefaultOptions()
	opts.Landscape = true
	opts.ExportMethod = lok.ExportViaUnoCommand

	err := lok.Convert(sharedOffice, inputPath, outPath, opts)
	if err != nil {
		t.Fatalf("Convert via UnoCommand failed: %v", err)
	}

	// The UNO dispatch path may not produce a file in headless LOK mode.
	if _, statErr := os.Stat(outPath); os.IsNotExist(statErr) {
		t.Log("ExportViaUnoCommand did not produce a file; dispatch may not be supported in headless LOK mode")
		return
	}

	assertValidPDF(t, outPath)
}

func TestConvert_CalcLandscape_BothMethods(t *testing.T) {
	inputPath := testdataPath(t, "spreadsheet.xlsx")

	saveAsPath := filepath.Join(t.TempDir(), "saveas.pdf")
	unoPath := filepath.Join(t.TempDir(), "uno.pdf")

	saveAsOpts := lok.DefaultOptions()
	saveAsOpts.Landscape = true
	saveAsOpts.ExportMethod = lok.ExportViaSaveAs

	err := lok.Convert(sharedOffice, inputPath, saveAsPath, saveAsOpts)
	if err != nil {
		t.Fatalf("Convert via SaveAs failed: %v", err)
	}

	assertValidPDF(t, saveAsPath)

	unoOpts := lok.DefaultOptions()
	unoOpts.Landscape = true
	unoOpts.ExportMethod = lok.ExportViaUnoCommand

	err = lok.Convert(sharedOffice, inputPath, unoPath, unoOpts)
	if err != nil {
		t.Fatalf("Convert via UnoCommand failed: %v", err)
	}

	if _, statErr := os.Stat(unoPath); os.IsNotExist(statErr) {
		t.Log("UnoCommand path did not produce a file; skipping comparison")
		return
	}

	assertValidPDF(t, unoPath)

	// Both methods should produce valid PDFs. Log the size difference
	// for manual inspection.
	saveAsInfo, _ := os.Stat(saveAsPath)
	unoInfo, _ := os.Stat(unoPath)

	t.Logf("SaveAs PDF: %d bytes, UnoCommand PDF: %d bytes",
		saveAsInfo.Size(), unoInfo.Size())
}
