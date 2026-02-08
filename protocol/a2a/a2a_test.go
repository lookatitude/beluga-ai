package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
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

func (m *mockAgent) ID() string               { return m.id }
func (m *mockAgent) Persona() agent.Persona    { return agent.Persona{Role: m.id} }
func (m *mockAgent) Tools() []tool.Tool        { return nil }
func (m *mockAgent) Children() []agent.Agent   { return nil }

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

	// Wait for the agent to complete processing.
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

	// Wait a moment so the task starts working.
	time.Sleep(50 * time.Millisecond)

	if err := client.CancelTask(context.Background(), task.ID); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}

	// Wait for the goroutine to notice the cancellation.
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
