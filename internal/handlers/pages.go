package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nuteo/nuteo-web/internal/models"
	"github.com/nuteo/nuteo-web/internal/storage"
)

// Home renders the landing page.
func (d *Deps) Home(c *gin.Context) {
	data := baseData(c, d, "home")
	data["PageDescription"] = "nuteo solution — cloud migration, DevOps, backend, and security consulting."
	data["FeaturedServices"]  = d.Store.FeaturedServices()
	data["FeaturedPortfolio"] = d.Store.FeaturedPortfolio()
	renderPage(c, data)
}

// Services renders the services list page.
func (d *Deps) Services(c *gin.Context) {
	data := baseData(c, d, "Services")
	data["PageDescription"] = "End-to-end engineering and operations — from a single component to a full platform."
	data["Services"] = d.Store.Services
	renderPage(c, data)
}

// ServiceDetail renders a single service page.
func (d *Deps) ServiceDetail(c *gin.Context) {
	slug := c.Param("slug")
	svc := d.Store.ServiceBySlug(slug)
	if svc == nil {
		d.NotFound(c)
		return
	}
	data := baseData(c, d, svc.Title)
	data["PageDescription"] = svc.Summary
	data["Service"] = svc
	renderPage(c, data)
}

// Portfolio renders the portfolio list.
func (d *Deps) Portfolio(c *gin.Context) {
	data := baseData(c, d, "Portfolio")
	data["PageDescription"] = "Selected projects across cloud migration, observability, and platform engineering."
	data["Projects"] = d.Store.Portfolio
	renderPage(c, data)
}

// ProjectDetail renders a single portfolio item.
func (d *Deps) ProjectDetail(c *gin.Context) {
	slug := c.Param("slug")
	p := d.Store.PortfolioBySlug(slug)
	if p == nil {
		d.NotFound(c)
		return
	}
	data := baseData(c, d, p.Title)
	data["PageDescription"] = p.Summary
	data["Project"] = p
	renderPage(c, data)
}

// About renders the about page.
func (d *Deps) About(c *gin.Context) {
	data := baseData(c, d, "About")
	data["PageDescription"] = "About nuteo solution — Bangkok-based engineering consultancy."
	renderPage(c, data)
}

// suggestionEntry is a single "did you mean?" link shown on the 404 page.
type suggestionEntry struct {
	Title string
	URL   string
}

// NotFound renders the 404 page. We suggest up to 3 existing pages
// whose slug contains the largest common substring with the requested
// path, so users who mistyped get a useful pointer.
func (d *Deps) NotFound(c *gin.Context) {
	c.Status(http.StatusNotFound)

	requested := c.Request.URL.Path
	suggestions := suggestPaths(requested, d.Store)

	data := baseData(c, d, "Not found")
	data["PageDescription"] = "Page not found."
	data["Path"]            = requested
	data["Suggestions"]    = suggestions
	// Force the 404 template — important when invoked from a route
	// middleware that set a different page (e.g. /services/:slug).
	overridePage(c, "404.html")
	renderPage(c, data)
}

// suggestPaths returns up to 3 existing content URLs whose slug or
// path shares the longest common substring (length ≥ 3) with the
// requested path. Cheap heuristic — perfect for a static site.
func suggestPaths(requested string, store *storage.Store) []suggestionEntry {
	type entry struct {
		title string
		url   string
		score int
	}
	var matches []entry

	// Trim leading/trailing slashes so segments don't include empties.
	reqSegs := strings.FieldsFunc(strings.Trim(requested, "/-_. "), func(r rune) bool {
		return r == '/' || r == '-' || r == '_' || r == '.' || r == ' '
	})

	add := func(title, url string, segs []string) {
		best := 0
		for _, a := range reqSegs {
			if len(a) < 3 {
				continue
			}
			for _, b := range segs {
				if a == b && len(a) > best {
					best = len(a)
				}
			}
		}
		if best >= 3 {
			matches = append(matches, entry{title, url, best})
		}
	}

	for _, s := range store.Services {
		add(s.Title, "/services/"+s.Slug, strings.FieldsFunc(s.Slug, isSep))
	}
	for _, p := range store.Portfolio {
		add(p.Title, "/portfolio/"+p.Slug, strings.FieldsFunc(p.Slug, isSep))
	}
	for _, post := range store.Posts {
		add(post.Title, "/blog/"+post.Slug, strings.FieldsFunc(post.Slug, isSep))
	}
	for _, path := range []struct {
		title, url string
	}{
		{"Home", "/"},
		{"Services", "/services"},
		{"Portfolio", "/portfolio"},
		{"Blog", "/blog"},
		{"FAQ", "/faq"},
		{"Contact", "/contact"},
	} {
		segs := strings.FieldsFunc(strings.Trim(path.url, "/"), isSep)
		add(path.title, path.url, segs)
	}

	// Sort by score desc, keep top 3.
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].score > matches[i].score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
	out := make([]suggestionEntry, 0, 3)
	for i, m := range matches {
		if i >= 3 {
			break
		}
		out = append(out, suggestionEntry{Title: m.title, URL: m.url})
	}
	return out
}

func isSep(r rune) bool { return r == '/' || r == '-' || r == '_' || r == '.' || r == ' ' }

func commonPrefixLen(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

// baseData fills the template vars common to every page.
func baseData(c *gin.Context, d *Deps, title string) gin.H {
	lang := i18nLang(c, d)

	// Resolve OG / Twitter defaults.
	ogImage := d.Cfg.SiteURL + d.Cfg.LogoPath
	ogType := "website"
	articleDate := ""
	if d.Store != nil && len(c.Request.URL.Path) > 6 && c.Request.URL.Path[:6] == "/blog/" {
		if p := findPostByPath(d.Store, c.Request.URL.Path); p != nil {
			ogType = "article"
			articleDate = p.PublishedAt.Format("2006-01-02")
		}
	}
	// Twitter handle — strip leading "@" if present.
	twHandle := d.Cfg.TwitterHandle
	if strings.HasPrefix(twHandle, "@") {
		twHandle = twHandle[1:]
	}

	// Build lang-switcher HTML (EN | TH).
	switchLang := func(code, label string) string {
		cls := ""
		if strings.EqualFold(lang, code) {
			cls = ` class="active"`
		}
		titleAttr := label
		if code == "en" {
			titleAttr = "English"
		} else {
			titleAttr = "ภาษาไทย"
		}
		return fmt.Sprintf(`<a href="?lang=%s" rel="alternate" hreflang="%s"%s title="%s">%s</a>`,
			code, code, cls, titleAttr, strings.ToUpper(code))
	}
	langSwitchHTML := switchLang("en", "English") + switchLang("th", "ภาษาไทย")

	return gin.H{
		"SiteName":      d.Cfg.SiteName,
		"SiteURL":       d.Cfg.SiteURL,
		"PageTitle":     title,
		"PageDescription": "Software engineering & IT consulting from Bangkok.",
		"PageURL":       c.Request.URL.Path,
		"LogoPath":      d.Cfg.LogoPath,
		"GitHubURL":     d.Cfg.GitHubURL,
		"LinkedInURL":   d.Cfg.LinkedInURL,
		"TwitterURL":    "",
		"TwitterHandle": twHandle,
		"Year":          time.Now().Year(),
		// Translation helpers — call from templates as `{{t "nav.services" .Lang}}`
		"Lang":           lang,
		"t":              d.I18n.T,
		"HasLang":        d.I18n.Has,
		"LangSwitchHTML": template.HTML(langSwitchHTML),
		// Open Graph / Twitter
		"OGType":      ogType,
		"OGImage":     ogImage,
		"ArticleDate": articleDate,
	}
}

// findPostByPath returns the post at /blog/<slug> or nil.
func findPostByPath(store *storage.Store, urlPath string) *models.Post {
	for i := range store.Posts {
		if urlPath == "/blog/"+store.Posts[i].Slug {
			return &store.Posts[i]
		}
	}
	return nil
}


// ContentImagePath returns the relative path for serving /static/images/<name>.
// Used in markdown image links.
func ContentImagePath(name string) string {
	return filepath.ToSlash(filepath.Join("static/images", name))
}

// i18nLang picks the user's preferred language for this request.
// Priority: ?lang=th cookie > Accept-Language header > default.
func i18nLang(c *gin.Context, d *Deps) string {
	if d.I18n == nil {
		return "en"
	}
	return pickLang(c, d)
}

// pickLang is the shared language picker used by handlers and middleware.
func pickLang(c *gin.Context, d *Deps) string {
	if l := c.Query("lang"); l != "" {
		return d.I18n.Negotiate(l, "")
	}
	return d.I18n.Negotiate("", c.GetHeader("Accept-Language"))
}

// adminErrorMessage returns a localized admin error string.
// Falls back to English when i18n is not wired (during tests).
func adminErrorMessage(lang string) string {
	switch lang {
	case "th":
		return "ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง"
	default:
		return "Invalid credentials."
	}
}
