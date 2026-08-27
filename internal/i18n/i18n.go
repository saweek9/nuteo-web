// Package i18n — minimal translation loader and accessor.
//
// Strategy:
//   - YAML files in i18n/<lang>.yaml, loaded once at startup
//   - "en" is the default; falls back to English if a key is missing
//   - Per-request language selected via Accept-Language or ?lang=...
//   - Template function `t` returns translated string by dotted key
package i18n

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Bundle holds all loaded translations keyed by language code.
type Bundle struct {
	dir     string
	locales map[string]map[string]any // lang → nested map
	mu      sync.RWMutex             // protects locales
}

// NewBundle loads every *.yaml file under dir. Each filename stem
// becomes a locale code (e.g. "en.yaml" → "en").
func NewBundle(dir string) (*Bundle, error) {
	b := &Bundle{
		dir:     dir,
		locales: make(map[string]map[string]any),
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("i18n: read dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		lang := strings.TrimSuffix(e.Name(), ".yaml")
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("i18n: read %s: %w", path, err)
		}
		var m map[string]any
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("i18n: parse %s: %w", path, err)
		}
		b.locales[lang] = m
	}
	if _, ok := b.locales["en"]; !ok {
		return nil, fmt.Errorf("i18n: en.yaml is required (default locale)")
	}
	slog.Info("i18n loaded", "locales", len(b.locales), "dir", dir)
	return b, nil
}

// Default returns the English locale (fallback).
func (b *Bundle) Default() string {
	return "en"
}

// Supported returns the list of loaded locale codes.
func (b *Bundle) Supported() []string {
	out := make([]string, 0, len(b.locales))
	for k := range b.locales {
		out = append(out, k)
	}
	return out
}

// T returns the translation for `key` in locale `lang`.
// If key not found, falls back to English; if still missing,
// returns the key itself (so missing keys are visible in output).
//
// Key format: "nav.services" → nested lookup.
func (b *Bundle) T(lang, key string) string {
	if v := lookup(b.locales[lang], key); v != "" {
		return v
	}
	if lang != b.Default() {
		if v := lookup(b.locales[b.Default()], key); v != "" {
			return v
		}
	}
	return key // visible marker for missing translation
}

// Has reports whether a non-empty translation exists for the given
// language and key.
func (b *Bundle) Has(lang, key string) bool {
	return lookup(b.locales[lang], key) != ""
}

// RawMap returns the underlying parsed YAML tree for a locale so that
// callers (e.g. the FAQ handler) can walk structured data like
// `faq.entries: [{q, a}, ...]` without needing a custom decoder.
//
// Returns nil if the locale is unknown.
func (b *Bundle) RawMap(lang string) map[string]any {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.locales[lang]
}

// lookup walks the nested map by dot-separated key.
func lookup(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	parts := strings.Split(key, ".")
	var cur any = m
	for _, p := range parts {
		asMap, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = asMap[p]
		if !ok {
			return ""
		}
	}
	if s, ok := cur.(string); ok {
		return s
	}
	return ""
}

// Negotiate picks the best locale for a request.
//
// Resolution order:
//  1. Explicit ?lang=th query param (validated against supported)
//  2. Accept-Language header (best match against supported)
//  3. Default
func (b *Bundle) Negotiate(queryLang, acceptLang string) string {
	if queryLang != "" {
		lang := normalize(queryLang)
		if _, ok := b.locales[lang]; ok {
			return lang
		}
	}
	// Parse Accept-Language: "th,en;q=0.9,*;q=0.5"
	for _, raw := range strings.Split(acceptLang, ",") {
		lang := strings.TrimSpace(strings.SplitN(raw, ";", 2)[0])
		lang = normalize(lang)
		if lang == "" {
			continue
		}
		if _, ok := b.locales[lang]; ok {
			return lang
		}
		// Try base language (e.g. "th-TH" → "th")
		base := strings.SplitN(lang, "-", 2)[0]
		if _, ok := b.locales[base]; ok {
			return base
		}
	}
	return b.Default()
}

func normalize(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}
