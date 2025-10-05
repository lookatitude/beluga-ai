// Package integration provides integration tests for LLM benchmark components.
// This file tests cross-provider benchmark comparison integration.
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

// TestCrossProviderBenchmarkIntegration tests benchmark comparison across providers
func TestCrossProviderBenchmarkIntegration(t *testing.T) {
	ctx := context.Background()

	// Test complete provider comparison workflow
	t.Run("CompleteProviderComparison", func(t *testing.T) {
		// Create benchmark infrastructure
		runner, err := benchmarks.NewBenchmarkRunner(benchmarks.BenchmarkRunnerOptions{
			EnableMetrics:  true,
			MaxConcurrency: 10,
			Timeout:        60 * time.Second,
		})
		require.NoError(t, err, "BenchmarkRunner creation should succeed")

		analyzer, err := benchmarks.NewPerformanceAnalyzer(benchmarks.PerformanceAnalyzerOptions{
			EnableTrendAnalysis: true,
			MinSampleSize:       3,
			ConfidenceLevel:     0.95,
		})
		require.NoError(t, err, "PerformanceAnalyzer creation should succeed")

		// Create mock providers for different "providers" to compare
		providers := createMultiProviderTestSet()

		// Create comprehensive benchmark scenario
		scenario := benchmarks.NewStandardScenario("comprehensive-comparison", benchmarks.StandardScenarioConfig{
			TestPrompts: []string{
				"What is the capital of France?",
				"Explain machine learning in simple terms.",
				"Write a short story about a robot learning to paint.",
				"Analyze the following data and provide insights: [1,2,3,4,5]",
			},
			OperationCount:    20,
			ConcurrencyLevel:  3,
			TimeoutDuration:   30 * time.Second,
			RequiresTools:     false,
			RequiresStreaming: false,
		})

		// Run comparison benchmark
		results, err := runner.RunComparisonBenchmark(ctx, providers, scenario)
		assert.NoError(t, err, "Cross-provider comparison should succeed")
		assert.NotEmpty(t, results, "Should have benchmark results")
		assert.Len(t, results, len(providers), "Should have results for all providers")

		// Analyze results
		var allResults []*benchmarks.BenchmarkResult
		for _, result := range results {
			allResults = append(allResults, result)
		}

		analysis, err := analyzer.AnalyzeResults(allResults)
		assert.NoError(t, err, "Result analysis should succeed")
		assert.NotNil(t, analysis, "Analysis should not be nil")

		// Generate provider comparison
		comparison, err := analyzer.CompareProviders(results)
		assert.NoError(t, err, "Provider comparison should succeed")
		assert.NotNil(t, comparison, "Comparison should not be nil")

		// Verify comparison results
		assert.Len(t, comparison.ProviderRankings, len(providers),
			"Should rank all providers")
		assert.NotNil(t, comparison.PerformanceMatrix,
			"Should have performance comparison matrix")

		// Verify metrics make sense
		for providerName, ranking := range comparison.ProviderRankings {
			assert.Contains(t, providers, providerName,
				"Ranking should be for tested provider")
			assert.GreaterOrEqual(t, ranking.OverallScore, 0.0,
				"Provider score should be non-negative")
			assert.LessOrEqual(t, ranking.OverallScore, 100.0,
				"Provider score should be ≤100")
		}

		t.Logf("Successfully compared %d providers with %d total operations",
			len(providers), len(allResults))
	})

	// Test benchmark result aggregation and statistical analysis
	t.Run("BenchmarkResultAggregation", func(t *testing.T) {
		// Run multiple benchmarks for statistical significance
		const numRuns = 5
		var aggregatedResults []*benchmarks.BenchmarkResult

		providers := createMultiProviderTestSet()
		scenario := benchmarks.NewStandardScenario("aggregation-test", benchmarks.StandardScenarioConfig{
			TestPrompts:      []string{"Statistical test prompt"},
			OperationCount:   10,
			ConcurrencyLevel: 2,
			TimeoutDuration:  20 * time.Second,
		})

		runner, err := benchmarks.NewBenchmarkRunner(benchmarks.BenchmarkRunnerOptions{})
		require.NoError(t, err)

		// Run benchmarks multiple times
		for run := 0; run < numRuns; run++ {
			results, err := runner.RunComparisonBenchmark(ctx, providers, scenario)
			assert.NoError(t, err, "Benchmark run %d should succeed", run)

			for _, result := range results {
				aggregatedResults = append(aggregatedResults, result)
			}

			// Short delay between runs
			time.Sleep(100 * time.Millisecond)
		}

		// Verify we have results from all runs
		expectedResults := numRuns * len(providers)
		assert.Len(t, aggregatedResults, expectedResults,
			"Should have results from all runs and providers")

		// Analyze aggregated results
		analyzer, err := benchmarks.NewPerformanceAnalyzer(benchmarks.PerformanceAnalyzerOptions{})
		require.NoError(t, err)

		analysis, err := analyzer.AnalyzeResults(aggregatedResults)
		assert.NoError(t, err, "Aggregated analysis should succeed")
		assert.Equal(t, expectedResults, analysis.ResultsAnalyzed,
			"Analysis should include all results")

		// Verify statistical significance
		if analysis.StatisticalSummary != nil {
			assert.GreaterOrEqual(t, analysis.StatisticalSummary.SampleSize, numRuns,
				"Should have sufficient sample size")
			assert.GreaterOrEqual(t, analysis.StatisticalSummary.ConfidenceLevel, 0.8,
				"Should have reasonable confidence level")
		}
	})

	// Test benchmark performance under load
	t.Run("BenchmarkUnderLoad", func(t *testing.T) {
		// Test that benchmarking itself performs well under concurrent load
		const numConcurrentBenchmarks = 5

		providers := createMultiProviderTestSet()
		scenario := benchmarks.NewStandardScenario("load-test", benchmarks.StandardScenarioConfig{
			TestPrompts:      []string{"Load test prompt"},
			OperationCount:   5,
			ConcurrencyLevel: 2,
			TimeoutDuration:  15 * time.Second,
		})

		runner, err := benchmarks.NewBenchmarkRunner(benchmarks.BenchmarkRunnerOptions{
			MaxConcurrency: 20, // Allow high concurrency
		})
		require.NoError(t, err)

		// Run concurrent benchmarks
		results := make(chan map[string]*benchmarks.BenchmarkResult, numConcurrentBenchmarks)
		errors := make(chan error, numConcurrentBenchmarks)

		start := time.Now()

		for i := 0; i < numConcurrentBenchmarks; i++ {
			go func(benchmarkNum int) {
				// Add small variation to test parameters
				modifiedProviders := make(map[string]iface.ChatModel)
				for name, provider := range providers {
					modifiedProviders[fmt.Sprintf("%s-run%d", name, benchmarkNum)] = provider
				}

				result, err := runner.RunComparisonBenchmark(ctx, modifiedProviders, scenario)
				if err != nil {
					errors <- err
					return
				}
				results <- result
			}(i)
		}

		// Collect results
		var allConcurrentResults []map[string]*benchmarks.BenchmarkResult
		for i := 0; i < numConcurrentBenchmarks; i++ {
			select {
			case result := <-results:
				allConcurrentResults = append(allConcurrentResults, result)
			case err := <-errors:
				t.Errorf("Concurrent benchmark %d failed: %v", i, err)
			case <-time.After(45 * time.Second): // Timeout
				t.Errorf("Concurrent benchmark %d timed out", i)
			}
		}

		totalDuration := time.Since(start)

		// Verify concurrent execution
		assert.Len(t, allConcurrentResults, numConcurrentBenchmarks,
			"All concurrent benchmarks should complete")
		assert.Less(t, totalDuration, 30*time.Second,
			"Concurrent benchmarks should complete within reasonable time")

		// Verify each benchmark produced valid results
		for i, benchmarkResult := range allConcurrentResults {
			assert.NotEmpty(t, benchmarkResult,
				"Concurrent benchmark %d should have results", i)

			for providerName, result := range benchmarkResult {
				assert.NotNil(t, result,
					"Result for provider %s in benchmark %d should not be nil", providerName, i)
				assert.Greater(t, result.Duration, time.Duration(0),
					"Duration should be positive")
			}
		}

		t.Logf("Completed %d concurrent benchmarks in %v",
			numConcurrentBenchmarks, totalDuration)
	})
}

// Helper functions for integration testing

func createMultiProviderTestSet() map[string]iface.ChatModel {
	// Create a set of mock providers representing different real providers
	// Will use enhanced mock infrastructure from LLMs package
	return map[string]iface.ChatModel{
		"mock-openai":    createMockProviderWithCharacteristics("openai", "gpt-4", 200*time.Millisecond),
		"mock-anthropic": createMockProviderWithCharacteristics("anthropic", "claude-3", 150*time.Millisecond),
		"mock-bedrock":   createMockProviderWithCharacteristics("bedrock", "titan", 300*time.Millisecond),
		"mock-ollama":    createMockProviderWithCharacteristics("ollama", "llama2", 500*time.Millisecond),
	}
}

func createMockProviderWithCharacteristics(provider, model string, baseLatency time.Duration) iface.ChatModel {
	// This will create mock providers with realistic performance characteristics
	// Will be implemented when enhanced mock infrastructure is available
	return nil
}
