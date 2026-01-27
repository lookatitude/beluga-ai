// Package internal provides internal implementation for voice session.
// This file defines agent instance types for agent integration.
package internal

import (
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// AgentInstance represents an agent instance integrated with a voice session.
// It manages the agent's lifecycle, state, and context within the session.
type AgentInstance struct {
	Agent   iface.StreamingAgent
	Context *AgentContext
	State   AgentState
	Config  schema.AgentConfig
	mu      sync.RWMutex
}

// NewAgentInstance creates a new agent instance with the provided agent and config.
func NewAgentInstance(agent iface.StreamingAgent, config schema.AgentConfig) *AgentInstance {
	return &AgentInstance{
		Agent:   agent,
		Config:  config,
		Context: NewAgentContext(),
		State:   AgentStateIdle,
	}
}

// GetState returns the current agent state (thread-safe).
func (ai *AgentInstance) GetState() AgentState {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.State
}

// SetState sets the agent state (thread-safe).
// It validates the state transition before applying it.
func (ai *AgentInstance) SetState(newState AgentState) error {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	oldState := ai.State
	if !IsValidTransition(oldState, newState) {
		return &AgentStateError{
			From: oldState,
			To:   newState,
			Msg:  "invalid state transition",
		}
	}

	ai.State = newState
	return nil
}

// GetContext returns the agent context (thread-safe).
func (ai *AgentInstance) GetContext() *AgentContext {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.Context
}

// UpdateContext updates the agent context (thread-safe).
func (ai *AgentInstance) UpdateContext(updater func(*AgentContext)) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	updater(ai.Context)
}

// AgentStateError represents an error in agent state transitions.
type AgentStateError struct {
	From AgentState
	To   AgentState
	Msg  string
}

func (e *AgentStateError) Error() string {
	return e.Msg + ": " + string(e.From) + " -> " + string(e.To)
}
