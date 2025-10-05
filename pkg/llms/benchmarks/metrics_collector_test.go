// Package benchmarks provides contract tests for metrics collector interfaces.
// This file tests the MetricsCollector interface contract compliance.
package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsCollector_Contract tests the MetricsCollector interface contract
func TestMetricsCollector_Contract(t *testing.T) {
	ctx := context.Background()

	// Create metrics collector (will fail until implemented)
	collector, err := NewMetricsCollector(MetricsCollectorOptions{
		EnableLatencyTracking: true,
		EnableTokenTracking:   true,
		EnableMemoryTracking:  true,
		BufferSize:           1000,
	})
	require.NoError(t, err, "MetricsCollector creation should succeed")
	require.NotNil(t, collector, "MetricsCollector should not be nil")

	// Test collection lifecycle
	t.Run("CollectionLifecycle", func(t *testing.T) {
		benchmarkID := "test-benchmark-001"

		// Start collection
		err := collector.StartCollection(ctx, benchmarkID)
		assert.NoError(t, err, "Starting collection should succeed")

		// Record various metrics
		operations := []OperationMetrics{
			{
				OperationType:    "generate",
				StartTime:        time.Now().Add(-100 * time.Millisecond),
				EndTime:          time.Now(),
				TokensUsed:       TokenUsage{InputTokens: 50, OutputTokens: 75, TotalTokens: 125},
				BytesTransferred: 1024,
				MemoryUsed:       2048,
				ErrorOccurred:    false,
			},
			{
				OperationType:    "stream",
				StartTime:        time.Now().Add(-200 * time.Millisecond),
				EndTime:          time.Now().Add(-50 * time.Millisecond),
				TokensUsed:       TokenUsage{InputTokens: 30, OutputTokens: 100, TotalTokens: 130},
				BytesTransferred: 2048,
				MemoryUsed:       3072,
				ErrorOccurred:    false,
			},
		}

		for i, op := range operations {
			err := collector.RecordOperation(ctx, op)
			assert.NoError(t, err, "Recording operation %d should succeed", i)
		}

		// Record specific metrics
		err = collector.RecordLatency(ctx, 75*time.Millisecond, "generate")
		assert.NoError(t, err, "Recording latency should succeed")

		err = collector.RecordTokenUsage(ctx, TokenUsage{
			InputTokens:  40,
			OutputTokens: 60,
			TotalTokens:  100,
		})
		assert.NoError(t, err, "Recording token usage should succeed")

		// Stop collection and get results
		result, err := collector.StopCollection(ctx, benchmarkID)
		assert.NoError(t, err, "Stopping collection should succeed")
		assert.NotNil(t, result, "Collection result should not be nil")

		// Verify aggregated results
		assert.Equal(t, benchmarkID, result.BenchmarkID, "Should have correct benchmark ID")
		assert.GreaterOrEqual(t, result.OperationCount, 2, "Should record operation count")
		assert.Greater(t, result.TokenUsage.TotalTokens, 0, "Should aggregate token usage")
		assert.NotZero(t, result.Duration, "Should calculate total duration")
	})

	// Test concurrent collection
	t.Run("ConcurrentCollection", func(t *testing.T) {
		const numConcurrent = 5
		benchmarkIDs := make([]string, numConcurrent)
		
		// Start multiple concurrent collections
		for i := 0; i < numConcurrent; i++ {
			benchmarkIDs[i] = fmt.Sprintf("concurrent-benchmark-%d", i)
			err := collector.StartCollection(ctx, benchmarkIDs[i])
			assert.NoError(t, err, "Starting concurrent collection %d should succeed", i)
		}

		// Record metrics for each collection
		for i, benchmarkID := range benchmarkIDs {
			op := OperationMetrics{
				OperationType:    fmt.Sprintf("op-type-%d", i),
				StartTime:        time.Now().Add(-50 * time.Millisecond),
				EndTime:          time.Now(),
				TokensUsed:       TokenUsage{InputTokens: 25, OutputTokens: 50, TotalTokens: 75},
				BytesTransferred: 512,
				MemoryUsed:       1024,
				ErrorOccurred:    false,
			}

			// Switch context for each operation
			opCtx := context.WithValue(ctx, "benchmark_id", benchmarkID)
			err := collector.RecordOperation(opCtx, op)
			assert.NoError(t, err, "Recording operation for collection %d should succeed", i)
		}

		// Stop all collections and verify results
		for i, benchmarkID := range benchmarkIDs {
			result, err := collector.StopCollection(ctx, benchmarkID)
			assert.NoError(t, err, "Stopping collection %d should succeed", i)
			assert.NotNil(t, result, "Result %d should not be nil", i)
			assert.Equal(t, benchmarkID, result.BenchmarkID, "Result should have correct ID")
		}
	})

	// Test metrics aggregation accuracy
	t.Run("MetricsAggregationAccuracy", func(t *testing.T) {
		benchmarkID := "accuracy-test"
		err := collector.StartCollection(ctx, benchmarkID)
		require.NoError(t, err)

		// Record known quantities for verification
		expectedTokens := TokenUsage{InputTokens: 0, OutputTokens: 0, TotalTokens: 0}
		expectedLatencies := []time.Duration{}

		for i := 0; i < 10; i++ {
			latency := time.Duration(50+i*10) * time.Millisecond
			tokens := TokenUsage{
				InputTokens:  10 + i,
				OutputTokens: 20 + i*2,
				TotalTokens:  30 + i*3,
			}

			expectedTokens.InputTokens += tokens.InputTokens
			expectedTokens.OutputTokens += tokens.OutputTokens
			expectedTokens.TotalTokens += tokens.TotalTokens
			expectedLatencies = append(expectedLatencies, latency)

			err := collector.RecordLatency(ctx, latency, "test-op")
			assert.NoError(t, err, "Recording latency %d should succeed", i)

			err = collector.RecordTokenUsage(ctx, tokens)
			assert.NoError(t, err, "Recording tokens %d should succeed", i)
		}

		result, err := collector.StopCollection(ctx, benchmarkID)
		assert.NoError(t, err, "Collection should complete")
		require.NotNil(t, result)

		// Verify aggregation accuracy
		assert.Equal(t, expectedTokens.InputTokens, result.TokenUsage.InputTokens,
			"Input tokens should be accurately aggregated")
		assert.Equal(t, expectedTokens.OutputTokens, result.TokenUsage.OutputTokens,
			"Output tokens should be accurately aggregated")
		assert.Equal(t, expectedTokens.TotalTokens, result.TokenUsage.TotalTokens,
			"Total tokens should be accurately aggregated")
	})
}

// TestMetricsCollector_Performance tests performance constraints
func TestMetricsCollector_Performance(t *testing.T) {
	ctx := context.Background()

	collector, err := NewMetricsCollector(MetricsCollectorOptions{
		EnableLatencyTracking: true,
		EnableTokenTracking:   true,
		BufferSize:           10000,
	})
	require.NoError(t, err)

	// Test recording performance
	t.Run("RecordingPerformance", func(t *testing.T) {
		benchmarkID := "perf-test"
		err := collector.StartCollection(ctx, benchmarkID)
		require.NoError(t, err)

		const numRecordings = 1000
		start := time.Now()

		for i := 0; i < numRecordings; i++ {
			// Record latency
			err := collector.RecordLatency(ctx, time.Duration(i)*time.Microsecond, "perf-test")
			assert.NoError(t, err, "Recording should not fail")

			// Record token usage
			err = collector.RecordTokenUsage(ctx, TokenUsage{
				InputTokens:  10,
				OutputTokens: 15,
				TotalTokens:  25,
			})
			assert.NoError(t, err, "Token recording should not fail")
		}

		recordingDuration := time.Since(start)
		avgRecordingTime := recordingDuration / numRecordings

		// Stop collection
		_, err := collector.StopCollection(ctx, benchmarkID)
		assert.NoError(t, err, "Collection should complete")

		t.Logf("Average recording time: %v", avgRecordingTime)
		assert.Less(t, avgRecordingTime, 10*time.Microsecond,
			"Average recording should be <10μs (was %v)", avgRecordingTime)
	})

	// Test memory efficiency
	t.Run("MemoryEfficiency", func(t *testing.T) {
		// Test with large number of metrics to verify memory management
		benchmarkID := "memory-test"
		err := collector.StartCollection(ctx, benchmarkID)
		require.NoError(t, err)

		// Record large number of operations
		const numOperations = 5000
		for i := 0; i < numOperations; i++ {
			op := OperationMetrics{
				OperationType:    fmt.Sprintf("op-%d", i),
				StartTime:        time.Now().Add(-time.Duration(i) * time.Millisecond),
				EndTime:          time.Now(),
				TokensUsed:       TokenUsage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
				BytesTransferred: 1024,
				MemoryUsed:       2048,
			}

			err := collector.RecordOperation(ctx, op)
			assert.NoError(t, err, "Recording operation should succeed")
		}

		result, err := collector.StopCollection(ctx, benchmarkID)
		assert.NoError(t, err, "Collection should complete")
		assert.Equal(t, numOperations, result.OperationCount,
			"Should record all operations")

		// Memory usage should be reasonable for number of operations
		if result.MemoryUsage.PeakUsageBytes > 0 {
			bytesPerOp := result.MemoryUsage.PeakUsageBytes / int64(numOperations)
			assert.Less(t, bytesPerOp, int64(1024),
				"Memory per operation should be reasonable")
		}
	})
}

// TestMetricsCollector_ErrorHandling tests error scenarios
func TestMetricsCollector_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	collector, err := NewMetricsCollector(MetricsCollectorOptions{})
	require.NoError(t, err)

	// Test duplicate start collection
	t.Run("DuplicateStartCollection", func(t *testing.T) {
		benchmarkID := "duplicate-test"
		
		err := collector.StartCollection(ctx, benchmarkID)
		assert.NoError(t, err, "First start should succeed")
		
		err = collector.StartCollection(ctx, benchmarkID)
		assert.Error(t, err, "Duplicate start should fail")
	})

	// Test stop without start
	t.Run("StopWithoutStart", func(t *testing.T) {
		_, err := collector.StopCollection(ctx, "nonexistent-benchmark")
		assert.Error(t, err, "Stop without start should fail")
	})

	// Test recording without collection
	t.Run("RecordWithoutCollection", func(t *testing.T) {
		// Try to record metrics without starting collection
		err := collector.RecordLatency(ctx, time.Millisecond, "orphan-op")
		// Should either fail or handle gracefully
		if err != nil {
			t.Logf("Recording without collection properly failed: %v", err)
		}
	})

	// Test context cancellation
	t.Run("ContextCancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		
		benchmarkID := "cancel-test"
		err := collector.StartCollection(cancelCtx, benchmarkID)
		assert.NoError(t, err, "Start should succeed")
		
		// Cancel context
		cancel()
		
		// Try to record - should handle cancellation gracefully
		err = collector.RecordLatency(cancelCtx, time.Millisecond, "cancel-test-op")
		if err != nil {
			t.Logf("Properly handled context cancellation: %v", err)
		}

		// Stop collection with original context (not cancelled)
		_, err = collector.StopCollection(ctx, benchmarkID)
		assert.NoError(t, err, "Stop should succeed even after context cancellation")
	})
}
