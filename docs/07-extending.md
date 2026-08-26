# Extending nuteo-web

How to add features, swap implementations, and grow the platform
without rewriting it.

This doc assumes you've read [06-architecture.md](./06-architecture.md)
and [03-templates.md](./03-templates.md).

## Design principles

When extending the codebase, respect these invariants:

1. **Handlers stay thin.** Parse input, call services, render templates.
   No business logic in HTTP handlers.

2. **Storage is the source of truth.** Add a `Store.Get/Set` method
   for new data, not a parallel cache.

3. **Templates get all data via `gin.H`.** Don't reach into Go structs
   from `html/template` — pre-compute what's needed.

4. **The `email.Sender` interface is sacred.** Swapping providers
   must not touch handlers.

5. **Static > dynamic.** Prefer markdown-driven content over
   a CMS database. Only add DBs when you need to.

## Common extension recipes

### 1. Add a custom email provider

Today we ship `email.NewSMTPSender`. To swap to Resend, SendGrid, etc.:

1. Create `internal/email/resend.go`:

   ```go
   package email

   type ResendSender struct {
       apiKey string
       from   string
   }

   func NewResendSender(cfg *config.Config) *ResendSender {
       return &ResendSender{
           apiKey: cfg.SMTPUser, // or add ResendAPIKey
           from:   cfg.ContactEmailFrom,
       }
   }

   func (r *ResendSender) SendInquiry(ctx context.Context, inq models.Inquiry) error {
       // POST to https://api.resend.com/emails
       // ...
   }
   ```

2. In `cmd/server/main.go`:

   ```go
   import "github.com/nuteo/nuteo-web/internal/email"
   // ...
   var mail email.Sender
   if cfg.UseResend {
       mail = email.NewResendSender(cfg)
   } else {
       mail = email.NewSMTPSender(cfg)
   }
   ```

3. Add a `UseResend` flag to `internal/config/config.go`.

That's it. Handlers don't change.

### 2. Add a contact-form reCAPTCHA / hCaptcha

1. Add env vars:

   ```go
   type Config struct {
       // ...
       RecaptchaSecret string `env:"RECAPTCHA_SECRET" envDefault:""`
       RecaptchaSiteKey string `env:"RECAPTCHA_SITE_KEY" envDefault:""`
   }
   ```

2. In `internal/handlers/contact.go`, before validation:

   ```go
   recaptchaResp := c.PostForm("g-recaptcha-response")
   if d.Cfg.RecaptchaSecret != "" {
       if err := verifyRecaptcha(d.Cfg.RecaptchaSecret, recaptchaResp, c.ClientIP()); err != nil {
           d.renderContact(c, "Captcha verification failed.", inq)
           return
       }
   }
   ```

3. In the contact template, add the script + widget:

   ```html
   {{if .RecaptchaSiteKey}}
   <script src="https://www.google.com/recaptcha/api.js"></script>
   <div class="g-recaptcha" data-sitekey="{{.RecaptchaSiteKey}}"></div>
   {{end}}
   ```

### 3. Add a blog (Post routing)

`content/posts/*.md` is already loaded into the store but not yet
exposed as routes. To wire it up:

1. **Handlers** (`internal/handlers/posts.go`):

   ```go
   func (d *Deps) Posts(c *gin.Context) {
       data := baseData(c, d, "Blog")
       data["PageDescription"] = "Engineering articles and updates."
       data["Posts"] = d.Store.Posts
       renderPage(c, data)
   }

   func (d *Deps) PostDetail(c *gin.Context) {
       slug := c.Param("slug")
       p := d.Store.PostBySlug(slug)
       if p == nil {
           d.NotFound(c)
           return
       }
       data := baseData(c, d, p.Title)
       data["PageDescription"] = p.Summary
       data["Post"] = p
       renderPage(c, data)
   }
   ```

2. **Templates** — create `web/templates/pages/posts.html` and
   `post.html`. Pattern: same as `portfolio.html` / `project.html`.

3. **Route** (`cmd/server/main.go`):

   ```go
   r.GET("/blog",              handlers.SetPage("posts.html"), deps.Posts)
   r.GET("/blog/:slug",        handlers.SetPage("post.html"),  deps.PostDetail)
   ```

4. **Sitemap** (`cmd/server/sitemap.go`):

   ```go
   for _, p := range store.Posts {
       urls = append(urls, sitemapURL{
           Loc:        cfg.SiteURL + "/blog/" + p.Slug,
           LastMod:    p.UpdatedAt,
       })
   }
   ```

5. **Header nav** (`web/templates/partials/header.html`) — add a
   "Blog" link.

### 4. Add an admin panel

Add auth + CRUD endpoints for content.

1. **Auth**: add a session middleware (`internal/middleware/auth.go`):

   ```go
   func RequireAuth(cfg *config.Config) gin.HandlerFunc {
       return func(c *gin.Context) {
           cookie, err := c.Cookie("admin_session")
           if err != nil || !isValidSession(cookie, cfg.AdminSecret) {
               c.Redirect(http.StatusFound, "/admin/login")
               c.Abort()
               return
           }
           c.Next()
       }
   }
   ```

2. **Login handler** (`internal/handlers/admin.go`):

   ```go
   func (d *Deps) AdminLogin(c *gin.Context) {
       if c.Request.Method == "GET" {
           data := baseData(c, d, "Admin Login")
           renderPage(c, data)
           return
       }
       user := c.PostForm("user")
       pass := c.PostForm("pass")
       if subtle.ConstantTimeCompare([]byte(user), []byte(d.Cfg.AdminUser)) == 1 &&
           subtle.ConstantTimeCompare([]byte(pass), []byte(d.Cfg.AdminPass)) == 1 {
           session := makeSession(d.Cfg.AdminSecret)
           c.SetCookie("admin_session", session, 86400, "/", "", true, true)
           c.Redirect(http.StatusFound, "/admin")
           return
       }
       // render error
   }
   ```

3. **CRUD**:

   ```go
   r.GET("/admin",               mw.RequireAuth(cfg), handlers.SetPage("admin/dashboard.html"), deps.AdminDashboard)
   r.POST("/admin/services",     mw.RequireAuth(cfg), deps.AdminCreateService)
   r.POST("/admin/services/:slug/edit", mw.RequireAuth(cfg), deps.AdminUpdateService)
   r.POST("/admin/services/:slug/delete", mw.RequireAuth(cfg), deps.AdminDeleteService)
   ```

4. **Templates** in `web/templates/pages/admin/*.html`. Use HTMX for
   inline edit / save.

5. **Storage layer** — add `Store.UpsertService(svc)` and
   `Store.DeleteService(slug)`. These write back to disk
   (`os.WriteFile`) **and** update the in-memory map.

> Admin writes to disk → next restart reloads. No DB needed for
> small sites (< 100 services).

### 5. Add a database (Postgres for blog comments)

1. Add `github.com/jackc/pgx/v5` to `go.mod`.

2. Create `internal/storage/db.go`:

   ```go
   type DB struct {
       pool *pgxpool.Pool
   }

   func OpenDB(ctx context.Context, url string) (*DB, error) {
       pool, err := pgxpool.New(ctx, url)
       return &DB{pool: pool}, err
   }
   ```

3. Migrations in `internal/storage/migrations/0001_comments.sql`:

   ```sql
   CREATE TABLE comments (
       id SERIAL PRIMARY KEY,
       post_slug TEXT NOT NULL,
       author TEXT NOT NULL,
       body TEXT NOT NULL,
       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
   );
   ```

4. Add a `*sql.DB` to `Deps`:

   ```go
   type Deps struct {
       Cfg   *config.Config
       Store *storage.Store
       Mail  email.Sender
       DB    *storage.DB   // ← new
   }
   ```

5. Use in handlers:

   ```go
   func (d *Deps) Comments(c *gin.Context) {
       rows, err := d.DB.Query(c.Request.Context(),
           "SELECT author, body, created_at FROM comments WHERE post_slug = $1 ORDER BY created_at DESC",
           c.Param("slug"))
       // ...
   }
   ```

### 6. Add an RSS / Atom feed

1. Create `internal/handlers/feed.go`:

   ```go
   func (d *Deps) Atom(c *gin.Context) {
       c.Header("Content-Type", "application/atom+xml; charset=utf-8")
       data := gin.H{
           "SiteName": d.Cfg.SiteName,
           "SiteURL":  d.Cfg.SiteURL,
           "Posts":    d.Store.Posts, // or filtered
       }
       tmpls := c.MustGet("tmpls").(map[string]*template.Template)
       tmpls["atom.xml"].Execute(c.Writer, data)
   }
   ```

2. Add `web/templates/atom.xml`:

   ```xml
   <?xml version="1.0" encoding="utf-8"?>
   <feed xmlns="http://www.w3.org/2005/Atom">
     <title>{{.SiteName}}</title>
     <link href="{{.SiteURL}}/atom.xml" rel="self"/>
     <link href="{{.SiteURL}}"/>
     <id>{{.SiteURL}}/</id>
     <updated>{{(index .Posts 0).UpdatedAt.Format "2006-01-02T15:04:05Z07:00"}}</updated>
     {{range .Posts}}
     <entry>
       <title>{{.Title}}</title>
       <link href="{{$.SiteURL}}/blog/{{.Slug}}"/>
       <id>{{$.SiteURL}}/blog/{{.Slug}}</id>
       <updated>{{.UpdatedAt.Format "2006-01-02T15:04:05Z07:00"}}</updated>
       <summary>{{.Summary}}</summary>
     </entry>
     {{end}}
   </feed>
   ```

3. Register in `pageFiles` and route:

   ```go
   r.GET("/atom.xml", deps.Atom)
   ```

4. Add `<link rel="alternate" type="application/atom+xml" href="/atom.xml" title="..." />`
   to `<head>` in the layout partial.

### 7. Add HTMX-driven search

Replace full-page render with inline search results:

1. Add `web/templates/partials/search_results.html`:

   ```html
   <div id="search-results">
     {{if .Results}}
     <ul>
       {{range .Results}}
       <li><a href="{{.URL}}">{{.Title}}</a> — {{.Snippet}}</li>
       {{end}}
     </ul>
     {{else}}
     <p>No matches.</p>
     {{end}}
   </div>
   ```

2. Handler:

   ```go
   func (d *Deps) Search(c *gin.Context) {
       q := strings.ToLower(c.Query("q"))
       results := []gin.H{}
       for _, s := range d.Store.Services {
           if strings.Contains(strings.ToLower(s.Title+s.Summary), q) {
               results = append(results, gin.H{
                   "Title": s.Title, "URL": "/services/" + s.Slug,
                   "Snippet": truncate(s.Summary, 100),
               })
           }
       }
       overridePage(c, "partials/search_results.html")
       data := gin.H{"Results": results}
       renderPage(c, data)
   }
   ```

3. In header search box:

   ```html
   <form hx-get="/search" hx-target="#search-results" hx-trigger="input changed delay:300ms">
     <input name="q" type="search" placeholder="Search...">
   </form>
   ```

### 8. Add Prometheus metrics

1. Add `github.com/prometheus/client_golang/prometheus/promhttp`.

2. In `cmd/server/main.go`:

   ```go
   import "github.com/prometheus/client_golang/prometheus/promhttp"

   r.GET("/metrics", gin.WrapH(promhttp.Handler()))
   ```

3. Add middleware for HTTP metrics:

   ```go
   func Metrics() gin.HandlerFunc {
       return func(c *gin.Context) {
           start := time.Now()
           c.Next()
           httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath()).Inc()
           httpRequestDuration.WithLabelValues(c.FullPath()).Observe(time.Since(start).Seconds())
       }
   }
   ```

4. Add `/metrics` to Caddy reverse proxy if behind it — usually scraped
   by Prometheus from internal network only.

### 9. Add multi-language (i18n)

1. Add `Accept-Language` middleware:

   ```go
   func Locale(c *gin.Context) {
       lang := c.GetHeader("Accept-Language")
       if lang == "" || !strings.HasPrefix(lang, "th") {
           lang = "en"
       } else {
           lang = "th"
       }
       c.Set("lang", lang)
       c.Next()
   }
   ```

2. Templates branch on `{{.Lang}}`:

   ```html
   {{if eq .Lang "th"}}
   ยินดีต้อนรับ
   {{else}}
   Welcome
   {{end}}
   ```

3. For full i18n, move strings to `web/i18n/{en,th}.toml` and load
   via a function:

   ```go
   func T(key string) string { /* lookup in current locale */ }
   ```

   Then in templates: `<h1>{{T "home.hero.title"}}</h1>`.

This is **a lot** of work for content-driven sites. Prefer per-page
templates per language over an i18n framework unless you really need it.

### 10. Add a JSON API (for mobile apps / SPA)

1. Create `internal/handlers/api.go`:

   ```go
   func (d *Deps) APIServices(c *gin.Context) {
       c.JSON(200, gin.H{"services": d.Store.Services})
   }
   ```

2. Mount API under `/api/v1/`:

   ```go
   api := r.Group("/api/v1")
   api.GET("/services",       deps.APIServices)
   api.GET("/services/:slug", deps.APIServiceDetail)
   api.POST("/inquiries",     deps.APISubmitInquiry)
   ```

3. Add rate limiting (`ulule/limiter`):

   ```go
   import "github.com/ulule/limiter/v3"
   import mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"

   rate, _ := limiter.NewRateFromFormatted("100-M")
   store := memory.NewStore()
   mw := mgin.NewMiddleware(limiter.New(store, rate))
   api.Use(mw)
   ```

## Performance optimization

The app is already fast (Go + compiled templates + in-memory store).
For most sites the bottleneck will be:

1. **Cold start**: < 100ms. Negligible.
2. **Markdown render**: ~1ms per file. Negligible unless you have
   thousands of files.
3. **Template execute**: ~1ms per page. Fine.
4. **Email send**: blocking. Move to async worker:

   ```go
   go func() {
       ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
       defer cancel()
       if err := d.Mail.SendInquiry(ctx, inq); err != nil {
           log.Printf("email send failed: %v", err)
       }
   }()
   d.renderThanks(c, inq)
   ```

   > Don't forget to add a queue (Redis / NATS / Postgres) for
   > durability — losing emails is worse than latency.

5. **Static assets**: served by Caddy directly, not the Go app.
   Already configured in `deploy/Caddyfile`.

## Where to add new things (cheat sheet)

| Want to add... | Touch these files |
|---|---|
| New page | `web/templates/pages/*.html` + `internal/handlers/page.go` + `internal/handlers/pages.go` + `cmd/server/main.go` |
| New field on service | `internal/models/models.go` + every page template that displays it + `internal/storage/parsers.go` wrapper |
| New env var | `internal/config/config.go` + use it somewhere |
| New email provider | new file in `internal/email/` + switch in `main.go` |
| New middleware | `internal/middleware/` + add to router in `main.go` |
| New partial template | `web/templates/partials/*.html` + reference in `page.go` |
| DB | `internal/storage/db.go` + add to `Deps` + migrate |

## Refactoring checklists

When making non-trivial changes:

- [ ] Run `go vet ./...` — catches shadowed vars, etc.
- [ ] Run `go build` — must succeed
- [ ] Restart and curl-test all routes
- [ ] Update docs (`docs/03-templates.md`, etc.)
- [ ] Commit with descriptive message
