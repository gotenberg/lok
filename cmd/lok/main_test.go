package main

import (
	"encoding/json"
	"flag"
	"testing"
)

// TestBuildOptsFromRequest_WatermarkZeros pins that a long-running request
// keeps an absent watermark color or rotation nil, so LibreOffice applies its
// default, and passes an explicit 0 through.
func TestBuildOptsFromRequest_WatermarkZeros(t *testing.T) {
	tests := []struct {
		name string
		line string
		// wantColor and wantAngle are nil when the option must stay unset.
		wantColor *int
		wantAngle *int
	}{
		{"absent", `{}`, nil, nil},
		{"rotation 0", `{"nativeWatermarkRotateAngle":0}`, nil, new(0)},
		{"color 0", `{"nativeWatermarkColor":0}`, new(0), nil},
		{"both set", `{"nativeWatermarkColor":16711680,"nativeWatermarkRotateAngle":450}`, new(0xFF0000), new(450)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req longRunningRequest
			if err := json.Unmarshal([]byte(tt.line), &req); err != nil {
				t.Fatalf("decoding %s: %v", tt.line, err)
			}

			opts := buildOptsFromRequest(req)

			assertOptionalInt(t, "NativeWatermarkColor", opts.NativeWatermarkColor, tt.wantColor)
			assertOptionalInt(t, "NativeWatermarkRotateAngle", opts.NativeWatermarkRotateAngle, tt.wantAngle)
		})
	}
}

// TestSetOptionalInt pins that an optional integer flag stays nil until given,
// takes an explicit 0, and parses hexadecimal like [flag.IntVar].
func TestSetOptionalInt(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want *int
	}{
		{"unset", nil, nil},
		{"zero", []string{"-n", "0"}, new(0)},
		{"hexadecimal", []string{"-n", "0xFF0000"}, new(0xFF0000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *int

			fs := flag.NewFlagSet("lok", flag.ContinueOnError)
			fs.Func("n", "", setOptionalInt(&got))

			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("parsing %v: %v", tt.args, err)
			}

			assertOptionalInt(t, "-n", got, tt.want)
		})
	}

	t.Run("invalid", func(t *testing.T) {
		var got *int

		if err := setOptionalInt(&got)("red"); err == nil {
			t.Fatal("expected an error for a non-integer value")
		}

		if got != nil {
			t.Fatalf("got %d after a parse error, want nil", *got)
		}
	})
}

func assertOptionalInt(t *testing.T, name string, got, want *int) {
	t.Helper()

	switch {
	case want == nil && got != nil:
		t.Errorf("%s = %d, want nil", name, *got)
	case want != nil && got == nil:
		t.Errorf("%s = nil, want %d", name, *want)
	case want != nil && *got != *want:
		t.Errorf("%s = %d, want %d", name, *got, *want)
	}
}
