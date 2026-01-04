// Package internal provides comprehensive tests for agent state management.
// T163: Add missing test cases for edge cases in agent integration
package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAgentState_Constants tests that all agent state constants are defined.
func TestAgentState_Constants(t *testing.T) {
	assert.Equal(t, AgentStateIdle, AgentState("idle"))
	assert.Equal(t, AgentStateListening, AgentState("listening"))
	assert.Equal(t, AgentStateProcessing, AgentState("processing"))
	assert.Equal(t, AgentStateStreaming, AgentState("streaming"))
	assert.Equal(t, AgentStateExecuting, AgentState("executing_tool"))
	assert.Equal(t, AgentStateSpeaking, AgentState("speaking"))
	assert.Equal(t, AgentStateInterrupted, AgentState("interrupted"))
}

// TestIsValidTransition_ValidTransitions tests all valid state transitions.
func TestIsValidTransition_ValidTransitions(t *testing.T) {
	tests := []struct {
		name      string
		from      AgentState
		to        AgentState
		wantValid bool
	}{
		// From Idle
		{
			name:      "idle -> listening",
			from:      AgentStateIdle,
			to:        AgentStateListening,
			wantValid: true,
		},
		// From Listening
		{
			name:      "listening -> processing",
			from:      AgentStateListening,
			to:        AgentStateProcessing,
			wantValid: true,
		},
		{
			name:      "listening -> idle",
			from:      AgentStateListening,
			to:        AgentStateIdle,
			wantValid: true,
		},
		// From Processing
		{
			name:      "processing -> streaming",
			from:      AgentStateProcessing,
			to:        AgentStateStreaming,
			wantValid: true,
		},
		{
			name:      "processing -> executing",
			from:      AgentStateProcessing,
			to:        AgentStateExecuting,
			wantValid: true,
		},
		{
			name:      "processing -> idle",
			from:      AgentStateProcessing,
			to:        AgentStateIdle,
			wantValid: true,
		},
		// From Streaming
		{
			name:      "streaming -> speaking",
			from:      AgentStateStreaming,
			to:        AgentStateSpeaking,
			wantValid: true,
		},
		{
			name:      "streaming -> interrupted",
			from:      AgentStateStreaming,
			to:        AgentStateInterrupted,
			wantValid: true,
		},
		{
			name:      "streaming -> processing",
			from:      AgentStateStreaming,
			to:        AgentStateProcessing,
			wantValid: true,
		},
		{
			name:      "streaming -> idle",
			from:      AgentStateStreaming,
			to:        AgentStateIdle,
			wantValid: true,
		},
		// From Executing
		{
			name:      "executing -> processing",
			from:      AgentStateExecuting,
			to:        AgentStateProcessing,
			wantValid: true,
		},
		{
			name:      "executing -> interrupted",
			from:      AgentStateExecuting,
			to:        AgentStateInterrupted,
			wantValid: true,
		},
		{
			name:      "executing -> idle",
			from:      AgentStateExecuting,
			to:        AgentStateIdle,
			wantValid: true,
		},
		// From Speaking
		{
			name:      "speaking -> interrupted",
			from:      AgentStateSpeaking,
			to:        AgentStateInterrupted,
			wantValid: true,
		},
		{
			name:      "speaking -> idle",
			from:      AgentStateSpeaking,
			to:        AgentStateIdle,
			wantValid: true,
		},
		{
			name:      "speaking -> listening",
			from:      AgentStateSpeaking,
			to:        AgentStateListening,
			wantValid: true,
		},
		// From Interrupted
		{
			name:      "interrupted -> processing",
			from:      AgentStateInterrupted,
			to:        AgentStateProcessing,
			wantValid: true,
		},
		{
			name:      "interrupted -> idle",
			from:      AgentStateInterrupted,
			to:        AgentStateIdle,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := IsValidTransition(tt.from, tt.to)
			assert.Equal(t, tt.wantValid, isValid, "Transition %s -> %s should be %v", tt.from, tt.to, tt.wantValid)
		})
	}
}

// TestIsValidTransition_InvalidTransitions tests invalid state transitions.
func TestIsValidTransition_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name      string
		from      AgentState
		to        AgentState
		wantValid bool
	}{
		{
			name:      "idle -> streaming (invalid)",
			from:      AgentStateIdle,
			to:        AgentStateStreaming,
			wantValid: false,
		},
		{
			name:      "idle -> executing (invalid)",
			from:      AgentStateIdle,
			to:        AgentStateExecuting,
			wantValid: false,
		},
		{
			name:      "idle -> speaking (invalid)",
			from:      AgentStateIdle,
			to:        AgentStateSpeaking,
			wantValid: false,
		},
		{
			name:      "listening -> streaming (invalid)",
			from:      AgentStateListening,
			to:        AgentStateStreaming,
			wantValid: false,
		},
		{
			name:      "processing -> speaking (invalid)",
			from:      AgentStateProcessing,
			to:        AgentStateSpeaking,
			wantValid: false,
		},
		{
			name:      "same state transition (invalid)",
			from:      AgentStateIdle,
			to:        AgentStateIdle,
			wantValid: false,
		},
		{
			name:      "same state transition - streaming",
			from:      AgentStateStreaming,
			to:        AgentStateStreaming,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := IsValidTransition(tt.from, tt.to)
			assert.Equal(t, tt.wantValid, isValid, "Transition %s -> %s should be %v", tt.from, tt.to, tt.wantValid)
		})
	}
}

// TestIsValidTransition_UnknownState tests transitions from unknown states.
func TestIsValidTransition_UnknownState(t *testing.T) {
	unknownState := AgentState("unknown_state")
	validState := AgentStateIdle

	// From unknown state should always be invalid
	assert.False(t, IsValidTransition(unknownState, validState))
	assert.False(t, IsValidTransition(validState, unknownState))
}

// TestValidAgentStateTransitions_Completeness tests that all states have transitions defined.
func TestValidAgentStateTransitions_Completeness(t *testing.T) {
	allStates := []AgentState{
		AgentStateIdle,
		AgentStateListening,
		AgentStateProcessing,
		AgentStateStreaming,
		AgentStateExecuting,
		AgentStateSpeaking,
		AgentStateInterrupted,
	}

	for _, state := range allStates {
		t.Run(string(state), func(t *testing.T) {
			transitions, exists := ValidAgentStateTransitions[state]
			assert.True(t, exists, "State %s should have transitions defined", state)
			assert.NotEmpty(t, transitions, "State %s should have at least one valid transition", state)
		})
	}
}

// TestValidAgentStateTransitions_NoCycles tests that there are no obvious cycles.
func TestValidAgentStateTransitions_NoCycles(t *testing.T) {
	// Test that we can't go from idle -> idle directly
	assert.False(t, IsValidTransition(AgentStateIdle, AgentStateIdle))

	// Test that we can't go from streaming -> streaming directly
	assert.False(t, IsValidTransition(AgentStateStreaming, AgentStateStreaming))
}

// TestIsValidTransition_AllCombinations tests all possible state combinations.
func TestIsValidTransition_AllCombinations(t *testing.T) {
	allStates := []AgentState{
		AgentStateIdle,
		AgentStateListening,
		AgentStateProcessing,
		AgentStateStreaming,
		AgentStateExecuting,
		AgentStateSpeaking,
		AgentStateInterrupted,
	}

	// Test all combinations
	for _, from := range allStates {
		for _, to := range allStates {
			if from == to {
				// Self-transitions should be invalid
				assert.False(t, IsValidTransition(from, to), "Self-transition %s -> %s should be invalid", from, to)
			} else {
				// Check if transition is defined in ValidAgentStateTransitions
				validTransitions, exists := ValidAgentStateTransitions[from]
				if exists {
					isValid := false
					for _, valid := range validTransitions {
						if valid == to {
							isValid = true
							break
						}
					}
					result := IsValidTransition(from, to)
					assert.Equal(t, isValid, result, "Transition %s -> %s: expected %v, got %v", from, to, isValid, result)
				}
			}
		}
	}
}
