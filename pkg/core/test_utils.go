// Package core provides advanced test utilities and comprehensive mocks for testing core implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package core

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// AdvancedMockRunnable provides a comprehensive mock implementation for testing Runnable.
type AdvancedMockRunnable struct {
	mock.Mock

	// Configuration
	name      string
	callCount int
	mu        sync.RWMutex

	// Configurable behavior
	errorToReturn error
	simulateDelay time.Duration
	shouldError   bool

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockRunnable creates a new advanced mock runnable with configurable behavior.
func NewAdvancedMockRunnable(name string, opts ...MockRunnableOption) *AdvancedMockRunnable {
	m := &AdvancedMockRunnable{
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

// MockRunnableOption defines functional options for mock configuration.
type MockRunnableOption func(*AdvancedMockRunnable)

// WithMockError configures the mock to return errors.
func WithMockError(shouldError bool, err error) MockRunnableOption {
	return func(m *AdvancedMockRunnable) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithMockDelay adds artificial delay to mock operations.
func WithMockDelay(delay time.Duration) MockRunnableOption {
	return func(m *AdvancedMockRunnable) {
		m.simulateDelay = delay
	}
}

// Invoke implements the Runnable interface.
func (m *AdvancedMockRunnable) Invoke(ctx context.Context, input any, options ...Option) (any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	args := m.Called(ctx, input, options)

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError && m.errorToReturn != nil {
		return nil, m.errorToReturn
	}

	if args.Get(0) == nil {
		if err := args.Error(1); err != nil {
			return nil, fmt.Errorf("mock invoke error: %w", err)
		}
		// Return empty result when no error and no value
		return "", nil
	}
	if err := args.Error(1); err != nil {
		return args.Get(0), fmt.Errorf("mock invoke error: %w", err)
	}
	return args.Get(0), nil
}

// Batch implements the Runnable interface.
func (m *AdvancedMockRunnable) Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error) {
	m.mu.Lock()
	m.callCount += len(inputs)
	m.mu.Unlock()

	args := m.Called(ctx, inputs, options)

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError && m.errorToReturn != nil {
		return nil, m.errorToReturn
	}

	if args.Get(0) == nil {
		if err := args.Error(1); err != nil {
			return nil, fmt.Errorf("mock batch error: %w", err)
		}
		// Return empty slice when no error and no value
		return []any{}, nil
	}
	if err := args.Error(1); err != nil {
		if val := args.Get(0); val != nil {
			if result, ok := val.([]any); ok {
				return result, fmt.Errorf("mock batch error: %w", err)
			}
		}
		return []any{}, fmt.Errorf("mock batch error: %w", err)
	}
	if val := args.Get(0); val != nil {
		if result, ok := val.([]any); ok {
			return result, nil
		}
	}
	return []any{}, nil
}

// Stream implements the Runnable interface.
func (m *AdvancedMockRunnable) Stream(ctx context.Context, input any, options ...Option) (<-chan any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	args := m.Called(ctx, input, options)

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	if m.shouldError && m.errorToReturn != nil {
		return nil, m.errorToReturn
	}

	if args.Get(0) == nil {
		if err := args.Error(1); err != nil {
			return nil, fmt.Errorf("mock stream error: %w", err)
		}
		// Return empty channel when no error and no value
		emptyChan := make(chan any)
		close(emptyChan)
		return emptyChan, nil
	}
	if err := args.Error(1); err != nil {
		if val := args.Get(0); val != nil {
			if result, ok := val.(<-chan any); ok {
				return result, fmt.Errorf("mock stream error: %w", err)
			}
		}
		emptyChan := make(chan any)
		close(emptyChan)
		return emptyChan, fmt.Errorf("mock stream error: %w", err)
	}
	if val := args.Get(0); val != nil {
		if result, ok := val.(<-chan any); ok {
			return result, nil
		}
	}
	emptyChan := make(chan any)
	close(emptyChan)
	return emptyChan, nil
}

// GetCallCount returns the number of times the mock was called.
func (m *AdvancedMockRunnable) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// AdvancedMockContainer provides a comprehensive mock implementation for testing Container.
type AdvancedMockContainer struct {
	mock.Mock

	// Configuration
	name      string
	callCount int
	mu        sync.RWMutex

	// Configurable behavior
	errorToReturn error
	shouldError   bool
}

// NewAdvancedMockContainer creates a new advanced mock container with configurable behavior.
func NewAdvancedMockContainer(name string, opts ...MockContainerOption) *AdvancedMockContainer {
	m := &AdvancedMockContainer{
		name: name,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// MockContainerOption defines functional options for mock configuration.
type MockContainerOption func(*AdvancedMockContainer)

// WithContainerMockError configures the mock to return errors.
func WithContainerMockError(shouldError bool, err error) MockContainerOption {
	return func(m *AdvancedMockContainer) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// Register implements the Container interface.
func (m *AdvancedMockContainer) Register(factoryFunc any) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	args := m.Called(factoryFunc)

	if m.shouldError && m.errorToReturn != nil {
		return m.errorToReturn
	}

	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock resolve error: %w", err)
	}
	return nil
}

// Resolve implements the Container interface.
func (m *AdvancedMockContainer) Resolve(target any) error {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	args := m.Called(target)

	if m.shouldError && m.errorToReturn != nil {
		return m.errorToReturn
	}

	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock resolve error: %w", err)
	}
	return nil
}

// MustResolve implements the Container interface.
func (m *AdvancedMockContainer) MustResolve(target any) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	m.Called(target)
}

// Has implements the Container interface.
func (m *AdvancedMockContainer) Has(typ interface{}) bool {
	args := m.Called(typ)
	return args.Bool(0)
}

// Clear implements the Container interface.
func (m *AdvancedMockContainer) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Called()
}

// Singleton implements the Container interface.
func (m *AdvancedMockContainer) Singleton(instance any) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	m.Called(instance)
}

// CheckHealth implements the HealthChecker interface.
func (m *AdvancedMockContainer) CheckHealth(ctx context.Context) error {
	args := m.Called(ctx)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock check health error: %w", err)
	}
	return nil
}

// ConcurrentTestRunner provides utilities for concurrent testing.
type ConcurrentTestRunner struct {
	testFunc      func() error
	NumGoroutines int
	TestDuration  time.Duration
}

// NewConcurrentTestRunner creates a new concurrent test runner.
func NewConcurrentTestRunner(
	numGoroutines int,
	testDuration time.Duration,
	testFunc func() error,
) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		NumGoroutines: numGoroutines,
		TestDuration:  testDuration,
		testFunc:      testFunc,
	}
}

// Run executes the concurrent test.
func (r *ConcurrentTestRunner) Run(t *testing.T) {
	t.Helper()
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
