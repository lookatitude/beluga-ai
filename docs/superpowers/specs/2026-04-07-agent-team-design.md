# Self-Improving Agent Team for Beluga AI v2 Architecture Migration

**Date**: 2026-04-07
**Status**: Approved
**Approach**: Skill-orchestrated pipeline with Agent tool, supervisor pattern, hook-driven feedback loops

---

## 1. Overview

A team of 6 specialized Claude Code agents, orchestrated by a supervisor skill, that:

1. Analyzes the gap between the current codebase and the new v2 architecture
2. Implements missing packages in dependency-ordered batches with security review gates
3. Updates project documentation, website, and Notion
4. Continuously self-improves through hook-driven feedback loops that persist learnings per-agent

## 2. Directory Structure

```
.claude/teams/
├── supervise.md                    # Supervisor skill — orchestrates everything
├── dispatch.md                     # Dispatch skill — routes tasks to agents
├── hooks/
│   ├── post-task-learn.sh          # Fires after any agent completes a task
│   ├── post-review-learn.sh        # Fires after security/code review
│   └── post-build-learn.sh         # Fires after go build/test
├── agents/
│   ├── arch-analyst/
│   │   ├── agent.md                # Agent definition (role, tools, constraints)
│   │   ├── rules/                  # Accumulated learnings (auto-updated by hooks)
│   │   └── skills/                 # Agent-specific skills
│   ├── implementer/
│   │   ├── agent.md
│   │   ├── rules/
│   │   └── skills/
│   ├── reviewer/
│   │   ├── agent.md
│   │   ├── rules/
│   │   └── skills/
│   ├── doc-writer/
│   │   ├── agent.md
│   │   ├── rules/
│   │   └── skills/
│   ├── website-dev/
│   │   ├── agent.md
│   │   ├── rules/
│   │   └── skills/
│   └── notion-syncer/
│       ├── agent.md
│       ├── rules/
│       └── skills/
└── state/
    ├── plan.md                     # Current implementation plan (arch-analyst writes)
    ├── progress.json               # Task completion tracking
    ├── notion-pages.json           # Local doc → Notion page ID mapping
    └── learnings-index.md          # Cross-agent learning index
```

## 3. Agent Roster

| Agent | Subagent Type | Role | Dispatched For |
|-------|--------------|------|----------------|
| **arch-analyst** | `architect` | Analyzes new vs current architecture, produces gap analysis & implementation plan | Phase 1: analysis & planning |
| **implementer** | `developer` | Implements packages, writes tests | Phase 2: development (parallelizable) |
| **reviewer** | `security-reviewer` | Security review + code review, 2 clean passes | Phase 2: after each implementation |
| **doc-writer** | `doc-writer` | Updates `docs/`, writes tutorials, API reference | Phase 3: after implementation approved |
| **website-dev** | `developer` | Updates Astro/Starlight website | Phase 3: parallel with doc-writer |
| **notion-syncer** | `general-purpose` | Syncs docs to Notion + updates project tracking | Phase 3: after docs written |

## 4. Agent Definitions

### 4.1 arch-analyst

- **Reads**: New architecture docs (`docs/beluga-ai-v2-comprehensive-architecture.md`, `docs/beluga_full_runtime_architecture.svg`), current codebase, existing `docs/`
- **Produces**: Structured gap analysis + implementation plan with dependency order + acceptance criteria per package
- **Tools**: Read, Grep, Glob, Bash (read-only)
- **Constraint**: Never writes code. Analysis and planning only.
- **Subagent type**: `architect` (uses `researcher` subagents for investigation phases)

### 4.2 implementer

- **Reads**: Implementation plan, assigned package spec, acceptance criteria, its own `rules/`
- **Produces**: Go code + tests in a worktree branch
- **Tools**: Read, Write, Edit, Bash, Glob, Grep
- **Constraint**: Must pass `go build ./...`, `go vet ./...`, `go test ./...` before signaling complete. Uses project skills (`go-framework`, `go-testing`, `go-interfaces`, `streaming-patterns`, `provider-implementation`).
- **Subagent type**: `developer`

### 4.3 reviewer

- **Reads**: Git diff of implementer's worktree branch, security rules (`.claude/rules/security.md`), its own `rules/`
- **Produces**: Structured review — PASS (with 2 clean passes) or REJECT (with specific findings)
- **Tools**: Read, Grep, Glob, Bash (read-only)
- **Constraint**: Follows `.claude/rules/security.md` checklist. Must achieve 2 consecutive clean passes. Any issue resets counter to 0.
- **Subagent type**: `security-reviewer`

### 4.4 doc-writer

- **Reads**: Implemented code, existing docs, acceptance criteria
- **Produces**: Updated `docs/` files (concepts, packages, providers, architecture), new tutorials, API reference
- **Tools**: Read, Write, Edit, Bash, Glob, Grep
- **Constraint**: Uses `doc-writing` skill. Docs must include code examples that compile.
- **Subagent type**: `doc-writer`

### 4.5 website-dev

- **Reads**: Website blueprint v2 (`docs/beluga-ai-website-blueprint-v2.md`), current site code, updated docs
- **Produces**: Updated/new pages in the Astro site (feature pages, comparisons, integrations)
- **Tools**: Read, Write, Edit, Bash, Glob, Grep
- **Constraint**: Uses `website-development` skill. Pages must match blueprint structure (18 pages from sitemap).
- **Subagent type**: `developer`

### 4.6 notion-syncer

- **Reads**: Updated docs, implementation progress, architecture decisions
- **Produces**: Notion pages (mirrored docs + project tracking dashboard)
- **Tools**: Read, Glob, Grep, Notion MCP tools (`mcp__claude_ai_Notion__*`)
- **Constraint**: Creates/updates pages, never deletes existing Notion content without confirmation. Maintains mapping in `.claude/teams/state/notion-pages.json`.
- **Subagent type**: `general-purpose`

## 5. Supervisor Orchestration Flow

### 5.1 Three Phases

```
Phase 1: Analysis & Planning
─────────────────────────────
Supervisor
  └→ arch-analyst (in worktree)
      ├── Reads new architecture docs (SVG + MD)
      ├── Scans current codebase packages
      ├── Produces gap analysis: what exists, what's new, what needs changes
      └── Returns implementation plan with acceptance criteria per package

Phase 2: Implementation (parallel dispatch)
────────────────────────────────────────────
Supervisor reads plan, identifies independent packages, dispatches:
  ├→ implementer-1 (worktree: runtime/)     ──→ reviewer ──→ ✓/✗
  ├→ implementer-2 (worktree: cost/)        ──→ reviewer ──→ ✓/✗
  ├→ implementer-3 (worktree: audit/)       ──→ reviewer ──→ ✓/✗
  │   ... (batched by dependency order)
  │
  │  On reviewer rejection:
  │  └→ re-dispatch to implementer with findings
  │     └→ re-review (loop until 2 clean passes)
  │
  │  On all batch complete:
  │  └→ Supervisor merges worktrees, runs full build/test
  │     └→ Next batch (dependent packages)

Phase 3: Documentation (parallel)
──────────────────────────────────
Supervisor dispatches simultaneously:
  ├→ doc-writer     — updates docs/, API reference, guides
  ├→ website-dev    — updates Astro site (feature pages, comparisons)
  └→ notion-syncer  — syncs docs to Notion + updates project dashboard
```

### 5.2 Key Behaviors

- **Worktree isolation**: Each implementer works in a git worktree, preventing conflicts between parallel agents.
- **Dependency batching**: Supervisor reads the plan's dependency order. Foundation packages ship first, then packages that depend on them.
- **Gate enforcement**: No package moves to Phase 3 until it has 2 consecutive clean security review passes + `go build/test/vet` all pass.
- **State tracking**: Supervisor writes progress to `.claude/teams/state/progress.json` after each dispatch, so it can resume if interrupted.

## 6. Implementation Plan Structure

The arch-analyst produces a plan the supervisor follows. Expected shape:

### Batch 1 — Foundation (no dependencies on new code)

| Package | What | Acceptance Criteria |
|---------|------|-------------------|
| `runtime/worker_pool.go` | Bounded concurrency primitive | Tests for submit, context cancellation, drain |
| `runtime/session.go` | SessionService interface + in-memory impl | CRUD + TTL expiry tests |
| `core/event_pool.go` | Zero-alloc event pool via `sync.Pool` | Benchmark showing zero heap allocs on hot path |
| `cost/` | Cost tracking types + budget enforcement | Per-request and per-tenant tracking, budget exceeded error |
| `audit/` | Audit log interface + in-memory store | Structured audit entries, query by tenant/time |

### Batch 2 — Runtime core (depends on Batch 1)

| Package | What | Acceptance Criteria |
|---------|------|-------------------|
| `runtime/plugin.go` | Plugin interface + built-in plugins (retry-reflect, audit, cost, ratelimit) | Plugin chain executes before/after turn, error propagation |
| `runtime/runner.go` | Runner lifecycle manager | Run(), RunDurable(), Serve(), graceful shutdown, health endpoints |
| `runtime/team.go` | Team as Agent, 5 orchestration patterns | Recursive composition, each pattern tested |

### Batch 3 — Enhanced capabilities (depends on Batch 2)

| Package | What | Acceptance Criteria |
|---------|------|-------------------|
| `agent/tool_dag.go` | Parallel tool execution with dependency detection | DAG builds correctly, parallel speedup measurable |
| `prompt/builder.go` | Cache-optimized prompt ordering | Static-first ordering, cache break support |
| `deploy/` | Dockerfile + Docker Compose generation | Generated configs valid and runnable |
| Enhanced `orchestration/` | Pipeline + Handoff patterns (if missing) | Pattern tests with mock agents |

### Batch 4 — Kubernetes (depends on Batch 2, optional)

| Package | What | Acceptance Criteria |
|---------|------|-------------------|
| `k8s/crds/` | CRD YAML definitions (Agent, Team, ModelConfig, ToolServer, GuardPolicy) | `kubectl apply --dry-run` passes |
| `k8s/operator/` | Reconciliation controllers | Unit tests for reconcile loop |
| `k8s/webhooks/` | Admission validation | Rejects invalid specs |
| `k8s/helm/` | Helm chart | `helm template` renders correctly |

### Batch 5 — Documentation & Website (depends on all above)

| Deliverable | Agent | Acceptance Criteria |
|------------|-------|-------------------|
| Updated `docs/` (architecture, packages, concepts, providers) | doc-writer | All new packages documented, code examples compile |
| Website feature pages per blueprint v2 | website-dev | All 18 pages from sitemap, matches blueprint |
| Notion — mirrored technical docs | notion-syncer | Every doc page has a Notion counterpart |
| Notion — project tracking dashboard | notion-syncer | Progress, decisions, architecture changes tracked |

## 7. Hook-Driven Feedback Loop

### 7.1 Three Hooks

The supervisor sets `BELUGA_AGENT_NAME` and `BELUGA_TASK_ID` environment variables before dispatching. Hooks validate agent names against an allowlist of known agent names (arch-analyst, implementer, reviewer, doc-writer, website-dev, notion-syncer) and reject any value containing path separators or special characters.

| Hook | Fires When | What It Extracts | Writes To |
|------|-----------|-----------------|-----------|
| `post-task-learn.sh` | Any agent completes a dispatched task | Failures encountered, retries, unexpected patterns, codebase quirks | `agents/<validated-agent-name>/rules/` |
| `post-review-learn.sh` | Reviewer finishes a review cycle | Common issues found, rejection reasons, patterns that pass/fail | `agents/reviewer/rules/` + `agents/implementer/rules/` |
| `post-build-learn.sh` | `go build/test/vet` runs | Build failures, test failures, import cycles, vet warnings | `agents/implementer/rules/` |

### 7.2 How Agents Consume Learnings

Each agent's `agent.md` includes:
```
Before starting work, read all files in your rules/ directory.
These are accumulated learnings from prior sessions. Apply them.
```

When the supervisor dispatches an agent, it includes the agent's `agent.md` in the prompt, which pulls in all rules. The agent starts every task with its full learning history.

### 7.3 Learning File Format

```markdown
# rules/<topic>.md
---
source: post-review-learn
date: 2026-04-07
trigger: reviewer rejected runtime/runner.go
---

## Learning
Runner methods that accept `schema.Message` must validate non-nil before
passing to plugin chain. The reviewer flagged this twice.

## Rule
Always nil-check Message inputs at Runner public API boundaries.
```

### 7.4 Cross-Pollination

When the reviewer rejects code, `post-review-learn.sh` writes to BOTH:
- `agents/reviewer/rules/` — what to watch for in future reviews
- `agents/implementer/rules/` — what to avoid in future implementations

The `learnings-index.md` tracks all learnings with tags so the supervisor can surface relevant rules when dispatching to any agent.

### 7.5 Pruning

Learnings that contradict current code get flagged as stale. The supervisor triggers a pruning pass when any agent's `rules/` directory exceeds 20 files. The agent reviews its own rules and removes outdated ones.

## 8. State Management & Resumability

### 8.1 progress.json

Updated after every dispatch/completion:

```json
{
  "currentPhase": 2,
  "currentBatch": 1,
  "tasks": [
    {
      "id": "batch1-worker-pool",
      "package": "runtime/worker_pool.go",
      "agent": "implementer",
      "status": "completed",
      "worktree": "wt-runtime-worker-pool",
      "branch": "feat/runtime-worker-pool",
      "reviewPasses": 2,
      "merged": true
    },
    {
      "id": "batch1-cost",
      "package": "cost/",
      "agent": "implementer",
      "status": "in_review",
      "worktree": "wt-cost",
      "branch": "feat/cost-package",
      "reviewPasses": 1,
      "merged": false
    }
  ],
  "learningsCount": { "implementer": 4, "reviewer": 2 },
  "lastUpdated": "2026-04-07T14:30:00Z"
}
```

### 8.2 Resumability Rules

- Supervisor reads `progress.json` at start. If a batch is partially complete, it resumes only the incomplete tasks.
- If an implementer task was `in_progress` but no worktree exists (crashed), supervisor re-dispatches from scratch.
- If a task was `in_review` with 1 pass, supervisor re-dispatches to reviewer for the second pass.
- Completed + merged tasks are never re-dispatched.

### 8.3 plan.md

The arch-analyst's full output, immutable after Phase 1. The supervisor references it but never modifies it. If the plan needs revision (e.g., a dependency was wrong), the supervisor re-dispatches to arch-analyst with the specific issue.

### 8.4 notion-pages.json

Maps local doc paths to Notion page IDs:

```json
{
  "docs/architecture.md": "notion-page-abc123",
  "docs/packages.md": "notion-page-def456",
  "project-dashboard": "notion-page-ghi789"
}
```

Prevents duplicate page creation on re-runs.

## 9. Design Invariants

1. **Agents never skip the review gate.** Every implementation must get 2 consecutive clean passes.
2. **Worktrees are the isolation boundary.** No two implementers modify the same files.
3. **Hooks write, agents read.** Agents never modify their own rules directly — only hooks do.
4. **The supervisor owns state.** Only the supervisor writes to `progress.json` and `plan.md`.
5. **Notion-syncer never deletes.** It creates and updates, never removes Notion content without user confirmation.
6. **Learnings are additive until pruned.** Rules accumulate across sessions. Pruning is explicit.
7. **The plan is immutable after Phase 1.** Changes require re-dispatch to arch-analyst.
