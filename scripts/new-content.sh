#!/bin/bash
# new-content.sh — scaffold a new .md file with proper frontmatter.
#
# Usage:
#   scripts/new-content.sh service cloud-cost-optimization
#   scripts/new-content.sh portfolio my-cool-project
#   scripts/new-content.sh post my-blog-post

set -e
KIND="${1:?usage: $0 <service|portfolio|post> <slug>}"
SLUG="${2:?usage: $0 <kind> <slug>}"
TODAY=$(date -u +%Y-%m-%d)

case "$KIND" in
  service)
    FILE="content/services/${SLUG}.md"
    cat > "$FILE" <<EOF
---
title: "${SLUG//-/ }"
slug: "${SLUG}"
summary: ""
icon: ""
order: 99
audience: []
tags: []
featured: false
cta_label: ""
cta_href: ""
published_at: ${TODAY}
updated_at: ${TODAY}
---

## Overview

Describe the service here.

## Engagement model

## Outcomes we deliver
EOF
    ;;
  portfolio)
    FILE="content/portfolio/${SLUG}.md"
    cat > "$FILE" <<EOF
---
title: ""
slug: "${SLUG}"
summary: ""
client: ""
industry: ""
year: $(date +%Y)
stack: []
tags: []
image: ""
url: ""
featured: false
published_at: ${TODAY}
---

## Challenge

## Approach

## Outcome
EOF
    ;;
  post)
    FILE="content/posts/${SLUG}.md"
    cat > "$FILE" <<EOF
---
title: "${SLUG//-/ }"
slug: "${SLUG}"
summary: ""
author: ""
tags: []
published_at: ${TODAY}
updated_at: ${TODAY}
---

Write your post here in Markdown.
EOF
    ;;
  *)
    echo "Unknown kind: $KIND (expected service|portfolio|post)"
    exit 1
    ;;
esac

echo "✓ Created $FILE"
echo "  Edit it, then restart the app to see changes."
