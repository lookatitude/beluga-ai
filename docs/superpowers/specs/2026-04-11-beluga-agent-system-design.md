# Beluga AI v2 — Self-Evolving Multi-Agent System Design

**Date:** 2026-04-11
**Status:** Approved — ready for implementation
**Source plan:** `docs/beluga-ai-v2-agents-plan.md`
**Visual reference:** `docs/beluga_agent_workflow_system.svg`

## Purpose

Replace the current `.claude/` agent setup with the 5-layer self-evolving system from the v2 plan, fusing it with the proven learning infrastructure already present in `.claude/teams/`. The result is a single unified system that can follow workflows, learn from every execution, retrieve codebase knowledge on demand, and evolve rules over time.

## Design principles

1. **Fuse, don't replace.** The existing `.claude/teams/` has better learning infrastructure (distributed per-agent rules, cross-pollinating hooks, resumable supervisor). The plan has better architecture boundaries (5 layers, file-scoped rules, <2500-token CLAUDE.md). We keep the best of both.
2. **Two-tier learning.** Per-agent `rules/` for tactical fast-capture; `.wiki/corrections.md` for strategic curated learnings. Promotion pipeline: per-agent → wiki → .claude/rules/ → CLAUDE.md (human-approved at the final step).
3. **Every workflow independently triggerable.** Composite commands chain standalone commands; no unique logic in composites.
4. **Deterministic enforcement via hooks.** PreToolUse / PostToolUse / SubagentStop / Stop. Hooks cost zero context tokens and cannot be circumvented.
5. **Knowledge retrieval contract.** Every agent follows a 3-step retrieval protocol before starting any task.
6. **Knowledge is extracted from code, not hand-written.** `/wiki-learn` is the primary source of wiki content; humans only curate.

## Architecture: 5 layers (from plan, faithfully adopted)

```
┌──────────────────────────────────────────────────────┐
│  Layer 5 — Evolution                                  │
│  per-agent rules → .wiki/ → .claude/rules → CLAUDE.md │
└──────────────────────────┬───────────────────────────┘
                           │
┌──────────────────────────┴───────────────────────────┐
│  Layer 4 — Workflows (15 commands, all standalone)    │
└──────────────────────────┬───────────────────────────┘
                           │
┌──────────────────────────┴───────────────────────────┐
│  Layer 3 — 10 agents (lean, <50 lines each)           │
└──────────────────────────┬───────────────────────────┘
                           │
┌──────────────────────────┴───────────────────────────┐
│  Layer 2 — Two-tier knowledge                         │
│  Tier 1: CLAUDE.md + .claude/rules/* (auto-loaded)    │
│  Tier 2: .wiki/ + raw/ (on-demand, agent-retrieved)   │
└──────────────────────────┬───────────────────────────┘
                           │
┌──────────────────────────┴───────────────────────────┐
│  Layer 1 — Deterministic enforcement (hooks)          │
└──────────────────────────────────────────────────────┘
```

## File layout

```
beluga-ai/
├── CLAUDE.md                # <2500 tokens, lean
├── AGENTS.md                # symlink → CLAUDE.md (for Codex/Cursor)
├── .claude/
│   ├── settings.json        # hooks (PreToolUse deny + security, PostToolUse gofmt, SubagentStop, Stop)
│   ├── settings.local.json  # preserved as-is
│   ├── agents/
│   │   ├── coordinator.md
│   │   ├── architect.md
│   │   ├── researcher.md
│   │   ├── developer-go.md
│   │   ├── developer-web.md
│   │   ├── reviewer-qa.md
│   │   ├── reviewer-security.md
│   │   ├── docs-writer.md
│   │   ├── marketeer.md
│   │   ├── notion-syncer.md
│   │   └── <agent>/rules/   # per-agent accumulated learnings (moved from teams/)
│   ├── commands/
│   │   ├── plan.md
│   │   ├── develop.md
│   │   ├── security-review.md
│   │   ├── qa-review.md
│   │   ├── doc-check.md
│   │   ├── document.md
│   │   ├── promote.md
│   │   ├── blog.md
│   │   ├── dependency-audit.md
│   │   ├── new-feature.md
│   │   ├── learn.md
│   │   ├── arch-validate.md
│   │   ├── arch-update.md
│   │   ├── notion-sync.md
│   │   ├── wiki-learn.md
│   │   └── status.md
│   ├── rules/
│   │   ├── go-packages.md   # file-scoped: *.go in framework packages
│   │   ├── security.md      # file-scoped: security-sensitive paths
│   │   ├── website.md       # file-scoped: website/ paths
│   │   ├── documentation.md # file-scoped: docs/, *.md
│   │   └── workflow.md      # global workflow routing guide
│   ├── hooks/               # moved from .claude/teams/hooks/
│   │   ├── post-task-learn.sh
│   │   ├── post-review-learn.sh
│   │   ├── post-build-learn.sh
│   │   └── wiki-query.sh    # NEW
│   ├── state/               # moved from .claude/teams/state/
│   │   ├── progress-v2-migration.json  # archived completed migration
│   │   ├── learnings-index.md
│   │   └── notion-pages.json
│   └── skills/              # preserved as-is
├── .wiki/                   # NEW — Karpathy two-tier knowledge
│   ├── index.md             # catalog + retrieval routing table
│   ├── log.md               # append-only chronological
│   ├── corrections.md       # global C-NNN correction log
│   ├── patterns/            # 8 canonical pattern files
│   │   ├── provider-registration.md
│   │   ├── middleware.md
│   │   ├── hooks.md
│   │   ├── streaming.md
│   │   ├── testing.md
│   │   ├── otel-instrumentation.md
│   │   ├── error-handling.md
│   │   └── security.md
│   ├── architecture/
│   │   ├── decisions.md     # ADR log
│   │   ├── package-map.md   # generated by /wiki-learn
│   │   └── invariants.md    # 10 invariants with WHY
│   ├── competitors/
│   │   ├── adk-go.md
│   │   ├── eino.md
│   │   └── langchaingo.md
│   └── releases/drafts/
└── raw/                     # NEW — immutable source docs
    ├── research/
    ├── reviews/
    ├── blog/
    └── marketing/
```

## Layer 1: Hooks (settings.json)

**Preserved from current setup:**
- `PreToolUse(Write|Edit)` — Haiku security scan on file edits
- `PreToolUse(Bash)` — Haiku safety check on bash
- `PostToolUse(Write|Edit)` — gofmt on Go files

**Added from plan:**
- `PreToolUse(Bash)` pattern deny — block `rm -rf /`, `DROP TABLE`, `curl *|sh`, `wget *|sh`, `eval`
- `SubagentStop` — verify completion + extract learnings via post-task-learn hook
- `Stop` — run `go vet` on modified Go files before session ends

**Permissions extended:** `gosec`, `govulncheck`, `gomarkdoc`, `go tool cover`.

## Layer 2: Knowledge

### Tier 1 — Always loaded (<2500 tokens)

**`CLAUDE.md`** — lean. Contains: critical rules (8 numbered), pre-code checklist (4 steps including wiki retrieval), command index, `@docs/concepts.md @docs/architecture.md @docs/packages.md` imports.

**`.claude/rules/*`** — file-scoped, auto-loaded only when editing matching paths:
- `go-packages.md` — applies to `*.go` in framework packages. Provider checklist + anti-rationalization table.
- `security.md` — applies to security-sensitive paths (auth/, guard/, tool/, protocol/, server/).
- `website.md` — applies to `website/**`.
- `documentation.md` — applies to `docs/**`, `*.md`.
- `workflow.md` — global reference: task routing table.

### Tier 2 — On-demand (.wiki/)

Agents retrieve only what the task needs. See retrieval protocol below. Content is generated by `/wiki-learn` (extraction from codebase), curated by humans via `/learn`.

**`.wiki/index.md`** — catalog of every wiki file with 1-line description, last-scan timestamp, and **retrieval routing table**:

```markdown
## Retrieval routing

| Task type                              | Read these files                                       |
|-----------------------------------------|--------------------------------------------------------|
| Implement provider in llm/providers/*   | patterns/provider-registration.md, patterns/streaming.md, patterns/otel-instrumentation.md, architecture/package-map.md#llm |
| Add streaming API                       | patterns/streaming.md, patterns/testing.md             |
| Security-sensitive edit                 | patterns/security.md, corrections.md (grep: security)  |
| Refactor interface                      | architecture/invariants.md, architecture/decisions.md  |
| New package                             | architecture/invariants.md, architecture/package-map.md, patterns/provider-registration.md |
| Bug fix in tests                        | patterns/testing.md, corrections.md (grep: test)       |
```

`/wiki-learn` maintains this routing table automatically.

**`.wiki/corrections.md`** — C-NNN format. Appended by `/learn`, by coordinator at the end of workflows, and optionally by `post-task-learn.sh` for HIGH-confidence findings.

**`.wiki/patterns/*.md`** — each file: "canonical example from `<file:line>`" + 10-line snippet + "variations" + "anti-patterns". Generated by `/wiki-learn`, updated when the canonical example moves.

**`.wiki/architecture/invariants.md`** — the 10 invariants, each with WHY and a real code reference.

**`.wiki/architecture/package-map.md`** — every top-level package, purpose, key types, dependencies. Fully generated.

### Agent retrieval protocol (3 steps — in every agent's "Before starting")

1. **Read `.wiki/index.md`** (small, always current) — find which files apply.
2. **Read targeted files** per the routing table for the task type.
3. **Grep corrections**: `.claude/hooks/wiki-query.sh <package-or-topic>` returns index entries + matching corrections + pattern files in one call.

## Layer 3: Agents (10 total)

Every agent file: YAML frontmatter + ≤50 lines of system prompt. Every agent has:
- A **role statement** (1 paragraph).
- A **"Before starting" section** with the 3-step retrieval protocol.
- A **workflow** (numbered steps).
- An **anti-rationalization table** mapping excuses to counters.
- An **output format**.

### The 10 agents

| Agent | Model | Source | Role |
|---|---|---|---|
| coordinator | opus | NEW — absorbs `teams/supervise.md` | Orchestrator. Breaks down work, dispatches, tracks state, captures learnings. |
| architect | opus | existing + `teams/arch-analyst` | Designs interfaces, writes ADRs, runs gap analysis. |
| researcher | sonnet | existing + `memory: user` | Evidence-gathering only. Never implements. |
| developer-go | opus | existing `developer.md` + `teams/implementer` worktree rules | Go implementation + tests. Red/Green TDD. |
| developer-web | sonnet | NEW — absorbs `teams/website-dev` | Astro/Starlight website. |
| reviewer-qa | opus | existing `qa-engineer.md` (renamed) | Validates against acceptance criteria. Read-only. |
| reviewer-security | opus | existing `security-reviewer.md` + `teams/reviewer` cross-pollination | 2-clean-pass security review. |
| docs-writer | sonnet | existing + `teams/doc-writer` targets | Package docs, tutorials, API reference. |
| marketeer | sonnet | NEW | Blog posts, release notes, social content. |
| notion-syncer | sonnet | preserved from `teams/` | Notion sync + tracking dashboard. |

All agents declare `memory: user` for per-agent persistent learnings.

## Layer 4: Workflows (15 commands)

Every command is standalone. `/new-feature` chains other commands without adding logic.

### Core plan-to-ship pipeline

- **`/plan $FEATURE`** — Coordinator → Architect → Researcher loop → Architect final plan → Task list. Saves ADR to `.wiki/architecture/decisions.md`.
- **`/develop $TASK`** — Developer-go Red/Green TDD → Reviewer-qa → fix loop (max 3). Learnings captured.
- **`/security-review $PATH`** — Reviewer-security: automated scan (gosec, govulncheck, grep) → manual checklist → fix loop → 2 independent clean passes required.
- **`/qa-review $PATH`** — Reviewer-qa standalone full checklist.
- **`/doc-check $PATH`** — Docs-writer verifies examples compile, docs match API; Architect reviews technical accuracy.
- **`/document $TARGET`** — Docs-writer writes/updates docs; Architect verifies.
- **`/promote $FEATURE`** — Researcher competitive context → Marketeer content → Architect verifies claims.
- **`/blog $TOPIC`** — Marketeer drafts → Architect technical review → final + social posts.
- **`/dependency-audit`** — Reviewer-security scans → Developer-go updates safe deps → Reviewer-qa verifies.

### Meta-workflows

- **`/new-feature $DESC`** — Composite: `/plan` → `/develop` (per task) → `/security-review` → `/document` → `/doc-check` → `/promote` → Coordinator wiki lint.
- **`/learn $DESCRIPTION`** — Explicit correction capture. Coordinator parses, appends C-NNN to `.wiki/corrections.md`, proposes rule updates, (if ≥3 occurrences or HIGH confidence) proposes CLAUDE.md update for human approval.

### Knowledge workflows

- **`/wiki-learn [$PATH|all]`** — Generates or refreshes `.wiki/` content from the codebase:
  1. Architect walks packages, extracts interfaces/registries/hooks/middleware/errors.
  2. Researcher examines `go.mod`, `docs/`, tests for canonical examples and external references.
  3. Docs-writer distills findings into `patterns/*.md`, `architecture/package-map.md`, `architecture/invariants.md` with real `file:line` references.
  4. Coordinator updates `.wiki/index.md` (including the retrieval routing table) + appends to `log.md`.
- **`/arch-validate [$PACKAGE|all]`** — Architect reads `.wiki/architecture/invariants.md`, scans package for violations (iter.Seq2 not channels, registry pattern, ≤4 method interfaces, context.Context first, no circular imports, no `interface{}`). Reports PASS/FAIL per invariant.
- **`/arch-update $CHANGE`** — After significant changes, Architect updates `docs/architecture.md`, `docs/packages.md`, `docs/concepts.md`, appends ADR to `.wiki/architecture/decisions.md`, refreshes `package-map.md`.
- **`/notion-sync`** — Standalone notion-syncer trigger (mirrors docs/, updates tracking dashboard).
- **`/status`** — Package health table (preserved from current).

## Layer 5: Evolution

### Learning pipeline

```
Mistake during workflow
  → extracted to per-agent .claude/agents/<agent>/rules/ (automatic, via hook)
  → coordinator periodically reviews per-agent rules
  → recurring patterns promoted to .wiki/corrections.md (C-NNN format)
  → patterns seen ≥3 times OR HIGH confidence → proposed rule in .claude/rules/<file>.md
  → mature rules → proposed CLAUDE.md update (human approves)
```

**Key insight:** per-agent rules are fast and automatic (capture every signal); `.wiki/corrections.md` is curated (only keep what matters); `.claude/rules/` is enforced (applies to every relevant edit); `CLAUDE.md` is always-loaded (only rules worth that cost).

### Hook behavior

- `post-task-learn.sh` — after any agent completes, extracts error/solution patterns from the task log, writes to that agent's `rules/`. **Extended**: also appends to `.wiki/corrections.md` if the log contains `CONFIDENCE: HIGH`.
- `post-review-learn.sh` — after review, writes findings to BOTH reviewer-security's rules AND developer-go's rules (cross-pollination). Unchanged.
- `post-build-learn.sh` — on build/test/vet failures, writes to developer-go's rules. Unchanged.
- `wiki-query.sh <topic>` — NEW helper. Greps `.wiki/index.md`, `.wiki/corrections.md`, `.wiki/patterns/*`, prints matching entries to stdout.

## Migration plan

### Phase 1: Scaffold (non-destructive)

1. `mkdir -p .wiki/{patterns,architecture,competitors,releases/drafts} raw/{research,reviews,blog,marketing} .claude/hooks .claude/state docs/superpowers/specs`
2. Seed `.wiki/index.md`, `log.md`, `corrections.md` (headers only).
3. Seed `.wiki/architecture/invariants.md` with the 10 invariants from current CLAUDE.md.
4. Seed `.wiki/patterns/*.md` with stub-and-pointer format (real content populated by `/wiki-learn` later).

### Phase 2: Migrate teams/ infrastructure

5. `git mv .claude/teams/hooks/*.sh .claude/hooks/`
6. `git mv .claude/teams/state/notion-pages.json .claude/state/`
7. `git mv .claude/teams/state/learnings-index.md .claude/state/`
8. `git mv .claude/teams/state/progress.json .claude/state/progress-v2-migration.json` (archive)
9. `git mv .claude/teams/agents/notion-syncer/agent.md .claude/agents/notion-syncer.md`
10. Preserve per-agent rules directories by moving them to the new agent layout once new agents are created.

### Phase 3: Rewrite global config

11. Rewrite `CLAUDE.md` (<2500 tokens).
12. Create `AGENTS.md` as symlink to `CLAUDE.md`.
13. Update `.claude/settings.json` — add SubagentStop, Stop, extended PreToolUse denies; preserve existing hooks.

### Phase 4: Rewrite rules

14. Create `.claude/rules/go-packages.md` — merges existing `go-framework.md` content.
15. Create `.claude/rules/website.md`, `documentation.md`.
16. Update `.claude/rules/security.md` (existing content kept, framed with anti-rationalization).
17. Keep `.claude/rules/workflow.md`.

### Phase 5: Rewrite agents

18. Create the 10 agent files, preserving checklists from existing agents.
19. Each agent definition includes the 3-step retrieval protocol.

### Phase 6: Create commands

20. Create 15 command files. Composite `/new-feature` chains others, no unique logic.

### Phase 7: Hook extensions

21. Extend `post-task-learn.sh` for optional wiki append.
22. Create `wiki-query.sh` helper.
23. Make all hooks executable.

### Phase 8: Seed the wiki from code

24. (Post-implementation, user-triggered) Run `/wiki-learn all` to populate patterns/, package-map.md, invariants.md with real file:line references.

### Phase 9: Cleanup

25. `git rm` old `.claude/agents/{architect,developer,doc-writer,qa-engineer,researcher,security-reviewer}.md` (after content absorbed).
26. `git rm` old `.claude/commands/{implement,plan-feature,review}.md` (after replacement).
27. `git rm -r .claude/teams/` once all useful content has been migrated.
28. `/status` smoke test.

## Acceptance criteria

- [ ] `CLAUDE.md` word count ≤ 1900 (≈2500 tokens at 1.3x).
- [ ] `AGENTS.md` exists and points to CLAUDE.md.
- [ ] 10 agent files in `.claude/agents/` exist; each ≤70 lines total including frontmatter.
- [ ] 15 command files in `.claude/commands/` exist.
- [ ] 5 rules files in `.claude/rules/` exist.
- [ ] `.wiki/` directory with index.md, log.md, corrections.md, patterns/, architecture/, competitors/ exists.
- [ ] `raw/` directory with research/, reviews/, blog/, marketing/ exists.
- [ ] `.claude/hooks/` contains 4 scripts (post-task-learn, post-review-learn, post-build-learn, wiki-query) and all are executable.
- [ ] `.claude/state/` contains archived migration state.
- [ ] `.claude/teams/` directory no longer exists.
- [ ] `.claude/settings.json` contains PreToolUse, PostToolUse, SubagentStop, Stop hook entries.
- [ ] Every agent definition includes the 3-step retrieval protocol verbatim.
- [ ] Every agent definition includes an anti-rationalization table.
- [ ] Commit(s) made with clear messages documenting the migration.

## Out of scope

- Running `/wiki-learn all` on day one (user triggers after implementation).
- Writing real pattern content (generated by /wiki-learn, not hand-authored).
- Notion integration validation (requires live Notion workspace).
- Modifying the Beluga AI Go framework code itself.

## Known trade-offs

- **Two learning stores** (per-agent rules + .wiki/corrections.md) means some duplication. Justified because they serve different speeds (fast capture vs. curated).
- **Notion-syncer kept** despite not being in the plan. Justified by user directive and unique Notion MCP capability.
- **Extra commands beyond plan's 11**: wiki-learn, arch-validate, arch-update, notion-sync. Justified by user directive.
- **Existing agent checklists preserved in new files**. More content than a pure lean agent, but losing the checklists would regress quality.
