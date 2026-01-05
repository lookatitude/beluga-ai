package s2s

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockS2SProvider struct {
	name        string
	shouldError bool
}

func (m *mockS2SProvider) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
	if m.shouldError {
		return nil, errors.New("provider error")
	}
	return &internal.AudioOutput{
		Data:     []byte{1, 2, 3},
		Provider: m.name,
	}, nil
}

func (m *mockS2SProvider) Name() string {
	return m.name
}

func TestNewProviderFallback(t *testing.T) {
	primary := &mockS2SProvider{name: "primary"}
	fallbacks := []iface.S2SProvider{
		&mockS2SProvider{name: "fallback1"},
		&mockS2SProvider{name: "fallback2"},
	}
	breaker := NewCircuitBreaker(5, 100, 5*time.Second)

	pf := NewProviderFallback(primary, fallbacks, breaker)
	assert.NotNil(t, pf)
	assert.False(t, pf.IsUsingFallback())
	assert.Equal(t, "primary", pf.GetCurrentProviderName())
}

func TestProviderFallback_GetProvider(t *testing.T) {
	primary := &mockS2SProvider{name: "primary"}
	fallbacks := []iface.S2SProvider{
		&mockS2SProvider{name: "fallback1"},
	}
	breaker := NewCircuitBreaker(5, 100, 5*time.Second)

	pf := NewProviderFallback(primary, fallbacks, breaker)

	// Should return primary initially
	provider := pf.GetProvider()
	assert.Equal(t, primary, provider)

	// Switch to fallback
	pf.SwitchToFallback()
	provider = pf.GetProvider()
	assert.Equal(t, fallbacks[0], provider)
}

func TestProviderFallback_SwitchToFallback(t *testing.T) {
	primary := &mockS2SProvider{name: "primary"}
	fallbacks := []iface.S2SProvider{
		&mockS2SProvider{name: "fallback1"},
		&mockS2SProvider{name: "fallback2"},
	}
	breaker := NewCircuitBreaker(5, 100, 5*time.Second)

	pf := NewProviderFallback(primary, fallbacks, breaker)

	// Switch to first fallback
	switched := pf.SwitchToFallback()
	assert.True(t, switched)
	assert.True(t, pf.IsUsingFallback())
	assert.Equal(t, "fallback1", pf.GetCurrentProviderName())

	// Switch to second fallback
	switched = pf.SwitchToFallback()
	assert.True(t, switched)
	assert.Equal(t, "fallback2", pf.GetCurrentProviderName())

	// Try to switch again (no more fallbacks)
	switched = pf.SwitchToFallback()
	assert.False(t, switched)
}

func TestProviderFallback_SwitchToPrimary(t *testing.T) {
	primary := &mockS2SProvider{name: "primary"}
	fallbacks := []iface.S2SProvider{
		&mockS2SProvider{name: "fallback1"},
	}
	breaker := NewCircuitBreaker(5, 100, 5*time.Second)

	pf := NewProviderFallback(primary, fallbacks, breaker)

	// Switch to fallback
	pf.SwitchToFallback()
	assert.True(t, pf.IsUsingFallback())

	// Switch back to primary
	pf.SwitchToPrimary()
	assert.False(t, pf.IsUsingFallback())
	assert.Equal(t, "primary", pf.GetCurrentProviderName())
}

func TestProviderFallback_ProcessWithFallback_Success(t *testing.T) {
	primary := &mockS2SProvider{name: "primary", shouldError: false}
	fallbacks := []iface.S2SProvider{
		&mockS2SProvider{name: "fallback1"},
	}
	breaker := NewCircuitBreaker(5, 100, 5*time.Second)

	pf := NewProviderFallback(primary, fallbacks, breaker)

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3},
	}
	convCtx := &internal.ConversationContext{
		ConversationID: "test",
		SessionID:      "test",
	}

	ctx := context.Background()
	output, err := pf.ProcessWithFallback(ctx, input, convCtx)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.False(t, pf.IsUsingFallback())
}

func TestProviderFallback_ProcessWithFallback_PrimaryFails(t *testing.T) {
	primary := &mockS2SProvider{name: "primary", shouldError: true}
	fallbacks := []iface.S2SProvider{
		&mockS2SProvider{name: "fallback1", shouldError: false},
	}
	breaker := NewCircuitBreaker(5, 100, 5*time.Second)

	pf := NewProviderFallback(primary, fallbacks, breaker)

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3},
	}
	convCtx := &internal.ConversationContext{
		ConversationID: "test",
		SessionID:      "test",
	}

	ctx := context.Background()
	output, err := pf.ProcessWithFallback(ctx, input, convCtx)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.True(t, pf.IsUsingFallback())
	assert.Equal(t, "fallback1", output.Provider)
}

func TestProviderFallback_ProcessWithFallback_AllFail(t *testing.T) {
	primary := &mockS2SProvider{name: "primary", shouldError: true}
	fallbacks := []iface.S2SProvider{
		&mockS2SProvider{name: "fallback1", shouldError: true},
		&mockS2SProvider{name: "fallback2", shouldError: true},
	}
	breaker := NewCircuitBreaker(5, 100, 5*time.Second)

	pf := NewProviderFallback(primary, fallbacks, breaker)

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3},
	}
	convCtx := &internal.ConversationContext{
		ConversationID: "test",
		SessionID:      "test",
	}

	ctx := context.Background()
	output, err := pf.ProcessWithFallback(ctx, input, convCtx)

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "all S2S providers failed")
}

func TestCircuitBreaker_Call(t *testing.T) {
	breaker := NewCircuitBreaker(3, 100, 1*time.Second)

	// First few calls should succeed
	for i := 0; i < 2; i++ {
		err := breaker.Call(func() error {
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, CircuitBreakerStateClosed, breaker.GetState())
	}

	// Fail enough times to open circuit
	for i := 0; i < 3; i++ {
		err := breaker.Call(func() error {
			return errors.New("test error")
		})
		require.Error(t, err)
	}

	// Circuit should be open
	assert.Equal(t, CircuitBreakerStateOpen, breaker.GetState())

	// Call should fail immediately
	err := breaker.Call(func() error {
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	breaker := NewCircuitBreaker(2, 100, 100*time.Millisecond)

	// Open circuit
	for i := 0; i < 2; i++ {
		_ = breaker.Call(func() error {
			return errors.New("test error")
		})
	}
	assert.Equal(t, CircuitBreakerStateOpen, breaker.GetState())

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Should be in half-open state and allow one call
	err := breaker.Call(func() error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, CircuitBreakerStateClosed, breaker.GetState())
}
