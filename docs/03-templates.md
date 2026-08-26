# Template Reference

How the HTML templates work, how to change them, and what data they
receive.

## Template engine

We use Go's standard library [`html/template`](https://pkg.go.dev/html/template).
It's type-safe-ish (templates are parsed at startup, missing fields
crash early), auto-escapes user input, and requires zero dependencies.

Templates are compiled **once at startup** into a map keyed by page
name. When a request comes in, `renderPage` looks up the template
matching the route, fills in the data, and writes the output.

## File layout

```
web/templates/
├── pages/
│   ├── home.html              ← / (homepage)
│   ├── services.html          ← /services (list)
│   ├── service.html           ← /services/:slug (detail)
│   ├── portfolio.html         ← /portfolio (list)
│   ├── project.html           ← /portfolio/:slug (detail)
│   ├── about.html             ← /about
│   ├── contact.html           ← /contact (GET)
│   ├── contact_thanks.html    ← /contact (POST success)
│   └── 404.html               ← NotFound handler
└── partials/
    ├── header.html            ← {{template "header" .}}
    └── footer.html            ← {{template "footer" .}}
```

Adding a new page = creating a new file under `pages/` and registering
it in `internal/handlers/page.go` (the `pageFiles` slice).

## Page template structure

Every page template is a **self-contained document** that starts with
`{{template "header" .}}` and ends with `{{template "footer" .}}`.

```html
{{template "header" .}}
<section class="container section">
  <h1>{{.PageTitle}}</h1>
  <p>{{.PageDescription}}</p>
  <!-- page-specific markup -->
</section>
{{template "footer" .}}
```

`{{template "header" .}}` includes the registered `header` template
with the current data context (the `.`).

## Data passed to every template

Every page receives `gin.H` (a `map[string]any`) with at least these keys:

| Key | Type | Description |
|---|---|---|
| `SiteName` | string | e.g. `nuteo solution` |
| `SiteURL` | string | e.g. `https://nuteo.example.com` |
| `PageTitle` | string | The current page's title (used in `<title>`) |
| `PageDescription` | string | Meta description |
| `PageURL` | string | The request path, e.g. `/services` |
| `LogoPath` | string | Logo URL |
| `GitHubURL` / `LinkedInURL` / `TwitterURL` | string | Social links |

Pages with additional data add more keys:

### `home.html`

```yaml
FeaturedServices:  []models.Service     # only services where featured: true
FeaturedPortfolio: []models.Portfolio   # only portfolio where featured: true
```

### `services.html`

```yaml
Services: []models.Service   # all services, sorted by .order
```

### `service.html`

```yaml
Service: models.Service       # one service object
```

### `portfolio.html`

```yaml
Projects: []models.Portfolio  # all projects
```

### `project.html`

```yaml
Project: models.Portfolio      # one project
```

### `contact.html`

```yaml
ContactEmail: string          # recipient address (from env)
CSRFToken:    string          # per-request token
Error:        string          # validation error message (empty on GET)
Name/Email/Company/Topic/Message: string  # previous submission (on error)
Consent:      bool            # previous consent (on error)
```

### `contact_thanks.html`

```yaml
Inquiry: models.Inquiry       # the submitted inquiry (echo back)
```

## Model fields

### `models.Service`

```go
type Service struct {
    Title       string    // "Cloud Migration & Infrastructure"
    Slug        string    // "cloud-migration"
    Summary     string    // "Lift, optimize, and operate..."
    Icon        string    // icon name, e.g. "cloud"
    Order       int       // display order
    Audience    []string  // ["enterprise", "startup"]
    Tags        []string  // ["aws", "gcp", "azure"]
    Featured    bool      // show on home?
    CTALabel    string    // "Discuss migration"
    CTAHref     string    // "/contact?topic=cloud-migration"
    PublishedAt time.Time
    UpdatedAt   time.Time
    BodyHTML    string    // rendered HTML — use with safeHTML
    BodyMD      string    // raw markdown
}
```

### `models.Portfolio`

```go
type Portfolio struct {
    Title       string
    Slug        string
    Summary     string
    Client      string    // "Confidential (APAC logistics)"
    Industry    string    // "Logistics"
    Year        int       // 2026
    Stack       []string  // ["Go", "Kubernetes", "Terraform"]
    Tags        []string
    Image       string    // "/static/images/foo.png"
    URL         string    // live URL if any
    Featured    bool
    PublishedAt time.Time
    BodyHTML    string
    BodyMD      string
}
```

### `models.Post` (for future /blog)

```go
type Post struct {
    Title       string
    Slug        string
    Summary     string
    Author      string
    Tags        []string
    PublishedAt time.Time
    UpdatedAt   time.Time
    BodyHTML    string
    BodyMD      string
}
```

### `models.Inquiry` (contact form)

```go
type Inquiry struct {
    ID         string    // uuid
    ReceivedAt time.Time
    Name       string
    Email      string
    Company    string
    Phone      string
    Topic      string
    Message    string
    Website    string    // honeypot
    IP         string
    UserAgent  string
    Referrer   string
    SpamScore  float64
    Consent    bool
}
```

## Template functions

The template engine has these custom functions:

| Function | Signature | Use |
|---|---|---|
| `safeHTML` | `safeHTML(string) template.HTML` | Mark a string as safe HTML (skips escaping). Use for `BodyHTML` fields. |
| `safeJS` | `safeJS(string) template.JS` | Mark as safe JS (e.g. inline JSON) |
| `safeCSS` | `safeCSS(string) template.CSS` | Mark as safe CSS |

**Important**: Never call `safeHTML` on user input. Only on content
you control (rendered from markdown).

Example:

```html
<article>
  <h1>{{.Service.Title}}</h1>
  <p class="lede">{{.Service.Summary}}</p>
  {{safeHTML .Service.BodyHTML}}
</article>
```

`{{.Service.BodyHTML}}` without `safeHTML` would show the HTML as
escaped text (`<h2>...</h2>` visible to the user).

## Built-in functions

Go's `html/template` also provides:

| Function | Example | Result |
|---|---|---|
| `len` | `{{len .Services}}` | `4` |
| `index` | `{{index .Services 0}}` | first service |
| `printf` | `{{printf "Hello, %s" .Name}}` | "Hello, X" |
| `and` / `or` / `not` | `{{if and .Featured .Published}}` | boolean |
| `eq` / `ne` / `lt` / `le` / `gt` / `ge` | `{{if eq .Order 1}}` | comparison |
| `range` | `{{range .Services}}...{{end}}` | loops |
| `with` | `{{with .Inquiry}}...{{end}}` | scoped block |
| `if` / `else` / `end` | `{{if .Error}}...{{else}}...{{end}}` | conditionals |

## Common patterns

### Looping over a list with join

```html
<ul class="tags">
  {{range .Service.Tags}}
    <li>{{.}}</li>
  {{end}}
</ul>
```

### Optional field (no fallback)

```html
{{with .Service.Image}}
  <img src="{{.}}" alt="">
{{end}}
```

### Date formatting

```html
<time datetime="{{.Service.PublishedAt.Format "2006-01-02"}}">
  {{.Service.PublishedAt.Format "January 2, 2006"}}
</time>
```

> Go uses a **reference time** `Mon Jan 2 15:04:05 MST 2006` as the format.
> The numbers mean: 1=month, 2=day, 3=hour, 4=minute, 5=second, 6=year, 7=timezone.

### Conditional class

```html
<a href="/services" class="{{if eq .PageURL "/services"}}active{{end}}">
  Services
</a>
```

## Adding a new page

1. Create `web/templates/pages/newpage.html`:

   ```html
   {{template "header" .}}
   <section class="container section">
     <h1>{{.PageTitle}}</h1>
     <p>{{.PageDescription}}</p>
   </section>
   {{template "footer" .}}
   ```

2. Add to `internal/handlers/page.go`:

   ```go
   var pageFiles = []string{
       "home.html",
       "services.html",
       // ... existing
       "newpage.html",  // ← add
   }
   ```

3. Add a handler in `internal/handlers/pages.go`:

   ```go
   func (d *Deps) NewPage(c *gin.Context) {
       data := baseData(c, d, "New Page")
       data["PageDescription"] = "What this page is about."
       renderPage(c, data)
   }
   ```

4. Register the route in `cmd/server/main.go`:

   ```go
   r.GET("/newpage", handlers.SetPage("newpage.html"), deps.NewPage)
   ```

5. Rebuild: `make build`

## Layout template anatomy

`web/templates/partials/header.html`:

```html
{{define "header"}}<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="{{.PageDescription}} — {{.SiteName}}">
  <meta name="theme-color" content="#0f172a">
  <link rel="icon" type="image/svg+xml" href="/favicon.ico">
  <link rel="stylesheet" href="/static/css/site.css">
  <title>{{.PageTitle}} · {{.SiteName}}</title>
</head>
<body>
  <header class="site-header">
    <div class="container">
      <a class="brand" href="/">
        <img src="{{.LogoPath}}" alt="{{.SiteName}} logo" width="32" height="32">
        <span>{{.SiteName}}</span>
      </a>
      <nav class="site-nav" aria-label="Primary">
        <a href="/services" {{if eq .PageURL "/services"}}class="active"{{end}}>Services</a>
        <a href="/portfolio" {{if eq .PageURL "/portfolio"}}class="active"{{end}}>Portfolio</a>
        <a href="/about" {{if eq .PageURL "/about"}}class="active"{{end}}>About</a>
        <a href="/contact" class="btn btn-primary {{if eq .PageURL "/contact"}}active{{end}}">Contact</a>
      </nav>
    </div>
  </header>
  <main id="main">
```

`web/templates/partials/footer.html`:

```html
{{define "footer"}}  </main>
  <footer class="site-footer">
    ...
  </footer>
  <script src="/static/js/htmx.min.js"></script>
</body>
</html>{{end}}
```

> The leading space and trailing `{{end}}` are intentional — they
> allow `{{define "footer"}}` and `{{define "header"}}` blocks to
> coexist with the page's own `{{template "header" .}}` invocation.

## CSS classes reference

The base stylesheet (`web/static/css/site.css`) provides these utility
classes:

| Class | Purpose |
|---|---|
| `.container` | Max-width 1100px centered with horizontal padding |
| `.section` | Vertical spacing (padding-block: 4rem) |
| `.section.narrow` | Narrower content column |
| `.grid` | CSS grid (3-column responsive) |
| `.grid-2` / `.grid-3` | Explicit grid templates |
| `.card` | White card with shadow + border-radius |
| `.hero` | Hero section with larger typography |
| `.lede` | Larger intro paragraph |
| `.meta` | Small gray text (for dates, industries) |
| `.alert.error` / `.alert.success` | Inline notification |
| `.btn` | Button base |
| `.btn-primary` | Primary CTA button |
| `.tag` | Pill-shaped tag |

## HTMX usage (optional)

HTMX 1.9.12 is loaded on every page (`web/static/js/htmx.min.js`). It's
available but **not currently used** by templates — it's there for
future enhancements.

To use HTMX in a form, e.g. for inline validation:

```html
<form hx-post="/contact" hx-target="#result" hx-swap="innerHTML">
  ...
</form>
<div id="result"></div>
```

Currently the form does a full-page POST. See
[07-extending.md](./07-extending.md) for how to add HTMX-powered
endpoints.

## Pitfalls

1. **Forgetting `{{template "header" .}}`** — page renders without the
   nav/footer. Always start with `{{template "header" .}}` and end with
   `{{template "footer" .}}`.

2. **Outputting raw user input as HTML** — XSS vulnerability. Always
   use `{{.X}}` (escaped). Only use `{{safeHTML .X}}` on content YOU
   produced (markdown rendering).

3. **Adding fields to the model without updating templates** — Go
   templates fail at runtime if you reference a non-existent field.
   Rebuild and check the logs.

4. **Renaming a template file without updating `pageFiles`** — the
   page returns 404 because the file isn't compiled into the map.

5. **Editing `partials/*.html` mid-deploy** — partials are loaded into
   every page template at startup. Any change requires a rebuild.
