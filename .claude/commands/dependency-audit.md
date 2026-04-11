---
name: dependency-audit
description: Audit dependencies for vulnerabilities and update status.
---

Dependency audit: $ARGUMENTS (use "full" for all modules, or a specific module path)

## Workflow

### Step 1 — Scan
`@agent-reviewer-security` runs:
```
govulncheck ./...
go list -m -u all
```

### Step 2 — Evaluate
For each finding:
- Severity assessment
- Is the vulnerable code path reachable?
- Patch version available?
- Breaking changes in the patch?

### Step 3 — Update (if safe)
`@agent-developer-go` updates safe dependencies and runs the full test suite.

### Step 4 — Report
Save to `raw/reviews/dependency-audit-<date>.md`. Append to `.wiki/log.md`.
