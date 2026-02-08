package sdk

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
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

func (a *mockAgent) ID() string                   { return a.id }
func (a *mockAgent) Persona() agent.Persona        { return a.persona }
func (a *mockAgent) Tools() []tool.Tool            { return a.tools }
func (a *mockAgent) Children() []agent.Agent        { return nil }

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

func TestBuildSkills(t *testing.T) {
	mockTool := &simpleTool{name: "calculator", desc: "Does math"}
	a := &mockAgent{
		id:      "agent-1",
		persona: agent.Persona{Role: "assistant", Goal: "help users"},
		tools:   []tool.Tool{mockTool},
	}

	skills := buildSkills(a)
	require.Len(t, skills, 2)

	// First skill is the agent itself.
	assert.Equal(t, "agent-1", skills[0].ID)
	assert.Equal(t, "assistant", skills[0].Name)
	assert.Contains(t, skills[0].Tags, "agent")

	// Second skill is the tool.
	assert.Equal(t, "calculator", skills[1].ID)
	assert.Equal(t, "calculator", skills[1].Name)
	assert.Equal(t, "Does math", skills[1].Description)
	assert.Contains(t, skills[1].Tags, "tool")
}

func TestBuildSkillsNoTools(t *testing.T) {
	a := newTestAgent()
	skills := buildSkills(a)
	require.Len(t, skills, 1) // Just the agent skill.
}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInput(tt.msg)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
			name: "task without message",
			result: &a2a.Task{
				ID:        "task-2",
				ContextID: "ctx-2",
				Status: a2a.TaskStatus{
					State: a2a.TaskStateCompleted,
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResultText(tt.result)
			assert.Equal(t, tt.want, got)
		})
	}
}

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

func TestNewRemoteAgentInvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// A non-existent server should fail.
	_, err := NewRemoteAgent(ctx, "http://127.0.0.1:1")
	require.Error(t, err)
}

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
