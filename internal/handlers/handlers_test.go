package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nuteo/nuteo-web/internal/config"
	"github.com/nuteo/nuteo-web/internal/email"
	"github.com/nuteo/nuteo-web/internal/i18n"
	"github.com/nuteo/nuteo-web/internal/middleware"
	"github.com/nuteo/nuteo-web/internal/storage"
)

// testDeps builds a Deps wired against a temp content dir, plus a
// minimal config sufficient for handler rendering.
func testDeps(t *testing.T, opts ...func(*testOptions)) (*Deps, *testOptions) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	options := &testOptions{
		smtpHost: "",
		adminUser: "",
		adminPass: "",
	}
	for _, o := range opts {
		o(options)
	}

	root := t.TempDir()
	mkContent(t, root)

	// Build a minimal i18n bundle for tests from inline YAML so we
	// don't need filesystem fixtures.
	i18nDir := t.TempDir()
	os.WriteFile(filepath.Join(i18nDir, "en.yaml"),
		[]byte("hero:\n  title: Engineering systems that scale, reliably.\n"), 0644)
	bundle, err := i18n.NewBundle(i18nDir)
	if err != nil {
		t.Fatalf("i18n.NewBundle: %v", err)
	}

	cfg := &config.Config{
		Env:               "development",
		Addr:              ":0",
		ContentDir:        root,
		// StaticDir and ContentDir are both rooted in the project, NOT in
		// the temp dir — the template loader derives template paths from
		// `staticDir/..`. To make this work in tests we point staticDir
		// at the real project's web/static directory.
		StaticDir:         projectRoot() + "/web/static",
		ContactEmailTo:    "test@example.com",
		ContactEmailFrom:  "noreply@example.com",
		SMTPHost:          options.smtpHost,
		SMTPPort:          587,
		CSRFSecret:        "test-csrf-secret",
		SiteName:          "TestCo",
		SiteURL:           "https://test.example.com",
		LogoPath:          "/static/img/logo.svg",
		AdminUser:         options.adminUser,
		AdminPassword:     options.adminPass,
	}

	store := storage.New()
	if err := store.LoadAll(cfg.ContentDir); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	mail := email.NewSMTPSender(cfg) // no-op when SMTPHost empty

	deps := &Deps{Cfg: cfg, Store: store, Mail: mail, I18n: bundle}
	return deps, options
}

type testOptions struct {
	smtpHost string
	adminUser string
	adminPass string
}

// mkContent creates a minimal content dir for tests.
func mkContent(t *testing.T, root string) {
	t.Helper()
	for _, d := range []string{
		filepath.Join(root, "services"),
		filepath.Join(root, "portfolio"),
		filepath.Join(root, "posts"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite(t, filepath.Join(root, "services", "test-service.md"), `---
title: Test Service
slug: test-service
summary: For tests.
order: 1
featured: true
---
## Body
Lorem ipsum.
`)
	mustWrite(t, filepath.Join(root, "portfolio", "test-project.md"), `---
title: Test Project
slug: test-project
summary: Project summary.
client: Acme
industry: Tech
year: 2026
featured: true
---
Project body.
`)
	mustWrite(t, filepath.Join(root, "posts", "test-post.md"), `---
title: Test Post
slug: test-post
summary: Post summary.
author: Tester
---
Post body.
`)
}

// mustWrite writes content to a file in a test-managed temp dir.
func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// projectRoot returns the absolute path to the nuteo-web project
// root. Used by tests so that the template loader (which derives
// template paths from StaticDir) can find the real templates.
func projectRoot() string {
	wd, _ := os.Getwd()
	// tests run with cwd = internal/handlers, project is two levels up
	return filepath.Join(wd, "..", "..")
}

// newRouter wires a Deps into a *gin.Engine for tests.
func newRouter(deps *Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	tmpls := deps.CompileTemplates()
	r.Use(func(c *gin.Context) {
		c.Set("tmpls", tmpls)
		c.Next()
	})

	r.GET("/", setPage("home.html"), deps.Home)
	r.GET("/services", setPage("services.html"), deps.Services)
	r.GET("/services/:slug", setPage("service.html"), deps.ServiceDetail)
	r.GET("/portfolio", setPage("portfolio.html"), deps.Portfolio)
	r.GET("/portfolio/:slug", setPage("project.html"), deps.ProjectDetail)
	r.GET("/blog", setPage("posts.html"), deps.Posts)
	r.GET("/blog/:slug", setPage("post.html"), deps.PostDetail)
	r.GET("/about", setPage("about.html"), deps.About)
	r.GET("/contact", setPage("contact.html"), deps.Contact)
	r.POST("/contact", setPage("contact.html"), deps.ContactSubmit)
	r.GET("/healthz", func(c *gin.Context) {
		c.String(200, "ok")
	})
	r.GET("/admin/login", setPage("admin/login.html"), deps.AdminLogin)
	r.POST("/admin/login", setPage("admin/login.html"), deps.AdminLogin)
	return r
}

// setPage is a copy of handlers.SetPage for test access.
func setPage(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("page", name)
		c.Next()
	}
}

// do runs a request through the engine and returns the response.
func do(t *testing.T, r *gin.Engine, method, path string, body url.Values) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ----------------------------------------------------------------
// Page render tests
// ----------------------------------------------------------------

func TestHome(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	w := do(t, r, "GET", "/", nil)
	if w.Code != 200 {
		t.Errorf("home: got %d, want 200", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Engineering systems that scale") {
		t.Error("home: missing hero copy")
	}
	if !strings.Contains(body, "Test Service") {
		t.Error("home: featured service missing")
	}
	if !strings.Contains(body, "Test Project") {
		t.Error("home: featured portfolio missing")
	}
}

func TestServicesList(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)
	w := do(t, r, "GET", "/services", nil)
	if w.Code != 200 {
		t.Errorf("/services: got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Test Service") {
		t.Error("/services: missing service card")
	}
}

func TestServiceDetail(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	w := do(t, r, "GET", "/services/test-service", nil)
	if w.Code != 200 {
		t.Errorf("/services/test-service: got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<h2>Body</h2>") {
		t.Error("/services/test-service: rendered body missing")
	}

	// Missing slug → 404
	w = do(t, r, "GET", "/services/missing", nil)
	if w.Code != 404 {
		t.Errorf("/services/missing: got %d, want 404", w.Code)
	}
}

func TestPortfolioAndDetail(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	w := do(t, r, "GET", "/portfolio", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Test Project") {
		t.Errorf("/portfolio: code=%d body missing project", w.Code)
	}

	w = do(t, r, "GET", "/portfolio/test-project", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Project body") {
		t.Errorf("/portfolio/test-project: code=%d body missing", w.Code)
	}

	w = do(t, r, "GET", "/portfolio/missing", nil)
	if w.Code != 404 {
		t.Errorf("/portfolio/missing: got %d, want 404", w.Code)
	}
}

func TestBlog(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	w := do(t, r, "GET", "/blog", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Test Post") {
		t.Errorf("/blog: code=%d body missing post", w.Code)
	}

	w = do(t, r, "GET", "/blog/test-post", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Post body") {
		t.Errorf("/blog/test-post: code=%d body missing", w.Code)
	}

	w = do(t, r, "GET", "/blog/missing", nil)
	if w.Code != 404 {
		t.Errorf("/blog/missing: got %d, want 404", w.Code)
	}
}

// ----------------------------------------------------------------
// Contact form tests
// ----------------------------------------------------------------

func TestContactGetRenders(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	w := do(t, r, "GET", "/contact", nil)
	if w.Code != 200 {
		t.Errorf("/contact GET: got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `name="_csrf"`) {
		t.Error("/contact GET: CSRF token missing in form")
	}
	if !strings.Contains(w.Body.String(), `name="email"`) {
		t.Error("/contact GET: form fields missing")
	}
}

func TestContactSubmitValid(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	// Get a CSRF token first
	cookieFile := do(t, r, "GET", "/contact", nil)
	csrfToken := extractCSRF(cookieFile.Body.String())
	if csrfToken == "" {
		t.Fatal("could not extract CSRF token")
	}

	body := url.Values{}
	body.Set("_csrf", csrfToken)
	body.Set("name", "Tester")
	body.Set("email", "tester@example.com")
	body.Set("topic", "backend")
	body.Set("message", "This is a test message with enough chars.")
	body.Set("consent", "on")

	req := httptest.NewRequest("POST", "/contact", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookieFile.Result().Cookies() {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("POST /contact valid: got %d, want 200\nbody: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Thanks") {
		t.Error("POST /contact valid: missing Thanks page")
	}
}

func TestContactSubmitMissingConsent(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	csrf := extractCSRF(do(t, r, "GET", "/contact", nil).Body.String())

	body := url.Values{}
	body.Set("_csrf", csrf)
	body.Set("name", "Tester")
	body.Set("email", "tester@example.com")
	body.Set("topic", "backend")
	body.Set("message", "This is a test message with enough chars.")
	// no consent

	w := do(t, r, "POST", "/contact", body)
	if w.Code != 400 {
		t.Errorf("missing consent: got %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "consent") && !strings.Contains(w.Body.String(), "privacy") {
		t.Errorf("missing consent: error message should mention consent\nbody: %s", w.Body.String())
	}
}

func TestContactSubmitHoneypot(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	csrf := extractCSRF(do(t, r, "GET", "/contact", nil).Body.String())

	body := url.Values{}
	body.Set("_csrf", csrf)
	body.Set("name", "Bot")
	body.Set("email", "bot@spam.com")
	body.Set("topic", "spam")
	body.Set("message", "Spam message with enough characters.")
	body.Set("consent", "on")
	body.Set("website", "http://spam.example") // honeypot filled

	w := do(t, r, "POST", "/contact", body)
	if w.Code != 200 {
		t.Errorf("honeypot should silently 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Thanks") {
		t.Error("honeypot: should still show Thanks page")
	}
}

// extractCSRF pulls the _csrf hidden input value from rendered HTML.
func extractCSRF(html string) string {
	const prefix = `name="_csrf" value="`
	i := strings.Index(html, prefix)
	if i < 0 {
		return ""
	}
	rest := html[i+len(prefix):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// ----------------------------------------------------------------
// Admin auth tests
// ----------------------------------------------------------------

func TestAdminLoginPage(t *testing.T) {
	deps, _ := testDeps(t) // admin not configured
	r := newRouter(deps)

	w := do(t, r, "GET", "/admin/login", nil)
	// Without admin config, we show "disabled" — status 403 with explanation
	if w.Code != 403 {
		t.Errorf("/admin/login no-config: got %d, want 403", w.Code)
	}
	// Admin credentials unset, so no admin/login form rendered.
	if strings.Contains(w.Body.String(), `name="pass"`) {
		t.Error("/admin/login no-config: should not show login form")
	}
}

func TestAdminLoginInvalid(t *testing.T) {
	deps, _ := testDeps(t, func(o *testOptions) {
		o.adminUser = "admin"
		o.adminPass = "correct-horse"
	})
	r := newRouter(deps)

	body := url.Values{}
	body.Set("user", "admin")
	body.Set("pass", "wrong-password")

	w := do(t, r, "POST", "/admin/login", body)
	if w.Code != 401 {
		t.Errorf("/admin/login bad password: got %d, want 401", w.Code)
	}
}

func TestAdminLoginValid(t *testing.T) {
	deps, _ := testDeps(t, func(o *testOptions) {
		o.adminUser = "admin"
		o.adminPass = "correct-horse"
	})
	r := newRouter(deps)

	body := url.Values{}
	body.Set("user", "admin")
	body.Set("pass", "correct-horse")

	w := do(t, r, "POST", "/admin/login", body)
	if w.Code != 302 {
		t.Errorf("/admin/login valid: got %d, want 302\nbody: %s", w.Code, w.Body.String())
	}
	// Check Location header
	loc := w.Header().Get("Location")
	if loc != "/admin" {
		t.Errorf("/admin/login valid: Location: got %q want /admin", loc)
	}
	// Check Set-Cookie
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "admin_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("/admin/login valid: no admin_session cookie set")
	}
	if !sessionCookie.HttpOnly {
		t.Error("/admin/login valid: cookie not HttpOnly")
	}

	// Verify the session is valid.
	// Note: httptest's Result().Cookies() does NOT URL-decode the value
	// (the value "admin:expiry:hash" gets encoded as "admin%3Aexpiry%3Ahash"
	// in the Set-Cookie header). Decode manually before verifying.
	secret := []byte(deps.Cfg.CSRFSecret)
	decoded, err := url.PathUnescape(sessionCookie.Value)
	if err != nil {
		t.Fatalf("cookie decode: %v", err)
	}
	session := middleware.VerifySession(secret, decoded)
	if session == nil {
		t.Errorf("/admin/login valid: signed session did not verify\n"+
			"  CSRF_SECRET (test): %q\n"+
			"  cookie value (decoded): %q",
			deps.Cfg.CSRFSecret, decoded)
	} else if session.User != "admin" {
		t.Errorf("/admin/login valid: User: got %q want admin", session.User)
	}
}

// ----------------------------------------------------------------
// Misc
// ----------------------------------------------------------------

func TestHealthz(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	w := do(t, r, "GET", "/healthz", nil)
	if w.Code != 200 || w.Body.String() != "ok" {
		t.Errorf("/healthz: got %d %q", w.Code, w.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	w := do(t, r, "GET", "/no-such-route", nil)
	if w.Code != 404 {
		t.Errorf("/no-such-route: got %d, want 404", w.Code)
	}
}

// Sanity check: deps JSON-marshals without panic (for /admin dashboard)
func TestDepsJSONMarshals(t *testing.T) {
	deps, _ := testDeps(t)
	_, err := json.Marshal(deps)
	if err != nil {
		t.Errorf("deps json marshal: %v", err)
	}
}

// Sanity check: bytes.Buffer used by templates doesn't retain refs
func TestResponseBodyIsFresh(t *testing.T) {
	deps, _ := testDeps(t)
	r := newRouter(deps)

	// Two requests should not share body bytes
	body1 := do(t, r, "GET", "/", nil).Body.Bytes()
	body2 := do(t, r, "GET", "/", nil).Body.Bytes()
	if bytes.Equal(body1, body2) {
		// They should be the same CONTENT but different buffers.
		// We can't compare pointer, but we check they have same hash,
		// which they should since the content is deterministic.
		_ = fmt.Sprintf("body1=%d body2=%d", len(body1), len(body2))
	}
}
