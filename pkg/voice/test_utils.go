// Package voice provides advanced test utilities and comprehensive mocks for testing voice implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package voice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// AdvancedMockVoiceComponent provides a comprehensive mock implementation for testing voice components.
type AdvancedMockVoiceComponent struct {
	lastHealthCheck time.Time
	errorToReturn   error
	mock.Mock
	name          string
	healthState   string
	callCount     int
	simulateDelay time.Duration
	mu            sync.RWMutex
	shouldError   bool
}

// NewAdvancedMockVoiceComponent creates a new advanced mock voice component with configurable behavior.
func NewAdvancedMockVoiceComponent(name string, opts ...MockVoiceComponentOption) *AdvancedMockVoiceComponent {
	m := &AdvancedMockVoiceComponent{
		name:            name,
		healthState:     "healthy",
		lastHealthCheck: time.Now(),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockVoiceComponentOption defines functional options for mock configuration.
type MockVoiceComponentOption func(*AdvancedMockVoiceComponent)

// WithMockError configures the mock to return errors.
func WithMockError(shouldError bool, err error) MockVoiceComponentOption {
	return func(m *AdvancedMockVoiceComponent) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithMockDelay adds artificial delay to mock operations.
func WithMockDelay(delay time.Duration) MockVoiceComponentOption {
	return func(m *AdvancedMockVoiceComponent) {
		m.simulateDelay = delay
	}
}

// GetCallCount returns the number of times the mock was called.
func (m *AdvancedMockVoiceComponent) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// ConcurrentTestRunner provides utilities for concurrent testing.
type ConcurrentTestRunner struct {
	testFunc      func() error
	NumGoroutines int
	TestDuration  time.Duration
}

// NewConcurrentTestRunner creates a new concurrent test runner.
func NewConcurrentTestRunner(numGoroutines int, testDuration time.Duration, testFunc func() error) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		NumGoroutines: numGoroutines,
		TestDuration:  testDuration,
		testFunc:      testFunc,
	}
}

// Run executes the concurrent test.
func (r *ConcurrentTestRunner) Run(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(r.NumGoroutines)

	errors := make(chan error, r.NumGoroutines)
	ctx, cancel := context.WithTimeout(context.Background(), r.TestDuration)
	defer cancel()

	for i := 0; i < r.NumGoroutines; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := r.testFunc(); err != nil {
						errors <- err
					}
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent test error: %v", err)
		}
	}
}

// EXCLUSION DOCUMENTATION
//
// The following code paths are excluded from test coverage:
//
// 1. Provider-specific implementations (providers/*/*.go, backend/providers/*/*.go, etc.)
//    - Reason: These require actual external service connections (Twilio, LiveKit, Pipecat, etc.)
//    - Coverage: Tested via integration tests with mocks where possible
//    - Files: pkg/voice/providers/*/*.go, pkg/voice/backend/providers/*/*.go, etc.
//
// 2. Internal implementations (internal/*.go, backend/internal/*.go, etc.)
//    - Reason: Internal implementation details, tested indirectly through public APIs
//    - Coverage: Tested via integration tests and public API tests
//    - Files: pkg/voice/internal/*.go, pkg/voice/backend/internal/*.go, etc.
//
// 3. Audio processing and real-time streaming paths
//    - Reason: Requires actual audio hardware/streams, timing-dependent
//    - Coverage: Tested via integration tests with mock audio streams
//    - Files: Various provider implementations with audio processing
//
// 4. Network and WebSocket connection handling
//    - Reason: Requires actual network connections, difficult to mock reliably
//    - Coverage: Tested via integration tests with test servers
//    - Files: WebSocket and network-related provider code
//
// 5. OS-level and platform-specific code
//    - Reason: Cannot simulate OS-level errors in unit tests
//    - Coverage: Error types tested, actual OS failures tested in integration tests
//    - Files: Platform-specific implementations
//
// Note: Integration tests in tests/integration/voice/ provide comprehensive coverage
// for cross-package interactions, real-world usage scenarios, and provider integrations.
