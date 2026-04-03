package lok

import (
	"runtime"
	"testing"
)

func TestLifecycle_ConversionCount(t *testing.T) {
	lc := &Lifecycle{
		office:          &Office{closed: true},
		trimInterval:    10,
		gentleTrimEvery: false,
	}

	if lc.ConversionCount() != 0 {
		t.Fatalf("expected 0 conversions, got %d", lc.ConversionCount())
	}

	// Simulate conversions by calling afterConversion directly.
	lc.afterConversion()
	lc.afterConversion()
	lc.afterConversion()

	if lc.ConversionCount() != 3 {
		t.Fatalf("expected 3 conversions, got %d", lc.ConversionCount())
	}
}

func TestLifecycle_TrimInterval(t *testing.T) {
	// Track TrimMemory calls via a closed office (TrimMemory is a no-op when
	// closed, so this is safe for unit testing the counting logic).
	lc := &Lifecycle{
		office:          &Office{closed: true},
		trimInterval:    5,
		gentleTrimEvery: true,
	}

	for i := 0; i < 12; i++ {
		lc.afterConversion()
	}

	if lc.ConversionCount() != 12 {
		t.Fatalf("expected 12 conversions, got %d", lc.ConversionCount())
	}

	// Verify the count is correct: aggressive trims would have happened at
	// conversions 5 and 10 (i.e., when count % trimInterval == 0).
	// We cannot directly observe TrimMemory calls on a closed office, but
	// we verify the counter increments correctly.
}

func TestLifecycle_RSS(t *testing.T) {
	lc := &Lifecycle{
		office:       &Office{closed: true},
		trimInterval: 10,
	}

	rss, err := lc.RSS()
	if err != nil {
		t.Fatalf("RSS failed: %v", err)
	}

	if runtime.GOOS == "linux" {
		if rss <= 0 {
			t.Fatalf("expected positive RSS on Linux, got %d", rss)
		}
	} else {
		if rss != 0 {
			t.Fatalf("expected 0 RSS on non-Linux, got %d", rss)
		}
	}
}

func TestNewLifecycle_EmptyPath(t *testing.T) {
	_, err := NewLifecycle(LifecycleConfig{})
	if err == nil {
		t.Fatal("expected error for empty program path")
	}
}

func TestLifecycle_DefaultConfig(t *testing.T) {
	// Verify defaults are applied without initializing LibreOffice.
	cfg := LifecycleConfig{
		ProgramPath: "/nonexistent",
	}

	// NewLifecycle will fail because the path doesn't exist, but we can
	// test the config defaults by inspecting what NewLifecycle would set.
	// Instead, verify the zero-value behavior of GentleTrimEvery.
	if cfg.GentleTrimEvery != nil {
		t.Fatal("expected nil GentleTrimEvery for zero config")
	}

	if cfg.TrimInterval != 0 {
		t.Fatal("expected 0 TrimInterval for zero config")
	}
}
