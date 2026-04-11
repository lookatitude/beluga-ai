---
name: plan
description: Plan a feature or change. Architect → Researcher loop → Architect produces final implementation plan with acceptance criteria.
---

Plan the feature: $ARGUMENTS

## Workflow

### Step 1 — Scope (coordinator)
Use `@agent-coordinator` to parse the request, identify affected packages, list unknowns, and produce an initial task breakdown.

### Step 2 — Architecture (architect)
Use `@agent-architect` to review against invariants in `.wiki/architecture/invariants.md`, draft interface changes, and produce a research brief for the Researcher.

### Step 3 — Research loop (max 3 iterations)
For each topic:
1. `@agent-researcher` investigates.
2. `@agent-architect` reviews findings.
3. If unknowns remain → new questions → loop.

Stop when the architect confirms the design is fully specified.

### Step 4 — Final architecture
`@agent-architect` produces:
- Interface definitions (Go code)
- Implementation tasks with acceptance criteria in dependency order
- An ADR appended to `.wiki/architecture/decisions.md`

### Step 5 — Task list
`@agent-coordinator` creates prioritized tasks with measurable acceptance criteria.

### Step 6 — Log
Append an entry to `.wiki/log.md` summarizing the plan.

## Output

Structured implementation plan ready for `/develop` to execute.
