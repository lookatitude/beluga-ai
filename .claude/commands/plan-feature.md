---
name: plan-feature
description: Plan a new feature or package. Architect produces design, Team lead breaks into tasks.
---

Plan the specified feature or package for Beluga AI v2.

## Steps

1. Read `docs/concepts.md`, `docs/packages.md`, `docs/architecture.md`.
2. Identify: dependencies, interfaces, extension points, providers needed.
3. Determine package layer: Foundation (core, schema, config, o11y), Capability (llm, tool, memory, rag, agent, voice), Cross-cutting (guard, resilience, cache, eval, state, prompt, auth, hitl), Protocol (protocol, server), Orchestration (orchestration, workflow).
4. Produce dependency-ordered task list with: description, acceptance criteria, constraints, suggested agent.
5. Group parallelizable tasks.

## Output

Structured plan suitable for Team lead to execute via develop/test/review loops.
