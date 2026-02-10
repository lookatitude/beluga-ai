---
name: status
description: Check package health and structure for Beluga AI v2.
---

Check health and structure of Beluga AI v2 packages.

## Steps

1. List existing top-level packages (core, schema, config, o11y, llm, tool, memory, rag, agent, voice, orchestration, workflow, protocol, guard, resilience, cache, hitl, auth, eval, state, prompt, server, internal).
2. For each, check: file count, test count, registry (func Register), hooks (type Hooks struct), middleware (type Middleware func), compiles (`go build`).
3. Present as status table: Package | Files | Tests | Registry | Hooks | Middleware | Compiles.
