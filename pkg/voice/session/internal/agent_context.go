// Package internal provides internal implementation for voice session.
// This file defines agent context types for agent integration.
package internal

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ToolResult represents the result of a tool execution.
// This is used in agent context to track tool execution history.
type ToolResult struct {
	ExecutedAt time.Time
	Err        error
	Input      map[string]any
	Output     map[string]any
	ToolName   string
	Duration   time.Duration
}

// AgentContext extends voice session context with agent-specific state.
// It maintains conversation history, tool results, and current plan state
// for agent-based voice sessions.
type AgentContext struct {
	LastInterruption    time.Time
	ConversationHistory []schema.Message
	ToolResults         []ToolResult
	CurrentPlan         []schema.Step
	StreamingActive     bool
}

// NewAgentContext creates a new agent context with initialized fields.
func NewAgentContext() *AgentContext {
	return &AgentContext{
		ConversationHistory: make([]schema.Message, 0),
		ToolResults:         make([]ToolResult, 0),
		CurrentPlan:         make([]schema.Step, 0),
		StreamingActive:     false,
		LastInterruption:    time.Time{},
	}
}

// StreamingState represents the current streaming state of an agent.
// It tracks active streams, buffered chunks, and interruption status.
type StreamingState struct {
	LastChunkTime time.Time
	CurrentStream <-chan iface.AgentStreamChunk
	Buffer        []iface.AgentStreamChunk
	Active        bool
	Interrupted   bool
}

// NewStreamingState creates a new streaming state with initialized fields.
func NewStreamingState() *StreamingState {
	return &StreamingState{
		Active:        false,
		CurrentStream: nil,
		Buffer:        make([]iface.AgentStreamChunk, 0),
		LastChunkTime: time.Time{},
		Interrupted:   false,
	}
}

// VoiceCallAgentContext represents the complete context for a voice call with agent integration.
// This extends the base voice session context with agent-specific fields.
type VoiceCallAgentContext struct {
	StartTime            time.Time
	LastActivity         time.Time
	AgentInstance        *AgentInstance
	StreamingState       *StreamingState
	SessionID            string
	UserID               string
	ConversationHistory  []schema.Message
	ToolExecutionResults []ToolResult
	CurrentPlan          []schema.Step
}

// NewVoiceCallAgentContext creates a new voice call agent context.
func NewVoiceCallAgentContext(sessionID, userID string) *VoiceCallAgentContext {
	return &VoiceCallAgentContext{
		SessionID:            sessionID,
		UserID:               userID,
		StartTime:            time.Now(),
		LastActivity:         time.Now(),
		AgentInstance:        nil,
		ConversationHistory:  make([]schema.Message, 0),
		ToolExecutionResults: make([]ToolResult, 0),
		CurrentPlan:          make([]schema.Step, 0),
		StreamingState:       NewStreamingState(),
	}
}
