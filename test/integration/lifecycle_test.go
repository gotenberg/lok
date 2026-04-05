//go:build integration

package integration

import (
	"path/filepath"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

func TestLifecycle_MultipleConversions(t *testing.T) {
	inputPath := testdataPath(t, "document.docx")

	// Validate that the shared office handles multiple sequential conversions
	// with trim calls between them (the same pattern Lifecycle automates).
	for i := 0; i < 5; i++ {
		outPath := filepath.Join(t.TempDir(), "output.pdf")

		err := lok.Convert(sharedOffice, inputPath, outPath, lok.DefaultOptions())
		if err != nil {
			t.Fatalf("Convert %d failed: %v", i, err)
		}

		assertValidPDF(t, outPath)

		// Gentle trim after each conversion.
		sharedOffice.TrimMemory(0)
	}

	// Aggressive trim at the end.
	sharedOffice.TrimMemory(2000)
}
