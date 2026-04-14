---
title: Workflow Durability
description: Crash-durable agent execution via the workflow package — Temporal, Inngest, Dapr, NATS, Kafka, or in-process backends.
---

# Workflow Durability

Long-running agents in production must survive process restarts. Beluga's `workflow/` package provides durable execution by recording every step to an event log so a redeploy, pod restart, or machine failure does not lose progress.

## The model

A workflow is a deterministic function whose every observable side effect — LLM call, tool invocation, sleep, signal — is recorded to the workflow store. On replay, completed steps return their cached result and execution resumes at the first step that had not yet completed.

This is the same model used by Temporal, Cadence, Inngest, and Dapr Workflows. Beluga abstracts over them so you write the workflow once and choose the backend per deployment.

## Backends

| Backend | When to use |
|---|---|
| `temporal` | Production at scale; you already run Temporal or want its operational tooling |
| `inngest` | You want a managed event-driven runtime with no infrastructure |
| `dapr` | You run Dapr in your cluster and want to share its state store |
| `nats` | NATS JetStream is your existing message bus |
| `kafka` | Kafka is your existing event log; you need at-least-once semantics |
| `inmemory` | Tests and local development |

## Wiring a workflow store

```go
import (
    "github.com/lookatitude/beluga-ai/workflow"
    _ "github.com/lookatitude/beluga-ai/workflow/providers/temporal"
)

store, err := workflow.New("temporal", workflow.Config{
    Endpoint:  "temporal:7233",
    Namespace: "agents",
})
if err != nil {
    panic(err)
}
```

## Resuming an interrupted run

```go
// runID is the deterministic identifier for the workflow instance.
// On a fresh process, this picks up from the last durable checkpoint.
wf, err := store.Resume(ctx, runID)
if err != nil {
    panic(err)
}

for event, err := range wf.Events(ctx) {
    if err != nil {
        panic(err)
    }
    handle(event)
}
```

## What you do not need to do

You do not need to implement application-level checkpointing. You do not need to design idempotent step handlers — Beluga's executor records the result of each step before returning to the workflow function. You do not need to choose between durable and non-durable agents at design time — the same `Agent` interface works in both modes.

## Determinism rules

Workflow functions must be deterministic. That means:

- No reads of `time.Now()`, `rand`, or other non-deterministic system calls inside the workflow body. Use `workflow.Now(ctx)` and `workflow.Random(ctx)` instead.
- No direct network or file I/O. Wrap external calls as workflow activities so they record to the event log.
- No goroutines spawned by your workflow code. Use `workflow.Go(ctx, fn)` for parallel branches.

These rules are enforced at runtime by the executor — violations panic on the first replay.

## Related

- [Resilience](/docs/guides/production/resilience/) — middleware that complements durability
- [Observability](/docs/guides/production/observability/) — workflow runs emit `gen_ai.workflow.*` spans
- [Architecture · 16 — Durable Workflows](/docs/reference/architecture/overview/) — the design rationale
