package mcp

import (
	"context"
	"sync"
	"time"

	"crypto/rand"
	"encoding/hex"

	"github.com/lookatitude/beluga-ai/core"
)

// AsyncStatus represents the current state of an asynchronous operation.
type AsyncStatus string

const (
	// AsyncStatusPending indicates the operation has been accepted but not started.
	AsyncStatusPending AsyncStatus = "pending"

	// AsyncStatusRunning indicates the operation is currently executing.
	AsyncStatusRunning AsyncStatus = "running"

	// AsyncStatusCompleted indicates the operation finished successfully.
	AsyncStatusCompleted AsyncStatus = "completed"

	// AsyncStatusFailed indicates the operation finished with an error.
	AsyncStatusFailed AsyncStatus = "failed"

	// AsyncStatusCancelled indicates the operation was cancelled before completion.
	AsyncStatusCancelled AsyncStatus = "cancelled"
)

// AsyncOperation represents the state and result of an asynchronous operation.
type AsyncOperation struct {
	// ID is the unique identifier for this operation.
	ID string `json:"id"`

	// Status is the current state of the operation.
	Status AsyncStatus `json:"status"`

	// Progress is an optional progress indicator (0.0 to 1.0).
	Progress float64 `json:"progress,omitempty"`

	// Result holds the operation result when Status is completed.
	Result any `json:"result,omitempty"`

	// Error holds the error message when Status is failed.
	Error string `json:"error,omitempty"`

	// CreatedAt is when the operation was created.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when the operation was last updated.
	UpdatedAt time.Time `json:"updatedAt"`
}

// AsyncHandler manages asynchronous operations. Implementations must be
// safe for concurrent use.
type AsyncHandler interface {
	// Start begins an asynchronous operation and returns its ID.
	// The provided function runs asynchronously; its result or error is
	// stored in the operation.
	Start(ctx context.Context, fn func(ctx context.Context) (any, error)) (string, error)

	// Poll returns the current state of an asynchronous operation.
	Poll(ctx context.Context, id string) (*AsyncOperation, error)

	// Cancel requests cancellation of a running operation. It returns an
	// error if the operation does not exist or cannot be cancelled.
	Cancel(ctx context.Context, id string) error
}

// asyncEntry holds a running operation's state and a cancel signal channel.
// Cancellation is delivered to the running goroutine via cancelCh; the
// goroutine owns its context and cancel function locally (always deferred)
// to satisfy static analysis that cancel functions are invoked.
type asyncEntry struct {
	op       AsyncOperation
	cancelCh chan struct{}
	// cancelOnce guards cancelCh against double close.
	cancelOnce sync.Once
}

// signalCancel closes the cancel channel at most once.
func (e *asyncEntry) signalCancel() {
	e.cancelOnce.Do(func() { close(e.cancelCh) })
}

// InMemoryAsyncHandler is an in-memory implementation of AsyncHandler suitable
// for single-process deployments and testing.
type InMemoryAsyncHandler struct {
	mu      sync.Mutex
	ops     map[string]*asyncEntry
	maxOps  int
	timeout time.Duration
}

// Compile-time interface check.
var _ AsyncHandler = (*InMemoryAsyncHandler)(nil)

// AsyncOption configures an InMemoryAsyncHandler.
type AsyncOption func(*InMemoryAsyncHandler)

// WithMaxOps sets the maximum number of tracked operations. When exceeded,
// Start returns an error.
func WithMaxOps(n int) AsyncOption {
	return func(h *InMemoryAsyncHandler) {
		h.maxOps = n
	}
}

// WithAsyncTimeout sets the default timeout for async operations.
func WithAsyncTimeout(d time.Duration) AsyncOption {
	return func(h *InMemoryAsyncHandler) {
		h.timeout = d
	}
}

// NewInMemoryAsyncHandler creates a new in-memory async handler.
func NewInMemoryAsyncHandler(opts ...AsyncOption) *InMemoryAsyncHandler {
	h := &InMemoryAsyncHandler{
		ops:     make(map[string]*asyncEntry),
		maxOps:  1000,
		timeout: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Start begins an asynchronous operation. The provided function fn runs in a
// separate goroutine with a context that is cancelled when Cancel is called or
// the timeout is reached.
func (h *InMemoryAsyncHandler) Start(ctx context.Context, fn func(ctx context.Context) (any, error)) (string, error) {
	id, err := generateOpID()
	if err != nil {
		return "", core.Errorf(core.ErrProviderDown, "mcp/async: generate ID: %w", err)
	}

	h.mu.Lock()
	if len(h.ops) >= h.maxOps {
		h.mu.Unlock()
		return "", core.Errorf(core.ErrInvalidInput, "mcp/async: maximum operations (%d) exceeded", h.maxOps)
	}

	now := time.Now()
	entry := &asyncEntry{
		op: AsyncOperation{
			ID:        id,
			Status:    AsyncStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		},
		cancelCh: make(chan struct{}),
	}
	h.ops[id] = entry
	h.mu.Unlock()

	go h.run(ctx, id, entry, fn)

	return id, nil
}

// run executes the async function and updates the operation state. The
// per-operation context (with timeout) is owned entirely by this goroutine,
// and its cancel function is always invoked via defer. External cancellation
// is delivered via entry.cancelCh and a small watchdog goroutine that cancels
// the local context when the signal fires.
func (h *InMemoryAsyncHandler) run(parentCtx context.Context, id string, entry *asyncEntry, fn func(ctx context.Context) (any, error)) {
	opCtx, cancel := context.WithTimeout(parentCtx, h.timeout)
	defer cancel()

	// Watchdog: propagate external cancel signal into opCtx cancellation.
	watchdogDone := make(chan struct{})
	go func() {
		defer close(watchdogDone)
		select {
		case <-entry.cancelCh:
			cancel()
		case <-opCtx.Done():
		}
	}()
	defer func() {
		// Ensure watchdog exits before run returns.
		cancel()
		<-watchdogDone
	}()

	h.mu.Lock()
	if e, ok := h.ops[id]; ok {
		e.op.Status = AsyncStatusRunning
		e.op.UpdatedAt = time.Now()
	}
	h.mu.Unlock()

	result, err := fn(opCtx)

	h.mu.Lock()
	defer h.mu.Unlock()

	e, ok := h.ops[id]
	if !ok {
		return
	}

	e.op.UpdatedAt = time.Now()

	// If already cancelled, don't overwrite status.
	if e.op.Status == AsyncStatusCancelled {
		return
	}

	if err != nil {
		e.op.Status = AsyncStatusFailed
		e.op.Error = err.Error()
	} else {
		e.op.Status = AsyncStatusCompleted
		e.op.Result = result
		e.op.Progress = 1.0
	}
}

// Poll returns a snapshot of the current operation state.
func (h *InMemoryAsyncHandler) Poll(_ context.Context, id string) (*AsyncOperation, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	e, ok := h.ops[id]
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "mcp/async: operation %q not found", id)
	}

	// Return a copy to avoid races.
	op := e.op
	return &op, nil
}

// Cancel requests cancellation of a running or pending operation.
func (h *InMemoryAsyncHandler) Cancel(_ context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	e, ok := h.ops[id]
	if !ok {
		return core.Errorf(core.ErrNotFound, "mcp/async: operation %q not found", id)
	}

	switch e.op.Status {
	case AsyncStatusCompleted, AsyncStatusFailed, AsyncStatusCancelled:
		return core.Errorf(core.ErrInvalidInput, "mcp/async: operation %q is already in terminal state %q", id, e.op.Status)
	}

	e.op.Status = AsyncStatusCancelled
	e.op.UpdatedAt = time.Now()
	e.signalCancel()

	return nil
}

// generateOpID creates a cryptographically random operation ID.
func generateOpID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
