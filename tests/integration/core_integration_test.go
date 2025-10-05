// Package integration provides integration tests for Container-Runnable interaction.
// T011: Integration test for Container-Runnable interaction
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/core/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainerRunnableIntegration tests the integration between DI Container and Runnable components
func TestContainerRunnableIntegration(t *testing.T) {
	t.Run("RegisterAndResolveRunnable", func(t *testing.T) {
		container := core.NewContainer()

		// Register a Runnable factory
		err := container.Register(func() iface.Runnable {
			return core.NewAdvancedMockRunnable("integrated_runnable",
				core.WithMockResponses([]any{"integration_result"}))
		})
		require.NoError(t, err)

		// Resolve the Runnable
		var runnable iface.Runnable
		err = container.Resolve(&runnable)
		require.NoError(t, err)
		assert.NotNil(t, runnable)

		// Test the resolved Runnable
		ctx := context.Background()
		result, err := runnable.Invoke(ctx, "integration_test")
		require.NoError(t, err)
		assert.Equal(t, "integration_result", result)
	})

	t.Run("RunnableWithDependencies", func(t *testing.T) {
		container := core.NewContainer()

		// Register dependencies
		err := container.Register(func() string { return "config_service" })
		require.NoError(t, err)

		err = container.Register(func() int { return 42 })
		require.NoError(t, err)

		// Register Runnable that depends on other services
		err = container.Register(func(config string, number int) iface.Runnable {
			return core.NewAdvancedMockRunnable(fmt.Sprintf("dependent_%s_%d", config, number),
				core.WithMockResponses([]any{fmt.Sprintf("result_%s_%d", config, number)}))
		})
		require.NoError(t, err)

		// Resolve the dependent Runnable
		var runnable iface.Runnable
		err = container.Resolve(&runnable)
		require.NoError(t, err)

		// Test that dependencies were properly injected
		ctx := context.Background()
		result, err := runnable.Invoke(ctx, "dependency_test")
		require.NoError(t, err)
		assert.Contains(t, result, "config_service")
		assert.Contains(t, result, "42")
	})

	t.Run("MultipleRunnableTypes", func(t *testing.T) {
		container := core.NewContainer()

		// Register different types of Runnables
		err := container.Register(func() iface.Runnable {
			return core.NewAdvancedMockRunnable("type1",
				core.WithMockResponses([]any{"type1_result"}))
		})
		require.NoError(t, err)

		// Register with different factory signature
		container.Singleton(core.NewAdvancedMockRunnable("singleton_runnable",
			core.WithMockResponses([]any{"singleton_result"})))

		// Resolve and test multiple Runnable instances
		var runnable1 iface.Runnable
		err = container.Resolve(&runnable1)
		require.NoError(t, err)

		var runnable2 iface.Runnable
		err = container.Resolve(&runnable2)
		require.NoError(t, err)

		ctx := context.Background()

		// Test both Runnables work
		result1, err := runnable1.Invoke(ctx, "test1")
		require.NoError(t, err)

		result2, err := runnable2.Invoke(ctx, "test2")
		require.NoError(t, err)

		// Results should be different (factory vs singleton)
		assert.NotEqual(t, result1, result2)
	})

	t.Run("ContainerHealthWithRunnables", func(t *testing.T) {
		container := core.NewContainer()
		ctx := context.Background()

		// Container should be healthy when empty
		err := container.CheckHealth(ctx)
		assert.NoError(t, err)

		// Register Runnable
		err = container.Register(func() iface.Runnable {
			return core.NewAdvancedMockRunnable("health_test")
		})
		require.NoError(t, err)

		// Container should still be healthy with registrations
		err = container.CheckHealth(ctx)
		assert.NoError(t, err)

		// Resolve and test Runnable health
		var runnable iface.Runnable
		err = container.Resolve(&runnable)
		require.NoError(t, err)

		// Test Runnable functionality as health indicator
		result, err := runnable.Invoke(ctx, "health_check")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestCoreIntegrationComplexScenarios tests complex integration scenarios
func TestCoreIntegrationComplexScenarios(t *testing.T) {
	t.Run("ChainedRunnableExecution", func(t *testing.T) {
		container := core.NewContainer()

		// Register chain of Runnables
		err := container.Register(func() iface.Runnable {
			return core.NewAdvancedMockRunnable("step1",
				core.WithMockResponses([]any{"step1_output"}))
		})
		require.NoError(t, err)

		// Simulate a chain where output of one becomes input of next
		var runnable iface.Runnable
		err = container.Resolve(&runnable)
		require.NoError(t, err)

		ctx := context.Background()

		// First step
		result1, err := runnable.Invoke(ctx, "initial_input")
		require.NoError(t, err)

		// Use result as input for second operation
		result2, err := runnable.Invoke(ctx, result1)
		require.NoError(t, err)
		assert.NotNil(t, result2)
	})

	t.Run("BatchProcessingIntegration", func(t *testing.T) {
		container := core.NewContainer()

		// Register batch-capable Runnable
		err := container.Register(func() iface.Runnable {
			return core.NewAdvancedMockRunnable("batch_processor",
				core.WithMockResponses([]any{"processed_item"}))
		})
		require.NoError(t, err)

		var runnable iface.Runnable
		err = container.Resolve(&runnable)
		require.NoError(t, err)

		// Test batch processing
		ctx := context.Background()
		inputs := []any{"item1", "item2", "item3", "item4", "item5"}

		results, err := runnable.Batch(ctx, inputs)
		require.NoError(t, err)
		assert.Len(t, results, len(inputs))

		// Verify all items were processed
		for i, result := range results {
			assert.Equal(t, "processed_item", result, "Item %d should be processed", i)
		}
	})

	t.Run("StreamingWithContainer", func(t *testing.T) {
		container := core.NewContainer()

		// Register streaming Runnable
		err := container.Register(func() iface.Runnable {
			return core.NewAdvancedMockRunnable("streamer",
				core.WithMockResponses([]any{"stream_data"}))
		})
		require.NoError(t, err)

		var runnable iface.Runnable
		err = container.Resolve(&runnable)
		require.NoError(t, err)

		// Test streaming
		ctx := context.Background()
		ch, err := runnable.Stream(ctx, "streaming_input")
		require.NoError(t, err)
		assert.NotNil(t, ch)

		// Collect streaming results
		var results []any
		timeout := time.After(time.Second)

		for {
			select {
			case result, ok := <-ch:
				if !ok {
					// Channel closed, streaming complete
					goto streamDone
				}
				results = append(results, result)
			case <-timeout:
				t.Error("Streaming should complete within reasonable time")
				goto streamDone
			}
		}

	streamDone:
		assert.NotEmpty(t, results, "Should receive streaming results")
	})
}

// TestCoreIntegrationErrorRecovery tests error recovery scenarios
func TestCoreIntegrationErrorRecovery(t *testing.T) {
	t.Run("ContainerRecoveryAfterError", func(t *testing.T) {
		container := core.NewContainer()

		// Register factory that might fail
		err := container.Register(func() iface.Runnable {
			return core.NewAdvancedMockRunnable("error_prone",
				core.WithMockError(core.NewInternalError("test_error", "Simulated error")))
		})
		require.NoError(t, err)

		// Resolve error-prone Runnable
		var errorRunnable iface.Runnable
		err = container.Resolve(&errorRunnable)
		require.NoError(t, err)

		ctx := context.Background()

		// Test that it fails as expected
		_, err = errorRunnable.Invoke(ctx, "error_test")
		assert.Error(t, err, "Should fail as configured")

		// Container should still be healthy and functional
		err = container.CheckHealth(ctx)
		assert.NoError(t, err, "Container should remain healthy after Runnable error")

		// Should be able to register and resolve new services
		err = container.Register(func() string { return "recovery_test" })
		require.NoError(t, err)

		var result string
		err = container.Resolve(&result)
		assert.NoError(t, err)
		assert.Equal(t, "recovery_test", result)
	})
}

// TestCoreIntegrationPerformance tests integration performance requirements
func TestCoreIntegrationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration performance tests in short mode")
	}

	t.Run("FullIntegrationLatency", func(t *testing.T) {
		container := core.NewContainer()

		// Register Runnable
		err := container.Register(func() iface.Runnable {
			return core.NewAdvancedMockRunnable("perf_integration")
		})
		require.NoError(t, err)

		// Measure full resolution + invoke latency
		iterations := 1000
		ctx := context.Background()

		start := time.Now()
		for i := 0; i < iterations; i++ {
			var runnable iface.Runnable
			err := container.Resolve(&runnable)
			require.NoError(t, err)

			_, err = runnable.Invoke(ctx, "perf_test")
			require.NoError(t, err)
		}
		elapsed := time.Since(start)

		avgTime := elapsed / time.Duration(iterations)
		t.Logf("Average full integration time: %v", avgTime)

		// Should meet combined performance target
		assert.Less(t, avgTime, 2*time.Millisecond,
			"Full integration (resolve + invoke) should be under 2ms")
	})
}
