---
title: Observability Overhaul for Fintech
slug: fintech-observability
summary: Replaced fragmented logging with a unified OpenTelemetry stack and SLO-driven alerting.
client: Confidential (Series B fintech)
industry: Fintech
year: 2026
stack: [Grafana, Loki, Tempo, Prometheus, OpenTelemetry, Kubernetes]
tags: [observability, slo, opentelemetry]
featured: true
published_at: 2026-03-10
---

## Challenge

Logs scattered across 4 vendors, no traces, on-call engineers relying on
"vibes" to triage incidents. Pager volume was unsustainable.

## Approach

- 2-week audit of existing telemetry, pain points, and on-call feedback
- Rolled out OpenTelemetry SDKs across all services (Go, Python, TypeScript)
- Stood up Grafana stack (Loki + Tempo + Mimir) on existing EKS
- Co-defined SLOs with product team — error budget policy enforced in alerts

## Outcome

- Mean time to detect dropped from 18 minutes to 90 seconds
- Page volume down 70% (SLO-driven alerts only fire on user-impacting issues)
- Engineering satisfaction with on-call: 2/10 → 8/10 (internal survey)
- New service onboarding to observability: under 1 hour
