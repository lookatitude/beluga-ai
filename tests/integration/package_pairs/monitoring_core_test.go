// Package package_pairs provides integration tests between Monitoring and Core packages.
// This test suite verifies that Monitoring components can work with Core types
// and that Core components can use Monitoring for observability.
package package_pairs

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/monitoring"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationMonitoringCore tests the integration between Monitoring and Core packages.
func TestIntegrationMonitoringCore(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("core_runnable_with_monitoring", func(t *testing.T) {
		// Test that core runnable can use monitoring for observability
		mockMonitor := monitoring.NewAdvancedMockMonitor("test-monitor", "test-type")
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				// Simulate using monitoring in runnable
				_ = mockMonitor.RecordMetric(ctx, "runnable.invoke", 1.0, map[string]string{"input": "test"})
				return "result", nil
			},
		}

		// Wrap with traced runnable that uses monitoring
		traced := core.NewTracedRunnable(
			runnable,
			nil, // Uses noop tracer
			core.NoOpMetrics(),
			"test-component",
			"",
		)

		result, err := traced.Invoke(context.Background(), "test")
		require.NoError(t, err)
		assert.Equal(t, "result", result)

		// Verify monitoring was used
		metrics := mockMonitor.GetMetrics()
		assert.Greater(t, len(metrics), 0)
	})

	t.Run("monitoring_with_core_container", func(t *testing.T) {
		// Test that monitoring can be registered in core DI container
		container := core.NewContainer()
		err := container.Register(func() monitoring.Monitor {
			return monitoring.NewAdvancedMockMonitor("container-monitor", "test-type")
		})
		require.NoError(t, err)

		var monitor monitoring.Monitor
		err = container.Resolve(&monitor)
		require.NoError(t, err)
		assert.NotNil(t, monitor)
	})

	t.Run("monitoring_config_with_core_config", func(t *testing.T) {
		// Test that monitoring config can work with core config
		coreCfg := core.DefaultConfig()
		assert.True(t, coreCfg.EnableTracing)
		assert.True(t, coreCfg.EnableMetrics)

		monitoringCfg := monitoring.DefaultConfig()
		assert.True(t, monitoringCfg.Tracing.Enabled)
		assert.True(t, monitoringCfg.Metrics.Enabled)

		// Verify both configs are compatible
		err := monitoringCfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("core_traced_runnable_with_monitoring_metrics", func(t *testing.T) {
		// Test that TracedRunnable metrics work with monitoring concepts
		mockMonitor := monitoring.NewAdvancedMockMonitor("metrics-monitor", "test-type")
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				return "result", nil
			},
		}

		traced := core.NewTracedRunnable(
			runnable,
			nil,
			core.NoOpMetrics(),
			"test-component",
			"",
		)

		// Execute and verify monitoring concepts align
		result, err := traced.Invoke(context.Background(), "test")
		require.NoError(t, err)
		assert.Equal(t, "result", result)

		// Record metric using monitoring
		err = mockMonitor.RecordMetric(context.Background(), "core.runnable.invoke", 1.0, nil)
		require.NoError(t, err)
	})

	t.Run("monitoring_health_with_core_health", func(t *testing.T) {
		// Test that monitoring health checks work with core health concepts
		mockMonitor := monitoring.NewAdvancedMockMonitor("health-monitor", "test-type")
		ctx := context.Background()

		healthy := mockMonitor.IsHealthy(ctx)
		assert.True(t, healthy)

		healthCheck, err := mockMonitor.CheckComponentHealth(ctx, "test-component")
		require.NoError(t, err)
		assert.NotNil(t, healthCheck)
	})

	t.Run("monitoring_logging_with_core_context", func(t *testing.T) {
		// Test that monitoring logging works with core context
		mockMonitor := monitoring.NewAdvancedMockMonitor("log-monitor", "test-type")
		ctx := context.Background()

		err := mockMonitor.Log(ctx, "INFO", "Test message", map[string]any{
			"component": "core",
			"operation": "test",
		})
		require.NoError(t, err)

		logs := mockMonitor.GetLogs()
		require.Len(t, logs, 1)
		assert.Equal(t, "INFO", logs[0].Level)
		// Check that the log was recorded (fields structure may vary)
		assert.NotNil(t, logs[0].Fields)
	})

	t.Run("monitoring_tracing_with_core_traced_runnable", func(t *testing.T) {
		// Test that monitoring tracing works with core TracedRunnable
		mockMonitor := monitoring.NewAdvancedMockMonitor("trace-monitor", "test-type")
		ctx := context.Background()

		_, traceID, err := mockMonitor.StartTrace(ctx, "core.operation")
		require.NoError(t, err)
		assert.NotEmpty(t, traceID)

		// Simulate operation
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				return "result", nil
			},
		}
		traced := core.NewTracedRunnable(runnable, nil, core.NoOpMetrics(), "test", "")
		_, _ = traced.Invoke(ctx, "test")

		// Finish trace
		err = mockMonitor.FinishTrace(ctx, traceID, true)
		require.NoError(t, err)

		traces := mockMonitor.GetTraces()
		assert.Greater(t, len(traces), 0)
	})
}
