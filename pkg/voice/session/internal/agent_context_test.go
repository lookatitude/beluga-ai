// Package internal provides comprehensive tests for agent context management.
// T163: Add missing test cases for edge cases in agent integration
package internal

import (
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
)

// TestNewAgentContext tests creating a new agent context.
func TestNewAgentContext(t *testing.T) {
	ctx := NewAgentContext()

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.ConversationHistory)
	assert.NotNil(t, ctx.ToolResults)
	assert.NotNil(t, ctx.CurrentPlan)
	assert.False(t, ctx.StreamingActive)
	assert.True(t, ctx.LastInterruption.IsZero())
}

// TestNewStreamingState tests creating a new streaming state.
func TestNewStreamingState(t *testing.T) {
	state := NewStreamingState()

	assert.NotNil(t, state)
	assert.False(t, state.Active)
	assert.Nil(t, state.CurrentStream)
	assert.NotNil(t, state.Buffer)
	assert.Empty(t, state.Buffer)
	assert.True(t, state.LastChunkTime.IsZero())
	assert.False(t, state.Interrupted)
}

// TestNewVoiceCallAgentContext tests creating a new voice call agent context.
func TestNewVoiceCallAgentContext(t *testing.T) {
	sessionID := "test-session-123"
	userID := "test-user-456"

	ctx := NewVoiceCallAgentContext(sessionID, userID)

	assert.NotNil(t, ctx)
	assert.Equal(t, sessionID, ctx.SessionID)
	assert.Equal(t, userID, ctx.UserID)
	assert.False(t, ctx.StartTime.IsZero())
	assert.False(t, ctx.LastActivity.IsZero())
	assert.Nil(t, ctx.AgentInstance)
	assert.NotNil(t, ctx.ConversationHistory)
	assert.NotNil(t, ctx.ToolExecutionResults)
	assert.NotNil(t, ctx.CurrentPlan)
	assert.NotNil(t, ctx.StreamingState)
	assert.False(t, ctx.StreamingState.Active)
}

// TestAgentContext_ConversationHistory tests conversation history management.
func TestAgentContext_ConversationHistory(t *testing.T) {
	ctx := NewAgentContext()

	// Add messages to history
	ctx.ConversationHistory = append(ctx.ConversationHistory,
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi there!"),
		schema.NewHumanMessage("How are you?"),
	)

	assert.Len(t, ctx.ConversationHistory, 3)
	assert.Equal(t, "Hello", ctx.ConversationHistory[0].GetContent())
	assert.Equal(t, "Hi there!", ctx.ConversationHistory[1].GetContent())
}

// TestAgentContext_ToolResults tests tool result management.
func TestAgentContext_ToolResults(t *testing.T) {
	ctx := NewAgentContext()

	// Add tool results
	toolResult1 := ToolResult{
		ToolName:   "calculator",
		Input:      map[string]any{"a": 2, "b": 3},
		Output:     map[string]any{"result": 5},
		Duration:   10 * time.Millisecond,
		ExecutedAt: time.Now(),
	}

	toolResult2 := ToolResult{
		ToolName:   "web_search",
		Input:      map[string]any{"query": "test"},
		Output:     map[string]any{"results": []string{"result1"}},
		Duration:   100 * time.Millisecond,
		ExecutedAt: time.Now(),
	}

	ctx.ToolResults = append(ctx.ToolResults, toolResult1, toolResult2)

	assert.Len(t, ctx.ToolResults, 2)
	assert.Equal(t, "calculator", ctx.ToolResults[0].ToolName)
	assert.Equal(t, "web_search", ctx.ToolResults[1].ToolName)
}

// TestAgentContext_ToolResultWithError tests tool result with error.
func TestAgentContext_ToolResultWithError(t *testing.T) {
	ctx := NewAgentContext()

	err := assert.AnError
	toolResult := ToolResult{
		ToolName:   "failing_tool",
		Input:      map[string]any{"param": "value"},
		Output:     nil,
		Duration:   5 * time.Millisecond,
		Err:        err,
		ExecutedAt: time.Now(),
	}

	ctx.ToolResults = append(ctx.ToolResults, toolResult)

	assert.Len(t, ctx.ToolResults, 1)
	assert.Equal(t, err, ctx.ToolResults[0].Err)
	assert.Nil(t, ctx.ToolResults[0].Output)
}

// TestStreamingState_Operations tests streaming state operations.
func TestStreamingState_Operations(t *testing.T) {
	state := NewStreamingState()

	// Initially inactive
	assert.False(t, state.Active)

	// Set active
	state.Active = true
	assert.True(t, state.Active)

	// Add to buffer
	chunk := iface.AgentStreamChunk{
		Content: "test",
	}
	state.Buffer = append(state.Buffer, chunk)
	assert.Len(t, state.Buffer, 1)

	// Set interrupted
	state.Interrupted = true
	assert.True(t, state.Interrupted)

	// Update last chunk time
	state.LastChunkTime = time.Now()
	assert.False(t, state.LastChunkTime.IsZero())
}

// TestVoiceCallAgentContext_Initialization tests voice call agent context initialization.
func TestVoiceCallAgentContext_Initialization(t *testing.T) {
	ctx := NewVoiceCallAgentContext("session-1", "user-1")

	assert.Equal(t, "session-1", ctx.SessionID)
	assert.Equal(t, "user-1", ctx.UserID)
	assert.WithinDuration(t, time.Now(), ctx.StartTime, 1*time.Second)
	assert.WithinDuration(t, time.Now(), ctx.LastActivity, 1*time.Second)
	assert.Nil(t, ctx.AgentInstance)
	assert.Empty(t, ctx.ConversationHistory)
	assert.Empty(t, ctx.ToolExecutionResults)
	assert.Empty(t, ctx.CurrentPlan)
	assert.NotNil(t, ctx.StreamingState)
}

// TestAgentContext_EmptyContext tests empty context operations.
func TestAgentContext_EmptyContext(t *testing.T) {
	ctx := NewAgentContext()

	// All collections should be empty but initialized
	assert.Empty(t, ctx.ConversationHistory)
	assert.Empty(t, ctx.ToolResults)
	assert.Empty(t, ctx.CurrentPlan)

	// Should be able to append without nil pointer errors
	ctx.ConversationHistory = append(ctx.ConversationHistory, schema.NewHumanMessage("test"))
	ctx.ToolResults = append(ctx.ToolResults, ToolResult{ToolName: "test"})
	assert.Len(t, ctx.ConversationHistory, 1)
	assert.Len(t, ctx.ToolResults, 1)
}

// TestStreamingState_Reset tests resetting streaming state.
func TestStreamingState_Reset(t *testing.T) {
	state := NewStreamingState()

	// Set some values
	state.Active = true
	state.Interrupted = true
	state.Buffer = append(state.Buffer, iface.AgentStreamChunk{Content: "test"})
	state.LastChunkTime = time.Now()

	// Reset to initial state
	state.Active = false
	state.Interrupted = false
	state.Buffer = make([]iface.AgentStreamChunk, 0)
	state.LastChunkTime = time.Time{}

	assert.False(t, state.Active)
	assert.False(t, state.Interrupted)
	assert.Empty(t, state.Buffer)
	assert.True(t, state.LastChunkTime.IsZero())
}

// TestToolResult_Fields tests ToolResult field initialization.
func TestToolResult_Fields(t *testing.T) {
	result := ToolResult{
		ToolName:   "test_tool",
		Input:      map[string]any{"key": "value"},
		Output:     map[string]any{"result": "success"},
		Duration:   50 * time.Millisecond,
		Err:        nil,
		ExecutedAt: time.Now(),
	}

	assert.Equal(t, "test_tool", result.ToolName)
	assert.Equal(t, "value", result.Input["key"])
	assert.Equal(t, "success", result.Output["result"])
	assert.Equal(t, 50*time.Millisecond, result.Duration)
	assert.NoError(t, result.Err)
	assert.False(t, result.ExecutedAt.IsZero())
}

// TestVoiceCallAgentContext_WithAgentInstance tests voice call context with agent instance.
// This test is kept simple to avoid circular dependencies - full testing done in agent_instance_test.go
func TestVoiceCallAgentContext_WithAgentInstance(t *testing.T) {
	ctx := NewVoiceCallAgentContext("session-1", "user-1")

	// Initially no agent instance
	assert.Nil(t, ctx.AgentInstance)

	// Note: AgentInstance assignment and testing is done in agent_instance_test.go
	// to avoid circular dependencies between test files
}

