// Package tts provides advanced test utilities and comprehensive mocks for testing TTS implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package tts

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockTTSProvider provides a comprehensive mock implementation for testing.
type AdvancedMockTTSProvider struct {
	errorToReturn error
	mock.Mock
	providerName         string
	audioResponses       [][]byte
	callCount            int
	audioIndex           int
	streamingDelay       time.Duration
	mu                   sync.RWMutex
	shouldError          bool
	simulateNetworkDelay bool
}

// NewAdvancedMockTTSProvider creates a new advanced mock with configurable behavior.
func NewAdvancedMockTTSProvider(providerName string, opts ...MockOption) *AdvancedMockTTSProvider {
	m := &AdvancedMockTTSProvider{
		providerName:   providerName,
		audioResponses: [][]byte{[]byte("default audio response")},
		streamingDelay: 10 * time.Millisecond,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockTTSProvider.
type MockOption func(*AdvancedMockTTSProvider)

// WithProviderName sets the provider name.
func WithProviderName(name string) MockOption {
	return func(m *AdvancedMockTTSProvider) {
		m.providerName = name
	}
}

// WithAudioResponses sets the audio responses to return.
func WithAudioResponses(audioResponses ...[]byte) MockOption {
	return func(m *AdvancedMockTTSProvider) {
		m.audioResponses = audioResponses
	}
}

// WithError configures the mock to return an error.
func WithError(err error) MockOption {
	return func(m *AdvancedMockTTSProvider) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithStreamingDelay sets the delay between streaming chunks.
func WithStreamingDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockTTSProvider) {
		m.streamingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation.
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockTTSProvider) {
		m.simulateNetworkDelay = enabled
	}
}

// GenerateSpeech implements the TTSProvider interface.
func (m *AdvancedMockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.ExpectedCalls != nil && len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, text)
		if args.Get(0) != nil {
			if audio, ok := args.Get(0).([]byte); ok {
				return audio, args.Error(1)
			}
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock error")
	}

	// Simulate network delay if enabled
	if m.simulateNetworkDelay {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Default behavior
	return m.getNextAudio(), nil
}

// StreamGenerate implements the TTSProvider interface.
func (m *AdvancedMockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock streaming error")
	}

	// Create a reader that streams audio chunks
	audio := m.getNextAudio()
	return bytes.NewReader(audio), nil
}

// getNextAudio returns the next audio response in the list.
func (m *AdvancedMockTTSProvider) getNextAudio() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.audioResponses) == 0 {
		return []byte("default audio response")
	}

	audio := m.audioResponses[m.audioIndex%len(m.audioResponses)]
	m.audioIndex++
	return audio
}

// GetCallCount returns the number of times GenerateSpeech has been called.
func (m *AdvancedMockTTSProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// AssertTTSProviderInterface ensures that a type implements the TTSProvider interface.
func AssertTTSProviderInterface(t *testing.T, provider iface.TTSProvider) {
	t.Helper()
	assert.NotNil(t, provider, "TTSProvider should not be nil")

	// Test GenerateSpeech method
	ctx := context.Background()
	text := "Test text"
	_, err := provider.GenerateSpeech(ctx, text)
	// We don't care about the result, just that the method exists and can be called
	_ = err

	// Test StreamGenerate method
	reader, err := provider.StreamGenerate(ctx, text)
	if err == nil && reader != nil {
		_ = reader
	}
}
