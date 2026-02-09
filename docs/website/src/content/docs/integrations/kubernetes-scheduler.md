---
title: Kubernetes Job Scheduler
description: Schedule and orchestrate Beluga AI workflows as Kubernetes Jobs for scalable, distributed execution with automatic resource management and fault tolerance.
---

Kubernetes provides container orchestration for running distributed AI workloads. This guide covers scheduling Beluga AI workflows as Kubernetes Jobs, enabling automatic scaling, resource management, and fault tolerance for long-running or compute-intensive agent tasks.

## Overview

Running Beluga AI workflows as Kubernetes Jobs provides:

- **Distributed execution** -- run workflows across a cluster of nodes
- **Resource management** -- set CPU and memory limits per workflow
- **Fault tolerance** -- automatic restart with configurable backoff
- **Auto-cleanup** -- TTL-based job cleanup after completion
- **Observability** -- native Kubernetes monitoring and logging

This integration uses the Kubernetes Go client (`client-go`) to create and manage Jobs programmatically from your application.

## Prerequisites

- Go 1.23 or later
- Access to a Kubernetes cluster (local or remote)
- `kubectl` configured with cluster access
- Beluga AI framework installed

## Installation

Install the Kubernetes Go client libraries:

```bash
go get k8s.io/client-go@latest
go get k8s.io/api@latest
go get k8s.io/apimachinery@latest
```

Verify cluster access:

```bash
kubectl cluster-info
kubectl config view --minify
```

## Configuration

### Job Scheduler

Create a scheduler that submits Beluga AI workflows as Kubernetes Jobs:

```go
package main

import (
    "context"
    "fmt"

    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
)

// KubernetesJobScheduler manages Beluga AI workflow execution as Kubernetes Jobs.
type KubernetesJobScheduler struct {
    clientset *kubernetes.Clientset
    namespace string
}

// NewKubernetesJobScheduler creates a scheduler. Pass an empty kubeconfig
// to use in-cluster configuration (when running inside a pod).
func NewKubernetesJobScheduler(kubeconfig string, namespace string) (*KubernetesJobScheduler, error) {
    var config *rest.Config
    var err error

    if kubeconfig == "" {
        config, err = rest.InClusterConfig()
    } else {
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to build config: %w", err)
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }

    return &KubernetesJobScheduler{
        clientset: clientset,
        namespace: namespace,
    }, nil
}

// ScheduleWorkflow creates a Kubernetes Job for the given workflow.
func (s *KubernetesJobScheduler) ScheduleWorkflow(ctx context.Context, workflowID string, image string, command []string) (*batchv1.Job, error) {
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("beluga-workflow-%s", workflowID),
            Namespace: s.namespace,
            Labels: map[string]string{
                "app":         "beluga-ai",
                "workflow-id": workflowID,
            },
        },
        Spec: batchv1.JobSpec{
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:    "workflow",
                            Image:   image,
                            Command: command,
                        },
                    },
                    RestartPolicy: corev1.RestartPolicyNever,
                },
            },
            BackoffLimit: int32Ptr(3),
        },
    }

    created, err := s.clientset.BatchV1().Jobs(s.namespace).Create(ctx, job, metav1.CreateOptions{})
    if err != nil {
        return nil, fmt.Errorf("failed to create job: %w", err)
    }

    return created, nil
}

func int32Ptr(i int32) *int32 { return &i }
```

### Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Namespace` | Kubernetes namespace for jobs | `default` | No |
| `Image` | Container image for the workflow | - | Yes |
| `BackoffLimit` | Maximum retry attempts on failure | `3` | No |
| `TTLSecondsAfterFinished` | Seconds to keep completed jobs | `3600` | No |

## Usage

### Scheduling a Workflow

Submit a Beluga AI workflow as a Kubernetes Job:

```go
func main() {
    ctx := context.Background()

    scheduler, err := NewKubernetesJobScheduler("", "default")
    if err != nil {
        log.Fatalf("failed to create scheduler: %v", err)
    }

    job, err := scheduler.ScheduleWorkflow(ctx, "workflow-123",
        "beluga-ai:latest",
        []string{"beluga", "run", "--workflow", "workflow-123"},
    )
    if err != nil {
        log.Fatalf("failed to schedule workflow: %v", err)
    }

    fmt.Printf("Job created: %s\n", job.Name)
}
```

### Monitoring Job Status

Poll for job completion with a timeout:

```go
import "time"

// WaitForCompletion polls the job status until it succeeds, fails, or the timeout expires.
func (s *KubernetesJobScheduler) WaitForCompletion(ctx context.Context, jobName string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        job, err := s.clientset.BatchV1().Jobs(s.namespace).Get(ctx, jobName, metav1.GetOptions{})
        if err != nil {
            return fmt.Errorf("failed to get job status: %w", err)
        }

        if job.Status.Succeeded > 0 {
            return nil
        }
        if job.Status.Failed > 0 {
            return fmt.Errorf("job %s failed after %d attempts", jobName, job.Status.Failed)
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(2 * time.Second):
        }
    }

    return fmt.Errorf("timeout waiting for job %s to complete", jobName)
}
```

### Scheduling with Resource Limits

For production workloads, specify CPU and memory constraints:

```go
import (
    "k8s.io/apimachinery/pkg/api/resource"
)

// WorkflowConfig holds resource requirements for a workflow job.
type WorkflowConfig struct {
    Image         string
    Command       []string
    CPURequest    resource.Quantity
    MemoryRequest resource.Quantity
    CPULimit      resource.Quantity
    MemoryLimit   resource.Quantity
}

func (s *KubernetesJobScheduler) ScheduleWithResources(ctx context.Context, workflowID string, config WorkflowConfig) (*batchv1.Job, error) {
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("beluga-%s-%d", workflowID, time.Now().Unix()),
            Namespace: s.namespace,
            Labels: map[string]string{
                "app":         "beluga-ai",
                "workflow-id": workflowID,
            },
        },
        Spec: batchv1.JobSpec{
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:    "workflow",
                            Image:   config.Image,
                            Command: config.Command,
                            Resources: corev1.ResourceRequirements{
                                Requests: corev1.ResourceList{
                                    corev1.ResourceCPU:    config.CPURequest,
                                    corev1.ResourceMemory: config.MemoryRequest,
                                },
                                Limits: corev1.ResourceList{
                                    corev1.ResourceCPU:    config.CPULimit,
                                    corev1.ResourceMemory: config.MemoryLimit,
                                },
                            },
                        },
                    },
                    RestartPolicy: corev1.RestartPolicyNever,
                },
            },
            BackoffLimit:            int32Ptr(3),
            TTLSecondsAfterFinished: int32Ptr(3600),
        },
    }

    created, err := s.clientset.BatchV1().Jobs(s.namespace).Create(ctx, job, metav1.CreateOptions{})
    if err != nil {
        return nil, fmt.Errorf("failed to create job: %w", err)
    }

    return created, nil
}
```

## Advanced Topics

### OpenTelemetry Instrumentation

Add tracing spans around job scheduling operations:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type TracedScheduler struct {
    *KubernetesJobScheduler
    tracer trace.Tracer
}

func NewTracedScheduler(kubeconfig, namespace string) (*TracedScheduler, error) {
    base, err := NewKubernetesJobScheduler(kubeconfig, namespace)
    if err != nil {
        return nil, err
    }

    return &TracedScheduler{
        KubernetesJobScheduler: base,
        tracer:                 otel.Tracer("beluga.orchestration.kubernetes"),
    }, nil
}

func (s *TracedScheduler) ScheduleWorkflow(ctx context.Context, workflowID string, image string, command []string) (*batchv1.Job, error) {
    ctx, span := s.tracer.Start(ctx, "kubernetes.schedule_workflow",
        trace.WithAttributes(
            attribute.String("workflow.id", workflowID),
            attribute.String("k8s.namespace", s.namespace),
            attribute.String("k8s.container.image", image),
        ),
    )
    defer span.End()

    job, err := s.KubernetesJobScheduler.ScheduleWorkflow(ctx, workflowID, image, command)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(attribute.String("k8s.job.name", job.Name))
    return job, nil
}
```

### RBAC Configuration

Create a Kubernetes Role that grants the scheduler permission to manage jobs:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: beluga-scheduler
  namespace: default
rules:
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["create", "get", "list", "watch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: beluga-scheduler-binding
  namespace: default
subjects:
  - kind: ServiceAccount
    name: beluga-scheduler
    namespace: default
roleRef:
  kind: Role
  name: beluga-scheduler
  apiGroup: rbac.authorization.k8s.io
```

### Production Considerations

When deploying Kubernetes-scheduled workflows to production:

- **Resource limits**: Always set CPU and memory limits. Unbounded pods can starve other workloads on the node.
- **Namespaces**: Use dedicated namespaces for Beluga AI workloads to isolate resources and RBAC policies.
- **TTL cleanup**: Set `TTLSecondsAfterFinished` to automatically remove completed jobs. Without this, completed jobs accumulate indefinitely.
- **Monitoring**: Use `kubectl get jobs -l app=beluga-ai` to list all Beluga jobs. Integrate with Prometheus for metrics on job duration and failure rates.
- **Image management**: Use versioned image tags (not `latest`) and configure image pull secrets for private registries.

## Troubleshooting

### "Unauthorized"

The service account lacks permission to create jobs. Apply the RBAC configuration shown above, then verify:

```bash
kubectl auth can-i create jobs --as=system:serviceaccount:default:beluga-scheduler
```

### "Image pull failed"

The container image is not accessible from the cluster. Verify the image exists and configure pull secrets if needed:

```bash
# Check if the image is accessible
kubectl run test --image=beluga-ai:latest --rm -it --restart=Never -- echo "ok"

# For private registries, create an image pull secret
kubectl create secret docker-registry beluga-registry \
  --docker-server=registry.example.com \
  --docker-username=user \
  --docker-password=pass
```

## Related Resources

- [NATS Message Bus](/integrations/nats-message-bus) -- Distributed messaging for agent coordination
- [Infrastructure](/integrations/infrastructure) -- Infrastructure integration patterns
- [Monitoring](/integrations/monitoring) -- Observability and tracing setup
