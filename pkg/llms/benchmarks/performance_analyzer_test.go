// Package benchmarks provides contract tests for performance analyzer interfaces.
// This file tests the PerformanceAnalyzer interface contract compliance.
package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerformanceAnalyzer_Contract tests the PerformanceAnalyzer interface contract
func TestPerformanceAnalyzer_Contract(t *testing.T) {
	// Create performance analyzer (will fail until implemented)
	analyzer, err := NewPerformanceAnalyzer(PerformanceAnalyzerOptions{
		EnableTrendAnalysis: true,
		MinSampleSize:       5,
		ConfidenceLevel:     0.95,
	})
	require.NoError(t, err, "PerformanceAnalyzer creation should succeed")
	require.NotNil(t, analyzer, "PerformanceAnalyzer should not be nil")

	// Test results analysis
	t.Run("AnalyzeResults", func(t *testing.T) {
		// Create test benchmark results
		results := createTestBenchmarkResults(5)

		analysis, err := analyzer.AnalyzeResults(results)
		assert.NoError(t, err, "Results analysis should succeed")
		assert.NotNil(t, analysis, "Analysis should not be nil")

		// Verify analysis structure
		assert.NotEmpty(t, analysis.AnalysisID, "Analysis should have ID")
		assert.NotZero(t, analysis.CreatedAt, "Analysis should have creation time")
		assert.Equal(t, len(results), analysis.ResultsAnalyzed, "Should analyze all results")
		assert.GreaterOrEqual(t, analysis.OverallScore, 0.0, "Overall score should be non-negative")
		assert.LessOrEqual(t, analysis.OverallScore, 100.0, "Overall score should be ≤100")
	})

	// Test provider comparison
	t.Run("CompareProviders", func(t *testing.T) {
		// Create results for different providers
		results := map[string]*BenchmarkResult{
			"openai":    createBenchmarkResult("openai", "gpt-4", 100*time.Millisecond),
			"anthropic": createBenchmarkResult("anthropic", "claude-3", 150*time.Millisecond),
			"bedrock":   createBenchmarkResult("bedrock", "titan", 200*time.Millisecond),
		}

		comparison, err := analyzer.CompareProviders(results)
		assert.NoError(t, err, "Provider comparison should succeed")
		assert.NotNil(t, comparison, "Comparison should not be nil")

		// Verify comparison structure
		assert.NotEmpty(t, comparison.ComparisonID, "Comparison should have ID")
		assert.Len(t, comparison.ProviderRankings, 3, "Should rank all providers")
		assert.NotNil(t, comparison.PerformanceMatrix, "Should have performance matrix")
	})

	// Test trend calculation
	t.Run("CalculateTrends", func(t *testing.T) {
		// Create historical results with trend
		historicalResults := createHistoricalResults(10, time.Hour)

		trends, err := analyzer.CalculateTrends(historicalResults)
		assert.NoError(t, err, "Trend calculation should succeed")
		assert.NotNil(t, trends, "Trends should not be nil")

		// Verify trend analysis
		assert.NotEmpty(t, trends.TrendID, "Trend analysis should have ID")
		assert.GreaterOrEqual(t, trends.ConfidenceLevel, 0.0, "Confidence should be non-negative")
		assert.LessOrEqual(t, trends.ConfidenceLevel, 1.0, "Confidence should be ≤1.0")
		assert.Positive(t, trends.DataPoints, "Should have data points")
	})

	// Test optimization recommendations
	t.Run("GenerateOptimizationRecommendations", func(t *testing.T) {
		results := createTestBenchmarkResults(3)
		analysis, err := analyzer.AnalyzeResults(results)
		require.NoError(t, err)

		recommendations, err := analyzer.GenerateOptimizationRecommendations(analysis)
		assert.NoError(t, err, "Generating recommendations should succeed")
		assert.NotNil(t, recommendations, "Recommendations should not be nil")

		// Verify recommendations structure
		for i, rec := range recommendations {
			assert.NotEmpty(t, rec.Title, "Recommendation %d should have title", i)
			assert.NotEmpty(t, rec.Description, "Recommendation %d should have description", i)
			assert.Contains(t, []string{"low", "medium", "high"}, rec.Priority,
				"Recommendation %d should have valid priority", i)
		}
	})
}

// TestPerformanceAnalyzer_StatisticalAccuracy tests statistical calculations
func TestPerformanceAnalyzer_StatisticalAccuracy(t *testing.T) {
	analyzer, err := NewPerformanceAnalyzer(PerformanceAnalyzerOptions{
		EnableTrendAnalysis: true,
		ConfidenceLevel:     0.95,
	})
	require.NoError(t, err)

	// Test percentile calculations
	t.Run("PercentileCalculations", func(t *testing.T) {
		// Create results with known latency distribution
		results := createResultsWithKnownLatencies([]time.Duration{
			10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond,
			40 * time.Millisecond, 50 * time.Millisecond, 60 * time.Millisecond,
			70 * time.Millisecond, 80 * time.Millisecond, 90 * time.Millisecond, 100 * time.Millisecond,
		})

		analysis, err := analyzer.AnalyzeResults(results)
		assert.NoError(t, err, "Analysis should succeed")

		// For 10 samples, verify percentile calculations
		// P50 should be around 50-55ms, P95 around 95ms, P99 around 99ms
		if analysis.LatencyAnalysis != nil {
			assert.Greater(t, analysis.LatencyAnalysis.P50, 40*time.Millisecond)
			assert.LessOrEqual(t, analysis.LatencyAnalysis.P50, 60*time.Millisecond)
			assert.Greater(t, analysis.LatencyAnalysis.P95, 90*time.Millisecond)
		}
	})

	// Test confidence interval calculations
	t.Run("ConfidenceIntervals", func(t *testing.T) {
		results := createTestBenchmarkResults(20) // Larger sample for confidence

		analysis, err := analyzer.AnalyzeResults(results)
		assert.NoError(t, err, "Analysis should succeed")

		if analysis.StatisticalSummary != nil {
			assert.NotZero(t, analysis.StatisticalSummary.ConfidenceInterval.Lower,
				"Should have lower confidence bound")
			assert.NotZero(t, analysis.StatisticalSummary.ConfidenceInterval.Upper,
				"Should have upper confidence bound")
			assert.Less(t, analysis.StatisticalSummary.ConfidenceInterval.Lower,
				analysis.StatisticalSummary.ConfidenceInterval.Upper,
				"Lower bound should be less than upper bound")
		}
	})
}

// Helper functions for testing (will be implemented later)

func createTestBenchmarkResults(count int) []*BenchmarkResult {
	results := make([]*BenchmarkResult, count)
	for i := 0; i < count; i++ {
		results[i] = createBenchmarkResult(
			fmt.Sprintf("provider-%d", i%2),         // Alternate providers
			fmt.Sprintf("model-%d", i%3),            // Cycle through models
			time.Duration(50+i*10)*time.Millisecond, // Varying latencies
		)
	}
	return results
}

func createHistoricalResults(count int, interval time.Duration) []*BenchmarkResult {
	results := make([]*BenchmarkResult, count)
	baseTime := time.Now().Add(-time.Duration(count) * interval)

	for i := 0; i < count; i++ {
		results[i] = createBenchmarkResult(
			"historical-provider",
			"historical-model",
			time.Duration(100+i*5)*time.Millisecond, // Slight performance degradation over time
		)
		results[i].Timestamp = baseTime.Add(time.Duration(i) * interval)
	}
	return results
}

func createResultsWithKnownLatencies(latencies []time.Duration) []*BenchmarkResult {
	results := make([]*BenchmarkResult, len(latencies))
	for i, latency := range latencies {
		results[i] = createBenchmarkResult("test-provider", "test-model", latency)
		results[i].LatencyMetrics.P50 = latency
		results[i].LatencyMetrics.Mean = latency
	}
	return results
}

func createBenchmarkResult(provider, model string, duration time.Duration) *BenchmarkResult {
	return &BenchmarkResult{
		BenchmarkID:  fmt.Sprintf("bench-%d", time.Now().UnixNano()),
		TestName:     "test-scenario",
		ProviderName: provider,
		ModelName:    model,
		Duration:     duration,
		Timestamp:    time.Now(),
		LatencyMetrics: LatencyMetrics{
			P50:  duration,
			P95:  duration + 10*time.Millisecond,
			P99:  duration + 20*time.Millisecond,
			Mean: duration,
		},
		TokenUsage: TokenUsage{
			InputTokens:  100,
			OutputTokens: 150,
			TotalTokens:  250,
		},
		ThroughputRPS:    10.0,
		ErrorRate:        0.01,
		OperationCount:   10,
		ConcurrencyLevel: 2,
		SuccessfulOps:    9,
		FailedOps:        1,
	}
}
