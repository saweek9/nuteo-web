---
title: Postmortem — How a 3am alert became a 40% cost reduction
slug: postmortem-3am-cost-reduction
summary: "What we learned when a single billing alert uncovered systemic over-provisioning — and the playbooks that prevented it from happening again."
author: sawee kumkubkij
tags: [postmortem, sre, finops, kubernetes]
featured: true
published_at: 2026-08-15
---

## TL;DR

On a Tuesday at 3am, a billing alert fired for a single client's EKS
cluster. By 9am we'd identified $14k/month in over-provisioning. By
Friday we'd shipped fixes that brought it down 40% — and the changes
prevented the same issue across our entire client base.

## What happened

The alert was a PagerDuty notification: "EKS bill 80% of monthly
forecast by day 7 of 30." On a normal month, this cluster costs $35k.
On this month it was on track for $52k.

We had two engineers investigating by 3:30am.

## Initial investigation

The first hypothesis was runaway workloads — a bad deploy, a stuck
loop, or a runaway cron. We checked:

- **Cluster autoscaler logs**: nothing unusual
- **HPA metrics**: no scaling events beyond normal traffic patterns
- **Recent deployments**: nothing in the last 48 hours
- **Node count**: 47 nodes (avg: 38, peak: 52 last month)

Node count was high but not absurd. We dug deeper.

## The real cause

**47 nodes × 5 instance types** = a sprawl that nobody had revisited
in 14 months. When we'd last tuned instance types, the recommenders
were older generation (m5, c5). The newer generation (m6i, c6i) was
20-30% cheaper per vCPU — but the cluster's node groups were pinned to
the old types via hardcoded launch templates.

Worse: **3 node groups were running at 0% utilization**. They'd been
created for a temporary workload 14 months ago, but the workload had
migrated. Nobody had deleted the node groups.

## What we did

### Tuesday (same day)

1. Deleted 3 unused node groups (-9 nodes, ~$8k/month saved)
2. Stopped scaling new nodes into the old-generation groups
3. Confirmed: no production workload disruption (HPA scaled up to
   absorb the deletions)

### Wednesday

4. Created new launch templates for m6i/c6i (same vCPU, lower cost)
5. Cordon + drain old-gen nodes in waves
6. Updated cluster autoscaler to prefer the new templates

### Thursday

7. Verified all workloads still healthy (latency, error rates, throughput)
8. Identified one workload (a batch processor) that was over-requesting
   by 3× — reduced requests to match actual usage
9. Set up a weekly FinOps review cron to catch similar drift

### Friday

10. Wrote the playbook for "billing alert → cost reduction" as a
    runnable workflow, not just a doc
11. Sent summary to all clients with similar setup

## Impact

| Metric | Before | After | Delta |
|---|---|---|---|
| Monthly EKS bill | $52k (projected) | $31k | -40% |
| Node count (avg) | 47 | 33 | -30% |
| Unused node groups | 3 | 0 | -100% |
| Time to detect next incident | hours | minutes (automated) | 10× faster |

## Lessons

1. **Billing alerts are first-class signals.** A 3am billing alert
   is exactly as important as a 3am error rate alert.

2. **Launch templates rot.** New instance generations come out every
   18 months. Anything pinned to old hardware is leaving money on the
   table.

3. **Unused node groups are a quiet leak.** The autoscaler can't
   shrink below what node groups demand. An idle group is forever
   billable.

4. **The blast radius of "we'll clean that up later" is bigger than
   you think.** 14 months of accumulated cruft cost our client ~$100k.

5. **FinOps needs a loop, not a moment.** A single alert that
   fixed 40% was great. A weekly cron that prevents the next 100%
   over-provisioning is better.

## What we'd do differently

- **Tag workloads with cost-center** so the alert could have routed
  directly to the owning team.
- **Right-size quarterly**, not when alerts fire. Drift is normal;
  prevent it before it triggers.
- **Have a cost-aware review** as part of every incident review. We
  always look at latency and errors; we should also look at cost.

## The runnable playbook

We turned this entire flow into a workflow that any engineer can run:

```bash
# Detect idle node groups (cheaper than the dashboard)
kubectl get nodegroups -o yaml | \
  yq '.nodegroups[] | select(.status.desired < 1) | .name'

# Compare instance types per workload (catches old-gen drift)
kubectl get pod -A -o json | \
  jq -r '.items[].spec.nodeName' | \
  xargs -I{} kubectl get node {} -o jsonpath='{.metadata.labels}' | \
  jq -r 'select(.node.kubernetes.io/instance-type) | .["node.kubernetes.io/instance-type"]' | \
  sort | uniq -c | sort -rn
```

Plus a weekly cron that posts a summary to Slack with anomalies
flagged. We shipped both to every client with an EKS cluster.

## Takeaway

Postmortems aren't just for outages. They're for *any* outcome that
materially affects the system — including cost. The post-mortem
format (timeline, impact, root cause, fixes, lessons) works just as
well for "we discovered we were wasting $14k/month" as it does for
"the API went down at 3am."

Write the postmortem. Even when nothing was on fire.
