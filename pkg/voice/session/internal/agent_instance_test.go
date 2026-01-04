// Package internal provides comprehensive tests for agent instance management.
// T163: Add missing test cases for edge cases in agent integration
package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
)

// mockStreamingAgentForInstance is a mock streaming agent for testing agent instances.
// It implements the full StreamingAgent interface.
type mockStreamingAgentForInstance struct {
	name string
}

// Agent interface methods.
func (m *mockStreamingAgentForInstance) InputVariables() []string {
	return []string{"input"}
}

func (m *mockStreamingAgentForInstance) OutputVariables() []string {
	return []string{"output"}
}

func (m *mockStreamingAgentForInstance) GetTools() []tools.Tool {
	return []tools.Tool{}
}

func (m *mockStreamingAgentForInstance) GetConfig() schema.AgentConfig {
	return schema.AgentConfig{Name: m.name}
}

func (m *mockStreamingAgentForInstance) GetLLM() llmsiface.LLM {
	return nil
}

func (m *mockStreamingAgentForInstance) GetMetrics() iface.MetricsRecorder {
	return nil
}

func (m *mockStreamingAgentForInstance) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	_ = ctx
	_ = intermediateSteps
	_ = inputs
	return iface.AgentAction{}, iface.AgentFinish{}, nil
}

// StreamingAgent interface methods.
func (m *mockStreamingAgentForInstance) StreamExecute(ctx context.Context, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	ch := make(chan iface.AgentStreamChunk)
	go func() {
		defer close(ch)
	}()
	return ch, nil
}

func (m *mockStreamingAgentForInstance) StreamPlan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	ch := make(chan iface.AgentStreamChunk)
	go func() {
		defer close(ch)
	}()
	return ch, nil
}

// TestNewAgentInstance tests creating a new agent instance.
func TestNewAgentInstance(t *testing.T) {
	agent := &mockStreamingAgentForInstance{name: "test-agent"}
	config := schema.AgentConfig{
		Name: "test-agent",
	}

	instance := NewAgentInstance(agent, config)

	assert.NotNil(t, instance)
	assert.Equal(t, agent, instance.Agent)
	assert.Equal(t, config, instance.Config)
	assert.Equal(t, AgentStateIdle, instance.State)
	assert.NotNil(t, instance.Context)
	assert.Equal(t, AgentStateIdle, instance.GetState())
}

// TestAgentInstance_GetState tests getting agent state.
func TestAgentInstance_GetState(t *testing.T) {
	agent := &mockStreamingAgentForInstance{name: "test-agent"}
	config := schema.AgentConfig{Name: "test-agent"}
	instance := NewAgentInstance(agent, config)

	// Initial state should be idle
	assert.Equal(t, AgentStateIdle, instance.GetState())

	// Manually set state (bypassing SetState for testing)
	instance.mu.Lock()
	instance.State = AgentStateListening
	instance.mu.Unlock()

	assert.Equal(t, AgentStateListening, instance.GetState())
}

// TestAgentInstance_SetState tests setting agent state with validation.
func TestAgentInstance_SetState(t *testing.T) {
	agent := &mockStreamingAgentForInstance{name: "test-agent"}
	config := schema.AgentConfig{Name: "test-agent"}
	instance := NewAgentInstance(agent, config)

	tests := []struct {
		name      string
		fromState AgentState
		toState   AgentState
		wantError bool
	}{
		{
			name:      "valid transition: idle -> listening",
			fromState: AgentStateIdle,
			toState:   AgentStateListening,
			wantError: false,
		},
		{
			name:      "valid transition: listening -> processing",
			fromState: AgentStateListening,
			toState:   AgentStateProcessing,
			wantError: false,
		},
		{
			name:      "valid transition: processing -> streaming",
			fromState: AgentStateProcessing,
			toState:   AgentStateStreaming,
			wantError: false,
		},
		{
			name:      "invalid transition: idle -> streaming",
			fromState: AgentStateIdle,
			toState:   AgentStateStreaming,
			wantError: true,
		},
		{
			name:      "invalid transition: idle -> executing",
			fromState: AgentStateIdle,
			toState:   AgentStateExecuting,
			wantError: true,
		},
		{
			name:      "same state transition",
			fromState: AgentStateIdle,
			toState:   AgentStateIdle,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance.mu.Lock()
			instance.State = tt.fromState
			instance.mu.Unlock()

			err := instance.SetState(tt.toState)

			if tt.wantError {
				assert.Error(t, err)
				assert.IsType(t, &AgentStateError{}, err)
				// State should remain unchanged
				assert.Equal(t, tt.fromState, instance.GetState())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.toState, instance.GetState())
			}
		})
	}
}

// TestAgentInstance_GetContext tests getting agent context.
func TestAgentInstance_GetContext(t *testing.T) {
	agent := &mockStreamingAgentForInstance{name: "test-agent"}
	config := schema.AgentConfig{Name: "test-agent"}
	instance := NewAgentInstance(agent, config)

	ctx := instance.GetContext()
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.ConversationHistory)
	assert.NotNil(t, ctx.ToolResults)
	assert.NotNil(t, ctx.CurrentPlan)
	assert.False(t, ctx.StreamingActive)
}

// TestAgentInstance_UpdateContext tests updating agent context.
func TestAgentInstance_UpdateContext(t *testing.T) {
	agent := &mockStreamingAgentForInstance{name: "test-agent"}
	config := schema.AgentConfig{Name: "test-agent"}
	instance := NewAgentInstance(agent, config)

	// Update context with a message
	instance.UpdateContext(func(ctx *AgentContext) {
		ctx.ConversationHistory = append(ctx.ConversationHistory, schema.NewHumanMessage("Hello"))
		ctx.StreamingActive = true
	})

	ctx := instance.GetContext()
	assert.Len(t, ctx.ConversationHistory, 1)
	assert.True(t, ctx.StreamingActive)
}

// TestAgentInstance_ConcurrentAccess tests thread-safety of agent instance.
func TestAgentInstance_ConcurrentAccess(t *testing.T) {
	agent := &mockStreamingAgentForInstance{name: "test-agent"}
	config := schema.AgentConfig{Name: "test-agent"}
	instance := NewAgentInstance(agent, config)

	// Concurrent state reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			// Read state
			_ = instance.GetState()
			// Update context
			instance.UpdateContext(func(ctx *AgentContext) {
				ctx.StreamingActive = id%2 == 0
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and context should be accessible
	ctx := instance.GetContext()
	assert.NotNil(t, ctx)
}

// TestAgentInstance_StateTransitions tests comprehensive state transition scenarios.
func TestAgentInstance_StateTransitions(t *testing.T) {
	agent := &mockStreamingAgentForInstance{name: "test-agent"}
	config := schema.AgentConfig{Name: "test-agent"}
	instance := NewAgentInstance(agent, config)

	// Test a valid state sequence
	validSequence := []struct {
		state     AgentState
		wantError bool
	}{
		{AgentStateListening, false},
		{AgentStateProcessing, false},
		{AgentStateStreaming, false},
		{AgentStateSpeaking, false},
		{AgentStateIdle, false},
	}

	for _, step := range validSequence {
		err := instance.SetState(step.state)
		if step.wantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, step.state, instance.GetState())
		}
	}
}

// TestAgentInstance_InvalidStateTransition tests invalid state transitions.
func TestAgentInstance_InvalidStateTransition(t *testing.T) {
	agent := &mockStreamingAgentForInstance{name: "test-agent"}
	config := schema.AgentConfig{Name: "test-agent"}
	instance := NewAgentInstance(agent, config)

	// Start in idle
	assert.Equal(t, AgentStateIdle, instance.GetState())

	// Try invalid transition
	err := instance.SetState(AgentStateStreaming)
	assert.Error(t, err)
	assert.IsType(t, &AgentStateError{}, err)

	stateErr := func() *AgentStateError {
		target := &AgentStateError{}
		_ = errors.As(err, &target)
		return target
	}()
	assert.Equal(t, AgentStateIdle, stateErr.From)
	assert.Equal(t, AgentStateStreaming, stateErr.To)
}

// TestAgentStateError tests AgentStateError formatting.
func TestAgentStateError(t *testing.T) {
	err := &AgentStateError{
		From: AgentStateIdle,
		To:   AgentStateStreaming,
		Msg:  "invalid state transition",
	}

	errorStr := err.Error()
	assert.Contains(t, errorStr, "invalid state transition")
	assert.Contains(t, errorStr, "idle")
	assert.Contains(t, errorStr, "streaming")
}
