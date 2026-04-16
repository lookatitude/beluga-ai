---
name: new-feature
description: Full end-to-end framework feature pipeline. Chains /plan → /develop → /security-review → /document → /doc-check. Promotion happens independently in the website repo.
---

End-to-end framework feature pipeline for: $ARGUMENTS

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

## Phase 5 — Wiki maintenance
`@agent-coordinator`:
- Updates `.wiki/index.md` with any new pages.
- Checks `.wiki/corrections.md` for entries missing prevention rules.
- Verifies `.wiki/log.md` is current.
- Proposes `CLAUDE.md` updates for any mature patterns (human approves).

## Promotion (handed off to website)

When the feature ships and is released, promotion (blog post, social, release note) happens independently in the `beluga-website` repo via its `/promote` command. This pipeline does NOT invoke promotion — the website team picks it up when content is ready and the framework release exists. Filing an issue in `lookatitude/beluga-website` referencing the new feature is the canonical handoff if you want to flag a release for promotion.
