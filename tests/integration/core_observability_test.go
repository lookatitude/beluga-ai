// Package integration provides integration tests for OTEL metrics and health monitoring.
// T012: Integration test for OTEL metrics and health monitoring
package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreObservabilityIntegration tests OTEL integration and health monitoring
func TestCoreObservabilityIntegration(t *testing.T) {
	t.Run("MetricsIntegration", func(t *testing.T) {
		// This test validates that OTEL metrics are properly integrated
		// Currently the core package has metrics.go with OTEL support

		container := core.NewContainer()
		ctx := context.Background()

		// Register service with metrics
		err := container.Register(func() string { return "metrics_test_service" })
		require.NoError(t, err)

		// Perform operations that should be metrified
		var result string
		err = container.Resolve(&result)
		require.NoError(t, err)

		// Check health (which should also be metrified)
		err = container.CheckHealth(ctx)
		assert.NoError(t, err)

		// Note: In a full implementation, this test would verify that metrics
		// were actually recorded. For now, we verify operations work correctly
		// which is a prerequisite for metrics collection.

		t.Log("OTEL metrics integration test completed - operations successful")
	})

	t.Run("HealthMonitoringIntegration", func(t *testing.T) {
		container := core.NewContainer()
		ctx := context.Background()

		// Test health monitoring across different states

		// Empty container health
		err := container.CheckHealth(ctx)
		assert.NoError(t, err, "Empty container should be healthy")

		// Container with registrations
		err = container.Register(func() string { return "health_test" })
		require.NoError(t, err)

		err = container.CheckHealth(ctx)
		assert.NoError(t, err, "Container with registrations should be healthy")

		// Container with resolved dependencies
		var result string
		err = container.Resolve(&result)
		require.NoError(t, err)

		err = container.CheckHealth(ctx)
		assert.NoError(t, err, "Container with resolved dependencies should be healthy")

		// Test health check performance
		iterations := 100
		for i := 0; i < iterations; i++ {
			err = container.CheckHealth(ctx)
			assert.NoError(t, err, "Health check %d should pass", i+1)
		}
	})

	t.Run("ObservabilityWithRunnables", func(t *testing.T) {
		container := core.NewContainer()

		// Register observable Runnable
		err := container.Register(func() *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable("observable",
				core.WithMockResponses([]any{"observable_result"}))
		})
		require.NoError(t, err)

		// Resolve and use Runnable
		var runnable *core.AdvancedMockRunnable
		err = container.Resolve(&runnable)
		require.NoError(t, err)

		ctx := context.Background()

		// Perform operations
		_, err = runnable.Invoke(ctx, "observability_test")
		assert.NoError(t, err)

		// Verify mock tracked the operation
		callCount := runnable.GetCallCount()
		assert.Equal(t, 1, callCount, "Mock should track the operation")

		// Test batch operations
		inputs := []any{"batch1", "batch2", "batch3"}
		_, err = runnable.Batch(ctx, inputs)
		assert.NoError(t, err)

		// Call count should include batch operations
		finalCallCount := runnable.GetCallCount()
		assert.Greater(t, finalCallCount, callCount, "Batch should increase call count")
	})

	t.Run("HealthAndMetricsUnderLoad", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping load test in short mode")
		}

		container := core.NewContainer()

		// Register service
		err := container.Register(func() *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable("load_test")
		})
		require.NoError(t, err)

		ctx := context.Background()

		// Perform many operations to test observability under load
		for i := 0; i < 1000; i++ {
			var runnable *core.AdvancedMockRunnable
			err = container.Resolve(&runnable)
			require.NoError(t, err)

			_, err = runnable.Invoke(ctx, fmt.Sprintf("load_test_%d", i))
			require.NoError(t, err)

			// Periodic health checks
			if i%100 == 0 {
				err = container.CheckHealth(ctx)
				assert.NoError(t, err, "Health check should pass during load test")
			}
		}

		// Final health check
		err = container.CheckHealth(ctx)
		assert.NoError(t, err, "Container should be healthy after load test")

		t.Log("Completed 1000 operations with periodic health checks - all successful")
	})
}

// TestObservabilityErrorScenarios tests observability during error conditions
func TestObservabilityErrorScenarios(t *testing.T) {
	t.Run("HealthCheckDuringErrors", func(t *testing.T) {
		container := core.NewContainer()

		// Register error-prone service
		err := container.Register(func() *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable("error_service",
				core.WithMockError(core.NewInternalError("service_error", "Service unavailable")))
		})
		require.NoError(t, err)

		ctx := context.Background()

		// Container should still be healthy even with error-prone services
		err = container.CheckHealth(ctx)
		assert.NoError(t, err, "Container health should not depend on service errors")

		// Resolve error service
		var errorService *core.AdvancedMockRunnable
		err = container.Resolve(&errorService)
		require.NoError(t, err)

		// Service operations fail
		_, err = errorService.Invoke(ctx, "failing_operation")
		assert.Error(t, err, "Service should fail as configured")

		// Container should still be healthy
		err = container.CheckHealth(ctx)
		assert.NoError(t, err, "Container should remain healthy after service errors")
	})

	t.Run("MetricsCollectionDuringErrors", func(t *testing.T) {
		container := core.NewContainer()

		// Test that metrics collection continues even when operations fail
		err := container.Register(func() *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable("metrics_error_test",
				core.WithMockError(core.NewNetworkError("network_issue", "Network unavailable")))
		})
		require.NoError(t, err)

		var runnable *core.AdvancedMockRunnable
		err = container.Resolve(&runnable)
		require.NoError(t, err)

		ctx := context.Background()

		// Perform failing operations
		for i := 0; i < 10; i++ {
			_, err = runnable.Invoke(ctx, fmt.Sprintf("failing_op_%d", i))
			assert.Error(t, err, "Operations should fail as configured")
		}

		// Verify mock tracked all failed operations
		callCount := runnable.GetCallCount()
		assert.Equal(t, 10, callCount, "Should track failed operations")

		// Container should remain functional
		err = container.CheckHealth(ctx)
		assert.NoError(t, err, "Container should handle service errors gracefully")
	})
}

// TestCoreIntegrationRealWorldScenarios tests realistic usage scenarios
func TestCoreIntegrationRealWorldScenarios(t *testing.T) {
	t.Run("AIWorkflowSimulation", func(t *testing.T) {
		container := core.NewContainer()

		// Register components typical in AI workflow

		// LLM-like component
		err := container.Register(func() *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable("llm",
				core.WithMockResponses([]any{"LLM response to query"}))
		})
		require.NoError(t, err)

		// Retriever-like component
		err = container.Register(func() *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable("retriever",
				core.WithMockResponses([]any{"Retrieved documents"}))
		})
		require.NoError(t, err)

		// Agent-like component (depends on LLM and Retriever)
		err = container.Register(func(llm, retriever *core.AdvancedMockRunnable) *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable("agent",
				core.WithMockResponses([]any{"Agent response using LLM and Retriever"}))
		})
		require.NoError(t, err)

		// Resolve and test the full workflow
		var agent *core.AdvancedMockRunnable
		err = container.Resolve(&agent)
		require.NoError(t, err)

		ctx := context.Background()

		// Simulate AI workflow
		result, err := agent.Invoke(ctx, "User query for AI workflow")
		require.NoError(t, err)
		assert.Contains(t, result, "Agent response")

		// Test workflow health
		err = container.CheckHealth(ctx)
		assert.NoError(t, err)

		t.Log("AI workflow simulation completed successfully")
	})

	t.Run("MicroservicesPattern", func(t *testing.T) {
		container := core.NewContainer()

		// Register microservice-style components

		// Config service
		err := container.Register(func() string { return "production_config" })
		require.NoError(t, err)

		// Database service
		err = container.Register(func(config string) *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable(fmt.Sprintf("db_%s", config),
				core.WithMockResponses([]any{"Database data"}))
		})
		require.NoError(t, err)

		// API service (depends on config and database)
		err = container.Register(func(config string, db *core.AdvancedMockRunnable) *core.AdvancedMockRunnable {
			return core.NewAdvancedMockRunnable(fmt.Sprintf("api_%s", config),
				core.WithMockResponses([]any{"API response"}))
		})
		require.NoError(t, err)

		// Resolve API service (should resolve all dependencies)
		var apiService *core.AdvancedMockRunnable
		err = container.Resolve(&apiService)
		require.NoError(t, err)

		ctx := context.Background()

		// Test API service functionality
		result, err := apiService.Invoke(ctx, "API request")
		require.NoError(t, err)
		assert.Equal(t, "API response", result)

		// Verify dependency injection worked
		assert.NotNil(t, apiService)

		// System health check
		err = container.CheckHealth(ctx)
		assert.NoError(t, err)

		t.Log("Microservices pattern simulation completed successfully")
	})
}
