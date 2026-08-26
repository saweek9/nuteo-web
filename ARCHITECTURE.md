# nuteo-web — Architecture

Corporate B2B website for nuteo solution. English-primary, Go + Gin +
Templ + HTMX stack, content-driven via Markdown files, deployed via
Docker compose on a VPS.

## Why this stack

| Concern | Choice | Why |
|---|---|---|
| Language | Go 1.22+ | Single static binary, fast cold-start, low memory, type-safe |
| HTTP router | Gin v1.10 | Battle-tested, fast, middleware ecosystem |
| Templating | Templ v0.2+ | Type-safe Go templates, no runtime errors, fast |
| Interactivity | HTMX 1.9 | Progressive enhancement, no SPA needed, SEO-friendly |
| Styling | Tailwind CSS 3.4 (CDN for now, build later) | Rapid styling, small CSS footprint |
| Content | Markdown files in `content/` | Git-versioned, easy to edit, no DB required |
| Email | `net/smtp` (stdlib) or Resend HTTP API | Zero deps or modern API |
| Logging | `log/slog` (stdlib) | Structured JSON logs, no third-party |
| Config | env vars via `os.Getenv` + `caarlos0/env` | 12-factor |
| Deploy | Docker compose (app + reverse proxy + mail relay optional) | Self-host, full control |

## Directory structure

```
nuteo-web/
├── cmd/server/             # main entrypoint
├── internal/
│   ├── config/             # env loading, validation
│   ├── handlers/           # HTTP handlers (one file per concern)
│   ├── services/           # business logic (services registry, email)
│   ├── models/             # data types (Service, Portfolio, Post, Inquiry)
│   ├── middleware/         # logging, security headers, rate limit
│   ├── storage/            # content loader (markdown → models)
│   └── email/              # contact form sender
├── web/
│   ├── templates/          # .templ files
│   │   ├── layouts/        # base, page
│   │   ├── pages/          # home, services, portfolio, post, contact, ...
│   │   ├── partials/       # header, footer, nav, contact-form, service-card
│   │   └── emails/         # inquiry notification (plain text + html)
│   └── static/             # css, js, images
├── content/                # editable content (git-versioned)
│   ├── services/*.md       # one file per service offering
│   ├── portfolio/*.md      # one file per project
│   └── posts/*.md          # optional blog
├── deploy/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── nginx.conf          # reverse proxy + TLS
│   └── Caddyfile           # alt (auto-TLS)
├── scripts/
│   └── new-content.sh      # scaffold a new .md file with frontmatter
├── go.mod / go.sum
├── Makefile
└── README.md
```

## URL surface

| Path | Handler | Source |
|---|---|---|
| `GET /` | Home | static (landing sections) |
| `GET /services` | Services index | content/services/*.md |
| `GET /services/{slug}` | Service detail | content/services/{slug}.md |
| `GET /portfolio` | Portfolio index | content/portfolio/*.md |
| `GET /portfolio/{slug}` | Portfolio detail | content/portfolio/{slug}.md |
| `GET /about` | About page | static |
| `GET /contact` | Contact form | form HTML |
| `POST /contact` | Contact submission | inquiry → email + storage |
| `GET /posts` | Blog index | content/posts/*.md |
| `GET /posts/{slug}` | Post detail | content/posts/{slug}.md |
| `GET /sitemap.xml` | SEO sitemap | generated from content |
| `GET /robots.txt` | SEO robots | static |
| `GET /healthz` | Liveness | no auth, returns 200 OK |
| `GET /static/*` | Static assets | served by Gin |

## Content frontmatter schema

```yaml
---
title: "Cloud Migration & Infrastructure"
slug: "cloud-migration"
summary: "Lift, optimize, and operate your workloads on AWS, GCP, or Azure."
icon: "cloud"                # matches icon set in static/icons/
order: 1                     # controls display order on /services
audience: ["enterprise"]     # filters / future segmentation
tags: ["aws", "gcp", "kubernetes"]
featured: true               # show on homepage
cta_label: "Discuss migration"
cta_href: "/contact?topic=cloud-migration"
published_at: 2026-01-15
updated_at: 2026-08-01
---

# Markdown body — full description
```

## Data flow

```
HTTP request
  → middleware (request-id, slog, security headers, rate-limit)
  → router (Gin)
  → handler
      → if static page: templ template (with models loaded from cache)
      → if /contact POST: validate → service.InquiryStore.Save → email.Sender.Send
  → response
```

## Extensibility (built-in hooks)

The framework ships with stubs already wired so adding new features is
additive, not a rewrite:

- **`internal/handlers/`** — add a new file, register route in router
- **`internal/storage/`** — swap Markdown for Postgres later without
  touching handlers (interface-based)
- **`internal/email/`** — implement `Sender` interface for SendGrid,
  Mailgun, Resend, SES without touching handlers
- **`content/`** — add a new directory (e.g. `content/team/`) and
  extend `models.Content` loader
- **`web/templates/`** — partials compose; add a new section by
  including an existing partial
- **`scripts/new-content.sh`** — scaffolds new markdown files with the
  right frontmatter

## Security defaults

- HTTPS-only via reverse proxy (Caddy auto-TLS or nginx + certbot)
- HTTP→HTTPS redirect
- CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy headers
- HSTS (1 year, includeSubDomains)
- CSRF token on POST /contact (double-submit cookie)
- Honeypot field on contact form (bots fill it, humans don't)
- Rate limit /contact (5/min per IP via in-memory token bucket)
- Input validation via `go-playground/validator`
- HTML escape all user input (templ does this by default)

## Observability

- Structured JSON logs to stdout (`log/slog`)
- Request-ID middleware → propagate to logs
- `/healthz` for liveness, `/readyz` (planned) for readiness
- Optional `/metrics` (Prometheus) — gated behind config

## Deploy

```bash
# Build
make build

# Run locally
./bin/nuteo-web

# Deploy via Docker compose
scp -r deploy/ user@server:/opt/nuteo/
ssh user@server 'cd /opt/nuteo && docker compose up -d'
```

See `deploy/docker-compose.yml` for full stack:
- `app` (Go binary, alpine)
- `caddy` (reverse proxy + auto-TLS)
- Optional: `mailhog` for local email testing