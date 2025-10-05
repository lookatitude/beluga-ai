package benchmarks

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// BenchmarkRunner implements the BenchmarkRunner interface for executing
// performance benchmarks against LLM providers.
type BenchmarkRunner struct {
	options        BenchmarkRunnerOptions
	metricsEnabled bool
	mu             sync.RWMutex
}

// NewBenchmarkRunner creates a new benchmark runner with the specified options
func NewBenchmarkRunner(options BenchmarkRunnerOptions) (*BenchmarkRunner, error) {
	if options.MaxConcurrency == 0 {
		options.MaxConcurrency = 10
	}
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}

	return &BenchmarkRunner{
		options:        options,
		metricsEnabled: options.EnableMetrics,
	}, nil
}

// RunBenchmark executes a benchmark scenario against the specified provider
func (br *BenchmarkRunner) RunBenchmark(ctx context.Context, provider iface.ChatModel, scenario BenchmarkScenario) (*BenchmarkResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}
	if scenario == nil {
		return nil, fmt.Errorf("scenario cannot be nil")
	}

	// Validate provider supports scenario requirements
	if err := scenario.ValidateProvider(provider); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	config := scenario.GetConfiguration()
	prompts := scenario.GetTestPrompts()

	// Create benchmark result
	result := &BenchmarkResult{
		BenchmarkID:      fmt.Sprintf("benchmark-%d", time.Now().UnixNano()),
		TestName:         scenario.GetName(),
		ProviderName:     br.extractProviderName(provider),
		ModelName:        br.extractModelName(provider),
		Timestamp:        time.Now(),
		ScenarioName:     scenario.GetName(),
		OperationCount:   config.OperationCount,
		ConcurrencyLevel: config.ConcurrencyLevel,
		Environment:      br.captureEnvironment(),
	}

	// Execute benchmark with timeout
	benchmarkCtx, cancel := context.WithTimeout(ctx, config.TimeoutDuration)
	defer cancel()

	start := time.Now()

	// Run operations with concurrency control
	err := br.executeOperations(benchmarkCtx, provider, prompts, config, result)

	result.Duration = time.Since(start)

	if err != nil {
		return result, fmt.Errorf("benchmark execution failed: %w", err)
	}

	// Calculate derived metrics
	br.calculateDerivedMetrics(result)

	return result, nil
}

// RunComparisonBenchmark executes the same scenario across multiple providers
func (br *BenchmarkRunner) RunComparisonBenchmark(ctx context.Context, providers map[string]iface.ChatModel, scenario BenchmarkScenario) (map[string]*BenchmarkResult, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers specified for comparison")
	}
	if scenario == nil {
		return nil, fmt.Errorf("scenario cannot be nil")
	}

	results := make(map[string]*BenchmarkResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Run benchmarks in parallel for each provider
	for providerName, provider := range providers {
		wg.Add(1)
		go func(name string, prov iface.ChatModel) {
			defer wg.Done()

			result, err := br.RunBenchmark(ctx, prov, scenario)
			if err != nil {
				// Log error but continue with other providers
				result = &BenchmarkResult{
					BenchmarkID:  fmt.Sprintf("failed-benchmark-%d", time.Now().UnixNano()),
					TestName:     scenario.GetName(),
					ProviderName: name,
					Timestamp:    time.Now(),
					ErrorRate:    1.0, // 100% error rate for failed benchmark
				}
			}

			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(providerName, provider)
	}

	wg.Wait()
	return results, nil
}

// RunLoadTest executes sustained load testing
func (br *BenchmarkRunner) RunLoadTest(ctx context.Context, provider iface.ChatModel, config LoadTestConfig) (*LoadTestResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid load test config: %w", err)
	}

	result := &LoadTestResult{
		TestID:         fmt.Sprintf("loadtest-%d", time.Now().UnixNano()),
		Duration:       config.Duration,
		TargetRPS:      config.TargetRPS,
		MaxConcurrency: config.MaxConcurrency,
		// Environment will be set after execution
	}

	// Execute load test
	start := time.Now()
	err := br.executeLoadTest(ctx, provider, config, result)
	actualDuration := time.Since(start)

	// Calculate actual RPS
	if actualDuration > 0 {
		result.ActualRPS = float64(result.TotalOperations) / actualDuration.Seconds()
	}

	if err != nil {
		return result, fmt.Errorf("load test execution failed: %w", err)
	}

	return result, nil
}

// GetSupportedMetrics returns the list of metrics this runner can collect
func (br *BenchmarkRunner) GetSupportedMetrics() []string {
	return []string{
		"latency",
		"throughput",
		"tokens",
		"errors",
		"memory",
		"concurrency",
		"cost",
		"efficiency",
	}
}

// Private helper methods

func (br *BenchmarkRunner) executeOperations(ctx context.Context, provider iface.ChatModel, prompts []string, config ScenarioConfig, result *BenchmarkResult) error {
	semaphore := make(chan struct{}, config.ConcurrencyLevel)
	var wg sync.WaitGroup
	var mu sync.Mutex

	var successfulOps, failedOps int
	var totalLatencies []time.Duration
	var totalTokens TokenUsage

	// Execute operations
	for i := 0; i < config.OperationCount; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case semaphore <- struct{}{}:
			wg.Add(1)
			go func(opIndex int) {
				defer wg.Done()
				defer func() { <-semaphore }()

				prompt := prompts[opIndex%len(prompts)]

				opStart := time.Now()
				response, err := provider.Invoke(ctx, prompt)
				opDuration := time.Since(opStart)

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					failedOps++
				} else {
					successfulOps++
					totalLatencies = append(totalLatencies, opDuration)

					// Extract token usage if available
					if response != nil {
						// Simulate token extraction - would integrate with actual response
						tokens := br.extractTokenUsage(response)
						totalTokens.InputTokens += tokens.InputTokens
						totalTokens.OutputTokens += tokens.OutputTokens
						totalTokens.TotalTokens += tokens.TotalTokens
					}
				}
			}(i)
		}
	}

	wg.Wait()

	// Update result
	result.SuccessfulOps = successfulOps
	result.FailedOps = failedOps
	result.ErrorRate = float64(failedOps) / float64(config.OperationCount)
	result.TokenUsage = totalTokens
	result.LatencyMetrics = br.calculateLatencyMetrics(totalLatencies)

	return nil
}

func (br *BenchmarkRunner) executeLoadTest(ctx context.Context, provider iface.ChatModel, config LoadTestConfig, result *LoadTestResult) error {
	// Implement load test execution with controlled RPS
	ticker := time.NewTicker(time.Second / time.Duration(config.TargetRPS))
	defer ticker.Stop()

	semaphore := make(chan struct{}, config.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	var successfulOps, failedOps, timeoutOps int
	var throughputPoints []ThroughputPoint
	var errorRatePoints []ErrorRatePoint

	// Start time tracking
	endTime := time.Now().Add(config.Duration)

	// Ramp up phase
	if config.RampUpDuration > 0 {
		// Implement gradual ramp up to target RPS
		time.Sleep(config.RampUpDuration)
	}

	// Main load test execution
	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case <-ticker.C:
			select {
			case semaphore <- struct{}{}:
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-semaphore }()

					// Execute operation
					opStart := time.Now()
					_, err := provider.Invoke(ctx, "Load test prompt")
					opDuration := time.Since(opStart)

					mu.Lock()
					if err != nil {
						if opDuration > 30*time.Second {
							timeoutOps++
						} else {
							failedOps++
						}
					} else {
						successfulOps++
					}

					totalOps := successfulOps + failedOps + timeoutOps

					// Record throughput and error rate periodically
					if totalOps%10 == 0 {
						currentRPS := float64(totalOps) / time.Since(time.Now().Add(-config.Duration)).Seconds()
						throughputPoints = append(throughputPoints, ThroughputPoint{
							Timestamp:   time.Now(),
							RPS:         currentRPS,
							Concurrency: len(semaphore),
							AvgLatency:  opDuration,
						})

						currentErrorRate := float64(failedOps+timeoutOps) / float64(totalOps)
						errorRatePoints = append(errorRatePoints, ErrorRatePoint{
							Timestamp:  time.Now(),
							ErrorRate:  currentErrorRate,
							ErrorCount: failedOps + timeoutOps,
						})
					}
					mu.Unlock()
				}()
			default:
				// Skip this tick if at max concurrency
			}
		}
	}

	wg.Wait()

	// Update load test results
	result.TotalOperations = successfulOps + failedOps + timeoutOps
	result.SuccessfulOps = successfulOps
	result.FailedOps = failedOps
	result.TimeoutOps = timeoutOps
	result.ThroughputCurve = throughputPoints
	result.ErrorRateOverTime = errorRatePoints

	return nil
}

func (br *BenchmarkRunner) calculateLatencyMetrics(latencies []time.Duration) LatencyMetrics {
	if len(latencies) == 0 {
		return LatencyMetrics{}
	}

	// Sort latencies for percentile calculation
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort for small datasets (would use proper sorting for larger)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// Calculate percentiles
	p50Index := len(sorted) * 50 / 100
	p95Index := len(sorted) * 95 / 100
	p99Index := len(sorted) * 99 / 100

	// Calculate mean
	var total time.Duration
	for _, latency := range sorted {
		total += latency
	}
	mean := total / time.Duration(len(sorted))

	return LatencyMetrics{
		P50:  sorted[p50Index],
		P95:  sorted[min(p95Index, len(sorted)-1)],
		P99:  sorted[min(p99Index, len(sorted)-1)],
		Mean: mean,
		Min:  sorted[0],
		Max:  sorted[len(sorted)-1],
	}
}

func (br *BenchmarkRunner) captureEnvironment() Environment {
	return Environment{
		GoVersion:            runtime.Version(),
		OS:                   runtime.GOOS,
		Architecture:         runtime.GOARCH,
		CPUCores:             runtime.NumCPU(),
		MemoryTotalGB:        8,   // Default - would get actual memory in real implementation
		NetworkLatencyMS:     10,  // Default - would measure actual latency
		BandwidthMbps:        100, // Default - would measure actual bandwidth
		SystemLoadAverage:    0.5, // Default - would get actual load
		ConcurrentBenchmarks: 1,
	}
}

func (br *BenchmarkRunner) calculateDerivedMetrics(result *BenchmarkResult) {
	// Calculate throughput
	if result.Duration > 0 {
		result.ThroughputRPS = float64(result.OperationCount) / result.Duration.Seconds()
	}

	// Calculate token efficiency
	if result.TokenUsage.TotalTokens > 0 && result.Duration > 0 {
		result.TokenUsage.TokensPerSecond = float64(result.TokenUsage.TotalTokens) / result.Duration.Seconds()
	}

	if result.TokenUsage.InputTokens > 0 {
		result.TokenUsage.EfficiencyRatio = float64(result.TokenUsage.OutputTokens) / float64(result.TokenUsage.InputTokens)
	}

	// Set memory usage (would integrate with actual memory tracking)
	result.MemoryUsage = MemoryMetrics{
		PeakUsageBytes:     1024 * 1024 * 5,   // 5MB default
		AverageUsageBytes:  1024 * 1024 * 3,   // 3MB default
		BaselineBytes:      1024 * 1024 * 1,   // 1MB default
		MemoryPerOperation: int64(1024 * 500), // 500KB per op
	}
}

func (br *BenchmarkRunner) extractProviderName(provider iface.ChatModel) string {
	// Extract provider name from the provider implementation
	if provider != nil {
		return provider.GetProviderName()
	}
	return "unknown-provider"
}

func (br *BenchmarkRunner) extractModelName(provider iface.ChatModel) string {
	// Extract model name from the provider implementation
	if provider != nil {
		return provider.GetModelName()
	}
	return "unknown-model"
}

func (br *BenchmarkRunner) extractTokenUsage(response interface{}) TokenUsage {
	// Extract token usage from the response
	// This would integrate with the actual response structure
	return TokenUsage{
		InputTokens:  25, // Default for testing
		OutputTokens: 50, // Default for testing
		TotalTokens:  75, // Default for testing
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BenchmarkScenario interface implementation for testing
type StandardScenario struct {
	name        string
	description string
	prompts     []string
	config      ScenarioConfig
}

// NewStandardScenario creates a new standard benchmark scenario
func NewStandardScenario(name string, config StandardScenarioConfig) *StandardScenario {
	return &StandardScenario{
		name:        name,
		description: fmt.Sprintf("Standard scenario: %s", name),
		prompts:     config.TestPrompts,
		config: ScenarioConfig{
			OperationCount:    config.OperationCount,
			ConcurrencyLevel:  config.ConcurrencyLevel,
			TimeoutDuration:   config.TimeoutDuration,
			RequiresTools:     config.RequiresTools,
			RequiresStreaming: config.RequiresStreaming,
		},
	}
}

// BenchmarkScenario interface implementation
func (s *StandardScenario) GetName() string                  { return s.name }
func (s *StandardScenario) GetDescription() string           { return s.description }
func (s *StandardScenario) GetTestPrompts() []string         { return s.prompts }
func (s *StandardScenario) GetConfiguration() ScenarioConfig { return s.config }
func (s *StandardScenario) ValidateProvider(provider iface.ChatModel) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	return nil
}

// BenchmarkScenario interface definition
type BenchmarkScenario interface {
	GetName() string
	GetDescription() string
	GetTestPrompts() []string
	GetConfiguration() ScenarioConfig
	ValidateProvider(provider iface.ChatModel) error
}
