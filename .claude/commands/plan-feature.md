---
name: plan-feature
description: Plan the implementation of a new feature or package for Beluga AI v2. Breaks down into tasks with dependencies.
---

Plan the implementation of the specified feature or package for Beluga AI v2.

## Steps

1. Read `docs/concepts.md` and `docs/packages.md` to understand existing architecture
2. Read `docs/architecture.md` for extensibility patterns
3. For the specified feature/package, identify:
   - Dependencies (what packages must exist first)
   - Files to create or modify
   - Interfaces to define or extend
   - Providers to implement (if applicable)
   - Extension points (registry, hooks, middleware)
   - Estimated complexity (S/M/L)
4. Create a dependency-ordered task list
5. Group tasks that can be parallelized
6. Identify which sub-agent should handle each task

## Package Categories

When planning a new package, determine which layer it belongs to:

- **Foundation** (core/, schema/, config/, o11y/) — Zero external deps, used by everything
- **Capability** (llm/, tool/, memory/, rag/, agent/, voice/) — Core AI capabilities with providers
- **Cross-Cutting** (guard/, resilience/, cache/, eval/, state/, prompt/, auth/, hitl/) — Infrastructure concerns
- **Protocol** (protocol/, server/) — External protocol support and HTTP adapters
- **Orchestration** (orchestration/, workflow/) — Composition and durable execution

Output a structured task breakdown suitable for assigning to sub-agents.
