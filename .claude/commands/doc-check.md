---
name: doc-check
description: Verify documentation accuracy against current code. Standalone.
---

Verify documentation: $ARGUMENTS

## Workflow

### Step 1 — Inventory
`@agent-docs-writer` lists all doc files for the specified package(s). Cross-reference against current exported API using `go doc`.

### Step 2 — Check accuracy
For each doc file:
- Do code examples compile? Verify with `go build` on a throwaway file.
- Do documented interfaces match actual code?
- Any undocumented exported types/functions?
- Do "common mistakes" match `.wiki/corrections.md`?

### Step 3 — Fix
`@agent-docs-writer` updates inaccurate documentation.

### Step 4 — Technical review
`@agent-architect` verifies technical accuracy of changes.

### Step 5 — Log
Append to `.wiki/log.md`.
