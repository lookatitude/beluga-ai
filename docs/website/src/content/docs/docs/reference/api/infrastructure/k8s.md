---
title: "K8s API — Kubernetes Operator, CRDs, and Helm Chart"
description: "K8s operator API reference for Beluga AI. Agent and Team custom resource types, Reconciler interface, Controller reconciliation loop, and Helm chart deployment."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "k8s API, Kubernetes operator, CRD, Agent CRD, Team CRD, Reconciler, Controller, Helm, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/k8s/operator"
```

Package operator implements a Kubernetes operator for Beluga AI agents and
teams. It defines custom resource types (Agent and Team), a Reconciler that
computes desired Kubernetes state from those resources, and a Controller that
drives the reconciliation loop. The operator has no dependency on any
Kubernetes client library — ResourceStore and StatusWriter are the seams
through which K8s client code is injected.

## Custom Resource Definitions

CRD manifests are located in `k8s/crds/`:

| File | Kind | API Group |
|------|------|-----------|
| `agent.yaml` | `Agent` | `beluga.ai/v1` |
| `team.yaml` | `Team` | `beluga.ai/v1` |
| `modelconfig.yaml` | `ModelConfig` | `beluga.ai/v1` |
| `toolserver.yaml` | `ToolServer` | `beluga.ai/v1` |
| `guardpolicy.yaml` | `GuardPolicy` | `beluga.ai/v1` |
| `memorystore.yaml` | `MemoryStore` | `beluga.ai/v1` |

Apply all CRDs before deploying the operator:

```bash
kubectl apply -f k8s/crds/
```

## Agent CRD

AgentResource is the Go representation of the `Agent` custom resource.

```go
type AgentResource struct {
    APIVersion string     // "beluga.ai/v1"
    Kind       string     // "Agent"
    Meta       ObjectMeta
    Spec       AgentSpec
    Status     AgentStatus
}

type AgentSpec struct {
    Persona       Persona
    Planner       string       // e.g. "react" or "openai-functions"
    MaxIterations int          // must be >= 0
    ModelRef      string       // name of a ModelConfig CR; required when ModelConfig is nil
    ModelConfig   *ModelConfig // inline model config; takes precedence over ModelRef
    ToolRefs      []string     // names of Tool CRDs this agent may invoke
    Replicas      int          // desired pods; defaults to 1
    Resources     Resources    // CPU/memory requests and limits
    Scaling       ScalingConfig
    Port          int          // container port; defaults to 8080
    Image         string       // container image; defaults to "beluga-agent:latest"
}

type AgentStatus struct {
    ReadyReplicas int
    Phase         string // "Pending", "Running", "Failed"
    Message       string
}
```

An Agent manifest:

```yaml
apiVersion: beluga.ai/v1
kind: Agent
metadata:
  name: planner
  namespace: beluga-system
spec:
  persona:
    role: "Planner"
    goal: "Decompose user requests into executable tasks"
  planner: react
  maxIterations: 10
  modelRef: gpt4o-config
  replicas: 2
  resources:
    requests:
      cpu: "250m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "256Mi"
  scaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 8
    targetCPUUtilization: 70
  port: 8080
```

## Team CRD

TeamResource is the Go representation of the `Team` custom resource.

```go
type TeamResource struct {
    APIVersion string     // "beluga.ai/v1"
    Kind       string     // "Team"
    Meta       ObjectMeta
    Spec       TeamSpec
    Status     TeamStatus
}

type TeamSpec struct {
    Members        []TeamMemberRef // at least one required; names must be unique
    Pattern        string          // "sequential", "parallel", "supervisor"
    Supervisor     string          // agent name; used when Pattern is "supervisor"
    MaxConcurrent  int             // 0 means unlimited
    SharedToolRefs []string        // Tool CRDs available to all members
}

type TeamMemberRef struct {
    Name string // Agent CRD name
    Role string // e.g. "planner", "executor"
}

type TeamStatus struct {
    ActiveMembers int
    Phase         string
    Message       string
}
```

A Team manifest:

```yaml
apiVersion: beluga.ai/v1
kind: Team
metadata:
  name: analysis-team
  namespace: beluga-system
spec:
  pattern: sequential
  members:
    - name: researcher
      role: planner
    - name: writer
      role: executor
    - name: reviewer
      role: reviewer
  maxConcurrent: 2
```

## Reconciler Interface

Reconciler computes the desired Kubernetes resource state from a CRD. It
returns plain Go structs — it never calls the Kubernetes API directly.
Callers apply the returned ReconcileResult using their preferred K8s client.

```go
type Reconciler interface {
    ReconcileAgent(ctx context.Context, agent *AgentResource) (*ReconcileResult, error)
    ReconcileTeam(ctx context.Context, team *TeamResource) (*TeamReconcileResult, error)
}
```

ReconcileResult contains the desired Deployment, Service, and optionally an
HPA spec:

```go
type ReconcileResult struct {
    Deployment DeploymentSpec
    Service    ServiceSpec
    HPA        *HPASpec // nil when Scaling.Enabled is false
}

type TeamReconcileResult struct {
    Members map[string]*ReconcileResult // keyed by member agent name
}
```

### DefaultReconciler

DefaultReconciler is the built-in Reconciler. It applies sensible defaults:
image `"beluga-agent:latest"`, port `8080`, replicas `1`. When Scaling is
enabled, HPA defaults to minReplicas=1, maxReplicas=5, targetCPUUtilization=80.

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/k8s/operator"
)

reconciler := operator.NewDefaultReconciler(
    operator.WithDefaultImage("my-registry/beluga-agent:v1.2.0"),
    operator.WithDefaultPort(8080),
    operator.WithDefaultReplicas(2),
)

result, err := reconciler.ReconcileAgent(context.Background(), &operator.AgentResource{
    APIVersion: "beluga.ai/v1",
    Kind:       "Agent",
    Meta:       operator.ObjectMeta{Name: "planner", Namespace: "beluga-system"},
    Spec: operator.AgentSpec{
        ModelRef:      "gpt4o-config",
        MaxIterations: 10,
        Replicas:      3,
        Scaling: operator.ScalingConfig{
            Enabled:              true,
            MaxReplicas:          10,
            TargetCPUUtilization: 75,
        },
    },
})
if err != nil {
    fmt.Println("reconcile error:", err)
} else {
    fmt.Printf("deployment replicas: %d\n", result.Deployment.Replicas)
    fmt.Printf("HPA max replicas: %d\n", result.HPA.MaxReplicas)
}
```

ReconcileAgent returns `core.ErrInvalidInput` when ModelRef is empty and
ModelConfig is nil, or when MaxIterations is negative.

ReconcileTeam reconciles each member independently using a synthesised
AgentResource. It returns the first member-level error if any member fails,
after attempting all members.

## Controller

Controller drives the reconciliation loop. It maintains an internal work queue
(capacity 256) and spawns configurable worker goroutines. The controller
depends on a ResourceStore to read CRDs and a StatusWriter to persist observed
status — both are caller-provided interfaces that wrap a K8s client.

```go
type ResourceStore interface {
    GetAgent(ctx context.Context, namespace, name string) (*AgentResource, error)
    GetTeam(ctx context.Context, namespace, name string) (*TeamResource, error)
}

type StatusWriter interface {
    UpdateAgentStatus(ctx context.Context, namespace, name string, status AgentStatus) error
    UpdateTeamStatus(ctx context.Context, namespace, name string, status TeamStatus) error
}
```

```go
import (
    "context"

    "github.com/lookatitude/beluga-ai/k8s/operator"
)

reconciler := operator.NewDefaultReconciler()
ctrl := operator.NewController(reconciler, myStore, myWriter,
    operator.WithNamespace("beluga-system"),
    operator.WithWorkers(4),
)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := ctrl.Start(ctx); err != nil {
    // handle error
}

// Enqueue a reconcile request when a watch event fires.
ctrl.EnqueueAgent("beluga-system", "planner")
ctrl.EnqueueTeam("beluga-system", "analysis-team")

// Graceful shutdown — waits for in-flight reconciliations.
ctrl.Stop()
```

### Controller Functional Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithNamespace(ns string)` | `""` (all namespaces) | Restrict the controller to one namespace. |
| `WithWorkers(n int)` | 1 | Number of concurrent reconciliation goroutines. Values <= 0 are normalized to 1. |

### ReconcileRequest

```go
type ReconcileRequest struct {
    Kind      string // "Agent" or "Team"
    Name      string
    Namespace string
}
```

EnqueueAgent and EnqueueTeam drop requests when the queue is full rather than
blocking. A production controller should log dropped requests and implement
re-queue with backoff via the watch layer.

## Helm Chart

The Helm chart at `k8s/helm/` deploys the operator and its RBAC resources into
a cluster.

```bash
# Install with default values.
helm install beluga-operator ./k8s/helm \
  --namespace beluga-system \
  --create-namespace

# Upgrade with custom values.
helm upgrade beluga-operator ./k8s/helm \
  --set operator.workers=4 \
  --set operator.image.tag=v1.2.0
```

Key values in `k8s/helm/values.yaml`:

| Value | Default | Description |
|-------|---------|-------------|
| `operator.image.repository` | `beluga-operator` | Operator container image. |
| `operator.image.tag` | `latest` | Image tag. |
| `operator.workers` | `2` | Reconciliation worker count. |
| `operator.namespace` | `""` | Restrict to namespace; empty means cluster-scoped. |
| `rbac.create` | `true` | Create ClusterRole and ClusterRoleBinding. |

## Related Packages

- `deploy` — Generates Dockerfiles and Docker Compose manifests for local agent packaging.
- `runtime` — Agent lifecycle management used inside each deployed pod.
- `agent` — Agent interface that the operator configures via CRD specs.
- `docs/packages.md` — Package layout showing where the operator fits in the overall architecture.
