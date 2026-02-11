package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id       string
	persona  agent.Persona
	tools    []tool.Tool
	invokeFn func(ctx context.Context, input string) (string, error)
}

func (a *mockAgent) ID() string            { return a.id }
func (a *mockAgent) Persona() agent.Persona { return a.persona }
func (a *mockAgent) Tools() []tool.Tool     { return a.tools }
func (a *mockAgent) Children() []agent.Agent { return nil }

func (a *mockAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	if a.invokeFn != nil {
		return a.invokeFn(ctx, input)
	}
	return "response to: " + input, nil
}

func (a *mockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, err := a.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{}, err)
			return
		}
		yield(agent.Event{Type: agent.EventText, Text: result, AgentID: a.id}, nil)
	}
}

func newTestAgent() *mockAgent {
	return &mockAgent{
		id:      "test-agent",
		persona: agent.Persona{Role: "tester", Goal: "testing agent"},
	}
}

// mockQueue implements eventqueue.Queue for direct Execute/Cancel testing.
type mockQueue struct {
	events   []a2a.Event
	writeErr error
	writeFn  func(ctx context.Context, event a2a.Event) error
}

func (q *mockQueue) Write(ctx context.Context, event a2a.Event) error {
	if q.writeFn != nil {
		return q.writeFn(ctx, event)
	}
	if q.writeErr != nil {
		return q.writeErr
	}
	q.events = append(q.events, event)
	return nil
}

func (q *mockQueue) WriteVersioned(ctx context.Context, event a2a.Event, _ a2a.TaskVersion) error {
	return q.Write(ctx, event)
}

func (q *mockQueue) Read(_ context.Context) (a2a.Event, a2a.TaskVersion, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (q *mockQueue) Close() error { return nil }

// Compile-time interface check for mockQueue.
var _ eventqueue.Queue = (*mockQueue)(nil)

// simpleTool is a minimal tool.Tool for testing.
type simpleTool struct {
	name string
	desc string
}

func (t *simpleTool) Name() string               { return t.name }
func (t *simpleTool) Description() string         { return t.desc }
func (t *simpleTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *simpleTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	return tool.TextResult("ok"), nil
}

// --- NewServer tests ---

func TestNewServer(t *testing.T) {
	a := newTestAgent()

	handler, card := NewServer(a, ServerConfig{
		Name:        "test-agent",
		Version:     "1.0.0",
		Description: "A test agent",
		URL:         "http://localhost:9090",
	})

	require.NotNil(t, handler)
	require.NotNil(t, card)
	assert.Equal(t, "test-agent", card.Name)
	assert.Equal(t, "1.0.0", card.Version)
	assert.Equal(t, "A test agent", card.Description)
	assert.Equal(t, "http://localhost:9090", card.URL)
}

// --- buildSkills tests ---

func TestBuildSkills(t *testing.T) {
	mockTool := &simpleTool{name: "calculator", desc: "Does math"}
	a := &mockAgent{
		id:      "agent-1",
		persona: agent.Persona{Role: "assistant", Goal: "help users"},
		tools:   []tool.Tool{mockTool},
	}

	skills := buildSkills(a)
	require.Len(t, skills, 2)

	assert.Equal(t, "agent-1", skills[0].ID)
	assert.Equal(t, "assistant", skills[0].Name)
	assert.Contains(t, skills[0].Tags, "agent")

	assert.Equal(t, "calculator", skills[1].ID)
	assert.Equal(t, "calculator", skills[1].Name)
	assert.Equal(t, "Does math", skills[1].Description)
	assert.Contains(t, skills[1].Tags, "tool")
}

func TestBuildSkillsNoTools(t *testing.T) {
	a := newTestAgent()
	skills := buildSkills(a)
	require.Len(t, skills, 1)
}

// --- AgentCard endpoint test ---

func TestAgentCardEndpoint(t *testing.T) {
	a := newTestAgent()

	handler, _ := NewServer(a, ServerConfig{
		Name:    "test-agent",
		Version: "1.0.0",
		URL:     "http://localhost:9090",
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/.well-known/agent-card.json")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var card a2a.AgentCard
	err = json.NewDecoder(resp.Body).Decode(&card)
	require.NoError(t, err)
	assert.Equal(t, "test-agent", card.Name)
}

// --- extractInput tests ---

func TestExtractInput(t *testing.T) {
	tests := []struct {
		name string
		msg  *a2a.Message
		want string
	}{
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
		{
			name: "text part",
			msg:  a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "hello"}),
			want: "hello",
		},
		{
			name: "empty message",
			msg:  a2a.NewMessage(a2a.MessageRoleUser),
			want: "",
		},
		{
			name: "pointer text part",
			msg: &a2a.Message{
				Role:  a2a.MessageRoleUser,
				Parts: a2a.ContentParts{&a2a.TextPart{Text: "pointer text"}},
			},
			want: "pointer text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInput(tt.msg)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- RemoteAgent interface tests ---

func TestRemoteAgentInterface(t *testing.T) {
	ra := &remoteAgent{
		card: &a2a.AgentCard{
			Name:        "remote",
			Description: "A remote agent",
		},
	}

	assert.Equal(t, "remote", ra.ID())
	assert.Equal(t, "remote", ra.Persona().Role)
	assert.Equal(t, "A remote agent", ra.Persona().Goal)
	assert.Nil(t, ra.Tools())
	assert.Nil(t, ra.Children())
}

// --- extractResultText tests ---

func TestExtractResultText(t *testing.T) {
	tests := []struct {
		name   string
		result a2a.SendMessageResult
		want   string
	}{
		{
			name:   "nil result",
			result: nil,
			want:   "",
		},
		{
			name: "task with status message",
			result: &a2a.Task{
				ID:        "task-1",
				ContextID: "ctx-1",
				Status: a2a.TaskStatus{
					State:   a2a.TaskStateCompleted,
					Message: a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{Text: "done"}),
				},
			},
			want: "done",
		},
		{
			name: "task without message no artifacts",
			result: &a2a.Task{
				ID:        "task-2",
				ContextID: "ctx-2",
				Status: a2a.TaskStatus{
					State: a2a.TaskStateCompleted,
				},
			},
			want: "",
		},
		{
			name: "task with artifact text part",
			result: &a2a.Task{
				ID:        "task-3",
				ContextID: "ctx-3",
				Status: a2a.TaskStatus{
					State: a2a.TaskStateCompleted,
				},
				Artifacts: []*a2a.Artifact{
					{Parts: a2a.ContentParts{a2a.TextPart{Text: "from artifact"}}},
				},
			},
			want: "from artifact",
		},
		{
			name: "task with artifact pointer text part",
			result: &a2a.Task{
				ID:        "task-4",
				ContextID: "ctx-4",
				Status: a2a.TaskStatus{
					State: a2a.TaskStateCompleted,
				},
				Artifacts: []*a2a.Artifact{
					{Parts: a2a.ContentParts{&a2a.TextPart{Text: "from pointer"}}},
				},
			},
			want: "from pointer",
		},
		{
			name:   "message type",
			result: a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{Text: "from message"}),
			want:   "from message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResultText(tt.result)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ServerConfig test ---

func TestServerConfig(t *testing.T) {
	cfg := ServerConfig{
		Name:        "test",
		Version:     "2.0.0",
		Description: "desc",
		URL:         "http://example.com",
	}
	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, "2.0.0", cfg.Version)
	assert.Equal(t, "desc", cfg.Description)
	assert.Equal(t, "http://example.com", cfg.URL)
}

// --- NewRemoteAgent tests ---

func TestNewRemoteAgentInvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewRemoteAgent(ctx, "http://127.0.0.1:1")
	require.Error(t, err)
}

// --- belugaExecutor.Execute tests ---

func TestBelugaExecutor_Execute_Success(t *testing.T) {
	a := newTestAgent()
	executor := &belugaExecutor{agent: a}

	q := &mockQueue{}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.NewTaskID(),
		ContextID: "test-ctx",
		Message:   a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "hello"}),
	}

	err := executor.Execute(context.Background(), reqCtx, q)
	require.NoError(t, err)
	assert.Len(t, q.events, 2) // working + completed
}

func TestBelugaExecutor_Execute_NilMessage(t *testing.T) {
	a := newTestAgent()
	executor := &belugaExecutor{agent: a}

	q := &mockQueue{}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.NewTaskID(),
		ContextID: "test-ctx",
		Message:   nil,
	}

	err := executor.Execute(context.Background(), reqCtx, q)
	require.NoError(t, err)
	assert.Len(t, q.events, 2) // working + completed (agent invoked with empty input)
}

func TestBelugaExecutor_Execute_AgentError(t *testing.T) {
	a := &mockAgent{
		id:      "fail",
		persona: agent.Persona{Role: "fail", Goal: "testing"},
		invokeFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("agent failed")
		},
	}
	executor := &belugaExecutor{agent: a}

	q := &mockQueue{}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.NewTaskID(),
		ContextID: "test-ctx",
		Message:   a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "hello"}),
	}

	err := executor.Execute(context.Background(), reqCtx, q)
	require.NoError(t, err) // Returns nil even when agent errors
	assert.Len(t, q.events, 2) // working + failed
}

func TestBelugaExecutor_Execute_WriteWorkingError(t *testing.T) {
	a := newTestAgent()
	executor := &belugaExecutor{agent: a}

	q := &mockQueue{writeErr: fmt.Errorf("queue full")}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.NewTaskID(),
		ContextID: "test-ctx",
		Message:   a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "hello"}),
	}

	err := executor.Execute(context.Background(), reqCtx, q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write working event")
}

func TestBelugaExecutor_Execute_WriteCompletedError(t *testing.T) {
	a := newTestAgent()
	executor := &belugaExecutor{agent: a}

	callCount := 0
	q := &mockQueue{
		writeFn: func(_ context.Context, _ a2a.Event) error {
			callCount++
			if callCount == 2 { // second write = completed event
				return fmt.Errorf("write failed")
			}
			return nil
		},
	}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.NewTaskID(),
		ContextID: "test-ctx",
		Message:   a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "hello"}),
	}

	err := executor.Execute(context.Background(), reqCtx, q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write completed event")
}

func TestBelugaExecutor_Execute_WriteFailedEventError(t *testing.T) {
	a := &mockAgent{
		id:      "fail",
		persona: agent.Persona{Role: "fail", Goal: "testing"},
		invokeFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("agent failed")
		},
	}
	executor := &belugaExecutor{agent: a}

	callCount := 0
	q := &mockQueue{
		writeFn: func(_ context.Context, _ a2a.Event) error {
			callCount++
			if callCount == 2 { // second write = failed event
				return fmt.Errorf("write failed")
			}
			return nil
		},
	}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.NewTaskID(),
		ContextID: "test-ctx",
		Message:   a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "hello"}),
	}

	err := executor.Execute(context.Background(), reqCtx, q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed event")
}

// --- belugaExecutor.Cancel tests ---

func TestBelugaExecutor_Cancel_Success(t *testing.T) {
	a := newTestAgent()
	executor := &belugaExecutor{agent: a}

	q := &mockQueue{}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.NewTaskID(),
		ContextID: "test-ctx",
	}

	err := executor.Cancel(context.Background(), reqCtx, q)
	require.NoError(t, err)
	assert.Len(t, q.events, 1)
}

func TestBelugaExecutor_Cancel_WriteError(t *testing.T) {
	a := newTestAgent()
	executor := &belugaExecutor{agent: a}

	q := &mockQueue{writeErr: fmt.Errorf("queue full")}
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.NewTaskID(),
		ContextID: "test-ctx",
	}

	err := executor.Cancel(context.Background(), reqCtx, q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write cancel event")
}

// --- Round-trip tests using a2aclient with explicit JSON-RPC transport ---

// setupSDKTestServer creates an A2A SDK server and returns the test server
// along with a remoteAgent backed by the official SDK client.
func setupSDKTestServer(t *testing.T, a agent.Agent) (*httptest.Server, *remoteAgent) {
	t.Helper()
	handler, sdkCard := NewServer(a, ServerConfig{
		Name:        "test-agent",
		Version:     "1.0.0",
		Description: "A test agent",
		URL:         "http://localhost",
	})
	ts := httptest.NewServer(handler)

	// Patch the card URL to point at the test server.
	sdkCard.URL = ts.URL
	if sdkCard.PreferredTransport == "" {
		sdkCard.PreferredTransport = a2a.TransportProtocolJSONRPC
	}

	ctx := context.Background()
	client, err := a2aclient.NewFromCard(ctx, sdkCard,
		a2aclient.WithJSONRPCTransport(http.DefaultClient))
	require.NoError(t, err)

	ra := &remoteAgent{client: client, card: sdkCard}
	return ts, ra
}

func TestRemoteAgent_Invoke_RoundTrip(t *testing.T) {
	ts, ra := setupSDKTestServer(t, newTestAgent())
	defer ts.Close()

	result, err := ra.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "response to: hello", result)
}

func TestRemoteAgent_Invoke_AgentError(t *testing.T) {
	failAgent := &mockAgent{
		id:      "fail",
		persona: agent.Persona{Role: "fail", Goal: "testing"},
		invokeFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("agent failed")
		},
	}
	ts, ra := setupSDKTestServer(t, failAgent)
	defer ts.Close()

	// The task should complete with the error message as the result text.
	result, err := ra.Invoke(context.Background(), "hello")
	if err != nil {
		// Some A2A implementations might return an error.
		assert.Contains(t, err.Error(), "invoke")
	} else {
		// If no error, the result should contain the failure message.
		assert.NotEmpty(t, result)
	}
}

func TestRemoteAgent_Stream_RoundTrip(t *testing.T) {
	ts, ra := setupSDKTestServer(t, newTestAgent())
	defer ts.Close()

	var events []agent.Event
	for event, err := range ra.Stream(context.Background(), "hello") {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		events = append(events, event)
	}

	require.Len(t, events, 2) // text + done
	assert.Equal(t, agent.EventText, events[0].Type)
	assert.Equal(t, "response to: hello", events[0].Text)
	assert.Equal(t, agent.EventDone, events[1].Type)
}

func TestRemoteAgent_Stream_Error(t *testing.T) {
	failAgent := &mockAgent{
		id:      "fail",
		persona: agent.Persona{Role: "fail", Goal: "testing"},
		invokeFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("agent failed")
		},
	}
	ts, ra := setupSDKTestServer(t, failAgent)
	defer ts.Close()

	var events []agent.Event
	var gotErr error
	for event, err := range ra.Stream(context.Background(), "hello") {
		if err != nil {
			gotErr = err
			break
		}
		events = append(events, event)
	}

	// Either an error was returned or we got events.
	if gotErr != nil {
		assert.Contains(t, gotErr.Error(), "invoke")
	} else {
		assert.NotEmpty(t, events)
	}
}

func TestRemoteAgent_Stream_EarlyTermination(t *testing.T) {
	ts, ra := setupSDKTestServer(t, newTestAgent())
	defer ts.Close()

	// Break after first event to test early termination (yield returns false).
	count := 0
	for event, err := range ra.Stream(context.Background(), "hello") {
		require.NoError(t, err)
		assert.Equal(t, agent.EventText, event.Type)
		count++
		break // terminate early
	}
	assert.Equal(t, 1, count)
}

func TestRemoteAgent_Invoke_SendError(t *testing.T) {
	ts, ra := setupSDKTestServer(t, newTestAgent())
	// Close the server before calling Invoke to trigger a send error.
	ts.Close()

	_, err := ra.Invoke(context.Background(), "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a2a/sdk/invoke")
}

func TestNewRemoteAgent_CardError(t *testing.T) {
	// Server that doesn't serve an agent card.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := NewRemoteAgent(ctx, ts.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a2a/sdk")
}
