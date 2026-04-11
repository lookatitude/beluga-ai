---
name: wiki-learn
description: Extract patterns, architecture, and invariants from the codebase into .wiki/. Populates patterns/, package-map.md, and invariants.md with real file:line references.
---

Wiki-learn scope: $ARGUMENTS (default: "all"; otherwise a package path)

## Workflow

### Step 1 — Scan (architect)
`@agent-architect` walks the target packages and extracts, for each:
- Interface definitions (method count, signatures)
- Registry sites (`Register`, `New`, `List`)
- Hook structs and middleware functions
- Error taxonomy (grep for `core.Errorf` and `ErrorCode`)
- Compile-time checks (`var _ Interface = (*Impl)(nil)`)
- Invariant violations (for a "known debt" list)

Output: structured JSON or markdown scratch file in `raw/research/wiki-scan-<date>.md`.

### Step 2 — Evidence (researcher)
`@agent-researcher`:
- Inspects `go.mod` for dependency surface per package.
- Walks `docs/` for concept coverage.
- Greps tests for canonical examples (the cleanest real instance per pattern).
- Scans git log for ADR-worthy commit messages.

### Step 3 — Distill (docs-writer)
`@agent-docs-writer` turns the scan into wiki content:

For each pattern in `.wiki/patterns/`:
- "Canonical example" — real `file:line` reference + 10-line snippet
- "Variations" — distinct real usages
- "Anti-patterns" — preserved + any new ones from the scan

For `.wiki/architecture/package-map.md`:
- One section per top-level package with purpose, key types, registry, deps, canonical file, test coverage.

For `.wiki/architecture/invariants.md`:
- Keep the 10 invariants + add a real file:line reference for each.

### Step 4 — Index (coordinator)
`@agent-coordinator`:
- Updates `.wiki/index.md` — refreshes the "Last scanned" timestamp and adjusts the retrieval routing table if new patterns were discovered.
- Appends a `/wiki-learn` entry to `.wiki/log.md`.

## Modes

- `/wiki-learn` or `/wiki-learn all` — full scan across the codebase.
- `/wiki-learn <package>` — targeted refresh for a specific package. Only the affected pattern files and the package-map entry are rewritten.

## Output

Updated `.wiki/` directory with real code references. Scan artifacts in `raw/research/wiki-scan-<date>.md`.
