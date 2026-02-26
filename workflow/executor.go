package workflow

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// seq is a package-level counter for generating unique IDs.
var seq atomic.Int64

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, seq.Add(1))
}

// ExecutorOption configures a DefaultExecutor.
type ExecutorOption func(*DefaultExecutor)

// WithStore sets the workflow store for the executor.
func WithStore(s WorkflowStore) ExecutorOption {
	return func(e *DefaultExecutor) {
		e.store = s
	}
}

// WithExecutorHooks sets the lifecycle hooks for the executor.
func WithExecutorHooks(h Hooks) ExecutorOption {
	return func(e *DefaultExecutor) {
		e.hooks = h
	}
}

// DefaultExecutor is a goroutine-based durable executor that runs workflows
// in-process. It records execution history for replay/recovery.
type DefaultExecutor struct {
	store    WorkflowStore
	hooks    Hooks
	running  map[string]*runningWorkflow
	mu       sync.RWMutex
}

type runningWorkflow struct {
	handle *defaultHandle
	cancel context.CancelFunc
	signals map[string]chan any
	mu      sync.Mutex
}

// NewExecutor creates a new DefaultExecutor with the given options.
func NewExecutor(opts ...ExecutorOption) *DefaultExecutor {
	e := &DefaultExecutor{
		running: make(map[string]*runningWorkflow),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Execute starts a new workflow execution.
func (e *DefaultExecutor) Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error) {
	if opts.ID == "" {
		opts.ID = generateID("wf")
	}

	runID := generateID("run")

	handle := &defaultHandle{
		id:     opts.ID,
		runID:  runID,
		status: StatusRunning,
		done:   make(chan struct{}),
	}

	wfCtx, cancel := context.WithCancel(ctx)
	if opts.Timeout > 0 {
		wfCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}

	rw := &runningWorkflow{
		handle:  handle,
		cancel:  cancel,
		signals: make(map[string]chan any),
	}

	e.mu.Lock()
	e.running[opts.ID] = rw
	e.mu.Unlock()

	// Record start event.
	state := WorkflowState{
		WorkflowID: opts.ID,
		RunID:      runID,
		Status:     StatusRunning,
		Input:      opts.Input,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		History: []HistoryEvent{
			{ID: 1, Type: EventWorkflowStarted, Timestamp: time.Now(), Input: opts.Input},
		},
	}
	if e.store != nil {
		e.store.Save(ctx, state)
	}

	if e.hooks.OnWorkflowStart != nil {
		e.hooks.OnWorkflowStart(ctx, opts.ID, opts.Input)
	}

	// Run the workflow function asynchronously.
	go func() {
		defer cancel()

		wfContext := &defaultWorkflowContext{
			Context:  wfCtx,
			executor: e,
			workflow: rw,
			wfID:     opts.ID,
		}

		result, err := fn(wfContext, opts.Input)

		e.mu.Lock()
		delete(e.running, opts.ID)
		e.mu.Unlock()

		if wfCtx.Err() != nil && err == nil {
			err = wfCtx.Err()
		}

		handle.mu.Lock()
		if err != nil {
			handle.status = StatusFailed
			handle.err = err
			if e.hooks.OnWorkflowFail != nil {
				e.hooks.OnWorkflowFail(ctx, opts.ID, err)
			}
		} else {
			handle.status = StatusCompleted
			handle.result = result
			if e.hooks.OnWorkflowComplete != nil {
				e.hooks.OnWorkflowComplete(ctx, opts.ID, result)
			}
		}
		handle.mu.Unlock()
		close(handle.done)

		// Persist final state.
		if e.store != nil {
			finalState := WorkflowState{
				WorkflowID: opts.ID,
				RunID:      runID,
				Status:     handle.Status(),
				Input:      opts.Input,
				Result:     result,
				UpdatedAt:  time.Now(),
			}
			if err != nil {
				finalState.Error = err.Error()
			}
			e.store.Save(ctx, finalState)
		}
	}()

	return handle, nil
}

// Signal sends a signal to a running workflow.
func (e *DefaultExecutor) Signal(ctx context.Context, workflowID string, signal Signal) error {
	e.mu.RLock()
	rw, ok := e.running[workflowID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("workflow/signal: workflow %q not found or not running", workflowID)
	}

	rw.mu.Lock()
	ch, exists := rw.signals[signal.Name]
	if !exists {
		ch = make(chan any, 10)
		rw.signals[signal.Name] = ch
	}
	rw.mu.Unlock()

	if e.hooks.OnSignal != nil {
		e.hooks.OnSignal(ctx, workflowID, signal)
	}

	ch <- signal.Payload
	return nil
}

// Query retrieves state from a running workflow. Currently returns the status.
func (e *DefaultExecutor) Query(ctx context.Context, workflowID string, queryType string) (any, error) {
	e.mu.RLock()
	rw, ok := e.running[workflowID]
	e.mu.RUnlock()

	if !ok {
		// Check the store for completed workflows.
		if e.store != nil {
			state, err := e.store.Load(ctx, workflowID)
			if err != nil {
				return nil, fmt.Errorf("workflow/query: %w", err)
			}
			if state == nil {
				return nil, fmt.Errorf("workflow/query: workflow %q not found", workflowID)
			}
			switch queryType {
			case "status":
				return state.Status, nil
			case "result":
				return state.Result, nil
			default:
				return nil, fmt.Errorf("workflow/query: unknown query type %q", queryType)
			}
		}
		return nil, fmt.Errorf("workflow/query: workflow %q not found", workflowID)
	}

	switch queryType {
	case "status":
		return rw.handle.Status(), nil
	default:
		return nil, fmt.Errorf("workflow/query: unknown query type %q", queryType)
	}
}

// Cancel requests cancellation of a running workflow.
func (e *DefaultExecutor) Cancel(_ context.Context, workflowID string) error {
	e.mu.RLock()
	rw, ok := e.running[workflowID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("workflow/cancel: workflow %q not found or not running", workflowID)
	}

	rw.cancel()

	rw.handle.mu.Lock()
	rw.handle.status = StatusCanceled
	rw.handle.mu.Unlock()

	return nil
}

// defaultHandle implements WorkflowHandle.
type defaultHandle struct {
	id     string
	runID  string
	status WorkflowStatus
	result any
	err    error
	done   chan struct{}
	mu     sync.RWMutex
}

func (h *defaultHandle) ID() string    { return h.id }
func (h *defaultHandle) RunID() string { return h.runID }

func (h *defaultHandle) Status() WorkflowStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status
}

func (h *defaultHandle) Result(ctx context.Context) (any, error) {
	select {
	case <-h.done:
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.result, h.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// defaultWorkflowContext implements WorkflowContext.
type defaultWorkflowContext struct {
	context.Context
	executor *DefaultExecutor
	workflow *runningWorkflow
	wfID     string
}

func (c *defaultWorkflowContext) ExecuteActivity(fn ActivityFunc, input any, opts ...ActivityOption) (any, error) {
	cfg := activityConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	actCtx := c.Context
	var cancel context.CancelFunc
	if cfg.timeout > 0 {
		actCtx, cancel = context.WithTimeout(c.Context, cfg.timeout)
		defer cancel()
	}

	if c.executor.hooks.OnActivityStart != nil {
		c.executor.hooks.OnActivityStart(c.Context, c.wfID, input)
	}

	var result any
	var actErr error

	if cfg.retryPolicy != nil {
		actErr = executeWithRetry(actCtx, *cfg.retryPolicy, func(ctx context.Context) error {
			var err error
			result, err = fn(ctx, input)
			if err != nil && c.executor.hooks.OnRetry != nil {
				c.executor.hooks.OnRetry(ctx, c.wfID, err)
			}
			return err
		})
	} else {
		result, actErr = fn(actCtx, input)
	}

	if actErr != nil {
		return nil, actErr
	}

	if c.executor.hooks.OnActivityComplete != nil {
		c.executor.hooks.OnActivityComplete(c.Context, c.wfID, result)
	}

	return result, nil
}

func (c *defaultWorkflowContext) ReceiveSignal(name string) <-chan any {
	c.workflow.mu.Lock()
	defer c.workflow.mu.Unlock()

	ch, exists := c.workflow.signals[name]
	if !exists {
		ch = make(chan any, 10)
		c.workflow.signals[name] = ch
	}
	return ch
}

func (c *defaultWorkflowContext) Sleep(d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-c.Context.Done():
		return c.Context.Err()
	}
}

// Compile-time interface checks.
var (
	_ DurableExecutor = (*DefaultExecutor)(nil)
	_ WorkflowHandle  = (*defaultHandle)(nil)
	_ WorkflowContext = (*defaultWorkflowContext)(nil)
)

func init() {
	Register("default", func(cfg Config) (DurableExecutor, error) {
		opts := []ExecutorOption{}
		if cfg.Store != nil {
			opts = append(opts, WithStore(cfg.Store))
		}
		return NewExecutor(opts...), nil
	})
}
