# Deployment

Production deployment recipes for nuteo-web.

## Deployment options

| Method | Best for | Complexity | Cost |
|---|---|---|---|
| [Local binary](#option-1-binary-on-vps) | Single VPS, full control | Low | ~$5/mo |
| [Docker Compose + Caddy](#option-2-docker-compose-with-caddy) | VPS, easy updates | Low | ~$5/mo |
| [systemd unit](#option-3-systemd-service-no-docker) | Existing infra | Medium | Free if you have a VM |
| [Fly.io / Render](#option-4-flyio-or-render) | Zero-config | Low | Free tier available |
| [Static export](#option-5-static-export-to-cdn) | Marketing-heavy site | Medium | Free tier |

> We're covering options 1, 2, 3 in detail. Options 4–5 are summarized.

---

## Option 1: Binary on VPS

Simplest path — single binary, systemd, Caddy reverse proxy.

### Architecture

```
Internet
   │
   ▼
Caddy (:80, :443, auto-TLS)
   │
   ▼ (reverse_proxy)
nuteo-web binary (:8080, internal only)
```

### Steps

1. SSH into your VPS:

   ```bash
   ssh user@your-vps
   ```

2. Install prerequisites:

   ```bash
   sudo apt update
   sudo apt install -y caddy
   # Go is only needed if you build on the VPS; otherwise skip
   ```

3. Create a deploy user and directory:

   ```bash
   sudo useradd -r -m -d /opt/nuteo-web -s /bin/bash nuteo
   sudo mkdir -p /opt/nuteo-web/{bin,content,web,deploy}
   sudo chown -R nuteo:nuteo /opt/nuteo-web
   ```

4. Copy artifacts from your local machine:

   ```bash
   # On your local machine, after `make build`:
   rsync -avz --delete \
     bin/nuteo-web content/ web/ deploy/ \
     nuteo@your-vps:/opt/nuteo-web/
   ```

5. Create `/opt/nuteo-web/.env` on the VPS with production values:

   ```bash
   sudo -u nuteo tee /opt/nuteo-web/.env > /dev/null <<EOF
   ENV=production
   ADDR=:8080
   SITE_URL=https://nuteo.example.com
   CONTACT_EMAIL_TO=hello@nuteo.example.com
   CONTACT_EMAIL_FROM=noreply@nuteo.example.com
   SMTP_HOST=smtp.resend.com
   SMTP_PORT=587
   SMTP_USER=resend
   SMTP_PASS=re_xxxxxxxxxx
   SMTP_USE_TLS=true
   CSRF_SECRET=$(openssl rand -hex 32)
   EOF
   sudo chmod 600 /opt/nuteo-web/.env
   ```

6. Install systemd unit:

   ```bash
   sudo tee /etc/systemd/system/nuteo-web.service > /dev/null <<'EOF'
   [Unit]
   Description=nuteo-web
   After=network.target

   [Service]
   Type=simple
   User=nuteo
   Group=nuteo
   WorkingDirectory=/opt/nuteo-web
   EnvironmentFile=/opt/nuteo-web/.env
   ExecStart=/opt/nuteo-web/bin/nuteo-web
   Restart=on-failure
   RestartSec=5s
   LimitNOFILE=65535

   [Install]
   WantedBy=multi-user.target
   EOF

   sudo systemctl daemon-reload
   sudo systemctl enable nuteo-web
   sudo systemctl start nuteo-web
   sudo systemctl status nuteo-web
   ```

7. Configure Caddy (auto-TLS):

   ```bash
   sudo tee /etc/caddy/Caddyfile > /dev/null <<EOF
   nuteo.example.com {
       encode zstd gzip
       reverse_proxy 127.0.0.1:8080

       header {
           Strict-Transport-Security "max-age=31536000; includeSubDomains"
           X-Content-Type-Options "nosniff"
           Referrer-Policy "strict-origin-when-cross-origin"
       }
   }
   EOF

   sudo systemctl reload caddy
   ```

8. Open firewall:

   ```bash
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   sudo ufw allow OpenSSH
   sudo ufw enable
   ```

9. Verify:

   ```bash
   curl -i https://nuteo.example.com/healthz
   # → HTTP/2 200
   # → content-length: 2
   # → body: ok
   ```

### Updating

```bash
# On local: rebuild
make build

# Push to VPS
rsync -avz bin/nuteo-web nuteo@your-vps:/opt/nuteo-web/bin/

# Restart on VPS
ssh nuteo@your-vps 'sudo systemctl restart nuteo-web'
```

### Logs

```bash
sudo journalctl -u nuteo-web -f   # follow live
sudo journalctl -u nuteo-web -n 100 --no-pager  # last 100 lines
```

---

## Option 2: Docker Compose with Caddy

What we ship in `deploy/`. Recommended for most users.

### Architecture

```
Internet
   │
   ▼
Caddy container (:80, :443, auto-TLS)
   │  (Docker network)
   ▼
nuteo-web container (:8080, internal)
```

### Steps

1. Copy the project to your VPS:

   ```bash
   scp -r . nuteo@your-vps:/opt/nuteo-web/
   ```

2. Create `.env` in `/opt/nuteo-web/` (not `deploy/`) with production values.

3. Edit `deploy/Caddyfile` — set your domain:

   ```caddy
   nuteo.example.com {
       ...
   }
   ```

4. Build and start:

   ```bash
   cd /opt/nuteo-web/deploy
   docker compose up -d --build
   ```

5. Check status:

   ```bash
   docker compose ps
   docker compose logs app
   docker compose logs caddy
   ```

### Updating

```bash
cd /opt/nuteo-web
git pull  # if using git
cd deploy
docker compose up -d --build
```

`docker compose up -d` does a **rolling recreate**:
- Starts the new container
- Waits for the new container to pass healthcheck
- Stops the old container

This gives near-zero downtime.

### Logs

```bash
docker compose logs -f app
docker compose logs -f caddy
```

To inspect logs from the host's syslog:

```bash
sudo journalctl -u docker
```

---

## Option 3: systemd service (no Docker)

Same as Option 1 but you **build on the VPS** directly.

1. Install Go 1.25+ on the VPS:

   ```bash
   wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
   echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
   source /etc/profile.d/go.sh
   ```

2. Clone and build:

   ```bash
   cd /opt
   sudo git clone https://github.com/saweek9/nuteo-web.git
   sudo chown -R nuteo:nuteo nuteo-web
   cd nuteo-web
   sudo -u nuteo go build -o bin/nuteo-web ./cmd/server
   ```

3. Continue with systemd + Caddy from Option 1.

---

## Option 4: Fly.io or Render

For zero-config PaaS deploys.

### Fly.io

```bash
fly launch --no-deploy
fly secrets set \
  CONTACT_EMAIL_TO=hello@nuteo.example.com \
  CONTACT_EMAIL_FROM=noreply@nuteo.example.com \
  SMTP_HOST=smtp.resend.com \
  SMTP_USER=resend \
  SMTP_PASS=$(cat .secrets/smtp-pass) \
  CSRF_SECRET=$(openssl rand -hex 32)
fly deploy
```

### Render

```bash
# Connect repo at https://render.com
# Set environment variables in dashboard
# Build: go build -o bin/nuteo-web ./cmd/server
# Start: bin/nuteo-web
# Health check: /healthz
```

Both auto-provision HTTPS.

---

## Option 5: Static export to CDN

For a pure-marketing site (no contact form backend), you can pre-render
HTML and serve from Cloudflare Pages / Netlify.

We don't ship a static-exporter today. To build one:

```go
// Sketch — add cmd/static/main.go
func main() {
    store := storage.New()
    store.LoadAll("content")
    for _, s := range store.Services {
        renderToFile("dist/services/"+s.Slug+"/index.html", "service.html", s)
    }
    // ... etc
}
```

For now, options 1–4 cover live sites.

---

## Backup strategy

The app is **stateless** — all content is in `content/` (markdown).
Backups = backing up `content/` + `.env`.

```bash
# Cron job — daily, 30-day retention
0 3 * * * tar -czf /backup/nuteo-web-$(date +\%Y\%m\%d).tar.gz \
  /opt/nuteo-web/content /opt/nuteo-web/.env \
  && find /backup -name "nuteo-web-*.tar.gz" -mtime +30 -delete
```

For database-backed extensions (see [07-extending.md](./07-extending.md)),
also back up the DB file.

---

## Monitoring

The app logs to stdout in JSON (one line per request). Capture and ship
to your aggregator of choice.

Recommended stack:
- **Logs**: `journalctl` → vector/logstash → Loki
- **Uptime**: UptimeRobot / Better Stack
- **Errors**: Sentry (track panics)
- **Métricas**: Prometheus + Grafana (add `/metrics` handler — see
  [07-extending.md](./07-extending.md))

Minimum viable:

```bash
# Healthcheck from external service
curl -fsS https://nuteo.example.com/healthz
```

If non-200, alert.

---

## Zero-downtime reloads

The app reads content **at startup** and doesn't watch for changes.
To pick up content edits without downtime:

### With systemd (Option 1)

The binary needs to drain in-flight requests before exiting. Go's
`http.Server.Shutdown(ctx)` does this — the systemd unit can
signal it correctly:

```ini
[Service]
ExecStop=/bin/kill -SIGTERM $MAINPID
TimeoutStopSec=30s  # give Shutdown() up to 30s to drain
```

Then `systemctl reload nuteo-web` (if you have a `ExecReload=` that
sends SIGHUP) or `systemctl restart nuteo-web` work fine.

### With Docker Compose (Option 2)

`docker compose up -d --build app` does rolling recreate. Caddy
keeps the old container alive until the new one is up.

### Watch mode (dev only)

For local development, use a watcher:

```bash
# Install air: go install github.com/cosmtrek/air@latest
air -c .air.toml
```

Sample `.air.toml`:

```toml
root = "."
tmp_dir = "build"
[build]
  cmd = "go build -o ./build/nuteo-web ./cmd/server"
  bin = "./build/nuteo-web"
  include_ext = ["go", "html", "md", "css", "js"]
  exclude_dir = ["bin", "build"]
[run]
  full_bin = "APP_ENV=dev ./build/nuteo-web"
```

---

## Scaling beyond one host

The app is stateless. To scale:

1. **Read replicas**: Run multiple instances behind Caddy / a load
   balancer. Each instance loads its own copy of `content/`. Use a
   shared filesystem (NFS, S3 via FUSE) if you want single-source-of-truth.

2. **DB-backed extensions**: When you add a blog (PostgreSQL) or
   comments (Redis), put the DB behind the app instances. The app
   remains stateless.

3. **CDN in front**: Put Cloudflare in front of Caddy. Static assets
   (CSS, JS, images) get served from edge — reduces load on origin.

---

## Disaster recovery

| Scenario | Recovery |
|---|---|
| Lost `content/` | Restore from backup (`tar.gz` daily snapshots) |
| Lost `.env` | Re-generate `CSRF_SECRET`, copy SMTP creds from password manager |
| Bad deploy | `docker compose down && docker compose up -d --build` (uses cached image; pull fresh if needed) |
| VPS gone | Provision new VPS, restore from backup, deploy — total ~15 min |
| TLS cert expired | Caddy auto-renews; if broken, `sudo systemctl reload caddy` |
| Disk full | Logs probably; rotate, prune old Docker images: `docker image prune -a` |
