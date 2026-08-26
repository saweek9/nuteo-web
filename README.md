# nuteo-web

Corporate B2B website for **nuteo solution** — software engineering and IT
consulting from Bangkok.

Built with **Go + Gin + Templ-style templates + HTMX**, content-driven via
Markdown files. Easy to extend with more sections, a blog, or an admin panel.

## Why this stack

| Concern | Choice | Benefit |
|---|---|---|
| Language | Go 1.25 | Single static binary, ~10 MB Docker image, low memory |
| HTTP router | Gin | Mature, fast, middleware ecosystem |
| Templates | `html/template` (stdlib) | No build step, ships with Go |
| Interactivity | HTMX | Progressive enhancement, no SPA overhead |
| Content | Markdown files | Git-versioned, no DB, edit with any editor |
| Email | `net/smtp` (stdlib) | Zero deps; swap to Resend/SES later |
| Deploy | Docker compose + Caddy | Auto-TLS, single VM, full control |

## Quick start

```bash
# 1. Configure
cp .env.example .env
# edit .env — at minimum set CONTACT_EMAIL_*, CSRF_SECRET, SITE_URL

# 2. Generate CSRF secret
openssl rand -hex 32  # paste into CSRF_SECRET

# 3. Run locally
make run

# Open http://localhost:8080
```

## Project structure

```
nuteo-web/
├── cmd/server/             main entrypoint
├── internal/
│   ├── config/             env loading
│   ├── handlers/           HTTP handlers (one per concern)
│   ├── models/             domain types
│   ├── middleware/         logging, security headers
│   ├── storage/            markdown content loader
│   └── email/              SMTP sender
├── web/
│   ├── templates/          .html templates (layouts/pages/partials)
│   └── static/             CSS, JS, images
├── content/                editable markdown content
│   ├── services/*.md
│   ├── portfolio/*.md
│   └── posts/*.md
├── deploy/
│   ├── Dockerfile          multi-stage, distroless base
│   ├── docker-compose.yml  app + Caddy reverse proxy
│   └── Caddyfile           auto-TLS config
├── scripts/
│   └── new-content.sh      scaffold new .md files
├── Makefile                build, run, test, docker commands
├── ARCHITECTURE.md         design rationale
└── .env.example
```

## URL surface

| Path | Source | Notes |
|---|---|---|
| `/` | rendered from `home.html` + featured content |
| `/services` | `content/services/*.md` |
| `/services/{slug}` | detail page, rendered markdown body |
| `/portfolio` | `content/portfolio/*.md` |
| `/portfolio/{slug}` | detail page |
| `/about` | static page |
| `/contact` | form |
| `POST /contact` | validates + sends email |
| `/posts/{slug}` | optional blog |
| `/sitemap.xml` | auto-generated from content |
| `/robots.txt` | static |
| `/healthz` | returns 200 OK |

## Adding content

```bash
# Create a new service (frontmatter pre-filled)
scripts/new-content.sh service cloud-cost-optimization

# Edit the file
$EDITOR content/services/cloud-cost-optimization.md

# Restart the app
make run
```

## Adding a new section / extending

This is where the framework shines — adding things is additive:

| Want to add | Where |
|---|---|
| Blog post section | New `content/posts/` files; templates in `web/templates/pages/posts.html` |
| Team page | New `content/team/*.md` + a `teamModel` wrapper in `internal/storage/parsers.go` |
| Admin panel | New route + handler + basic-auth middleware |
| Subscribe to newsletter | Add `POST /subscribe` handler + table-backed `Subscriber` model |
| CMS UI | Mount the content dir over SFTP / git push; the app re-reads on restart |
| Search | Add `internal/search/` with bleve; index on startup |

## Security baseline (built-in)

- CSRF token on POST /contact
- Honeypot field on contact form
- Rate limit (5/min per IP) on contact form
- CSP, X-Frame-Options, X-Content-Type-Options, HSTS via middleware + Caddy
- Input validation via `go-playground/validator`
- HTML auto-escape by Go templates
- Structured JSON logs (`log/slog`)
- Graceful shutdown
- Health endpoint for monitoring

## Deploy

```bash
# 1. Build the image
make docker-build

# 2. Copy deploy dir + .env to your VPS
scp -r deploy/ .env user@server:/opt/nuteo/

# 3. On the server
ssh user@server 'cd /opt/nuteo/deploy && docker compose up -d'

# 4. Update DNS to point at the server
#    A nuteo.example.com → <server-ip>

# Caddy will automatically obtain a Let's Encrypt cert.
```

## Development workflow

```bash
make build       # compile
make run         # build + run with .env
make test        # run tests
make vet         # go vet
make fmt         # go fmt
make docker-up   # run via docker compose
```

## License

MIT (or your preferred license).
