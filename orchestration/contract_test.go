package orchestration

import (
	"context"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
	"github.com/stretchr/testify/assert"
)

// contractMockAgent is a test agent with a contract.
type contractMockAgent struct {
	id       string
	contract *schema.Contract
}

var _ agent.Agent = (*contractMockAgent)(nil)

func (m *contractMockAgent) ID() string                 { return m.id }
func (m *contractMockAgent) Persona() agent.Persona     { return agent.Persona{} }
func (m *contractMockAgent) Tools() []tool.Tool         { return nil }
func (m *contractMockAgent) Children() []agent.Agent    { return nil }
func (m *contractMockAgent) Contract() *schema.Contract { return m.contract }

func (m *contractMockAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	return input, nil
}

func (m *contractMockAgent) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {}
}

func TestValidatePipelineContracts_Compatible(t *testing.T) {
	a1 := &contractMockAgent{
		id: "agent-1",
		contract: &schema.Contract{
			Name: "agent-1",
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{"type": "string"},
				},
			},
		},
	}
	a2 := &contractMockAgent{
		id: "agent-2",
		contract: &schema.Contract{
			Name: "agent-2",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{"type": "string"},
				},
				"required": []any{"summary"},
			},
		},
	}

	err := ValidatePipelineContracts(a1, a2)
	assert.NoError(t, err)
}

func TestValidatePipelineContracts_Incompatible(t *testing.T) {
	a1 := &contractMockAgent{
		id: "agent-1",
		contract: &schema.Contract{
			Name: "agent-1",
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"result": map[string]any{"type": "string"},
				},
			},
		},
	}
	a2 := &contractMockAgent{
		id: "agent-2",
		contract: &schema.Contract{
			Name: "agent-2",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary": map[string]any{"type": "string"},
				},
				"required": []any{"summary"},
			},
		},
	}

	err := ValidatePipelineContracts(a1, a2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "summary")
}

func TestValidatePipelineContracts_NoContracts(t *testing.T) {
	a1 := &contractMockAgent{id: "agent-1"}
	a2 := &contractMockAgent{id: "agent-2"}

	err := ValidatePipelineContracts(a1, a2)
	assert.NoError(t, err)
}

func TestValidatePipelineContracts_SingleAgent(t *testing.T) {
	a1 := &contractMockAgent{id: "agent-1"}
	err := ValidatePipelineContracts(a1)
	assert.NoError(t, err)
}
