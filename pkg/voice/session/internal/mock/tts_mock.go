// Package mock provides mocks for voice session internal testing.
// This file provides TTS streaming mocks specifically for agent integration testing.
package mock

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// MockTTSStreaming provides a mock implementation of TTS streaming for agent integration testing.
type MockTTSStreaming struct {
	errorToReturn  error
	audioResponses [][]byte
	audioIndex     int
	streamingDelay time.Duration
	callCount      int
	mu             sync.RWMutex
	shouldError    bool
	simulateDelay  bool
}

// NewMockTTSStreaming creates a new mock TTS streaming provider.
func NewMockTTSStreaming() *MockTTSStreaming {
	return &MockTTSStreaming{
		audioResponses: [][]byte{[]byte("default audio response")},
		streamingDelay: 10 * time.Millisecond,
		shouldError:    false,
	}
}

// WithAudioResponses sets the audio responses to return.
func (m *MockTTSStreaming) WithAudioResponses(responses [][]byte) *MockTTSStreaming {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audioResponses = responses
	return m
}

// WithStreamingDelay sets the delay between chunks.
func (m *MockTTSStreaming) WithStreamingDelay(delay time.Duration) *MockTTSStreaming {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamingDelay = delay
	return m
}

// WithError configures the mock to return an error.
func (m *MockTTSStreaming) WithError(err error) *MockTTSStreaming {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = true
	m.errorToReturn = err
	return m
}

// StreamGenerate implements the TTSProvider StreamGenerate interface for testing.
func (m *MockTTSStreaming) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	audio := m.getNextAudio()
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, errors.New("mock TTS error")
	}

	// Return audio as a reader
	return bytes.NewReader(audio), nil
}

// GenerateSpeech implements the TTSProvider GenerateSpeech interface for testing.
func (m *MockTTSStreaming) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	audio := m.getNextAudio()
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, errors.New("mock TTS error")
	}

	return audio, nil
}

// getNextAudio returns the next audio response in the list.
func (m *MockTTSStreaming) getNextAudio() []byte {
	if len(m.audioResponses) == 0 {
		return []byte("default audio response")
	}

	audio := m.audioResponses[m.audioIndex%len(m.audioResponses)]
	m.audioIndex++
	return audio
}

// GetCallCount returns the number of calls made.
func (m *MockTTSStreaming) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// MockTTSProvider wraps MockTTSStreaming to implement iface.TTSProvider fully.
type MockTTSProvider struct {
	*MockTTSStreaming
}

// NewMockTTSProvider creates a new mock TTS provider for testing.
func NewMockTTSProvider() iface.TTSProvider {
	return &MockTTSProvider{
		MockTTSStreaming: NewMockTTSStreaming(),
	}
}
