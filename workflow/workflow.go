// Package workflow provides a durable execution engine for the Beluga AI framework.
// It enables reliable, long-running workflows with activity execution, signal handling,
// retry policies, and event-sourced state persistence.
//
// Usage:
//
//	executor := workflow.NewExecutor(workflow.WithStore(store))
//	handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
//	    ID: "order-123",
//	})
//	result, err := handle.Result(ctx)
package workflow

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// WorkflowStatus represents the lifecycle state of a workflow execution.
type WorkflowStatus string

const (
	// StatusRunning indicates the workflow is currently executing.
	StatusRunning WorkflowStatus = "running"
	// StatusCompleted indicates the workflow has finished successfully.
	StatusCompleted WorkflowStatus = "completed"
	// StatusFailed indicates the workflow has failed.
	StatusFailed WorkflowStatus = "failed"
	// StatusCanceled indicates the workflow was canceled.
	StatusCanceled WorkflowStatus = "canceled"
)

// Signal carries an external message to a running workflow.
type Signal struct {
	// Name identifies the signal type.
	Name string
	// Payload carries the signal data.
	Payload any
}

// WorkflowOptions configures a workflow execution.
type WorkflowOptions struct {
	// ID is a user-supplied workflow identifier. If empty, one is generated.
	ID string
	// Input is the initial input to the workflow function.
	Input any
	// Timeout is the maximum duration for the workflow.
	Timeout time.Duration
}

// WorkflowFunc is a function that defines workflow logic. It receives a
// WorkflowContext for deterministic execution.
type WorkflowFunc func(ctx WorkflowContext, input any) (any, error)

// DurableExecutor is the interface for executing and managing durable workflows.
type DurableExecutor interface {
	// Execute starts a new workflow execution.
	Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error)

	// Signal sends a signal to a running workflow.
	Signal(ctx context.Context, workflowID string, signal Signal) error

	// Query retrieves state from a running workflow.
	Query(ctx context.Context, workflowID string, queryType string) (any, error)

	// Cancel requests cancellation of a running workflow.
	Cancel(ctx context.Context, workflowID string) error
}

// WorkflowHandle provides access to a running or completed workflow.
type WorkflowHandle interface {
	// ID returns the workflow identifier.
	ID() string
	// RunID returns the run identifier for this execution.
	RunID() string
	// Status returns the current workflow status.
	Status() WorkflowStatus
	// Result blocks until the workflow completes and returns its result.
	Result(ctx context.Context) (any, error)
}

// ActivityFunc is a function that performs a unit of work within a workflow.
type ActivityFunc func(ctx context.Context, input any) (any, error)

// ActivityOption configures an activity execution.
type ActivityOption func(*activityConfig)

type activityConfig struct {
	retryPolicy *RetryPolicy
	timeout     time.Duration
}

// WithActivityRetry sets the retry policy for an activity.
func WithActivityRetry(p RetryPolicy) ActivityOption {
	return func(c *activityConfig) {
		c.retryPolicy = &p
	}
}

// WithActivityTimeout sets a timeout for an activity execution.
func WithActivityTimeout(d time.Duration) ActivityOption {
	return func(c *activityConfig) {
		c.timeout = d
	}
}

// Factory creates a DurableExecutor from configuration.
type Factory func(cfg Config) (DurableExecutor, error)

// Config holds configuration for creating a DurableExecutor via the registry.
type Config struct {
	// Store is the workflow state store.
	Store WorkflowStore
	// Extra holds provider-specific configuration.
	Extra map[string]any
}

// Package-level registry.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a named DurableExecutor factory. Panics if name is empty,
// factory is nil, or name is already registered.
func Register(name string, f Factory) {
	if name == "" {
		panic("workflow: Register called with empty name")
	}
	if f == nil {
		panic("workflow: Register called with nil factory for " + name)
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry[name]; dup {
		panic("workflow: Register called twice for " + name)
	}
	registry[name] = f
}

// New creates a DurableExecutor by registered name.
func New(name string, cfg Config) (DurableExecutor, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("workflow: unknown executor %q", name)
	}
	return f(cfg)
}

// List returns sorted names of registered executor factories.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
