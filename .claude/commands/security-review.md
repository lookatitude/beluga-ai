---
name: security-review
description: Run a security review. Requires 2 consecutive clean passes.
---

Run a security review on: $ARGUMENTS

## Workflow

### Step 1 — Automated scan
`@agent-reviewer-security` runs `gosec -quiet ./...` (CI pins v2.30.0), `govulncheck ./...` (v1.9.0), and targeted greps for injection patterns.

### Step 2 — Manual review
`@agent-reviewer-security` performs the full manual checklist from `.claude/rules/security.md`.

### Step 3 — Fix (if issues found)
For each CRITICAL or HIGH issue:
`@agent-developer-go` fixes with explanation of what was wrong and why the fix is correct.

### Step 4 — Re-review → Clean Pass 1
`@agent-reviewer-security` re-reviews. Must report "CLEAN PASS 1" or list remaining issues. If issues remain → fix → re-review (max 3 iterations).

### Step 5 — Clean Pass 2 (independent, fresh context)
`@agent-reviewer-security` does a full independent review. No reference to Pass 1 findings — start fresh. Must report "CLEAN PASS 2" or list issues.

### Gate
Both passes must be clean consecutively. If Pass 2 finds something Pass 1 missed → fix → restart from Step 4.

### Step 6 — Learning
The coordinator (main session, following `.claude/agents/coordinator.md`) captures findings to `.guild/wiki/corrections/architecture.md` (or the matching category file). Updates `.claude/rules/security.md` if new patterns emerge. Saves review report to `raw/reviews/security-<date>.md`. Appends to `.guild/wiki/log.md`.
