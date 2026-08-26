---
title: Postmortem — How a 3am alert became a 40% cost reduction
slug: postmortem-3am-cost-reduction
summary: "What we learned when a single billing alert uncovered systemic over-provisioning — and the playbooks that prevented it from happening again."
author: sawee kumkubkij
published_at: 2026-08-15
updated_at: 2026-08-26
tags: [postmortem, finops, kubernetes, observability]
featured: false
---

## What happened

At 03:14 local time on a Tuesday, our PagerDuty integration sent a single
"billing anomaly" alert to the on-call channel. It was supposed to be a
"your AWS bill is 20% over forecast this month" notification — the kind we get
weekly and usually auto-resolve when we close the loop on a runaway dev
environment.

This one didn't auto-resolve. **Our monthly burn was 40% over forecast**, and
when we traced the spike back through the CloudWatch metrics, we found three
independent problems that had been hiding in plain sight for weeks.

This post walks through what we found, what we fixed, and the playbooks we
built so it doesn't happen again.

## What we found

The 03:14 alert was triggered by a single EC2 instance running at full CPU
for 72+ hours. The instance was a **forgotten Redis cluster** in a now-archived
account, originally part of a 2024 customer pilot that had been deemed
"keep running, just in case."

But that explained maybe 15% of the spike. The rest came from:

1. **An autoscaling group with no upper bound.** Our batch ETL was configured
   to scale on a queue-depth metric, but nobody had set a `maxSize`. A
   late-arriving data backlog caused it to spin up 47 new c5.4xlarge
   instances overnight, each burning through 200 GB of provisioned IOPS
   they didn't actually use.

2. **A NAT gateway logging rule we'd never enabled deliberately.** A
   well-meaning SRE had turned on VPC Flow Logs to CloudWatch Logs during a
   2025 security incident and never turned them off. We'd been paying for
   ~2 TB/month of flow log ingestion at full retention since then.

3. **Snapshots of a 2019 RDS instance.** Still running. Still being snapshotted
   nightly. Still being billed. The instance itself had been terminated in
   2023.

| Resource | What | Cost/month |
|---|---|---|
| Forgotten Redis cluster | m5.large, 100% CPU | $70 |
| Runaway ETL ASG | 47× c5.4xlarge | $5,200 |
| VPC flow logs (stragglers) | 2 TB logs/mo | $1,100 |
| 2019 RDS snapshots | 500 GB GP2 | $50 |
| **Total waste** | | **~$6,420/mo** |

## What we fixed

Within 36 hours of the alert, we:

1. Terminated the Redis cluster (and archived the pilot — turned out we never
   actually used it after the proof-of-concept).
2. Set a hard `maxSize` on every ASG in the org. We found two more with
   unbounded scaling during the audit.
3. Added a lifecycle policy to the flow logs CloudWatch Log Group that
   expires them after 30 days. Also wrote a tag-based policy to prevent
   this from recurring: any flow log older than 90 days is auto-deleted.
4. Deleted 14,000 snapshots that pointed at terminated instances. Set a
   monthly cron that prunes orphaned snapshots.

## What we built to prevent recurrence

- **Resource quarantine:** any resource with no `Owner` tag for 30 days gets
  stopped automatically. Slack notification 7 days before.
- **ASG limits:** every ASG must have `maxSize` set; CI fails if missing.
- **Cost guardrails:** PagerDuty + Slack on any resource whose monthly cost
  increased more than 50% week-over-week, regardless of absolute cost.
- **Tag hygiene:** every billable resource must have `Service` and `Owner`
  tags; searching for untagged resources is a quarterly ritual now.

## What we learned

The alert itself was doing its job. The system was working as designed —
it was *designed* to warn about billing anomalies. The problem was that
the entire category of "quiet, slow, accruing waste" was invisible to our
existing observability stack.

Three months after the postmortem, we've reclaimed roughly **$72k/year** in
runaway spend — most of which we'd never noticed was happening.

Boring conclusion: **tag your things, audit your things, and make sure
someone is on the hook for each thing.** We've added all three to our
operating rhythm now.
