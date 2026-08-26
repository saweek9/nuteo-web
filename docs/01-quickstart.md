# Quickstart

From zero to a running nuteo-web instance in 5 minutes.

## Prerequisites

- **Go 1.25+** (we tested on 1.26)
- **Git**
- **curl** (for testing)
- Optional: **Docker** + **Docker Compose** for production-style deploy

Verify Go:

```bash
go version  # should print go1.25 or higher
```

If missing, install from https://go.dev/dl/.

## Get the code

```bash
git clone https://github.com/saweek9/nuteo-web.git ~/projects/nuteo-web
cd ~/projects/nuteo-web
go mod download
```

## Configure

Copy the env template and generate a CSRF secret:

```bash
cp .env.example .env
sed -i "s|^CSRF_SECRET=.*|CSRF_SECRET=$(openssl rand -hex 32)|" .env
```

The defaults are sane for local development. The main knobs are:

| Variable | Default | Description |
|---|---|---|
| `ENV` | `production` | `production` or `development` |
| `ADDR` | `:8080` | HTTP listen address |
| `CONTACT_EMAIL_TO` | `hello@nuteo.example.com` | Where contact inquiries go |
| `CONTACT_EMAIL_FROM` | `noreply@nuteo.example.com` | From-address |
| `SMTP_HOST` | _(empty)_ | SMTP server (no email until set) |

> Without `SMTP_HOST`, contact submissions are logged but not sent.
> That's fine for local dev — form submissions will still appear in the
> server logs.

See **[04-configuration.md](./04-configuration.md)** for the full list.

## Build and run

```bash
make build       # produces bin/nuteo-web (≈ 22 MB)
make run         # foreground, Ctrl-C to stop
```

Open <http://localhost:8080> in your browser. You should see:

- The home page with featured services and portfolio items
- `/services` listing all 4 seed services
- `/services/cloud-migration` (and the other slugs) showing the detail page
- `/portfolio`, `/about`, `/contact` working
- `/sitemap.xml` and `/robots.txt` available

## Test the contact form

1. Visit <http://localhost:8080/contact>
2. Fill in name, email, topic, message, tick consent
3. Submit → you should see "Thanks" page with HTTP 200

Without SMTP configured, you'll see `X-Email-Status: failed` in the
response headers (set by the handler when send fails). The submission
is still logged.

## Run with Docker

If you prefer containers:

```bash
cd deploy
docker compose up --build
```

This starts:

- The app on port 8080 (internal)
- Caddy reverse proxy on ports 80/443 (with auto-TLS when domain is set)

See **[05-deployment.md](./05-deployment.md)** for production deploy details.

## Common next steps

- **Add content**: `./scripts/new-content.sh service <slug>` then edit the file
- **Change copy**: edit `content/services/*.md` directly — restart app to reload
- **Change styling**: edit `web/static/css/site.css` and refresh browser
- **Change contact email destination**: edit `CONTACT_EMAIL_TO` in `.env`

## Sanity checklist

Before moving on, confirm:

- [ ] `curl http://localhost:8080/healthz` returns `ok`
- [ ] Home page loads with hero text "Engineering systems that scale, reliably."
- [ ] `/services/cloud-migration` shows markdown body (h2, ul, etc.)
- [ ] `/sitemap.xml` returns valid XML with `<urlset>`
- [ ] `/robots.txt` returns `User-agent: *\nAllow: /`

If any of these fail, see **[08-troubleshooting.md](./08-troubleshooting.md)**.
