// Package integration provides integration tests for LLM load testing infrastructure.
// This file tests load testing integration and stress analysis.
package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/llms/benchmarks"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// TestLoadTestingIntegration tests comprehensive load testing capabilities
func TestLoadTestingIntegration(t *testing.T) {
	ctx := context.Background()

	// Test sustained load testing
	t.Run("SustainedLoadTesting", func(t *testing.T) {
		// Create load tester
		loadTester, err := benchmarks.NewLoadTester(benchmarks.LoadTesterOptions{
			MaxConcurrency:     50,
			DefaultTimeout:     30 * time.Second,
			EnableMetrics:      true,
			EnableStressMode:   false, // Normal load testing
		})
		require.NoError(t, err, "LoadTester creation should succeed")

		// Create test provider
		provider := createMockProviderForLoad("load-test", "model", 100*time.Millisecond)
		
		// Configure load test
		loadConfig := benchmarks.LoadTestConfig{
			Duration:       5 * time.Second,
			TargetRPS:      20,
			MaxConcurrency: 10,
			RampUpDuration: 1 * time.Second,
			ScenarioName:   "sustained-load-test",
		}

		// Run load test
		loadResult, err := loadTester.RunLoadTest(ctx, provider, loadConfig)
		assert.NoError(t, err, "Load test should succeed")
		assert.NotNil(t, loadResult, "Load result should not be nil")

		// Verify load test results
		assert.NotEmpty(t, loadResult.TestID, "Load test should have ID")
		assert.Equal(t, loadConfig.Duration, loadResult.Duration, "Duration should match")
		assert.Equal(t, loadConfig.TargetRPS, loadResult.TargetRPS, "Target RPS should match")
		assert.GreaterOrEqual(t, loadResult.TotalOperations, 50, "Should execute reasonable number of operations")
		assert.GreaterOrEqual(t, loadResult.SuccessfulOps, 0, "Should track successful operations")
		assert.GreaterOrEqual(t, loadResult.FailedOps, 0, "Should track failed operations")

		// Verify performance metrics
		assert.Greater(t, loadResult.LatencyMetrics.Mean, time.Duration(0),
			"Should measure latency")
		assert.GreaterOrEqual(t, loadResult.ActualRPS, 0.0,
			"Should measure actual RPS")
		assert.NotEmpty(t, loadResult.ThroughputCurve,
			"Should have throughput curve data")

		t.Logf("Load test completed: %d ops (%d success, %d failed), actual RPS: %.1f",
			loadResult.TotalOperations, loadResult.SuccessfulOps, loadResult.FailedOps, loadResult.ActualRPS)
	})

	// Test stress testing with high concurrency
	t.Run("StressTesting", func(t *testing.T) {
		loadTester, err := benchmarks.NewLoadTester(benchmarks.LoadTesterOptions{
			MaxConcurrency:   100,
			DefaultTimeout:   60 * time.Second,
			EnableStressMode: true,
		})
		require.NoError(t, err)

		provider := createMockProviderForLoad("stress-test", "model", 200*time.Millisecond)

		// Configure stress test
		stressConfig := benchmarks.LoadTestConfig{
			Duration:       3 * time.Second,
			TargetRPS:      100, // High load
			MaxConcurrency: 50,  // High concurrency
			RampUpDuration: 500 * time.Millisecond,
			ScenarioName:   "stress-test",
		}

		// Run stress test
		stressResult, err := loadTester.RunLoadTest(ctx, provider, stressConfig)
		assert.NoError(t, err, "Stress test should succeed")
		assert.NotNil(t, stressResult, "Stress result should not be nil")

		// Verify stress test handles high load
		assert.GreaterOrEqual(t, stressResult.TotalOperations, 100,
			"Should execute many operations under stress")
		
		// Error rate might be higher under stress, but should be tracked
		errorRate := float64(stressResult.FailedOps) / float64(stressResult.TotalOperations)
		assert.LessOrEqual(t, errorRate, 0.5, "Error rate under stress should be ≤50%")
		
		// Should still achieve reasonable performance
		if stressResult.ActualRPS > 0 {
			efficiencyPercent := (stressResult.ActualRPS / float64(stressConfig.TargetRPS)) * 100
			assert.GreaterOrEqual(t, efficiencyPercent, 20.0,
				"Should achieve at least 20% of target RPS under stress")
		}

		t.Logf("Stress test: target %d RPS, achieved %.1f RPS, error rate %.1f%%",
			stressConfig.TargetRPS, stressResult.ActualRPS, errorRate*100)
	})

	// Test load testing with error simulation
	t.Run("LoadTestingWithErrors", func(t *testing.T) {
		loadTester, err := benchmarks.NewLoadTester(benchmarks.LoadTesterOptions{})
		require.NoError(t, err)

		// Create provider that simulates errors under load
		errorProvider := createMockProviderWithErrors("error-test", "model", 0.1) // 10% error rate

		loadConfig := benchmarks.LoadTestConfig{
			Duration:       4 * time.Second,
			TargetRPS:      30,
			MaxConcurrency: 15,
			RampUpDuration: 1 * time.Second,
			ScenarioName:   "error-simulation-test",
		}

		loadResult, err := loadTester.RunLoadTest(ctx, errorProvider, loadConfig)
		assert.NoError(t, err, "Load test with errors should succeed")

		// Verify error tracking
		assert.Greater(t, loadResult.FailedOps, 0, "Should have some failed operations")
		assert.NotEmpty(t, loadResult.ErrorRateOverTime, "Should track error rate over time")

		// Verify error analysis
		for _, errorPoint := range loadResult.ErrorRateOverTime {
			assert.GreaterOrEqual(t, errorPoint.ErrorRate, 0.0, "Error rate should be non-negative")
			assert.LessOrEqual(t, errorPoint.ErrorRate, 1.0, "Error rate should be ≤1.0")
			assert.GreaterOrEqual(t, errorPoint.ErrorCount, 0, "Error count should be non-negative")
		}

		// Calculate overall error rate
		overallErrorRate := float64(loadResult.FailedOps) / float64(loadResult.TotalOperations)
		assert.InDelta(t, 0.1, overallErrorRate, 0.05, 
			"Overall error rate should be close to configured 10%")

		t.Logf("Load test with errors: %.1f%% error rate (%d/%d operations)",
			overallErrorRate*100, loadResult.FailedOps, loadResult.TotalOperations)
	})
}

// TestLoadTestingMultiProvider tests load testing across multiple providers
func TestLoadTestingMultiProvider(t *testing.T) {
	ctx := context.Background()

	loadTester, err := benchmarks.NewLoadTester(benchmarks.LoadTesterOptions{})
	require.NoError(t, err)

	// Test load balancing across providers
	t.Run("LoadBalancingAcrossProviders", func(t *testing.T) {
		// Create providers with different performance characteristics
		providers := map[string]iface.ChatModel{
			"fast-provider":   createMockProviderForLoad("fast", "model", 50*time.Millisecond),
			"medium-provider": createMockProviderForLoad("medium", "model", 150*time.Millisecond),
			"slow-provider":   createMockProviderForLoad("slow", "model", 400*time.Millisecond),
		}

		loadConfig := benchmarks.LoadTestConfig{
			Duration:       6 * time.Second,
			TargetRPS:      45, // Total across all providers
			MaxConcurrency: 25,
			RampUpDuration: 1 * time.Second,
			ScenarioName:   "multi-provider-load-test",
		}

		// Run load test against each provider
		var allResults []*benchmarks.LoadTestResult
		var totalDuration time.Duration

		for providerName, provider := range providers {
			// Adjust target RPS for individual providers (distribute the load)
			individualConfig := loadConfig
			individualConfig.TargetRPS = 15 // Distribute 45 RPS across 3 providers

			start := time.Now()
			result, err := loadTester.RunLoadTest(ctx, provider, individualConfig)
			providerDuration := time.Since(start)
			totalDuration += providerDuration

			assert.NoError(t, err, "Load test for %s should succeed", providerName)
			assert.NotNil(t, result, "Result for %s should not be nil", providerName)

			allResults = append(allResults, result)

			t.Logf("Provider %s: %.1f RPS achieved (target: %d), latency p95: %v",
				providerName, result.ActualRPS, individualConfig.TargetRPS,
				result.LatencyMetrics.P95)
		}

		// Verify multi-provider results
		assert.Len(t, allResults, len(providers), "Should have results for all providers")

		// Calculate aggregate metrics
		var totalOps, totalSuccessful, totalFailed int
		var totalActualRPS float64

		for _, result := range allResults {
			totalOps += result.TotalOperations
			totalSuccessful += result.SuccessfulOps
			totalFailed += result.FailedOps
			totalActualRPS += result.ActualRPS
		}

		overallSuccessRate := float64(totalSuccessful) / float64(totalOps) * 100

		t.Logf("Multi-provider load test summary:")
		t.Logf("  Total operations: %d (%d success, %d failed)", totalOps, totalSuccessful, totalFailed)
		t.Logf("  Success rate: %.1f%%", overallSuccessRate)
		t.Logf("  Total RPS achieved: %.1f (target: %d)", totalActualRPS, loadConfig.TargetRPS)
		t.Logf("  Total execution time: %v", totalDuration)

		// Multi-provider load testing should maintain reasonable success rates
		assert.GreaterOrEqual(t, overallSuccessRate, 80.0,
			"Multi-provider success rate should be ≥80%")
	})

	// Test provider failover during load testing
	t.Run("ProviderFailoverDuringLoad", func(t *testing.T) {
		// Create providers where one fails partway through
		providers := map[string]iface.ChatModel{
			"reliable-provider": createMockProviderForLoad("reliable", "model", 100*time.Millisecond),
			"failing-provider":  createMockProviderWithTimedFailure("failing", "model", 2*time.Second), // Fails after 2s
		}

		loadConfig := benchmarks.LoadTestConfig{
			Duration:       5 * time.Second, // Longer than failure time
			TargetRPS:      25,
			MaxConcurrency: 12,
			RampUpDuration: 500 * time.Millisecond,
			ScenarioName:   "failover-test",
		}

		var results []*benchmarks.LoadTestResult
		var wg sync.WaitGroup

		// Run load tests concurrently
		for providerName, provider := range providers {
			wg.Add(1)
			go func(name string, p iface.ChatModel) {
				defer wg.Done()
				
				result, err := loadTester.RunLoadTest(ctx, p, loadConfig)
				if err != nil {
					t.Logf("Provider %s failed as expected: %v", name, err)
					return
				}
				
				if result != nil {
					results = append(results, result)
				}
			}(providerName, provider)
		}

		wg.Wait()

		// At least one provider should have completed successfully
		assert.Greater(t, len(results), 0, "At least one provider should succeed")

		// Verify that failing provider behavior is captured
		for _, result := range results {
			assert.NotEmpty(t, result.TestID, "Result should have test ID")
			
			// Check if error rate increased over time (indicating failure)
			if len(result.ErrorRateOverTime) > 1 {
				firstErrorRate := result.ErrorRateOverTime[0].ErrorRate
				lastErrorRate := result.ErrorRateOverTime[len(result.ErrorRateOverTime)-1].ErrorRate
				
				if lastErrorRate > firstErrorRate {
					t.Logf("Detected error rate increase from %.1f%% to %.1f%% (failure captured)",
						firstErrorRate*100, lastErrorRate*100)
				}
			}
		}
	})
}

// TestLoadTestingPerformance tests the performance of load testing infrastructure itself
func TestLoadTestingPerformance(t *testing.T) {
	ctx := context.Background()

	loadTester, err := benchmarks.NewLoadTester(benchmarks.LoadTesterOptions{
		MaxConcurrency: 100,
	})
	require.NoError(t, err)

	// Test load test overhead
	t.Run("LoadTestOverhead", func(t *testing.T) {
		provider := createMockProviderForLoad("overhead-test", "model", 10*time.Millisecond)

		// Configure test with known parameters
		loadConfig := benchmarks.LoadTestConfig{
			Duration:       2 * time.Second,
			TargetRPS:      50,
			MaxConcurrency: 20,
			RampUpDuration: 300 * time.Millisecond,
			ScenarioName:   "overhead-measurement",
		}

		// Measure load test execution time
		start := time.Now()
		result, err := loadTester.RunLoadTest(ctx, provider, loadConfig)
		totalDuration := time.Since(start)

		assert.NoError(t, err, "Load test should succeed")
		assert.NotNil(t, result, "Result should not be nil")

		// Load test overhead should be minimal
		expectedDuration := loadConfig.Duration + loadConfig.RampUpDuration + 500*time.Millisecond // Allow 500ms overhead
		assert.Less(t, totalDuration, expectedDuration,
			"Load test overhead should be minimal (took %v, expected ≤%v)", totalDuration, expectedDuration)

		// Verify test executed expected number of operations
		expectedOps := int(loadConfig.Duration.Seconds() * float64(loadConfig.TargetRPS))
		actualOpsPercent := float64(result.TotalOperations) / float64(expectedOps) * 100
		assert.GreaterOrEqual(t, actualOpsPercent, 70.0,
			"Should achieve at least 70% of target operations")

		t.Logf("Load test efficiency: %.1f%% (%d/%d ops), overhead: %v",
			actualOpsPercent, result.TotalOperations, expectedOps, 
			totalDuration-(loadConfig.Duration+loadConfig.RampUpDuration))
	})

	// Test memory efficiency during load testing
	t.Run("LoadTestMemoryEfficiency", func(t *testing.T) {
		provider := createMockProviderForLoad("memory-test", "model", 75*time.Millisecond)

		loadConfig := benchmarks.LoadTestConfig{
			Duration:       3 * time.Second,
			TargetRPS:      60,
			MaxConcurrency: 30,
			RampUpDuration: 500 * time.Millisecond,
			ScenarioName:   "memory-efficiency-test",
		}

		result, err := loadTester.RunLoadTest(ctx, provider, loadConfig)
		assert.NoError(t, err, "Memory efficiency test should succeed")

		// Verify memory usage is tracked and reasonable
		if result.MemoryMetrics.PeakUsageBytes > 0 {
			memoryPerOp := result.MemoryMetrics.PeakUsageBytes / int64(result.TotalOperations)
			assert.Less(t, memoryPerOp, int64(1024*1024), // <1MB per operation
				"Memory per operation should be reasonable")

			memoryEfficiency := float64(result.MemoryMetrics.AverageUsageBytes) / 
				float64(result.MemoryMetrics.PeakUsageBytes) * 100
			assert.GreaterOrEqual(t, memoryEfficiency, 60.0,
				"Memory efficiency should be ≥60%")

			t.Logf("Memory usage: peak %d bytes, avg %d bytes, efficiency %.1f%%",
				result.MemoryMetrics.PeakUsageBytes, result.MemoryMetrics.AverageUsageBytes, memoryEfficiency)
		}
	})

	// Test concurrent load tests
	t.Run("ConcurrentLoadTests", func(t *testing.T) {
		const numConcurrentTests = 3
		providers := map[string]iface.ChatModel{
			"concurrent-1": createMockProviderForLoad("concurrent-1", "model", 80*time.Millisecond),
			"concurrent-2": createMockProviderForLoad("concurrent-2", "model", 120*time.Millisecond),
			"concurrent-3": createMockProviderForLoad("concurrent-3", "model", 160*time.Millisecond),
		}

		loadConfig := benchmarks.LoadTestConfig{
			Duration:       2 * time.Second,
			TargetRPS:      20,
			MaxConcurrency: 10,
			RampUpDuration: 300 * time.Millisecond,
			ScenarioName:   "concurrent-load-test",
		}

		// Run concurrent load tests
		results := make(chan *benchmarks.LoadTestResult, numConcurrentTests)
		errors := make(chan error, numConcurrentTests)
		var wg sync.WaitGroup

		start := time.Now()

		for providerName, provider := range providers {
			wg.Add(1)
			go func(name string, p iface.ChatModel) {
				defer wg.Done()
				
				result, err := loadTester.RunLoadTest(ctx, p, loadConfig)
				if err != nil {
					errors <- err
					return
				}
				results <- result
			}(providerName, provider)
		}

		wg.Wait()
		close(results)
		close(errors)

		concurrentDuration := time.Since(start)

		// Collect results
		var loadResults []*benchmarks.LoadTestResult
		for result := range results {
			loadResults = append(loadResults, result)
		}

		// Check for errors
		errorCount := 0
		for err := range errors {
			if err != nil {
				errorCount++
				t.Logf("Load test error: %v", err)
			}
		}

		// Verify concurrent execution
		assert.Equal(t, 0, errorCount, "No load tests should fail")
		assert.Len(t, loadResults, numConcurrentTests, "All load tests should complete")

		// Concurrent load tests should not interfere with each other significantly
		expectedDuration := loadConfig.Duration + loadConfig.RampUpDuration + time.Second // 1s overhead
		assert.Less(t, concurrentDuration, expectedDuration*2,
			"Concurrent load tests should not have excessive mutual interference")

		// Verify individual test quality
		for i, result := range loadResults {
			assert.Greater(t, result.TotalOperations, 20,
				"Load test %d should execute reasonable number of operations", i)
			assert.GreaterOrEqual(t, result.ActualRPS, 10.0,
				"Load test %d should achieve reasonable RPS", i)
		}

		t.Logf("Completed %d concurrent load tests in %v", numConcurrentTests, concurrentDuration)
	})
}

// TestLoadTestingScenarios tests various load testing scenarios
func TestLoadTestingScenarios(t *testing.T) {
	ctx := context.Background()

	loadTester, err := benchmarks.NewLoadTester(benchmarks.LoadTesterOptions{})
	require.NoError(t, err)

	// Test different load patterns
	loadPatterns := []struct {
		name   string
		config benchmarks.LoadTestConfig
	}{
		{
			name: "burst-load",
			config: benchmarks.LoadTestConfig{
				Duration:       1 * time.Second,
				TargetRPS:      100, // High burst
				MaxConcurrency: 50,
				RampUpDuration: 100 * time.Millisecond, // Fast ramp
				ScenarioName:   "burst-pattern",
			},
		},
		{
			name: "gradual-ramp",
			config: benchmarks.LoadTestConfig{
				Duration:       4 * time.Second,
				TargetRPS:      30,
				MaxConcurrency: 15,
				RampUpDuration: 2 * time.Second, // Slow ramp
				ScenarioName:   "gradual-pattern",
			},
		},
		{
			name: "sustained-moderate",
			config: benchmarks.LoadTestConfig{
				Duration:       5 * time.Second,
				TargetRPS:      25,
				MaxConcurrency: 12,
				RampUpDuration: 500 * time.Millisecond,
				ScenarioName:   "sustained-pattern",
			},
		},
	}

	provider := createMockProviderForLoad("pattern-test", "model", 100*time.Millisecond)

	for _, pattern := range loadPatterns {
		t.Run(pattern.name, func(t *testing.T) {
			result, err := loadTester.RunLoadTest(ctx, provider, pattern.config)
			assert.NoError(t, err, "Load pattern %s should succeed", pattern.name)
			assert.NotNil(t, result, "Result should not be nil")

			// Verify pattern-specific expectations
			switch pattern.name {
			case "burst-load":
				// Burst should achieve high peak RPS
				assert.GreaterOrEqual(t, result.ActualRPS, 50.0,
					"Burst pattern should achieve high RPS")
			case "gradual-ramp":
				// Gradual should have lower error rates due to gentle ramp
				errorRate := float64(result.FailedOps) / float64(result.TotalOperations)
				assert.LessOrEqual(t, errorRate, 0.05,
					"Gradual ramp should have low error rate")
			case "sustained-moderate":
				// Sustained should have consistent performance
				assert.GreaterOrEqual(t, result.TotalOperations, 100,
					"Sustained pattern should execute many operations")
			}

			t.Logf("Pattern %s: %d ops (%.1f RPS), %.1f%% success rate",
				pattern.name, result.TotalOperations, result.ActualRPS,
				float64(result.SuccessfulOps)/float64(result.TotalOperations)*100)
		})
	}
}

// Helper functions for load testing

func createMockProviderForLoad(provider, model string, baseLatency time.Duration) iface.ChatModel {
	// This will create a mock provider optimized for load testing
	// Will be implemented when load testing infrastructure is available
	return nil
}

func createMockProviderWithErrors(provider, model string, errorRate float64) iface.ChatModel {
	// This will create a mock provider that simulates errors at specified rate
	// Will be implemented when enhanced mock infrastructure is available
	return nil
}

func createMockProviderWithTimedFailure(provider, model string, failAfter time.Duration) iface.ChatModel {
	// This will create a mock provider that fails after specified duration
	// Will be implemented when enhanced mock infrastructure is available
	return nil
}
