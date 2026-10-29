//go:build integration

package integration

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

const fixturePassword = "s3cret"

// encryptedCopy saves document.docx in format, encrypted with fixturePassword.
// LibreOffice writes the fixture itself, so its encryption is one LibreOffice
// reads back.
func encryptedCopy(t *testing.T, format string) string {
	t.Helper()

	doc, err := sharedOffice.LoadDocument(testdataPath(t, "document.docx"))
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}
	defer doc.Close()

	path := filepath.Join(t.TempDir(), "encrypted."+format)

	// LibreOfficeKit's saveAs reads ",Password=...PASSWORDEND" out of the filter
	// options and encrypts the saved file with it.
	if err := doc.SaveAs(path, format, ",Password="+fixturePassword+"PASSWORDEND"); err != nil {
		t.Fatalf("SaveAs encrypted %s failed: %v", format, err)
	}

	if _, err := sharedOffice.LoadDocument(path); err == nil {
		t.Fatalf("the %s fixture opens without a password, so it is not encrypted", format)
	}

	return path
}

func TestPassword(t *testing.T) {
	for _, format := range []string{"odt", "docx"} {
		t.Run(format, func(t *testing.T) {
			input := encryptedCopy(t, format)

			t.Run("right password converts", func(t *testing.T) {
				opts := lok.DefaultOptions()
				opts.Password = fixturePassword

				assertValidPDF(t, convertFixture(t, input, opts))
			})

			t.Run("wrong password fails without blocking", func(t *testing.T) {
				opts := lok.DefaultOptions()
				opts.Password = "wrong"

				err := lok.Convert(sharedOffice, input, filepath.Join(t.TempDir(), "out.pdf"), opts)
				if !errors.Is(err, lok.ErrLoadFailed) {
					t.Fatalf("Convert with a wrong password: got %v, want ErrLoadFailed", err)
				}
			})

			t.Run("missing password fails", func(t *testing.T) {
				err := lok.Convert(sharedOffice, input, filepath.Join(t.TempDir(), "out.pdf"), lok.DefaultOptions())
				if !errors.Is(err, lok.ErrLoadFailed) {
					t.Fatalf("Convert without a password: got %v, want ErrLoadFailed", err)
				}
			})

			t.Run("office still converts afterwards", func(t *testing.T) {
				assertValidPDF(t, convertFixture(t, testdataPath(t, "document.docx"), lok.DefaultOptions()))
			})
		})
	}
}
