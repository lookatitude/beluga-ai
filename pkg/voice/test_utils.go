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
	mock.Mock

	// Configuration
	name      string
	callCount int
	mu        sync.RWMutex

	// Configurable behavior
	shouldError   bool
	errorToReturn error
	simulateDelay time.Duration

	// Health check data
	healthState     string
	lastHealthCheck time.Time
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
	NumGoroutines int
	TestDuration  time.Duration
	testFunc      func() error
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
