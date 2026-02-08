package schema

import "time"

// StreamChunk represents an incremental piece of a streaming model response.
// Each chunk may contain a text delta, tool call updates, and/or completion
// metadata such as the finish reason and token usage.
type StreamChunk struct {
	// Delta is the incremental text content in this chunk.
	Delta string
	// ToolCalls contains any incremental tool call data in this chunk.
	ToolCalls []ToolCall
	// FinishReason indicates why generation stopped (e.g., "stop", "tool_calls", "length").
	// Empty if generation is still in progress.
	FinishReason string
	// Usage contains token usage statistics. May be nil for intermediate chunks.
	Usage *Usage
	// ModelID identifies the model that produced this chunk.
	ModelID string
}

// AgentEvent represents a discrete event emitted during agent execution.
// Events provide visibility into the agent's reasoning, tool calls, and state changes.
type AgentEvent struct {
	// Type identifies the kind of event (e.g., "agent_start", "tool_call", "thought", "handoff").
	Type string
	// AgentID identifies which agent emitted this event.
	AgentID string
	// Payload carries event-specific data. The concrete type depends on the event Type.
	Payload any
	// Timestamp is when the event was created.
	Timestamp time.Time
}
