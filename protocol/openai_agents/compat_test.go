package openai_agents

import (
	"context"
	"encoding/json"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id       string
	persona  agent.Persona
	tools    []tool.Tool
	children []agent.Agent
	output   string
}

func (a *mockAgent) ID() string                  { return a.id }
func (a *mockAgent) Persona() agent.Persona       { return a.persona }
func (a *mockAgent) Tools() []tool.Tool            { return a.tools }
func (a *mockAgent) Children() []agent.Agent       { return a.children }
func (a *mockAgent) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return a.output, nil
}
func (a *mockAgent) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: a.output}, nil)
	}
}

// mockTool implements tool.Tool for testing.
type mockTool struct {
	name        string
	description string
	schema      map[string]any
}

func (t *mockTool) Name() string                  { return t.name }
func (t *mockTool) Description() string            { return t.description }
func (t *mockTool) InputSchema() map[string]any    { return t.schema }
func (t *mockTool) Execute(_ context.Context, _ map[string]any) (*tool.Result, error) {
	return &tool.Result{
		Content: []schema.ContentPart{schema.TextPart{Text: "result"}},
	}, nil
}

func TestFromAgent(t *testing.T) {
	t.Run("basic agent", func(t *testing.T) {
		a := &mockAgent{
			id:      "test-agent",
			persona: agent.Persona{Goal: "Be helpful"},
		}

		def := FromAgent(a)
		assert.Equal(t, "test-agent", def.Name)
		assert.Equal(t, "Be helpful", def.Instructions)
		assert.Empty(t, def.Tools)
		assert.Empty(t, def.Handoffs)
	})

	t.Run("with tools", func(t *testing.T) {
		a := &mockAgent{
			id: "agent-with-tools",
			tools: []tool.Tool{
				&mockTool{name: "calculator", description: "Do math", schema: map[string]any{"type": "object"}},
				&mockTool{name: "search", description: "Search the web"},
			},
		}

		def := FromAgent(a)
		assert.Len(t, def.Tools, 2)
		assert.Equal(t, "function", def.Tools[0].Type)
		assert.Equal(t, "calculator", def.Tools[0].Function.Name)
		assert.Equal(t, "Do math", def.Tools[0].Function.Description)
		assert.Equal(t, "search", def.Tools[1].Function.Name)
	})

	t.Run("with handoffs", func(t *testing.T) {
		child := &mockAgent{
			id:      "specialist",
			persona: agent.Persona{Goal: "Specialized task"},
		}
		a := &mockAgent{
			id:       "orchestrator",
			children: []agent.Agent{child},
		}

		def := FromAgent(a)
		assert.Len(t, def.Handoffs, 1)
		assert.Equal(t, "specialist", def.Handoffs[0].Name)
		assert.Equal(t, "Specialized task", def.Handoffs[0].Description)
	})
}

func TestFromTools(t *testing.T) {
	tools := []tool.Tool{
		&mockTool{name: "t1", description: "Tool 1", schema: map[string]any{"type": "object"}},
		&mockTool{name: "t2", description: "Tool 2"},
	}

	defs := FromTools(tools)
	assert.Len(t, defs, 2)
	assert.Equal(t, "function", defs[0].Type)
	assert.Equal(t, "t1", defs[0].Function.Name)
	assert.Equal(t, "Tool 2", defs[1].Function.Description)
}

func TestRunner(t *testing.T) {
	t.Run("run success", func(t *testing.T) {
		a := &mockAgent{id: "assistant", output: "Hello there!"}
		r := NewRunner(a)

		resp, err := r.Run(context.Background(), RunRequest{
			AgentName: "assistant",
			Input:     "Hi",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello there!", resp.Output)
		assert.Equal(t, "assistant", resp.Agent)
	})

	t.Run("unknown agent", func(t *testing.T) {
		r := NewRunner()
		_, err := r.Run(context.Background(), RunRequest{AgentName: "nonexistent"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown agent")
	})

	t.Run("list agents", func(t *testing.T) {
		a1 := &mockAgent{id: "agent1", persona: agent.Persona{Goal: "Goal 1"}}
		a2 := &mockAgent{id: "agent2", persona: agent.Persona{Goal: "Goal 2"}}
		r := NewRunner(a1, a2)

		defs := r.ListAgents()
		assert.Len(t, defs, 2)
	})
}

func TestMarshalAgentDef(t *testing.T) {
	def := AgentDef{
		Name:         "test",
		Instructions: "Be helpful",
		Tools: []ToolDef{
			{Type: "function", Function: FunctionDef{Name: "calc", Description: "Calculate"}},
		},
	}

	data, err := MarshalAgentDef(def)
	require.NoError(t, err)

	var decoded AgentDef
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "test", decoded.Name)
	assert.Len(t, decoded.Tools, 1)
}

func TestUnmarshalRunRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		data := []byte(`{"agent_name":"assistant","input":"hello","stream":true}`)
		req, err := UnmarshalRunRequest(data)
		require.NoError(t, err)
		assert.Equal(t, "assistant", req.AgentName)
		assert.Equal(t, "hello", req.Input)
		assert.True(t, req.Stream)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := UnmarshalRunRequest([]byte(`{invalid}`))
		assert.Error(t, err)
	})
}
