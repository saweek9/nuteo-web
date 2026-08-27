// Package handlers — template loading and base renderer.
package handlers

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var _ = http.StatusOK // keep import for compile; remove if unused

// partialFiles lists the header/footer partials included into every page
// template set. These live under web/templates/partials/ and must use
// {{define "header"}} and {{define "footer"}}.
var partialFiles = []string{
	"header.html",
	"footer.html",
	"search_results.html",
	"404_illustration.svg.html",
	"cookie_banner.html",
	"og_meta.html",
}

// pageFiles lists every page template.
// Add new pages here and create the matching web/templates/pages/<name>.html
// file using `{{define "header"}}...{{end}}` / `{{define "footer"}}...{{end}}`
// at the top of the file so the partials are available to each page.
var pageFiles = []string{
	"home.html",
	"services.html",
	"service.html",
	"portfolio.html",
	"project.html",
	"about.html",
	"contact.html",
	"contact_thanks.html",
	"404.html",
	"posts.html",
	"post.html",
	"faq.html",
	"search.html",
	"admin/login.html",
	"admin/dashboard.html",
}

// loadTemplates parses all HTML templates and returns a *template.Template
// ready to render individual page templates.
//
// Strategy: each page is a self-contained template. To avoid Go's
// global-scope {{define}} clobbering across files, we load each page
// into its own template set keyed by page name.
//
// The header/footer partials are parsed into a shared set first, then
// each page is cloned+parsed into that set so they can {{template "header"}}
// without conflict.
func (d *Deps) loadTemplates() map[string]*template.Template {
	sets := make(map[string]*template.Template, len(pageFiles))

	// Shared funcs map. We register a placeholder for `t` and replace it
	// after we have access to the i18n bundle (see CompileTemplates).
	funcs := template.FuncMap{
		"safeHTML":  func(s string) template.HTML { return template.HTML(s) },
		"safeJS":    func(s string) template.JS { return template.JS(s) },
		"Year":      func() int { return time.Now().Year() },
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"join":      strings.Join,
		"hasPrefix": strings.HasPrefix,
		"contains":  strings.Contains,
		"replace":   strings.ReplaceAll,
		"trim":      strings.TrimSpace,
		"default": func(def, v any) any {
			if v == nil {
				return def
			}
			if s, ok := v.(string); ok && s == "" {
				return def
			}
			return v
		},
		"substr": func(s string, start, length int) string {
			if start < 0 || start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"isActive": isActive,
	}

	// Translation function — closes over the i18n bundle.
	// Falls back to key itself if the bundle is nil (during tests).
	funcs["t"] = func(key, lang string) string {
		if d.I18n == nil {
			return key
		}
		return d.I18n.T(lang, key)
	}

	// List helpers — Go templates have no built-in list type.
	// makeList() returns an empty []any; appendTo(list, item) returns a new
	// list with item appended. Used by the search-results partial to
	// group results by type.
	funcs["makeList"] = func() []any { return []any{} }
	funcs["appendTo"] = func(list []any, item any) []any {
		return append(list, item)
	}

	// Date formatters — wrap exported helpers so templates can call
	// them with `{{FormatDate .Post.PublishedAt .Lang}}`.
	funcs["FormatDate"] = func(t time.Time, lang string) string {
		return FormatDate(t, lang)
	}
	funcs["RelativeDate"] = func(t time.Time, lang string) string {
		return RelativeDate(t, lang)
	}

	for _, page := range pageFiles {
		tmpl := template.New(page).Funcs(funcs)
		// Clone-with-parts: parse partials into each set so the page
		// can reference headers/footers by name.
		for _, p := range partialFiles {
			if _, err := tmpl.ParseFiles(filepath.Join(d.Cfg.StaticDir, "..", "templates", "partials", p)); err != nil {
				log.Fatalf("template partial %s for %s: %v", p, page, err)
			}
		}
		// Add the page itself.
		if _, err := tmpl.ParseFiles(filepath.Join(d.Cfg.StaticDir, "..", "templates", "pages", page)); err != nil {
			log.Fatalf("template page %s: %v", page, err)
		}
		sets[page] = tmpl
	}
	return sets
}

// CompileTemplates exposes template compilation to main.go at startup.
func (d *Deps) CompileTemplates() map[string]*template.Template {
	return d.loadTemplates()
}

// tplMiddleware injects the template set for the current page into the
// request context. Pages are looked up by their declared name in the route.
//
// Use: r.GET("/about", tplMiddleware("about.html"), AboutHandler)
func (d *Deps) tplMiddleware(page string) gin.HandlerFunc {
	tmpls := d.loadTemplates()
	tmpl, ok := tmpls[page]
	if !ok {
		log.Fatalf("page template not registered: %s", page)
	}
	return func(c *gin.Context) {
		c.Set("tmpl", tmpl)
		c.Set("page", page)
		c.Next()
	}
}

// renderPage looks up the right template set for this route based on the
// page name registered by handlers.SetPage middleware (default) or overridden
// by the handler via setPage(c, "page.html").
func renderPage(c *gin.Context, data gin.H) {
	tmplsAny, _ := c.Get("tmpls")
	tmpls, ok := tmplsAny.(map[string]*template.Template)
	if !ok {
		c.String(http.StatusInternalServerError, "templates not loaded")
		return
	}
	page, _ := c.Get("page")
	pageStr, ok := page.(string)
	if !ok {
		c.String(http.StatusInternalServerError, "page not set on context")
		return
	}
	tmpl := tmpls[pageStr]
	if tmpl == nil {
		c.String(http.StatusInternalServerError, "template %q not found", pageStr)
		return
	}
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		log.Printf("template exec: %v", err)
	}
}

// overridePage lets a handler swap the active template mid-request,
// e.g. for form errors that should re-render the form page.
func overridePage(c *gin.Context, name string) {
	c.Set("page", name)
}

// SetPage is a small middleware that records which template to render.
// Use: r.GET("/about", handlers.SetPage("about.html"), deps.About)
func SetPage(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("page", name)
		c.Next()
	}
}

// isActive returns "active" when current URL matches path — used by nav.
func isActive(currentPath, linkPath string) string {
	if currentPath == linkPath {
		return "active"
	}
	// Treat /services/foo as /services too.
	if linkPath != "/" && strings.HasPrefix(currentPath, linkPath+"/") {
		return "active"
	}
	return ""
}
