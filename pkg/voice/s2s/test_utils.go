// Package s2s provides advanced test utilities and comprehensive mocks for testing S2S implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package s2s

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockS2SProvider provides a comprehensive mock implementation for testing.
type AdvancedMockS2SProvider struct {
	errorToReturn error
	mock.Mock
	providerName         string
	audioOutputs         []*internal.AudioOutput
	streamingSessions    []*MockStreamingSession
	callCount            int
	outputIndex          int
	processingDelay      time.Duration
	mu                   sync.RWMutex
	shouldError          bool
	simulateNetworkDelay bool
}

// NewAdvancedMockS2SProvider creates a new advanced mock with configurable behavior.
func NewAdvancedMockS2SProvider(providerName string, opts ...MockOption) *AdvancedMockS2SProvider {
	defaultOutput := &internal.AudioOutput{
		Data: []byte{1, 2, 3, 4, 5},
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
		Provider:  providerName,
		Latency:   100 * time.Millisecond,
	}

	m := &AdvancedMockS2SProvider{
		providerName:      providerName,
		audioOutputs:      []*internal.AudioOutput{defaultOutput},
		processingDelay:   10 * time.Millisecond,
		streamingSessions: make([]*MockStreamingSession, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockS2SProvider.
type MockOption func(*AdvancedMockS2SProvider)

// WithProviderName sets the provider name.
func WithProviderName(name string) MockOption {
	return func(m *AdvancedMockS2SProvider) {
		m.providerName = name
	}
}

// WithAudioOutputs sets the audio outputs to return.
func WithAudioOutputs(outputs ...*internal.AudioOutput) MockOption {
	return func(m *AdvancedMockS2SProvider) {
		m.audioOutputs = outputs
	}
}

// WithError configures the mock to return an error.
func WithError(err error) MockOption {
	return func(m *AdvancedMockS2SProvider) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithMockDelay sets the processing delay.
func WithMockDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockS2SProvider) {
		m.processingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation.
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockS2SProvider) {
		m.simulateNetworkDelay = enabled
	}
}

// Process implements the S2SProvider interface.
func (m *AdvancedMockS2SProvider) Process(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.ExpectedCalls != nil && len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, input, convCtx, opts)
		if args.Get(0) != nil {
			if output, ok := args.Get(0).(*internal.AudioOutput); ok {
				return output, args.Error(1)
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
		case <-time.After(m.processingDelay):
		}
	}

	// Default behavior
	return m.getNextOutput(), nil
}

// Name implements the S2SProvider interface.
func (m *AdvancedMockS2SProvider) Name() string {
	return m.providerName
}

// StartStreaming implements the StreamingS2SProvider interface.
func (m *AdvancedMockS2SProvider) StartStreaming(ctx context.Context, convCtx *internal.ConversationContext, opts ...internal.STSOption) (iface.StreamingSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, errors.New("mock streaming error")
	}

	session := NewMockStreamingSession(ctx, m.processingDelay, m.audioOutputs)
	m.streamingSessions = append(m.streamingSessions, session)
	return session, nil
}

// getNextOutput returns the next audio output in the list.
func (m *AdvancedMockS2SProvider) getNextOutput() *internal.AudioOutput {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.audioOutputs) == 0 {
		return &internal.AudioOutput{
			Data: []byte{1, 2, 3, 4, 5},
			Format: internal.AudioFormat{
				SampleRate: 24000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "PCM",
			},
			Timestamp: time.Now(),
			Provider:  m.providerName,
			Latency:   100 * time.Millisecond,
		}
	}

	output := m.audioOutputs[m.outputIndex%len(m.audioOutputs)]
	m.outputIndex++
	// Create a copy to avoid race conditions
	return &internal.AudioOutput{
		Data:                 append([]byte(nil), output.Data...),
		Format:               output.Format,
		Timestamp:            time.Now(),
		Provider:             output.Provider,
		VoiceCharacteristics: output.VoiceCharacteristics,
		Latency:              output.Latency,
	}
}

// GetCallCount returns the number of times Process has been called.
func (m *AdvancedMockS2SProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// MockStreamingSession provides a mock implementation of StreamingSession.
type MockStreamingSession struct {
	ctx          context.Context
	audioCh      chan iface.AudioOutputChunk
	audioOutputs []*internal.AudioOutput
	index        int
	delay        time.Duration
	mu           sync.RWMutex
	closed       bool
}

// NewMockStreamingSession creates a new mock streaming session.
func NewMockStreamingSession(ctx context.Context, delay time.Duration, outputs []*internal.AudioOutput) *MockStreamingSession {
	session := &MockStreamingSession{
		ctx:          ctx,
		audioOutputs: outputs,
		delay:        delay,
		audioCh:      make(chan iface.AudioOutputChunk, 10),
	}

	// Start sending audio chunks
	go session.sendAudioChunks()

	return session
}

// sendAudioChunks sends audio chunks to the output channel.
func (m *MockStreamingSession) sendAudioChunks() {
	defer close(m.audioCh)

	for i, output := range m.audioOutputs {
		select {
		case <-m.ctx.Done():
			return
		case <-time.After(m.delay):
			isFinal := i == len(m.audioOutputs)-1
			chunk := iface.AudioOutputChunk{
				Audio:     output.Data,
				Timestamp: time.Now().UnixNano(),
				IsFinal:   isFinal,
			}
			select {
			case m.audioCh <- chunk:
			case <-m.ctx.Done():
				return
			}
		}
	}
}

// SendAudio implements the StreamingSession interface.
func (m *MockStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return errors.New("session closed")
	}
	return nil
}

// ReceiveAudio implements the StreamingSession interface.
func (m *MockStreamingSession) ReceiveAudio() <-chan iface.AudioOutputChunk {
	return m.audioCh
}

// Close implements the StreamingSession interface.
func (m *MockStreamingSession) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true
	return nil
}

// AssertS2SProviderInterface ensures that a type implements the S2SProvider interface.
func AssertS2SProviderInterface(t *testing.T, provider iface.S2SProvider) {
	t.Helper()
	assert.NotNil(t, provider, "S2SProvider should not be nil")

	// Test Process method
	ctx := context.Background()
	input := &internal.AudioInput{
		Data: []byte{1, 2, 3, 4, 5},
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}
	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}
	_, err := provider.Process(ctx, input, convCtx)
	// We don't care about the result, just that the method exists and can be called
	_ = err

	// Test Name method
	name := provider.Name()
	assert.NotEmpty(t, name, "Provider name should not be empty")
}
