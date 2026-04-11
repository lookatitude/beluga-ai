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

### C-001 | 2026-04-11 | arch-validate | agent · RESOLVED 2026-04-11
**Symptom:** `agent.Agent` interface at `agent/agent.go:32` had 6 directly-declared methods, exceeding the ≤4-method invariant.
**Root cause:** Interface was grown by addition over time without composition refactor.
**Correction:** Split into `AgentMetadata` (ID, Persona, Tools, Children — 4 methods) and `AgentExecutor` (Invoke, Stream — 2 methods). `Agent` now embeds both; every existing implementation continues to satisfy it without modification.
**Commit:** (this commit) — 0 implementation changes, purely additive refactor.
**Verification:** `go build ./...` PASS, `go vet ./...` PASS, full test suite PASS (207 packages, 0 failures).
**Prevention rule:** `.wiki/architecture/invariants.md` encodes the ≤4 rule; `/arch-validate` flags violations. Consider a `golangci-lint` custom linter for permanent enforcement.
**Confidence:** HIGH — programmatically detected, verified by test suite.

### C-002 | 2026-04-11 | arch-validate | voice/s2s · RESOLVED 2026-04-11
**Symptom:** `voice/s2s.Session` interface at `voice/s2s/s2s.go:81` had 6 directly-declared methods.
**Root cause:** Session lifecycle + send + receive combined in one interface.
**Correction:** Split into `SessionSender` (3 send methods), `SessionReceiver` (1 method), `SessionControl` (Interrupt, Close — 2 methods). `Session` embeds all three.
**Commit:** (this commit) — 0 implementation changes, purely additive refactor.
**Verification:** All `voice/s2s/providers/{openai,gemini,nova}` tests pass unchanged. Full suite green.
**Prevention rule:** Same as C-001.
**Confidence:** HIGH.

### C-003 | 2026-04-11 | arch-validate | cross-package
**Symptom:** 190 `errors.New`/`fmt.Errorf` occurrences across 50+ files, including capability-layer public returns. Invariant 6 requires `core.Error` with `ErrorCode` on public errors.
**Root cause:** Partial adoption of typed errors during the v2 migration; some paths still return untyped errors that middleware can't classify (`core.IsRetryable` returns false for all of them).
**Correction:** Per-package audit. For each public function that returns an error, either wrap it in `core.Errorf(core.ErrXxx, ...)` with an appropriate `ErrorCode`, or document why it's exempt. Internal-only wrapping of stdlib errors is fine.
**Prevention rule:** `.claude/rules/go-packages.md` already includes anti-rationalization for this. Consider a `/arch-validate` pass per package when touched.
**Confidence:** MEDIUM — some of these are legitimate.

