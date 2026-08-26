# Configuration

All runtime configuration comes from environment variables. We use
[`caarlos0/env`](https://github.com/caarlos0/env) for tagged struct
parsing.

## Where config comes from

1. **`.env` file** at the project root (gitignored, development only)
2. **Real environment variables** (production / CI / Docker)
3. **Defaults** baked into the code

`.env` is loaded automatically when you run `make run` (via `set -a;
. ./.env`). For Docker, `docker-compose.yml` passes env to the app
container.

## The full list

| Variable | Type | Default | Required | Description |
|---|---|---|---|---|
| `ENV` | string | `production` | | `production` or `development` |
| `ADDR` | string | `:8080` | | HTTP listen address |
| `READ_TIMEOUT` | duration | `10s` | | http.Server.ReadTimeout |
| `WRITE_TIMEOUT` | duration | `10s` | | http.Server.WriteTimeout |
| `CONTENT_DIR` | string | `./content` | | Path to markdown content directory |
| `STATIC_DIR` | string | `./web/static` | | Path to static assets |
| `CSRF_SECRET` | string | _(random per-request)_ | ✅ | Per-request CSRF token seed |
| `SITE_NAME` | string | `nuteo solution` | | Display name in nav and footer |
| `SITE_URL` | string | `https://nuteo.example.com` | | Canonical URL (used in sitemap) |
| `LOGO_PATH` | string | `/static/images/logo.svg` | | Logo URL |
| `GITHUB_URL` | string | `https://github.com/nuteo` | | GitHub link in footer |
| `LINKEDIN_URL` | string | `https://www.linkedin.com/company/nuteo` | | LinkedIn link |
| `TWITTER_HANDLE` | string | _(empty)_ | | Twitter/X handle (optional) |
| `CONTACT_EMAIL_TO` | string | — | ✅ | Where contact inquiries are sent |
| `CONTACT_EMAIL_FROM` | string | — | ✅ | From-address on sent emails |
| `SMTP_HOST` | string | _(empty)_ | | SMTP server hostname |
| `SMTP_PORT` | int | `587` | | SMTP server port |
| `SMTP_USER` | string | _(empty)_ | | SMTP auth user |
| `SMTP_PASS` | string | _(empty)_ | | SMTP auth password |
| `SMTP_USE_TLS` | bool | `true` | | Use STARTTLS |

## Examples

### Local development

`.env`:

```bash
ENV=development
ADDR=:8080

# Open in browser
SITE_URL=http://localhost:8080

# Don't send emails locally — log instead
# Leave SMTP_HOST empty

CONTACT_EMAIL_TO=sawee@nuteo.example.com
CONTACT_EMAIL_FROM=nuteo-web@localhost
CSRF_SECRET=any-arbitrary-string-works-for-dev
```

### Production (single VPS)

`.env`:

```bash
ENV=production
ADDR=:8080  # Caddy is in front; app listens on internal port
READ_TIMEOUT=10s
WRITE_TIMEOUT=10s

SITE_URL=https://nuteo.example.com

# Real email — use a transactional provider
SMTP_HOST=smtp.resend.com
SMTP_PORT=587
SMTP_USER=resend
SMTP_PASS=re_xxxxxxxxxxxxxxxxxxxx
SMTP_USE_TLS=true

CONTACT_EMAIL_TO=hello@nuteo.example.com
CONTACT_EMAIL_FROM=noreply@nuteo.example.com
CSRF_SECRET=<output of `openssl rand -hex 32`>
```

### Production (Docker Compose)

`deploy/docker-compose.yml` is already wired to read env from
`.env` at the project root. Just put your real values there.

```yaml
services:
  app:
    build: ..
    environment:
      ENV: production
      ADDR: :8080
      CONTACT_EMAIL_TO: ${CONTACT_EMAIL_TO}
      CONTACT_EMAIL_FROM: ${CONTACT_EMAIL_FROM}
      CSRF_SECRET: ${CSRF_SECRET}
      SMTP_HOST: ${SMTP_HOST}
      # ...
```

## Required vs optional

The app refuses to start if a **required** variable is missing. You'll
see this in the log:

```
config load err="config: env: required environment variable \"CONTACT_EMAIL_TO\" is not set"
```

Required variables:

- `CONTACT_EMAIL_TO`
- `CONTACT_EMAIL_FROM`
- `CSRF_SECRET`

All others have sane defaults.

## Sensitive values

Treat the following as secrets — **never commit to git**:

- `CSRF_SECRET` (rotates per-request anyway, but still keep clean)
- `SMTP_PASS` (your SMTP password)
- `SMTP_USER` (often equals the password for API-style SMTP)

`git status` should never show `.env` as modified. The `.gitignore`
in this repo excludes it.

## Where each value is used

```
Config field         Used in
─────────────────────────────────────────────────────
ENV                  main.go (gin.SetMode)
ADDR                 main.go (http.Server.Addr)
READ/WRITE_TIMEOUT   main.go (http.Server.ReadTimeout / WriteTimeout)
CONTENT_DIR          main.go → storage.LoadAll
STATIC_DIR           main.go → r.Static("/static", ...)
CSRF_SECRET          handlers (cookie signing — current impl uses per-request random, kept for future HMAC)
SITE_NAME            templates
SITE_URL             sitemap, robots
LOGO_PATH            header partial
GITHUB_URL, etc.    footer partial
CONTACT_EMAIL_TO     contact handler (recipient), config email sender
CONTACT_EMAIL_FROM   email sender (From header)
SMTP_*               email.Send()
```

## Rotating secrets

To rotate `CSRF_SECRET`:

1. Generate new: `openssl rand -hex 32`
2. Update `.env`
3. Roll restart: `docker compose up -d app` (no downtime if you have
   2+ replicas; see [05-deployment.md](./05-deployment.md))

To rotate `SMTP_PASS`:

1. Issue new credential in your SMTP provider's dashboard
2. Update `.env`
3. Roll restart

## Troubleshooting config

Use `hermes config show` — wait, that's for Hermes the agent, not
nuteo-web. For nuteo-web the easiest sanity check is:

```bash
# Show what config the app sees
make run 2>&1 | grep -E "(SMTP|Contact|Addr|Env)"
```

Or add debug logging to `cmd/server/main.go` temporarily.

## Future enhancements

- **Mail provider switch**: today, `email.Sender` is an interface.
  Swap `NewSMTPSender` for `NewResendSender`, `NewSESender`, etc.
  without touching handlers. Tracked in
  [07-extending.md](./07-extending.md).

- **Multiple contact recipients**: change `CONTACT_EMAIL_TO` parsing
  to accept comma-separated list, fan out in `Mail.SendInquiry`.

- **Per-page meta overrides**: add `meta.yaml` in `content/` to
  override template defaults per section.
