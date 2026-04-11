---
name: arch-update
description: Update architecture docs (docs/architecture.md, docs/packages.md, docs/concepts.md) and append ADR after significant changes.
---

Update architecture docs for change: $ARGUMENTS

## Workflow

### Step 1 — Gather
`@agent-architect`:
- Reads the diff of the change (git log, git diff).
- Reads current `docs/architecture.md`, `docs/packages.md`, `docs/concepts.md`.
- Reads `.wiki/architecture/decisions.md` for prior ADRs.
- Reads `.wiki/architecture/package-map.md` for current state.

### Step 2 — Identify impact
Classify:
- Did an interface change?
- Did a new package appear?
- Did an invariant change? (requires ADR)
- Did a concept change? (requires `concepts.md` update)

### Step 3 — Update docs
`@agent-docs-writer` updates the affected files under Architect guidance. Examples and references stay compilable.

### Step 4 — Append ADR
`@agent-architect` appends an ADR to `.wiki/architecture/decisions.md` using the format in that file.

### Step 5 — Refresh package-map
Run `/wiki-learn <affected-package>` to refresh `.wiki/architecture/package-map.md`.

### Step 6 — Technical review
`@agent-reviewer-qa` verifies the docs match actual code.

### Step 7 — Log
Append to `.wiki/log.md`.
