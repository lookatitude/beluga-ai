# Architecture

The 18 documents in this directory explain the Beluga AI v2 architecture, layer by layer.

## The 7-layer model

```
┌──────────────────────────────────────────────────────┐
│  Layer 7 — Application                                │
│  User code, CLI tools, API servers, CRDs              │
├──────────────────────────────────────────────────────┤
│  Layer 6 — Agent runtime                              │
│  Runner · Agent · Executor · Planner · Team           │
├──────────────────────────────────────────────────────┤
│  Layer 5 — Orchestration                              │
│  Supervisor · Handoff · Scatter-Gather · Pipeline · … │
├──────────────────────────────────────────────────────┤
│  Layer 4 — Protocol gateway                           │
│  MCP · A2A · REST/SSE · gRPC · WebSocket              │
├──────────────────────────────────────────────────────┤
│  Layer 3 — Capability layer                           │
│  LLM · Tools · Memory · RAG · Voice · Guard · …       │
├──────────────────────────────────────────────────────┤
│  Layer 2 — Cross-cutting concerns                     │
│  Resilience · Auth · Audit · Cost · State · Sandbox   │
├──────────────────────────────────────────────────────┤
│  Layer 1 — Foundation                                 │
│  core · schema · config · o11y                        │
└──────────────────────────────────────────────────────┘
```

Data flows down. Each layer imports only from layers below.

## Documents

### Foundation (read first)
- [01 — Architecture Overview](./01-overview.md)
- [02 — Core Primitives](./02-core-primitives.md)
- [03 — Extensibility Patterns](./03-extensibility-patterns.md)

### Runtime core
- [04 — Data Flow](./04-data-flow.md)
- [05 — Agent Anatomy](./05-agent-anatomy.md)
- [06 — Reasoning Strategies](./06-reasoning-strategies.md)
- [07 — Orchestration Patterns](./07-orchestration-patterns.md)
- [08 — Runner and Lifecycle](./08-runner-and-lifecycle.md)

### Capabilities
- [09 — Memory Architecture](./09-memory-architecture.md)
- [10 — RAG Pipeline](./10-rag-pipeline.md)
- [11 — Voice Pipeline](./11-voice-pipeline.md)
- [12 — Protocol Layer](./12-protocol-layer.md)
- [13 — Security Model](./13-security-model.md)

### Cross-cutting & infrastructure
- [14 — Observability](./14-observability.md)
- [15 — Resilience](./15-resilience.md)
- [16 — Durable Workflows](./16-durable-workflows.md)
- [17 — Deployment Modes](./17-deployment-modes.md)
- [18 — Package Dependency Map](./18-package-dependency-map.md)
- [19 — Prompt Management](./19-prompt-management.md)

## Visual sources

- [`../beluga_full_layered_architecture.svg`](../beluga_full_layered_architecture.svg) — the master 7-layer diagram, embedded in DOC-01.
- [`../beluga_request_lifecycle.svg`](../beluga_request_lifecycle.svg) — complete text chat lifecycle, embedded in DOC-04.
