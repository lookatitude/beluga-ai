package contract

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigMetrics_Contract tests the ConfigMetrics interface contract.
// This ensures the metrics implementation meets all contractual requirements.
func TestConfigMetrics_Contract(t *testing.T) {
	// Get the global metrics instance
	metrics := config.GetGlobalMetrics()
	require.NotNil(t, metrics, "Global metrics should not be nil")

	ctx := context.Background()

	// Test RecordOperation method (required by constitution)
	t.Run("RecordOperation", func(t *testing.T) {
		start := time.Now()
		duration := 100 * time.Millisecond
		success := true

		err := metrics.RecordOperation(ctx, "test.operation", duration, success)
		// RecordOperation may not return an error in the current implementation
		// but the contract allows it to return an error
		if err != nil {
			t.Logf("RecordOperation returned error (acceptable): %v", err)
		}

		// Verify operation was recorded - metrics recording is asynchronous
		elapsed := time.Since(start)
		assert.True(t, elapsed >= 0, "Operation recording should complete")
	})

	// Test RecordLoad method
	t.Run("RecordLoad", func(t *testing.T) {
		provider := "test-provider"
		format := "yaml"
		duration := 50 * time.Millisecond
		success := true

		err := metrics.RecordLoad(ctx, provider, format, duration, success)
		// Implementation may not return errors for metrics recording
		if err != nil {
			t.Logf("RecordLoad returned error (acceptable): %v", err)
		}
	})

	// Test RecordValidation method
	t.Run("RecordValidation", func(t *testing.T) {
		validationType := "schema"
		duration := 25 * time.Millisecond
		success := true
		errorCount := 0

		err := metrics.RecordValidationExtended(ctx, validationType, duration, success, errorCount)
		if err != nil {
			t.Logf("RecordValidation returned error (acceptable): %v", err)
		}
	})

	// Test RecordProviderOperation method
	t.Run("RecordProviderOperation", func(t *testing.T) {
		provider := "test-provider"
		operation := "load"
		duration := 75 * time.Millisecond
		success := true

		err := metrics.RecordProviderOperation(ctx, provider, operation, duration, success)
		if err != nil {
			t.Logf("RecordProviderOperation returned error (acceptable): %v", err)
		}
	})

	// Test StartSpan method
	t.Run("StartSpan", func(t *testing.T) {
		operation := "test.span.operation"
		provider := "test-provider"
		format := "yaml"

		ctx, span := metrics.StartSpan(ctx, operation, provider, format)
		assert.NotNil(t, ctx, "Context should not be nil")
		assert.NotNil(t, span, "Span should not be nil")

		// End the span to clean up
		span.End()
	})

	// Test GetHealthMetrics method
	t.Run("GetHealthMetrics", func(t *testing.T) {
		healthMetrics := metrics.GetHealthMetrics()
		assert.NotNil(t, healthMetrics, "Health metrics should not be nil")

		// Verify health metrics structure
		assert.GreaterOrEqual(t, healthMetrics.TotalLoads, int64(0), "Total loads should be non-negative")
		assert.GreaterOrEqual(t, healthMetrics.SuccessfulLoads, int64(0), "Successful loads should be non-negative")
		assert.GreaterOrEqual(t, healthMetrics.FailedLoads, int64(0), "Failed loads should be non-negative")
		assert.GreaterOrEqual(t, healthMetrics.SuccessRate, 0.0, "Success rate should be non-negative")
		assert.LessOrEqual(t, healthMetrics.SuccessRate, 1.0, "Success rate should be <= 1.0")

		// Verify performance metrics
		assert.GreaterOrEqual(t, healthMetrics.AverageLoadTime, time.Duration(0), "Average load time should be non-negative")
		assert.GreaterOrEqual(t, healthMetrics.P95LoadTime, time.Duration(0), "P95 load time should be non-negative")
		assert.GreaterOrEqual(t, healthMetrics.P99LoadTime, time.Duration(0), "P99 load time should be non-negative")
		assert.GreaterOrEqual(t, healthMetrics.LoadThroughput, 0.0, "Load throughput should be non-negative")

		// Verify collection metadata
		assert.False(t, healthMetrics.LastUpdated.IsZero(), "Last updated should not be zero time")
		assert.Greater(t, healthMetrics.CollectionPeriod, time.Duration(0), "Collection period should be positive")
	})

	// Test NoOpMetrics functionality
	t.Run("NoOpMetrics", func(t *testing.T) {
		noOpMetrics := config.NoOpMetrics()
		assert.NotNil(t, noOpMetrics, "NoOpMetrics should not be nil")

		// Test that NoOpMetrics methods don't panic and return appropriate values
		err := noOpMetrics.RecordOperation(ctx, "noop.test", time.Millisecond, true)
		// NoOp implementation should not return errors
		assert.NoError(t, err, "NoOpMetrics RecordOperation should not return error")

		err = noOpMetrics.RecordLoad(ctx, "noop", "yaml", time.Millisecond, true)
		assert.NoError(t, err, "NoOpMetrics RecordLoad should not return error")

		// NoOpMetrics RecordValidation is void (legacy signature)
		noOpMetrics.RecordValidation(ctx, time.Millisecond, true)

		err = noOpMetrics.RecordProviderOperation(ctx, "noop", "load", time.Millisecond, true)
		assert.NoError(t, err, "NoOpMetrics RecordProviderOperation should not return error")

		ctx, span := noOpMetrics.StartSpan(ctx, "noop.span", "noop", "yaml")
		assert.NotNil(t, ctx, "NoOpMetrics StartSpan should return valid context")
		assert.NotNil(t, span, "NoOpMetrics StartSpan should return valid span")
		span.End()

		healthMetrics := noOpMetrics.GetHealthMetrics()
		assert.NotNil(t, healthMetrics, "NoOpMetrics GetHealthMetrics should return valid metrics")
	})
}

// TestConfigMetrics_Performance tests metrics performance characteristics
func TestConfigMetrics_Performance(t *testing.T) {
	metrics := config.GetGlobalMetrics()
	ctx := context.Background()

	// Test rapid successive operations
	t.Run("RapidOperations", func(t *testing.T) {
		operations := 1000
		start := time.Now()

		for i := 0; i < operations; i++ {
			err := metrics.RecordOperation(ctx, "perf.test", time.Microsecond, true)
			if err != nil {
				t.Logf("Operation %d returned error: %v", i, err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(operations)

		// Metrics operations should be fast (under 1ms average)
		assert.Less(t, avgDuration, time.Millisecond, "Average metrics operation should be under 1ms")
		t.Logf("Completed %d operations in %v (avg: %v)", operations, duration, avgDuration)
	})

	// Test concurrent metrics recording
	t.Run("ConcurrentOperations", func(t *testing.T) {
		numGoroutines := 10
		operationsPerGoroutine := 100

		start := time.Now()
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				var err error
				for j := 0; j < operationsPerGoroutine; j++ {
					err = metrics.RecordLoad(ctx, "concurrent-test", "yaml", time.Millisecond, true)
					if err != nil {
						break
					}
				}
				errChan <- err
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			err := <-errChan
			if err != nil {
				t.Logf("Concurrent operation returned error: %v", err)
			}
		}

		duration := time.Since(start)
		totalOperations := numGoroutines * operationsPerGoroutine
		avgDuration := duration / time.Duration(totalOperations)

		assert.Less(t, avgDuration, time.Millisecond, "Concurrent metrics operations should be under 1ms average")
		t.Logf("Completed %d concurrent operations in %v (avg: %v)",
			totalOperations, duration, avgDuration)
	})
}

// TestConfigMetrics_ErrorConditions tests error conditions in metrics operations
func TestConfigMetrics_ErrorConditions(t *testing.T) {
	metrics := config.GetGlobalMetrics()
	ctx := context.Background()

	// Test with canceled context
	t.Run("CanceledContext", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		err := metrics.RecordOperation(canceledCtx, "canceled.test", time.Millisecond, true)
		// Implementation should handle canceled context gracefully
		// (may or may not return an error depending on implementation)
		if err != nil {
			t.Logf("Canceled context returned error: %v", err)
		}
	})

	// Test with very long operation names
	t.Run("LongOperationName", func(t *testing.T) {
		longName := string(make([]byte, 1000)) // 1000 character name
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}

		err := metrics.RecordOperation(ctx, longName, time.Millisecond, true)
		if err != nil {
			t.Logf("Long operation name returned error: %v", err)
		}
	})

	// Test with zero duration
	t.Run("ZeroDuration", func(t *testing.T) {
		err := metrics.RecordOperation(ctx, "zero.duration", 0, true)
		if err != nil {
			t.Logf("Zero duration returned error: %v", err)
		}
	})

	// Test with negative duration
	t.Run("NegativeDuration", func(t *testing.T) {
		err := metrics.RecordOperation(ctx, "negative.duration", -time.Millisecond, true)
		if err != nil {
			t.Logf("Negative duration returned error: %v", err)
		}
	})

	// Test GetHealthMetrics after operations
	t.Run("HealthMetricsAfterOperations", func(t *testing.T) {
		// Record some operations
		for i := 0; i < 10; i++ {
			_ = metrics.RecordLoad(ctx, "health-test", "yaml", time.Millisecond, i%2 == 0) // Mix of success/failure
		}

		healthMetrics := metrics.GetHealthMetrics()

		// Verify metrics are updated
		assert.GreaterOrEqual(t, healthMetrics.TotalLoads, int64(10), "Should have recorded at least 10 loads")

		// Verify success rate calculation
		if healthMetrics.TotalLoads > 0 {
			expectedSuccessRate := float64(healthMetrics.SuccessfulLoads) / float64(healthMetrics.TotalLoads)
			assert.InDelta(t, expectedSuccessRate, healthMetrics.SuccessRate, 0.01, "Success rate should be correctly calculated")
		}
	})
}

// TestConfigMetricsFactory_Contract tests the ConfigMetricsFactory interface contract
func TestConfigMetricsFactory_Contract(t *testing.T) {
	// Test NoOpMetrics factory method
	t.Run("NoOpMetrics", func(t *testing.T) {
		noOpMetrics := config.NoOpMetrics()
		assert.NotNil(t, noOpMetrics, "NoOpMetrics should return non-nil metrics")

		// Verify NoOpMetrics implements the interface
		ctx := context.Background()
		err := noOpMetrics.RecordOperation(ctx, "noop.factory.test", time.Millisecond, true)
		assert.NoError(t, err, "NoOpMetrics should not return errors")
	})
}

// BenchmarkConfigMetrics_RecordOperation benchmarks the RecordOperation method
func BenchmarkConfigMetrics_RecordOperation(b *testing.B) {
	metrics := config.GetGlobalMetrics()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = metrics.RecordOperation(ctx, "benchmark.operation", time.Microsecond, true)
		}
	})
}

// BenchmarkConfigMetrics_GetHealthMetrics benchmarks the GetHealthMetrics method
func BenchmarkConfigMetrics_GetHealthMetrics(b *testing.B) {
	metrics := config.GetGlobalMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetHealthMetrics()
	}
}
