// Package workflow provides a durable execution engine for the Beluga AI framework.
//
// It enables reliable, long-running workflows with activity execution, signal
// handling, retry policies, and event-sourced state persistence. The package
// provides its own built-in execution engine and supports external providers
// (Temporal, Dapr, Inngest, Kafka, NATS) via the registry pattern.
//
// # Core Interfaces
//
// The [DurableExecutor] interface manages workflow lifecycle:
//
//	type DurableExecutor interface {
//	    Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error)
//	    Signal(ctx context.Context, workflowID string, signal Signal) error
//	    Query(ctx context.Context, workflowID string, queryType string) (any, error)
//	    Cancel(ctx context.Context, workflowID string) error
//	}
//
// [WorkflowContext] extends context.Context with deterministic execution primitives:
//
//	type WorkflowContext interface {
//	    context.Context
//	    ExecuteActivity(fn ActivityFunc, input any, opts ...ActivityOption) (any, error)
//	    ReceiveSignal(name string) <-chan any
//	    Sleep(d time.Duration) error
//	}
//
// [WorkflowStore] persists workflow state for recovery and auditing.
//
// # Defining Workflows
//
// Workflows are plain Go functions that receive a [WorkflowContext]:
//
//	func OrderWorkflow(ctx workflow.WorkflowContext, input any) (any, error) {
//	    // Execute an activity with retry
//	    result, err := ctx.ExecuteActivity(processPayment, input,
//	        workflow.WithActivityRetry(workflow.DefaultRetryPolicy()),
//	        workflow.WithActivityTimeout(30 * time.Second),
//	    )
//	    if err != nil {
//	        return nil, err
//	    }
//
//	    // Wait for an external signal
//	    ch := ctx.ReceiveSignal("approval")
//	    select {
//	    case approval := <-ch:
//	        return approval, nil
//	    case <-ctx.Done():
//	        return nil, ctx.Err()
//	    }
//	}
//
// # Executing Workflows
//
// Use the [DefaultExecutor] or create one via the registry:
//
//	executor := workflow.NewExecutor(
//	    workflow.WithStore(inmemory.New()),
//	    workflow.WithExecutorHooks(hooks),
//	)
//
//	handle, err := executor.Execute(ctx, OrderWorkflow, workflow.WorkflowOptions{
//	    ID:      "order-123",
//	    Input:   orderData,
//	    Timeout: 10 * time.Minute,
//	})
//
//	result, err := handle.Result(ctx)
//
// # Activity Helpers
//
// Pre-built activity constructors integrate with framework components:
//
//   - [LLMActivity] — wraps an LLM invocation as an activity
//   - [ToolActivity] — wraps a tool execution as an activity
//   - [HumanActivity] — wraps human-in-the-loop interaction as an activity
//
// # Retry Policies
//
// [RetryPolicy] configures exponential backoff with jitter for activities:
//
//	policy := workflow.RetryPolicy{
//	    MaxAttempts:        5,
//	    InitialInterval:   100 * time.Millisecond,
//	    BackoffCoefficient: 2.0,
//	    MaxInterval:        30 * time.Second,
//	}
//
// # Signals and Queries
//
// Running workflows can receive external [Signal] messages and respond
// to queries:
//
//	err := executor.Signal(ctx, "order-123", workflow.Signal{
//	    Name:    "approval",
//	    Payload: "approved",
//	})
//
//	status, err := executor.Query(ctx, "order-123", "status")
//
// # Event-Sourced State
//
// Workflow execution is recorded as a sequence of [HistoryEvent] values in
// [WorkflowState]. This enables replay-based recovery and audit trails.
//
// # Registry
//
// External providers register via [Register] and are created with [New]:
//
//	// Registration (typically in init())
//	workflow.Register("temporal", temporalFactory)
//
//	// Creation
//	executor, err := workflow.New("temporal", workflow.Config{
//	    Extra: map[string]any{"client": temporalClient},
//	})
//
//	providers := workflow.List() // ["default", "temporal", ...]
//
// # Hooks and Middleware
//
// [Hooks] provide lifecycle callbacks for workflow events. [Middleware] wraps
// a DurableExecutor to add cross-cutting behavior:
//
//	hooks := workflow.Hooks{
//	    OnWorkflowStart:    func(ctx context.Context, id string, input any) { ... },
//	    OnWorkflowComplete: func(ctx context.Context, id string, result any) { ... },
//	    OnWorkflowFail:     func(ctx context.Context, id string, err error) { ... },
//	}
//
//	wrapped := workflow.ApplyMiddleware(executor, workflow.WithHooks(hooks))
package workflow
