package contract

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent is a minimal agent for testing.
type mockAgent struct {
	id       string
	contract *schema.Contract
	result   string
}

var _ agent.Agent = (*mockAgent)(nil)

func (m *mockAgent) ID() string                 { return m.id }
func (m *mockAgent) Persona() agent.Persona     { return agent.Persona{} }
func (m *mockAgent) Tools() []tool.Tool         { return nil }
func (m *mockAgent) Children() []agent.Agent    { return nil }
func (m *mockAgent) Contract() *schema.Contract { return m.contract }

func (m *mockAgent) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return m.result, nil
}

func (m *mockAgent) Stream(_ context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: m.result, AgentID: m.id}, nil)
	}
}

// mockAgentNoContract is an agent without ContractProvider.
type mockAgentNoContract struct {
	id string
}

var _ agent.Agent = (*mockAgentNoContract)(nil)

func (m *mockAgentNoContract) ID() string              { return m.id }
func (m *mockAgentNoContract) Persona() agent.Persona  { return agent.Persona{} }
func (m *mockAgentNoContract) Tools() []tool.Tool      { return nil }
func (m *mockAgentNoContract) Children() []agent.Agent { return nil }

func (m *mockAgentNoContract) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return "ok", nil
}

func (m *mockAgentNoContract) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: "ok"}, nil)
	}
}

func TestValidationMiddleware_PassThrough(t *testing.T) {
	mw := ValidationMiddleware()
	a := &mockAgentNoContract{id: "no-contract"}
	wrapped := mw(a)

	// Should be the same agent (no wrapping).
	result, err := wrapped.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestValidationMiddleware_ValidInput(t *testing.T) {
	mw := ValidationMiddleware()
	a := &mockAgent{
		id: "test",
		contract: &schema.Contract{
			Name:        "test",
			InputSchema: map[string]any{"type": "string"},
		},
		result: "output",
	}
	wrapped := mw(a)

	result, err := wrapped.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "output", result)
}

func TestValidationMiddleware_InvalidInput(t *testing.T) {
	mw := ValidationMiddleware()
	a := &mockAgent{
		id: "test",
		contract: &schema.Contract{
			Name: "test",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
				"required": []any{"name"},
			},
		},
		result: "output",
	}
	wrapped := mw(a)

	// A plain string that is not valid JSON object should fail.
	_, err := wrapped.Invoke(context.Background(), "not-json-object")
	require.Error(t, err)
	var coreErr *core.Error
	assert.True(t, errors.As(err, &coreErr))
}

func TestValidationMiddleware_InvalidOutput(t *testing.T) {
	mw := ValidationMiddleware()
	a := &mockAgent{
		id: "test",
		contract: &schema.Contract{
			Name:         "test",
			OutputSchema: map[string]any{"type": "number"},
		},
		result: "not-a-number",
	}
	wrapped := mw(a)

	_, err := wrapped.Invoke(context.Background(), "hello")
	require.Error(t, err)
	var coreErr *core.Error
	assert.True(t, errors.As(err, &coreErr))
}

func TestValidationMiddleware_StreamInvalidInput(t *testing.T) {
	mw := ValidationMiddleware()
	a := &mockAgent{
		id: "test",
		contract: &schema.Contract{
			Name: "test",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
				"required": []any{"name"},
			},
		},
		result: "output",
	}
	wrapped := mw(a)

	var gotErr error
	for _, err := range wrapped.Stream(context.Background(), "not-json") {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)
}

func TestValidationMiddleware_StreamValidInput(t *testing.T) {
	mw := ValidationMiddleware()
	a := &mockAgent{
		id: "test",
		contract: &schema.Contract{
			Name:        "test",
			InputSchema: map[string]any{"type": "string"},
		},
		result: "streamed-output",
	}
	wrapped := mw(a)

	var events []agent.Event
	for event, err := range wrapped.Stream(context.Background(), "hello") {
		require.NoError(t, err)
		events = append(events, event)
	}
	assert.Len(t, events, 1)
	assert.Equal(t, "streamed-output", events[0].Text)
}

func TestValidationMiddleware_PreservesContractProvider(t *testing.T) {
	mw := ValidationMiddleware()
	c := &schema.Contract{Name: "test", InputSchema: map[string]any{"type": "string"}}
	a := &mockAgent{
		id:       "test",
		contract: c,
		result:   "output",
	}
	wrapped := mw(a)

	cp, ok := wrapped.(ContractProvider)
	require.True(t, ok)
	assert.Equal(t, c, cp.Contract())
}
