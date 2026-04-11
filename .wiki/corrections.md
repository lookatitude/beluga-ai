# Corrections Log

Every correction is an opportunity to prevent future mistakes. Agents search this file for their target package before starting work.

## Format

```
### C-NNN | YYYY-MM-DD | <workflow> | <package>
**Symptom:** what went wrong
**Root cause:** why it happened
**Correction:** what's right
**Prevention rule:** where the rule was added
**Confidence:** HIGH / MEDIUM / LOW
```

## Promotion pipeline

Per-agent `rules/` → this file → `.claude/rules/<file>.md` → (human-approved) `CLAUDE.md`.
Entries reach `.claude/rules/` when seen ≥3 times or HIGH confidence.

---

### C-001 | 2026-04-11 | arch-validate | agent
**Symptom:** `agent.Agent` interface at `agent/agent.go:32` has 6 methods, exceeding the ≤4-method invariant.
**Root cause:** Interface was grown by addition over time without composition refactor.
**Correction:** Split into smaller interfaces and compose via embedding. Suggested decomposition: `Identifier` (ID, Card), `Introspection` (Persona, Tools, Children), `Executor` (Stream). Existing implementations can embed a `BaseAgent` that satisfies the composed form.
**Prevention rule:** `.wiki/architecture/invariants.md` already encodes the rule; `/arch-validate` now flags >4-method interfaces. Consider a `golangci-lint` custom linter for enforcement.
**Confidence:** HIGH — programmatically detected.

### C-002 | 2026-04-11 | arch-validate | voice/s2s
**Symptom:** `voice/s2s.Session` interface at `voice/s2s/s2s.go:81` has 6 methods.
**Root cause:** Session lifecycle + I/O combined in one interface.
**Correction:** Split into `SessionLifecycle` (Start/Stop/Close) and `SessionIO` (send/receive/configure). Consumers can require whichever they need.
**Prevention rule:** Same as C-001.
**Confidence:** HIGH — programmatically detected.

### C-003 | 2026-04-11 | arch-validate | cross-package
**Symptom:** 190 `errors.New`/`fmt.Errorf` occurrences across 50+ files, including capability-layer public returns. Invariant 6 requires `core.Error` with `ErrorCode` on public errors.
**Root cause:** Partial adoption of typed errors during the v2 migration; some paths still return untyped errors that middleware can't classify (`core.IsRetryable` returns false for all of them).
**Correction:** Per-package audit. For each public function that returns an error, either wrap it in `core.Errorf(core.ErrXxx, ...)` with an appropriate `ErrorCode`, or document why it's exempt. Internal-only wrapping of stdlib errors is fine.
**Prevention rule:** `.claude/rules/go-packages.md` already includes anti-rationalization for this. Consider a `/arch-validate` pass per package when touched.
**Confidence:** MEDIUM — some of these are legitimate.

