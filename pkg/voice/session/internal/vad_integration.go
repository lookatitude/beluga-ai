package internal

import (
	"context"
	"errors"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// VADIntegration manages VAD provider integration.
type VADIntegration struct {
	provider iface.VADProvider
	mu       sync.RWMutex
}

// NewVADIntegration creates a new VAD integration.
func NewVADIntegration(provider iface.VADProvider) *VADIntegration {
	return &VADIntegration{
		provider: provider,
	}
}

// DetectSpeech detects speech in audio using the VAD provider.
func (vi *VADIntegration) DetectSpeech(ctx context.Context, audio []byte) (bool, error) {
	vi.mu.RLock()
	provider := vi.provider
	vi.mu.RUnlock()

	if provider == nil {
		return false, errors.New("VAD provider not set")
	}

	return provider.Process(ctx, audio)
}

// DetectSpeechStream detects speech in a stream of audio.
func (vi *VADIntegration) DetectSpeechStream(ctx context.Context, audioCh <-chan []byte) (<-chan iface.VADResult, error) {
	vi.mu.RLock()
	provider := vi.provider
	vi.mu.RUnlock()

	if provider == nil {
		return nil, errors.New("VAD provider not set")
	}

	return provider.ProcessStream(ctx, audioCh)
}
