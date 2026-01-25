package s2s

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// ProviderManager manages multiple S2S providers with fallback support.
type ProviderManager struct {
	primary   iface.S2SProvider
	fallback  *ProviderFallback
	fallbacks []iface.S2SProvider
	mu        sync.RWMutex
}

// NewProviderManager creates a new provider manager with primary and fallback providers.
func NewProviderManager(primary iface.S2SProvider, fallbacks []iface.S2SProvider) (*ProviderManager, error) {
	if primary == nil {
		return nil, NewS2SError("NewProviderManager", ErrCodeInvalidConfig,
			errors.New("primary provider cannot be nil"))
	}

	breaker := NewCircuitBreaker(5, 100, 5*time.Second)
	fallback := NewProviderFallback(primary, fallbacks, breaker)

	return &ProviderManager{
		primary:   primary,
		fallbacks: fallbacks,
		fallback:  fallback,
	}, nil
}

// GetPrimaryProvider returns the primary provider.
func (pm *ProviderManager) GetPrimaryProvider() iface.S2SProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.primary
}

// GetFallbackProviders returns the fallback providers.
func (pm *ProviderManager) GetFallbackProviders() []iface.S2SProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.fallbacks
}

// GetCurrentProvider returns the currently active provider (primary or fallback).
func (pm *ProviderManager) GetCurrentProvider() iface.S2SProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.fallback.GetProvider()
}

// IsUsingFallback returns whether a fallback provider is currently active.
func (pm *ProviderManager) IsUsingFallback() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.fallback.IsUsingFallback()
}

// GetCurrentProviderName returns the name of the currently active provider.
func (pm *ProviderManager) GetCurrentProviderName() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.fallback.GetCurrentProviderName()
}

// Process processes audio using the provider manager with automatic fallback.
func (pm *ProviderManager) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
	pm.mu.RLock()
	fallback := pm.fallback
	pm.mu.RUnlock()

	return fallback.ProcessWithFallback(ctx, input, convCtx, opts...)
}

// StartStreaming starts a streaming session using the current provider.
func (pm *ProviderManager) StartStreaming(ctx context.Context, convCtx *internal.ConversationContext, opts ...internal.STSOption) (iface.StreamingSession, error) {
	provider := pm.GetCurrentProvider()
	if provider == nil {
		return nil, NewS2SError("StartStreaming", ErrCodeInvalidConfig,
			errors.New("no provider available"))
	}

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	if !ok {
		return nil, NewS2SError("StartStreaming", ErrCodeUnsupportedProvider,
			fmt.Errorf("provider '%s' does not support streaming", provider.Name()))
	}

	return streamingProvider.StartStreaming(ctx, convCtx, opts...)
}
