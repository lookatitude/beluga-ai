package internal

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

// S2SIntegration manages S2S provider integration.
// Note: This integration works around Go's restriction on importing internal packages
// by using reflection-like approaches or by accepting that we need to create
// compatible types. For now, we provide a simplified interface that works with
// the S2S provider interface.
type S2SIntegration struct {
	provider         iface.S2SProvider
	fallback         *s2s.ProviderFallback // Optional fallback support
	streamingSession iface.StreamingSession
	mu               sync.RWMutex
}

// NewS2SIntegration creates a new S2S integration.
func NewS2SIntegration(provider iface.S2SProvider) *S2SIntegration {
	return &S2SIntegration{
		provider: provider,
	}
}

// NewS2SIntegrationWithFallback creates a new S2S integration with fallback support.
func NewS2SIntegrationWithFallback(primary iface.S2SProvider, fallbacks []iface.S2SProvider) *S2SIntegration {
	breaker := s2s.NewCircuitBreaker(5, 100, 5*time.Second)
	fallback := s2s.NewProviderFallback(primary, fallbacks, breaker)
	return &S2SIntegration{
		provider: primary,
		fallback: fallback,
	}
}

// ProcessAudioWithSessionID processes audio using the S2S provider, creating conversation context from session ID.
// If fallback is configured, it will automatically fallback to other providers on failure.
func (s2si *S2SIntegration) ProcessAudioWithSessionID(ctx context.Context, audio []byte, sessionID string) ([]byte, error) {
	s2si.mu.RLock()
	fallback := s2si.fallback
	s2si.mu.RUnlock()

	// Create audio input and conversation context using helper functions
	input := s2s.NewAudioInput(audio, sessionID)
	convCtx := s2s.NewConversationContext(sessionID)

	// Use fallback if available, otherwise use direct provider
	if fallback != nil {
		output, err := fallback.ProcessWithFallback(ctx, input, convCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to process audio with S2S providers (including fallback): %w", err)
		}
		return s2s.ExtractAudioData(output), nil
	}

	// Direct provider (no fallback)
	s2si.mu.RLock()
	provider := s2si.provider
	s2si.mu.RUnlock()

	if provider == nil {
		return nil, errors.New("S2S provider not set")
	}

	// Process audio through S2S provider
	output, err := provider.Process(ctx, input, convCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to process audio with S2S provider: %w", err)
	}

	// Extract audio data from output
	return s2s.ExtractAudioData(output), nil
}

// StartStreaming starts a streaming S2S session.
func (s2si *S2SIntegration) StartStreaming(ctx context.Context, sessionID string) error {
	s2si.mu.Lock()
	defer s2si.mu.Unlock()

	if s2si.provider == nil {
		return errors.New("S2S provider not set")
	}

	streamingProvider, ok := s2si.provider.(iface.StreamingS2SProvider)
	if !ok {
		return errors.New("S2S provider does not support streaming")
	}

	// Create conversation context using helper function
	convCtx := s2s.NewConversationContext(sessionID)

	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	if err != nil {
		return fmt.Errorf("failed to start streaming session: %w", err)
	}

	s2si.streamingSession = session
	return nil
}

// SendAudio sends audio to the streaming session.
func (s2si *S2SIntegration) SendAudio(ctx context.Context, audio []byte) error {
	s2si.mu.RLock()
	session := s2si.streamingSession
	s2si.mu.RUnlock()

	if session == nil {
		return errors.New("streaming session not started")
	}

	return session.SendAudio(ctx, audio)
}

// ReceiveAudio receives audio output from the streaming session.
func (s2si *S2SIntegration) ReceiveAudio() <-chan iface.AudioOutputChunk {
	s2si.mu.RLock()
	session := s2si.streamingSession
	s2si.mu.RUnlock()

	if session == nil {
		// Return closed channel
		ch := make(chan iface.AudioOutputChunk)
		close(ch)
		return ch
	}

	return session.ReceiveAudio()
}

// CloseStreaming closes the streaming session.
func (s2si *S2SIntegration) CloseStreaming() error {
	s2si.mu.Lock()
	defer s2si.mu.Unlock()

	if s2si.streamingSession == nil {
		return nil
	}

	err := s2si.streamingSession.Close()
	s2si.streamingSession = nil
	return err
}
