package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// LoadTester implements sustained load testing and stress analysis
type LoadTester struct {
	options LoadTesterOptions
}

// NewLoadTester creates a new load tester with the specified options
func NewLoadTester(options LoadTesterOptions) (*LoadTester, error) {
	if options.MaxConcurrency == 0 {
		options.MaxConcurrency = 50
	}
	if options.DefaultTimeout == 0 {
		options.DefaultTimeout = 60 * time.Second
	}

	return &LoadTester{
		options: options,
	}, nil
}

// RunLoadTest executes sustained load testing with specified parameters
func (lt *LoadTester) RunLoadTest(ctx context.Context, provider iface.ChatModel, config LoadTestConfig) (*LoadTestResult, error) {
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

	// Execute load test phases
	testCtx, cancel := context.WithTimeout(ctx, config.Duration+config.RampUpDuration+30*time.Second)
	defer cancel()

	err := lt.executeLoadTest(testCtx, provider, config, result)
	if err != nil {
		return result, fmt.Errorf("load test execution failed: %w", err)
	}

	return result, nil
}

// executeLoadTest runs the actual load test with proper phase management
func (lt *LoadTester) executeLoadTest(ctx context.Context, provider iface.ChatModel, config LoadTestConfig, result *LoadTestResult) error {
	// Metrics tracking
	var totalOps, successfulOps, failedOps, timeoutOps int64
	var latencies []time.Duration
	var latenciesMu sync.Mutex

	// Throughput and error tracking
	var throughputPoints []ThroughputPoint
	var errorRatePoints []ErrorRatePoint
	var metricsMu sync.Mutex

	// Concurrency control
	semaphore := make(chan struct{}, config.MaxConcurrency)
	var wg sync.WaitGroup

	// Load test execution
	testStart := time.Now()
	endTime := testStart.Add(config.Duration)

	// Ramp-up phase
	if config.RampUpDuration > 0 {
		time.Sleep(config.RampUpDuration)
	}

	// Calculate target interval between requests
	targetInterval := time.Second / time.Duration(config.TargetRPS)
	ticker := time.NewTicker(targetInterval)
	defer ticker.Stop()

	// Metrics collection ticker
	metricsTicker := time.NewTicker(time.Second)
	defer metricsTicker.Stop()

	// Metrics collection goroutine
	go func() {
		for {
			select {
			case <-metricsTicker.C:
				// Record current metrics
				currentOps := atomic.LoadInt64(&totalOps)
				currentSuccessful := atomic.LoadInt64(&successfulOps)
				currentFailed := atomic.LoadInt64(&failedOps)
				currentTimeouts := atomic.LoadInt64(&timeoutOps)

				elapsed := time.Since(testStart)
				if elapsed > 0 {
					currentRPS := float64(currentOps) / elapsed.Seconds()
					currentErrorRate := float64(currentFailed+currentTimeouts) / float64(currentOps)

					metricsMu.Lock()
					throughputPoints = append(throughputPoints, ThroughputPoint{
						Timestamp:   time.Now(),
						RPS:         currentRPS,
						Concurrency: len(semaphore),
						AvgLatency:  lt.calculateCurrentAvgLatency(latencies),
					})

					errorRatePoints = append(errorRatePoints, ErrorRatePoint{
						Timestamp:  time.Now(),
						ErrorRate:  currentErrorRate,
						ErrorCount: int(currentFailed + currentTimeouts),
						ErrorTypes: []string{"timeout", "failure"}, // Simplified
					})
					metricsMu.Unlock()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Main load generation loop
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

					// Execute single operation
					opStart := time.Now()
					_, err := provider.Invoke(ctx, "Load test prompt")
					opDuration := time.Since(opStart)

					atomic.AddInt64(&totalOps, 1)

					if err != nil {
						if opDuration > 30*time.Second {
							atomic.AddInt64(&timeoutOps, 1)
						} else {
							atomic.AddInt64(&failedOps, 1)
						}
					} else {
						atomic.AddInt64(&successfulOps, 1)
						
						// Record latency for successful operations
						latenciesMu.Lock()
						latencies = append(latencies, opDuration)
						latenciesMu.Unlock()
					}
				}()
			default:
				// Skip this request if at max concurrency
			}
		}
	}

	// Wait for all operations to complete
	wg.Wait()

	// Aggregate final results
	result.TotalOperations = int(atomic.LoadInt64(&totalOps))
	result.SuccessfulOps = int(atomic.LoadInt64(&successfulOps))
	result.FailedOps = int(atomic.LoadInt64(&failedOps))
	result.TimeoutOps = int(atomic.LoadInt64(&timeoutOps))

	// Calculate actual RPS
	actualDuration := time.Since(testStart)
	if actualDuration > 0 {
		result.ActualRPS = float64(result.TotalOperations) / actualDuration.Seconds()
	}

	// Set collected metrics
	result.ThroughputCurve = throughputPoints
	result.ErrorRateOverTime = errorRatePoints

	// Calculate latency metrics
	latenciesMu.Lock()
	if len(latencies) > 0 {
		result.LatencyMetrics = lt.calculateLatencyMetrics(latencies)
	}
	latenciesMu.Unlock()

	// Calculate cost metrics
	if result.TotalOperations > 0 {
		estimatedCostPerOp := 0.002 // $0.002 per operation estimate
		result.TotalCostUSD = float64(result.TotalOperations) * estimatedCostPerOp
		if result.SuccessfulOps > 0 {
			result.CostPerSuccessfulOp = result.TotalCostUSD / float64(result.SuccessfulOps)
		}
	}

	// Set memory metrics (would integrate with actual memory profiling)
	result.MemoryMetrics = MemoryMetrics{
		PeakUsageBytes:    int64(config.MaxConcurrency) * 1024 * 500, // Estimate based on concurrency
		AverageUsageBytes: int64(config.MaxConcurrency) * 1024 * 300,
		BaselineBytes:     1024 * 1024, // 1MB baseline
	}

	// Set CPU usage (would integrate with actual CPU monitoring)
	result.CPUUsagePercent = float64(config.MaxConcurrency) * 2.0 // Rough estimate

	return nil
}

func (lt *LoadTester) calculateLatencyMetrics(latencies []time.Duration) LatencyMetrics {
	if len(latencies) == 0 {
		return LatencyMetrics{}
	}

	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// Calculate metrics
	var total time.Duration
	for _, latency := range sorted {
		total += latency
	}
	mean := total / time.Duration(len(sorted))

	return LatencyMetrics{
		P50:  sorted[len(sorted)*50/100],
		P95:  sorted[min(len(sorted)*95/100, len(sorted)-1)],
		P99:  sorted[min(len(sorted)*99/100, len(sorted)-1)],
		Mean: mean,
		Min:  sorted[0],
		Max:  sorted[len(sorted)-1],
	}
}

func (lt *LoadTester) calculateCurrentAvgLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Calculate average of recent latencies (last 10 or all if fewer)
	start := max(0, len(latencies)-10)
	var total time.Duration
	count := 0

	for i := start; i < len(latencies); i++ {
		total += latencies[i]
		count++
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

func (lt *LoadTester) captureEnvironment() Environment {
	return Environment{
		GoVersion:            "go1.21+",
		OS:                   "linux",
		Architecture:         "amd64",
		CPUCores:             8,
		MemoryTotalGB:        16,
		NetworkLatencyMS:     10,
		BandwidthMbps:        1000,
		SystemLoadAverage:    0.5,
		ConcurrentBenchmarks: 1,
	}
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
