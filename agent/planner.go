package agent

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// Planner is the interface for agent reasoning strategies. A planner decides
// what actions the agent should take given the current state, and can replan
// based on observations from executed actions.
type Planner interface {
	// Plan generates a list of actions given the current planner state.
	Plan(ctx context.Context, state PlannerState) ([]Action, error)

	// Replan generates updated actions based on new observations. Called after
	// actions have been executed and observations collected.
	Replan(ctx context.Context, state PlannerState) ([]Action, error)
}

// PlannerState carries all context needed by a planner to make decisions.
type PlannerState struct {
	// Input is the original user input.
	Input string
	// Messages is the conversation history.
	Messages []schema.Message
	// Tools is the set of tools available to the agent.
	Tools []tool.Tool
	// Observations contains results from previously executed actions.
	Observations []Observation
	// Iteration is the current reasoning loop iteration (0-based).
	Iteration int
	// Metadata carries planner-specific state between iterations.
	Metadata map[string]any
}

// ActionType identifies the kind of action the planner wants to take.
type ActionType string

const (
	// ActionTool indicates the action is a tool invocation.
	ActionTool ActionType = "tool"
	// ActionRespond indicates the action is a text response to the user.
	ActionRespond ActionType = "respond"
	// ActionFinish indicates the agent should finish with a final answer.
	ActionFinish ActionType = "finish"
	// ActionHandoff indicates transfer to another agent.
	ActionHandoff ActionType = "handoff"
)

// Action describes a single step the planner wants the executor to take.
type Action struct {
	// Type identifies the kind of action.
	Type ActionType
	// ToolCall carries the tool call details for ActionTool.
	ToolCall *schema.ToolCall
	// Message carries the text for ActionRespond or ActionFinish.
	Message string
	// Metadata holds action-specific data.
	Metadata map[string]any
}

// Observation records the outcome of executing an action.
type Observation struct {
	// Action is the action that was executed.
	Action Action
	// Result is the tool result, if the action was a tool call.
	Result *tool.Result
	// Error is any error that occurred during execution.
	Error error
	// Latency is how long the action took to execute.
	Latency time.Duration
}
