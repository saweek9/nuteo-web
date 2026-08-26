package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

// NotFound renders the 404 page.
func (d *Deps) NotFound(c *gin.Context) {
	c.Status(http.StatusNotFound)
	data := baseData(c, d, "Not found")
	data["PageDescription"] = "Page not found."
	renderPage(c, data)
}

// baseData fills the template vars common to every page.
func baseData(c *gin.Context, d *Deps, title string) gin.H {
	lang := i18nLang(c, d)

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
		return fmt.Sprintf(`<a href="?lang=%s"%s title="%s">%s</a>`,
			code, cls, titleAttr, strings.ToUpper(code))
	}
	langSwitchHTML := switchLang("en", "English") + switchLang("th", "ภาษาไทย")

	return gin.H{
		"SiteName":    d.Cfg.SiteName,
		"SiteURL":     d.Cfg.SiteURL,
		"PageTitle":   title,
		"PageURL":     c.Request.URL.Path,
		"LogoPath":    d.Cfg.LogoPath,
		"GitHubURL":   d.Cfg.GitHubURL,
		"LinkedInURL": d.Cfg.LinkedInURL,
		"TwitterURL":  "",
		"Year":        time.Now().Year(),
		// Translation helpers — call from templates as `{{t "nav.services" .Lang}}`
		"Lang":           lang,
		"t":              d.I18n.T,
		"HasLang":        d.I18n.Has,
		"LangSwitchHTML": template.HTML(langSwitchHTML),
	}
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
