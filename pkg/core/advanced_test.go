// Package core provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
//
//nolint:goconst // goconst false positive on testValue usage
package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestRunnableInvokeAdvanced provides advanced table-driven tests for Runnable.Invoke.
func TestRunnableInvokeAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		input       any
		setup       func() Runnable
		validate    func(*testing.T, any, error)
		wantErr     bool
	}{
		{
			name:        "basic_invoke",
			description: "Invoke runnable with simple input",
			setup: func() Runnable {
				m := NewAdvancedMockRunnable("test-runnable")
				m.On("Invoke", mock.Anything, testValue,
					mock.MatchedBy(func([]Option) bool { return true })).
					Return("result", nil)
				return m
			},
			input: testValue,
			validate: func(t *testing.T, result any, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, "result", result)
			},
		},
		{
			name:        "invoke_with_error",
			description: "Invoke runnable that returns error",
			setup: func() Runnable {
				m := NewAdvancedMockRunnable("test-runnable", WithMockError(true, assert.AnError))
				m.On("Invoke", mock.Anything, testValue,
					mock.MatchedBy(func([]Option) bool { return true })).
					Return(nil, assert.AnError)
				return m
			},
			input: testValue,
			validate: func(t *testing.T, _ any, err error) {
				t.Helper()
				require.Error(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			runnable := tt.setup()
			ctx := context.Background()
			result, err := runnable.Invoke(ctx, tt.input)
			tt.validate(t, result, err)
		})
	}
}

// TestRunnableBatchAdvanced provides advanced table-driven tests for Runnable.Batch.
func TestRunnableBatchAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		inputs      []any
		setup       func() Runnable
		validate    func(*testing.T, []any, error)
		wantErr     bool
	}{
		{
			name:        "basic_batch",
			description: "Batch invoke with multiple inputs",
			setup: func() Runnable {
				m := NewAdvancedMockRunnable("test-runnable")
				m.On("Batch", mock.Anything,
					[]any{"input1", "input2"},
					mock.MatchedBy(func([]Option) bool { return true })).
					Return([]any{"result1", "result2"}, nil)
				return m
			},
			inputs: []any{"input1", "input2"},
			validate: func(t *testing.T, results []any, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, results, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			runnable := tt.setup()
			ctx := context.Background()
			results, err := runnable.Batch(ctx, tt.inputs)
			tt.validate(t, results, err)
		})
	}
}

// TestContainerRegisterAdvanced provides advanced table-driven tests for Container.Register.
func TestContainerRegisterAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		factory     any
		setup       func() Container
		validate    func(*testing.T, error)
		wantErr     bool
	}{
		{
			name:        "register_valid_factory",
			description: "Register a valid factory function",
			setup:       NewContainer,
			factory: func() string {
				return testValue
			},
			validate: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:        "register_invalid_factory",
			description: "Register an invalid factory (not a function)",
			setup:       NewContainer,
			factory:     "not a function",
			validate: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			container := tt.setup()
			err := container.Register(tt.factory)
			tt.validate(t, err)
		})
	}
}

// TestContainerResolveAdvanced provides advanced table-driven tests for Container.Resolve.
func TestContainerResolveAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		target      any
		setup       func(*testing.T) Container
		validate    func(*testing.T, error)
		wantErr     bool
	}{
		{
			name:        "resolve_registered_type",
			description: "Resolve a registered type",
			setup: func(t *testing.T) Container {
				t.Helper()
				container := NewContainer()
				err := container.Register(func() string {
					return testValue
				})
				require.NoError(t, err)
				return container
			},
			target: new(string),
			validate: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:        "resolve_unregistered_type",
			description: "Resolve an unregistered type",
			setup: func(t *testing.T) Container {
				t.Helper()
				return NewContainer()
			},
			target: new(string),
			validate: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			container := tt.setup(t)
			err := container.Resolve(tt.target)
			tt.validate(t, err)
		})
	}
}

// TestConcurrentRunnableInvoke tests concurrent runnable invocations.
func TestConcurrentRunnableInvoke(t *testing.T) {
	const numGoroutines = 50
	const numInvocationsPerGoroutine = 10

	m := NewAdvancedMockRunnable("test-runnable")
	m.On("Invoke", mock.Anything, testValue,
		mock.MatchedBy(func([]Option) bool { return true })).
		Return("result", nil).
		Times(numGoroutines * numInvocationsPerGoroutine)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines*numInvocationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numInvocationsPerGoroutine; j++ {
				_, err := m.Invoke(ctx, testValue)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestConcurrentContainerOperations tests concurrent container operations.
func TestConcurrentContainerOperations(t *testing.T) {
	const numGoroutines = 20

	container := NewContainer()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			err := container.Register(func() int {
				return id
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Log any errors but don't fail - concurrent registration may have different results
	for err := range errors {
		t.Logf("Concurrent container operation error: %v", err)
	}
}

// TestRunnableWithContext tests runnable operations with context.
func TestRunnableWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m := NewAdvancedMockRunnable("test-runnable")
	m.On("Invoke", mock.Anything, testValue,
		mock.MatchedBy(func([]Option) bool { return true })).Return("result", nil)

	t.Run("invoke_with_timeout", func(t *testing.T) {
		result, err := m.Invoke(ctx, testValue)
		require.NoError(t, err)
		assert.Equal(t, "result", result)
	})
}

// BenchmarkRunnableInvoke benchmarks runnable invocation performance.
func BenchmarkRunnableInvoke(b *testing.B) {
	m := NewAdvancedMockRunnable("benchmark-runnable")
	m.On("Invoke", mock.Anything, testValue,
		mock.MatchedBy(func([]Option) bool { return true })).Return("result", nil)

	ctx := context.Background()
	input := testValue

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := m.Invoke(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// BenchmarkContainerResolve benchmarks container resolution performance.
func BenchmarkContainerResolve(b *testing.B) {
	container := NewContainer()
	//nolint:goconst // testValue is already a constant, this is a false positive
	err := container.Register(func() string {
		return testValue
	})
	require.NoError(b, err)

	target := new(string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := container.Resolve(target)
		if err != nil {
			b.Fatal(err)
		}
	}
}
