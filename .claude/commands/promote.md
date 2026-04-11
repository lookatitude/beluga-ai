---
name: promote
description: Create marketing materials for a feature or release.
---

Promote: $ARGUMENTS

## Workflow

### Step 1 — Competitive context
`@agent-researcher` checks how ADK / Eino / LangChainGo handle this, sourcing from `.wiki/competitors/*.md`.

### Step 2 — Create content
`@agent-marketeer` produces:
- Blog post (800-1200 words)
- Twitter/X thread (5-7 tweets)
- LinkedIn post
- Release note

### Step 3 — Technical review
`@agent-architect` verifies all claims and code examples.

### Step 4 — Output
Save to `raw/marketing/<feature>-<date>/`. Append to `.wiki/log.md`.
