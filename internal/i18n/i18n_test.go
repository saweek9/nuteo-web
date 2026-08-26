package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

// writeLocale writes a tiny YAML locale file to a temp dir.
func writeLocale(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNewBundleRequiresEnglish(t *testing.T) {
	dir := t.TempDir()
	writeLocale(t, dir, "th.yaml", "greeting: สวัสดี\n")

	_, err := NewBundle(dir)
	if err == nil {
		t.Fatal("expected error when en.yaml is missing")
	}
}

func TestNewBundleLoadsAllLocales(t *testing.T) {
	dir := t.TempDir()
	writeLocale(t, dir, "en.yaml", "greeting: hello\nnav:\n  home: Home\n")
	writeLocale(t, dir, "th.yaml", "greeting: สวัสดี\nnav:\n  home: หน้าแรก\n")
	writeLocale(t, dir, "de.yaml", "greeting: hallo\n") // unused but loaded

	b, err := NewBundle(dir)
	if err != nil {
		t.Fatalf("NewBundle: %v", err)
	}
	if got := len(b.Supported()); got != 3 {
		t.Errorf("Supported: got %d want 3", got)
	}
}

func TestTReturnsTranslation(t *testing.T) {
	dir := t.TempDir()
	writeLocale(t, dir, "en.yaml", "greeting: hello\nnav:\n  home: Home\n")
	writeLocale(t, dir, "th.yaml", "greeting: สวัสดี\n")

	b, _ := NewBundle(dir)

	cases := []struct {
		lang string
		key  string
		want string
	}{
		{"en", "greeting", "hello"},
		{"en", "nav.home", "Home"},
		{"th", "greeting", "สวัสดี"},
	}
	for _, tc := range cases {
		got := b.T(tc.lang, tc.key)
		if got != tc.want {
			t.Errorf("T(%q, %q): got %q want %q", tc.lang, tc.key, got, tc.want)
		}
	}
}

func TestTFallsBackToEnglish(t *testing.T) {
	dir := t.TempDir()
	writeLocale(t, dir, "en.yaml", "greeting: hello\nonly_english: english-only\n")
	writeLocale(t, dir, "th.yaml", "greeting: สวัสดี\n") // no only_english

	b, _ := NewBundle(dir)
	got := b.T("th", "only_english")
	if got != "english-only" {
		t.Errorf("fallback: got %q want english-only", got)
	}
}

func TestTReturnsKeyIfMissing(t *testing.T) {
	dir := t.TempDir()
	writeLocale(t, dir, "en.yaml", "greeting: hello\n")

	b, _ := NewBundle(dir)
	got := b.T("en", "missing.key.path")
	if got != "missing.key.path" {
		t.Errorf("missing key: got %q want %q", got, "missing.key.path")
	}
}

func TestHas(t *testing.T) {
	dir := t.TempDir()
	writeLocale(t, dir, "en.yaml", "yes: yes\nno:\n")
	writeLocale(t, dir, "th.yaml", "yes: ใช่\n")

	b, _ := NewBundle(dir)

	cases := []struct {
		lang string
		key  string
		want bool
	}{
		{"en", "yes", true},
		{"en", "no", false},
		{"th", "yes", true},
		{"th", "no", false}, // not in th, fallback to en
	}
	for _, tc := range cases {
		got := b.Has(tc.lang, tc.key)
		if got != tc.want {
			t.Errorf("Has(%q, %q): got %v want %v", tc.lang, tc.key, got, tc.want)
		}
	}
}

func TestNegotiateQueryParam(t *testing.T) {
	dir := t.TempDir()
	writeLocale(t, dir, "en.yaml", "x: 1\n")
	writeLocale(t, dir, "th.yaml", "x: 1\n")

	b, _ := NewBundle(dir)

	cases := []struct {
		query string
		accept string
		want string
	}{
		{"th", "", "th"},
		{"en", "th", "en"},          // query wins
		{"fr", "th", "th"},          // invalid query → fallback to accept
		{"", "th-TH,en;q=0.9", "th"}, // exact match
		{"", "th-TH", "th"},         // base lang match (strip -TH)
		{"", "ja,ko;q=0.8", "en"},   // no match → default
		{"", "", "en"},              // empty → default
	}
	for _, tc := range cases {
		got := b.Negotiate(tc.query, tc.accept)
		if got != tc.want {
			t.Errorf("Negotiate(%q, %q): got %q want %q", tc.query, tc.accept, got, tc.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := []string{"EN", "th-TH", " th ", "  ", "ja-JP"}
	for _, c := range cases {
		got := normalize(c)
		_ = got // just ensure no panic
	}
}
