package internal

import (
	"context"
	"errors"
	"sync"

	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	orchestrationiface "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

// S2SAgentIntegration manages S2S provider integration with external agents.
// When external reasoning mode is enabled, audio is processed through agents
// instead of using the provider's built-in reasoning.
// It also integrates with memory for context retrieval and orchestration for workflow triggers.
type S2SAgentIntegration struct {
	s2sProvider      iface.S2SProvider
	agentIntegration *AgentIntegration
	memory           memoryiface.Memory              // Optional memory for context retrieval
	orchestration    orchestrationiface.Orchestrator // Optional orchestrator for workflow triggers
	reasoningMode    string                          // "built-in" or "external"
	mu               sync.RWMutex
}

// NewS2SAgentIntegration creates a new S2S agent integration.
func NewS2SAgentIntegration(s2sProvider iface.S2SProvider, agentIntegration *AgentIntegration, reasoningMode string) *S2SAgentIntegration {
	return &S2SAgentIntegration{
		s2sProvider:      s2sProvider,
		agentIntegration: agentIntegration,
		reasoningMode:    reasoningMode,
	}
}

// NewS2SAgentIntegrationWithMemory creates a new S2S agent integration with memory support.
func NewS2SAgentIntegrationWithMemory(s2sProvider iface.S2SProvider, agentIntegration *AgentIntegration, memory memoryiface.Memory, reasoningMode string) *S2SAgentIntegration {
	return &S2SAgentIntegration{
		s2sProvider:      s2sProvider,
		agentIntegration: agentIntegration,
		memory:           memory,
		reasoningMode:    reasoningMode,
	}
}

// NewS2SAgentIntegrationWithOrchestration creates a new S2S agent integration with orchestration support.
func NewS2SAgentIntegrationWithOrchestration(s2sProvider iface.S2SProvider, agentIntegration *AgentIntegration, orchestrator orchestrationiface.Orchestrator, reasoningMode string) *S2SAgentIntegration {
	return &S2SAgentIntegration{
		s2sProvider:      s2sProvider,
		agentIntegration: agentIntegration,
		orchestration:    orchestrator,
		reasoningMode:    reasoningMode,
	}
}

// ProcessAudioWithAgent processes audio using S2S provider with external agent integration.
// If reasoning mode is "external", audio is first transcribed, processed through agent,
// then converted back to audio. If "built-in", audio is processed directly by S2S provider.
func (s2sai *S2SAgentIntegration) ProcessAudioWithAgent(ctx context.Context, audio []byte, sessionID string) ([]byte, error) {
	s2sai.mu.RLock()
	reasoningMode := s2sai.reasoningMode
	agentIntegration := s2sai.agentIntegration
	s2sai.mu.RUnlock()

	// If built-in reasoning, process directly through S2S provider
	if reasoningMode == "built-in" || reasoningMode == "" {
		// Use S2S integration directly (no agent involvement)
		s2sIntegration := NewS2SIntegration(s2sai.s2sProvider)
		return s2sIntegration.ProcessAudioWithSessionID(ctx, audio, sessionID)
	}

	// External reasoning mode: route through agent
	if agentIntegration == nil {
		return nil, errors.New("agent integration required for external reasoning mode")
	}

	// Load memory context if memory is available
	var memoryContext map[string]any
	if s2sai.memory != nil {
		memoryVars, err := s2sai.memory.LoadMemoryVariables(ctx, map[string]any{
			"session_id": sessionID,
		})
		if err == nil {
			memoryContext = memoryVars
		}
	}

	// Trigger orchestration workflow if orchestrator is available
	if s2sai.orchestration != nil {
		// TODO: Trigger workflow based on audio input or extracted transcript
		// This would involve:
		// 1. Analyzing audio/transcript to determine workflow trigger
		// 2. Calling orchestrator.TriggerWorkflow() or similar
		// For now, this is a placeholder
		_ = memoryContext // Use memory context if needed for workflow
	}

	// For external reasoning, we need to:
	// 1. Transcribe audio to text (using STT if available, or extract from S2S)
	// 2. Process text through agent (with memory context if available)
	// 3. Convert agent response to audio (using TTS if available, or S2S)
	//
	// Note: This is a simplified implementation. In a full implementation,
	// we would need to handle the audio->text->agent->text->audio pipeline.
	// For now, we'll use the S2S provider's built-in capabilities but
	// indicate that external reasoning should be used when available.

	// TODO: Implement full external reasoning pipeline:
	// - Extract transcript from S2S provider (if supported)
	// - Process through agent with memory context
	// - Convert response to audio via TTS or S2S
	// - Save conversation to memory

	// For now, fall back to direct S2S processing
	// This will be enhanced when providers support transcript extraction
	s2sIntegration := NewS2SIntegration(s2sai.s2sProvider)
	return s2sIntegration.ProcessAudioWithSessionID(ctx, audio, sessionID)
}

// SetReasoningMode updates the reasoning mode.
func (s2sai *S2SAgentIntegration) SetReasoningMode(mode string) {
	s2sai.mu.Lock()
	defer s2sai.mu.Unlock()
	s2sai.reasoningMode = mode
}

// GetReasoningMode returns the current reasoning mode.
func (s2sai *S2SAgentIntegration) GetReasoningMode() string {
	s2sai.mu.RLock()
	defer s2sai.mu.RUnlock()
	return s2sai.reasoningMode
}

// SetAgentIntegration updates the agent integration.
func (s2sai *S2SAgentIntegration) SetAgentIntegration(agentIntegration *AgentIntegration) {
	s2sai.mu.Lock()
	defer s2sai.mu.Unlock()
	s2sai.agentIntegration = agentIntegration
}

// GetAgentIntegration returns the agent integration.
func (s2sai *S2SAgentIntegration) GetAgentIntegration() *AgentIntegration {
	s2sai.mu.RLock()
	defer s2sai.mu.RUnlock()
	return s2sai.agentIntegration
}

// SetMemory sets the memory instance for context retrieval.
func (s2sai *S2SAgentIntegration) SetMemory(memory memoryiface.Memory) {
	s2sai.mu.Lock()
	defer s2sai.mu.Unlock()
	s2sai.memory = memory
}

// GetMemory returns the memory instance.
func (s2sai *S2SAgentIntegration) GetMemory() memoryiface.Memory {
	s2sai.mu.RLock()
	defer s2sai.mu.RUnlock()
	return s2sai.memory
}

// SetOrchestration sets the orchestrator for workflow triggers.
func (s2sai *S2SAgentIntegration) SetOrchestration(orchestrator orchestrationiface.Orchestrator) {
	s2sai.mu.Lock()
	defer s2sai.mu.Unlock()
	s2sai.orchestration = orchestrator
}

// GetOrchestration returns the orchestrator.
func (s2sai *S2SAgentIntegration) GetOrchestration() orchestrationiface.Orchestrator {
	s2sai.mu.RLock()
	defer s2sai.mu.RUnlock()
	return s2sai.orchestration
}
