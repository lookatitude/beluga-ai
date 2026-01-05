package s2s

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockS2SProviderForManager struct {
	name        string
	shouldError bool
}

func (m *mockS2SProviderForManager) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
	if m.shouldError {
		return nil, errors.New("provider error")
	}
	return &internal.AudioOutput{
		Data:     []byte{1, 2, 3},
		Provider: m.name,
	}, nil
}

func (m *mockS2SProviderForManager) Name() string {
	return m.name
}

func TestNewProviderManager(t *testing.T) {
	primary := &mockS2SProviderForManager{name: "primary"}
	fallbacks := []iface.S2SProvider{
		&mockS2SProviderForManager{name: "fallback1"},
	}

	manager, err := NewProviderManager(primary, fallbacks)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, primary, manager.GetPrimaryProvider())
	assert.Equal(t, fallbacks, manager.GetFallbackProviders())
}

func TestNewProviderManager_NilPrimary(t *testing.T) {
	manager, err := NewProviderManager(nil, nil)
	require.Error(t, err)
	assert.Nil(t, manager)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestProviderManager_GetCurrentProvider(t *testing.T) {
	primary := &mockS2SProviderForManager{name: "primary"}
	fallbacks := []iface.S2SProvider{
		&mockS2SProviderForManager{name: "fallback1"},
	}

	manager, err := NewProviderManager(primary, fallbacks)
	require.NoError(t, err)

	// Initially should be primary
	provider := manager.GetCurrentProvider()
	assert.Equal(t, primary, provider)
	assert.False(t, manager.IsUsingFallback())
	assert.Equal(t, "primary", manager.GetCurrentProviderName())
}

func TestProviderManager_Process_Success(t *testing.T) {
	primary := &mockS2SProviderForManager{name: "primary", shouldError: false}
	fallbacks := []iface.S2SProvider{
		&mockS2SProviderForManager{name: "fallback1"},
	}

	manager, err := NewProviderManager(primary, fallbacks)
	require.NoError(t, err)

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3},
	}
	convCtx := &internal.ConversationContext{
		ConversationID: "test",
		SessionID:      "test",
	}

	ctx := context.Background()
	output, err := manager.Process(ctx, input, convCtx)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.False(t, manager.IsUsingFallback())
}

func TestProviderManager_Process_Fallback(t *testing.T) {
	primary := &mockS2SProviderForManager{name: "primary", shouldError: true}
	fallbacks := []iface.S2SProvider{
		&mockS2SProviderForManager{name: "fallback1", shouldError: false},
	}

	manager, err := NewProviderManager(primary, fallbacks)
	require.NoError(t, err)

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3},
	}
	convCtx := &internal.ConversationContext{
		ConversationID: "test",
		SessionID:      "test",
	}

	ctx := context.Background()
	output, err := manager.Process(ctx, input, convCtx)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.True(t, manager.IsUsingFallback())
	assert.Equal(t, "fallback1", manager.GetCurrentProviderName())
}

func TestProviderManager_Process_AllFail(t *testing.T) {
	primary := &mockS2SProviderForManager{name: "primary", shouldError: true}
	fallbacks := []iface.S2SProvider{
		&mockS2SProviderForManager{name: "fallback1", shouldError: true},
	}

	manager, err := NewProviderManager(primary, fallbacks)
	require.NoError(t, err)

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3},
	}
	convCtx := &internal.ConversationContext{
		ConversationID: "test",
		SessionID:      "test",
	}

	ctx := context.Background()
	output, err := manager.Process(ctx, input, convCtx)

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "all S2S providers failed")
}
