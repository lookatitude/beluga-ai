// Package monitoring provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
//
// REFERENCE IMPLEMENTATION: This file serves as the reference for testing patterns
// that should be followed by all other packages in the Beluga AI framework.
package monitoring

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SECTION 1: Table-Driven Tests (Reference Pattern)
// =============================================================================

// TestMonitorCreationAdvanced provides advanced table-driven tests for monitor creation.
// REFERENCE: This pattern should be used for all creation/factory tests.
func TestMonitorCreationAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func(t *testing.T) *AdvancedMockMonitor
		validate    func(t *testing.T, monitor *AdvancedMockMonitor)
		wantErr     bool
	}{
		{
			name:        "basic_monitor_creation",
			description: "Create basic monitor with minimal config",
			setup: func(t *testing.T) *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("test-monitor", "test-type")
			},
			validate: func(t *testing.T, monitor *AdvancedMockMonitor) {
				assert.NotNil(t, monitor)
				assert.Equal(t, "test-monitor", monitor.GetName())
				assert.Equal(t, "test-type", monitor.GetMonitorType())
				assert.Equal(t, 0, monitor.GetCallCount())
			},
		},
		{
			name:        "monitor_with_error_config",
			description: "Create monitor configured to return errors",
			setup: func(t *testing.T) *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("error-monitor", "test-type",
					WithMockError(true, errors.New("simulated error")))
			},
			validate: func(t *testing.T, monitor *AdvancedMockMonitor) {
				assert.NotNil(t, monitor)
				ctx := context.Background()
				err := monitor.RecordMetric(ctx, "test", 1.0, nil)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "simulated error")
			},
		},
		{
			name:        "monitor_with_delay_config",
			description: "Create monitor with simulated delay",
			setup: func(t *testing.T) *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("delayed-monitor", "test-type",
					WithMockDelay(10*time.Millisecond))
			},
			validate: func(t *testing.T, monitor *AdvancedMockMonitor) {
				assert.NotNil(t, monitor)
				ctx := context.Background()
				start := time.Now()
				err := monitor.RecordMetric(ctx, "test", 1.0, nil)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, time.Since(start), 10*time.Millisecond)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			monitor := tt.setup(t)
			tt.validate(t, monitor)
		})
	}
}

// TestMonitorOperationsAdvanced provides advanced table-driven tests for monitor operations.
// REFERENCE: This pattern should be used for all operation/method tests.
func TestMonitorOperationsAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func(t *testing.T) *AdvancedMockMonitor
		operation   func(t *testing.T, ctx context.Context, monitor *AdvancedMockMonitor) error
		validate    func(t *testing.T, monitor *AdvancedMockMonitor, err error)
		wantErr     bool
	}{
		{
			name:        "health_check_operation",
			description: "Test monitor health check",
			setup: func(t *testing.T) *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("test-monitor", "test-type")
			},
			operation: func(t *testing.T, ctx context.Context, monitor *AdvancedMockMonitor) error {
				healthy := monitor.IsHealthy(ctx)
				if !healthy {
					return errors.New("monitor unhealthy")
				}
				return nil
			},
			validate: func(t *testing.T, monitor *AdvancedMockMonitor, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:        "record_metric_operation",
			description: "Test metric recording",
			setup: func(t *testing.T) *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("metric-monitor", "test-type")
			},
			operation: func(t *testing.T, ctx context.Context, monitor *AdvancedMockMonitor) error {
				return monitor.RecordMetric(ctx, "test_metric", 42.5, map[string]string{
					"component": "test",
					"operation": "test_op",
				})
			},
			validate: func(t *testing.T, monitor *AdvancedMockMonitor, err error) {
				assert.NoError(t, err)
				metrics := monitor.GetMetrics()
				assert.NotEmpty(t, metrics)
				assert.Equal(t, 1, monitor.GetCallCount())
			},
		},
		{
			name:        "trace_lifecycle_operation",
			description: "Test trace start and finish",
			setup: func(t *testing.T) *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("trace-monitor", "test-type")
			},
			operation: func(t *testing.T, ctx context.Context, monitor *AdvancedMockMonitor) error {
				traceCtx, traceID, err := monitor.StartTrace(ctx, "test_operation")
				if err != nil {
					return err
				}
				time.Sleep(5 * time.Millisecond) // Simulate work
				return monitor.FinishTrace(traceCtx, traceID, true)
			},
			validate: func(t *testing.T, monitor *AdvancedMockMonitor, err error) {
				assert.NoError(t, err)
				traces := monitor.GetTraces()
				require.Len(t, traces, 1)
				assert.Equal(t, "test_operation", traces[0].Operation)
				assert.True(t, traces[0].Success)
				assert.Greater(t, traces[0].Duration, time.Duration(0))
			},
		},
		{
			name:        "logging_operation",
			description: "Test structured logging",
			setup: func(t *testing.T) *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("log-monitor", "test-type")
			},
			operation: func(t *testing.T, ctx context.Context, monitor *AdvancedMockMonitor) error {
				return monitor.Log(ctx, "INFO", "Test log message", map[string]any{
					"request_id": "test-123",
					"user_id":    "user-456",
				})
			},
			validate: func(t *testing.T, monitor *AdvancedMockMonitor, err error) {
				assert.NoError(t, err)
				logs := monitor.GetLogs()
				require.Len(t, logs, 1)
				assert.Equal(t, "INFO", logs[0].Level)
				assert.Equal(t, "Test log message", logs[0].Message)
				assert.Equal(t, "test-123", logs[0].Fields["request_id"])
			},
		},
		{
			name:        "component_health_check_operation",
			description: "Test component health check",
			setup: func(t *testing.T) *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("health-monitor", "test-type")
			},
			operation: func(t *testing.T, ctx context.Context, monitor *AdvancedMockMonitor) error {
				result, err := monitor.CheckComponentHealth(ctx, "test-component")
				if err != nil {
					return err
				}
				if result["status"] != "healthy" {
					return errors.New("component unhealthy")
				}
				return nil
			},
			validate: func(t *testing.T, monitor *AdvancedMockMonitor, err error) {
				assert.NoError(t, err)
				healthChecks := monitor.GetHealthChecks()
				require.Len(t, healthChecks, 1)
				assert.Equal(t, "test-component", healthChecks[0].Component)
				assert.Equal(t, "healthy", healthChecks[0].Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			monitor := tt.setup(t)
			ctx := context.Background()
			err := tt.operation(t, ctx, monitor)
			tt.validate(t, monitor, err)
		})
	}
}

// =============================================================================
// SECTION 2: Concurrency Tests (Reference Pattern)
// =============================================================================

// TestConcurrentMonitorOperations tests concurrent monitor operations.
// REFERENCE: This pattern should be used for all concurrency-safe tests.
func TestConcurrentMonitorOperations(t *testing.T) {
	const numGoroutines = 20
	const numOperationsPerGoroutine = 10

	monitor := NewAdvancedMockMonitor("concurrent-monitor", "test-type")

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines*numOperationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				healthy := monitor.IsHealthy(ctx)
				if !healthy {
					errChan <- fmt.Errorf("goroutine %d: monitor unhealthy", id)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

// TestConcurrentMetricRecording tests concurrent metric recording.
func TestConcurrentMetricRecording(t *testing.T) {
	const numGoroutines = 50
	const numMetricsPerGoroutine = 20

	monitor := NewAdvancedMockMonitor("concurrent-metric-monitor", "test-type")

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines*numMetricsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numMetricsPerGoroutine; j++ {
				err := monitor.RecordMetric(ctx, fmt.Sprintf("metric_%d_%d", id, j), float64(j), map[string]string{
					"goroutine": fmt.Sprintf("%d", id),
				})
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors
	for err := range errChan {
		t.Errorf("Concurrent metric recording error: %v", err)
	}

	// Verify call count
	assert.Equal(t, numGoroutines*numMetricsPerGoroutine, monitor.GetCallCount())
}

// TestConcurrentTracing tests concurrent tracing operations.
func TestConcurrentTracing(t *testing.T) {
	const numGoroutines = 30
	const numTracesPerGoroutine = 5

	monitor := NewAdvancedMockMonitor("concurrent-trace-monitor", "test-type")

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines*numTracesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numTracesPerGoroutine; j++ {
				_, traceID, err := monitor.StartTrace(ctx, fmt.Sprintf("operation_%d_%d", id, j))
				if err != nil {
					errChan <- err
					continue
				}
				time.Sleep(time.Microsecond) // Minimal simulated work
				err = monitor.FinishTrace(ctx, traceID, true)
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors
	for err := range errChan {
		t.Errorf("Concurrent tracing error: %v", err)
	}

	// Verify traces were recorded
	traces := monitor.GetTraces()
	assert.Len(t, traces, numGoroutines*numTracesPerGoroutine)
}

// TestConcurrentLogging tests concurrent logging operations.
func TestConcurrentLogging(t *testing.T) {
	const numGoroutines = 40
	const numLogsPerGoroutine = 10

	monitor := NewAdvancedMockMonitor("concurrent-log-monitor", "test-type")

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines*numLogsPerGoroutine)
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numLogsPerGoroutine; j++ {
				level := levels[j%len(levels)]
				err := monitor.Log(ctx, level, fmt.Sprintf("Log from goroutine %d, message %d", id, j), map[string]any{
					"goroutine_id": id,
					"message_id":   j,
				})
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors
	for err := range errChan {
		t.Errorf("Concurrent logging error: %v", err)
	}

	// Verify logs were recorded
	logs := monitor.GetLogs()
	assert.Len(t, logs, numGoroutines*numLogsPerGoroutine)
}

// =============================================================================
// SECTION 3: Context Tests (Reference Pattern)
// =============================================================================

// TestMonitorWithContext tests monitor operations with context.
// REFERENCE: This pattern should be used for all context-aware tests.
func TestMonitorWithContext(t *testing.T) {
	t.Run("operations_with_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		monitor := NewAdvancedMockMonitor("context-monitor", "test-type")

		healthy := monitor.IsHealthy(ctx)
		assert.True(t, healthy)

		err := monitor.RecordMetric(ctx, "test_metric", 1.0, nil)
		assert.NoError(t, err)
	})

	t.Run("operations_with_cancelled_context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		monitor := NewAdvancedMockMonitor("cancel-monitor", "test-type")

		// Record before cancellation
		err := monitor.RecordMetric(ctx, "before_cancel", 1.0, nil)
		assert.NoError(t, err)

		// Cancel context
		cancel()

		// Operations after cancellation (should still work for mock)
		err = monitor.RecordMetric(ctx, "after_cancel", 2.0, nil)
		assert.NoError(t, err)
	})

	t.Run("operations_with_value_context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "request_id", "test-123")

		monitor := NewAdvancedMockMonitor("value-monitor", "test-type")

		err := monitor.Log(ctx, "INFO", "Test message", map[string]any{
			"request_id": ctx.Value("request_id"),
		})
		assert.NoError(t, err)

		logs := monitor.GetLogs()
		require.Len(t, logs, 1)
		assert.Equal(t, "test-123", logs[0].Fields["request_id"])
	})
}

// =============================================================================
// SECTION 4: Error Handling Tests (Reference Pattern)
// =============================================================================

// TestMonitorErrorHandling tests error handling scenarios.
// REFERENCE: This pattern should be used for all error handling tests.
func TestMonitorErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setup       func() *AdvancedMockMonitor
		operation   func(ctx context.Context, monitor *AdvancedMockMonitor) error
		expectedErr string
	}{
		{
			name:        "metric_recording_error",
			description: "Handle metric recording error",
			setup: func() *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("error-monitor", "test-type",
					WithMockError(true, errors.New("metric recording failed")))
			},
			operation: func(ctx context.Context, monitor *AdvancedMockMonitor) error {
				return monitor.RecordMetric(ctx, "test", 1.0, nil)
			},
			expectedErr: "metric recording failed",
		},
		{
			name:        "trace_start_error",
			description: "Handle trace start error",
			setup: func() *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("trace-error-monitor", "test-type",
					WithMockError(true, errors.New("trace start failed")))
			},
			operation: func(ctx context.Context, monitor *AdvancedMockMonitor) error {
				_, _, err := monitor.StartTrace(ctx, "test_op")
				return err
			},
			expectedErr: "trace start failed",
		},
		{
			name:        "logging_error",
			description: "Handle logging error",
			setup: func() *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("log-error-monitor", "test-type",
					WithMockError(true, errors.New("logging failed")))
			},
			operation: func(ctx context.Context, monitor *AdvancedMockMonitor) error {
				return monitor.Log(ctx, "ERROR", "Test message", nil)
			},
			expectedErr: "logging failed",
		},
		{
			name:        "health_check_error",
			description: "Handle health check error",
			setup: func() *AdvancedMockMonitor {
				return NewAdvancedMockMonitor("health-error-monitor", "test-type",
					WithMockError(true, errors.New("health check failed")))
			},
			operation: func(ctx context.Context, monitor *AdvancedMockMonitor) error {
				_, err := monitor.CheckComponentHealth(ctx, "test-component")
				return err
			},
			expectedErr: "health check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			monitor := tt.setup()
			ctx := context.Background()

			err := tt.operation(ctx, monitor)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// =============================================================================
// SECTION 5: Integration Tests (Reference Pattern)
// =============================================================================

// TestMonitorIntegration tests monitor integration scenarios.
// REFERENCE: This pattern should be used for integration scenario tests.
func TestMonitorIntegration(t *testing.T) {
	t.Run("full_observability_scenario", func(t *testing.T) {
		monitor := NewAdvancedMockMonitor("integration-monitor", "test-type")
		runner := NewMonitoringScenarioRunner(monitor)

		operations := []string{
			"user_authentication",
			"data_retrieval",
			"llm_inference",
			"response_formatting",
		}

		err := runner.RunFullObservabilityScenario(context.Background(), operations)
		assert.NoError(t, err)

		// Verify all operations recorded
		traces := monitor.GetTraces()
		assert.Len(t, traces, len(operations))

		logs := monitor.GetLogs()
		assert.GreaterOrEqual(t, len(logs), len(operations)*2) // Start + end for each

		healthChecks := monitor.GetHealthChecks()
		assert.Len(t, healthChecks, len(operations))
	})

	t.Run("load_test_scenario", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping load test in short mode")
		}

		monitor := NewAdvancedMockMonitor("load-test-monitor", "test-type")

		// Run load test with 100 operations and 10 concurrent workers
		RunLoadTest(t, monitor, 100, 10)
	})
}

// TestMultiMonitorIntegration tests integration with multiple monitors.
func TestMultiMonitorIntegration(t *testing.T) {
	helper := NewIntegrationTestHelper()

	// Add multiple monitors
	helper.AddMonitor("metrics", NewAdvancedMockMonitor("metrics-monitor", "metrics"))
	helper.AddMonitor("tracing", NewAdvancedMockMonitor("tracing-monitor", "tracing"))
	helper.AddMonitor("logging", NewAdvancedMockMonitor("logging-monitor", "logging"))

	ctx := context.Background()

	// Simulate a request through all monitors
	metricsMonitor := helper.GetMonitor("metrics")
	tracingMonitor := helper.GetMonitor("tracing")
	loggingMonitor := helper.GetMonitor("logging")

	// Start trace
	_, traceID, err := tracingMonitor.StartTrace(ctx, "multi_monitor_request")
	require.NoError(t, err)

	// Log request start
	err = loggingMonitor.Log(ctx, "INFO", "Request started", map[string]any{"trace_id": traceID})
	require.NoError(t, err)

	// Record metrics
	err = metricsMonitor.RecordMetric(ctx, "request_count", 1, map[string]string{"endpoint": "/api/test"})
	require.NoError(t, err)

	// Finish trace
	err = tracingMonitor.FinishTrace(ctx, traceID, true)
	require.NoError(t, err)

	// Log request completion
	err = loggingMonitor.Log(ctx, "INFO", "Request completed", map[string]any{"trace_id": traceID})
	require.NoError(t, err)

	// Verify all monitors received data
	assert.NotEmpty(t, metricsMonitor.GetMetrics())
	assert.NotEmpty(t, tracingMonitor.GetTraces())
	assert.NotEmpty(t, loggingMonitor.GetLogs())

	// Reset and verify
	helper.Reset()
	assert.Empty(t, metricsMonitor.GetMetrics())
	assert.Empty(t, tracingMonitor.GetTraces())
	assert.Empty(t, loggingMonitor.GetLogs())
}

// =============================================================================
// SECTION 6: Assertion Helper Tests (Reference Pattern)
// =============================================================================

// TestAssertionHelpers tests custom assertion helpers.
func TestAssertionHelpers(t *testing.T) {
	t.Run("assert_monitoring_data", func(t *testing.T) {
		monitor := NewAdvancedMockMonitor("assertion-monitor", "test-type")
		ctx := context.Background()

		// Add some data
		monitor.RecordMetric(ctx, "metric1", 1.0, nil)
		monitor.RecordMetric(ctx, "metric2", 2.0, nil)
		monitor.StartTrace(ctx, "trace1")
		monitor.Log(ctx, "INFO", "log1", nil)
		monitor.Log(ctx, "INFO", "log2", nil)
		monitor.Log(ctx, "INFO", "log3", nil)

		// Use assertion helper
		AssertMonitoringData(t, monitor, 2, 1, 3)
	})

	t.Run("assert_trace_record", func(t *testing.T) {
		monitor := NewAdvancedMockMonitor("trace-assertion-monitor", "test-type")
		ctx := context.Background()

		_, traceID, _ := monitor.StartTrace(ctx, "test_operation")
		time.Sleep(time.Millisecond)
		monitor.FinishTrace(ctx, traceID, true)

		traces := monitor.GetTraces()
		require.Len(t, traces, 1)

		AssertTraceRecord(t, traces[0], "test_operation")
	})

	t.Run("assert_log_record", func(t *testing.T) {
		monitor := NewAdvancedMockMonitor("log-assertion-monitor", "test-type")
		ctx := context.Background()

		monitor.Log(ctx, "ERROR", "Test error message", map[string]any{"code": 500})

		logs := monitor.GetLogs()
		require.Len(t, logs, 1)

		AssertLogRecord(t, logs[0], "ERROR")
	})

	t.Run("assert_monitor_health", func(t *testing.T) {
		monitor := NewAdvancedMockMonitor("health-assertion-monitor", "test-type")

		health := monitor.CheckHealth()
		AssertMonitorHealth(t, health, "healthy")
	})
}

// =============================================================================
// SECTION 7: Benchmarks (Reference Pattern)
// =============================================================================

// BenchmarkMonitorCreation benchmarks monitor creation performance.
// REFERENCE: This pattern should be used for creation benchmarks.
func BenchmarkMonitorCreation(b *testing.B) {
	b.Run("basic_creation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewAdvancedMockMonitor("benchmark-monitor", "benchmark-type")
		}
	})

	b.Run("creation_with_options", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewAdvancedMockMonitor("benchmark-monitor", "benchmark-type",
				WithMockDelay(time.Millisecond))
		}
	})
}

// BenchmarkMonitorOperations benchmarks monitor operations performance.
// REFERENCE: This pattern should be used for operation benchmarks.
func BenchmarkMonitorOperations(b *testing.B) {
	monitor := NewAdvancedMockMonitor("benchmark-monitor", "benchmark-type")
	ctx := context.Background()

	b.Run("health_check", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = monitor.IsHealthy(ctx)
		}
	})

	b.Run("metric_recording", func(b *testing.B) {
		helper := NewBenchmarkHelper(monitor)
		duration, err := helper.BenchmarkMetricRecording(b.N)
		require.NoError(b, err)
		b.ReportMetric(float64(b.N)/duration.Seconds(), "metrics/sec")
	})

	b.Run("tracing", func(b *testing.B) {
		helper := NewBenchmarkHelper(monitor)
		duration, err := helper.BenchmarkTracing(b.N)
		require.NoError(b, err)
		b.ReportMetric(float64(b.N)/duration.Seconds(), "traces/sec")
	})

	b.Run("logging", func(b *testing.B) {
		helper := NewBenchmarkHelper(monitor)
		duration, err := helper.BenchmarkLogging(b.N)
		require.NoError(b, err)
		b.ReportMetric(float64(b.N)/duration.Seconds(), "logs/sec")
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent operation performance.
func BenchmarkConcurrentOperations(b *testing.B) {
	monitor := NewAdvancedMockMonitor("concurrent-benchmark-monitor", "benchmark-type")
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = monitor.IsHealthy(ctx)
		}
	})
}
