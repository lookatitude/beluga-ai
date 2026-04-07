---
description: Development workflow rules defining agent roles, handoff protocol, and quality gates.
alwaysApply: true
---

# Workflow Rules

## Task Routing

| Task Type | Workflow | Agents |
|-----------|----------|--------|
| New feature / complex task | Full workflow | Architect → Researcher → Architect → Developer → Security Reviewer → QA |
| Bug fix / small change | Short workflow | Developer → Security Reviewer → QA |
| Documentation only | Direct | Doc Writer |
| Code review | Security review | Security Reviewer |

## Full Workflow

1. **Architect** analyzes the task, produces a research brief with specific topics.
2. **Researcher** investigates each topic, returns structured findings with evidence.
3. **Architect** receives findings, makes design decisions, produces an implementation plan with:
   - Interface definitions (Go code)
   - Implementation tasks in dependency order
   - Acceptance criteria per task (measurable, verifiable by QA)
4. **Developer** implements code AND tests per the plan.
   - Must pass: `go build ./...`, `go vet ./...`, `go test ./...` before submitting.
5. **Security Reviewer** performs thorough review against all security checklists.
   - If issues found → return to Developer with specific findings → Developer fixes → resubmit.
   - **Must achieve 2 consecutive clean passes with zero issues** before proceeding.
   - Any issue resets the clean pass counter to 0.
6. **QA Engineer** validates every acceptance criterion with evidence.
   - If any criterion fails → return to Developer → fix → back through Security Review.

## Quality Gates

### Before Security Review (Developer must verify)
- `go build ./...` passes
- `go vet ./...` passes
- `go test ./...` passes
- No `TODO` or `FIXME` without associated tracking

### Security Review (2 clean passes required)
- Input validation and injection prevention
- Authentication and authorization
- Cryptography and data protection
- Concurrency and resource safety
- Error handling and information disclosure
- Dependencies and supply chain
- Architecture compliance

### QA Validation (final gate)
- Every acceptance criterion has explicit PASS/FAIL with evidence
- All verification commands run and output confirmed
- Architecture compliance verified (iter.Seq2, registry, context.Context, etc.)
- Test coverage on critical paths confirmed

## Handoff Protocol

- Each agent produces structured output for the next agent.
- Never skip an agent in the chain.
- If blocked, escalate to the user — don't guess.
- Document decisions and rationale at each step.
