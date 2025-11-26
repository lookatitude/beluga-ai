package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// ProviderFallback manages fallback switching between providers.
type ProviderFallback struct {
	primary       any
	fallback      any
	breaker       *CircuitBreaker
	usingFallback bool
	mu            sync.RWMutex
}

// NewProviderFallback creates a new provider fallback manager.
func NewProviderFallback(primary, fallback any, breaker *CircuitBreaker) *ProviderFallback {
	return &ProviderFallback{
		primary:       primary,
		fallback:      fallback,
		breaker:       breaker,
		usingFallback: false,
	}
}

// GetSTTProvider returns the current STT provider (primary or fallback).
func (pf *ProviderFallback) GetSTTProvider() iface.STTProvider {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	if pf.usingFallback {
		if fallback, ok := pf.fallback.(iface.STTProvider); ok {
			return fallback
		}
	}

	if primary, ok := pf.primary.(iface.STTProvider); ok {
		return primary
	}

	return nil
}

// GetTTSProvider returns the current TTS provider (primary or fallback).
func (pf *ProviderFallback) GetTTSProvider() iface.TTSProvider {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	if pf.usingFallback {
		if fallback, ok := pf.fallback.(iface.TTSProvider); ok {
			return fallback
		}
	}

	if primary, ok := pf.primary.(iface.TTSProvider); ok {
		return primary
	}

	return nil
}

// SwitchToFallback switches to the fallback provider.
func (pf *ProviderFallback) SwitchToFallback() {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	pf.usingFallback = true
}

// SwitchToPrimary switches back to the primary provider.
func (pf *ProviderFallback) SwitchToPrimary() {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	pf.usingFallback = false
}

// IsUsingFallback returns whether fallback is currently active.
func (pf *ProviderFallback) IsUsingFallback() bool {
	pf.mu.RLock()
	defer pf.mu.RUnlock()
	return pf.usingFallback
}

// ExecuteWithFallback executes a function with automatic fallback on failure.
func (pf *ProviderFallback) ExecuteWithFallback(ctx context.Context, fn func(provider any) error) error {
	// Try primary first
	err := pf.breaker.Call(func() error {
		return fn(pf.primary)
	})

	if err == nil {
		// Success - switch back to primary if we were using fallback
		if pf.IsUsingFallback() {
			pf.SwitchToPrimary()
		}
		return nil
	}

	// Primary failed - try fallback if available
	if pf.fallback != nil {
		fallbackErr := fn(pf.fallback)
		if fallbackErr == nil {
			pf.SwitchToFallback()
			return nil
		}
		return fmt.Errorf("both primary and fallback providers failed: primary: %w, fallback: %w", err, fallbackErr)
	}

	return err
}
