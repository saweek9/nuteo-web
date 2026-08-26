package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTemp writes content to a temp dir and returns the path.
func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestParseMarkdownFile(t *testing.T) {
	const body = `---
title: Test Service
slug: test-service
summary: A test service for unit tests.
order: 5
audience: [enterprise, startup]
tags: [aws, go]
featured: true
---

## What we do

This is the body of the test service.

- Item 1
- Item 2
`
	path := writeTemp(t, "test.md", body)

	v := &serviceModel{}
	if err := parseMarkdownFile(path, v); err != nil {
		t.Fatalf("parseMarkdownFile: %v", err)
	}

	if v.Title != "Test Service" {
		t.Errorf("Title: got %q want %q", v.Title, "Test Service")
	}
	if v.Slug != "test-service" {
		t.Errorf("Slug: got %q want %q", v.Slug, "test-service")
	}
	if v.Summary != "A test service for unit tests." {
		t.Errorf("Summary: got %q", v.Summary)
	}
	if v.Order != 5 {
		t.Errorf("Order: got %d want %d", v.Order, 5)
	}
	if len(v.Audience) != 2 || v.Audience[0] != "enterprise" || v.Audience[1] != "startup" {
		t.Errorf("Audience: got %v", v.Audience)
	}
	if len(v.Tags) != 2 || v.Tags[0] != "aws" || v.Tags[1] != "go" {
		t.Errorf("Tags: got %v", v.Tags)
	}
	if !v.Featured {
		t.Error("Featured: got false want true")
	}
	if v.BodyHTML == "" {
		t.Error("BodyHTML: got empty")
	}
	if !contains(v.BodyHTML, "<h2>What we do</h2>") {
		t.Errorf("BodyHTML missing rendered h2: %q", v.BodyHTML)
	}
	if !contains(v.BodyHTML, "<li>Item 1</li>") {
		t.Errorf("BodyHTML missing rendered list: %q", v.BodyHTML)
	}
	if !contains(v.BodyMD, "This is the body") {
		t.Errorf("BodyMD: %q", v.BodyMD)
	}
}

func TestParseMarkdownFileMissingDelimiter(t *testing.T) {
	const body = `---
title: Bad
slug: bad
# no closing ---
`
	path := writeTemp(t, "bad.md", body)
	v := &serviceModel{}
	err := parseMarkdownFile(path, v)
	if err == nil {
		t.Fatal("expected error for missing closing ---")
	}
}

func TestParseMarkdownFileInvalidYAML(t *testing.T) {
	const body = `---
title: [invalid: yaml
slug: x
---
body
`
	path := writeTemp(t, "invalid.md", body)
	v := &serviceModel{}
	err := parseMarkdownFile(path, v)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestStoreLoadAll(t *testing.T) {
	root := t.TempDir()
	svcDir := filepath.Join(root, "services")
	portDir := filepath.Join(root, "portfolio")
	postDir := filepath.Join(root, "posts")
	for _, d := range []string{svcDir, portDir, postDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	os.WriteFile(filepath.Join(svcDir, "a.md"),
		[]byte("---\ntitle: A\nslug: a\norder: 2\n---\nA body"), 0644)
	os.WriteFile(filepath.Join(svcDir, "b.md"),
		[]byte("---\ntitle: B\nslug: b\norder: 1\nfeatured: true\n---\nB body"), 0644)
	os.WriteFile(filepath.Join(portDir, "x.md"),
		[]byte("---\ntitle: X\nslug: x\n---\nX body"), 0644)
	os.WriteFile(filepath.Join(postDir, "p.md"),
		[]byte("---\ntitle: P\nslug: p\n---\nP body"), 0644)

	s := New()
	if err := s.LoadAll(root); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if got := len(s.Services); got != 2 {
		t.Errorf("Services: got %d want 2", got)
	}
	// Order should put b (order=1) before a (order=2)
	if s.Services[0].Slug != "b" || s.Services[1].Slug != "a" {
		t.Errorf("Order: got %s then %s", s.Services[0].Slug, s.Services[1].Slug)
	}

	if got := len(s.Portfolio); got != 1 {
		t.Errorf("Portfolio: got %d want 1", got)
	}
	if got := len(s.Posts); got != 1 {
		t.Errorf("Posts: got %d want 1", got)
	}

	// Lookups
	if svc := s.ServiceBySlug("a"); svc == nil {
		t.Error("ServiceBySlug(a): got nil")
	} else if svc.Title != "A" {
		t.Errorf("ServiceBySlug(a).Title: got %q", svc.Title)
	}

	if svc := s.ServiceBySlug("missing"); svc != nil {
		t.Errorf("ServiceBySlug(missing): got %+v, want nil", svc)
	}

	// FeaturedServices — only 'b' has featured: true
	featured := s.FeaturedServices()
	if len(featured) != 1 || featured[0].Slug != "b" {
		t.Errorf("FeaturedServices: got %+v", featured)
	}

	// Missing dir is OK (not an error)
	root2 := t.TempDir()
	s2 := New()
	if err := s2.LoadAll(root2); err != nil {
		t.Errorf("LoadAll with empty dir: %v", err)
	}
}

func TestSetHTMLMethods(t *testing.T) {
	// Embedded struct: setHTML must mutate the inner Service.BodyHTML.
	const body = "---\ntitle: X\nslug: x\n---\nbody"
	path := writeTemp(t, "x.md", body)

	v := &serviceModel{}
	if err := parseMarkdownFile(path, v); err != nil {
		t.Fatal(err)
	}
	if v.BodyHTML == "" {
		t.Error("BodyHTML not set after parseMarkdownFile")
	}

	// Same for portfolio + post wrappers.
	pPath := writeTemp(t, "p.md", body)
	pv := &portfolioModel{}
	if err := parseMarkdownFile(pPath, pv); err != nil {
		t.Fatal(err)
	}
	if pv.BodyHTML == "" {
		t.Error("portfolio BodyHTML not set")
	}

	postPath := writeTemp(t, "post.md", body)
	post := &postModel{}
	if err := parseMarkdownFile(postPath, post); err != nil {
		t.Fatal(err)
	}
	if post.BodyHTML == "" {
		t.Error("post BodyHTML not set")
	}
}

// contains is a tiny strings.Contains to avoid importing strings in every test.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
