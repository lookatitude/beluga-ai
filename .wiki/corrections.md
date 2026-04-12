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

### C-005 | 2026-04-11 | arch-validate | voice · RESOLVED 2026-04-11
**Symptom:** Two coupled channel-based interfaces in the voice subtree violated Invariant 1: `voice.FrameProcessor.Process(ctx, in <-chan Frame, out chan<- Frame) error` (`voice/processor.go:8-13`) and `voice.Transport.Recv(ctx) (<-chan Frame, error)` (`voice/pipeline.go:33-34`). The pipeline wired `Transport.Recv()` into `FrameProcessor.Chain`, so both had to convert together.
**Root cause:** Pre-`iter.Seq2` voice pipeline modelled each stage as a goroutine connected by buffered channels, with `runChain`/`resolveChainIO` orchestration. Chain composition required N-1 intermediate channels and errgroup-style wait.
**Correction:** Converted both interfaces to `iter.Seq2[Frame, error]`:
  - `FrameProcessor.Process(ctx, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error]` — pure transformer.
  - `Transport.Recv(ctx) iter.Seq2[Frame, error]` — early/dial errors delivered as `(Frame{}, err)` then end.
  - `FrameHandler` redesigned to `func(ctx, Frame) ([]Frame, error)` (slice-return matched every existing handler body: STT/TTS/VAD each emit 0, 1, or 2 frames per input).
  - `Chain` composes closures left-to-right — no intermediate channels, no goroutines. `runChain`, `resolveChainIO`, `passthroughProcessor`'s old form all removed/simplified.
  - Transport provider impls (websocket/livekit/daily/pipecat) keep their internal `chan voice.Frame` but expose it via a Seq2 closure that `select`s over `ctx.Done()` + the channel.
  - `voice/s2s.AsFrameProcessor` kept its fan-in architecture (session output + input forwarding) but migrated to `iter.Seq2` via an input-pump goroutine that calls `iter.Pull2(in)` + `defer stop()` and a separate output-pump goroutine for `session.Recv`.
**Notable constraint (new discovery):** When converting a bidirectional FrameProcessor like `s2s.AsFrameProcessor` that fans input+output onto a single events channel, the classic "send on closed channel" race is trivially re-introducible: two writers (input pump, output pump) sharing one channel with `defer close(events)` on one of them creates a race where the other writer blocks in `select` on a send that executes after close. **Rule:** every fan-in channel must have exactly one writer. Use separate result channels (one per writer) + the select-nil-channel trick to disable drained branches — see `voice/s2s/s2s.go:AsFrameProcessor` for the canonical layout: `outResults` (owned and closed by output pump), `inputErr` + `inputDone` (owned by input pump). The main consumer `select`s over all three plus `ctx.Done()` and nils out branches as they complete. Invisible without `-race`.
**Pattern reference:** Transport Seq2 producers mirror `voice/s2s/providers/openai/openai.go` Recv (wraps internal channel with ctx select). FrameLoop mirrors `memory/shared/shared.go` Watch minus the eager-subscribe phase since pure transformers have no shared state.
**Scope:** 16 files touched — `voice/{processor,pipeline,hybrid,processor_test,pipeline_test}.go`, `voice/stt/{stt,stt_test}.go`, `voice/tts/{tts,tts_test}.go`, `voice/s2s/{s2s,s2s_test}.go`, `voice/transport/{transport,websocket,transport_test}.go`, `voice/transport/providers/{daily,livekit,pipecat}/{*,*_test}.go`.
**Verification:** `go build ./...` PASS, `go vet ./...` PASS, `go test -race ./voice/...` — all 28 voice packages PASS.
**Prevention rule:** Invariant 1 already forbids channels in public APIs. Add a `-race` requirement to any fan-in goroutine refactor; shared-owner channel closes are invisible without it.
**Confidence:** HIGH — compile-time enforced by interface change; race-detector clean.

---

### C-010 | 2026-04-12 | docs-writer | prompt · OPEN
**Symptom:** Doc code examples for `prompt` package used `mgr = prompt.ApplyMiddleware(mgr, ...)` where `mgr` was declared as `*file.FileManager` (the concrete type returned by `NewFileManager`). `ApplyMiddleware` returns `prompt.PromptManager` (an interface), causing a compile error: "cannot use … as *file.FileManager value in assignment: need type assertion".
**Root cause:** `NewFileManager` returns a concrete pointer type, not the interface. The middleware wrapping pattern returns the interface, so the variable must be typed as the interface before or at the point of wrapping. The pattern is easy to misread because in other packages (e.g., `llm`) the constructor already returns the interface, so reassignment works without an explicit type declaration.
**Correction:** Always introduce an explicit interface-typed variable before calling `ApplyMiddleware`:
```go
base, err := promptfile.NewFileManager(dir)
// ... error check ...
var mgr prompt.PromptManager = base
mgr = prompt.ApplyMiddleware(mgr, prompt.WithTracing())
```
**Prevention rule:** When a provider constructor returns a concrete type (pointer to struct), you cannot directly reassign the result of `ApplyMiddleware` to that variable. The variable must be declared as the interface type before wrapping. Verify all doc code examples compile with `go build` before committing — not just after.
**Confidence:** HIGH — caught by `go build` during post-submission verification; fix confirmed compile-clean.
### C-007 | 2026-04-12 | docs-writer | docs/feature-status
**Symptom:** When writing a feature-status page describing "Planned" features, the doc-writer used `gh pr view N --json state` to determine whether features existed in `main`. All five PRs returned `"state":"MERGED"`. The doc-writer initially assumed this confirmed the feature was not in `main` (via prior context from the task), but the evidence was contradictory and required clarification.
**Root cause:** GitHub PR state `"MERGED"` means the PR was closed via merge into *some* branch — not necessarily `main`. PRs can be merged into `develop`, `release`, or staging branches and show as MERGED while `main` HEAD has none of their artifacts.
**Correction:** Never use GitHub PR merge state alone to assert code presence in `main`. The authoritative check is filesystem presence: `ls <expected-path>` on the actual `main` worktree. If the directory/file does not exist, the feature is not in `main` regardless of PR state.
**Rule:** Feature presence in `main` = `ls <code-path>` returns a result. PR "MERGED" = PR was closed via merge, not "code is in main".
**Prevention rule:** Before marking a feature as "Stable" or "in main" in any doc, verify `ls <code-path>` in the worktree rooted at the branch being documented. Record the verification command and output in the commit message.
**Confidence:** HIGH — verified by `ls agent/codeact/ tool/computeruse/ eval/judge/ cmd/beluga/ website/` all returning "No such file or directory" on main HEAD (commit `67f854c6`) despite PRs #234, #232, #243, #218, #228 showing MERGED.

### C-009 | 2026-04-12 | docs-writer | feature-presence-invariant
**Symptom:** Five feature PRs (#234 CLI, #232 Playground, #243 CodeAct, #218 Computer Use, #228 LLM-as-Judge) all show `state=MERGED` on GitHub. Their code artifacts (`cmd/beluga/`, `website/`, `agent/codeact/`, `tool/computeruse/`, `eval/judge/`) do not exist at `main` HEAD (`git log main | head` shows no commit for these features). A doc-writer checking PR state alone would incorrectly classify these features as "shipped."
**Root cause:** This project uses a multi-branch merge workflow. Feature branches are merged into an integration or staging branch (not `main`) before being promoted. GitHub shows `MERGED` as soon as the PR is closed via merge into *any* target — there is no visual distinction between "merged into main" and "merged into staging."
**Correction:** The canonical test for feature presence in `main` is `git ls-files -- <path>` or `ls <path>` on the `main` worktree. For this repo specifically:
- `ls cmd/beluga/` → absent on main = CLI not shipped
- `ls agent/codeact/` → absent on main = CodeAct not shipped
- `ls tool/computeruse/` → absent on main = Computer Use not shipped
- `ls eval/judge/` → absent on main = LLM-as-Judge not shipped
- `ls website/` → absent on main = Playground not shipped
**Invariant:** Feature is in `main` ⟺ `git ls-files main -- <expected-path>` is non-empty. GitHub PR MERGED state is neither necessary nor sufficient for this.
**Pattern implication for docs:** Any `docs/feature-status.md` "Stable" classification requires `git ls-files main -- <path>` to return at least one file. If the path is absent, the feature belongs in "Planned" regardless of PR state. Re-verify on every docs branch rebase.
**Verification command:** `git ls-files origin/main -- agent/codeact tool/computeruse eval/judge cmd/beluga website` → empty output on 2026-04-12 at main HEAD `67f854c6`.
**Confidence:** HIGH — independently confirmed by `ls`, `git log`, and filesystem enumeration.

### C-008 | 2026-04-12 | docs-writer | worktree-awareness
**Symptom:** When running inside a git worktree at `.claude/worktrees/agent-ada144bd/`, Write tool calls used absolute paths rooted at the main repository (`/home/miguelp/Projects/lookatitude/beluga-ai/`) instead of the worktree. Files were written to the main checkout, not to the branch being prepared. `git status` in the worktree showed "nothing to commit" despite apparent file edits.
**Root cause:** The agent's working directory (`pwd`) was the worktree, but absolute paths were explicitly constructed using the known main repo root. The Write tool honors the path given without inferring which git worktree should be active.
**Correction:** When operating in a git worktree, always derive the base path from `pwd` (which returns the worktree root), not from any hardcoded repo root. Verify with `git status` in the worktree after every Write/Edit call to confirm the file is tracked.
**Prevention rule:** In any worktree session, run `git status --short` after the first Write/Edit to confirm modified files appear. If `git status` is clean after a write, the write landed in the wrong tree.
**Confidence:** HIGH — reproducible; discovered by checking `git status` which was clean despite edits.

### C-010 | 2026-04-12 | docs-writer | rag/retriever
**Symptom:** A developer calls `retriever.New("colbert", cfg)` expecting a working `ColBERTRetriever`, but receives an error: "colbert: use colbert.NewColBERTRetriever() with WithEmbedder and WithIndex options". Same for `retriever.New("raptor", cfg)`.
**Root cause:** ColBERT and RAPTOR require dependencies (`ColBERTIndex`/`MultiVectorEmbedder`, or a pre-built `*Tree`/`Embedder`) that cannot be sourced from a generic `config.ProviderConfig`. Their `init()` registrations deliberately return descriptive errors to guide callers away from the generic factory path. See `rag/retriever/colbert/retriever.go:13-16` and `rag/retriever/raptor/retriever.go:15-21`.
**Correction:** Use the typed constructors: `colbert.NewColBERTRetriever(colbert.WithEmbedder(...), colbert.WithIndex(...))` and `raptor.NewRAPTORRetriever(raptor.WithTree(...), raptor.WithRetrieverEmbedder(...))`. The `retriever.New("colbert", cfg)` / `retriever.New("raptor", cfg)` registry paths exist solely so `retriever.List()` includes these names for discovery — not for construction.
**Contrast:** `StructuredRetriever` has the same pattern (`retriever.New("structured", cfg)` errors on purpose) but uses `structured.NewStructuredRetriever(structured.WithGenerator(...), structured.WithExecutor(...))`.
**Prevention rule:** Documented in DOC-10 "Common mistakes". Any new retriever that requires non-config dependencies should follow this same "register with descriptive error, provide typed constructor" pattern.
**Confidence:** HIGH — error text read directly from source; confirmed by source code at cited lines.
