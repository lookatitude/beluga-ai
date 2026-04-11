---
name: develop
description: Implement a planned task using Red/Green TDD, followed by QA review and fix loop.
---

Implement task: $ARGUMENTS

## Workflow

### Step 1 — Context
`@agent-developer-go` runs the retrieval protocol: reads `.wiki/index.md`, runs `.claude/hooks/wiki-query.sh <package>`, reads relevant patterns + corrections + existing code.

### Step 2 — Red: failing tests
`@agent-developer-go` writes tests that define expected behavior. Run and confirm RED.

### Step 3 — Green: implement
`@agent-developer-go` writes minimum code to pass. Run full suite. Confirm GREEN.

### Step 4 — Refactor
Clean up while keeping tests green.

### Step 5 — QA review
`@agent-reviewer-qa` validates against the Architect's acceptance criteria.

### Step 6 — Fix loop (max 3 iterations)
If QA found issues:
1. `@agent-developer-go` fixes each issue.
2. `@agent-reviewer-qa` re-reviews.
3. Loop until approved.

### Step 7 — Learning capture
`@agent-coordinator`:
- Checks for corrections made during fix loop.
- Appends C-NNN entries to `.wiki/corrections.md`.
- Updates `.wiki/patterns/*.md` if a canonical pattern was discovered.
- Appends to `.wiki/log.md`.

## Prerequisites

An Architect's plan with acceptance criteria. Run `/plan` first if none exists.
