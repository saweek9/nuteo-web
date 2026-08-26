# Troubleshooting

A field guide to "the app isn't doing the thing."

Each section: **symptom → diagnosis → fix**.

---

## App won't start

### Symptom: `config load err="config: env: required environment variable X is not set"`

**Cause**: A required env var is missing.

**Fix**: Either set it in `.env` or export it in your shell:

```bash
echo 'CONTACT_EMAIL_TO=hello@example.com' >> .env
echo 'CSRF_SECRET='$(openssl rand -hex 32) >> .env
make run
```

### Symptom: `bind: address already in use`

**Cause**: Port 8080 is occupied by another process.

**Fix**:

```bash
# Find what's using it
sudo lsof -i :8080
# or
sudo ss -tlnp | grep 8080

# Either kill it or use a different port
ADDR=:8081 make run
```

### Symptom: `template not loaded` / `templates not loaded`

**Cause**: `tmpls` middleware didn't run before the handler. Almost
always means a route was registered before the middleware in
`cmd/server/main.go`.

**Fix**: Check ordering — middleware must be `r.Use()`d **before** any
route handler:

```go
r := gin.New()
r.Use(middleware.RequestID())      // ← middleware first
r.Use(middleware.SecurityHeaders())
r.GET("/", ...)                    // ← routes after
```

### Symptom: Go build fails with `undefined: X`

**Cause**: A function or type is missing — usually after a partial edit.

**Fix**: Run `go build ./...` (not `go build ./cmd/server`) to see
**all** errors. Read the column number in the error message.

---

## Pages render but content is wrong

### Symptom: Missing fields like `.Service.Title` shows empty

**Cause**: Either the markdown frontmatter is missing the field, or
the slug doesn't match.

**Diagnosis**:

```bash
cd /home/admin_nuteo/nuteo-web
go run ./scripts/debug-load/main.go
```

This prints the parsed models as JSON. Look for empty strings.

**Fix**: Edit the markdown file. Restart the server.

### Symptom: Service detail returns 404 for an existing file

**Cause**: `slug:` in frontmatter doesn't match the filename, or the
file isn't where it should be.

**Fix**: Either rename the file or remove the `slug` line (default =
filename stem).

### Symptom: Body markdown renders as escaped text (`<h2>` visible)

**Cause**: Template uses `{{.Service.BodyHTML}}` instead of
`{{safeHTML .Service.BodyHTML}}`.

**Fix**: Add `safeHTML` wrapper in the template:

```html
{{safeHTML .Service.BodyHTML}}
```

### Symptom: List page shows services out of order

**Cause**: `order` field missing or wrong.

**Fix**: Each service markdown file should have an `order:` field. Lower
= earlier. Numbers don't need to be consecutive — gaps are OK.

---

## Contact form

### Symptom: Form submission returns 400 with "Please fill in name, email, and message"

**Cause**: Either missing required fields, or `consent=on` not sent.

**Diagnosis**:

```bash
curl -v -X POST http://localhost:8080/contact \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --cookie /tmp/cookies.txt \
  --data-binary "_csrf=$TOKEN&name=Test&email=test@example.com&topic=hello&message=This+is+a+test+message+with+enough+chars&consent=on"
```

**Fix**:
- Check that the browser is sending `consent=on` (tick the box)
- Check that the field name is `consent` (case-sensitive)
- Check that `_csrf` cookie matches `_csrf` form field

### Symptom: Form submits, shows "Thanks", but no email arrives

**Cause**: SMTP not configured or wrong credentials.

**Diagnosis**:

```bash
# Check app logs for "send" / "email"
journalctl -u nuteo-web | grep -i email

# Or run with debug
ENV=development make run
```

Look for SMTP connection errors. Common issues:

- **Wrong port**: Most SMTP uses 587 (STARTTLS) or 465 (TLS). Check your
  provider's docs.
- **Wrong credentials**: For Resend, `SMTP_USER=resend` and
  `SMTP_PASS=re_xxx` (the API key).
- **Firewall blocked port 25**: Some providers / VPSes block outgoing
  port 25. Use port 587 or 465.

### Symptom: Form says "Please confirm you've read the privacy notice"

**Cause**: Consent checkbox was not ticked.

**Fix**: Tick the checkbox. (Server-side check is intentional — GDPR.)

### Symptom: Bots are spamming the contact form

**Cause**: Honeypot is being filled (good bots ignore hidden fields; bad
bots fill everything).

**Fix**: Currently we silently 200 the honeypot. To block harder:

1. **Add hCaptcha** (see [07-extending.md](./07-extending.md))
2. **Add IP rate limiting** (`internal/middleware/ratelimit.go`)
3. **Log spam attempts** to a separate file, then ban IPs at firewall

### Symptom: CSRF error (form refuses to submit)

**Cause**: The `_csrf` cookie was lost (browser rejected third-party
cookies, expired, etc.).

**Fix**: Refresh the page to get a fresh token.

---

## Static assets

### Symptom: CSS / JS / images return 404

**Cause**: `STATIC_DIR` env var points to wrong directory.

**Diagnosis**:

```bash
ls -la /home/admin_nuteo/nuteo-web/web/static/css/site.css
echo $STATIC_DIR
```

**Fix**: Set `STATIC_DIR=./web/static` (or absolute path) in `.env`.

### Symptom: HTMX not working (forms do full reload)

**Cause**: `htmx.min.js` not loaded, or CSP blocks it.

**Diagnosis**:

```bash
curl -i http://localhost:8080/static/js/htmx.min.js
```

Should return 200 with `application/javascript`.

**Fix**: If CSP is the issue, check `Content-Security-Policy` header
in response. CSP is set by middleware in `internal/middleware/`.

---

## Performance

### Symptom: First request after deploy is slow

**Cause**: First request to a cold start compiles templates. Subsequent
requests are fast.

**Fix**: Add a warmup step in deployment:

```bash
make build && \
  bin/nuteo-web & \
  sleep 1 && \
  curl -fs http://localhost:8080/healthz && \
  kill %1
```

### Symptom: Memory grows unbounded

**Cause**: Memory leak — usually a goroutine that never exits.

**Diagnosis**:

```bash
# Check RSS over time
while true; do
  ps -o rss= -p $(pgrep -f nuteo-web)
  sleep 60
done
```

**Fix**: Use `pprof`:

```go
import _ "net/http/pprof"
// in main:
go http.ListenAndServe("localhost:6060", nil)
```

Then capture heap:

```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## Deployment

### Symptom: `docker compose up` fails with port conflict

**Cause**: Another container is using port 80 or 443.

**Diagnosis**:

```bash
sudo lsof -i :80
sudo lsof -i :443
```

**Fix**: Stop the conflicting container, or change Caddy ports (and
update DNS / firewall).

### Symptom: TLS cert won't provision

**Cause**: Caddy needs port 80 to validate via ACME.

**Diagnosis**:

```bash
docker compose logs caddy | grep -i "acme\|tls\|cert"
```

**Fix**: Ensure port 80 is reachable from internet. Some cloud
providers block it by default — check security group / firewall.

### Symptom: Logs spam "no space left on device"

**Fix**:

```bash
# Docker images
docker image prune -a
# Logs
sudo journalctl --vacuum-time=7d
# /var/lib/docker (overlay2)
du -sh /var/lib/docker
```

---

## Logs

### Where to find logs

| Setup | Logs |
|---|---|
| Local | stdout (your terminal) |
| systemd | `journalctl -u nuteo-web` |
| Docker | `docker compose logs app` |
| Caddy | `docker compose logs caddy` (separate container) |

### Reading JSON logs

Each log line is one JSON object. Pipe to `jq`:

```bash
journalctl -u nuteo-web -o cat --since today | jq -c 'select(.level=="ERROR")'
docker compose logs app | jq -c 'select(.status>=500)'
```

### What gets logged

- Every HTTP request (method, path, status, duration, IP, request ID)
- App lifecycle (start, stop)
- Config errors
- Panic stack traces
- Email send results
- Content load results (services/portfolio/posts count)

### What's NOT logged

- Form values (could contain PII; only logged on error)
- User passwords / API keys (always redacted)
- Email body

---

## When all else fails

1. **Run with debug**:

   ```bash
   ENV=development make run
   ```

   Gin prints every route registration and request.

2. **Check the obvious**: Is the binary up to date? Did you restart?
   Did you copy `.env`?

3. **Use the debug loader**:

   ```bash
   cd /home/admin_nuteo/nuteo-web
   go run ./scripts/debug-load/main.go
   ```

4. **Open a shell in the running container** (Docker):

   ```bash
   docker compose exec app sh
   ```

5. **Use git to bisect**:

   ```bash
   git log --oneline
   git checkout HEAD~5  # try a known-good version
   # if it works, the bug is in commits 4-0
   ```

6. **Ask in the issues** (if applicable).
