// Package storage — markdown parsing helpers.
package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"

	"github.com/nuteo/nuteo-web/internal/models"
)

// markdownConverter is a configured goldmark instance (GFM, autolinks).
var markdownConverter = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Linkify),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

// parseMarkdownFile reads a .md file, splits YAML frontmatter from body,
// unmarshals frontmatter into v, and renders body to HTML stored in v.BodyHTML
// (and raw markdown in v.BodyMD). v must be one of the *serviceModel /
// *portfolioModel / *postModel wrapper pointers (already-pointer, no & needed).
func parseMarkdownFile(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	parts := bytes.SplitN(raw, []byte("\n---\n"), 2)
	if len(parts) != 2 {
		return fmt.Errorf("missing second --- delimiter in %s", path)
	}
	fm := bytes.TrimPrefix(parts[0], []byte("---\n"))
	bodyMD := parts[1]

	if err := yaml.Unmarshal(fm, v); err != nil {
		return fmt.Errorf("yaml: %w", err)
	}

	// Default PublishedAt if zero
	type publishedAter interface{ SetPublishedIfZero(time.Time) }
	if p, ok := v.(publishedAter); ok {
		p.SetPublishedIfZero(time.Now().UTC())
	}

	// Render markdown
	var htmlBuf bytes.Buffer
	if err := markdownConverter.Convert(bodyMD, &htmlBuf); err != nil {
		return fmt.Errorf("markdown: %w", err)
	}

	type htmlSink interface {
		SetHTML(string)
		SetMD(string)
	}
	if h, ok := v.(htmlSink); ok {
		h.SetHTML(htmlBuf.String())
		h.SetMD(string(bodyMD))
	}
	return nil
}

// loadMarkdownDir loads every *.md file in dir into a slice via the
// newFn callback that returns a fresh wrapper struct for each file.
func loadMarkdownDir[T any](dir string, newFn func() T) ([]T, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []T{}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []T{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		v := newFn()
		if err := parseMarkdownFile(filepath.Join(dir, e.Name()), v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// --- Wrapper types ---
// We can't define methods on imported types in Go, so each model is wrapped
// in a local struct that embeds it and adds the storage hooks.

type serviceModel struct {
	models.Service `yaml:",inline"`
}

func (s *serviceModel) SetHTML(h string)  { s.Service.BodyHTML = h }
func (s *serviceModel) SetMD(m string)    { s.Service.BodyMD = m }
func (s *serviceModel) SetPublishedIfZero(t time.Time) {
	if s.Service.PublishedAt.IsZero() { s.Service.PublishedAt = t }
}

type portfolioModel struct {
	models.Portfolio `yaml:",inline"`
}

func (p *portfolioModel) SetHTML(h string)  { p.Portfolio.BodyHTML = h }
func (p *portfolioModel) SetMD(m string)    { p.Portfolio.BodyMD = m }
func (p *portfolioModel) SetPublishedIfZero(t time.Time) {
	if p.Portfolio.PublishedAt.IsZero() { p.Portfolio.PublishedAt = t }
}

type postModel struct {
	models.Post `yaml:",inline"`
}

func (p *postModel) SetHTML(h string)  { p.Post.BodyHTML = h }
func (p *postModel) SetMD(m string)    { p.Post.BodyMD = m }
func (p *postModel) SetPublishedIfZero(t time.Time) {
	if p.Post.PublishedAt.IsZero() { p.Post.PublishedAt = t }
}
