// Package mock provides mocks for voice session internal testing.
// This file provides STT streaming mocks specifically for agent integration testing.
package mock

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// MockSTTStreaming provides a mock implementation of STT streaming for agent integration testing.
type MockSTTStreaming struct {
	errorToReturn   error
	transcripts     []string
	transcriptIndex int
	streamingDelay  time.Duration
	callCount       int
	mu              sync.RWMutex
	shouldError     bool
	simulateDelay   bool
	interimResults  bool
}

// NewMockSTTStreaming creates a new mock STT streaming provider.
func NewMockSTTStreaming() *MockSTTStreaming {
	return &MockSTTStreaming{
		transcripts:    []string{"default transcript"},
		streamingDelay: 10 * time.Millisecond,
		shouldError:    false,
		interimResults: true,
	}
}

// WithTranscripts sets the transcripts to return.
func (m *MockSTTStreaming) WithTranscripts(transcripts []string) *MockSTTStreaming {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transcripts = transcripts
	return m
}

// WithStreamingDelay sets the delay between chunks.
func (m *MockSTTStreaming) WithStreamingDelay(delay time.Duration) *MockSTTStreaming {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamingDelay = delay
	return m
}

// WithError configures the mock to return an error.
func (m *MockSTTStreaming) WithError(err error) *MockSTTStreaming {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = true
	m.errorToReturn = err
	return m
}

// Transcribe implements the STTProvider Transcribe interface.
func (m *MockSTTStreaming) Transcribe(ctx context.Context, audio []byte) (string, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	transcript := m.getNextTranscript()
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return "", errorToReturn
		}
		return "", errors.New("mock STT error")
	}

	return transcript, nil
}

// StartStreaming implements the STTProvider StartStreaming interface.
func (m *MockSTTStreaming) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	transcript := m.getNextTranscript()
	streamingDelay := m.streamingDelay
	interimResults := m.interimResults
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, errors.New("mock STT streaming error")
	}

	return NewMockStreamingSession(ctx, transcript, streamingDelay, interimResults), nil
}

// getNextTranscript returns the next transcript in the list.
func (m *MockSTTStreaming) getNextTranscript() string {
	if len(m.transcripts) == 0 {
		return "default transcript"
	}

	transcript := m.transcripts[m.transcriptIndex%len(m.transcripts)]
	m.transcriptIndex++
	return transcript
}

// GetCallCount returns the number of calls made.
func (m *MockSTTStreaming) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// MockStreamingSession provides a mock implementation of StreamingSession.
type MockStreamingSession struct {
	ctx            context.Context
	transcriptCh   chan iface.TranscriptResult
	transcript     string
	streamingDelay time.Duration
	mu             sync.RWMutex
	interimResults bool
	closed         bool
}

// NewMockStreamingSession creates a new mock streaming session.
func NewMockStreamingSession(ctx context.Context, transcript string, delay time.Duration, interimResults bool) *MockStreamingSession {
	session := &MockStreamingSession{
		ctx:            ctx,
		transcript:     transcript,
		streamingDelay: delay,
		interimResults: interimResults,
		transcriptCh:   make(chan iface.TranscriptResult, 10),
		closed:         false,
	}

	// Start streaming transcripts
	go session.streamTranscripts()

	return session
}

// streamTranscripts streams transcript results.
func (m *MockStreamingSession) streamTranscripts() {
	defer close(m.transcriptCh)

	// Send interim results if enabled
	if m.interimResults {
		words := []string{m.transcript} // Simplified: send transcript as single chunk
		for i, word := range words {
			select {
			case <-m.ctx.Done():
				return
			case m.transcriptCh <- iface.TranscriptResult{
				Text:    word,
				IsFinal: false,
			}:
			}

			if i < len(words)-1 && m.streamingDelay > 0 {
				select {
				case <-m.ctx.Done():
					return
				case <-time.After(m.streamingDelay):
				}
			}
		}
	}

	// Send final result
	select {
	case <-m.ctx.Done():
		return
	case m.transcriptCh <- iface.TranscriptResult{
		Text:    m.transcript,
		IsFinal: true,
	}:
	}
}

// SendAudio implements the StreamingSession interface.
func (m *MockStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	m.mu.RLock()
	closed := m.closed
	m.mu.RUnlock()

	if closed {
		return errors.New("streaming session closed")
	}

	// Mock: just acknowledge audio received
	return nil
}

// ReceiveTranscript implements the StreamingSession interface.
func (m *MockStreamingSession) ReceiveTranscript() <-chan iface.TranscriptResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.transcriptCh
}

// Close implements the StreamingSession interface.
func (m *MockStreamingSession) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	close(m.transcriptCh)
	return nil
}

// MockSTTProvider wraps MockSTTStreaming to implement iface.STTProvider fully.
type MockSTTProvider struct {
	*MockSTTStreaming
}

// NewMockSTTProvider creates a new mock STT provider for testing.
func NewMockSTTProvider() iface.STTProvider {
	return &MockSTTProvider{
		MockSTTStreaming: NewMockSTTStreaming(),
	}
}
