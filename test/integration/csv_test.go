//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

// TestCSV_ColumnsAreSeparateCells pins that a .csv input loads with its
// separators applied. Without import filter options, LibreOfficeKit puts each
// line in one cell and the commas stay in the text.
func TestCSV_ColumnsAreSeparateCells(t *testing.T) {
	input := testdataPath(t, "contacts.csv")

	text := pdfText(t, convertFixture(t, input, lok.DefaultOptions()))

	for _, want := range []string{"Name", "City", "Qty", "Paris, France", "Café"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF text lacks %q:\n%s", want, text)
		}
	}

	for _, unwanted := range []string{"Name,City", "City,Qty", "Alice,", `"`} {
		if strings.Contains(text, unwanted) {
			t.Errorf("the PDF text still carries %q, so the CSV was not split:\n%s", unwanted, text)
		}
	}
}

// TestCSV_UppercaseExtension pins that the extension match ignores the case.
func TestCSV_UppercaseExtension(t *testing.T) {
	data, err := os.ReadFile(testdataPath(t, "contacts.csv"))
	if err != nil {
		t.Fatal(err)
	}

	input := filepath.Join(t.TempDir(), "CONTACTS.CSV")
	if err := os.WriteFile(input, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if text := pdfText(t, convertFixture(t, input, lok.DefaultOptions())); strings.Contains(text, "Name,City") {
		t.Fatalf("the CSV was not split:\n%s", text)
	}
}

// TestCSV_PlainTextIsUnaffected pins that a .txt input never receives the CSV
// token: the Writer text filter reads it as an encoding name.
func TestCSV_PlainTextIsUnaffected(t *testing.T) {
	input := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(input, []byte("Name,City,Qty\nAlice,Paris,3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if text := pdfText(t, convertFixture(t, input, lok.DefaultOptions())); !strings.Contains(text, "Name,City,Qty") {
		t.Fatalf("the text file was altered:\n%s", text)
	}
}

// TestCSV_FilterOptionsOverride pins that explicit FilterOptions replace the
// default CSV token, here with a semicolon-only separator.
func TestCSV_FilterOptionsOverride(t *testing.T) {
	input := filepath.Join(t.TempDir(), "semi.csv")
	if err := os.WriteFile(input, []byte("a,b;c\n1,2;3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	opts := lok.DefaultOptions()
	opts.FilterOptions = "59,34,76,1,,0,false,true,true,false"

	if text := pdfText(t, convertFixture(t, input, opts)); !strings.Contains(text, "a,b") {
		t.Fatalf("the comma was treated as a separator despite the override:\n%s", text)
	}
}
