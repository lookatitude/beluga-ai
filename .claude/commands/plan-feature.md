---
name: plan-feature
description: Plan a new feature or package using the full Architect → Researcher → Architect workflow.
---

Plan the specified feature or package for Beluga AI v2.

## Workflow

1. **Architect** analyzes the request:
   - Read `docs/concepts.md`, `docs/packages.md`, `docs/architecture.md`.
   - Identify affected packages, interfaces, dependencies.
   - Produce a research brief with topics for the Researcher.

2. **Researcher** investigates each topic:
   - Search codebase, external docs, competitor frameworks.
   - Return structured findings per topic.

3. **Architect** produces the final plan:
   - Design decisions with rationale.
   - Interface definitions (Go code).
   - Implementation tasks with acceptance criteria.
   - Dependency order.

## Output

A structured implementation plan ready for the Developer to execute, with acceptance criteria the QA Engineer can verify.
