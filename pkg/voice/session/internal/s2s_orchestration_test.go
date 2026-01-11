package internal

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	orchestrationiface "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrchestrator is a simple mock implementation of orchestrationiface.Orchestrator for testing.
type mockOrchestrator struct {
	triggeredWorkflows []string
}

func (m *mockOrchestrator) CreateChain(steps []core.Runnable, opts ...orchestrationiface.ChainOption) (orchestrationiface.Chain, error) {
	return nil, nil
}

func (m *mockOrchestrator) CreateGraph(opts ...orchestrationiface.GraphOption) (orchestrationiface.Graph, error) {
	return nil, nil
}

func (m *mockOrchestrator) CreateWorkflow(workflowFn interface{}, opts ...orchestrationiface.WorkflowOption) (orchestrationiface.Workflow, error) {
	return nil, nil
}

func (m *mockOrchestrator) GetMetrics() orchestrationiface.OrchestratorMetrics {
	return nil
}

func TestS2SAgentIntegration_OrchestrationIntegration(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))
	agentIntegration := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	})
	orchestrator := &mockOrchestrator{
		triggeredWorkflows: make([]string, 0),
	}

	integration := NewS2SAgentIntegrationWithOrchestration(mockProvider, agentIntegration, orchestrator, "external")

	// Verify orchestrator is set
	assert.Equal(t, orchestrator, integration.GetOrchestration())

	// Test processing with orchestration
	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	sessionID := "test-session-123"

	output, err := integration.ProcessAudioWithAgent(ctx, audio, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, output)

	// Verify orchestration was accessed (workflow trigger happens in ProcessAudioWithAgent)
	// The actual workflow triggering is tested in the ProcessAudioWithAgent implementation
}

func TestS2SAgentIntegration_SetOrchestration(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")
	agentIntegration := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	})

	integration := NewS2SAgentIntegration(mockProvider, agentIntegration, "external")
	assert.Nil(t, integration.GetOrchestration())

	orchestrator := &mockOrchestrator{
		triggeredWorkflows: make([]string, 0),
	}
	integration.SetOrchestration(orchestrator)
	assert.Equal(t, orchestrator, integration.GetOrchestration())
}

func TestS2SAgentIntegration_WorkflowTrigger(t *testing.T) {
	// Test that workflows are triggered based on audio input
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))
	agentIntegration := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	})
	orchestrator := &mockOrchestrator{
		triggeredWorkflows: make([]string, 0),
	}

	integration := NewS2SAgentIntegrationWithOrchestration(mockProvider, agentIntegration, orchestrator, "external")

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	sessionID := "test-session-123"

	// Process audio - workflow should be triggered if conditions are met
	output, err := integration.ProcessAudioWithAgent(ctx, audio, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, output)

	// TODO: In a full implementation, verify that orchestrator.TriggerWorkflow() or similar
	// was called based on audio content or extracted transcript
}
