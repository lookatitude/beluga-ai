# DOC-17: Deployment Modes

**Audience:** Operators, platform engineers, and anyone choosing how to run Beluga.
**Prerequisites:** [08 — Runner and Lifecycle](./08-runner-and-lifecycle.md).
**Related:** [16 — Durable Workflows](./16-durable-workflows.md), [Deploy on Docker](../guides/deploy-docker.md), [Deploy on Kubernetes](../guides/deploy-kubernetes.md), [Deploy on Temporal](../guides/deploy-temporal.md).

## Overview

The same Beluga agent can be deployed four different ways without changing its code. The decision is a matter of operational requirements — scale, durability, integration with existing platforms — not of agent design.

```mermaid
graph TD
  A[Agent code]
  A --> L[Library mode]
  A --> D[Docker mode]
  A --> K[Kubernetes mode]
  A --> T[Temporal mode]
```

## Mode comparison

| Mode | Infra needed | Durable? | Scale | Best for |
|---|---|---|---|---|
| **Library** | None (embedded) | No | Single process | CLIs, desktop apps, embedded |
| **Docker** | Docker Engine | Optional (persistent volumes) | Single host or small cluster | Small to medium production |
| **Kubernetes** | K8s cluster | Optional (PVs) + HPA | Multi-node, autoscaling | Mid-to-large production |
| **Temporal** | Temporal cluster | Yes | Depends on Temporal deployment | Long-running / HITL workflows |

## Library mode

```mermaid
graph LR
  User[User code] --> B[import beluga-ai]
  B --> Agent[Agent in-process]
```

Import Beluga as a Go library. Build and run an agent in the same process. No network, no runner, no session service (or an in-memory one).

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/v2/agent"
    "github.com/lookatitude/beluga-ai/v2/config"
    "github.com/lookatitude/beluga-ai/v2/llm"
    _ "github.com/lookatitude/beluga-ai/v2/llm/providers/openai"
)

func main() {
    model, err := llm.New("openai", config.ProviderConfig{
        Provider: "openai",
        APIKey:   os.Getenv("OPENAI_API_KEY"),
        Model:    "gpt-4o",
    })
    if err != nil {
        log.Fatal(err)
    }
    a := agent.New("assistant",
        agent.WithPersona(agent.Persona{Role: "assistant"}),
        agent.WithLLM(model),
    )
    out, err := a.Invoke(context.Background(), "Hello")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(out)
}
```

Use when: you're building a CLI, a desktop app, an embedded tool, or a unit test. Simplest mode, zero infrastructure.

## Docker mode

```mermaid
graph TD
  subgraph Host[Docker host]
    R[Beluga runner container]
    OT[OTel collector]
  end
  Client --> R
  R --> OT
  R --> LLM[External LLM API]
```

Wrap a `Runner` in a container behind a small HTTP handler, add an OTel collector, compose them. This is the typical "one-box production" setup. The Runner's default session service is in memory; for sessions shared across restarts or replicas, implement `runtime.SessionService` over a store such as Redis.

Typical `docker-compose.yml`:

```yaml
services:
  agent:
    image: myorg/beluga-agent:v1
    environment:
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      OTEL_EXPORTER_OTLP_ENDPOINT: http://otel-collector:4318
    ports: ["8080:8080"]
    depends_on: [otel-collector]
  otel-collector:
    image: otel/opentelemetry-collector:latest
```

Use when: you want production-grade hosting without a Kubernetes cluster. Works well for single-tenant SaaS and internal tools.

See [Deploy on Docker guide](../guides/deploy-docker.md).

## Kubernetes mode

```mermaid
graph TD
  CR[Agent CR] --> Rec[operator.Reconciler]
  Rec --> Dep[Deployment]
  Rec --> Svc[Service]
  Rec --> HPA[HorizontalPodAutoscaler]
  Dep --> Pod1[Runner pod 1]
  Dep --> Pod2[Runner pod N]
  Pod1 --> Card[/.well-known/agent.json]
```

Define an `Agent` custom resource (`beluga.ai/v1alpha1`). `k8s/operator`'s `Reconciler` computes the Deployment, Service and optional HPA for it as plain structs. A controller that watches the cluster and applies them is not shipped yet, and NetworkPolicy/ServiceMonitor generation is not implemented; see the [Kubernetes guide](../guides/deploy-kubernetes.md) status note.

```yaml
apiVersion: beluga.ai/v1alpha1
kind: Agent
metadata:
  name: research-assistant
spec:
  replicas: 3
  image: myorg/research-agent:v2.1.0
  persona:
    role: research assistant
  planner: react
  maxIterations: 10
  modelRef: openai-gpt4o
  resources:
    requests: { cpu: 500m, memory: 1Gi }
    limits: { cpu: "2", memory: 4Gi }
  scaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
```

Use when: you're running multi-tenant or high-volume production, you need autoscaling, you already have a Kubernetes platform.

See [Deploy on Kubernetes guide](../guides/deploy-kubernetes.md).

## Temporal mode

```mermaid
graph LR
  Client --> App[Your service]
  App --> Ex[workflow.DurableExecutor]
  Ex --> TC[Temporal cluster]
  TC --> W[Temporal worker]
  W --> LLM
  W --> Tool
  TC --> EL[Event log · persistent]
```

Write the agent loop as a `workflow.WorkflowFunc`, with LLM and tool calls as activities, and run it through a `workflow.DurableExecutor`. `workflow/providers/temporal` starts workflows on Temporal and forwards signals, queries and cancellation, but does not yet ship the worker side; see the [Temporal guide](../guides/deploy-temporal.md) status note. The Runner does not hand its loop to a workflow engine by itself.

Use when:
- Workflows run for minutes, hours, or days.
- You need human-in-the-loop (signals).
- You already run Temporal and want Beluga agents alongside other workflows.

See [DOC-16](./16-durable-workflows.md) and [Deploy on Temporal guide](../guides/deploy-temporal.md).

## Choosing a mode

```mermaid
graph TD
  Start[What's your deployment context?]
  Start --> Emb[Embedded in another app]
  Start --> Small[Small/medium production, single host]
  Start --> Big[Multi-node production]
  Start --> Long[Long-running / HITL]

  Emb --> Lib[Library]
  Small --> Dock[Docker]
  Big --> K8s[Kubernetes]
  Long --> Temp[Temporal]
```

These aren't mutually exclusive. A large deployment might run some agents in Kubernetes for general use and a subset in Temporal for long-running research tasks — same agent code, different runners in each case.

## What changes between modes

| Concern | Library | Docker | Kubernetes | Temporal |
|---|---|---|---|---|
| Runner | optional | yes | yes | yes |
| Session service | in-memory | redis/postgres | redis/postgres | Temporal state |
| Protocol endpoints | no | REST/SSE/WS | REST/SSE/A2A/MCP | internal |
| Durability | no | optional | optional | yes |
| Autoscaling | no | manual | HPA | Temporal worker scaling |
| Secrets | env vars | env/compose | K8s Secrets | K8s Secrets + Temporal config |

## What doesn't change

- The agent code.
- The tools, memory, planner, hooks.
- The guards.
- The reasoning strategy.
- The integration tests.

This is the payoff of keeping the Runner separate from the Agent — the decision about how to deploy is orthogonal to the decision about how the agent behaves.

## Common mistakes

- **Library mode in production.** Fine for CLIs. Disastrous for multi-user services (no session isolation, no rate limiting, no observability).
- **Kubernetes for a single-tenant internal tool.** Docker is simpler and cheaper if you don't need the features.
- **Temporal for short turns.** Temporal's overhead per activity is non-trivial. If turns finish in seconds, use the built-in engine or no durability at all.
- **Custom session services when Redis works.** Redis handles sessions for most deployments. Only roll your own if you have specific requirements.

## Related reading

- [08 — Runner and Lifecycle](./08-runner-and-lifecycle.md) — the Runner is the unit being deployed.
- [16 — Durable Workflows](./16-durable-workflows.md) — when to choose Temporal mode.
- [12 — Protocol Layer](./12-protocol-layer.md) — protocol exposure varies by mode.
