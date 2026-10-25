# Beluga AI v2 — Documentation

Go-native agentic AI framework. `github.com/lookatitude/beluga-ai/v2`. Go 1.26+.

## Start here

New to Beluga? Read in this order:

1. [Architecture Overview](./architecture/01-overview.md) — the 7-layer model, the master map.
2. [Core Primitives](./architecture/02-core-primitives.md) — `Stream`, `Event`, `Runnable`, `Context`.
3. [Extensibility Patterns](./architecture/03-extensibility-patterns.md) — the 4 mechanisms every package uses.
4. [Build Your First Agent](./guides/first-agent.md) — 5-minute quickstart.

## Production readiness

Beluga is designed for the production agent stack. Every extensible package ships
OpenTelemetry GenAI tracing middleware ([DOC-14](architecture/14-observability.md)),
`resilience/` provides circuit breakers, retry, hedging and rate limits
([DOC-15](architecture/15-resilience.md)), and the `workflow/` package defines
durable workflow execution ([DOC-16](architecture/16-durable-workflows.md)).
Deployment targets — Docker, Kubernetes, Temporal, and embedded — are documented in
[DOC-17](architecture/17-deployment-modes.md), with the current status of each.
For what ships today versus what is planned, see [Feature Status](feature-status.md).
Before deploying to production, work through the [Production Checklist](./production-checklist.md). It maps each enterprise-grade capability — observability, resilience, safety guards, auth, durability, cost enforcement, and evaluation — to the exact package and file that implements it, with verification steps.

## Sections

### [Architecture](./architecture/README.md)
21 documents explaining the 7-layer architecture, runtime, capabilities, cross-cutting concerns, and deployment. Read this to understand *why* Beluga is built the way it is.

### [Patterns](./patterns/README.md)
8 reusable patterns (registry, middleware, hooks, streaming, options, provider template, errors, context). Read this to learn *how* to extend any Beluga package.

### [Guides](./guides/README.md)
9 task-oriented how-tos. Read this to *build something*: first agent, dev loop, evaluation, custom provider, custom planner, multi-agent teams, Kubernetes/Temporal/Docker deployments.

### [Reference](./reference/README.md)
API surface: interfaces, configuration, CLI, providers catalog, glossary.

## By role

- **Evaluator** — start with [Architecture Overview](./architecture/01-overview.md) and [Package Dependency Map](./architecture/18-package-dependency-map.md).
- **Application developer** — [First Agent](./guides/first-agent.md) → [Multi-agent Team](./guides/multi-agent-team.md) → [Deployment](./guides/deploy-docker.md).
- **Extension developer** — [Extensibility Patterns](./architecture/03-extensibility-patterns.md) → [Provider Template](./patterns/provider-template.md) → [Custom Provider](./guides/custom-provider.md).
- **Operator** — [Observability](./architecture/14-observability.md) → [Resilience](./architecture/15-resilience.md) → [Deployment Modes](./architecture/17-deployment-modes.md).

## Design principles

The framework is built around ten architectural invariants documented in [`.guild/wiki/architecture/invariants.md`](../.guild/wiki/architecture/invariants.md). Docs cite source files from the codebase, and patterns link back to canonical implementations. Line numbers drift; the file is the reference.

## Not documentation

- [`../AGENTS.md`](../AGENTS.md) — instructions for the AI agent team that maintains this codebase.
- [`../.claude/`](../.claude/) — agent, command, hook, and rule definitions.
- [`../.guild/wiki/`](../.guild/wiki/) — extracted codebase knowledge for agent retrieval.
- [`./consultations/`](./consultations/) — point-in-time specialist consultations produced by `/consult`. Historical records; their code is not maintained against the current API.
