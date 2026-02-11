---
title: Production
description: Patterns and best practices for deploying, scaling, and operating Beluga AI v2 applications in production environments.
sidebar:
  order: 0
---

Moving an AI application from prototype to production introduces challenges that do not exist in local development: unreliable upstream providers, unpredictable model outputs, multi-tenant isolation, observability across distributed traces, and the need for safety guardrails that prevent harmful content from reaching end users. These guides address each of these concerns with concrete patterns and working code.

Every pattern described here follows Beluga's core design principles: composable middleware, the registry pattern for extensibility, and `iter.Seq2` streaming throughout the stack. The guides build on each other but can be read independently depending on your immediate needs.

| Guide | Description |
|-------|-------------|
| [Orchestration & Workflows](./orchestration/) | Deterministic and LLM-driven orchestration patterns including sequential pipelines, parallel fan-out, conditional handoffs, supervisor delegation, and durable workflow execution with checkpointing |
| [Multi-Agent Systems](./multi-agent-systems/) | Coordinate multiple specialized agents using handoff tools, supervisor decomposition, event-driven communication, and shared memory for collaborative context |
| [Safety & Guards](./safety-and-guards/) | Three-stage guard pipeline for input validation, output filtering, and tool-call protection, plus PII redaction and human-in-the-loop approval workflows for high-risk operations |
| [Observability](./observability/) | Full-stack observability using OpenTelemetry GenAI semantic conventions for distributed tracing, token usage metrics, structured logging with trace correlation, and health check endpoints |
| [Deployment](./deployment/) | Framework-agnostic HTTP server adapters, resilience patterns (circuit breaker, retry, hedge, rate limit), configuration hot-reload, and container-ready deployment with Kubernetes manifests |
