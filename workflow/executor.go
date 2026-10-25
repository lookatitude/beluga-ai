package workflow

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
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

// DefaultExecutor is a goroutine-based executor that runs workflows
// in-process. It records each workflow's execution history (start, activity
// and terminal events) and, when a WorkflowStore is configured, saves that
// history with the start and final state.
//
// DefaultExecutor does not replay or resume workflows: a workflow that is
// running when the process exits is not restarted, and the stored state
// only describes what happened. Use a durable engine provider (for example
// workflow/providers/temporal) when executions must survive restarts.
type DefaultExecutor struct {
	store   WorkflowStore
	hooks   Hooks
	running map[string]*runningWorkflow
	mu      sync.RWMutex
}

type runningWorkflow struct {
	handle  *defaultHandle
	cancel  context.CancelFunc
	signals map[string]chan any
	mu      sync.Mutex

	// canceled is set by Cancel so the run finishes as StatusCanceled.
	canceled atomic.Bool

	// history and createdAt make up the recorded execution history;
	// guarded by histMu.
	histMu    sync.Mutex
	history   []HistoryEvent
	createdAt time.Time
}

// record appends an event to the workflow history, assigning its ID.
func (rw *runningWorkflow) record(ev HistoryEvent) {
	rw.histMu.Lock()
	defer rw.histMu.Unlock()
	ev.ID = len(rw.history) + 1
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}
	rw.history = append(rw.history, ev)
}

// snapshotHistory returns a copy of the recorded history.
func (rw *runningWorkflow) snapshotHistory() []HistoryEvent {
	rw.histMu.Lock()
	defer rw.histMu.Unlock()
	out := make([]HistoryEvent, len(rw.history))
	copy(out, rw.history)
	return out
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

	// Create the workflow context. The cancel function is stored on the
	// runningWorkflow and invoked when the workflow completes, times out,
	// or is cancelled via CancelWorkflow.
	//
	//nolint:gosec // G601: cancel is stored in rw.cancel and invoked in runWorkflow's
	// deferred finalize path; gosec cannot trace storage through a struct field.
	wfCtx, cancel := newWorkflowContext(ctx, opts.Timeout)

	now := time.Now()
	rw := &runningWorkflow{
		handle:    handle,
		cancel:    cancel,
		signals:   make(map[string]chan any),
		createdAt: now,
	}
	rw.record(HistoryEvent{Type: EventWorkflowStarted, Timestamp: now, Input: opts.Input})

	e.mu.Lock()
	e.running[opts.ID] = rw
	e.mu.Unlock()

	// Record start event.
	state := WorkflowState{
		WorkflowID: opts.ID,
		RunID:      runID,
		Status:     StatusRunning,
		Input:      opts.Input,
		CreatedAt:  now,
		UpdatedAt:  now,
		History:    rw.snapshotHistory(),
	}
	if e.store != nil {
		_ = e.store.Save(ctx, state)
	}

	if e.hooks.OnWorkflowStart != nil {
		e.hooks.OnWorkflowStart(ctx, opts.ID, opts.Input)
	}

	// Run the workflow function asynchronously.
	go e.runWorkflow(ctx, wfCtx, runWorkflowParams{
		cancel: cancel,
		fn:     fn,
		rw:     rw,
		handle: handle,
		opts:   opts,
		runID:  runID,
	})

	return handle, nil
}

// runWorkflowParams groups the parameters for runWorkflow.
type runWorkflowParams struct {
	cancel context.CancelFunc
	fn     WorkflowFunc
	rw     *runningWorkflow
	handle *defaultHandle
	opts   WorkflowOptions
	runID  string
}

// runWorkflow is the goroutine body for a workflow execution.
func (e *DefaultExecutor) runWorkflow(parentCtx, wfCtx context.Context, p runWorkflowParams) {
	defer p.cancel()

	wfContext := &defaultWorkflowContext{
		Context:  wfCtx,
		executor: e,
		workflow: p.rw,
		wfID:     p.opts.ID,
	}

	result, err := p.fn(wfContext, p.opts.Input)

	e.mu.Lock()
	delete(e.running, p.opts.ID)
	e.mu.Unlock()

	if wfCtx.Err() != nil && err == nil {
		err = wfCtx.Err()
	}

	canceled := p.rw.canceled.Load()
	if canceled {
		err = core.Errorf(core.ErrCanceled, "workflow/cancel: workflow %q was canceled: %w", p.opts.ID, context.Canceled)
	}

	e.finalizeHandle(parentCtx, p.handle, p.opts.ID, result, err, canceled)

	switch {
	case canceled:
		p.rw.record(HistoryEvent{Type: EventWorkflowCanceled, Error: err.Error()})
	case err != nil:
		p.rw.record(HistoryEvent{Type: EventWorkflowFailed, Error: err.Error()})
	default:
		p.rw.record(HistoryEvent{Type: EventWorkflowCompleted, Result: result})
	}

	e.persistFinalState(parentCtx, p, result, err)
}

// finalizeHandle updates the handle status, result, and error, then signals
// completion. A workflow canceled through Cancel ends as StatusCanceled and
// does not fire OnWorkflowFail.
func (e *DefaultExecutor) finalizeHandle(ctx context.Context, handle *defaultHandle, wfID string, result any, err error, canceled bool) {
	handle.mu.Lock()
	if canceled {
		handle.status = StatusCanceled
		handle.err = err
	} else if err != nil {
		handle.status = StatusFailed
		handle.err = err
		if e.hooks.OnWorkflowFail != nil {
			e.hooks.OnWorkflowFail(ctx, wfID, err)
		}
	} else {
		handle.status = StatusCompleted
		handle.result = result
		if e.hooks.OnWorkflowComplete != nil {
			e.hooks.OnWorkflowComplete(ctx, wfID, result)
		}
	}
	handle.mu.Unlock()
	close(handle.done)
}

// persistFinalState saves the final workflow state, including the recorded
// history and the original creation time, to the store if configured.
func (e *DefaultExecutor) persistFinalState(ctx context.Context, p runWorkflowParams, result any, err error) {
	if e.store == nil {
		return
	}
	finalState := WorkflowState{
		WorkflowID: p.opts.ID,
		RunID:      p.runID,
		Status:     p.handle.Status(),
		Input:      p.opts.Input,
		Result:     result,
		History:    p.rw.snapshotHistory(),
		CreatedAt:  p.rw.createdAt,
		UpdatedAt:  time.Now(),
	}
	if err != nil {
		finalState.Error = err.Error()
	}
	_ = e.store.Save(ctx, finalState)
}

// Signal sends a signal to a running workflow. Each signal name buffers up
// to 10 undelivered payloads; when the buffer is full Signal waits until the
// workflow receives one, ctx is done (returning ctx's error), or the
// workflow finishes (returning a not_found error).
func (e *DefaultExecutor) Signal(ctx context.Context, workflowID string, signal Signal) error {
	e.mu.RLock()
	rw, ok := e.running[workflowID]
	e.mu.RUnlock()

	if !ok {
		return core.Errorf(core.ErrNotFound, "workflow/signal: workflow %q not found or not running", workflowID)
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

	select {
	case ch <- signal.Payload:
		rw.record(HistoryEvent{Type: EventSignalReceived, ActivityName: signal.Name, Input: signal.Payload})
		return nil
	case <-ctx.Done():
		return core.Wrapf(core.ErrTimeout, "workflow/signal: %w", ctx.Err())
	case <-rw.handle.done:
		return core.Errorf(core.ErrNotFound, "workflow/signal: workflow %q finished before the signal was delivered", workflowID)
	}
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
				return nil, core.Wrapf(core.ErrProviderDown, "workflow/query: %w", err)
			}
			if state == nil {
				return nil, core.Errorf(core.ErrNotFound, "workflow/query: workflow %q not found", workflowID)
			}
			switch queryType {
			case "status":
				return state.Status, nil
			case "result":
				return state.Result, nil
			default:
				return nil, core.Errorf(core.ErrInvalidInput, "workflow/query: unknown query type %q", queryType)
			}
		}
		return nil, core.Errorf(core.ErrNotFound, "workflow/query: workflow %q not found", workflowID)
	}

	switch queryType {
	case "status":
		return rw.handle.Status(), nil
	default:
		return nil, core.Errorf(core.ErrInvalidInput, "workflow/query: unknown query type %q", queryType)
	}
}

// Cancel requests cancellation of a running workflow. The workflow's context
// is canceled and, once the workflow function returns, the handle reports
// StatusCanceled and Result returns an ErrCanceled error. OnWorkflowFail is
// not called for a canceled workflow.
func (e *DefaultExecutor) Cancel(_ context.Context, workflowID string) error {
	e.mu.RLock()
	rw, ok := e.running[workflowID]
	e.mu.RUnlock()

	if !ok {
		return core.Errorf(core.ErrNotFound, "workflow/cancel: workflow %q not found or not running", workflowID)
	}

	rw.canceled.Store(true)
	rw.cancel()

	rw.handle.mu.Lock()
	if rw.handle.status == StatusRunning {
		rw.handle.status = StatusCanceled
	}
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

// newWorkflowContext returns a workflow context and its cancel function. If
// timeout > 0 the context has a deadline; otherwise it is a plain cancellable
// child of the parent. The returned cancel MUST be stored and invoked by the
// caller when the workflow finishes or is cancelled.
func newWorkflowContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(parent, timeout) // #nosec G118 -- cancel returned to caller, stored in rw.cancel
	}
	return context.WithCancel(parent) // #nosec G118 -- cancel returned to caller, stored in rw.cancel
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
	c.workflow.record(HistoryEvent{Type: EventActivityStarted, Input: input})

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
		c.workflow.record(HistoryEvent{Type: EventActivityFailed, Input: input, Error: actErr.Error()})
		return nil, actErr
	}
	c.workflow.record(HistoryEvent{Type: EventActivityCompleted, Input: input, Result: result})

	if c.executor.hooks.OnActivityComplete != nil {
		c.executor.hooks.OnActivityComplete(c.Context, c.wfID, result)
	}

	return result, nil
}

func (c *defaultWorkflowContext) ReceiveSignal(name string) iter.Seq2[any, error] {
	// Eagerly create/lookup the shared channel so that a Signal() delivered
	// between ReceiveSignal() returning and the caller iterating is still
	// buffered and not lost.
	c.workflow.mu.Lock()
	ch, exists := c.workflow.signals[name]
	if !exists {
		ch = make(chan any, 10)
		c.workflow.signals[name] = ch
	}
	c.workflow.mu.Unlock()

	ctx := c.Context
	return func(yield func(any, error) bool) {
		for {
			select {
			case payload := <-ch:
				if !yield(payload, nil) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func (c *defaultWorkflowContext) Sleep(d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-c.Done():
		return c.Err()
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
