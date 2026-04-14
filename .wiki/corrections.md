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

### C-006 | 2026-04-12 | docs-audit | docs/** · OPEN
**Symptom:** User-facing documentation under `docs/` has drifted from the current codebase across multiple vectors:
1. `docs/guides/first-agent.md` — the headline 5-minute onboarding doc — uses APIs that do not exist: `llm.Config{...}` map literal (real signature takes `config.ProviderConfig` struct), `agent.NewLLMAgent(...)` (real constructor is `agent.New(id, ...Option) *BaseAgent` at `agent/base.go:23`), and `stream.Range` channel-style iteration (real API is `iter.Seq2[Event, error]` at `agent/base.go:87`). None of the quickstart code compiles.
2. `docs/architecture/03-extensibility-patterns.md` — the canonical teaching doc for the 4-ring extension contract — shows `tool.Hooks{OnStart, OnEnd, OnError}` but actual `tool/hooks.go:11-25` defines `{BeforeExecute, AfterExecute, OnError}`. Line citations (e.g., `llm/registry.go:19-27`) are also off by several lines (actual `Register` lives at `llm/registry.go:22-26`).
3. `docs/reference/providers.md` — tables undercount every category: LLM lists 7 of 22 actual providers (`.wiki/log.md:68` already recorded "22/22 LLM providers auto-register" during the 2026-04-11 arch-validate sweep — the data was available but never propagated); vector stores list 7 of 13; embeddings list 5 of 9. The file also references a `tool/builtin/<name>` directory structure that does not exist — the tool package only contains `learning/` and `sandbox/` subdirs.
4. `README.md` advertises features that are not in `main`: `cmd/beluga` CLI (actual `cmd/` contains only `docgen/`), `website/` Agent Playground UI (no such directory), and `agent/codeact` (only present in `.claude/worktrees/agent-affc291b/`, not in the main tree). README prose is shipped-tense for all three.
5. `docs/architecture/18-package-dependency-map.md` omits `prompt/`, `eval/`, `cache/`, `hitl/`, `cost/`, `state/`, `audit/`, `optimize/` from its per-package dependency tables despite each being a top-level package with a `doc.go`.

**Root cause:** Three compounding factors:
- **No generation pipeline for reference material.** `providers.md` and `18-package-dependency-map.md` are hand-curated; providers get added to `llm/providers/` in PRs that never touch the docs.
- **No compile-check for doc code examples.** The pre-commit gate covers `go build`, `go vet`, `go test`, `gofmt`, `golangci-lint`, `gosec`, `govulncheck` — none extract and compile Go fences from `docs/**/*.md`. Drift is invisible to CI.
- **README feature copy written ahead of implementation.** CodeAct/CLI/Playground were advertised as shipping while live on feature branches; no status-gate prevents unreleased features from appearing in headline bullets.

**Correction (recommended, not yet applied):**
- **P0 — rewrite `first-agent.md`** to match the README quickstart (which IS correct). Single source of truth for the working quickstart.
- **P0 — fix `03-extensibility-patterns.md` Hooks example** to use real field names; re-run line citations against current source.
- **P0 — either ship or retract** the CodeAct/CLI/Playground README claims. A "Roadmap" section with tracking links is acceptable; silent-tense fiction is not.
- **P1 — generate `reference/providers.md`** from the filesystem. A small Go program walking `llm/providers/*`, `rag/{embedding,vectorstore}/providers/*`, `voice/{stt,tts,s2s,transport}/providers/*`, `memory/stores/*`, `guard/providers/*`, `protocol/*`, `auth/providers/*`, `cache/providers/*` and emitting the tables eliminates drift structurally.
- **P1 — add a doc-compile pre-commit step.** Extract ```go fences from `docs/**/*.md`, write to temp `main.go`, run `go vet`. Blocks non-compiling examples.
- **P1 — add architecture docs (or short sections in DOC-14/15)** for `prompt/`, `eval/`, `cache/`, `hitl/`, `state/`, `cost/`, `audit/`.

**Prevention rule:**
1. **Docs-as-code gate:** doc code examples must compile as part of CI. Add `doc-check` step to `_ci-checks.yml` using a small extract-and-vet helper.
2. **Reference material must be generated,** not hand-curated. Hand-curated provider tables drift within a single release cycle — already demonstrated by `.wiki/log.md:68` knowing about all 22 LLM providers on 2026-04-11 while `docs/reference/providers.md` still lists 7 on 2026-04-12.
3. **README feature bullets require an evidence link** (`cmd/beluga/main.go`, `website/package.json`, `agent/codeact/doc.go`). PR review rejects bullets whose evidence path doesn't exist in `main`.
4. **When arch-validate or any sweep discovers a count/inventory fact** (e.g., "22/22 LLM providers auto-register"), the sweep's follow-up tasks must include a diff against `docs/reference/*` to propagate. Add a step to the `/arch-validate` command checklist.

**Confidence:** HIGH for all five findings — each backed by concrete file reads and grep verification against the current tree (`llm/registry.go`, `tool/{tool,hooks,middleware}.go`, `agent/base.go`, directory listings of `llm/providers/`, `rag/vectorstore/providers/`, `rag/embedding/providers/`, `cmd/`, `website/` absence, `agent/codeact` worktree-only presence).
**Confidence:** HIGH for all five findings — each backed by concrete file reads and grep verification against the current tree (`llm/registry.go`, `tool/{tool,hooks,middleware}.go`, `agent/base.go`, directory listings of `llm/providers/`, `rag/vectorstore/providers/`, `rag/embedding/providers/`, `cmd/`, `website/` absence, `agent/codeact` worktree-only presence).

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

### C-012 | 2026-04-12 | marketeer | .wiki/architecture/package-map
**Symptom:** `.wiki/architecture/package-map.md` covers only 7 packages (core, tool, llm, guard, o11y, memory, protocol). The arch-validate sweep (commits e97d0771, 33d12ca0, 19741097, f9c06d30) added `WithTracing()` to 17 packages and migrated chan→iter.Seq2 across voice, workflow, and state. The wiki scan predates those commits (last scan: 2026-04-11 before the sweep).
**Root cause:** `/wiki-learn` was run once before the sweep. Package-map entries are not auto-updated on architecture changes.
**Correction:** Run `/wiki-learn all` after any architecture sweep to regenerate package-map entries. Until then, use `docs/architecture/03-extensibility-patterns.md` and `docs/architecture/14-observability.md` as the authoritative count of instrumented packages — not the wiki package-map.
**Prevention rule:** After any multi-package refactor, run `/wiki-learn all` before the next `/promote` or `/blog` task to keep the wiki current.
**Confidence:** HIGH — stale count confirmed by comparing log.md scan date against commit timestamps.

### C-011 | 2026-04-12 | marketeer | docs/architecture/06-reasoning-strategies
**Symptom:** Competitor wiki stubs (`.wiki/competitors/adk-go.md`, `.wiki/competitors/eino.md`) claim "7 reasoning strategies vs ADK's 1" and "7 reasoning strategies vs Eino's 3". The canonical source `docs/architecture/06-reasoning-strategies.md` lists 8 strategies: ReAct, Reflexion, Self-Discover, MindMap, Tree-of-Thought, Graph-of-Thought, LATS, Mixture-of-Agents.
**Root cause:** Competitor stubs were written when the canonical count was 7. A new strategy (Mixture-of-Agents) was added to the planner registry without updating the stub files.
**Correction:** The authoritative strategy count is in `docs/architecture/06-reasoning-strategies.md`. As of 2026-04-12 the count is 8. All marketing copy and competitor comparisons must read this doc directly — never the stubs.
**Prevention rule:** Before any `/promote` or `/blog` task, verify the reasoning strategy count from `docs/architecture/06-reasoning-strategies.md`, not from `.wiki/competitors/*.md`. The competitor stubs are secondary references that lag the canonical doc.
**Confidence:** HIGH — count verified by direct read of `docs/architecture/06-reasoning-strategies.md` strategy table on 2026-04-12.

### C-010 | 2026-04-12 | docs-writer | rag/retriever
**Symptom:** A developer calls `retriever.New("colbert", cfg)` expecting a working `ColBERTRetriever`, but receives an error: "colbert: use colbert.NewColBERTRetriever() with WithEmbedder and WithIndex options". Same for `retriever.New("raptor", cfg)`.
**Root cause:** ColBERT and RAPTOR require dependencies (`ColBERTIndex`/`MultiVectorEmbedder`, or a pre-built `*Tree`/`Embedder`) that cannot be sourced from a generic `config.ProviderConfig`. Their `init()` registrations deliberately return descriptive errors to guide callers away from the generic factory path. See `rag/retriever/colbert/retriever.go:13-16` and `rag/retriever/raptor/retriever.go:15-21`.
**Correction:** Use the typed constructors: `colbert.NewColBERTRetriever(colbert.WithEmbedder(...), colbert.WithIndex(...))` and `raptor.NewRAPTORRetriever(raptor.WithTree(...), raptor.WithRetrieverEmbedder(...))`. The `retriever.New("colbert", cfg)` / `retriever.New("raptor", cfg)` registry paths exist solely so `retriever.List()` includes these names for discovery — not for construction.
**Contrast:** `StructuredRetriever` has the same pattern (`retriever.New("structured", cfg)` errors on purpose) but uses `structured.NewStructuredRetriever(structured.WithGenerator(...), structured.WithExecutor(...))`.
**Prevention rule:** Documented in DOC-10 "Common mistakes". Any new retriever that requires non-config dependencies should follow this same "register with descriptive error, provide typed constructor" pattern.
**Confidence:** HIGH — error text read directly from source; confirmed by source code at cited lines.

### C-013 | 2026-04-12 | docs-writer | llm · RESOLVED 2026-04-12
**Symptom:** Concept-section doc examples used four non-existent APIs: `llm.Config{}`, `llm.WithRetry(n)`, `ChatModel.SetHooks(...)`, and `ChatModel.Invoke(ctx, msgs)`. All four caused compile errors.
**Root cause:** The docs were written from architecture docs and wiki patterns without verifying against the actual `llm` package source. The architecture docs describe the *pattern* generically (registry takes a `Config`, middleware includes retry, hooks attach to the model) — but the concrete `llm` package uses `config.ProviderConfig` as the registry argument, does not export `WithRetry`, applies hooks via `WithHooks(Hooks) Middleware` (not `SetHooks`), and exposes `Generate` not `Invoke` on `ChatModel`.
**Correction:**
- `llm.Config{...}` → `config.ProviderConfig{...}` (`github.com/lookatitude/beluga-ai/config`)
- `llm.WithRetry(n)` → does not exist in `llm`; retry lives in `resilience/` or is a custom middleware
- `model.SetHooks(hooks)` → `llm.ApplyMiddleware(model, llm.WithHooks(hooks))` — hooks are applied as Ring 4 middleware, not a setter method
- `model.Invoke(ctx, msgs)` → `model.Generate(ctx, msgs)` — the non-streaming path on `llm.ChatModel`
- `schema.StreamChunk.Text` → `schema.StreamChunk.Delta` — the incremental text field is `Delta`, not `Text`
- `llm.ChatModel.Stream` returns `iter.Seq2[schema.StreamChunk, error]` directly, not `core.Stream[T]` — consumer variables should be `chunk, err` not `event, err`
**Prevention rule:** Before writing any doc example that calls a package API, read the package's primary `.go` file (not just the architecture docs). Architecture docs describe canonical patterns — concrete packages may use different field names, constructor signatures, or omit certain middleware. Run `go build` on every example before committing.
**Confidence:** HIGH — all 17 examples compiled clean after fixes; verified by `go build ./...` in a local replace-directive module.

### C-014 | 2026-04-12 | docs-writer | docs/architecture · drift
**Symptom:** `docs/architecture/01-overview.md:15`, `docs/architecture/04-data-flow.md`, `docs/architecture/README.md:72`, and `docs/.redesign/marketing-brief.md:170` reference `docs/beluga_full_layered_architecture.svg` and `docs/beluga_request_lifecycle.svg`. Neither file exists in the working tree — `find docs/ -maxdepth 2 -name "*.svg"` returns nothing. The files were deleted during the DOC-01 rewrite but the markdown references were not updated.
**Root cause:** No static check scans markdown image references against filesystem state. `doc-check` currently validates code example compilation but not asset existence.
**Correction:** Either (a) restore the SVGs from git history and retheme to brand palette, or (b) delete the `![](...)` lines and let the adjacent mermaid fences do the work. Recommendation is (b) — the mermaid fence at `01-overview.md:45` is the same 7-layer graph, and `LayerStack.astro` on the marketing homepage already covers the visual. Single-source from markdown.
**Prevention rule:** Extend `/doc-check` (and/or add a `.claude/hooks/` pre-commit check) to grep `!\[[^]]*\]\(([^)]+\.(svg|png|jpg|gif))\)` across `docs/`, resolve the path relative to the markdown file, and fail when the asset is missing. One-line ripgrep → one-line stat loop. Catches this class of drift permanently.
**Confidence:** HIGH — four independent references verified, zero SVG files found via Glob.

### C-015 | 2026-04-12 | docs-writer | docs/website/src/lib/mermaid · version drift
**Symptom:** When proposing a mermaid theme for the Beluga website, the initial draft used v10-era `themeVariables` keys (`arrowheadColor`, `loopTextColor`) and described behaviour as "mermaid 10+". The actual installed version is `mermaid@11.12.3` per `docs/website/package.json`. In v11, `arrowheadColor` was removed (arrowheads inherit `lineColor`) and `loopTextColor` was deprecated in favour of `noteTextColor` inheritance. The keys would compile (because `themeVariables` is typed as `any` in v11) but silently no-op.
**Root cause:** Same shape as C-013: pattern-matching a library API from memory against its documented shape, without checking the installed version's `*.d.ts` in `node_modules/`. "Mermaid has `arrowheadColor`" is generic, version-free knowledge — but the task was version-specific.
**Correction:** Before writing configuration or example code for any website library, read `docs/website/node_modules/<pkg>/dist/*.d.ts` (TypeScript) or `package.json` + the installed source (JS). Go equivalent: read the actual package source at `go.mod`-pinned version, not generic package documentation.
**Prevention rule:** Amend `.claude/rules/website.md` § "Before editing" to include: "Verify library API shape against `node_modules/<pkg>/dist/*.d.ts` at the version pinned in `package.json`. Generic documentation is version-free; your task is not." Matching the existing C-013 rule for Go packages, this closes the same hole on the JS side.
**Confidence:** HIGH — version verified via `grep '"mermaid"' docs/website/package.json` → `^11.12.3`; type shape verified via `config.type.d.ts`.

### C-017 | 2026-04-12 | docs-writer | docs/website · mermaid diagram divergence
**Symptom:** Three website pages (`voice/voice-ai.md`, `memory/memory-system.md`, `reference/architecture/packages.md`) each contained a simplified or structurally different mermaid diagram compared to the canonical source in `docs/architecture/`. The website diagrams had been written independently of the architecture docs and had diverged: `voice-ai.md` used a flat `graph LR` with collapsed labels; `memory-system.md` showed a flat 3-node chain without tier subgraphs; `packages.md` used a custom `graph TB` with different subgraph groupings than DOC-18's 7-layer model.
**Root cause:** Website pages were authored from the architecture docs' *prose* descriptions, not by copying the mermaid fences directly. Each author re-drew the diagram from understanding, producing a structurally valid but non-canonical variant. No automated check enforces "website diagram == architecture source diagram."
**Correction:** When embedding a diagram designated as canonical (from `docs/.redesign/diagram-inventory.md` or any future inventory), always copy the fence verbatim from the source file — never redraw. If the target page already has a mermaid block for the same concept, replace it; do not add a second block.
**Prevention rule:** Before inserting a mermaid diagram into a website page, grep the target file for an existing ` ```mermaid ` fence. If one exists, confirm it matches the canonical source. If it differs, replace rather than insert. Add this check to `/doc-check`.
**Confidence:** HIGH — three independent instances confirmed during diagram-embedding sweep on 2026-04-12. Replacement edits verified by `npx astro build` (0 warnings).

### C-016 | 2026-04-12 | coordinator | retrieval-protocol · docs-only tasks
**Symptom:** A pure-documentation task (cataloguing 78 mermaid diagrams across 18 architecture docs for website placement) skipped the 3-step retrieval protocol — went directly to grep + Read without first consulting `.wiki/index.md` or running `.claude/hooks/wiki-query.sh`. The stop-hook reviewer flagged this as a retrieval protocol miss.
**Root cause:** Retrieval protocol is perceived as code-only ("why would I check invariants for a docs task?"). But documentation tasks are the *most* prone to drift because they lack compile-time feedback, and the wiki contains corrections (C-012, C-013) that directly alter the output. In this case, querying the wiki surfaced: (a) `.wiki/architecture/package-map.md` is stale for the observability sweep, meaning the span-hierarchy diagram's target page should cite DOC-14 not the wiki; (b) five diagrams anchor directly to invariants in `.wiki/architecture/invariants.md`, which changes how their captions should be written.
**Correction:** Retrieval protocol is mandatory for *any* task that touches `docs/architecture/` or references `file:line` anchors, regardless of whether code is being written.
**Prevention rule:** Amend `.claude/rules/documentation.md` § "Sources to consult before writing" to make the 3-step protocol explicit and ordered: (1) `.wiki/index.md`, (2) `bash .claude/hooks/wiki-query.sh <topic>`, (3) targeted `.wiki/patterns/*.md` or `.wiki/architecture/*.md` files. Only then fall through to grep/Read on `docs/`.
**Confidence:** HIGH — empirically confirmed: post-hoc wiki query surfaced C-012 and C-013 which each changed specific cells in the output report.
