# Content Authoring

All page content lives as **Markdown files** under `content/`. You don't
need to know HTML, Go, or any templating language to add or edit content.

## Directory structure

```
content/
├── services/        # /services and /services/:slug
│   ├── cloud-migration.md
│   ├── devops.md
│   ├── backend.md
│   └── security.md
├── portfolio/       # /portfolio and /portfolio/:slug
│   ├── logistics-migration.md
│   └── fintech-observability.md
└── posts/           # (template — not yet routed, future /blog)
```

Each `.md` file = one item. The filename's stem is the URL slug.

| File path | Renders at |
|---|---|
| `content/services/cloud-migration.md` | `/services/cloud-migration` |
| `content/portfolio/fintech-observability.md` | `/portfolio/fintech-observability` |

## File anatomy

Each file has two parts: **frontmatter** (YAML, between `---` markers)
and a **body** (Markdown).

```markdown
---
title: Cloud Migration & Infrastructure
slug: cloud-migration
summary: Lift, optimize, and operate your workloads on AWS, GCP, or Azure.
icon: cloud
order: 1
audience: [enterprise, startup]
tags: [aws, gcp, azure, kubernetes, terraform]
featured: true
cta_label: Discuss migration
cta_href: /contact?topic=cloud-migration
published_at: 2026-01-15
updated_at: 2026-08-01
---

## What we do

We design and execute end-to-end cloud migrations...

## Engagement model

- **Discovery (2 weeks)** — ...
- **Build (6–10 weeks)** — ...

## Outcomes we deliver

- Predictable monthly run costs
- Zero-downtime cutovers
- ...
```

The `---` markers are required. Everything between the markers is
YAML; everything after the second marker is Markdown.

## Field reference

### Common fields (services + portfolio + posts)

| Field | Type | Required | Notes |
|---|---|---|---|
| `title` | string | ✅ | Page heading; also used in `<title>` |
| `slug` | string | ✅ | URL segment, must match filename stem |
| `summary` | string | ✅ | Lead paragraph + meta description |
| `tags` | list | optional | Shown as labels |
| `featured` | bool | optional | If true, appears on home page |
| `order` | int | optional | Display order on list page (lower = first) |
| `published_at` | date | optional | ISO date. Defaults to file mtime |
| `updated_at` | date | optional | ISO date. Defaults to file mtime |

### Service-specific

| Field | Type | Notes |
|---|---|---|
| `icon` | string | Material/emoji name. Hint for icon picker |
| `audience` | list | `enterprise`, `startup`, `scaleup` |
| `cta_label` | string | Call-to-action button text |
| `cta_href` | string | CTA button URL |

### Portfolio-specific

| Field | Type | Notes |
|---|---|---|
| `client` | string | Client name (or "Confidential") |
| `industry` | string | e.g. `Logistics`, `FinTech` |
| `year` | int | Project year |
| `stack` | list | Technologies used |
| `image` | string | Hero image path (relative to `/static/`) |
| `url` | string | Live project URL if available |

### Post-specific

| Field | Type | Notes |
|---|---|---|
| `author` | string | Author name |

## Markdown features

The body uses [GitHub-Flavored Markdown](https://github.github.com/gfm/),
plus:

- **Tables** via pipe syntax
- **Task lists** `- [ ]` / `- [x]`
- **Strikethrough** `~~text~~`
- **Autolinks** `<https://example.com>`
- **Code blocks** with language hints (Go, Python, bash, yaml, ...)

Example code block:

````markdown
```yaml
apiVersion: v1
kind: Service
metadata:
  name: nuteo-web
```
````

Renders with syntax highlighting.

## Quickstart: add a new service

The fastest path is the helper script:

```bash
./scripts/new-content.sh service cloud-cost-optimization
```

This scaffolds a file with the right frontmatter keys:

```
content/services/cloud-cost-optimization.md
```

Then:
1. Edit the file — fill in `title`, `summary`, body
2. Set `order: <number>` to control display position
3. Set `featured: true` if it should appear on the home page
4. Restart the server (`make run`) to reload content

> The app loads content **at startup**, not on every request. Any
> change requires a restart to take effect. For zero-downtime reload,
> see [05-deployment.md](./05-deployment.md) for the rolling-restart
> recipe.

## Quickstart: add a new portfolio item

```bash
./scripts/new-content.sh portfolio my-cool-project
```

Edit the generated file and fill in `client`, `industry`, `year`,
`stack`, `summary`, and the body.

Set `featured: true` to surface it on the home page.

## Linking between pages

Use **relative URLs** in body content:

```markdown
See our [DevOps offering](/services/devops) and our
[observability case study](/portfolio/fintech-observability).
```

> Do **not** hardcode the full domain (e.g. `https://nuteo.example.com/...`).
> Relative URLs work in any deployment.

Images inside markdown should go in `web/static/images/` and use
`/static/images/`:

```markdown
![Architecture diagram](/static/images/cloud-architecture.png)
```

## Frontmatter pitfalls

- **Quotes**: Strings with `:` need quoting in YAML:

  ```yaml
  title: "Cloud Migration: from monolith to microservices"
  ```

- **Booleans**: Lowercase only — `true` / `false`, not `True` / `TRUE`.

- **Dates**: Use ISO format `YYYY-MM-DD`. Other formats may parse but
  ISO is safest.

- **Empty arrays**: Use `[]` not `[ ]` (with whitespace), and never omit
  a list field if other items have it — YAML treats absent vs empty
  inconsistently.

- **Slug mismatch**: If `slug: foo` doesn't match the filename
  `foo.md`, the page will 404. Either rename the file or remove the
  slug field (default = filename stem).

## Preview locally

After editing content:

```bash
make run   # foreground — see changes immediately on restart
```

Or use a markdown previewer for richer editing:

```bash
# VS Code: open the .md file and hit Cmd/Ctrl+K, V
# Or use grip
grip content/services/cloud-migration.md
```

## What you CAN'T do from content alone

- Change page layout — requires editing `web/templates/pages/*.html`
- Add new routes — requires editing `cmd/server/main.go`
- Change site metadata (footer, navigation) — requires editing
  `web/templates/partials/*.html` or `internal/handlers/pages.go`

Those are **developer** changes, not content changes. See the rest of
this manual.
