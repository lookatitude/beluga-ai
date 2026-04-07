---
name: implement
description: Implement a feature using the full workflow — Developer codes + tests → Security Review loop → QA validation.
---

Implement the specified feature for Beluga AI v2.

## Workflow

1. **Developer** implements code + tests per the Architect's plan:
   - Read relevant docs and existing code.
   - Implement following framework conventions.
   - Write tests alongside code.
   - Verify: `go build ./...`, `go vet ./...`, `go test ./...`

2. **Security Reviewer** performs thorough review:
   - Review against security checklists.
   - If issues found → return to Developer → fix → re-review.
   - Loop until **2 consecutive clean passes** with zero issues.

3. **QA Engineer** validates against acceptance criteria:
   - Verify each criterion with evidence.
   - Run all verification commands.
   - If any fail → return to Developer → fix → back through Security Review.

## Prerequisites

An Architect's plan with acceptance criteria. Run `/plan-feature` first if none exists.
