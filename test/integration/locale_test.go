//go:build integration

package integration

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotenberg/lok/pkg/lok"
)

// localeCase is what locale.xlsx shows under a locale: 8 January 2026 in
// Excel's short date (numFmtId 14) and system long date ([$-F800]), and
// 1234567.891 in "#,##0.00" (4) and the currency format (7). The sheet sets no
// page size, so its paper is the locale's fallback.
type localeCase struct {
	locale string
	want   []string
	a4     bool
}

var (
	systemLocale = localeCase{"", []string{"1/8/2026", "Thursday, January 8, 2026", "1,234,567.89", "$1,234,567.89"}, true}
	germanLocale = localeCase{"de-DE", []string{"08.01.2026", "Donnerstag, 8. Januar 2026", "1.234.567,89", "1.234.567,89 €"}, true}
)

func assertLocaleConversion(t *testing.T, step string, c localeCase, opts lok.Options) {
	t.Helper()

	opts.Locale = c.locale
	out := convertFixture(t, testdataPath(t, "locale.xlsx"), opts)

	text := pdfText(t, out)
	for _, w := range c.want {
		if !strings.Contains(text, w) {
			t.Errorf("%s: text %q lacks %q", step, text, w)
		}
	}

	w, h := pdfPageSize(t, out)
	if a4 := approxEqual(w, 595.3, 1) && approxEqual(h, 841.9, 1); a4 != c.a4 {
		t.Errorf("%s: page %.1f x %.1f, want A4 %t", step, w, h, c.a4)
	}
}

// TestLocale_FormatsFollowLocale validates that the locale formats dates,
// numbers and currencies, and picks the fallback paper size.
func TestLocale_FormatsFollowLocale(t *testing.T) {
	for _, c := range []localeCase{
		systemLocale,
		germanLocale,
		{"en-GB", []string{"08/01/2026", "Thursday 8 January 2026", "1,234,567.89", "£1,234,567.89"}, true},
		{"en-AU", []string{"8/01/2026", "Thursday, 8 January 2026", "$1,234,567.89"}, true},
		{"en-US", []string{"1/8/2026", "Thursday, January 8, 2026", "$1,234,567.89"}, false},
	} {
		assertLocaleConversion(t, "locale "+c.locale, c, lok.DefaultOptions())
	}
}

// TestLocale_DoesNotLeak validates that a conversion without a locale formats
// with the system locale, whatever an earlier conversion set. LibreOfficeKit
// keeps the language of a load for the next ones.
func TestLocale_DoesNotLeak(t *testing.T) {
	assertLocaleConversion(t, "German", germanLocale, lok.DefaultOptions())
	assertLocaleConversion(t, "system after German", systemLocale, lok.DefaultOptions())
	assertLocaleConversion(t, "system again", systemLocale, lok.DefaultOptions())

	// A Language key in the filter options sets the locale too.
	viaFilter := lok.DefaultOptions()
	viaFilter.FilterOptions = "Language=de-DE"
	assertLocaleConversion(t, "German through the filter options", germanLocale, viaFilter)
	assertLocaleConversion(t, "system after the filter options", systemLocale, lok.DefaultOptions())
}

// TestLocale_Invalid validates that a value that is not a language tag fails
// before loading, so it cannot add load options.
func TestLocale_Invalid(t *testing.T) {
	opts := lok.DefaultOptions()
	opts.Locale = "en-AU,Batch=false"

	err := lok.Convert(sharedOffice, testdataPath(t, "locale.xlsx"), filepath.Join(t.TempDir(), "out.pdf"), opts)
	if !errors.Is(err, lok.ErrInvalidLocale) {
		t.Fatalf("Convert() = %v, want ErrInvalidLocale", err)
	}
}
