// Package agent provides the agent runtime for the Beluga AI framework.
// It defines the Agent interface, a composable BaseAgent, the Executor
// reasoning loop, pluggable Planners (ReAct, Reflexion), handoffs-as-tools,
// lifecycle hooks, middleware, an event bus, and deterministic workflow agents.
//
// Usage:
//
//	a := agent.New("assistant",
//	    agent.WithLLM(model),
//	    agent.WithTools(tools),
//	    agent.WithPersona(agent.Persona{Role: "helpful assistant"}),
//	)
//
//	// Synchronous
//	result, err := a.Invoke(ctx, "What is 2+2?")
//
//	// Streaming
//	for event, err := range a.Stream(ctx, "What is 2+2?") {
//	    if err != nil { break }
//	    fmt.Println(event.Text)
//	}
package agent

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// Agent is the primary interface for all agents. An agent has an identity,
// persona, tools, optional children (for orchestration), and can be invoked
// synchronously or streamed.
type Agent interface {
	// ID returns the unique identifier for this agent.
	ID() string

	// Persona returns the agent's persona (role, goal, backstory).
	Persona() Persona

	// Tools returns the tools available to this agent.
	Tools() []tool.Tool

	// Children returns child agents for orchestration.
	Children() []Agent

	// Invoke executes the agent synchronously and returns a text result.
	Invoke(ctx context.Context, input string, opts ...Option) (string, error)

	// Stream executes the agent and returns an iterator of events.
	Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error]
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
