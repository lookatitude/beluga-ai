// Package temporal provides a Temporal-backed DurableExecutor and WorkflowStore
// for the Beluga workflow engine. It maps Beluga workflows to Temporal workflows,
// and Beluga activities to Temporal activities.
//
// Usage:
//
//	executor, err := temporal.NewExecutor(temporal.Config{
//	    Client:    temporalClient,
//	    TaskQueue: "beluga-workflows",
//	})
//
//	handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
//	    ID: "order-123",
//	})
//	result, err := handle.Result(ctx)
package temporal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	temporalworkflow "go.temporal.io/sdk/workflow"

	"github.com/lookatitude/beluga-ai/workflow"
)

// Config holds configuration for the Temporal executor.
type Config struct {
	// Client is the Temporal SDK client.
	Client client.Client
	// TaskQueue is the Temporal task queue for workflows and activities.
	TaskQueue string
	// DefaultTimeout is the default workflow execution timeout.
	DefaultTimeout time.Duration
}

// Executor implements workflow.DurableExecutor backed by Temporal.
type Executor struct {
	client    client.Client
	taskQueue string
	timeout   time.Duration
	workflows map[string]*temporalHandle
	mu        sync.RWMutex
}

// Compile-time interface check.
var _ workflow.DurableExecutor = (*Executor)(nil)

// NewExecutor creates a new Temporal-backed DurableExecutor.
func NewExecutor(cfg Config) (*Executor, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("temporal: client is required")
	}
	if cfg.TaskQueue == "" {
		cfg.TaskQueue = "beluga-workflows"
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 10 * time.Minute
	}

	return &Executor{
		client:    cfg.Client,
		taskQueue: cfg.TaskQueue,
		timeout:   cfg.DefaultTimeout,
		workflows: make(map[string]*temporalHandle),
	}, nil
}

// BelugaWorkflowInput is the serializable input passed to the Temporal workflow wrapper.
type BelugaWorkflowInput struct {
	Input any `json:"input"`
}

// BelugaWorkflowResult is the serializable result from the Temporal workflow wrapper.
type BelugaWorkflowResult struct {
	Result any    `json:"result"`
	Error  string `json:"error,omitempty"`
}

// Execute starts a new workflow using Temporal.
func (e *Executor) Execute(ctx context.Context, fn workflow.WorkflowFunc, opts workflow.WorkflowOptions) (workflow.WorkflowHandle, error) {
	if opts.ID == "" {
		opts.ID = fmt.Sprintf("beluga-wf-%d", time.Now().UnixNano())
	}

	timeout := e.timeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	startOpts := client.StartWorkflowOptions{
		ID:                       opts.ID,
		TaskQueue:                e.taskQueue,
		WorkflowExecutionTimeout: timeout,
	}

	// Store the workflow function for registration with workers.
	// The actual Temporal workflow is a wrapper that creates a temporalContext
	// and delegates to the Beluga WorkflowFunc.
	wrapper := newWorkflowWrapper(fn, e.taskQueue)

	run, err := e.client.ExecuteWorkflow(ctx, startOpts, wrapper.Run, opts.Input)
	if err != nil {
		return nil, fmt.Errorf("temporal/execute: %w", err)
	}

	handle := &temporalHandle{
		client: e.client,
		run:    run,
		id:     opts.ID,
		runID:  run.GetRunID(),
	}

	e.mu.Lock()
	e.workflows[opts.ID] = handle
	e.mu.Unlock()

	return handle, nil
}

// Signal sends a signal to a running Temporal workflow.
func (e *Executor) Signal(ctx context.Context, workflowID string, signal workflow.Signal) error {
	return e.client.SignalWorkflow(ctx, workflowID, "", signal.Name, signal.Payload)
}

// Query queries a running Temporal workflow.
func (e *Executor) Query(ctx context.Context, workflowID string, queryType string) (any, error) {
	encoded, err := e.client.QueryWorkflow(ctx, workflowID, "", queryType)
	if err != nil {
		return nil, fmt.Errorf("temporal/query: %w", err)
	}

	var result any
	if err := encoded.Get(&result); err != nil {
		return nil, fmt.Errorf("temporal/query: decode: %w", err)
	}
	return result, nil
}

// Cancel requests cancellation of a running Temporal workflow.
func (e *Executor) Cancel(ctx context.Context, workflowID string) error {
	return e.client.CancelWorkflow(ctx, workflowID, "")
}

// temporalHandle implements workflow.WorkflowHandle backed by a Temporal WorkflowRun.
type temporalHandle struct {
	client client.Client
	run    client.WorkflowRun
	id     string
	runID  string
}

// Compile-time interface check.
var _ workflow.WorkflowHandle = (*temporalHandle)(nil)

func (h *temporalHandle) ID() string    { return h.id }
func (h *temporalHandle) RunID() string { return h.runID }

func (h *temporalHandle) Status() workflow.WorkflowStatus {
	return workflow.StatusRunning
}

func (h *temporalHandle) Result(ctx context.Context) (any, error) {
	var result any
	if err := h.run.Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("temporal/result: %w", err)
	}
	return result, nil
}

// workflowWrapper wraps a Beluga WorkflowFunc as a Temporal-compatible function.
type workflowWrapper struct {
	fn        workflow.WorkflowFunc
	taskQueue string
}

func newWorkflowWrapper(fn workflow.WorkflowFunc, taskQueue string) *workflowWrapper {
	return &workflowWrapper{fn: fn, taskQueue: taskQueue}
}

// Run is the Temporal-compatible workflow function.
func (w *workflowWrapper) Run(ctx temporalworkflow.Context, input any) (any, error) {
	wfCtx := &temporalContext{
		tCtx:      ctx,
		taskQueue: w.taskQueue,
	}
	return w.fn(wfCtx, input)
}

// temporalContext implements workflow.WorkflowContext wrapping Temporal's
// workflow.Context. It does NOT embed workflow.Context because Temporal's
// Context.Done() returns internal.Channel, not <-chan struct{}.
type temporalContext struct {
	tCtx      temporalworkflow.Context
	taskQueue string
}

// Deadline implements context.Context.
func (c *temporalContext) Deadline() (time.Time, bool) {
	return c.tCtx.Deadline()
}

// Done implements context.Context. Returns nil since Temporal uses its own
// channel type. Callers should use workflow.Sleep or signal channels for
// cancellation-aware waiting.
func (c *temporalContext) Done() <-chan struct{} {
	// Temporal's Done() returns a workflow.Channel, not a Go channel.
	// We return nil; callers should check Err() instead.
	return nil
}

// Err implements context.Context.
func (c *temporalContext) Err() error {
	return c.tCtx.Err()
}

// Value implements context.Context.
func (c *temporalContext) Value(key any) any {
	return c.tCtx.Value(key)
}

func (c *temporalContext) ExecuteActivity(fn workflow.ActivityFunc, input any, opts ...workflow.ActivityOption) (any, error) {
	activityTimeout := 5 * time.Minute

	actOpts := temporalworkflow.ActivityOptions{
		TaskQueue:           c.taskQueue,
		StartToCloseTimeout: activityTimeout,
	}

	ctx := temporalworkflow.WithActivityOptions(c.tCtx, actOpts)
	future := temporalworkflow.ExecuteActivity(ctx, fn, input)

	var result any
	if err := future.Get(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *temporalContext) ReceiveSignal(name string) <-chan any {
	ch := make(chan any, 10)

	// Note: In a real Temporal workflow, this must be done using a Temporal
	// coroutine (workflow.Go). This is a bridge for the Beluga interface.
	temporalworkflow.Go(c.tCtx, func(ctx temporalworkflow.Context) {
		signalCh := temporalworkflow.GetSignalChannel(ctx, name)
		for {
			var payload any
			signalCh.Receive(ctx, &payload)
			ch <- payload
		}
	})

	return ch
}

func (c *temporalContext) Sleep(d time.Duration) error {
	return temporalworkflow.Sleep(c.tCtx, d)
}

// toTemporalRetryPolicy converts a Beluga RetryPolicy to a Temporal RetryPolicy.
func toTemporalRetryPolicy(p *workflow.RetryPolicy) *temporal.RetryPolicy {
	if p == nil {
		return nil
	}

	return &temporal.RetryPolicy{
		MaximumAttempts:    int32(p.MaxAttempts),
		InitialInterval:   p.InitialInterval,
		BackoffCoefficient: p.BackoffCoefficient,
		MaximumInterval:    p.MaxInterval,
	}
}

// Store implements workflow.WorkflowStore using Temporal's visibility API
// for listing and querying workflows. Since Temporal manages workflow state
// internally, Save and Delete are no-ops.
type Store struct {
	client    client.Client
	namespace string
}

// NewStore creates a new Temporal-backed workflow store.
func NewStore(c client.Client, namespace string) *Store {
	if namespace == "" {
		namespace = "default"
	}
	return &Store{
		client:    c,
		namespace: namespace,
	}
}

// Save is a no-op for Temporal as state is managed internally.
func (s *Store) Save(_ context.Context, _ workflow.WorkflowState) error {
	return nil
}

// Load retrieves workflow state by getting the Temporal workflow run.
func (s *Store) Load(ctx context.Context, workflowID string) (*workflow.WorkflowState, error) {
	run := s.client.GetWorkflow(ctx, workflowID, "")

	state := &workflow.WorkflowState{
		WorkflowID: workflowID,
		RunID:      run.GetRunID(),
		Status:     workflow.StatusRunning,
		UpdatedAt:  time.Now(),
	}

	return state, nil
}

// List returns an empty list. Full listing requires Temporal's list workflow
// API with visibility features enabled.
func (s *Store) List(_ context.Context, _ workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
	return nil, nil
}

// Delete is a no-op for Temporal.
func (s *Store) Delete(_ context.Context, _ string) error {
	return nil
}

// Compile-time interface checks.
var (
	_ workflow.DurableExecutor = (*Executor)(nil)
	_ workflow.WorkflowHandle  = (*temporalHandle)(nil)
	_ workflow.WorkflowStore   = (*Store)(nil)
	_ workflow.WorkflowContext = (*temporalContext)(nil)
)

func init() {
	workflow.Register("temporal", func(cfg workflow.Config) (workflow.DurableExecutor, error) {
		c, ok := cfg.Extra["client"].(client.Client)
		if !ok || c == nil {
			return nil, fmt.Errorf("temporal: client is required in Extra[\"client\"]")
		}
		taskQueue, _ := cfg.Extra["task_queue"].(string)
		if taskQueue == "" {
			taskQueue = "beluga-workflows"
		}
		return NewExecutor(Config{
			Client:    c,
			TaskQueue: taskQueue,
		})
	})
}
