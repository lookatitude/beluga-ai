---
name: Add New Agent
description: Guide to creating a new agent type in Beluga AI.
---

# Add New Agent

Agents in Beluga AI are specialized implementations that often embed a `BaseAgent`.

## Steps

1.  **Create Agent Directory**
    -   `pkg/agents/providers/<agent_type>/`.

2.  **Implement Agent Code**
    -   Create `pkg/agents/providers/<agent_type>/agent.go`.
    -   Embed `pkg/agents/base.BaseAgent` (check current location of BaseAgent).
    -   Implement the `Agent` interface methods.

3.  **Define Capabilities**
    -   Does it support tools?
    -   Does it support memory?
    -   Configure these in the `New` function.

4.  **Register Agent**
    -   Register in `pkg/agents/registry.go`.

5.  **Add Tests**
    -   Unit tests for specific logic.
    -   Integration tests with mocked LLMs.
