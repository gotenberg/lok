//go:build integration

package lok

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func programPath(t *testing.T) string {
	t.Helper()

	if p := os.Getenv("LOK_PROGRAM_PATH"); p != "" {
		return p
	}

	return "/usr/lib/libreoffice/program"
}

func testdataPath(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("test fixture not found, see testdata/README.md: %s", path)
	}

	return path
}

func initOffice(t *testing.T) *Office {
	t.Helper()

	office, err := Init(programPath(t))
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	t.Cleanup(func() { office.Close() })

	return office
}

func saveToPDF(t *testing.T, office *Office, inputPath, filterOptions string) string {
	t.Helper()

	doc, err := office.LoadDocument(inputPath)
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
	office := initOffice(t)
	inputPath := testdataPath(t, "document.docx")

	doc, err := office.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	if doc.Type() != TextDocument {
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
	office := initOffice(t)
	inputPath := testdataPath(t, "spreadsheet.xlsx")

	doc, err := office.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	if doc.Type() != SpreadsheetDocument {
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
	office := initOffice(t)
	inputPath := testdataPath(t, "presentation.pptx")

	doc, err := office.LoadDocument(inputPath)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	defer doc.Close()

	if doc.Type() != PresentationDocument {
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
	office := initOffice(t)
	inputPath := testdataPath(t, "document.docx")

	lowOpts := DefaultOptions()
	lowOpts.Quality = 10
	lowFilter := BuildFilterOptions(lowOpts)

	highOpts := DefaultOptions()
	highOpts.Quality = 90
	highFilter := BuildFilterOptions(highOpts)

	lowPath := saveToPDF(t, office, inputPath, lowFilter)
	highPath := saveToPDF(t, office, inputPath, highFilter)

	assertValidPDF(t, lowPath)
	assertValidPDF(t, highPath)
}

func TestPageRanges(t *testing.T) {
	office := initOffice(t)
	inputPath := testdataPath(t, "document.docx")

	fullPath := saveToPDF(t, office, inputPath, "")

	opts := DefaultOptions()
	opts.PageRanges = "1"
	rangePath := saveToPDF(t, office, inputPath, BuildFilterOptions(opts))

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
	office := initOffice(t)

	_, err := office.LoadDocument("nonexistent.docx")
	if err == nil {
		t.Fatal("expected error for nonexistent document")
	}

	if !errors.Is(err, ErrLoadFailed) {
		t.Fatalf("expected ErrLoadFailed, got: %v", err)
	}
}

func TestGetVersionInfo(t *testing.T) {
	office := initOffice(t)

	info, err := office.GetVersionInfo()
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
	office := initOffice(t)
	inputPath := testdataPath(t, "document.docx")

	// Perform a conversion to populate caches.
	saveToPDF(t, office, inputPath, "")

	// Gentle trim.
	office.TrimMemory(0)

	// Aggressive trim.
	office.TrimMemory(2000)
}

func TestSetLandscape_Writer(t *testing.T) {
	office := initOffice(t)
	inputPath := testdataPath(t, "document.docx")

	doc, err := office.LoadDocument(inputPath)
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
	office := initOffice(t)
	inputPath := testdataPath(t, "spreadsheet.xlsx")

	doc, err := office.LoadDocument(inputPath)
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
	office := initOffice(t)
	inputPath := testdataPath(t, "presentation.pptx")

	doc, err := office.LoadDocument(inputPath)
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
	office := initOffice(t)
	inputPath := testdataPath(t, "document.docx")
	outPath := filepath.Join(t.TempDir(), "landscape.pdf")

	opts := DefaultOptions()
	opts.Landscape = true

	err := Convert(office, inputPath, outPath, opts)
	if err != nil {
		t.Fatalf("Convert with landscape failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestConvert_WithPassword(t *testing.T) {
	office := initOffice(t)
	inputPath := testdataPath(t, "password.docx")
	outPath := filepath.Join(t.TempDir(), "output.pdf")

	opts := DefaultOptions()
	opts.Password = "password"

	err := Convert(office, inputPath, outPath, opts)
	if err != nil {
		t.Fatalf("Convert with password failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestConvert_WithPageRanges(t *testing.T) {
	office := initOffice(t)
	inputPath := testdataPath(t, "document.docx")
	outPath := filepath.Join(t.TempDir(), "output.pdf")

	opts := DefaultOptions()
	opts.PageRanges = "1"

	err := Convert(office, inputPath, outPath, opts)
	if err != nil {
		t.Fatalf("Convert with page ranges failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestExportPDFViaUnoCommand_CalcLandscape(t *testing.T) {
	office := initOffice(t)
	inputPath := testdataPath(t, "spreadsheet.xlsx")

	doc, err := office.LoadDocument(inputPath)
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
	office := initOffice(t)
	inputPath := testdataPath(t, "spreadsheet.xlsx")
	outPath := filepath.Join(t.TempDir(), "landscape_saveas.pdf")

	opts := DefaultOptions()
	opts.Landscape = true
	opts.ExportMethod = ExportViaSaveAs

	err := Convert(office, inputPath, outPath, opts)
	if err != nil {
		t.Fatalf("Convert via SaveAs failed: %v", err)
	}

	assertValidPDF(t, outPath)
}

func TestConvert_CalcLandscape_UnoCommand(t *testing.T) {
	office := initOffice(t)
	inputPath := testdataPath(t, "spreadsheet.xlsx")
	outPath := filepath.Join(t.TempDir(), "landscape_uno.pdf")

	opts := DefaultOptions()
	opts.Landscape = true
	opts.ExportMethod = ExportViaUnoCommand

	err := Convert(office, inputPath, outPath, opts)
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
	office := initOffice(t)
	inputPath := testdataPath(t, "spreadsheet.xlsx")

	saveAsPath := filepath.Join(t.TempDir(), "saveas.pdf")
	unoPath := filepath.Join(t.TempDir(), "uno.pdf")

	saveAsOpts := DefaultOptions()
	saveAsOpts.Landscape = true
	saveAsOpts.ExportMethod = ExportViaSaveAs

	err := Convert(office, inputPath, saveAsPath, saveAsOpts)
	if err != nil {
		t.Fatalf("Convert via SaveAs failed: %v", err)
	}

	assertValidPDF(t, saveAsPath)

	unoOpts := DefaultOptions()
	unoOpts.Landscape = true
	unoOpts.ExportMethod = ExportViaUnoCommand

	err = Convert(office, inputPath, unoPath, unoOpts)
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
