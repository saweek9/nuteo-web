package handlers

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nuteo/nuteo-web/internal/middleware"
)

// AdminLogin handles GET /admin/login (form) and POST /admin/login
// (submit).
//
// Auth model: ADMIN_USER + ADMIN_PASSWORD from env, constant-time
// comparison. On success, sets a signed session cookie and redirects
// to /admin.
//
// If admin credentials are not configured (both empty), /admin/login
// shows a "disabled" message.
func (d *Deps) AdminLogin(c *gin.Context) {
	if d.Cfg.AdminUser == "" || d.Cfg.AdminPassword == "" {
		c.Status(http.StatusForbidden)
		data := baseData(c, d, "Admin Disabled")
		data["PageDescription"] = "Admin panel is not configured."
		renderPage(c, data)
		return
	}

	switch c.Request.Method {
	case http.MethodGet:
	if d.Cfg.AdminUser == "" || d.Cfg.AdminPassword == "" {
		c.Status(http.StatusForbidden)
		data := baseData(c, d, "Admin Disabled")
		data["PageDescription"] = "Admin panel is not configured."
		renderPage(c, data)
		return
	}
	data := baseData(c, d, "Admin Login")
	data["PageDescription"] = "Sign in to manage content."
	data["Error"] = ""
	data["AdminLoginTitle"] = "Admin Login"
	data["UsernameLabel"] = "Username"
	data["PasswordLabel"] = "Password"
	data["SignInLabel"] = "Sign in"
	renderPage(c, data)

	case http.MethodPost:
	user := c.PostForm("user")
	pass := c.PostForm("pass")

	// Constant-time compare on both fields
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(d.Cfg.AdminUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(d.Cfg.AdminPassword)) == 1

	if !userOK || !passOK {
		slog.Warn("admin login failed", "user", user)
		c.Status(http.StatusUnauthorized)
		switchLang := pickLang(c, d)
		data := baseData(c, d, "Admin Login")
		data["PageDescription"] = "Sign in to manage content."
		data["Error"] = adminErrorMessage(switchLang)
		data["AdminLoginTitle"] = "Admin Login"
		data["UsernameLabel"] = "Username"
		data["PasswordLabel"] = "Password"
		data["SignInLabel"] = "Sign in"
		renderPage(c, data)
		return
	}

		// Success — set signed cookie
		expires := time.Now().Add(8 * time.Hour)
		cookie := middleware.SignSession([]byte(d.Cfg.CSRFSecret), user, expires)
		c.SetCookie("admin_session", cookie, int(8*60*60), "/admin", "", true, true)
		slog.Info("admin login", "user", user)
		c.Redirect(http.StatusFound, "/admin")
		return
	}
}

// AdminLogout clears the session cookie and redirects to /admin/login.
func (d *Deps) AdminLogout(c *gin.Context) {
	c.SetCookie("admin_session", "", -1, "/admin", "", true, true)
	c.Redirect(http.StatusFound, "/admin/login")
}

// AdminDashboard renders the admin home — quick stats + shortcuts.
func (d *Deps) AdminDashboard(c *gin.Context) {
	data := baseData(c, d, "Admin")
	data["PageDescription"] = "Manage content and site settings."
	data["AdminUser"] = middleware.AdminUser(c)
	data["ServiceCount"] = len(d.Store.Services)
	data["PortfolioCount"] = len(d.Store.Portfolio)
	data["PostCount"] = len(d.Store.Posts)
	data["SubscriberCount"] = SubscriberCount()

	// Sort content by updated_at desc
	var recent []contentItem
	for _, s := range d.Store.Services {
		recent = append(recent, contentItem{
			Title:     s.Title,
			URL:       "/services/" + s.Slug,
			Type:      "service",
			UpdatedAt: s.UpdatedAt,
		})
	}
	for _, p := range d.Store.Portfolio {
		recent = append(recent, contentItem{
			Title:     p.Title,
			URL:       "/portfolio/" + p.Slug,
			Type:      "project",
			UpdatedAt: p.PublishedAt,
		})
	}
	for _, post := range d.Store.Posts {
		recent = append(recent, contentItem{
			Title:     post.Title,
			URL:       "/blog/" + post.Slug,
			Type:      "post",
			UpdatedAt: post.UpdatedAt,
		})
	}
	sort.SliceStable(recent, func(i, j int) bool {
		return recent[i].UpdatedAt.After(recent[j].UpdatedAt)
	})
	if len(recent) > 10 {
		recent = recent[:10]
	}
	data["Recent"] = recent
	renderPage(c, data)
}

// contentItem is a unified view used in the admin dashboard's
// "recent activity" list.
type contentItem struct {
	Title     string
	URL       string
	Type      string
	UpdatedAt time.Time
}
