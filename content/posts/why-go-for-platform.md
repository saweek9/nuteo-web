---
title: Why we picked Go for the nuteo-web platform (and when you shouldn't)
slug: why-go-for-platform
summary: "Go's single-binary deploys, tiny memory footprint, and sub-millisecond response times made it the obvious choice. But it's not for every team."
author: sawee kumkubkij
tags: [golang, architecture, devops]
featured: true
published_at: 2026-07-22
---

## TL;DR

We picked Go for our internal platform and the nuteo-web corporate site
because the deploy model, runtime cost, and performance profile fit our
needs better than Node.js or Python. This post is for teams considering
Go for their own platform — what we got right, what we'd change, and
when Go is the wrong choice.

## Why Go for our use case

Our requirements were unusual but specific:

1. **Single binary deploy** — no `node_modules`, no virtualenv, no
   container-with-Python-runtime. `scp` a 22 MB binary, run it.
2. **Sub-100ms cold start** — so we could `systemctl restart` without
   users noticing.
3. **< 50 MB RAM** under normal load — because we self-host on small
   VPSes.
4. **Long-running, low-maintenance** — once deployed, we don't want
   to think about it for years.

Go hits all four. The same code in Node would have needed ~150 MB
RAM and a container image. Python (FastAPI) similar.

## The deployment story is the real win

This is the underrated advantage. With Go:

- **No Dockerfile needed for simple apps.** `make build && scp binary
  server && ssh server './binary &'` — that's it.

- **Reproducible deploys by construction.** A Go binary with `go
  build -ldflags="-s -w"` is byte-identical for the same source
  commit. Good luck getting that out of npm.

- **Container images are tiny.** A scratch-based image for a 22 MB
  Go binary is ~25 MB total. A typical Node container is 200+ MB
  before your code runs.

- **Memory is predictable.** Go doesn't have a JIT warmup. The 5 MB
  you see at startup is the 5 MB you'll see at 3am under load.

## Where Go is genuinely the right choice

- **Network services** that need low latency and predictable memory:
  APIs, reverse proxies, ingress controllers
- **CLI tools** that ship to users (kubectl, terraform, gh, hugo,
  docker, ...)
- **Long-running daemons** that need to be reliable and forgettable
- **Container workloads** where you want fast startup + low memory
- **Infrastructure tooling** (Terraform providers, controllers,
  operators)

## Where Go is the wrong choice

We'd push back if:

- **Your team has zero Go experience** and a tight deadline. The
  learning curve is real, and Go's strictness (no generics until
  recently, error handling is verbose) is a culture shift if you're
  coming from Python or Ruby.

- **You're building a CRUD-heavy internal tool** with lots of
  templating. Python + Django or Node + Next.js will ship faster
  because of the ecosystem. Don't pick Go to be cool.

- **Your domain has heavy math or data science**. Go's data science
  story is much weaker than Python's (numpy/scipy ecosystem).

- **You need a real-time web frontend.** Go templates are fine for
  server-rendered HTML, but if you want a SPA, you'll need a separate
  frontend stack anyway — might as well pick the stack that's best
  for the frontend work.

## What we learned the hard way

### 1. The dependency story is spartan

`go.mod` works, but there's no equivalent of `npm` or `pip` for
discovering libraries. You'll find yourself writing more code than
you'd expect (CSV parsing, JSON streaming, etc.). The standard library
is excellent but everything else is a separate dependency.

### 2. Error handling is verbose

```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doSomething: %w", err)
}
```

That pattern repeats everywhere. It catches bugs. It also produces a
lot of code. After 6 months you'll stop noticing.

### 3. Build times get long

A medium-sized Go project (50k LOC) has 5-15 second clean builds.
That's fine for `go build` but a non-starter for tight inner loops.
Solutions: build cache, `go test -short`, Bazel for monorepos.

### 4. Goroutine debugging is hard

You have 1000+ goroutines at any moment. `dlv` works for small cases
but doesn't scale. Plan for `pprof` from day one.

### 5. No good ORM

`database/sql` + `sqlx` is the standard. For relational work, GORM
exists but is heavy. For everything else, you write SQL. This is
**good** in the long run but requires more up-front discipline than
Django or Rails.

## The verdict

For our platform (long-running, low-maintenance, self-hosted, no SPA),
Go was the right choice. For a B2C SaaS frontend, it would have been
the wrong choice.

The way I'd recommend evaluating it: ignore the language comparison
(Go vs. Python vs. Node) and focus on the **operational** characteristics:

| Concern | Go | Python | Node |
|---|---|---|---|
| Single binary | ✅ | ❌ | ❌ |
| Memory under load | Excellent | OK | Poor |
| Cold start | 80ms | 200ms | 200ms |
| Library ecosystem | Smaller | Largest | Largest |
| Hiring | Hard (smaller pool) | Easy | Easy |
| Standard library | Excellent | Excellent | Good |
| Async | Goroutines | asyncio | Promises |
| Deploy | `scp binary` | `pip install` | `npm ci` |

If "single binary" and "memory under load" matter most, Go wins.
If "library ecosystem" and "hiring" matter most, Python or Node
wins.

## What I'd recommend

Try Go on a small internal tool first. Build a CLI or a small
service. See if you like the development loop. If you do, you'll
know it's right for your platform.

Don't rewrite a working Python platform in Go just because you read
a blog post. The language doesn't matter nearly as much as the
operational characteristics — and those depend on what you're
building, not which language you pick.

Want to see what Go looks like in production? [Read about our
nuteo-web stack](https://github.com/saweek9/nuteo-web) — full source,
deployment configs, and architecture notes.
