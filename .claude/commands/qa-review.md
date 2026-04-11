---
name: qa-review
description: Run a standalone QA review against acceptance criteria and code quality standards.
---

Run QA review on: $ARGUMENTS

## Workflow

### Step 1 — Review
`@agent-reviewer-qa` performs the full checklist on the specified packages using the retrieval protocol.

### Step 2 — Fix loop (max 3 iterations)
If issues found:
1. `@agent-developer-go` fixes each issue.
2. `@agent-reviewer-qa` re-reviews.
3. Loop until pass.

### Step 3 — Learning
`@agent-coordinator` captures corrections. Appends to `.wiki/log.md`.
