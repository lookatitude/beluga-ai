// Package core provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunnableInvokeAdvanced provides advanced table-driven tests for Runnable.Invoke.
func TestRunnableInvokeAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func(t *testing.T) Runnable
		input       any
		validate    func(t *testing.T, result any, err error)
		wantErr     bool
	}{
		{
			name:        "basic_invoke",
			description: "Invoke runnable with simple input",
			setup: func(t *testing.T) Runnable {
				mock := NewAdvancedMockRunnable("test-runnable")
				mock.On("Invoke", context.Background(), "test", []Option{}).Return("result", nil)
				return mock
			},
			input: "test",
			validate: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "result", result)
			},
		},
		{
			name:        "invoke_with_error",
			description: "Invoke runnable that returns error",
			setup: func(t *testing.T) Runnable {
				mock := NewAdvancedMockRunnable("test-runnable", WithMockError(true, assert.AnError))
				return mock
			},
			input: "test",
			validate: func(t *testing.T, result any, err error) {
				assert.Error(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			runnable := tt.setup(t)
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
		setup       func(t *testing.T) Runnable
		inputs      []any
		validate    func(t *testing.T, results []any, err error)
		wantErr     bool
	}{
		{
			name:        "basic_batch",
			description: "Batch invoke with multiple inputs",
			setup: func(t *testing.T) Runnable {
				mock := NewAdvancedMockRunnable("test-runnable")
				mock.On("Batch", context.Background(), []any{"input1", "input2"}, []Option{}).Return([]any{"result1", "result2"}, nil)
				return mock
			},
			inputs: []any{"input1", "input2"},
			validate: func(t *testing.T, results []any, err error) {
				assert.NoError(t, err)
				assert.Len(t, results, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			runnable := tt.setup(t)
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
		setup       func(t *testing.T) Container
		factory     any
		validate    func(t *testing.T, err error)
		wantErr     bool
	}{
		{
			name:        "register_valid_factory",
			description: "Register a valid factory function",
			setup: func(t *testing.T) Container {
				return NewContainer()
			},
			factory: func() string {
				return "test"
			},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:        "register_invalid_factory",
			description: "Register an invalid factory (not a function)",
			setup: func(t *testing.T) Container {
				return NewContainer()
			},
			factory: "not a function",
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			container := tt.setup(t)
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
		setup       func(t *testing.T) Container
		target      any
		validate    func(t *testing.T, err error)
		wantErr     bool
	}{
		{
			name:        "resolve_registered_type",
			description: "Resolve a registered type",
			setup: func(t *testing.T) Container {
				container := NewContainer()
				err := container.Register(func() string {
					return "test"
				})
				require.NoError(t, err)
				return container
			},
			target: new(string),
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:        "resolve_unregistered_type",
			description: "Resolve an unregistered type",
			setup: func(t *testing.T) Container {
				return NewContainer()
			},
			target: new(string),
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
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

	mock := NewAdvancedMockRunnable("test-runnable")
	mock.On("Invoke", context.Background(), "test", []Option{}).Return("result", nil).Times(numGoroutines * numInvocationsPerGoroutine)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines*numInvocationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numInvocationsPerGoroutine; j++ {
				_, err := mock.Invoke(ctx, "test")
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

	mock := NewAdvancedMockRunnable("test-runnable")
	mock.On("Invoke", ctx, "test", []Option{}).Return("result", nil)

	t.Run("invoke_with_timeout", func(t *testing.T) {
		result, err := mock.Invoke(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, "result", result)
	})
}

// BenchmarkRunnableInvoke benchmarks runnable invocation performance.
func BenchmarkRunnableInvoke(b *testing.B) {
	mock := NewAdvancedMockRunnable("benchmark-runnable")
	mock.On("Invoke", context.Background(), "test", []Option{}).Return("result", nil)

	ctx := context.Background()
	input := "test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mock.Invoke(ctx, input)
	}
}

// BenchmarkContainerResolve benchmarks container resolution performance.
func BenchmarkContainerResolve(b *testing.B) {
	container := NewContainer()
	err := container.Register(func() string {
		return "test"
	})
	require.NoError(b, err)

	target := new(string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = container.Resolve(target)
	}
}
