// Package internal provides internal implementation for voice session.
// This file defines agent state types for agent integration.
package internal

// AgentState represents the state of an agent in a voice session.
type AgentState string

const (
	// AgentStateIdle indicates the agent is idle and waiting for input
	AgentStateIdle AgentState = "idle"

	// AgentStateListening indicates the agent is listening for user input
	AgentStateListening AgentState = "listening"

	// AgentStateProcessing indicates the agent is processing user input
	AgentStateProcessing AgentState = "processing"

	// AgentStateStreaming indicates the agent is streaming a response
	AgentStateStreaming AgentState = "streaming"

	// AgentStateExecuting indicates the agent is executing a tool
	AgentStateExecuting AgentState = "executing_tool"

	// AgentStateSpeaking indicates the agent is speaking (TTS output)
	AgentStateSpeaking AgentState = "speaking"

	// AgentStateInterrupted indicates the agent was interrupted by new user input
	AgentStateInterrupted AgentState = "interrupted"
)

// ValidAgentStateTransitions defines valid state transitions for agents.
// This helps ensure state machine correctness.
var ValidAgentStateTransitions = map[AgentState][]AgentState{
	AgentStateIdle:        {AgentStateListening},
	AgentStateListening:   {AgentStateProcessing, AgentStateIdle},
	AgentStateProcessing:  {AgentStateStreaming, AgentStateExecuting, AgentStateIdle},
	AgentStateStreaming:   {AgentStateSpeaking, AgentStateInterrupted, AgentStateProcessing, AgentStateIdle},
	AgentStateExecuting:   {AgentStateProcessing, AgentStateInterrupted, AgentStateIdle},
	AgentStateSpeaking:    {AgentStateInterrupted, AgentStateIdle, AgentStateListening},
	AgentStateInterrupted: {AgentStateProcessing, AgentStateIdle},
}

// IsValidTransition checks if a transition from one state to another is valid.
func IsValidTransition(from, to AgentState) bool {
	validTransitions, exists := ValidAgentStateTransitions[from]
	if !exists {
		return false
	}
	for _, valid := range validTransitions {
		if valid == to {
			return true
		}
	}
	return false
}

