package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// StreamingSTT manages streaming STT integration
type StreamingSTT struct {
	mu               sync.RWMutex
	sttProvider      iface.STTProvider
	streamingSession iface.StreamingSession
	active           bool
}

// NewStreamingSTT creates a new streaming STT manager
func NewStreamingSTT(provider iface.STTProvider) *StreamingSTT {
	return &StreamingSTT{
		sttProvider: provider,
		active:      false,
	}
}

// Start starts a streaming STT session
func (sstt *StreamingSTT) Start(ctx context.Context) error {
	sstt.mu.Lock()
	defer sstt.mu.Unlock()

	if sstt.active {
		return fmt.Errorf("streaming session already active")
	}

	if sstt.sttProvider == nil {
		return fmt.Errorf("STT provider not set")
	}

	session, err := sstt.sttProvider.StartStreaming(ctx)
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	sstt.streamingSession = session
	sstt.active = true
	return nil
}

// SendAudio sends audio to the streaming session
func (sstt *StreamingSTT) SendAudio(ctx context.Context, audio []byte) error {
	sstt.mu.RLock()
	session := sstt.streamingSession
	active := sstt.active
	sstt.mu.RUnlock()

	if !active || session == nil {
		return fmt.Errorf("streaming session not active")
	}

	return session.SendAudio(ctx, audio)
}

// ReceiveTranscript receives transcripts from the streaming session
func (sstt *StreamingSTT) ReceiveTranscript() <-chan iface.TranscriptResult {
	sstt.mu.RLock()
	session := sstt.streamingSession
	sstt.mu.RUnlock()

	if session == nil {
		ch := make(chan iface.TranscriptResult)
		close(ch)
		return ch
	}

	return session.ReceiveTranscript()
}

// Stop stops the streaming session
func (sstt *StreamingSTT) Stop() error {
	sstt.mu.Lock()
	defer sstt.mu.Unlock()

	if !sstt.active {
		return nil
	}

	if sstt.streamingSession != nil {
		err := sstt.streamingSession.Close()
		sstt.streamingSession = nil
		sstt.active = false
		return err
	}

	sstt.active = false
	return nil
}

// IsActive returns whether the streaming session is active
func (sstt *StreamingSTT) IsActive() bool {
	sstt.mu.RLock()
	defer sstt.mu.RUnlock()
	return sstt.active
}
