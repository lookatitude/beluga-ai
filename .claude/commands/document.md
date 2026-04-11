---
name: document
description: Write or update documentation for a package or feature.
---

Document: $ARGUMENTS

## Workflow

### Step 1 — Gather
`@agent-docs-writer` runs the retrieval protocol: reads `.wiki/index.md`, runs `.claude/hooks/wiki-query.sh <package>`, reads source code, ADRs, patterns, and corrections.

### Step 2 — Write
Produce: concept overview, quick start, API reference, full example, common mistakes, related packages.

### Step 3 — Verify examples
All code examples must compile. Run `go build` on each.

### Step 4 — Technical review
`@agent-architect` verifies technical accuracy.

### Step 5 — Finalize
Fix any review findings. Append to `.wiki/log.md`.
