// Package turndetection provides advanced test utilities and comprehensive mocks for testing Turn Detection implementations.
package turndetection

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockTurnDetector provides a comprehensive mock implementation for testing.
type AdvancedMockTurnDetector struct {
	errorToReturn error
	mock.Mock
	detectorName         string
	turnResults          []bool
	callCount            int
	resultIndex          int
	processingDelay      time.Duration
	mu                   sync.RWMutex
	shouldError          bool
	simulateNetworkDelay bool
}

// NewAdvancedMockTurnDetector creates a new advanced mock with configurable behavior.
func NewAdvancedMockTurnDetector(detectorName string, opts ...MockOption) *AdvancedMockTurnDetector {
	m := &AdvancedMockTurnDetector{
		detectorName:    detectorName,
		turnResults:     []bool{false},
		processingDelay: 10 * time.Millisecond,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockOption configures the behavior of AdvancedMockTurnDetector.
type MockOption func(*AdvancedMockTurnDetector)

// WithDetectorName sets the detector name.
func WithDetectorName(name string) MockOption {
	return func(m *AdvancedMockTurnDetector) {
		m.detectorName = name
	}
}

// WithTurnResults sets the turn detection results to return.
func WithTurnResults(results ...bool) MockOption {
	return func(m *AdvancedMockTurnDetector) {
		m.turnResults = results
	}
}

// WithError configures the mock to return an error.
func WithError(err error) MockOption {
	return func(m *AdvancedMockTurnDetector) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithProcessingDelay sets the delay for processing.
func WithProcessingDelay(delay time.Duration) MockOption {
	return func(m *AdvancedMockTurnDetector) {
		m.processingDelay = delay
	}
}

// WithNetworkDelay enables network delay simulation.
func WithNetworkDelay(enabled bool) MockOption {
	return func(m *AdvancedMockTurnDetector) {
		m.simulateNetworkDelay = enabled
	}
}

// DetectTurn implements the TurnDetector interface.
func (m *AdvancedMockTurnDetector) DetectTurn(ctx context.Context, audio []byte) (bool, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Check if mock expectations are set up
	if m.ExpectedCalls != nil && len(m.ExpectedCalls) > 0 {
		args := m.Called(ctx, audio)
		if args.Get(0) != nil {
			if turn, ok := args.Get(0).(bool); ok {
				return turn, args.Error(1)
			}
		}
	}

	if m.shouldError {
		if m.errorToReturn != nil {
			return false, m.errorToReturn
		}
		return false, &TurnDetectionError{
			Op:   "DetectTurn",
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

// DetectTurnWithSilence implements the TurnDetector interface.
func (m *AdvancedMockTurnDetector) DetectTurnWithSilence(ctx context.Context, audio []byte, silenceDuration time.Duration) (bool, error) {
	// For simplicity, use the same logic as DetectTurn
	// In a real implementation, silence duration would influence the decision
	return m.DetectTurn(ctx, audio)
}

// getNextResult returns the next turn detection result in the list.
func (m *AdvancedMockTurnDetector) getNextResult() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.turnResults) == 0 {
		return false
	}

	result := m.turnResults[m.resultIndex%len(m.turnResults)]
	m.resultIndex++
	return result
}

// GetCallCount returns the number of times DetectTurn has been called.
func (m *AdvancedMockTurnDetector) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// AssertTurnDetectorInterface ensures that a type implements the TurnDetector interface.
func AssertTurnDetectorInterface(t *testing.T, detector iface.TurnDetector) {
	t.Helper()
	assert.NotNil(t, detector, "TurnDetector should not be nil")

	// Test DetectTurn method
	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}
	_, err := detector.DetectTurn(ctx, audio)
	// We don't care about the result, just that the method exists and can be called
	_ = err
}
