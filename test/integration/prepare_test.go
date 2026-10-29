//go:build integration

package integration

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

// trackedChangesDocument writes a DOCX holding one tracked insertion and one
// tracked deletion.
func trackedChangesDocument(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "tracked.docx")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	for _, e := range []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
 <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
 <Default Extension="xml" ContentType="application/xml"/>
 <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
 <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`},
		{"word/document.xml", `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
 <w:body>
  <w:p>
   <w:r><w:t xml:space="preserve">KEPTTEXT </w:t></w:r>
   <w:ins w:id="1" w:author="lok" w:date="2026-01-01T00:00:00Z"><w:r><w:t xml:space="preserve">INSERTEDTEXT </w:t></w:r></w:ins>
   <w:del w:id="2" w:author="lok" w:date="2026-01-01T00:00:00Z"><w:r><w:delText>DELETEDTEXT</w:delText></w:r></w:del>
  </w:p>
 </w:body>
</w:document>`},
	} {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestPrepare_TrackedChanges(t *testing.T) {
	input := trackedChangesDocument(t)

	t.Run("hidden by default", func(t *testing.T) {
		text := pdfText(t, convertFixture(t, input, lok.DefaultOptions()))
		if !strings.Contains(text, "KEPTTEXT") || !strings.Contains(text, "INSERTEDTEXT") {
			t.Fatalf("the accepted text is missing:\n%s", text)
		}
		if strings.Contains(text, "DELETEDTEXT") {
			t.Fatalf("the tracked deletion is exported:\n%s", text)
		}
	})

	t.Run("shown on request", func(t *testing.T) {
		opts := lok.DefaultOptions()
		opts.ShowTrackedChanges = true

		if text := pdfText(t, convertFixture(t, input, opts)); !strings.Contains(text, "DELETEDTEXT") {
			t.Fatalf("the tracked deletion is not exported:\n%s", text)
		}
	})
}

func TestPrepare_DisableCalcHeader(t *testing.T) {
	// Calc names a CSV's only sheet after the file and prints the sheet name in
	// the default page header.
	input := filepath.Join(t.TempDir(), "LEAKEDNAME.csv")
	if err := os.WriteFile(input, []byte("a,b\n1,2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if text := pdfText(t, convertFixture(t, input, lok.DefaultOptions())); !strings.Contains(text, "LEAKEDNAME") {
		t.Fatalf("the default header does not print the file name, so this test proves nothing:\n%s", text)
	}

	opts := lok.DefaultOptions()
	opts.DisableCalcHeader = true

	if text := pdfText(t, convertFixture(t, input, opts)); strings.Contains(text, "LEAKEDNAME") {
		t.Fatalf("the header still prints the file name:\n%s", text)
	}
}
