// Package noise provides advanced test utilities and comprehensive mocks for testing Noise Cancellation implementations.
package noise

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockNoiseCancellation provides a comprehensive mock implementation for testing
type AdvancedMockNoiseCancellation struct {
	mock.Mock

	// Configuration
	cancellationName string
	callCount        int
	mu               sync.RWMutex

	// Configurable behavior
	shouldError          bool
	errorToReturn        error
	processedAudio       [][]byte
	audioIndex           int
	processingDelay      time.Duration
	simulateNetworkDelay bool
	noiseReductionLevel  float64
}

// NewAdvancedMockNoiseCancellation creates a new advanced mock with configurable behavior
func NewAdvancedMockNoiseCancellation(cancellationName string, opts ...MockOption) *AdvancedMockNoiseCancellation {
	m := &AdvancedMockNoiseCancellation{
		cancellationName:    cancellationName,
		processedAudio:      [][]byte{},
		processingDelay:     10 * time.Millisecond,
		noiseReductionLevel: 0.5,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockNoiseCancellation
type MockOption func(*AdvancedMockNoiseCancellation)

// WithCancellationName sets the cancellation name
func WithCancellationName(name string) MockOption {
	return func(m *AdvancedMockNoiseCancellation) {
		m.cancellationName = name
	}
}

// WithProcessedAudio sets the processed audio to return
func WithProcessedAudio(audio ...[]byte) MockOption {
	return func(m *AdvancedMockNoiseCancellation) {
		m.processedAudio = audio
	}
}

// WithError configures the mock to return an error
func WithError(err error) MockOption {
	return func(m *AdvancedMockNoiseCancellation) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithProcessingDelay sets the delay for processing
func WithProcessingDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockNoiseCancellation) {
		m.processingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockNoiseCancellation) {
		m.simulateNetworkDelay = enabled
	}
}

// WithMockNoiseReductionLevel sets the noise reduction level for the mock
func WithMockNoiseReductionLevel(level float64) MockOption {
	return func(m *AdvancedMockNoiseCancellation) {
		m.noiseReductionLevel = level
	}
}

// Process implements the NoiseCancellation interface
func (m *AdvancedMockNoiseCancellation) Process(ctx context.Context, audio []byte) ([]byte, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.Mock.ExpectedCalls != nil && len(m.Mock.ExpectedCalls) > 0 {
		args := m.Called(ctx, audio)
		if args.Get(0) != nil {
			if processed, ok := args.Get(0).([]byte); ok {
				return processed, args.Error(1)
			}
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, &NoiseCancellationError{
			Op:   "Process",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	// Simulate network delay if enabled
	if m.simulateNetworkDelay {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Simulate processing delay
	if m.processingDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.processingDelay):
		}
	}

	// Default behavior: return processed audio or original audio with noise reduction
	processed := m.getNextProcessedAudio()
	if processed != nil {
		return processed, nil
	}

	// If no processed audio provided, return original (simulating no change)
	return audio, nil
}

// ProcessStream implements the NoiseCancellation interface
func (m *AdvancedMockNoiseCancellation) ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan []byte, error) {
	processedCh := make(chan []byte, 10)

	if m.shouldError {
		close(processedCh)
		if m.errorToReturn != nil {
			return processedCh, m.errorToReturn
		}
		return processedCh, &NoiseCancellationError{
			Op:   "ProcessStream",
			Code: ErrCodeInternalError,
			Err:  nil,
		}
	}

	go func() {
		defer close(processedCh)

		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioCh:
				if !ok {
					return
				}

				// Process audio chunk
				processed, err := m.Process(ctx, audio)
				if err != nil {
					return
				}

				select {
				case <-ctx.Done():
					return
				case processedCh <- processed:
				}
			}
		}
	}()

	return processedCh, nil
}

// getNextProcessedAudio returns the next processed audio in the list
func (m *AdvancedMockNoiseCancellation) getNextProcessedAudio() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.processedAudio) == 0 {
		return nil
	}

	audio := m.processedAudio[m.audioIndex%len(m.processedAudio)]
	m.audioIndex++
	return audio
}

// GetCallCount returns the number of times Process has been called
func (m *AdvancedMockNoiseCancellation) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// AssertNoiseCancellationInterface ensures that a type implements the NoiseCancellation interface
func AssertNoiseCancellationInterface(t *testing.T, cancellation iface.NoiseCancellation) {
	assert.NotNil(t, cancellation, "NoiseCancellation should not be nil")

	// Test Process method
	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	_, err := cancellation.Process(ctx, audio)
	// We don't care about the result, just that the method exists and can be called
	_ = err
}
