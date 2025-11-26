package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// StreamingTTS manages streaming TTS integration.
type StreamingTTS struct {
	ttsProvider   iface.TTSProvider
	currentReader io.Reader
	mu            sync.RWMutex
	active        bool
}

// NewStreamingTTS creates a new streaming TTS manager.
func NewStreamingTTS(provider iface.TTSProvider) *StreamingTTS {
	return &StreamingTTS{
		ttsProvider: provider,
		active:      false,
	}
}

// StartStream starts a streaming TTS session for the given text.
func (stts *StreamingTTS) StartStream(ctx context.Context, text string) (io.Reader, error) {
	stts.mu.Lock()
	defer stts.mu.Unlock()

	if stts.ttsProvider == nil {
		return nil, errors.New("TTS provider not set")
	}

	reader, err := stts.ttsProvider.StreamGenerate(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to start streaming: %w", err)
	}

	stts.currentReader = reader
	stts.active = true
	return reader, nil
}

// Stop stops the streaming TTS session.
func (stts *StreamingTTS) Stop() {
	stts.mu.Lock()
	defer stts.mu.Unlock()
	stts.active = false
	stts.currentReader = nil
}

// IsActive returns whether the streaming session is active.
func (stts *StreamingTTS) IsActive() bool {
	stts.mu.RLock()
	defer stts.mu.RUnlock()
	return stts.active
}

// GetCurrentReader returns the current streaming reader.
func (stts *StreamingTTS) GetCurrentReader() io.Reader {
	stts.mu.RLock()
	defer stts.mu.RUnlock()
	return stts.currentReader
}
