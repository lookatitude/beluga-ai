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
	ToolName    string
	Input       map[string]any
	Output      map[string]any
	Duration    time.Duration
	Err         error
	ExecutedAt  time.Time
}

// AgentContext extends voice session context with agent-specific state.
// It maintains conversation history, tool results, and current plan state
// for agent-based voice sessions.
type AgentContext struct {
	// ConversationHistory stores the message history for the conversation
	ConversationHistory []schema.Message

	// ToolResults stores the results of tool executions
	ToolResults []ToolResult

	// CurrentPlan stores the current execution plan if the agent is planning
	CurrentPlan []schema.Step

	// StreamingActive indicates whether streaming is currently active
	StreamingActive bool

	// LastInterruption stores the timestamp of the last interruption
	LastInterruption time.Time
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
	// Active indicates whether streaming is currently active
	Active bool

	// CurrentStream is the channel receiving streaming chunks
	CurrentStream <-chan iface.AgentStreamChunk

	// Buffer stores buffered chunks waiting to be processed
	Buffer []iface.AgentStreamChunk

	// LastChunkTime is the timestamp when the last chunk was received
	LastChunkTime time.Time

	// Interrupted indicates whether the stream was interrupted
	Interrupted bool
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
	// SessionID is the unique identifier for this session
	SessionID string

	// UserID is the identifier for the user in this session
	UserID string

	// StartTime is when the session started
	StartTime time.Time

	// LastActivity is the timestamp of the last activity
	LastActivity time.Time

	// AgentInstance is the agent instance associated with this session
	AgentInstance *AgentInstance

	// ConversationHistory stores the message history
	ConversationHistory []schema.Message

	// ToolExecutionResults stores tool execution results
	ToolExecutionResults []ToolResult

	// CurrentPlan stores the current execution plan
	CurrentPlan []schema.Step

	// StreamingState tracks the streaming state
	StreamingState *StreamingState
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

