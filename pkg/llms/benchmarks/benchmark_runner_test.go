// Package benchmarks provides contract tests for LLM benchmark runner interfaces.
// This file tests the BenchmarkRunner interface contract compliance.
package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// TestBenchmarkRunner_Contract tests the BenchmarkRunner interface contract
func TestBenchmarkRunner_Contract(t *testing.T) {
	ctx := context.Background()

	// Create benchmark runner (will fail until implemented)
	runner, err := NewBenchmarkRunner(BenchmarkRunnerOptions{
		EnableMetrics:  true,
		MaxConcurrency: 10,
		Timeout:        30 * time.Second,
	})
	require.NoError(t, err, "BenchmarkRunner creation should succeed")
	require.NotNil(t, runner, "BenchmarkRunner should not be nil")

	// Test basic benchmark execution
	t.Run("RunBenchmark", func(t *testing.T) {
		// Create mock provider (will use existing mock infrastructure)
		mockProvider := createMockChatModel(t, "test-provider", "test-model")

		// Create test scenario
		scenario := &TestBenchmarkScenario{
			name:    "basic-test",
			prompts: []string{"Test prompt 1", "Test prompt 2"},
			config: ScenarioConfig{
				OperationCount:   5,
				ConcurrencyLevel: 2,
				TimeoutDuration:  10 * time.Second,
			},
		}

		// Run benchmark
		result, err := runner.RunBenchmark(ctx, mockProvider, scenario)
		assert.NoError(t, err, "Benchmark execution should succeed")
		assert.NotNil(t, result, "Benchmark result should not be nil")

		// Verify result structure
		assert.NotEmpty(t, result.TestName, "Result should have test name")
		assert.NotEmpty(t, result.ProviderName, "Result should have provider name")
		assert.NotEmpty(t, result.ModelName, "Result should have model name")
		assert.Positive(t, result.Duration, "Result should have positive duration")
		assert.NotZero(t, result.Timestamp, "Result should have timestamp")
	})

	// Test comparison benchmark
	t.Run("RunComparisonBenchmark", func(t *testing.T) {
		// Create multiple mock providers
		providers := map[string]iface.ChatModel{
			"provider-a": createMockChatModel(t, "provider-a", "model-a"),
			"provider-b": createMockChatModel(t, "provider-b", "model-b"),
		}

		scenario := &TestBenchmarkScenario{
			name:    "comparison-test",
			prompts: []string{"Comparison prompt"},
			config: ScenarioConfig{
				OperationCount:   3,
				ConcurrencyLevel: 1,
				TimeoutDuration:  15 * time.Second,
			},
		}

		// Run comparison benchmark
		results, err := runner.RunComparisonBenchmark(ctx, providers, scenario)
		assert.NoError(t, err, "Comparison benchmark should succeed")
		assert.NotNil(t, results, "Comparison results should not be nil")
		assert.Len(t, results, 2, "Should have results for both providers")

		// Verify each result
		for providerName, result := range results {
			assert.NotNil(t, result, "Result for %s should not be nil", providerName)
			assert.Equal(t, providerName, result.ProviderName, "Result provider name should match")
		}
	})

	// Test load testing
	t.Run("RunLoadTest", func(t *testing.T) {
		mockProvider := createMockChatModel(t, "load-test-provider", "load-test-model")

		loadConfig := LoadTestConfig{
			Duration:       2 * time.Second,
			TargetRPS:      10,
			MaxConcurrency: 5,
			RampUpDuration: 500 * time.Millisecond,
			ScenarioName:   "load-test",
		}

		result, err := runner.RunLoadTest(ctx, mockProvider, loadConfig)
		assert.NoError(t, err, "Load test should succeed")
		assert.NotNil(t, result, "Load test result should not be nil")

		// Verify load test results
		assert.NotEmpty(t, result.TestID, "Load test should have ID")
		assert.Equal(t, loadConfig.Duration, result.Duration, "Duration should match config")
		assert.Equal(t, loadConfig.TargetRPS, result.TargetRPS, "Target RPS should match")
		assert.GreaterOrEqual(t, result.TotalOperations, 0, "Should record total operations")
	})

	// Test metrics support
	t.Run("GetSupportedMetrics", func(t *testing.T) {
		metrics := runner.GetSupportedMetrics()
		assert.NotEmpty(t, metrics, "Should support some metrics")

		expectedMetrics := []string{"latency", "throughput", "tokens", "errors", "memory"}
		for _, expectedMetric := range expectedMetrics {
			assert.Contains(t, metrics, expectedMetric, "Should support %s metric", expectedMetric)
		}
	})
}

// TestBenchmarkRunner_ErrorHandling tests error handling scenarios
func TestBenchmarkRunner_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	runner, err := NewBenchmarkRunner(BenchmarkRunnerOptions{})
	require.NoError(t, err)

	// Test with nil provider
	t.Run("NilProvider", func(t *testing.T) {
		scenario := &TestBenchmarkScenario{name: "nil-test"}

		_, err := runner.RunBenchmark(ctx, nil, scenario)
		assert.Error(t, err, "Should fail with nil provider")
	})

	// Test with nil scenario
	t.Run("NilScenario", func(t *testing.T) {
		mockProvider := createMockChatModel(t, "test", "test")

		_, err := runner.RunBenchmark(ctx, mockProvider, nil)
		assert.Error(t, err, "Should fail with nil scenario")
	})

	// Test with cancelled context
	t.Run("CancelledContext", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		mockProvider := createMockChatModel(t, "test", "test")
		scenario := &TestBenchmarkScenario{
			name: "cancel-test",
			config: ScenarioConfig{
				OperationCount:  1,
				TimeoutDuration: 10 * time.Second,
			},
		}

		_, err := runner.RunBenchmark(cancelCtx, mockProvider, scenario)
		assert.Error(t, err, "Should fail with cancelled context")
	})
}

// TestBenchmarkRunner_Performance tests performance constraints
func TestBenchmarkRunner_Performance(t *testing.T) {
	ctx := context.Background()

	runner, err := NewBenchmarkRunner(BenchmarkRunnerOptions{
		EnableMetrics:  true,
		MaxConcurrency: 20,
	})
	require.NoError(t, err)

	// Test benchmark suite performance (<30s target)
	t.Run("BenchmarkSuitePerformance", func(t *testing.T) {
		providers := make(map[string]iface.ChatModel)
		for i := 0; i < 4; i++ { // Simulate 4 providers
			providerName := fmt.Sprintf("perf-provider-%d", i)
			providers[providerName] = createMockChatModel(t, providerName, "model")
		}

		scenario := &TestBenchmarkScenario{
			name:    "performance-test",
			prompts: []string{"Performance test prompt"},
			config: ScenarioConfig{
				OperationCount:   10,
				ConcurrencyLevel: 2,
				TimeoutDuration:  30 * time.Second,
			},
		}

		start := time.Now()
		results, err := runner.RunComparisonBenchmark(ctx, providers, scenario)
		duration := time.Since(start)

		assert.NoError(t, err, "Performance benchmark should succeed")
		assert.Len(t, results, 4, "Should have results for all providers")

		// Performance target: <30s for full provider comparison
		assert.Less(t, duration, 30*time.Second,
			"Full provider comparison should complete in <30s (took %v)", duration)
	})

	// Test memory overhead (<10MB target)
	t.Run("MemoryOverhead", func(t *testing.T) {
		mockProvider := createMockChatModel(t, "memory-test", "model")

		scenario := &TestBenchmarkScenario{
			name:    "memory-test",
			prompts: []string{"Memory test prompt"},
			config: ScenarioConfig{
				OperationCount:   50,
				ConcurrencyLevel: 5,
				TimeoutDuration:  20 * time.Second,
			},
		}

		// Run benchmark and verify memory usage is reasonable
		result, err := runner.RunBenchmark(ctx, mockProvider, scenario)
		assert.NoError(t, err, "Memory test benchmark should succeed")

		if result != nil && result.MemoryUsage.PeakUsageBytes > 0 {
			// Memory overhead should be <10MB (10485760 bytes)
			assert.Less(t, result.MemoryUsage.PeakUsageBytes, int64(10485760),
				"Benchmark memory overhead should be <10MB")
		}
	})
}

// Helper types and functions for testing (will be implemented later)

// TestBenchmarkScenario implements BenchmarkScenario for testing
type TestBenchmarkScenario struct {
	name    string
	prompts []string
	config  ScenarioConfig
}

func (s *TestBenchmarkScenario) GetName() string { return s.name }
func (s *TestBenchmarkScenario) GetDescription() string {
	return fmt.Sprintf("Test scenario: %s", s.name)
}
func (s *TestBenchmarkScenario) GetTestPrompts() []string                        { return s.prompts }
func (s *TestBenchmarkScenario) GetConfiguration() ScenarioConfig                { return s.config }
func (s *TestBenchmarkScenario) ValidateProvider(provider iface.ChatModel) error { return nil }

// Helper function to create mock ChatModel (using existing infrastructure)
func createMockChatModel(t *testing.T, provider, model string) iface.ChatModel {
	// Use the existing AdvancedMockChatModel from test_utils.go
	mock := llms.NewAdvancedMockChatModel(model, llms.WithProviderName(provider))
	require.NotNil(t, mock, "Mock provider should not be nil")

	// Debug: check that provider name is set correctly
	actualProvider := mock.GetProviderName()
	require.Equal(t, provider, actualProvider, "Mock provider name should be set correctly")

	return mock
}
