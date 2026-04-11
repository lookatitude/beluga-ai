# Project Log

Append-only chronological record of workflow runs. Most recent entries at the bottom.

## Format

`## [YYYY-MM-DD] <command> | <target>`
Short summary: what was done, key outcomes, files changed.

---

## [2026-04-11] system-migration | .claude/ + .wiki/
Migrated from v1 agent setup to unified self-evolving multi-agent system.
Created 10 agents, 15 commands, 5 rules files, two-tier knowledge system,
per-agent + global learning pipeline, and 4 enforcement hooks.
See `docs/superpowers/specs/2026-04-11-beluga-agent-system-design.md`.

## 2026-04-11 — wiki-learn Full Scan

**Scope:** All patterns across codebase

**Patterns Captured:** 8
- provider-registration.md
- middleware.md
- hooks.md
- streaming.md
- testing.md
- otel-instrumentation.md
- error-handling.md
- security-guards.md

**Packages Documented:** 7
- core (Stream, Error, ErrorCode)
- tool (Tool, Registry, Middleware, Hooks)
- llm (Provider, Client, Factory)
- guard (Guard, GuardResult, Decision)
- o11y (Span, Attrs, StatusCode)
- memory (MessageStore)
- protocol (Request/Response)

**Invariants:** 10 (all with file:line references)

**Output Files:**
- `.wiki/patterns/*.md` — 8 pattern files
- `.wiki/architecture/package-map.md` — 7 packages
- `.wiki/architecture/invariants.md` — 10 invariants
- `.wiki/index.md` — Main index
- `raw/research/wiki-scan-2026-04-11.md` — Full scan artifact

**Status:** Complete. All patterns validated against real code with canonical examples, variations, anti-patterns, and invariants.


## 2026-04-11 — 3-Step Retrieval Protocol Validation

**All 3 steps executed and verified:**

1. **Index Lookup** — `.wiki/index.md` contains all 8 patterns with direct links to pattern files ✓
2. **Query Execution** — `wiki-query.sh` created and tested; successfully retrieves any pattern ✓
3. **Source Validation** — All 19 file:line references cross-checked against actual codebase; 100% pass rate ✓

**Validation Tool:** `wiki-query.sh` (executable retrieval script)
**Validation Report:** `.wiki/VALIDATION_REPORT.md`

**Result:** Wiki system fully operational and source-validated.


## [2026-04-11] arch-validate | all
/arch-validate all: 7/10 invariants PASS cleanly. FAIL: 2 interfaces with >4 methods (agent.Agent, voice/s2s.Session). SOFT: 190 errors.New/fmt.Errorf sites needing per-package audit. go vet clean; core/schema zero external deps; 22/22 LLM providers auto-register; 155 compile-time interface checks. See corrections C-001, C-002, C-003.

## [2026-04-11] develop | agent + voice/s2s interface splits (C-001, C-002)
Split agent.Agent into AgentMetadata (4) + AgentExecutor (2), both embedded.
Split voice/s2s.Session into SessionSender (3) + SessionReceiver (1) + SessionControl (2), all embedded.
Zero implementation changes — purely additive composition. go build/vet/test all clean (207 packages, 0 failures).

## [2026-04-11] arch-validate sweep | framework-wide (invariants #1, #6, #8)
Four-commit framework-wide invariant-compliance sweep. Status: all 10 invariants green.

- `e97d0771` refactor(memory): fix arch-validate violations — ~90 fmt.Errorf→core.Errorf sites,
  SharedMemory.Watch migrated to iter.Seq2 with sync.Once unsubscribe, first WithTracing()
  wire-up establishing the template for the rest of the sweep.
- `33d12ca0` refactor: arch-validate sweep across framework — ~300 core.Errorf migrations
  across ~35 packages, WithTracing() middleware added to 13 more packages (17 total), three
  packages (prompt, rag/splitter, llm/routing) gained a minimal middleware.go as a
  precondition. state module-level sentinels kept as errors.New to preserve errors.Is identity.
- `19741097` refactor: chan→iter.Seq2 cross-package cascades (3/5) — workflow.ReceiveSignal,
  state.Store.Watch, voice/s2s.SessionReceiver.Recv. Discovered C-004 (Temporal
  Context.Done() == nil) and the iter.Pull2-single-goroutine rule.
- `f9c06d30` refactor(voice): FrameProcessor + Transport chan→iter.Seq2 cascade — final two
  deferred cascades. New FrameHandler `([]Frame, error)` shape; Chain is now pure functional
  composition with no goroutines per stage. Discovered C-005 (fan-in single-writer rule).

Docs updated: 14-observability.md (new WithTracing section), 03-extensibility-patterns.md
(Ring 4 cross-reference), 11-voice-pipeline.md (iter.Seq2 signatures), 16-durable-workflows.md
(ReceiveSignal + Temporal constraint). ADRs 002 and 003 appended.
