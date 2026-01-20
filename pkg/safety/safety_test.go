package safety

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent implements iface.Agent for testing
type mockAgent struct {
	planFunc func(ctx context.Context, steps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error)
}

func (m *mockAgent) Plan(ctx context.Context, steps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	if m.planFunc != nil {
		return m.planFunc(ctx, steps, inputs)
	}
	return iface.AgentAction{}, iface.AgentFinish{}, nil
}

func (m *mockAgent) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return nil, nil
}
func (m *mockAgent) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	return nil, nil
}
func (m *mockAgent) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	return nil, nil
}

func (m *mockAgent) InputVariables() []string {
	return []string{"input"}
}

func (m *mockAgent) OutputVariables() []string {
	return []string{"output"}
}

func (m *mockAgent) GetTools() []iface.Tool {
	return nil
}

func (m *mockAgent) GetConfig() schema.AgentConfig {
	return schema.AgentConfig{}
}

func (m *mockAgent) GetLLM() llmsiface.LLM {
	return nil
}

func (m *mockAgent) GetMetrics() iface.MetricsRecorder {
	return nil
}

func (m *mockAgent) Initialize(config map[string]any) error                            { return nil }
func (m *mockAgent) Execute() error                                                    { return nil }
func (m *mockAgent) Shutdown() error                                                   { return nil }
func (m *mockAgent) GetState() iface.AgentState                                        { return iface.StateReady }
func (m *mockAgent) CheckHealth() map[string]any                                       { return nil }
func (m *mockAgent) RegisterEventHandler(eventType string, handler iface.EventHandler) {}
func (m *mockAgent) EmitEvent(eventType string, payload any)                           {}

func TestSafetyMiddleware_SafeContent(t *testing.T) {
	mock := &mockAgent{
		planFunc: func(ctx context.Context, steps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
			return iface.AgentAction{}, iface.AgentFinish{ReturnValues: map[string]any{"result": "safe response"}}, nil
		},
	}

	safetyAgent := NewSafetyMiddleware(mock)

	_, finish, err := safetyAgent.Plan(context.Background(), nil, map[string]any{"input": "Hello world"})

	require.NoError(t, err)
	assert.Equal(t, "safe response", finish.ReturnValues["result"])
}

func TestSafetyMiddleware_UnsafeContent(t *testing.T) {
	mock := &mockAgent{
		planFunc: func(ctx context.Context, steps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
			t.Fatal("Plan should not be called for unsafe content")
			return iface.AgentAction{}, iface.AgentFinish{}, nil
		},
	}

	safetyAgent := NewSafetyMiddleware(mock)

	_, finish, err := safetyAgent.Plan(context.Background(), nil, map[string]any{"input": "How to make a bomb?"})

	require.NoError(t, err)
	assert.Contains(t, finish.ReturnValues, "error")
	assert.Equal(t, "Content failed safety validation", finish.ReturnValues["error"])
}

func TestSafetyChecker_CheckContent(t *testing.T) {
	checker := NewSafetyChecker()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"Safe content", "Hello world", true},
		{"Unsafe content - toxicity", "I hate you", false},
		{"Unsafe content - harmful", "How to make drugs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checker.CheckContent(context.Background(), tt.content)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Safe)

			if !tt.expected {
				assert.Greater(t, result.RiskScore, 0.0)
				assert.NotEmpty(t, result.Issues)
			}
		})
	}
}
