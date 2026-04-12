package codeact

import (
	"sync"
)

// stateKey is the metadata key for storing ExecutionState in PlannerState.Metadata.
const stateKey = "codeact_state"

// ExecutionState tracks variables and outputs across code execution steps
// within a session. It is stored in PlannerState.Metadata under stateKey.
type ExecutionState struct {
	mu sync.RWMutex
	// Variables holds named values persisted across code steps.
	variables map[string]string
	// Outputs records the output of each code execution in order.
	outputs []StepOutput
}

// StepOutput records the code and result of a single execution step.
type StepOutput struct {
	// Language is the programming language used.
	Language string
	// Code is the source code that was executed.
	Code string
	// Output is the stdout from execution.
	Output string
	// Error is the stderr or error from execution.
	Error string
	// ExitCode is the process exit code.
	ExitCode int
}

// NewExecutionState creates a new empty ExecutionState.
func NewExecutionState() *ExecutionState {
	return &ExecutionState{
		variables: make(map[string]string),
	}
}

// SetVariable stores a named value in the execution state.
func (s *ExecutionState) SetVariable(name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.variables[name] = value
}

// GetVariable retrieves a named value from the execution state.
// Returns the value and whether it was found.
func (s *ExecutionState) GetVariable(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.variables[name]
	return v, ok
}

// Variables returns a copy of all stored variables.
func (s *ExecutionState) Variables() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]string, len(s.variables))
	for k, v := range s.variables {
		cp[k] = v
	}
	return cp
}

// AddOutput appends a step output to the execution history.
func (s *ExecutionState) AddOutput(output StepOutput) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outputs = append(s.outputs, output)
}

// Outputs returns a copy of all step outputs.
func (s *ExecutionState) Outputs() []StepOutput {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]StepOutput, len(s.outputs))
	copy(cp, s.outputs)
	return cp
}

// StepCount returns the number of code execution steps recorded.
func (s *ExecutionState) StepCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.outputs)
}

// LastOutput returns the most recent step output, or nil if none.
func (s *ExecutionState) LastOutput() *StepOutput {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.outputs) == 0 {
		return nil
	}
	out := s.outputs[len(s.outputs)-1]
	return &out
}

// GetOrCreateState retrieves the ExecutionState from metadata, or creates
// and stores a new one if not present.
func GetOrCreateState(metadata map[string]any) *ExecutionState {
	if v, ok := metadata[stateKey]; ok {
		if s, ok := v.(*ExecutionState); ok {
			return s
		}
	}
	s := NewExecutionState()
	metadata[stateKey] = s
	return s
}
