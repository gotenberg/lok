//go:build benchmark

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

func BenchmarkLifecycle_200Conversions(b *testing.B) {
	inputPath := filepath.Join("testdata", "document.docx")
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		b.Skip("test fixture not found, see testdata/README.md")
	}

	progPath := os.Getenv("LOK_PROGRAM_PATH")
	if progPath == "" {
		progPath = "/usr/lib/libreoffice/program"
	}

	lc, err := lok.NewLifecycle(lok.LifecycleConfig{
		ProgramPath:  progPath,
		TrimInterval: 10,
	})
	if err != nil {
		b.Fatalf("NewLifecycle failed: %v", err)
	}

	defer lc.Close()

	b.ResetTimer()

	for i := 0; i < 200; i++ {
		outPath := filepath.Join(b.TempDir(), fmt.Sprintf("output_%d.pdf", i))

		err = lc.Convert(inputPath, outPath, lok.DefaultOptions())
		if err != nil {
			b.Fatalf("Convert %d failed: %v", i, err)
		}

		if (i+1)%20 == 0 {
			rss, rssErr := lc.RSS()
			if rssErr != nil {
				b.Logf("conversion %d: RSS unavailable: %v", i+1, rssErr)
			} else {
				b.Logf("conversion %d: RSS = %d bytes (%.1f MiB)",
					i+1, rss, float64(rss)/(1024*1024))
			}
		}
	}

	b.StopTimer()

	b.Logf("total conversions: %d", lc.ConversionCount())

	rss, err := lc.RSS()
	if err == nil {
		b.Logf("final RSS: %d bytes (%.1f MiB)", rss, float64(rss)/(1024*1024))
	}
}
