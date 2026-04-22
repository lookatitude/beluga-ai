# Guides

Task-oriented how-to documents. Each guide assumes you've read [Architecture Overview](../architecture/01-overview.md) and [Extensibility Patterns](../architecture/03-extensibility-patterns.md).

## Getting started

- [Build Your First Agent](./first-agent.md) — a working LLM agent in under 5 minutes.
- [Dev Loop](./dev-loop.md) — `beluga run`, `beluga dev`, and `beluga test` end-to-end, no API key required.
- [Evaluate Your First Agent](./evaluation.md) — `beluga eval` from hand-authored dataset to mock-mode smoke to CI gating.

## Extending the framework

- [Implement a Custom Provider](./custom-provider.md) — build your own LLM, embedding, or vector store provider.
- [Build a Custom Planner](./custom-planner.md) — implement a new reasoning strategy.
- [Create a Multi-Agent Team](./multi-agent-team.md) — compose agents into teams with orchestration patterns.

## Deployment

- [Deploy on Docker](./deploy-docker.md) — Docker Compose with multiple agents, Redis, and NATS.
- [Deploy on Kubernetes](./deploy-kubernetes.md) — CRDs, operator, HPA, and the reconcile loop.
- [Deploy on Temporal](./deploy-temporal.md) — durable agent workflows with crash recovery.

## Which guide do I need?

- **I just want to see it work.** → [First Agent](./first-agent.md).
- **I want a hot-rebuild loop with a deterministic backend.** → [Dev Loop](./dev-loop.md).
- **I want to measure whether my agent is actually good.** → [Evaluation](./evaluation.md).
- **I want to use my own LLM.** → [Custom Provider](./custom-provider.md).
- **My agent needs a reasoning style not in the box.** → [Custom Planner](./custom-planner.md).
- **I have two agents that should talk to each other.** → [Multi-Agent Team](./multi-agent-team.md).
- **I want to ship this to production.** → [Docker](./deploy-docker.md) → [Kubernetes](./deploy-kubernetes.md) → [Temporal](./deploy-temporal.md).
