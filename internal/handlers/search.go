package handlers

import (
	"html/template"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// SearchResult is a single match returned to the search results partial.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
	Type    string // "service" | "project" | "post" | "page"
}

// Search handles GET /search?q=... and renders the search results.
// Designed for HTMX-driven inline search; returns a partial that
// targets #search-results.
//
// `q` empty → empty results (no error).
func (d *Deps) Search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))

	if q == "" {
		// Empty query — render full search page with prompt
		data := baseData(c, d, "Search")
		data["Query"] = ""
		data["Results"] = []SearchResult{}
		data["Count"] = 0
		renderPage(c, data)
		return
	}

	needle := strings.ToLower(q)
	var results []SearchResult

	// Services
	for _, s := range d.Store.Services {
		if matchScore(needle, s.Title, s.Summary, s.BodyMD) > 0 {
			results = append(results, SearchResult{
				Title:   s.Title,
				URL:     "/services/" + s.Slug,
				Snippet: trim(s.Summary, 140),
				Type:    "service",
			})
		}
	}

	// Portfolio
	for _, p := range d.Store.Portfolio {
		if matchScore(needle, p.Title, p.Summary, p.BodyMD) > 0 {
			results = append(results, SearchResult{
				Title:   p.Title,
				URL:     "/portfolio/" + p.Slug,
				Snippet: trim(p.Summary, 140),
				Type:    "project",
			})
		}
	}

	// Posts
	for _, post := range d.Store.Posts {
		if matchScore(needle, post.Title, post.Summary, post.BodyMD) > 0 {
			results = append(results, SearchResult{
				Title:   post.Title,
				URL:     "/blog/" + post.Slug,
				Snippet: trim(post.Summary, 140),
				Type:    "post",
			})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Type != results[j].Type {
			return typeOrder(results[i].Type) < typeOrder(results[j].Type)
		}
		return strings.ToLower(results[i].Title) < strings.ToLower(results[j].Title)
	})

	// Detect HTMX request — return only the results fragment.
	if c.GetHeader("HX-Request") == "true" {
		tmplsAny, _ := c.Get("tmpls")
		tmpls, _ := tmplsAny.(map[string]*template.Template)
		tmpl := tmpls["search_results.html"]
		if tmpl != nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			data := gin.H{
				"Query":   q,
				"Results": results,
				"Count":   len(results),
			}
			_ = tmpl.Execute(c.Writer, data)
			return
		}
	}

	// Full page (non-HTMX)
	data := baseData(c, d, "Search")
	data["Query"] = q
	data["Results"] = results
	data["Count"] = len(results)
	renderPage(c, data)
}

// matchScore returns a simple rank; 0 means no match.
//
// Title matches weigh 3×; summary 2×; body 1×. Substring match.
func matchScore(needle, title, summary, body string) int {
	score := 0
	if strings.Contains(strings.ToLower(title), needle) {
		score += 3
	}
	if strings.Contains(strings.ToLower(summary), needle) {
		score += 2
	}
	if strings.Contains(strings.ToLower(body), needle) {
		score += 1
	}
	return score
}

func typeOrder(t string) int {
	switch t {
	case "service":
		return 0
	case "project":
		return 1
	case "post":
		return 2
	default:
		return 99
	}
}

func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
