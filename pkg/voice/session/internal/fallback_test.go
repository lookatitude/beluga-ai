package internal

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSTTProvider struct {
	shouldError bool
}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	if m.shouldError {
		return "", errors.New("STT error")
	}
	return "transcript", nil
}

func (m *mockSTTProvider) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
	return nil, errors.New("not implemented")
}

type mockTTSProvider struct {
	shouldError bool
}

func (m *mockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	if m.shouldError {
		return nil, errors.New("TTS error")
	}
	return []byte{1, 2, 3}, nil
}

func (m *mockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return nil, errors.New("not implemented")
}

func TestNewProviderFallback(t *testing.T) {
	primary := &mockSTTProvider{}
	fallback := &mockSTTProvider{}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)
	assert.NotNil(t, pf)
	assert.False(t, pf.IsUsingFallback())
}

func TestProviderFallback_GetSTTProvider(t *testing.T) {
	primary := &mockSTTProvider{}
	fallback := &mockSTTProvider{}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)

	// Should return primary initially
	provider := pf.GetSTTProvider()
	assert.Equal(t, primary, provider)

	// Switch to fallback
	pf.SwitchToFallback()
	provider = pf.GetSTTProvider()
	assert.Equal(t, fallback, provider)
}

func TestProviderFallback_GetTTSProvider(t *testing.T) {
	primary := &mockTTSProvider{}
	fallback := &mockTTSProvider{}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)

	// Should return primary initially
	provider := pf.GetTTSProvider()
	assert.Equal(t, primary, provider)

	// Switch to fallback
	pf.SwitchToFallback()
	provider = pf.GetTTSProvider()
	assert.Equal(t, fallback, provider)
}

func TestProviderFallback_SwitchToFallback(t *testing.T) {
	primary := &mockSTTProvider{}
	fallback := &mockSTTProvider{}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)
	assert.False(t, pf.IsUsingFallback())

	pf.SwitchToFallback()
	assert.True(t, pf.IsUsingFallback())
}

func TestProviderFallback_SwitchToPrimary(t *testing.T) {
	primary := &mockSTTProvider{}
	fallback := &mockSTTProvider{}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)
	pf.SwitchToFallback()
	assert.True(t, pf.IsUsingFallback())

	pf.SwitchToPrimary()
	assert.False(t, pf.IsUsingFallback())
}

func TestProviderFallback_IsUsingFallback(t *testing.T) {
	primary := &mockSTTProvider{}
	fallback := &mockSTTProvider{}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)
	assert.False(t, pf.IsUsingFallback())

	pf.SwitchToFallback()
	assert.True(t, pf.IsUsingFallback())
}

func TestProviderFallback_ExecuteWithFallback_Success(t *testing.T) {
	primary := &mockSTTProvider{shouldError: false}
	ctx := context.Background()
	fallback := &mockSTTProvider{}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)

	err := pf.ExecuteWithFallback(ctx, func(provider any) error {
		stt := provider.(*mockSTTProvider)
		_, transcribeErr := stt.Transcribe(ctx, []byte{1, 2, 3})
		return transcribeErr
	})

	require.NoError(t, err)
	assert.False(t, pf.IsUsingFallback())
}

func TestProviderFallback_ExecuteWithFallback_PrimaryFails(t *testing.T) {
	primary := &mockSTTProvider{shouldError: true}
	ctx := context.Background()
	fallback := &mockSTTProvider{shouldError: false}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)

	err := pf.ExecuteWithFallback(ctx, func(provider any) error {
		stt := provider.(*mockSTTProvider)
		_, transcribeErr := stt.Transcribe(ctx, []byte{1, 2, 3})
		return transcribeErr
	})

	require.NoError(t, err)
	assert.True(t, pf.IsUsingFallback())
}

func TestProviderFallback_ExecuteWithFallback_BothFail(t *testing.T) {
	primary := &mockSTTProvider{shouldError: true}
	ctx := context.Background()
	fallback := &mockSTTProvider{shouldError: true}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)

	err := pf.ExecuteWithFallback(ctx, func(provider any) error {
		stt := provider.(*mockSTTProvider)
		_, transcribeErr := stt.Transcribe(ctx, []byte{1, 2, 3})
		return transcribeErr
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "both primary and fallback providers failed")
}

func TestProviderFallback_ExecuteWithFallback_NoFallback(t *testing.T) {
	primary := &mockSTTProvider{shouldError: true}
	ctx := context.Background()
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, nil, breaker)

	err := pf.ExecuteWithFallback(ctx, func(provider any) error {
		stt := provider.(*mockSTTProvider)
		_, transcribeErr := stt.Transcribe(ctx, []byte{1, 2, 3})
		return transcribeErr
	})

	require.Error(t, err)
}

func TestProviderFallback_ExecuteWithFallback_SwitchBackToPrimary(t *testing.T) {
	primary := &mockSTTProvider{shouldError: false}
	ctx := context.Background()
	fallback := &mockSTTProvider{}
	breaker := NewCircuitBreaker(5, 10*1000, 5*time.Second)

	pf := NewProviderFallback(primary, fallback, breaker)
	pf.SwitchToFallback()
	assert.True(t, pf.IsUsingFallback())

	err := pf.ExecuteWithFallback(ctx, func(provider any) error {
		stt := provider.(*mockSTTProvider)
		_, transcribeErr := stt.Transcribe(ctx, []byte{1, 2, 3})
		return transcribeErr
	})

	require.NoError(t, err)
	// Should switch back to primary after success
	assert.False(t, pf.IsUsingFallback())
}
