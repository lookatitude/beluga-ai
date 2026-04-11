---
name: arch-validate
description: Validate code against the 10 architecture invariants in .wiki/architecture/invariants.md. Reports PASS/FAIL per invariant with file:line evidence.
---

Validate architecture for: $ARGUMENTS (package path or "all")

## Workflow

### Step 1 — Load invariants
`@agent-architect` reads `.wiki/architecture/invariants.md` and the target package(s).

### Step 2 — Scan
For each invariant, scan the target for violations:

| Invariant | Check |
|---|---|
| 1. iter.Seq2, not channels | `grep -rn 'chan ' <package>` in public APIs (exclude tests) |
| 2. Registry pattern | Look for `Register`/`New`/`List` if extensible |
| 3. init() auto-register | `grep 'func init' <package>` and trace to Register |
| 4. Interfaces ≤4 methods | Parse interface declarations, count methods |
| 5. `context.Context` first param | Parse exported function signatures |
| 6. Typed errors via `core.Error` | `grep -rn 'errors.New\|fmt.Errorf' <package>` (should be minimal) |
| 7. No `interface{}` in public APIs | `grep -rn 'interface{}\b' <package>` |
| 8. No circular imports | `go vet ./...` and `go list -deps` |
| 9. Zero external deps in core/schema | Check `go.mod` and imports |
| 10. Compile-time interface checks | `grep -rn 'var _ .* = (\*.*)(nil)' <package>` |

### Step 3 — Report
For each invariant: PASS / FAIL with file:line evidence for any failures.

```
## Architecture Validation: <package>

| # | Invariant | Status | Evidence |
|---|-----------|--------|----------|
| 1 | iter.Seq2 | PASS |  |
| 2 | Registry | FAIL | foo/bar.go:42 uses a map without Register wrapper |
| ... | ... | ... | ... |

### Verdict
COMPLIANT / VIOLATIONS FOUND (N)
```

### Step 4 — Learning
If violations are found, `@agent-coordinator` appends a correction entry to `.wiki/corrections.md` and (if HIGH confidence) proposes a rule update.
