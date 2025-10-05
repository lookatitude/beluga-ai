// Package integration provides integration tests for LLM streaming benchmark components.
// This file tests streaming performance analysis integration.
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/llms/benchmarks"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// TestStreamingPerformanceIntegration tests streaming benchmark integration
func TestStreamingPerformanceIntegration(t *testing.T) {
	ctx := context.Background()

	// Test Time-To-First-Token (TTFT) measurement
	t.Run("TTFTMeasurement", func(t *testing.T) {
		// Create streaming analyzer
		analyzer, err := benchmarks.NewStreamingAnalyzer(benchmarks.StreamingAnalyzerOptions{
			EnableTTFTTracking:       true,
			EnableThroughputTracking: true,
			SampleRate:               1.0, // Track all operations
		})
		require.NoError(t, err, "StreamingAnalyzer creation should succeed")

		// Create mock streaming provider
		streamingProvider := createMockStreamingProvider("streaming-test", "model", 50*time.Millisecond)

		// Test TTFT measurement
		ttftResult, err := analyzer.MeasureTTFT(ctx, streamingProvider, "Test streaming prompt for TTFT")
		assert.NoError(t, err, "TTFT measurement should succeed")
		assert.NotNil(t, ttftResult, "TTFT result should not be nil")

		// Verify TTFT metrics
		assert.Greater(t, ttftResult.TimeToFirstToken, time.Duration(0),
			"TTFT should be positive")
		assert.LessOrEqual(t, ttftResult.TimeToFirstToken, 5*time.Second,
			"TTFT should be reasonable for test")
		assert.NotEmpty(t, ttftResult.ProviderName, "Should have provider name")
		assert.NotEmpty(t, ttftResult.ModelName, "Should have model name")
	})

	// Test streaming throughput analysis
	t.Run("StreamingThroughputAnalysis", func(t *testing.T) {
		analyzer, err := benchmarks.NewStreamingAnalyzer(benchmarks.StreamingAnalyzerOptions{
			EnableThroughputTracking: true,
			ThroughputWindowSize:     time.Second,
		})
		require.NoError(t, err)

		streamingProvider := createMockStreamingProvider("throughput-test", "model", 25*time.Millisecond)

		// Analyze streaming throughput
		throughputResult, err := analyzer.AnalyzeThroughput(ctx, streamingProvider,
			"Long streaming prompt to analyze throughput characteristics over extended period")
		assert.NoError(t, err, "Throughput analysis should succeed")
		assert.NotNil(t, throughputResult, "Throughput result should not be nil")

		// Verify throughput metrics
		assert.Greater(t, throughputResult.TokensPerSecond, 0.0,
			"Tokens per second should be positive")
		assert.Greater(t, throughputResult.TotalStreamingTime, time.Duration(0),
			"Total streaming time should be positive")
		assert.Greater(t, throughputResult.TotalTokensStreamed, 0,
			"Should stream some tokens")
		assert.NotEmpty(t, throughputResult.ThroughputCurve,
			"Should have throughput curve data")
	})

	// Test backpressure handling in streaming
	t.Run("BackpressureHandling", func(t *testing.T) {
		analyzer, err := benchmarks.NewStreamingAnalyzer(benchmarks.StreamingAnalyzerOptions{
			EnableBackpressureTracking: true,
		})
		require.NoError(t, err)

		// Create provider with simulated backpressure
		backpressureProvider := createMockProviderWithBackpressure("backpressure-test", "model")

		// Test backpressure detection
		backpressureResult, err := analyzer.TestBackpressureHandling(ctx, backpressureProvider,
			"Long prompt that will cause backpressure during streaming response generation")
		assert.NoError(t, err, "Backpressure test should succeed")
		assert.NotNil(t, backpressureResult, "Backpressure result should not be nil")

		// Verify backpressure metrics
		assert.GreaterOrEqual(t, backpressureResult.BackpressureEvents, 0,
			"Should track backpressure events")
		assert.GreaterOrEqual(t, backpressureResult.MaxBufferSize, 0,
			"Should track buffer usage")
		assert.NotZero(t, backpressureResult.RecoveryTime,
			"Should measure recovery time")
	})

	// Test streaming memory analysis
	t.Run("StreamingMemoryAnalysis", func(t *testing.T) {
		analyzer, err := benchmarks.NewStreamingAnalyzer(benchmarks.StreamingAnalyzerOptions{
			EnableMemoryTracking: true,
		})
		require.NoError(t, err)

		provider := createMockStreamingProvider("memory-test", "model", 30*time.Millisecond)

		// Analyze memory usage during streaming
		memoryResult, err := analyzer.AnalyzeStreamingMemory(ctx, provider,
			"Very long prompt designed to test memory usage patterns during extended streaming operations")
		assert.NoError(t, err, "Streaming memory analysis should succeed")
		assert.NotNil(t, memoryResult, "Memory result should not be nil")

		// Verify memory metrics
		assert.GreaterOrEqual(t, memoryResult.PeakMemoryUsage, int64(0),
			"Peak memory should be non-negative")
		assert.GreaterOrEqual(t, memoryResult.AverageMemoryUsage, int64(0),
			"Average memory should be non-negative")
		assert.GreaterOrEqual(t, memoryResult.MemoryEfficiencyScore, 0.0,
			"Memory efficiency should be non-negative")
		assert.LessOrEqual(t, memoryResult.MemoryEfficiencyScore, 100.0,
			"Memory efficiency should be ≤100")
	})
}

// TestStreamingBenchmarkPerformance tests performance of streaming benchmarks
func TestStreamingBenchmarkPerformance(t *testing.T) {
	ctx := context.Background()

	// Test streaming benchmark suite performance
	t.Run("StreamingBenchmarkSuitePerformance", func(t *testing.T) {
		analyzer, err := benchmarks.NewStreamingAnalyzer(benchmarks.StreamingAnalyzerOptions{
			EnableTTFTTracking:       true,
			EnableThroughputTracking: true,
			EnableMemoryTracking:     true,
		})
		require.NoError(t, err)

		// Create multiple streaming providers
		streamingProviders := map[string]iface.ChatModel{
			"fast-streaming":   createMockStreamingProvider("fast", "model", 10*time.Millisecond),
			"medium-streaming": createMockStreamingProvider("medium", "model", 50*time.Millisecond),
			"slow-streaming":   createMockStreamingProvider("slow", "model", 200*time.Millisecond),
		}

		start := time.Now()

		// Run comprehensive streaming analysis on all providers
		for providerName, provider := range streamingProviders {
			// TTFT test
			ttftResult, err := analyzer.MeasureTTFT(ctx, provider, "TTFT test prompt")
			assert.NoError(t, err, "TTFT for %s should succeed", providerName)
			assert.NotNil(t, ttftResult, "TTFT result should not be nil")

			// Throughput test
			throughputResult, err := analyzer.AnalyzeThroughput(ctx, provider, "Throughput test prompt")
			assert.NoError(t, err, "Throughput for %s should succeed", providerName)
			assert.NotNil(t, throughputResult, "Throughput result should not be nil")

			// Memory test
			memoryResult, err := analyzer.AnalyzeStreamingMemory(ctx, provider, "Memory test prompt")
			assert.NoError(t, err, "Memory analysis for %s should succeed", providerName)
			assert.NotNil(t, memoryResult, "Memory result should not be nil")
		}

		totalDuration := time.Since(start)

		// Streaming benchmark suite should complete quickly
		assert.Less(t, totalDuration, 15*time.Second,
			"Streaming benchmark suite should complete in <15s (took %v)", totalDuration)

		t.Logf("Completed streaming benchmarks for %d providers in %v",
			len(streamingProviders), totalDuration)
	})

	// Test concurrent streaming benchmark execution
	t.Run("ConcurrentStreamingBenchmarks", func(t *testing.T) {
		analyzer, err := benchmarks.NewStreamingAnalyzer(benchmarks.StreamingAnalyzerOptions{})
		require.NoError(t, err)

		provider := createMockStreamingProvider("concurrent", "model", 75*time.Millisecond)

		const numConcurrentStreams = 10
		results := make(chan *benchmarks.TTFTResult, numConcurrentStreams)
		errors := make(chan error, numConcurrentStreams)

		start := time.Now()

		// Launch concurrent TTFT measurements
		for i := 0; i < numConcurrentStreams; i++ {
			go func(streamID int) {
				prompt := fmt.Sprintf("Concurrent streaming test prompt %d", streamID)
				result, err := analyzer.MeasureTTFT(ctx, provider, prompt)
				if err != nil {
					errors <- err
					return
				}
				results <- result
			}(i)
		}

		// Collect all results
		var ttftResults []*benchmarks.TTFTResult
		for i := 0; i < numConcurrentStreams; i++ {
			select {
			case result := <-results:
				ttftResults = append(ttftResults, result)
			case err := <-errors:
				t.Errorf("Concurrent streaming benchmark %d failed: %v", i, err)
			case <-time.After(30 * time.Second):
				t.Errorf("Concurrent streaming benchmark %d timed out", i)
			}
		}

		concurrentDuration := time.Since(start)

		// Verify concurrent execution
		assert.Len(t, ttftResults, numConcurrentStreams,
			"All concurrent streaming benchmarks should complete")

		// Concurrent execution should be faster than sequential
		expectedSequentialTime := time.Duration(numConcurrentStreams) * 100 * time.Millisecond
		assert.Less(t, concurrentDuration, expectedSequentialTime,
			"Concurrent execution should be faster than sequential")

		// Verify result consistency
		for i, result := range ttftResults {
			assert.Greater(t, result.TimeToFirstToken, time.Duration(0),
				"TTFT result %d should be positive", i)
			assert.NotEmpty(t, result.ProviderName,
				"TTFT result %d should have provider name", i)
		}

		t.Logf("Completed %d concurrent streaming benchmarks in %v",
			numConcurrentStreams, concurrentDuration)
	})
}

// Helper functions for streaming testing

func createMockStreamingProvider(provider, model string, latency time.Duration) iface.ChatModel {
	// This will create a mock provider that simulates streaming behavior
	// Will be implemented when enhanced mock infrastructure is available
	return nil
}

func createMockProviderWithBackpressure(provider, model string) iface.ChatModel {
	// This will create a mock provider that simulates backpressure scenarios
	// Will be implemented when enhanced mock infrastructure is available
	return nil
}
