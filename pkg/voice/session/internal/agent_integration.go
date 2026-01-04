package internal

import (
	"context"
	"errors"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// AgentIntegration manages agent package integration for generating responses.
// It supports both legacy callback-based integration and new agent instance-based integration.
type AgentIntegration struct {
	// Legacy callback support (deprecated: use agentInstance)
	agentCallback func(ctx context.Context, transcript string) (string, error)

	// Agent instance for streaming agent integration
	agentInstance *AgentInstance

	mu sync.RWMutex
}

// NewAgentIntegration creates a new agent integration with a callback.
// Deprecated: Use NewAgentIntegrationWithInstance instead.
func NewAgentIntegration(agentCallback func(ctx context.Context, transcript string) (string, error)) *AgentIntegration {
	return &AgentIntegration{
		agentCallback: agentCallback,
	}
}

// NewAgentIntegrationWithInstance creates a new agent integration with an agent instance.
func NewAgentIntegrationWithInstance(agent iface.StreamingAgent, config schema.AgentConfig) *AgentIntegration {
	instance := NewAgentInstance(agent, config)
	return &AgentIntegration{
		agentInstance: instance,
	}
}

// GetAgentInstance returns the agent instance (thread-safe).
func (ai *AgentIntegration) GetAgentInstance() *AgentInstance {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.agentInstance
}

// SetAgentInstance sets the agent instance (thread-safe).
func (ai *AgentIntegration) SetAgentInstance(instance *AgentInstance) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.agentInstance = instance
	// Clear callback when instance is set
	ai.agentCallback = nil
}

// GenerateResponse generates a response from the agent.
// This supports both callback-based and instance-based responses.
func (ai *AgentIntegration) GenerateResponse(ctx context.Context, transcript string) (string, error) {
	ai.mu.RLock()
	instance := ai.agentInstance
	callback := ai.agentCallback
	ai.mu.RUnlock()

	// Prefer agent instance over callback
	if instance != nil && instance.Agent != nil {
		// Use Invoke for non-streaming response
		if runnable, ok := instance.Agent.(interface {
			Invoke(ctx context.Context, input any, config map[string]any) (any, error)
		}); ok {
			result, err := runnable.Invoke(ctx, transcript, nil)
			if err != nil {
				return "", err
			}
			if str, ok := result.(string); ok {
				return str, nil
			}
			return "", errors.New("agent returned non-string result")
		}
		return "", errors.New("agent does not support Invoke method")
	}

	// Fallback to callback
	if callback != nil {
		return callback(ctx, transcript)
	}

	return "", errors.New("neither agent instance nor callback is set")
}

// SetAgentCallback sets the agent callback (thread-safe).
// Deprecated: Use SetAgentInstance instead.
func (ai *AgentIntegration) SetAgentCallback(callback func(ctx context.Context, transcript string) (string, error)) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.agentCallback = callback
	// Clear instance when callback is set
	ai.agentInstance = nil
}
