package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CustomFailingRunnable is a custom implementation for specific test cases
type CustomFailingRunnable struct {
	name  string
	error error
}

func (c *CustomFailingRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	return nil, c.error
}

func (c *CustomFailingRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = nil
	}
	return results, c.error
}

func (c *CustomFailingRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	close(ch)
	return ch, c.error
}

// ConditionalFailingRunnable fails based on input conditions
type ConditionalFailingRunnable struct {
	name string
}

func (c *ConditionalFailingRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	inputMap := input.(map[string]any)
	if inputMap["should_fail"] == true {
		return nil, fmt.Errorf("conditional failure for input: %v", inputMap["id"])
	}
	return map[string]any{
		"result": fmt.Sprintf("success_%v", inputMap["id"]),
		"id":     inputMap["id"],
	}, nil
}

func (c *ConditionalFailingRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := c.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (c *ConditionalFailingRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := c.Invoke(ctx, input, opts...)
		if err != nil {
			return
		}
		ch <- result
	}()
	return ch, nil
}

// FailingRunnable simulates various types of failures
type FailingRunnable struct {
	name           string
	failOnAttempt  int
	currentAttempt int64
	failType       string
	mutex          sync.Mutex
}

func (f *FailingRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	attempt := atomic.AddInt64(&f.currentAttempt, 1)

	switch f.failType {
	case "permanent":
		return nil, fmt.Errorf("permanent failure in %s", f.name)

	case "transient":
		if int(attempt) <= f.failOnAttempt {
			return nil, fmt.Errorf("transient failure attempt %d in %s", attempt, f.name)
		}
		return map[string]any{
			"result":       fmt.Sprintf("%s_success", f.name),
			"attempts":     attempt,
			"recovered_at": attempt,
		}, nil

	case "timeout":
		select {
		case <-time.After(2 * time.Second):
			return map[string]any{"result": "timeout_success"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}

	case "panic":
		if int(attempt) == f.failOnAttempt {
			panic(fmt.Sprintf("simulated panic in %s", f.name))
		}
		// After panicking, return an error to indicate the panic was handled
		return nil, fmt.Errorf("panic occurred and was recovered in %s", f.name)

	case "resource_exhausted":
		return nil, fmt.Errorf("resource exhausted in %s", f.name)

	default:
		return map[string]any{"result": fmt.Sprintf("%s_success", f.name)}, nil
	}
}

func (f *FailingRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := f.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (f *FailingRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := f.Invoke(ctx, input, opts...)
		if err != nil {
			return
		}
		ch <- result
	}()
	return ch, nil
}

// RecoveryRunnable simulates recovery mechanisms
type RecoveryRunnable struct {
	name      string
	recovered bool
	mutex     sync.Mutex
}

func (r *RecoveryRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.recovered = true
	return map[string]any{
		"recovered":     true,
		"recovery_step": r.name,
		"input":         input,
	}, nil
}

func (r *RecoveryRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := r.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (r *RecoveryRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := r.Invoke(ctx, input, opts...)
		if err != nil {
			return
		}
		ch <- result
	}()
	return ch, nil
}

// Test permanent failure scenarios
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestPermanentFailureScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("single step permanent failure", func(t *testing.T) {
		permanentFailure := &FailingRunnable{
			name:     "permanent-step",
			failType: "permanent",
		}

		chain, err := orch.CreateChain([]core.Runnable{permanentFailure})
		require.NoError(t, err)

		_, err = chain.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permanent failure")
	})

	t.Run("chain with permanent failure in middle", func(t *testing.T) {
		workingStep1 := &FailingRunnable{name: "working-1", failType: "success"}
		permanentFailure := &FailingRunnable{name: "permanent", failType: "permanent"}
		workingStep2 := &FailingRunnable{name: "working-2", failType: "success"}

		chain, err := orch.CreateChain([]core.Runnable{workingStep1, permanentFailure, workingStep2})
		require.NoError(t, err)

		_, err = chain.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permanent failure")
		assert.Contains(t, err.Error(), "permanent")
	})
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

// Test transient failure and recovery scenarios
func TestTransientFailureRecovery(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("transient failure with eventual success", func(t *testing.T) {
		transientFailure := &FailingRunnable{
			name:          "transient-step",
			failType:      "transient",
			failOnAttempt: 2, // Fail first 2 attempts, succeed on 3rd
		}

		chain, err := orch.CreateChain([]core.Runnable{transientFailure})
		require.NoError(t, err)

		result, err := chain.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, resultMap, "result")
		assert.Contains(t, resultMap, "attempts")
		assert.Equal(t, int64(3), resultMap["attempts"])
	})

	t.Run("transient failure exceeds retry limit", func(t *testing.T) {
		transientFailure := &FailingRunnable{
			name:          "persistent-transient",
			failType:      "transient",
			failOnAttempt: 5, // Fail more than typical retry limit (3)
		}

		chain, err := orch.CreateChain([]core.Runnable{transientFailure})
		require.NoError(t, err)

		_, err = chain.Invoke(context.Background(), map[string]any{"input": "test"})
		// This should fail after exhausting all retry attempts
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "after 3 attempts")
	})
}

// Test timeout scenarios
func TestTimeoutScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("step timeout", func(t *testing.T) {
		slowStep := &FailingRunnable{
			name:     "slow-step",
			failType: "timeout",
		}

		chain, err := orch.CreateChain([]core.Runnable{slowStep})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err = chain.Invoke(ctx, map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "Expected context deadline exceeded error, got: %v", err)
	})

	t.Run("chain timeout configuration", func(t *testing.T) {
		slowStep := &FailingRunnable{
			name:     "slow-step",
			failType: "timeout",
		}

		chain, err := orch.CreateChain([]core.Runnable{slowStep})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		result, err := chain.Invoke(ctx, map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "Expected context deadline exceeded error, got: %v", err)
		assert.Nil(t, result)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	})
}

// Test panic recovery
func TestPanicRecovery(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("single step panic", func(t *testing.T) {
		panicStep := &FailingRunnable{
			name:          "panic-step",
			failType:      "panic",
			failOnAttempt: 1,
		}

		chain, err := orch.CreateChain([]core.Runnable{panicStep})
		require.NoError(t, err)

		assert.NotPanics(t, func() {
			_, err = chain.Invoke(context.Background(), map[string]any{"input": "test"})
			// Should fail gracefully, not panic
			assert.Error(t, err)
		})
	})

	t.Run("panic in middle of chain", func(t *testing.T) {
		workingStep := &FailingRunnable{name: "working", failType: "success"}
		panicStep := &FailingRunnable{
			name:          "panic-middle",
			failType:      "panic",
			failOnAttempt: 1,
		}

		chain, err := orch.CreateChain([]core.Runnable{workingStep, panicStep})
		require.NoError(t, err)

		assert.NotPanics(t, func() {
			_, err = chain.Invoke(context.Background(), map[string]any{"input": "test"})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			assert.Error(t, err)
		})
	})
}

// Test resource exhaustion scenarios
func TestResourceExhaustionScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("resource exhausted in single step", func(t *testing.T) {
		resourceFailure := &FailingRunnable{
			name:     "resource-step",
			failType: "resource_exhausted",
		}

		chain, err := orch.CreateChain([]core.Runnable{resourceFailure})
		require.NoError(t, err)

		_, err = chain.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource exhausted")
	})

	t.Run("resource exhausted in batch processing", func(t *testing.T) {
		resourceFailure := &FailingRunnable{
			name:     "batch-resource",
			failType: "resource_exhausted",
		}

		chain, err := orch.CreateChain([]core.Runnable{resourceFailure})
		require.NoError(t, err)

		inputs := []any{
			map[string]any{"input": "test1"},
			map[string]any{"input": "test2"},
		}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

		_, err = chain.Batch(context.Background(), inputs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource exhausted")
	})
}

// Test graph failure scenarios
func TestGraphFailureScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("graph with failing node", func(t *testing.T) {
		graph, err := orch.CreateGraph()
		require.NoError(t, err)

		workingNode := &FailingRunnable{name: "working", failType: "success"}
		failingNode := &FailingRunnable{name: "failing", failType: "permanent"}

		err = graph.AddNode("working", workingNode)
		require.NoError(t, err)
		err = graph.AddNode("failing", failingNode)
		require.NoError(t, err)

		err = graph.AddEdge("working", "failing")
		require.NoError(t, err)

		err = graph.SetEntryPoint([]string{"working"})
		require.NoError(t, err)
		err = graph.SetFinishPoint([]string{"failing"})
		require.NoError(t, err)

		_, err = graph.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permanent failure")
	})

	t.Run("graph with transient failures", func(t *testing.T) {
		graph, err := orch.CreateGraph()
		require.NoError(t, err)

		transientNode := &FailingRunnable{
			name:          "transient",
			failType:      "transient",
			failOnAttempt: 1,
		}

		err = graph.AddNode("transient", transientNode)
		require.NoError(t, err)

		err = graph.SetEntryPoint([]string{"transient"})
		require.NoError(t, err)
		err = graph.SetFinishPoint([]string{"transient"})
		require.NoError(t, err)

		// Graph doesn't retry automatically, so transient failures will cause an error
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		// on the first attempt. The node will fail on attempt 1, then succeed on attempt 2,
		// but the graph doesn't retry, so we expect an error.
		result, err := graph.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transient failure attempt 1")
		assert.Nil(t, result) // Expect nil result on error
	})
}

// Test concurrent failure scenarios
func TestConcurrentFailureScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("concurrent chains with failures", func(t *testing.T) {
		const numChains = 10
		var wg sync.WaitGroup
		results := make(chan error, numChains)

		for i := 0; i < numChains; i++ {
			wg.Add(1)
			go func(chainID int) {
				defer wg.Done()

				// Mix of working and failing chains
				var steps []core.Runnable
				if chainID%2 == 0 {
					// Even chains succeed
					steps = []core.Runnable{
						&FailingRunnable{name: fmt.Sprintf("chain-%d-step-1", chainID), failType: "success"},
						&FailingRunnable{name: fmt.Sprintf("chain-%d-step-2", chainID), failType: "success"},
					}
				} else {
					// Odd chains fail
					steps = []core.Runnable{
						&FailingRunnable{name: fmt.Sprintf("chain-%d-step-1", chainID), failType: "success"},
						&FailingRunnable{name: fmt.Sprintf("chain-%d-step-2", chainID), failType: "permanent"},
					}
				}

				chain, err := orch.CreateChain(steps)
				if err != nil {
					results <- err
					return
				}

				_, err = chain.Invoke(context.Background(), map[string]any{"input": "concurrent"})
				results <- err
			}(i)
		}

		wg.Wait()
		close(results)

		// Count successes and failures
		successCount := 0
		failureCount := 0
		for err := range results {
			if err == nil {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				successCount++
			} else {
				failureCount++
			}
		}

		assert.Equal(t, 5, successCount) // Even chains should succeed
		assert.Equal(t, 5, failureCount) // Odd chains should fail
	})
}

// Test batch processing failure scenarios
func TestBatchProcessingFailureScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("batch with partial failures", func(t *testing.T) {
		// Create a step that fails on specific inputs
		conditionalFailure := &ConditionalFailingRunnable{
			name: "conditional",
		}

		chain, err := orch.CreateChain([]core.Runnable{conditionalFailure})
		require.NoError(t, err)

		inputs := []any{
			map[string]any{"id": 1, "should_fail": false},
			map[string]any{"id": 2, "should_fail": true},
			map[string]any{"id": 3, "should_fail": false},
			map[string]any{"id": 4, "should_fail": true},
		}

		results, err := chain.Batch(context.Background(), inputs)

		// Should return partial results even with failures
		assert.Error(t, err)
		assert.Len(t, results, 4)

		// Check individual results
		assert.NotNil(t, results[0]) // Should succeed
		assert.Nil(t, results[1])    // Should fail
		assert.NotNil(t, results[2]) // Should succeed
		assert.Nil(t, results[3])    // Should fail
	})

	t.Run("batch with all failures", func(t *testing.T) {
		permanentFailure := &FailingRunnable{
			name:     "batch-fail",
			failType: "permanent",
		}

		chain, err := orch.CreateChain([]core.Runnable{permanentFailure})
		require.NoError(t, err)

		inputs := []any{
			map[string]any{"input": "test1"},
			map[string]any{"input": "test2"},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		}

		results, err := chain.Batch(context.Background(), inputs)

		assert.Error(t, err)
		assert.Len(t, results, 2)
		// Results should still be populated even with errors
		for _, result := range results {
			assert.Nil(t, result)
		}
	})
}

// Test cascading failure scenarios
func TestCascadingFailureScenarios(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("cascading failures in complex graph", func(t *testing.T) {
		graph, err := orch.CreateGraph()
		require.NoError(t, err)

		// Create a diamond graph with failures
		failAtStart := &FailingRunnable{name: "fail-start", failType: "permanent"}
		workingPath1 := &FailingRunnable{name: "working-1", failType: "success"}
		workingPath2 := &FailingRunnable{name: "working-2", failType: "success"}
		failAtEnd := &FailingRunnable{name: "fail-end", failType: "permanent"}

		nodes := map[string]core.Runnable{
			"start": failAtStart,
			"path1": workingPath1,
			"path2": workingPath2,
			"end":   failAtEnd,
		}

		for name, node := range nodes {
			err = graph.AddNode(name, node)
			require.NoError(t, err)
		}

		// Diamond topology: start -> [path1, path2] -> end
		err = graph.AddEdge("start", "path1")
		require.NoError(t, err)
		err = graph.AddEdge("start", "path2")
		require.NoError(t, err)
		err = graph.AddEdge("path1", "end")
		require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		err = graph.AddEdge("path2", "end")
		require.NoError(t, err)

		err = graph.SetEntryPoint([]string{"start"})
		require.NoError(t, err)
		err = graph.SetFinishPoint([]string{"end"})
		require.NoError(t, err)

		// Should fail at the start node
		_, err = graph.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fail-start")
	})
}

// Test recovery mechanisms
func TestRecoveryMechanisms(t *testing.T) {

	t.Run("manual recovery simulation", func(t *testing.T) {
		failingStep := &FailingRunnable{
			name:          "recoverable-step",
			failType:      "transient",
			failOnAttempt: 1,
		}

		recoveryStep := &RecoveryRunnable{name: "recovery-step"}

		// Test recovery by running steps individually
		_, err := failingStep.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err) // First attempt should fail

		result, err := failingStep.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.NoError(t, err) // Second attempt should succeed

		recoveryResult, err := recoveryStep.Invoke(context.Background(), result)
		assert.NoError(t, err)
		assert.NotNil(t, recoveryResult)
	})

	t.Run("circuit breaker pattern simulation", func(t *testing.T) {
		// Simulate circuit breaker behavior
		failingStep := &FailingRunnable{
			name:          "circuit-step",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			failType:      "transient",
			failOnAttempt: 3, // Fail first 3 attempts
		}

		// Simulate multiple calls
		for i := 1; i <= 5; i++ {
			_, err := failingStep.Invoke(context.Background(), map[string]any{"input": "test"})

			if i <= 3 {
				assert.Error(t, err, "Attempt %d should fail", i)
			} else {
				assert.NoError(t, err, "Attempt %d should succeed", i)
			}
		}
	})
}

// Test graceful degradation
func TestGracefulDegradation(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	t.Run("degraded service continues processing", func(t *testing.T) {
		// Create steps with different reliability levels
		reliableStep := &FailingRunnable{name: "reliable", failType: "success"}
		unreliableStep := &FailingRunnable{
			name:          "unreliable",
			failType:      "transient",
			failOnAttempt: 1,
		}
		essentialStep := &FailingRunnable{name: "essential", failType: "success"}

		chain, err := orch.CreateChain([]core.Runnable{reliableStep, unreliableStep, essentialStep})
		require.NoError(t, err)

		result, err := chain.Invoke(context.Background(), map[string]any{"input": "degraded"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// Test error propagation and context
func TestErrorPropagation(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("error context preservation", func(t *testing.T) {
		customError := errors.New("custom chain error")

		// Create a custom runnable with specific error
		failingStep := &CustomFailingRunnable{
			name:  "custom-error",
			error: customError,
		}

		chain, err := orch.CreateChain([]core.Runnable{failingStep})
		require.NoError(t, err)

		_, err = chain.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err)

		// Check that the custom error is preserved as the underlying error
		assert.Error(t, err)
		assert.True(t, errors.Is(err, customError) || strings.Contains(err.Error(), "custom chain error"))
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	})

	t.Run("nested error context", func(t *testing.T) {
		nestedError := fmt.Errorf("nested: %w", errors.New("original error"))

		// Create a custom runnable with nested error
		failingStep := &CustomFailingRunnable{
			name:  "nested-error",
			error: nestedError,
		}

		chain, err := orch.CreateChain([]core.Runnable{failingStep})
		require.NoError(t, err)

		_, err = chain.Invoke(context.Background(), map[string]any{"input": "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nested")
		assert.Contains(t, err.Error(), "original error")
	})
}

// Test performance under failure conditions
func TestPerformanceUnderFailure(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("failure handling performance", func(t *testing.T) {
		permanentFailure := &FailingRunnable{
			name:     "perf-fail",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			failType: "permanent",
		}

		chain, err := orch.CreateChain([]core.Runnable{permanentFailure})
		require.NoError(t, err)

		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			_, err := chain.Invoke(context.Background(), map[string]any{"input": fmt.Sprintf("perf-test-%d", i)})
			assert.Error(t, err)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		// Should handle failures efficiently
		assert.True(t, avgDuration < 10*time.Millisecond,
			"Failure handling too slow: %v per operation", avgDuration)
	})
}

// Test cleanup after failures
func TestCleanupAfterFailures(t *testing.T) {
	orch, err := NewDefaultOrchestrator()
	require.NoError(t, err)

	t.Run("resource cleanup after chain failure", func(t *testing.T) {
		permanentFailure := &FailingRunnable{
			name:     "cleanup-test",
			failType: "permanent",
		}

		chain, err := orch.CreateChain([]core.Runnable{permanentFailure})
		require.NoError(t, err)

		// Execute and expect failure
		_, err = chain.Invoke(context.Background(), map[string]any{"input": "cleanup"})
		assert.Error(t, err)

		// Metrics should still be consistent
		metrics := orch.GetMetrics()
		assert.Equal(t, 1, metrics.GetActiveChains())

		// Orchestrator should still be functional
		workingChain, err := orch.CreateChain([]core.Runnable{
			&FailingRunnable{name: "working-after-failure", failType: "success"},
		})
		assert.NoError(t, err)

		result, err := workingChain.Invoke(context.Background(), map[string]any{"input": "post-failure"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}
