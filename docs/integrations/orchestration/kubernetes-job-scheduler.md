# Kubernetes Job Scheduler

Welcome, colleague! In this integration guide, we're going to integrate Kubernetes for scheduling and orchestrating Beluga AI workflows. Kubernetes provides container orchestration for running distributed AI workloads.

## What you will build

You will configure Beluga AI to schedule and manage workflows as Kubernetes Jobs, enabling scalable, distributed execution of AI tasks with automatic scaling, resource management, and fault tolerance.

## Learning Objectives

- ✅ Create Kubernetes Jobs from Beluga AI workflows
- ✅ Schedule workflow execution
- ✅ Monitor job status
- ✅ Handle job completion and failures

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Kubernetes cluster access
- Kubernetes Go client

## Step 1: Setup and Installation

Install Kubernetes Go client:
bash
```bash
go get k8s.io/client-go/kubernetes
go get k8s.io/client-go/tools/clientcmd
```

Configure kubectl:
```bash
kubectl config view


## Step 2: Create Kubernetes Job Scheduler

Create a Kubernetes job scheduler:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
)

type KubernetesJobScheduler struct {
    clientset *kubernetes.Clientset
    namespace string
}

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

func (s *KubernetesJobScheduler) ScheduleWorkflow(ctx context.Context, workflowID string, image string, command []string) (*batchv1.Job, error) {
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("beluga-workflow-%s", workflowID),
            Namespace: s.namespace,
            Labels: map[string]string{
                "app":        "beluga-ai",
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
```

## Step 3: Monitor Job Status

Monitor job execution:
```go
func (s *KubernetesJobScheduler) WaitForCompletion(ctx context.Context, jobName string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        job, err := s.clientset.BatchV1().Jobs(s.namespace).Get(ctx, jobName, metav1.GetOptions{})
        if err != nil {
            return fmt.Errorf("failed to get job: %w", err)
        }

        

        if job.Status.Succeeded > 0 {
            return nil // Job succeeded
        }
        if job.Status.Failed > 0 {
            return fmt.Errorf("job failed")
        }
        
        time.Sleep(2 * time.Second)
    }
    
    return fmt.Errorf("timeout waiting for job completion")
}
```

## Step 4: Use with Beluga AI Workflows

Schedule Beluga AI workflows:
```go
func main() {
    ctx := context.Background()
    
    // Create scheduler
    scheduler, err := NewKubernetesJobScheduler("", "default")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    // Schedule workflow
    job, err := scheduler.ScheduleWorkflow(ctx, "workflow-123", 
        "beluga-ai:latest",
        []string{"beluga", "run", "--workflow", "workflow-123"},
    )
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    

    fmt.Printf("Job created: %s\n", job.Name)
    
    // Wait for completion
    err = scheduler.WaitForCompletion(ctx, job.Name, 5*time.Minute)
    if err != nil {
        log.Fatalf("Job failed: %v", err)
    }
    
    fmt.Println("Job completed successfully")
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionKubernetesScheduler struct {
    clientset *kubernetes.Clientset
    namespace string
    tracer    trace.Tracer
}

func NewProductionKubernetesScheduler(namespace string) (*ProductionKubernetesScheduler, error) {
    config, err := rest.InClusterConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to get config: %w", err)
    }
    
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }
    
    return &ProductionKubernetesScheduler{
        clientset: clientset,
        namespace: namespace,
        tracer:    otel.Tracer("beluga.orchestration.kubernetes"),
    }, nil
}

func (s *ProductionKubernetesScheduler) ScheduleWorkflow(ctx context.Context, workflowID string, config WorkflowConfig) (*batchv1.Job, error) {
    ctx, span := s.tracer.Start(ctx, "kubernetes.schedule",
        trace.WithAttributes(
            attribute.String("workflow_id", workflowID),
            attribute.String("namespace", s.namespace),
        ),
    )
    defer span.End()
    
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("beluga-%s-%d", workflowID, time.Now().Unix()),
            Namespace: s.namespace,
            Labels: map[string]string{
                "app":        "beluga-ai",
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
            BackoffLimit: int32Ptr(3),
            TTLSecondsAfterFinished: int32Ptr(3600), // Clean up after 1 hour
        },
    }
    
    created, err := s.clientset.BatchV1().Jobs(s.namespace).Create(ctx, job, metav1.CreateOptions{})
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("create failed: %w", err)
    }
    
    span.SetAttributes(attribute.String("job_name", created.Name))
    return created, nil
}

type WorkflowConfig struct {
    Image        string
    Command      []string
    CPURequest   resource.Quantity
    MemoryRequest resource.Quantity
    CPULimit     resource.Quantity
    MemoryLimit  resource.Quantity
}

func int32Ptr(i int32) *int32 { return &i }

func main() {
    ctx := context.Background()
    
    scheduler, err := NewProductionKubernetesScheduler("default")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    config := WorkflowConfig{
        Image:   "beluga-ai:latest",
        Command: []string{"beluga", "run", "--workflow", "my-workflow"},
    }
    
    job, err := scheduler.ScheduleWorkflow(ctx, "my-workflow", config)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    
    fmt.Printf("Job scheduled: %s\n", job.Name)
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Namespace` | Kubernetes namespace | `default` | No |
| `Image` | Container image | - | Yes |
| `BackoffLimit` | Maximum retries | `3` | No |
| `TTL` | Job cleanup TTL | `1h` | No |

## Common Issues

### "Unauthorized"

**Problem**: Missing Kubernetes permissions.

**Solution**: Create RBAC role:apiVersion: rbac.authorization.k8s.io/v1
```yaml
kind: Role
metadata:
  name: beluga-scheduler
rules:
```
- apiGroups: ["batch"]
```text
  resources: ["jobs"]
  verbs: ["create", "get", "list"]


### "Image pull failed"

**Problem**: Container image not accessible.

**Solution**: Ensure image is available in cluster or use image pull secrets.

## Production Considerations

When using Kubernetes in production:

- **Resource limits**: Set appropriate CPU/memory limits
- **RBAC**: Configure proper permissions
- **Namespaces**: Use dedicated namespaces
- **Monitoring**: Monitor job status and resource usage
- **Cleanup**: Configure TTL for job cleanup

## Next Steps

Congratulations! You've integrated Kubernetes with Beluga AI. Next, learn how to:

- **[NATS Message Bus](./nats-message-bus.md)** - Distributed messaging
- **[Orchestration Package Documentation](../../api-docs/packages/orchestration.md)** - Deep dive into orchestration
- **[Orchestration Tutorial](../../getting-started/06-orchestration-basics.md)** - Orchestration patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
