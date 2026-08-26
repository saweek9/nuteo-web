# Architecture

A tour through the codebase, top-down. By the end of this you should
be able to confidently modify any part of nuteo-web.

## The 30-second tour

```
Request
   │
   ▼
[ Gin Router ] ─── middleware ──▶ handler ──▶ response
   │                                  │
   │                                  ├─▶ Storage (load content)
   │                                  ├─▶ email.Sender (send inquiry)
   │                                  └─▶ Templates (render HTML)
   ▼
static files (CSS, JS, images)
```

That's it. Every request flows through these components.

## Layer cake

```
┌─────────────────────────────────────────────────────────┐
│ Templates (html/template)        web/templates/*.html   │
│   - partials  (header, footer)                          │
│   - pages    (home, services, contact, ...)             │
└─────────────────────────────────────────────────────────┘
                            ▲
                            │ renders gin.H
┌─────────────────────────────────────────────────────────┐
│ Handlers (Gin)                  internal/handlers/*.go   │
│   - Parse input                                          │
│   - Call services                                        │
│   - Fill template data                                   │
└─────────────────────────────────────────────────────────┘
            ▲                              │
            │ uses                         │ depends on
┌──────────────────────┐    ┌────────────────────────────────┐
│ Storage              │    │ External integrations         │
│  internal/storage    │    │  - email.Sender (SMTP, etc.)  │
│  - Store (services,  │    │  - future: db, payment, ...  │
│    portfolio, posts) │    └────────────────────────────────┘
│  - YAML + Markdown   │
│    loaders           │
└──────────────────────┘
            ▲
            │ fills at startup
┌─────────────────────────────────────────────────────────┐
│ Content (filesystem)      content/**/*.md               │
│   - services/*.md                                        │
│   - portfolio/*.md                                      │
│   - posts/*.md                                          │
└─────────────────────────────────────────────────────────┘

            ▲
            │ configured by
┌─────────────────────────────────────────────────────────┐
│ Config (env-based)        internal/config/config.go      │
│   - Site name, URL                                       │
│   - SMTP credentials                                    │
│   - CSRF secret                                          │
└─────────────────────────────────────────────────────────┘
```

## Key files

```
cmd/server/main.go           ← entry point. Wires everything.
internal/config/config.go    ← env → Config struct
internal/models/models.go    ← Service, Portfolio, Post, Inquiry
internal/storage/
    storage.go               ← Store (in-memory content cache)
    parsers.go               ← Markdown → model loaders
internal/email/email.go      ← Sender interface + SMTP impl
internal/middleware/
    middleware.go            ← RequestID, Logger, SecurityHeaders
internal/handlers/
    handlers.go              ← Deps struct (shared by all handlers)
    page.go                  ← Template loading + render helpers
    pages.go                 ← Home, Services, Portfolio, About, NotFound
    contact.go               ← Contact GET + POST
cmd/server/sitemap.go        ← dynamic sitemap.xml
deploy/
    Dockerfile               ← multi-stage build
    docker-compose.yml       ← app + caddy
    Caddyfile                ← reverse proxy with auto-TLS
scripts/
    new-content.sh           ← scaffold new .md
    debug-load/main.go       ← loader debug tool
    service.sh               ← systemctl helper (start/stop/logs/build)
```

## Request lifecycle

For `GET /services/cloud-migration`:

1. **DNS** resolves the domain → server IP.
2. **Caddy** (in Docker Compose setup) accepts TLS, terminates HTTPS,
   reverse-proxies to `app:8080`. (In systemd-only setup, requests hit
   the app directly via Caddy on the host.)
3. **Gin router** matches `/services/:slug` → `ServiceDetail` handler.
4. **Middleware** runs (RequestID, Logger, SecurityHeaders, SetPage).
5. **`ServiceDetail`**:
   - Reads `slug = "cloud-migration"` from path.
   - Asks `Store.ServiceBySlug(slug)` for the content.
   - Stores are loaded **once at startup** (see below).
   - Builds `gin.H{"PageTitle": svc.Title, ...}`.
   - Calls `renderPage(c, data)`.
6. **`renderPage`**:
   - Looks up the template set for `service.html`.
   - Compiled at startup (one `*template.Template` per page,
     each including header + footer partials).
   - Executes the template, writing HTML to the response.
7. **Response** flows back through middleware → Gin → Caddy → browser.
8. **Browser** renders HTML, requests static assets (CSS, JS, images).

For `POST /contact`:

Same path, but `ContactSubmit`:
1. Reads form values via `c.PostForm(...)`.
2. Converts `consent=on` → bool.
3. Checks honeypot (silent 200 if filled).
4. Validates (name, email, message required).
5. Sends email via `email.Sender.SendInquiry` (best-effort, logs on
   failure but doesn't block success).
6. Renders `contact_thanks.html`.

## Startup sequence

`cmd/server/main.go`:

```
1.  logger := slog.New(...)           // JSON to stdout
2.  cfg, err := config.Load()         // env → Config struct
3.  if cfg.Env == "development" → gin.SetMode(DebugMode)
4.  store := storage.New()
5.  store.LoadAll(cfg.ContentDir)     // reads content/**/*.md
                                     // into store.Services/Portfolio/Posts
6.  mail := email.NewSMTPSender(cfg)
7.  deps := &handlers.Deps{Cfg, Store, Mail}
8.  tmpls := deps.CompileTemplates()  // parse every .html file
9.  r := gin.New()
10. r.Use(...)                        // middleware (request ID, logger, ...)
11. r.Use(func(c) { c.Set("tmpls", tmpls) })   // inject templates
12. r.Use(handlers.SetPage("..."))    // tag template per route
13. r.GET/POST(...)                   // routes
14. r.Static("/static", ...)          // static files
15. go srv.ListenAndServe()           // listen (graceful shutdown later)
```

In **systemd mode**, this is wrapped:

```
[Unit]     nuteo-web.service
[Service]  EnvironmentFile=/home/admin_nuteo/nuteo-web/.env
           ExecStart=/home/admin_nuteo/nuteo-web/bin/nuteo-web
           Restart=on-failure
[Install]  WantedBy=multi-user.target
```

systemd auto-starts on boot and restarts on crash. See
[09-auto-start.md](./09-auto-start.md).

## Data flow for content

```
content/services/cloud-migration.md
       │
       │ parseMarkdownFile()
       ▼
serviceModel (wrapper with SetHTML/SetMD hooks)
       │
       │ unmarshal YAML → Service fields
       │ render markdown → BodyHTML
       ▼
[]models.Service (stored in store.Services)
       │
       │ queried by handler
       ▼
gin.H passed to template
       │
       │ executed
       ▼
HTML response
```

The `serviceModel` wrapper is a small trick: it embeds `models.Service`
and adds three methods (`SetHTML`, `SetMD`, `SetPublishedIfZero`) so
the loader can populate rendered fields without exposing them in
`models.Service`.

## Why these choices?

### Why Go + Gin, not Node.js / Python?

- **Single binary deployment** — `scp` the binary, run it. No Python
  venv dance, no `node_modules`.
- **Type safety** — Go catches schema mismatches at compile time.
  Templates reference struct fields; if the model changes, the
  template breaks immediately.
- **Performance** — sub-millisecond response times for static-ish
  pages. Easy 1k+ req/s per core.
- **Standard library** — `html/template`, `net/http`, `net/smtp` —
  all the basics we need, no framework churn.
- **systemd-friendly** — one static binary, no Node/Python runtime to
  install alongside.

### Why `html/template`, not Templ or React?

- **Zero build step** for templates (vs Templ which generates Go code).
- **Auto-escaping** — user content can't accidentally inject script.
- **Familiar** — anyone who knows Django/Jinja can read Go templates.
- **Small** — ~150 lines of templates, vs React's ecosystem overhead.

We may add Templ later for component reuse. For now, stdlib is enough.

### Why Markdown content, not a CMS DB?

- **Version control** — every content change is a git commit. Diffs,
  blame, revert all "just work."
- **Editor support** — VS Code, Vim, IntelliJ all have great markdown.
- **No migrations** — adding a field to a model? Edit frontmatter,
  edit parser. No DB migration to write.
- **Fast** — read 4 files on startup. Sub-millisecond query.
- **Safe** — atomic file ops. No DB to back up.

For content > 1000 items, you'd want DB. We're nowhere near that.

### Why Caddy, not Nginx?

- **Auto-TLS** — one line in Caddyfile, done.
- **Hot reload** — no service restart to update config.
- **Single binary** — easier to install.
- **HTTP/3 ready** — out of the box.

Trade-off: Caddy is single-language (Go), but that's fine for a proxy.

### Why systemd (not Docker-only)?

- **No Docker dependency** — direct binary, faster cold start (~80 ms vs
  ~300 ms).
- **Native logs** — stdout/stderr go to journald, queryable via
  `journalctl`.
- **Simpler debugging** — no container layer to peel back.
- **Single-file unit** — `/etc/systemd/system/nuteo-web.service`.

Use Docker Compose when you need Caddy + app on the same host with
reproducible builds, or when deploying to PaaS. See [05-deployment.md](./05-deployment.md).

## Extension points

These are the seams where you add features without rewriting:

| Extension point | How to use |
|---|---|
| New page | Add `.html` + register in `pageFiles` + handler in `pages.go` + route in `main.go` |
| New model field | Add to `models.X` + frontmatter parser + template |
| New env var | Add to `Config` struct with `env:` tag |
| New email provider | Implement `Sender` interface, swap in `main.go` |
| New middleware | Add to `internal/middleware/`, `r.Use(...)` it |
| DB | Add to `Deps`, write queries in `storage/db.go` |
| Admin panel | Add routes under `/admin`, auth middleware, CRUD handlers |

See [07-extending.md](./07-extending.md) for worked examples.

## Security posture

Defaults:

- ✅ **HTTPS** — Caddy auto-TLS via Let's Encrypt (in Docker mode)
- ✅ **Security headers** — CSP, HSTS, X-Frame-Options, etc.
- ✅ **CSRF protection** — per-request tokens + cookie double-submit
- ✅ **Honeypot** — `website` field traps naive bots
- ✅ **HTML escaping** — html/template auto-escapes user content
- ✅ **Input validation** — go-playground/validator
- ✅ **Sane defaults** — read/write timeouts, max multipart size (1 MB)
- ✅ **Single-binary attack surface** — no Python runtime, no Node
- ✅ **systemd hardening** — `MemoryMax=256M`, `RestartSec=5s`,
  `StandardOutput=journal` for log auditing

What's NOT enforced by default (you may want to add):

- **Rate limiting** — add `ulule/limiter` middleware
- **reCAPTCHA** — add when spam becomes a problem
- **CSP nonce** — currently `'unsafe-inline'` allowed for inline styles
- **Subresource Integrity** — for CDN assets (we ship HTMX locally)

## Performance characteristics

Empirical (tested on a $5/mo VPS):

| Metric | Value |
|---|---|
| Cold start (systemd) | ~80 ms |
| Per-request latency (cached) | 200-500 µs |
| Template render (one page) | 1-2 ms |
| Markdown render (one file) | 500 µs - 2 ms |
| Memory (RSS) | 5-30 MB |
| Memory with 1000 services | 60-80 MB |
| Throughput (single core) | ~5k req/s |
| systemd-managed idle CPU | <1% |

Not benchmarked:

- 10k+ services (likely fine, but test before scaling)
- DB-backed extensions

## Open questions / known limitations

- **No hot reload** — restart to pick up content changes. For
  zero-downtime, use rolling restart via systemd or Docker Compose.
- **Markdown-only content** — no rich editor in admin panel yet.
  Add when you build the admin UI.
- **No CMS** — adding a CMS is a deliberate choice; for now, edit
  markdown in git.
- **Single language per page** — no `lang` attribute switching.
  Easy to add (see extending).
- **No inline edit** — content changes require a code edit + restart.
  Future: admin panel.

## Directory tree (post-move)

```
/home/admin_nuteo/nuteo-web/
├── README.md                      ← project overview
├── ARCHITECTURE.md                ← (older doc, retained for reference)
├── Makefile                       ← build / run / docker-up / docker-down
├── go.mod / go.sum                ← Go module + dependencies
├── .env / .env.example            ← runtime config (gitignored, 600 perms)
├── .gitignore
│
├── cmd/
│   └── server/
│       ├── main.go                ← entry point
│       └── sitemap.go             ← sitemap.xml generator
│
├── internal/
│   ├── config/config.go           ← env → Config
│   ├── models/models.go           ← Service, Portfolio, Post, Inquiry
│   ├── storage/
│   │   ├── storage.go             ← Store (in-memory)
│   │   └── parsers.go             ← Markdown loaders
│   ├── email/email.go             ← Sender interface + SMTP
│   ├── middleware/middleware.go   ← RequestID, Logger, SecurityHeaders
│   └── handlers/
│       ├── handlers.go            ← Deps struct
│       ├── page.go                ← templates + render helpers
│       ├── pages.go               ← Home, Services, Portfolio, About, NotFound
│       └── contact.go             ← Contact GET + POST
│
├── web/
│   ├── templates/
│   │   ├── pages/                 ← 8 page templates
│   │   └── partials/              ← header, footer
│   └── static/
│       ├── css/site.css           ← 3 KB CSS
│       ├── js/htmx.min.js         ← HTMX 1.9.12 (offline)
│       └── images/                ← logo, favicon
│
├── content/                       ← Markdown content (CMS)
│   ├── services/                  ← 4 services
│   ├── portfolio/                 ← 2 projects
│   └── posts/                     ← (empty, future /blog)
│
├── deploy/
│   ├── Dockerfile                 ← multi-stage build
│   ├── docker-compose.yml         ← app + Caddy
│   └── Caddyfile                  ← reverse proxy
│
├── docs/                          ← this directory
│   ├── README.md                  ← index
│   ├── 01-quickstart.md
│   ├── 02-content-authoring.md
│   ├── 03-templates.md
│   ├── 04-configuration.md
│   ├── 05-deployment.md
│   ├── 06-architecture.md         ← you are here
│   ├── 07-extending.md
│   ├── 08-troubleshooting.md
│   └── 09-auto-start.md
│
├── scripts/
│   ├── new-content.sh             ← scaffold new .md
│   ├── debug-load/main.go         ← loader debug tool
│   └── service.sh                 ← systemctl wrapper (start/stop/logs/build)
│
├── bin/
│   └── nuteo-web                  ← compiled binary (~22 MB)
│
└── /etc/systemd/system/nuteo-web.service   ← service unit (outside repo)
```
