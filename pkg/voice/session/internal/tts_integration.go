package internal

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// TTSIntegration manages TTS provider integration.
type TTSIntegration struct {
	provider iface.TTSProvider
	mu       sync.RWMutex
}

// NewTTSIntegration creates a new TTS integration.
func NewTTSIntegration(provider iface.TTSProvider) *TTSIntegration {
	return &TTSIntegration{
		provider: provider,
	}
}

// GenerateSpeech generates speech from text using the TTS provider.
func (tti *TTSIntegration) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	tti.mu.RLock()
	provider := tti.provider
	tti.mu.RUnlock()

	if provider == nil {
		return nil, errors.New("TTS provider not set")
	}

	return provider.GenerateSpeech(ctx, text)
}

// StreamGenerate starts streaming speech generation.
func (tti *TTSIntegration) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	tti.mu.RLock()
	provider := tti.provider
	tti.mu.RUnlock()

	if provider == nil {
		return nil, errors.New("TTS provider not set")
	}

	return provider.StreamGenerate(ctx, text)
}
