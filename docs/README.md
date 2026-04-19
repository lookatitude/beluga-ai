# Beluga AI v2 — Documentation

Go-native agentic AI framework. `github.com/lookatitude/beluga-ai/v2`. Go 1.23+.

## Start here

New to Beluga? Read in this order:

1. [Architecture Overview](./architecture/01-overview.md) — the 7-layer model, the master map.
2. [Core Primitives](./architecture/02-core-primitives.md) — `Stream`, `Event`, `Runnable`, `Context`.
3. [Extensibility Patterns](./architecture/03-extensibility-patterns.md) — the 4 mechanisms every package uses.
4. [Build Your First Agent](./guides/first-agent.md) — 5-minute quickstart.

## Production readiness

Beluga is designed for the production agent stack. Every package ships with
OpenTelemetry GenAI spans ([DOC-14](architecture/14-observability.md)),
circuit breakers and rate limits are middleware on the same interface as your
LLM calls ([DOC-15](architecture/15-resilience.md)), and the `workflow/`
package provides crash-durable execution ([DOC-16](architecture/16-durable-workflows.md)).
Deployment targets — Docker, Kubernetes, Temporal, and embedded — are documented in
[DOC-17](architecture/17-deployment-modes.md).
See the [Production Checklist](production-checklist.md).
Before deploying to production, work through the [Production Checklist](./production-checklist.md). It maps each enterprise-grade capability — observability, resilience, safety guards, auth, durability, cost enforcement, and evaluation — to the exact package and file that implements it, with verification steps.

## Sections

### [Architecture](./architecture/README.md)
18 documents explaining the 7-layer architecture, runtime, capabilities, cross-cutting concerns, and deployment. Read this to understand *why* Beluga is built the way it is.

### [Patterns](./patterns/README.md)
8 reusable patterns (registry, middleware, hooks, streaming, options, provider template, errors, context). Read this to learn *how* to extend any Beluga package.

### [Guides](./guides/README.md)
7 task-oriented how-tos. Read this to *build something*: first agent, custom provider, custom planner, multi-agent teams, Kubernetes/Temporal/Docker deployments.

### [Reference](./reference/README.md)
Stable API surface: interfaces, configuration, providers catalog, glossary.

## By role

- **Evaluator** — start with [Architecture Overview](./architecture/01-overview.md) and [Package Dependency Map](./architecture/18-package-dependency-map.md).
- **Application developer** — [First Agent](./guides/first-agent.md) → [Multi-agent Team](./guides/multi-agent-team.md) → [Deployment](./guides/deploy-docker.md).
- **Extension developer** — [Extensibility Patterns](./architecture/03-extensibility-patterns.md) → [Provider Template](./patterns/provider-template.md) → [Custom Provider](./guides/custom-provider.md).
- **Operator** — [Observability](./architecture/14-observability.md) → [Resilience](./architecture/15-resilience.md) → [Deployment Modes](./architecture/17-deployment-modes.md).

## Design principles

The framework is built around ten architectural invariants documented in [`.wiki/architecture/invariants.md`](../.wiki/architecture/invariants.md). Every doc in this set cites real `file:line` references from the codebase — patterns link back to canonical implementations so nothing drifts from reality.

## Not documentation

- [`../CLAUDE.md`](../CLAUDE.md) — instructions for the AI agent team that maintains this codebase.
- [`../.claude/`](../.claude/) — agent, command, hook, and rule definitions.
- [`../.wiki/`](../.wiki/) — extracted codebase knowledge for agent retrieval.
- [`./plans/`](./plans/) — historical design docs for features unrelated to core architecture.
- [`./beluga-ai-website-blueprint-v2.md`](./beluga-ai-website-blueprint-v2.md) — documentation *website* blueprint (separate concern).
