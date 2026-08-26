# Auto-start (systemd)

nuteo-web runs as a **systemd service** that auto-starts on boot and
auto-restarts on failure.

## Current state (on this host)

```bash
scripts/service.sh status
```

Expected output (after install):

```
● nuteo-web.service - nuteo-web — Corporate site for nuteo solution (Go + Gin)
     Loaded: loaded (/etc/systemd/system/nuteo-web.service; enabled; preset: enabled)
     Active: active (running) since ...
   Main PID: 12345 (nuteo-web)
     Memory: 5.3M (max: 256M, available: 250.6M, peak: 5.3M)
```

## Service file

Located at `/etc/systemd/system/nuteo-web.service`:

```ini
[Unit]
Description=nuteo-web — Corporate site for nuteo solution (Go + Gin)
Documentation=https://github.com/saweek9/nuteo-web
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=120
StartLimitBurst=5

[Service]
Type=simple
User=admin_nuteo
Group=admin_nuteo
WorkingDirectory=/home/admin_nuteo/nuteo-web
EnvironmentFile=/home/admin_nuteo/nuteo-web/.env
ExecStart=/home/admin_nuteo/nuteo-web/bin/nuteo-web
Restart=on-failure
RestartSec=5s
TimeoutStopSec=30s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nuteo-web
LimitNOFILE=65535
MemoryMax=256M

[Install]
WantedBy=multi-user.target
```

### What each setting means

| Setting | Why |
|---|---|
| `User=admin_nuteo` | Run as non-root user (security) |
| `Group=admin_nuteo` | Match group for file ownership |
| `WorkingDirectory=...` | App expects to run from project root (for relative paths like `./content`) |
| `EnvironmentFile=...` | Load env vars from `.env` (secrets stay out of the unit file) |
| `ExecStart=.../bin/nuteo-web` | The compiled binary |
| `Restart=on-failure` | Auto-restart ONLY on non-zero exit (not on clean `systemctl stop`) |
| `RestartSec=5s` | Wait 5 seconds before restart (prevents tight crash loop) |
| `TimeoutStopSec=30s` | On SIGTERM, give the app 30s to drain in-flight HTTP requests |
| `StartLimitIntervalSec=120` + `StartLimitBurst=5` | If 5 crashes within 120s, give up auto-restart (prevents infinite loops) |
| `StandardOutput=journal` + `StandardError=journal` | Capture logs in systemd journal |
| `SyslogIdentifier=nuteo-web` | Tag for log filtering (`journalctl -u nuteo-web`) |
| `MemoryMax=256M` | Cap RSS at 256MB (OOM-kills the app before it eats the box) |

## Helper script

`scripts/service.sh` wraps `systemctl` for convenience:

```bash
scripts/service.sh status          # full status
scripts/service.sh start           # start (no-op if already running)
scripts/service.sh stop            # stop
scripts/service.sh restart         # restart
scripts/service.sh logs -n 50      # last 50 log lines
scripts/service.sh follow          # tail -f the logs
scripts/service.sh build           # rebuild + restart (for after code edits)
scripts/service.sh health          # one-shot HTTP /healthz check
scripts/service.sh enable-boot     # ensure auto-start on boot
scripts/service.sh disable-boot    # turn off auto-start
```

## Operating procedures

### After a code edit

```bash
cd /home/admin_nuteo/nuteo-web
# Edit some .go file
scripts/service.sh build    # runs `make build` + `systemctl restart`
```

The script:
1. Runs `make build` → produces `bin/nuteo-web`
2. Runs `sudo systemctl restart nuteo-web`
3. Shows new status (PID, memory, etc.)

### After a content edit

```bash
# Edit content/services/new-service.md
scripts/service.sh restart
```

The app reads content at startup, so any content change requires a
restart. There is **no hot reload** for content.

### Inspect logs

```bash
# Last 100 lines
scripts/service.sh logs -n 100

# Live tail
scripts/service.sh follow
```

JSON output — pipe to `jq` for pretty-printing:

```bash
scripts/service.sh logs -n 200 | jq -c 'select(.status >= 500)'
```

### Disable auto-start (without uninstalling)

```bash
scripts/service.sh disable-boot
```

The service is still available — start it manually any time with
`scripts/service.sh start`. To re-enable boot start:

```bash
scripts/service.sh enable-boot
```

## Boot behavior

On boot, systemd:

1. Loads the unit file
2. Resolves dependencies (`After=network-online.target` waits for network)
3. Starts the service
4. Captures stdout/stderr to journal
5. Retries on failure (up to 5 times in 120 seconds)
6. Gives up (and marks `failed`) if crashes continue

To verify boot will work:

```bash
# Manually trigger boot-time conditions
sudo systemctl daemon-reload
sudo systemctl enable nuteo-web.service
sudo systemctl status nuteo-web.service  # should show "enabled"

# (Optional) Simulate crash → restart cycle once
MAINPID=$(sudo systemctl show nuteo-web --property=MainPID --value)
sudo kill -9 $MAINPID
sleep 6
sudo systemctl is-active nuteo-web.service  # should be "active"
```

## Uninstalling

To remove auto-start entirely:

```bash
sudo systemctl stop nuteo-web.service
sudo systemctl disable nuteo-web.service
sudo rm /etc/systemd/system/nuteo-web.service
sudo systemctl daemon-reload
```

The project files stay in `/home/admin_nuteo/nuteo-web/` — you can
start manually with `cd /home/admin_nuteo/nuteo-web && make run`.

## Manual install (on a new VPS)

```bash
# 1. Install prerequisites (Go for building, nothing else needed at runtime)
sudo apt install -y golang-go

# 2. Create admin_nuteo user (or use an existing one)
sudo useradd -m -s /bin/bash admin_nuteo

# 3. Copy project
sudo cp -r /path/to/nuteo-web /home/admin_nuteo/
sudo chown -R admin_nuteo:admin_nuteo /home/admin_nuteo/nuteo-web

# 4. Build
cd /home/admin_nuteo/nuteo-web
make build

# 5. Create .env (production values)
cp .env.example .env
nano .env  # fill in SMTP, CSRF_SECRET, etc.
chmod 600 .env

# 6. Install systemd service
sudo tee /etc/systemd/system/nuteo-web.service > /dev/null <<'EOF'
[Unit]
Description=nuteo-web — Corporate site for nuteo solution
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=admin_nuteo
Group=admin_nuteo
WorkingDirectory=/home/admin_nuteo/nuteo-web
EnvironmentFile=/home/admin_nuteo/nuteo-web/.env
ExecStart=/home/admin_nuteo/nuteo-web/bin/nuteo-web
Restart=on-failure
RestartSec=5s
TimeoutStopSec=30s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nuteo-web
LimitNOFILE=65535
MemoryMax=256M

[Install]
WantedBy=multi-user.target
EOF

# 7. Enable and start
sudo systemctl daemon-reload
sudo systemctl enable nuteo-web.service
sudo systemctl start nuteo-web.service
sudo systemctl status nuteo-web.service

# 8. Verify
curl http://localhost:8080/healthz  # → "ok"
```

## Troubleshooting

See [08-troubleshooting.md](./08-troubleshooting.md) — search for
"auto-start", "systemd", or "service".

Quick checks:

```bash
# Is the service loaded?
sudo systemctl status nuteo-web.service

# Is it enabled on boot?
sudo systemctl is-enabled nuteo-web.service

# Latest logs?
sudo journalctl -u nuteo-web.service -n 50 --no-pager

# Force a reset after repeated failures
sudo systemctl reset-failed nuteo-web.service
sudo systemctl start nuteo-web.service
```

## Why systemd (not Docker-only)?

The simpler option is **run as a systemd service**, since:

- No Docker daemon dependency
- No overlay/network overhead
- ~80ms faster cold start
- Logs go to journald (built-in)
- Direct binary (no image build step)

Use Docker Compose when:

- You're deploying to Fly.io / Render / etc.
- You want Caddy in the same Compose file (one `docker compose up`)
- You need reproducible environments across hosts

See [05-deployment.md](./05-deployment.md) for the Docker path.
