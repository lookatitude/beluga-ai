package agent

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// AgentMetadata exposes an agent's static identity and composition. Consumers
// that only need to introspect an agent (for routing, telemetry, rendering an
// AgentCard) can depend on AgentMetadata rather than the full Agent.
type AgentMetadata interface {
	// ID returns the unique identifier for this agent.
	ID() string

	// Persona returns the agent's persona (role, goal, backstory).
	Persona() Persona

	// Tools returns the tools available to this agent.
	Tools() []tool.Tool

	// Children returns child agents for orchestration.
	Children() []Agent
}

// AgentExecutor is the runtime-behaviour surface of an agent. Consumers that
// only need to run an agent (middleware, retry wrappers, orchestration
// patterns) can depend on AgentExecutor rather than the full Agent.
type AgentExecutor interface {
	// Invoke executes the agent synchronously and returns a text result.
	Invoke(ctx context.Context, input string, opts ...Option) (string, error)

	// Stream executes the agent and returns an iterator of events.
	Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error]
}

// Agent is the primary interface for all agents. An agent has an identity,
// persona, tools, optional children (for orchestration), and can be invoked
// synchronously or streamed.
//
// Agent is composed from two smaller interfaces (AgentMetadata and
// AgentExecutor) so consumers can depend on the narrowest surface they need.
// Every existing implementation of Agent automatically satisfies both
// sub-interfaces; no migration is required.
type Agent interface {
	AgentMetadata
	AgentExecutor
}

// EventType identifies the kind of event emitted during agent execution.
type EventType string

const (
	// EventText indicates a text chunk from the agent's response.
	EventText EventType = "text"
	// EventToolCall indicates the agent is requesting a tool invocation.
	EventToolCall EventType = "tool_call"
	// EventToolResult indicates the result of a tool invocation.
	EventToolResult EventType = "tool_result"
	// EventHandoff indicates an agent-to-agent transfer.
	EventHandoff EventType = "handoff"
	// EventDone indicates the agent has finished execution.
	EventDone EventType = "done"
	// EventError indicates an error occurred during execution.
	EventError EventType = "error"
)

// Event represents a discrete event emitted during agent execution.
type Event struct {
	// Type identifies the kind of event.
	Type EventType
	// Text carries the text content for EventText events.
	Text string
	// ToolCall carries the tool call for EventToolCall events.
	ToolCall *schema.ToolCall
	// ToolResult carries the tool result for EventToolResult events.
	ToolResult *tool.Result
	// AgentID identifies which agent emitted this event.
	AgentID string
	// Metadata holds arbitrary key-value pairs associated with this event.
	Metadata map[string]any
}
