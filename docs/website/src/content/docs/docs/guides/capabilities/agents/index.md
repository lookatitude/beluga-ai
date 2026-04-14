---
title: Agents
description: "Agent runtime, planners, tools, memory, and the Plan→Act→Observe loop."
---

An agent is the atomic unit of behaviour in Beluga. It combines a persona, tools, a planner, memory, hooks, and middleware behind the `Agent` interface. Because teams implement the same interface, multi-agent systems are just recursive agent composition.

## Agent composition

Each agent is assembled from optional components wired around a common executor.

```mermaid
graph TD
  Agent[Agent]
  Agent --> Persona[Persona: role, goal, backstory]
  Agent --> Planner[Planner: reasoning strategy]
  Agent --> Tools[Tools: native and MCP]
  Agent --> Memory[Memory: 3-tier + graph]
  Agent --> Executor[Executor: Plan/Act/Observe loop]
  Agent --> Hooks[Hooks: lifecycle interception]
  Agent --> Middleware[Middleware: cross-cutting]
  Agent --> Guards[Guards: I/O/Tool pipeline]
  Agent --> Card[A2A AgentCard: exposed capabilities]
```

## Reasoning strategies

Beluga ships eight planners ordered from cheapest to most expensive. Every planner implements the same `Planner` interface, so switching is a one-line change.

```mermaid
graph LR
  ReAct[ReAct · cheap] --> Reflexion[Reflexion]
  Reflexion --> SD[Self-Discover]
  SD --> MM[MindMap]
  MM --> ToT[Tree-of-Thought]
  ToT --> GoT[Graph-of-Thought]
  GoT --> LATS[LATS]
  LATS --> MoA[Mixture-of-Agents · expensive]
```

| Strategy | LLM calls/turn | Best for |
|---|---|---|
| ReAct | 1 | Simple tool use, general tasks |
| Reflexion | 2–3 | Quality-sensitive, iterative improvement |
| Self-Discover | 2 | Cost-sensitive planning, reusable plans |
| MindMap | 2–4 | Structured reasoning with contradiction detection |
| Tree-of-Thought | 5–20 | Combinatorial search, puzzles |
| Graph-of-Thought | 5–20 | Reasoning with cycles and merging |
| LATS | 20–100 | Deep reasoning, math proofs, code synthesis |
| Mixture-of-Agents | 10–50 | Diverse perspectives, final ensemble |

## Composing agents

`BaseAgent` is an embeddable struct — every agent type derives from it. The type hierarchy shows the available subtypes:

```mermaid
graph TD
  BA[BaseAgent]
  BA --> LLM[LLMAgent: planner-driven]
  BA --> Seq[SequentialAgent]
  BA --> Par[ParallelAgent]
  BA --> Loop[LoopAgent]
  BA --> Cust[CustomAgent: full control]
  BA --> Team[TeamAgent: orchestrated children]
```

`LLMAgent` drives the Plan/Act/Observe loop via a planner. `SequentialAgent`, `ParallelAgent`, and `LoopAgent` are deterministic workflow agents with no LLM reasoning. `CustomAgent` gives you full `Stream` control. `TeamAgent` delegates to an `OrchestrationPattern` to coordinate children.

## Picking a strategy

Use this decision tree to select the least expensive strategy that meets your quality bar.

```mermaid
graph TD
  Start[What kind of task?]
  Start --> Simple[Simple tool use or Q&A]
  Start --> Quality[Quality-sensitive, iterative]
  Start --> Plan[Plans repeatable]
  Start --> Structured[Structured reasoning with contradictions]
  Start --> Combinatorial[Combinatorial search]
  Start --> Deep[Deep reasoning, math/code]
  Start --> Ensemble[Need diverse perspectives]

  Simple --> R1[ReAct]
  Quality --> R2[Reflexion]
  Plan --> R3[Self-Discover]
  Structured --> R4[MindMap]
  Combinatorial --> R5[Tree-of-Thought]
  Deep --> R6[LATS]
  Ensemble --> R7[MoA]
```

Start with ReAct. Upgrade only when you have a measurable quality gap and a budget for more tokens. Never pick LATS for a use case that ReAct handles — the cost difference is 20–100×.

## Related

- [Reasoning Strategies (DOC-06)](../../../../../../architecture/06-reasoning-strategies.md)
- [Agent Anatomy (DOC-05)](../../../../../../architecture/05-agent-anatomy.md)
- [Orchestration](../orchestration/)
- [Memory System](../memory/memory-system)

TODO: expand this guide with full agent construction example, hooks reference, and handoff patterns.
