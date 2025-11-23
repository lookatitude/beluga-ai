// Package stt provides advanced test utilities and comprehensive mocks for testing STT implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package stt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockSTTProvider provides a comprehensive mock implementation for testing
type AdvancedMockSTTProvider struct {
	mock.Mock

	// Configuration
	providerName string
	callCount    int
	mu           sync.RWMutex

	// Configurable behavior
	shouldError          bool
	errorToReturn        error
	transcriptions       []string
	transcriptionIndex   int
	streamingDelay       time.Duration
	simulateNetworkDelay bool

	// Streaming support
	streamingSessions []*MockStreamingSession
}

// NewAdvancedMockSTTProvider creates a new advanced mock with configurable behavior
func NewAdvancedMockSTTProvider(providerName string, opts ...MockOption) *AdvancedMockSTTProvider {
	m := &AdvancedMockSTTProvider{
		providerName:      providerName,
		transcriptions:    []string{"Default transcription"},
		streamingDelay:    10 * time.Millisecond,
		streamingSessions: make([]*MockStreamingSession, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockSTTProvider
type MockOption func(*AdvancedMockSTTProvider)

// WithProviderName sets the provider name
func WithProviderName(name string) MockOption {
	return func(m *AdvancedMockSTTProvider) {
		m.providerName = name
	}
}

// WithTranscriptions sets the transcriptions to return
func WithTranscriptions(transcriptions ...string) MockOption {
	return func(m *AdvancedMockSTTProvider) {
		m.transcriptions = transcriptions
	}
}

// WithError configures the mock to return an error
func WithError(err error) MockOption {
	return func(m *AdvancedMockSTTProvider) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithStreamingDelay sets the delay between streaming chunks
func WithStreamingDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockSTTProvider) {
		m.streamingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockSTTProvider) {
		m.simulateNetworkDelay = enabled
	}
}

// Transcribe implements the STTProvider interface
func (m *AdvancedMockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, audio)
		if args.Get(0) != nil {
			if text, ok := args.Get(0).(string); ok {
				return text, args.Error(1)
			}
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return "", m.errorToReturn
		}
		return "", fmt.Errorf("mock error")
	}

	// Simulate network delay if enabled
	if m.simulateNetworkDelay {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Default behavior
	return m.getNextTranscription(), nil
}

// StartStreaming implements the STTProvider interface
func (m *AdvancedMockSTTProvider) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, fmt.Errorf("mock streaming error")
	}

	session := NewMockStreamingSession(ctx, m.streamingDelay, m.transcriptions)
	m.streamingSessions = append(m.streamingSessions, session)
	return session, nil
}

// getNextTranscription returns the next transcription in the list
func (m *AdvancedMockSTTProvider) getNextTranscription() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.transcriptions) == 0 {
		return "Default transcription"
	}

	text := m.transcriptions[m.transcriptionIndex%len(m.transcriptions)]
	m.transcriptionIndex++
	return text
}

// GetCallCount returns the number of times Transcribe has been called
func (m *AdvancedMockSTTProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// MockStreamingSession provides a mock implementation of StreamingSession
type MockStreamingSession struct {
	ctx            context.Context
	transcriptions []string
	index          int
	delay          time.Duration
	resultCh       chan iface.TranscriptResult
	closed         bool
	mu             sync.RWMutex
}

// NewMockStreamingSession creates a new mock streaming session
func NewMockStreamingSession(ctx context.Context, delay time.Duration, transcriptions []string) *MockStreamingSession {
	session := &MockStreamingSession{
		ctx:            ctx,
		transcriptions: transcriptions,
		delay:          delay,
		resultCh:       make(chan iface.TranscriptResult, 10),
	}

	// Start sending transcriptions
	go session.sendTranscriptions()

	return session
}

// sendTranscriptions sends transcriptions to the result channel
func (m *MockStreamingSession) sendTranscriptions() {
	defer close(m.resultCh)

	for i, text := range m.transcriptions {
		select {
		case <-m.ctx.Done():
			return
		case <-time.After(m.delay):
			isFinal := i == len(m.transcriptions)-1
			result := iface.TranscriptResult{
				Text:       text,
				IsFinal:    isFinal,
				Language:   "en",
				Confidence: 0.95,
			}
			select {
			case m.resultCh <- result:
			case <-m.ctx.Done():
				return
			}
		}
	}
}

// SendAudio implements the StreamingSession interface
func (m *MockStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return errors.New("session closed")
	}
	return nil
}

// ReceiveTranscript implements the StreamingSession interface
func (m *MockStreamingSession) ReceiveTranscript() <-chan iface.TranscriptResult {
	return m.resultCh
}

// Close implements the StreamingSession interface
func (m *MockStreamingSession) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true
	return nil
}

// AssertSTTProviderInterface ensures that a type implements the STTProvider interface
func AssertSTTProviderInterface(t *testing.T, provider iface.STTProvider) {
	assert.NotNil(t, provider, "STTProvider should not be nil")

	// Test Transcribe method
	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	_, err := provider.Transcribe(ctx, audio)
	// We don't care about the result, just that the method exists and can be called
	_ = err

	// Test StartStreaming method
	session, err := provider.StartStreaming(ctx)
	if err == nil && session != nil {
		_ = session.Close()
	}
}
