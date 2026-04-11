---
name: new-feature
description: Full end-to-end feature pipeline. Chains /plan → /develop → /security-review → /document → /doc-check → /promote.
---

End-to-end feature pipeline for: $ARGUMENTS

Each phase uses standalone workflows that can also be run independently. No unique logic in this composite.

## Phase 1 — Plan
Run `/plan $ARGUMENTS`.
Output: architecture doc + prioritized task list with acceptance criteria.

## Phase 2 — Develop
For each task in dependency order:
Run `/develop <task-spec>`.
Each task must pass QA before the next begins.

## Phase 3 — Security review
Run `/security-review <affected-packages>`.
Gate: 2 consecutive clean passes required.

## Phase 4 — Documentation
Run `/document <feature>` then `/doc-check <affected-packages>`.

## Phase 5 — Promotion
Run `/promote <feature>`.

## Phase 6 — Wiki maintenance
`@agent-coordinator`:
- Updates `.wiki/index.md` with any new pages.
- Checks `.wiki/corrections.md` for entries missing prevention rules.
- Verifies `.wiki/log.md` is current.
- Proposes `CLAUDE.md` updates for any mature patterns (human approves).
