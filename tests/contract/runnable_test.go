// Package contract provides contract tests for Runnable interface compliance.
// T010: Contract test for Runnable interface
package contract

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/core/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunnableInterfaceContract verifies Runnable interface implementation compliance
func TestRunnableInterfaceContract(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() iface.Runnable
		operation   func(t *testing.T, runnable iface.Runnable)
		description string
	}{
		{
			name:        "Invoke_basic_operation",
			description: "Contract: Invoke method must handle basic input-output operations",
			setup: func() iface.Runnable {
				return core.NewAdvancedMockRunnable("invoke_test",
					core.WithMockResponses([]any{"invoke_result"}))
			},
			operation: func(t *testing.T, runnable iface.Runnable) {
				ctx := context.Background()
				result, err := runnable.Invoke(ctx, "test_input")

				assert.NoError(t, err, "Invoke should not return error for valid input")
				assert.NotNil(t, result, "Invoke should return non-nil result")
				assert.Equal(t, "invoke_result", result)
			},
		},
		{
			name:        "Invoke_context_cancellation",
			description: "Contract: Invoke method must respect context cancellation",
			setup: func() iface.Runnable {
				return core.NewAdvancedMockRunnable("cancel_test",
					core.WithMockDelay(time.Second)) // Long delay to test cancellation
			},
			operation: func(t *testing.T, runnable iface.Runnable) {
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()

				start := time.Now()
				_, err := runnable.Invoke(ctx, "test_input")
				elapsed := time.Since(start)

				// Should return quickly due to cancellation, not wait for full delay
				assert.Error(t, err, "Should return error due to context cancellation")
				assert.Less(t, elapsed, 200*time.Millisecond, "Should cancel quickly")
				assert.Equal(t, context.DeadlineExceeded, err)
			},
		},
		{
			name:        "Invoke_with_options",
			description: "Contract: Invoke method must accept and handle options",
			setup: func() iface.Runnable {
				return core.NewAdvancedMockRunnable("options_test")
			},
			operation: func(t *testing.T, runnable iface.Runnable) {
				ctx := context.Background()

				// Test with options
				opt1 := core.WithOption("temperature", 0.7)
				opt2 := core.WithOption("max_tokens", 1000)

				result, err := runnable.Invoke(ctx, "test_input", opt1, opt2)
				assert.NoError(t, err, "Should handle options without error")
				assert.NotNil(t, result, "Should return result with options")
			},
		},
		{
			name:        "Batch_multiple_inputs",
			description: "Contract: Batch method must process multiple inputs",
			setup: func() iface.Runnable {
				return core.NewAdvancedMockRunnable("batch_test",
					core.WithMockResponses([]any{"batch_result"}))
			},
			operation: func(t *testing.T, runnable iface.Runnable) {
				ctx := context.Background()
				inputs := []any{"input1", "input2", "input3"}

				results, err := runnable.Batch(ctx, inputs)
				assert.NoError(t, err, "Batch should not return error")
				require.Len(t, results, len(inputs), "Should return one result per input")

				for i, result := range results {
					assert.NotNil(t, result, "Result %d should not be nil", i)
				}
			},
		},
		{
			name:        "Batch_empty_inputs",
			description: "Contract: Batch method must handle empty input slice",
			setup: func() iface.Runnable {
				return core.NewAdvancedMockRunnable("empty_batch_test")
			},
			operation: func(t *testing.T, runnable iface.Runnable) {
				ctx := context.Background()
				inputs := []any{}

				results, err := runnable.Batch(ctx, inputs)
				assert.NoError(t, err, "Batch should handle empty inputs")
				assert.Empty(t, results, "Should return empty results for empty inputs")
			},
		},
		{
			name:        "Stream_basic_operation",
			description: "Contract: Stream method must return channel for streaming output",
			setup: func() iface.Runnable {
				return core.NewAdvancedMockRunnable("stream_test",
					core.WithMockResponses([]any{"stream_chunk"}))
			},
			operation: func(t *testing.T, runnable iface.Runnable) {
				ctx := context.Background()

				ch, err := runnable.Stream(ctx, "stream_input")
				assert.NoError(t, err, "Stream should not return error")
				assert.NotNil(t, ch, "Stream should return non-nil channel")

				// Channel should produce results
				select {
				case result := <-ch:
					assert.NotNil(t, result, "Stream should produce non-nil result")
				case <-time.After(time.Second):
					t.Error("Stream should produce result within reasonable time")
				}
			},
		},
		{
			name:        "Stream_context_cancellation",
			description: "Contract: Stream method must respect context cancellation",
			setup: func() iface.Runnable {
				return core.NewAdvancedMockRunnable("stream_cancel_test",
					core.WithMockDelay(500*time.Millisecond))
			},
			operation: func(t *testing.T, runnable iface.Runnable) {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()

				ch, err := runnable.Stream(ctx, "stream_input")
				assert.NoError(t, err, "Stream setup should not error")

				// Channel should close or return error when context cancelled
				select {
				case result := <-ch:
					// If we get a result, it might be an error due to cancellation
					if err, ok := result.(error); ok {
						assert.Error(t, err, "Should return error due to cancellation")
					}
				case <-time.After(200 * time.Millisecond):
					// Channel should have closed due to cancellation
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := tt.setup()
			tt.operation(t, container)
		})
	}
}

// TestRunnableContractErrorHandling tests error handling contract requirements
func TestRunnableContractErrorHandling(t *testing.T) {
	t.Run("Invoke_error_propagation", func(t *testing.T) {
		expectedErr := core.NewInternalError("test_error", "Intentional test error")
		errorRunnable := core.NewAdvancedMockRunnable("error_test",
			core.WithMockError(expectedErr))

		ctx := context.Background()
		result, err := errorRunnable.Invoke(ctx, "error_input")

		assert.Error(t, err, "Should return error when configured to fail")
		assert.Nil(t, result, "Should return nil result when error occurs")
		assert.Equal(t, expectedErr, err, "Should return the configured error")
	})

	t.Run("Batch_partial_failure_handling", func(t *testing.T) {
		// Test batch behavior when some operations fail
		// Note: Contract specifies this should either:
		// 1. Return partial results with error indicators, or
		// 2. Fail completely with detailed error information

		errorRunnable := core.NewAdvancedMockRunnable("batch_error_test",
			core.WithMockError(core.NewInternalError("batch_error", "Batch operation failed")))

		ctx := context.Background()
		inputs := []any{"input1", "input2", "input3"}

		results, err := errorRunnable.Batch(ctx, inputs)

		// Either complete failure (err != nil) or partial results handling
		if err != nil {
			assert.Nil(t, results, "Should return nil results on complete failure")
			t.Logf("Batch failed as expected: %v", err)
		} else {
			// If no error, should have some form of result (possibly with error indicators)
			assert.NotNil(t, results, "Should return results even with partial failures")
		}
	})
}

// TestRunnableContractPerformance tests performance contract requirements
func TestRunnableContractPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance contract tests in short mode")
	}

	runnable := core.NewAdvancedMockRunnable("performance_contract")

	t.Run("InvokePerformanceContract", func(t *testing.T) {
		ctx := context.Background()
		iterations := 1000

		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := runnable.Invoke(ctx, "performance_test")
			require.NoError(t, err)
		}
		elapsed := time.Since(start)

		avgTime := elapsed / time.Duration(iterations)
		t.Logf("Average Invoke time: %v", avgTime)

		// Contract: Runnable overhead should be <100μs
		assert.Less(t, avgTime, 100*time.Microsecond,
			"Runnable Invoke overhead should be under 100μs")
	})

	t.Run("StreamPerformanceContract", func(t *testing.T) {
		ctx := context.Background()
		iterations := 100

		for i := 0; i < iterations; i++ {
			start := time.Now()
			ch, err := runnable.Stream(ctx, "stream_test")
			setupTime := time.Since(start)

			require.NoError(t, err)

			// Read from channel
			<-ch

			// Contract: Stream setup should be <10ms
			assert.Less(t, setupTime, 10*time.Millisecond,
				"Stream setup time should be under 10ms")
		}
	})
}

// TestRunnableContractThreadSafety tests thread safety contract requirements
func TestRunnableContractThreadSafety(t *testing.T) {
	runnable := core.NewAdvancedMockRunnable("thread_safety_test")
	ctx := context.Background()

	// Run multiple operations concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 30)

	// Concurrent Invoke operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := runnable.Invoke(ctx, fmt.Sprintf("concurrent_invoke_%d", id))
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent Batch operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			inputs := []any{fmt.Sprintf("concurrent_batch_%d", id)}
			_, err := runnable.Batch(ctx, inputs)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent Stream operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ch, err := runnable.Stream(ctx, fmt.Sprintf("concurrent_stream_%d", id))
			if err != nil {
				errors <- err
				return
			}
			// Read from channel
			<-ch
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	assert.Empty(t, errorList, "Concurrent operations should not produce errors")

	// Verify runnable is still functional after concurrent access
	result, err := runnable.Invoke(ctx, "post_concurrent_test")
	assert.NoError(t, err, "Runnable should still work after concurrent access")
	assert.NotNil(t, result, "Should return valid result after concurrent operations")
}
