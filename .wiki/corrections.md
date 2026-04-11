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

### C-004 | 2026-04-11 | arch-validate | workflow · RESOLVED 2026-04-11
**Symptom:** `workflow.WorkflowContext.ReceiveSignal` at `workflow/context.go:20` returned `<-chan any`, exposing a raw channel in a public interface. Invariant 1 requires `iter.Seq2[T, error]` in public streaming APIs — never channels.
**Root cause:** Signal delivery was modeled as a shared buffered chan that `Signal()` pushes into and `ReceiveSignal()` returns directly. Predates the `iter.Seq2` streaming convention adopted elsewhere (e.g. `memory/shared.SharedMemory.Watch` at `memory/shared/shared.go:251`).
**Correction:** Changed the interface to `ReceiveSignal(name string) iter.Seq2[any, error]`. The default implementation (`workflow/executor.go:367`) eagerly creates/looks up the shared chan under mutex (preserving the "signal-before-subscribe is buffered" guarantee), then returns a closure that selects on `ch` + `ctx.Done()`. The Temporal implementation (`workflow/providers/temporal/temporal.go:230`) eagerly spawns the `temporalworkflow.Go` bridge coroutine and uses a `sync.Once`-guarded `done` channel so the coroutine exits cleanly when the caller stops pulling — a side improvement over the prior impl, which leaked the bridge goroutine forever.
**Pattern reference:** The conversion mirrors `memory/shared/shared.go:251` `Watch`. Tests use the `iter.Pull2` + `defer stop()` + `next()` pattern (e.g. `workflow/executor_test.go` `TestExecutor_Signal`).
**Notable constraint (new discovery):** Temporal's `workflow.Context.Done()` is NOT a Go `<-chan struct{}` — it returns `nil` (see comment at `workflow/providers/temporal/temporal.go:192-199`). This breaks the canonical `Watch` pattern's `case <-ctx.Done()` termination branch when wrapping Temporal. Workaround: rely on `yield`-returns-false (driven by caller `for range` break or `iter.Pull2` + `stop()`) plus a `sync.Once`-guarded `done` channel that the bridge goroutine selects on when pushing. Any future `iter.Seq2` conversion of a Temporal-wrapping method hits the same wall and should follow this workaround.
**Verification:** `go build ./workflow/...`, `go vet ./workflow/...`, `go test -race ./workflow/...` — all 7 packages PASS.
**Prevention rule:** Invariant 1 already forbids channels in public APIs. Consider adding a `.wiki/patterns/streaming.md` sub-section documenting the Temporal termination workaround, and scan `workflow/` in `/arch-validate` when touching streaming surfaces.
**Confidence:** HIGH — compile-time enforced by interface change; test suite green.

