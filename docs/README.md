# nuteo-web Documentation

This directory contains the full operator + developer manual for
**nuteo-web**, the corporate website for nuteo solution.

## 📚 Documents

| Doc | Audience | Read time |
|---|---|---|
| [Quickstart](./01-quickstart.md) | Anyone | 5 min |
| [Content Authoring](./02-content-authoring.md) | Marketing / sales | 10 min |
| [Template Reference](./03-templates.md) | Designers / devs | 15 min |
| [Configuration](./04-configuration.md) | DevOps / SysAdmin | 10 min |
| [Deployment](./05-deployment.md) | SysAdmin | 20 min |
| [Architecture](./06-architecture.md) | Senior dev / architect | 15 min |
| [Extending](./07-extending.md) | Go dev | 30 min |
| [Troubleshooting](./08-troubleshooting.md) | On-call / ops | 10 min |
| [Auto-start (systemd)](./09-auto-start.md) | SysAdmin | 5 min |

**Total**: ~2 hours from "what is this?" to "I can deploy and extend it."

## 🗺️ Reading Order

```
Quickstart
  └── Content Authoring     (if you'll be writing content)
  └── Template Reference    (if you'll be changing templates)
  └── Configuration         (if you'll be deploying)
  └── Deployment            (if you'll be deploying)
       └── Troubleshooting  (when something breaks)
  └── Architecture          (if you'll be extending)
  └── Extending             (when you're ready to add features)
```

## 🎯 TL;DR

```bash
# Build and run
make build && make run

# Add a service or portfolio item
./scripts/new-content.sh service cloud-cost-optimization

# Deploy to production
cd deploy && docker compose up -d
```

That's it. Everything else in this directory is reference material for
when you need it.

## 📝 Conventions used in this manual

- **`code`** is shell command or file path
- `> note` is a side-note or warning
- `[link](#anchor)` is an intra-doc anchor
- Code blocks have language tags (` ```bash `, ` ```go `, ` ```yaml `)
- "You" means the human operator unless otherwise specified
