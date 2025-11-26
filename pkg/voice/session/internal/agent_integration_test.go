package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentIntegration(t *testing.T) {
	ai := NewAgentIntegration(nil)
	assert.NotNil(t, ai)
	assert.Nil(t, ai.agentCallback)
}

func TestAgentIntegration_GenerateResponse(t *testing.T) {
	ai := NewAgentIntegration(nil)

	ctx := context.Background()
	transcript := "Hello, how are you?"

	// Without callback, should return error
	response, err := ai.GenerateResponse(ctx, transcript)
	require.Error(t, err)
	assert.Empty(t, response)
	assert.Contains(t, err.Error(), "agent callback not set")

	// Set callback
	called := false
	var receivedTranscript string
	ai.SetAgentCallback(func(ctx context.Context, t string) (string, error) {
		called = true
		receivedTranscript = t
		return "I'm doing well, thank you!", nil
	})

	response, err = ai.GenerateResponse(ctx, transcript)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, transcript, receivedTranscript)
	assert.Equal(t, "I'm doing well, thank you!", response)
}

func TestAgentIntegration_GenerateResponse_Error(t *testing.T) {
	ai := NewAgentIntegration(nil)

	expectedErr := errors.New("agent error")
	ai.SetAgentCallback(func(ctx context.Context, t string) (string, error) {
		return "", expectedErr
	})

	ctx := context.Background()
	_, err := ai.GenerateResponse(ctx, "test")
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestAgentIntegration_SetAgentCallback(t *testing.T) {
	ai := NewAgentIntegration(nil)

	called := false
	callback := func(ctx context.Context, transcript string) (string, error) {
		called = true
		return "response", nil
	}

	ai.SetAgentCallback(callback)

	ctx := context.Background()
	_, err := ai.GenerateResponse(ctx, "test")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAgentIntegration_GenerateResponse_ContextCancellation(t *testing.T) {
	ai := NewAgentIntegration(nil)

	ai.SetAgentCallback(func(ctx context.Context, t string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ai.GenerateResponse(ctx, "test")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
