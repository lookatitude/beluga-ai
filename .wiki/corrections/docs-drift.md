# Corrections — Docs drift

Scope: docs-writer / docs-audit / marketeer findings — docs↔code divergence, wiki drift, website/mermaid issues.

See [README.md](./README.md) for format + promotion-pipeline rules.

---

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

### C-010 | 2026-04-12 | docs-writer | rag/retriever
**Symptom:** A developer calls `retriever.New("colbert", cfg)` expecting a working `ColBERTRetriever`, but receives an error: "colbert: use colbert.NewColBERTRetriever() with WithEmbedder and WithIndex options". Same for `retriever.New("raptor", cfg)`.
**Root cause:** ColBERT and RAPTOR require dependencies (`ColBERTIndex`/`MultiVectorEmbedder`, or a pre-built `*Tree`/`Embedder`) that cannot be sourced from a generic `config.ProviderConfig`. Their `init()` registrations deliberately return descriptive errors to guide callers away from the generic factory path. See `rag/retriever/colbert/retriever.go:13-16` and `rag/retriever/raptor/retriever.go:15-21`.
**Correction:** Use the typed constructors: `colbert.NewColBERTRetriever(colbert.WithEmbedder(...), colbert.WithIndex(...))` and `raptor.NewRAPTORRetriever(raptor.WithTree(...), raptor.WithRetrieverEmbedder(...))`. The `retriever.New("colbert", cfg)` / `retriever.New("raptor", cfg)` registry paths exist solely so `retriever.List()` includes these names for discovery — not for construction.
**Contrast:** `StructuredRetriever` has the same pattern (`retriever.New("structured", cfg)` errors on purpose) but uses `structured.NewStructuredRetriever(structured.WithGenerator(...), structured.WithExecutor(...))`.
**Prevention rule:** Documented in DOC-10 "Common mistakes". Any new retriever that requires non-config dependencies should follow this same "register with descriptive error, provide typed constructor" pattern.
**Confidence:** HIGH — error text read directly from source; confirmed by source code at cited lines.

### C-011 | 2026-04-12 | marketeer | docs/architecture/06-reasoning-strategies
**Symptom:** Competitor wiki stubs (`.wiki/competitors/adk-go.md`, `.wiki/competitors/eino.md`) claim "7 reasoning strategies vs ADK's 1" and "7 reasoning strategies vs Eino's 3". The canonical source `docs/architecture/06-reasoning-strategies.md` lists 8 strategies: ReAct, Reflexion, Self-Discover, MindMap, Tree-of-Thought, Graph-of-Thought, LATS, Mixture-of-Agents.
**Root cause:** Competitor stubs were written when the canonical count was 7. A new strategy (Mixture-of-Agents) was added to the planner registry without updating the stub files.
**Correction:** The authoritative strategy count is in `docs/architecture/06-reasoning-strategies.md`. As of 2026-04-12 the count is 8. All marketing copy and competitor comparisons must read this doc directly — never the stubs.
**Prevention rule:** Before any `/promote` or `/blog` task, verify the reasoning strategy count from `docs/architecture/06-reasoning-strategies.md`, not from `.wiki/competitors/*.md`. The competitor stubs are secondary references that lag the canonical doc.
**Confidence:** HIGH — count verified by direct read of `docs/architecture/06-reasoning-strategies.md` strategy table on 2026-04-12.

### C-012 | 2026-04-12 | marketeer | .wiki/architecture/package-map
**Symptom:** `.wiki/architecture/package-map.md` covers only 7 packages (core, tool, llm, guard, o11y, memory, protocol). The arch-validate sweep (commits e97d0771, 33d12ca0, 19741097, f9c06d30) added `WithTracing()` to 17 packages and migrated chan→iter.Seq2 across voice, workflow, and state. The wiki scan predates those commits (last scan: 2026-04-11 before the sweep).
**Root cause:** `/wiki-learn` was run once before the sweep. Package-map entries are not auto-updated on architecture changes.
**Correction:** Run `/wiki-learn all` after any architecture sweep to regenerate package-map entries. Until then, use `docs/architecture/03-extensibility-patterns.md` and `docs/architecture/14-observability.md` as the authoritative count of instrumented packages — not the wiki package-map.
**Prevention rule:** After any multi-package refactor, run `/wiki-learn all` before the next `/promote` or `/blog` task to keep the wiki current.
**Confidence:** HIGH — stale count confirmed by comparing log.md scan date against commit timestamps.

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
