package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// STTIntegration manages STT provider integration
type STTIntegration struct {
	provider         iface.STTProvider
	streamingSession iface.StreamingSession
	mu               sync.RWMutex
}

// NewSTTIntegration creates a new STT integration
func NewSTTIntegration(provider iface.STTProvider) *STTIntegration {
	return &STTIntegration{
		provider: provider,
	}
}

// Transcribe transcribes audio using the STT provider
func (sti *STTIntegration) Transcribe(ctx context.Context, audio []byte) (string, error) {
	sti.mu.RLock()
	provider := sti.provider
	sti.mu.RUnlock()

	if provider == nil {
		return "", fmt.Errorf("STT provider not set")
	}

	return provider.Transcribe(ctx, audio)
}

// StartStreaming starts a streaming transcription session
func (sti *STTIntegration) StartStreaming(ctx context.Context) error {
	sti.mu.Lock()
	defer sti.mu.Unlock()

	if sti.provider == nil {
		return fmt.Errorf("STT provider not set")
	}

	session, err := sti.provider.StartStreaming(ctx)
	if err != nil {
		return fmt.Errorf("failed to start streaming session: %w", err)
	}

	sti.streamingSession = session
	return nil
}

// SendAudio sends audio to the streaming session
func (sti *STTIntegration) SendAudio(ctx context.Context, audio []byte) error {
	sti.mu.RLock()
	session := sti.streamingSession
	sti.mu.RUnlock()

	if session == nil {
		return fmt.Errorf("streaming session not started")
	}

	return session.SendAudio(ctx, audio)
}

// ReceiveTranscript receives transcripts from the streaming session
func (sti *STTIntegration) ReceiveTranscript() <-chan iface.TranscriptResult {
	sti.mu.RLock()
	session := sti.streamingSession
	sti.mu.RUnlock()

	if session == nil {
		// Return closed channel
		ch := make(chan iface.TranscriptResult)
		close(ch)
		return ch
	}

	return session.ReceiveTranscript()
}

// CloseStreaming closes the streaming session
func (sti *STTIntegration) CloseStreaming() error {
	sti.mu.Lock()
	defer sti.mu.Unlock()

	if sti.streamingSession == nil {
		return nil
	}

	err := sti.streamingSession.Close()
	sti.streamingSession = nil
	return err
}
