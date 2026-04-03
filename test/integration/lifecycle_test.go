//go:build integration

package integration

import (
	"path/filepath"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

func TestLifecycle_Integration(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")

	lc, err := lok.NewLifecycle(lok.LifecycleConfig{
		ProgramPath:  programPath(t),
		TrimInterval: 3,
	})
	if err != nil {
		t.Fatalf("NewLifecycle failed: %v", err)
	}

	defer lc.Close()

	for i := 0; i < 5; i++ {
		outPath := filepath.Join(t.TempDir(), "output.pdf")

		err = lc.Convert(inputPath, outPath, lok.DefaultOptions())
		if err != nil {
			t.Fatalf("Convert %d failed: %v", i, err)
		}

		assertValidPDF(t, outPath)
	}

	if lc.ConversionCount() != 5 {
		t.Fatalf("expected 5 conversions, got %d", lc.ConversionCount())
	}

	rss, err := lc.RSS()
	if err != nil {
		t.Fatalf("RSS failed: %v", err)
	}

	t.Logf("RSS after 5 conversions: %d bytes (%.1f MiB)", rss, float64(rss)/(1024*1024))
}
