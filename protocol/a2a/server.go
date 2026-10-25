package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/protocol/internal/httpsafe"
)

const (
	contentTypeHeader = "Content-Type"
	contentTypeJSON   = "application/json"
)

// Default limits for the legacy A2A server. See ServerOption.
const (
	// DefaultTaskTTL is how long finished (completed, failed or canceled)
	// tasks are retained for polling before they are evicted.
	DefaultTaskTTL = 15 * time.Minute
	// DefaultMaxConcurrentTasks bounds the number of agent invocations
	// running at once. Further task creations get HTTP 503.
	DefaultMaxConcurrentTasks = 64
	// DefaultTaskTimeout bounds the lifetime of a single task's agent
	// invocation.
	DefaultTaskTimeout = 10 * time.Minute
	// defaultWriteTimeout bounds response writes. The legacy server never
	// streams, so a write deadline is safe here.
	defaultWriteTimeout = 30 * time.Second
)

// A2AServer exposes a Beluga agent as a remote agent over a legacy,
// Beluga-specific REST shape (POST /tasks, GET /tasks/{id}, …).
//
// It is NOT wire-compatible with the A2A protocol specification (which uses
// JSON-RPC methods such as message/send and tasks/get) and only
// interoperates with this package's A2AClient. For spec-compliant A2A use
// protocol/a2a/sdk.
//
// Finished tasks are evicted after a TTL (DefaultTaskTTL), concurrent task
// executions are bounded (DefaultMaxConcurrentTasks), each execution is
// bounded by a timeout (DefaultTaskTimeout), and request bodies are limited
// to 1 MiB. The server performs no authentication; wrap Handler with your
// own auth middleware.
type A2AServer struct { //nolint:revive // exported name is public API; renaming to remove the stutter would break callers
	agent  agent.Agent
	card   AgentCard
	tasks  map[string]*Task
	cancel map[string]context.CancelFunc
	mu     sync.RWMutex

	// finished records when each task reached a terminal state, for TTL
	// eviction. Guarded by mu.
	finished    map[string]time.Time
	sem         chan struct{}
	taskTTL     time.Duration
	taskTimeout time.Duration
	now         func() time.Time
}

// ServerOption configures an A2AServer.
type ServerOption func(*A2AServer)

// WithTaskTTL sets how long finished tasks are retained before eviction.
// Non-positive values are ignored (DefaultTaskTTL applies).
func WithTaskTTL(d time.Duration) ServerOption {
	return func(s *A2AServer) {
		if d > 0 {
			s.taskTTL = d
		}
	}
}

// WithMaxConcurrentTasks bounds concurrently running tasks. When the limit
// is reached, POST /tasks returns HTTP 503. Non-positive values are ignored
// (DefaultMaxConcurrentTasks applies).
func WithMaxConcurrentTasks(n int) ServerOption {
	return func(s *A2AServer) {
		if n > 0 {
			s.sem = make(chan struct{}, n)
		}
	}
}

// WithTaskTimeout bounds each task's agent invocation. A task that exceeds
// it is marked failed. Non-positive values are ignored (DefaultTaskTimeout
// applies).
func WithTaskTimeout(d time.Duration) ServerOption {
	return func(s *A2AServer) {
		if d > 0 {
			s.taskTimeout = d
		}
	}
}

// NewServer creates a new A2A server wrapping the given agent and card.
func NewServer(a agent.Agent, card AgentCard, opts ...ServerOption) *A2AServer {
	s := &A2AServer{
		agent:       a,
		card:        card,
		tasks:       make(map[string]*Task),
		cancel:      make(map[string]context.CancelFunc),
		finished:    make(map[string]time.Time),
		sem:         make(chan struct{}, DefaultMaxConcurrentTasks),
		taskTTL:     DefaultTaskTTL,
		taskTimeout: DefaultTaskTimeout,
		now:         time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Handler returns an http.Handler for the A2A server.
func (s *A2AServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/agent.json", s.handleCard)
	mux.HandleFunc("POST /tasks", s.handleCreateTask)
	mux.HandleFunc("GET /tasks/", s.handleGetTask)
	mux.HandleFunc("POST /tasks/", s.handleTaskAction)
	return mux
}

// Serve starts the A2A server on the given address. It blocks until the context
// is canceled or an error occurs.
func (s *A2AServer) Serve(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: httpsafe.DefaultReadHeaderTimeout,
		ReadTimeout:       httpsafe.DefaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       httpsafe.DefaultIdleTimeout,
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return core.Errorf(core.ErrProviderDown, "a2a/serve: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		if shutdownErr := srv.Close(); shutdownErr != nil {
			return core.Errorf(core.ErrProviderDown, "a2a/serve: shutdown: %w", shutdownErr)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return core.Errorf(core.ErrProviderDown, "a2a/serve: %w", err)
	}
}

func (s *A2AServer) handleCard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(s.card)
}

func (s *A2AServer) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, httpsafe.DefaultMaxBodyBytes)).Decode(&req); err != nil {
		if httpsafe.IsBodyTooLarge(err) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Input == "" {
		writeJSONError(w, http.StatusBadRequest, "input is required")
		return
	}

	// Bound concurrent executions: reject instead of queueing goroutines.
	select {
	case s.sem <- struct{}{}:
	default:
		w.Header().Set("Retry-After", "1")
		writeJSONError(w, http.StatusServiceUnavailable, "too many concurrent tasks")
		return
	}

	task := &Task{
		ID:       uuid.New().String(),
		Status:   StatusSubmitted,
		Input:    req.Input,
		Metadata: req.Metadata,
	}

	// The task outlives the HTTP request, so detach from the request's
	// cancellation (keeping its values, e.g. tenant/trace) and bound it with
	// the task timeout. The cancel func is stored in s.cancel for
	// POST /tasks/{id}/cancel and is always called by runTask on exit.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), s.taskTimeout)

	// Take a snapshot for the response before the goroutine can modify the task.
	snapshot := *task

	s.mu.Lock()
	s.evictExpiredLocked()
	s.tasks[task.ID] = task
	s.cancel[task.ID] = cancel
	s.mu.Unlock()

	// Run the agent asynchronously.
	go s.runTask(ctx, cancel, task)

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(TaskResponse{Task: snapshot})
}

func (s *A2AServer) runTask(ctx context.Context, cancel context.CancelFunc, task *Task) {
	defer func() { <-s.sem }()
	defer cancel()

	s.mu.Lock()
	if task.Status == StatusSubmitted {
		task.Status = StatusWorking
	}
	s.mu.Unlock()

	result, err := s.agent.Invoke(ctx, task.Input)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cancel, task.ID)
	s.finished[task.ID] = s.now()

	switch {
	case task.Status == StatusCanceled:
		// Canceled via the API; keep that state.
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		task.Status = StatusFailed
		task.Error = "task timed out"
	case ctx.Err() != nil:
		task.Status = StatusCanceled
	case err != nil:
		reqID := httpsafe.NewRequestID()
		slog.Error("a2a: task failed",
			"request_id", reqID,
			"task_id", task.ID,
			"err", httpsafe.LogValue(err.Error()))
		task.Status = StatusFailed
		// Never expose internal error text to remote callers.
		task.Error = httpsafe.GenericInternalError + " (request_id=" + reqID + ")"
	default:
		task.Status = StatusCompleted
		task.Output = result
	}
}

// evictExpiredLocked removes finished tasks older than the TTL. The caller
// must hold s.mu for writing.
func (s *A2AServer) evictExpiredLocked() {
	now := s.now()
	for id, at := range s.finished {
		if now.Sub(at) >= s.taskTTL {
			delete(s.finished, id)
			delete(s.tasks, id)
			delete(s.cancel, id)
		}
	}
}

func (s *A2AServer) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskID(r.URL.Path)
	if taskID == "" {
		writeJSONError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	// Reject paths with extra segments (e.g., /tasks/{id}/cancel via GET).
	if strings.Contains(taskID, "/") {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	s.mu.Lock()
	s.evictExpiredLocked()
	task, ok := s.tasks[taskID]
	var snapshot Task
	if ok {
		snapshot = *task
	}
	s.mu.Unlock()

	if !ok {
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(TaskResponse{Task: snapshot})
}

func (s *A2AServer) handleTaskAction(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// Expect /tasks/{id}/cancel
	parts := strings.Split(strings.TrimPrefix(path, "/tasks/"), "/")
	if len(parts) != 2 || parts[1] != "cancel" {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	taskID := parts[0]

	s.mu.Lock()
	task, ok := s.tasks[taskID]
	if !ok {
		s.mu.Unlock()
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}

	cancelFn, hasCancel := s.cancel[taskID]
	if hasCancel {
		cancelFn()
		delete(s.cancel, taskID)
	}

	task.Status = StatusCanceled
	if !hasCancel {
		// Already finished (or never started): start its TTL clock if
		// runTask has not.
		if _, done := s.finished[taskID]; !done {
			s.finished[taskID] = s.now()
		}
	}
	snapshot := *task
	s.mu.Unlock()

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(TaskResponse{Task: snapshot})
}

func extractTaskID(path string) string {
	// path = /tasks/{id} or /tasks/{id}/...
	trimmed := strings.TrimPrefix(path, "/tasks/")
	return trimmed
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
