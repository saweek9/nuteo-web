---
title: Cloud Migration for Logistics Platform
slug: logistics-migration
summary: Migrated 40+ microservices from on-prem to EKS with zero customer-facing downtime.
client: Confidential (APAC logistics unicorn)
industry: Logistics & Supply Chain
year: 2025
stack: [AWS EKS, Terraform, Go, PostgreSQL, Kafka]
tags: [aws, kubernetes, migration, zero-downtime]
featured: true
published_at: 2026-01-20
---

## Challenge

40+ Java/Go microservices running on bare-metal Kubernetes with manual
capacity planning. Scaling was slow (3 weeks for new nodes), and the
team had no consistent observability or cost visibility.

## Approach

- 6-week discovery: built a service inventory, dependency graph, and TCO model
- Established a multi-account AWS landing zone with Control Tower
- Wrote reusable Terraform modules for EKS, RDS, and networking
- Phased migration over 4 months: stateless services first, then stateful,
  then batch jobs

## Outcome

- All 40+ services on EKS within budget and on schedule
- P95 latency improved 22% from better networking and instance types
- Monthly run cost 18% lower than the on-prem equivalent (incl. AWS premium)
- New service deployment time: 3 weeks → 1 day
