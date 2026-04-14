---
title: Orchestration
description: "Five built-in orchestration patterns for multi-agent coordination."
---

Beluga ships five orchestration patterns as implementations of the `OrchestrationPattern` interface. Because teams are themselves agents, patterns compose recursively — a team can be a member of a larger team with no special wiring.

## Supervisor

A central coordinator decomposes the task, delegates to specialists, validates results, and aggregates. Use when one agent has planning capability and the others have narrow expertise.

```mermaid
graph TD
  U[User] --> S[Supervisor]
  S -->|delegate| A[Specialist A]
  S -->|delegate| B[Specialist B]
  S -->|delegate| C[Specialist C]
  A --> S
  B --> S
  C --> S
  S --> Validate[Validate results]
  Validate --> Agg[Aggregate]
  Agg --> U
```

## Scatter-Gather

Fan the same input out to N agents in parallel, collect results, then aggregate. The aggregator is itself an agent that can vote, average, or synthesise a combined answer.

```mermaid
graph LR
  Orch[Orchestrator] --> A1[Agent 1]
  Orch --> A2[Agent 2]
  Orch --> A3[Agent 3]
  A1 --> Agg[Aggregator]
  A2 --> Agg
  A3 --> Agg
  Agg --> Final[Result]
```

## Pipeline

Linear sequence where each stage's output becomes the next stage's input. Stages can be mixed (LLM agent, retrieval-only agent, tool-only agent).

```mermaid
graph LR
  In[Input] --> S1[Stage 1]
  S1 --> S2[Stage 2]
  S2 --> S3[Stage 3]
  S3 --> Out[Output]
```

## Picking a pattern

```mermaid
graph TD
  Start[What do the agents need to do together?]
  Start --> D1[Clear decomposition, one leader]
  Start --> D2[Sequential specialisation]
  Start --> D3[Parallel independent work]
  Start --> D4[Deterministic pipeline]
  Start --> D5[Exploratory, emergent]
  D1 --> Sup[Supervisor]
  D2 --> HO[Handoff]
  D3 --> SG[Scatter-Gather]
  D4 --> Pipe[Pipeline]
  D5 --> BB[Blackboard]
```

Start with Supervisor or Handoff. Use Scatter-Gather only when the agents genuinely work independently. Use Pipeline when stages are truly linear. Blackboard is powerful but hard to debug — use it last.

## Related

- [Orchestration Patterns (DOC-07)](../../../../../../architecture/07-orchestration-patterns.md)
- [Agent Anatomy (DOC-05)](../../../../../../architecture/05-agent-anatomy.md)
- [Multi-Agent Team guide](../agents/)

TODO: expand this guide with code examples for each pattern and the EventBus reference.
