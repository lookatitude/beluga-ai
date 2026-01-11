package internal

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewS2SAgentIntegration(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")
	agentIntegration := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	})

	integration := NewS2SAgentIntegration(mockProvider, agentIntegration, "external")
	assert.NotNil(t, integration)
	assert.Equal(t, "external", integration.GetReasoningMode())
}

func TestS2SAgentIntegration_ProcessAudioWithAgent_BuiltIn(t *testing.T) {
	// Test built-in reasoning mode (direct S2S processing)
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))
	agentIntegration := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	})

	integration := NewS2SAgentIntegration(mockProvider, agentIntegration, "built-in")

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	sessionID := "test-session-123"

	output, err := integration.ProcessAudioWithAgent(ctx, audio, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output)
}

func TestS2SAgentIntegration_ProcessAudioWithAgent_External(t *testing.T) {
	// Test external reasoning mode (should route through agent, but currently falls back to direct S2S)
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))
	agentIntegration := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "agent response", nil
	})

	integration := NewS2SAgentIntegration(mockProvider, agentIntegration, "external")

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	sessionID := "test-session-123"

	output, err := integration.ProcessAudioWithAgent(ctx, audio, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, output)
	// Note: Currently falls back to direct S2S processing until full external pipeline is implemented
}

func TestS2SAgentIntegration_ProcessAudioWithAgent_External_NoAgent(t *testing.T) {
	// Test external reasoning mode without agent integration (should error)
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")

	integration := NewS2SAgentIntegration(mockProvider, nil, "external")

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	sessionID := "test-session-123"

	output, err := integration.ProcessAudioWithAgent(ctx, audio, sessionID)
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "agent integration required")
}

func TestS2SAgentIntegration_SetReasoningMode(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")
	agentIntegration := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	})

	integration := NewS2SAgentIntegration(mockProvider, agentIntegration, "built-in")
	assert.Equal(t, "built-in", integration.GetReasoningMode())

	integration.SetReasoningMode("external")
	assert.Equal(t, "external", integration.GetReasoningMode())
}

func TestS2SAgentIntegration_SetAgentIntegration(t *testing.T) {
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider")
	agentIntegration1 := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "response1", nil
	})
	agentIntegration2 := NewAgentIntegration(func(ctx context.Context, transcript string) (string, error) {
		return "response2", nil
	})

	integration := NewS2SAgentIntegration(mockProvider, agentIntegration1, "external")
	assert.Equal(t, agentIntegration1, integration.GetAgentIntegration())

	integration.SetAgentIntegration(agentIntegration2)
	assert.Equal(t, agentIntegration2, integration.GetAgentIntegration())
}
