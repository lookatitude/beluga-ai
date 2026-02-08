package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/agent"
)

// A2AServer exposes a Beluga agent as an A2A remote agent via HTTP.
// It provides endpoints for the Agent Card, task creation, status, and cancellation.
type A2AServer struct {
	agent  agent.Agent
	card   AgentCard
	tasks  map[string]*Task
	cancel map[string]context.CancelFunc
	mu     sync.RWMutex
}

// NewServer creates a new A2A server wrapping the given agent and card.
func NewServer(a agent.Agent, card AgentCard) *A2AServer {
	return &A2AServer{
		agent:  a,
		card:   card,
		tasks:  make(map[string]*Task),
		cancel: make(map[string]context.CancelFunc),
	}
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
		Addr:    addr,
		Handler: s.Handler(),
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("a2a/serve: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		if shutdownErr := srv.Close(); shutdownErr != nil {
			return fmt.Errorf("a2a/serve: shutdown: %w", shutdownErr)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("a2a/serve: %w", err)
	}
}

func (s *A2AServer) handleCard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.card)
}

func (s *A2AServer) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Input == "" {
		writeJSONError(w, http.StatusBadRequest, "input is required")
		return
	}

	task := &Task{
		ID:       uuid.New().String(),
		Status:   StatusSubmitted,
		Input:    req.Input,
		Metadata: req.Metadata,
	}

	// Use background context since the task runs asynchronously beyond the
	// lifetime of the HTTP request.
	ctx, cancel := context.WithCancel(context.Background())

	// Take a snapshot for the response before the goroutine can modify the task.
	snapshot := *task

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.cancel[task.ID] = cancel
	s.mu.Unlock()

	// Run the agent asynchronously.
	go s.runTask(ctx, task)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(TaskResponse{Task: snapshot})
}

func (s *A2AServer) runTask(ctx context.Context, task *Task) {
	s.mu.Lock()
	task.Status = StatusWorking
	s.mu.Unlock()

	result, err := s.agent.Invoke(ctx, task.Input)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cancel, task.ID)

	if ctx.Err() != nil {
		task.Status = StatusCanceled
		return
	}

	if err != nil {
		task.Status = StatusFailed
		task.Error = err.Error()
		return
	}

	task.Status = StatusCompleted
	task.Output = result
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

	s.mu.RLock()
	task, ok := s.tasks[taskID]
	var snapshot Task
	if ok {
		snapshot = *task
	}
	s.mu.RUnlock()

	if !ok {
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TaskResponse{Task: snapshot})
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
	snapshot := *task
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TaskResponse{Task: snapshot})
}

func extractTaskID(path string) string {
	// path = /tasks/{id} or /tasks/{id}/...
	trimmed := strings.TrimPrefix(path, "/tasks/")
	return trimmed
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
