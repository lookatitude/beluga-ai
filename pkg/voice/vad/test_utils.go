// Package vad provides advanced test utilities and comprehensive mocks for testing VAD implementations.
package vad

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockVADProvider provides a comprehensive mock implementation for testing.
type AdvancedMockVADProvider struct {
	errorToReturn error
	mock.Mock
	providerName         string
	speechResults        []bool
	callCount            int
	resultIndex          int
	processingDelay      time.Duration
	mu                   sync.RWMutex
	shouldError          bool
	simulateNetworkDelay bool
}

// NewAdvancedMockVADProvider creates a new advanced mock with configurable behavior.
func NewAdvancedMockVADProvider(providerName string, opts ...MockOption) *AdvancedMockVADProvider {
	m := &AdvancedMockVADProvider{
		providerName:    providerName,
		speechResults:   []bool{true},
		processingDelay: 10 * time.Millisecond,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockVADProvider.
type MockOption func(*AdvancedMockVADProvider)

// WithProviderName sets the provider name.
func WithProviderName(name string) MockOption {
	return func(m *AdvancedMockVADProvider) {
		m.providerName = name
	}
}

// WithSpeechResults sets the speech detection results to return.
func WithSpeechResults(results ...bool) MockOption {
	return func(m *AdvancedMockVADProvider) {
		m.speechResults = results
	}
}

// WithError configures the mock to return an error.
func WithError(err error) MockOption {
	return func(m *AdvancedMockVADProvider) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithProcessingDelay sets the delay for processing.
func WithProcessingDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockVADProvider) {
		m.processingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation.
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockVADProvider) {
		m.simulateNetworkDelay = enabled
	}
}

// Process implements the VADProvider interface.
func (m *AdvancedMockVADProvider) Process(ctx context.Context, audio []byte) (bool, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.ExpectedCalls != nil && len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, audio)
		if args.Get(0) != nil {
			if speech, ok := args.Get(0).(bool); ok {
				return speech, args.Error(1)
			}
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return false, m.errorToReturn
		}
		return false, &VADError{
			Op:   "Process",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	// Simulate network delay if enabled
	if m.simulateNetworkDelay {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Simulate processing delay
	if m.processingDelay > 0 {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(m.processingDelay):
		}
	}

	// Default behavior
	return m.getNextResult(), nil
}

// ProcessStream implements the VADProvider interface.
func (m *AdvancedMockVADProvider) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan iface.VADResult, error) {
	resultCh := make(chan iface.VADResult, 10)

	go func() {
		defer close(resultCh)

		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioCh:
				if !ok {
					return
				}

				// Process audio chunk
				speech, err := m.Process(ctx, audio)
				if err != nil {
					resultCh <- iface.VADResult{
						HasVoice:   false,
						Confidence: 0.0,
						Error:      err,
					}
					return
				}

				confidence := 0.0
				if speech {
					confidence = 0.9
				}

				resultCh <- iface.VADResult{
					HasVoice:   speech,
					Confidence: confidence,
					Error:      nil,
				}
			}
		}
	}()

	return resultCh, nil
}

// getNextResult returns the next speech detection result in the list.
func (m *AdvancedMockVADProvider) getNextResult() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.speechResults) == 0 {
		return true
	}

	result := m.speechResults[m.resultIndex%len(m.speechResults)]
	m.resultIndex++
	return result
}

// GetCallCount returns the number of times Process has been called.
func (m *AdvancedMockVADProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// AssertVADProviderInterface ensures that a type implements the VADProvider interface.
func AssertVADProviderInterface(t *testing.T, provider iface.VADProvider) {
	t.Helper()
	assert.NotNil(t, provider, "VADProvider should not be nil")

	// Test Process method
	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	_, err := provider.Process(ctx, audio)
	// We don't care about the result, just that the method exists and can be called
	_ = err
}
