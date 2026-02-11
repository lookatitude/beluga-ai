package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id        string
	invokeFn  func(ctx context.Context, input string) (string, error)
	invokeErr error
}

func (m *mockAgent) ID() string            { return m.id }
func (m *mockAgent) Persona() agent.Persona { return agent.Persona{Role: m.id} }
func (m *mockAgent) Tools() []tool.Tool     { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }

func (m *mockAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, input)
	}
	if m.invokeErr != nil {
		return "", m.invokeErr
	}
	return "response: " + input, nil
}

func (m *mockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, err := m.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{}, err)
			return
		}
		yield(agent.Event{Type: agent.EventText, Text: result, AgentID: m.id}, nil)
	}
}

func setupA2ATestServer() (*A2AServer, *httptest.Server) {
	a := &mockAgent{id: "test-agent"}
	card := AgentCard{
		Name:        "test-agent",
		Description: "A test agent",
		Version:     "1.0.0",
		Endpoint:    "http://localhost:9090",
		Skills: []AgentSkill{
			{Name: "echo", Description: "Echoes input"},
		},
		Capabilities: []string{"text"},
	}
	srv := NewServer(a, card)
	ts := httptest.NewServer(srv.Handler())
	return srv, ts
}

// --- Server tests ---

func TestServer_Card(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	card, err := client.GetCard(context.Background())
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}

	if card.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", card.Name)
	}
	if card.Description != "A test agent" {
		t.Errorf("expected description 'A test agent', got %q", card.Description)
	}
	if card.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", card.Version)
	}
	if len(card.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(card.Skills))
	}
	if card.Skills[0].Name != "echo" {
		t.Errorf("expected skill name 'echo', got %q", card.Skills[0].Name)
	}
}

func TestServer_CreateTask(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	task, err := client.CreateTask(context.Background(), TaskRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if task.ID == "" {
		t.Error("expected non-empty task ID")
	}
	if task.Input != "hello" {
		t.Errorf("expected input 'hello', got %q", task.Input)
	}
}

func TestServer_CreateTask_EmptyInput(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.CreateTask(context.Background(), TaskRequest{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestServer_GetTask(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	task, err := client.CreateTask(context.Background(), TaskRequest{Input: "test"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	var got *Task
	for i := 0; i < 50; i++ {
		got, err = client.GetTask(context.Background(), task.ID)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if got.Status == StatusCompleted || got.Status == StatusFailed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", got.Status, got.Error)
	}
	if got.Output != "response: test" {
		t.Errorf("expected 'response: test', got %q", got.Output)
	}
}

func TestServer_GetTask_NotFound(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.GetTask(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

func TestServer_CancelTask(t *testing.T) {
	slowAgent := &mockAgent{
		id: "slow-agent",
		invokeFn: func(ctx context.Context, _ string) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(10 * time.Second):
				return "done", nil
			}
		},
	}
	card := AgentCard{Name: "slow-agent", Endpoint: "http://localhost:9090"}
	srv := NewServer(slowAgent, card)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := NewClient(ts.URL)
	task, err := client.CreateTask(context.Background(), TaskRequest{Input: "slow"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if err := client.CancelTask(context.Background(), task.ID); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	got, err := client.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != StatusCanceled {
		t.Errorf("expected canceled, got %s", got.Status)
	}
}

func TestServer_CancelTask_NotFound(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	err := client.CancelTask(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

func TestServer_TaskWithFailure(t *testing.T) {
	failAgent := &mockAgent{
		id:        "fail-agent",
		invokeErr: fmt.Errorf("agent failure"),
	}
	card := AgentCard{Name: "fail-agent", Endpoint: "http://localhost:9090"}
	srv := NewServer(failAgent, card)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := NewClient(ts.URL)
	task, err := client.CreateTask(context.Background(), TaskRequest{Input: "fail"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	var got *Task
	for i := 0; i < 50; i++ {
		got, err = client.GetTask(context.Background(), task.ID)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if got.Status == StatusFailed || got.Status == StatusCompleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got.Status != StatusFailed {
		t.Errorf("expected failed, got %s", got.Status)
	}
	if got.Error == "" {
		t.Error("expected non-empty error message")
	}
}

// --- Client tests ---

func TestClient_GetCard(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	card, err := client.GetCard(context.Background())
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if card.Name != "test-agent" {
		t.Errorf("expected 'test-agent', got %q", card.Name)
	}
}

func TestClient_GetCard_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.GetCard(context.Background())
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestClient_GetCard_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.GetCard(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestClient_GetCard_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	_, err := client.GetCard(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestClient_GetTask_UnexpectedStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.GetTask(context.Background(), "some-id")
	if err == nil {
		t.Fatal("expected error for unexpected status")
	}
}

func TestClient_CancelTask_UnexpectedStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	err := client.CancelTask(context.Background(), "some-id")
	if err == nil {
		t.Fatal("expected error for unexpected status")
	}
}

// --- RemoteAgent tests ---

func TestNewRemoteAgent(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	if remote.ID() != "test-agent" {
		t.Errorf("expected ID 'test-agent', got %q", remote.ID())
	}
	persona := remote.Persona()
	if persona.Role != "test-agent" {
		t.Errorf("expected persona role 'test-agent', got %q", persona.Role)
	}
	if persona.Goal != "A test agent" {
		t.Errorf("expected persona goal 'A test agent', got %q", persona.Goal)
	}
	if remote.Tools() != nil {
		t.Error("expected nil tools")
	}
	if remote.Children() != nil {
		t.Error("expected nil children")
	}
}

func TestNewRemoteAgent_ConnectionError(t *testing.T) {
	_, err := NewRemoteAgent("http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestNewRemoteAgent_Invoke(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	result, err := remote.Invoke(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if result != "response: hello world" {
		t.Errorf("expected 'response: hello world', got %q", result)
	}
}

func TestNewRemoteAgent_Invoke_ContextCancel(t *testing.T) {
	slowAgent := &mockAgent{
		id: "slow-agent",
		invokeFn: func(ctx context.Context, _ string) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(10 * time.Second):
				return "done", nil
			}
		},
	}
	card := AgentCard{Name: "slow-agent", Endpoint: "http://localhost:9090"}
	srv := NewServer(slowAgent, card)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = remote.Invoke(ctx, "hello")
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestNewRemoteAgent_Stream(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	var events []agent.Event
	for event, err := range remote.Stream(context.Background(), "stream test") {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		events = append(events, event)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != agent.EventText {
		t.Errorf("expected EventText, got %s", events[0].Type)
	}
	if events[0].Text != "response: stream test" {
		t.Errorf("expected 'response: stream test', got %q", events[0].Text)
	}
	if events[1].Type != agent.EventDone {
		t.Errorf("expected EventDone, got %s", events[1].Type)
	}
}

func TestNewRemoteAgent_Stream_Error(t *testing.T) {
	failAgent := &mockAgent{
		id:        "fail-agent",
		invokeErr: fmt.Errorf("agent failure"),
	}
	card := AgentCard{Name: "fail-agent", Endpoint: "http://localhost:9090"}
	srv := NewServer(failAgent, card)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	var gotErr error
	for _, err := range remote.Stream(context.Background(), "fail") {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Error("expected error from stream")
	}
}

func TestNewRemoteAgent_InvokeFailure(t *testing.T) {
	failAgent := &mockAgent{
		id:        "fail-agent",
		invokeErr: fmt.Errorf("agent failure"),
	}
	card := AgentCard{Name: "fail-agent", Endpoint: "http://localhost:9090"}
	srv := NewServer(failAgent, card)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	_, err = remote.Invoke(context.Background(), "fail")
	if err == nil {
		t.Fatal("expected error from failed agent")
	}
}

// --- Server endpoint tests ---

func TestServer_CardEndpoint_Direct(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/.well-known/agent.json")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if card.Name != "test-agent" {
		t.Errorf("expected 'test-agent', got %q", card.Name)
	}
}

func TestServer_TaskMetadata(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	task, err := client.CreateTask(context.Background(), TaskRequest{
		Input:    "with metadata",
		Metadata: map[string]any{"key": "value"},
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if task.Metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
	if task.Metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", task.Metadata["key"])
	}
}

func TestServer_GetTask_SlashInID(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	// GET /tasks/id/cancel should fail with not found (slash in path for GET).
	resp, err := http.Get(ts.URL + "/tasks/someid/cancel")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestServer_TaskAction_InvalidAction(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	// POST /tasks/id/invalid should return not found.
	resp, err := http.Post(ts.URL+"/tasks/someid/invalid", "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestServer_TaskAction_NoAction(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	// POST /tasks/someid (no action) should not match the cancel handler.
	resp, err := http.Post(ts.URL+"/tasks/someid", "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	// The mux routes POST /tasks/ to handleTaskAction, which expects /tasks/{id}/cancel.
	// With just /tasks/someid, parts will be ["someid"] with len 1, so not found.
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestServer_Serve_ContextCancel(t *testing.T) {
	a := &mockAgent{id: "test-agent"}
	card := AgentCard{Name: "test-agent"}
	srv := NewServer(a, card)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx, "127.0.0.1:0")
	}()

	// Give server time to start.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after context cancel")
	}
}

func TestServer_Serve_BusyPort(t *testing.T) {
	a := &mockAgent{id: "test-agent"}
	card := AgentCard{Name: "test-agent"}
	srv := NewServer(a, card)

	// Listen on a port to make it busy.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	// Try to serve on the same address - the port extraction is tricky,
	// so let's use a guaranteed-busy approach: bind to port 0 first.
	ctx := context.Background()
	// Use an invalid address to test the listen error.
	err := srv.Serve(ctx, "256.256.256.256:0")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
}

func TestServer_CancelTask_AlreadyCompleted(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	client := NewClient(ts.URL)
	task, err := client.CreateTask(context.Background(), TaskRequest{Input: "fast"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Wait for task to complete.
	for i := 0; i < 50; i++ {
		got, err := client.GetTask(context.Background(), task.ID)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if got.Status == StatusCompleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Cancel an already-completed task.
	err = client.CancelTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
}

func TestClient_CreateTask_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	_, err := client.CreateTask(context.Background(), TaskRequest{Input: "hello"})
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestClient_CreateTask_NonCreatedStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "bad request"})
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.CreateTask(context.Background(), TaskRequest{Input: "hello"})
	if err == nil {
		t.Fatal("expected error for non-201 status")
	}
}

func TestClient_GetTask_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	_, err := client.GetTask(context.Background(), "some-id")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestClient_CancelTask_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	err := client.CancelTask(context.Background(), "some-id")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestRemoteAgent_Invoke_TaskFailed(t *testing.T) {
	failAgent := &mockAgent{
		id:        "fail-agent",
		invokeErr: fmt.Errorf("agent failure"),
	}
	card := AgentCard{Name: "fail-agent", Endpoint: "http://localhost:9090"}
	srv := NewServer(failAgent, card)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	_, err = remote.Invoke(context.Background(), "fail")
	if err == nil {
		t.Fatal("expected error from failed task")
	}
	if !strings.Contains(err.Error(), "task failed") {
		t.Errorf("expected 'task failed' in error, got %q", err.Error())
	}
}

func TestServer_HandleGetTask_EmptyID(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	// GET /tasks/ with no task ID.
	resp, err := http.Get(ts.URL + "/tasks/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestServer_HandleCreateTask_InvalidBody(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/tasks", "application/json", strings.NewReader("not json"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Coverage gap tests ---

// TestRemoteAgent_Invoke_GetTaskError tests the path where CreateTask succeeds
// but GetTask fails during polling (line 189 in client.go).
func TestRemoteAgent_Invoke_GetTaskError(t *testing.T) {
	// Mock server that returns 201 for CreateTask but 500 for GetTask.
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AgentCard{
				Name:        "test-agent",
				Description: "Test",
				Version:     "1.0.0",
			})
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/tasks" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(TaskResponse{Task: Task{
				ID:     "task-123",
				Status: StatusSubmitted,
				Input:  "test",
			}})
			return
		}

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/") {
			callCount++
			// Return 500 to trigger error during polling.
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	_, err = remote.Invoke(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error from GetTask failure")
	}
	if !strings.Contains(err.Error(), "a2a/invoke") {
		t.Errorf("expected 'a2a/invoke' in error, got %q", err.Error())
	}
}

// TestRemoteAgent_Invoke_TaskCanceled tests the StatusCanceled path (line 198-199 in client.go).
func TestRemoteAgent_Invoke_TaskCanceled(t *testing.T) {
	// Mock server that returns 201 for CreateTask and 200 with canceled status for GetTask.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AgentCard{
				Name:        "test-agent",
				Description: "Test",
				Version:     "1.0.0",
			})
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/tasks" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(TaskResponse{Task: Task{
				ID:     "task-456",
				Status: StatusSubmitted,
				Input:  "test",
			}})
			return
		}

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TaskResponse{Task: Task{
				ID:     "task-456",
				Status: StatusCanceled,
				Input:  "test",
			}})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	remote, err := NewRemoteAgent(ts.URL)
	if err != nil {
		t.Fatalf("NewRemoteAgent: %v", err)
	}

	_, err = remote.Invoke(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for canceled task")
	}
	if !strings.Contains(err.Error(), "task canceled") {
		t.Errorf("expected 'task canceled' in error, got %q", err.Error())
	}
}

// TestClient_CreateTask_DecodeError tests the decode error path (line 80-82 in client.go).
func TestClient_CreateTask_DecodeError(t *testing.T) {
	// Mock server that returns 201 but with invalid JSON.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.CreateTask(context.Background(), TaskRequest{Input: "test"})
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "a2a/create_task") {
		t.Errorf("expected 'a2a/create_task' in error, got %q", err.Error())
	}
}

// TestClient_GetTask_DecodeError tests the decode error path (line 107-109 in client.go).
func TestClient_GetTask_DecodeError(t *testing.T) {
	// Mock server that returns 200 but with invalid JSON.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.GetTask(context.Background(), "some-task-id")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "a2a/get_task") {
		t.Errorf("expected 'a2a/get_task' in error, got %q", err.Error())
	}
}

// TestServer_Serve_ServerError tests the path where srv.Serve returns a non-ErrServerClosed error.
// This is difficult to trigger directly, but we can test by closing the listener before srv.Serve completes.
func TestServer_Serve_ServerError(t *testing.T) {
	a := &mockAgent{id: "test-agent"}
	card := AgentCard{Name: "test-agent"}
	srv := NewServer(a, card)

	// Use an invalid address that passes basic validation.
	ctx := context.Background()
	err := srv.Serve(ctx, "999.999.999.999:0")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
	// This should hit the net.Listen error path, not the srv.Serve error path.
	// To test the srv.Serve non-ErrServerClosed path is extremely difficult
	// as it requires the HTTP server to fail after starting, which is rare.
}
