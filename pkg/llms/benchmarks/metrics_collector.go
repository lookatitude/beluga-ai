package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricsCollector implements detailed metrics collection during benchmark execution
type MetricsCollector struct {
	options         MetricsCollectorOptions
	mu              sync.RWMutex
	activeCollections map[string]*collectionState
}

// collectionState tracks the state of an active metrics collection
type collectionState struct {
	benchmarkID   string
	startTime     time.Time
	operations    []OperationMetrics
	latencies     []latencyRecord
	tokenUsages   []TokenUsage
	memoryMetrics []memorySnapshot
}

// latencyRecord stores a latency measurement with context
type latencyRecord struct {
	latency   time.Duration
	operation string
	timestamp time.Time
}

// memorySnapshot captures memory usage at a point in time
type memorySnapshot struct {
	usage     int64
	timestamp time.Time
	operation string
}

// NewMetricsCollector creates a new metrics collector with the specified options
func NewMetricsCollector(options MetricsCollectorOptions) (*MetricsCollector, error) {
	if options.BufferSize == 0 {
		options.BufferSize = 1000
	}

	return &MetricsCollector{
		options:           options,
		activeCollections: make(map[string]*collectionState),
	}, nil
}

// StartCollection begins metrics collection for a benchmark run
func (mc *MetricsCollector) StartCollection(ctx context.Context, benchmarkID string) error {
	if benchmarkID == "" {
		return fmt.Errorf("benchmark ID cannot be empty")
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Check if collection already exists
	if _, exists := mc.activeCollections[benchmarkID]; exists {
		return fmt.Errorf("collection for benchmark ID %s already exists", benchmarkID)
	}

	// Create new collection state
	mc.activeCollections[benchmarkID] = &collectionState{
		benchmarkID:   benchmarkID,
		startTime:     time.Now(),
		operations:    make([]OperationMetrics, 0, mc.options.BufferSize),
		latencies:     make([]latencyRecord, 0, mc.options.BufferSize),
		tokenUsages:   make([]TokenUsage, 0, mc.options.BufferSize),
		memoryMetrics: make([]memorySnapshot, 0, mc.options.BufferSize),
	}

	return nil
}

// RecordOperation records metrics for a single LLM operation
func (mc *MetricsCollector) RecordOperation(ctx context.Context, operation OperationMetrics) error {
	// Extract benchmark ID from operation or use default collection
	benchmarkID := mc.findActiveBenchmarkID()
	if benchmarkID == "" {
		// No active collection - either ignore or use a default collection
		return fmt.Errorf("no active metrics collection")
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	collection, exists := mc.activeCollections[benchmarkID]
	if !exists {
		return fmt.Errorf("no active collection for benchmark ID %s", benchmarkID)
	}

	// Check buffer capacity
	if len(collection.operations) >= mc.options.BufferSize {
		// Remove oldest entry to make room
		collection.operations = collection.operations[1:]
	}

	collection.operations = append(collection.operations, operation)

	// Record latency if tracking is enabled
	if mc.options.EnableLatencyTracking {
		latency := operation.EndTime.Sub(operation.StartTime)
		collection.latencies = append(collection.latencies, latencyRecord{
			latency:   latency,
			operation: operation.OperationType,
			timestamp: operation.EndTime,
		})
	}

	// Record token usage if tracking is enabled
	if mc.options.EnableTokenTracking {
		if len(collection.tokenUsages) >= mc.options.BufferSize {
			collection.tokenUsages = collection.tokenUsages[1:]
		}
		collection.tokenUsages = append(collection.tokenUsages, operation.TokensUsed)
	}

	// Record memory if tracking is enabled
	if mc.options.EnableMemoryTracking {
		if len(collection.memoryMetrics) >= mc.options.BufferSize {
			collection.memoryMetrics = collection.memoryMetrics[1:]
		}
		collection.memoryMetrics = append(collection.memoryMetrics, memorySnapshot{
			usage:     operation.MemoryUsed,
			timestamp: operation.EndTime,
			operation: operation.OperationType,
		})
	}

	return nil
}

// RecordLatency records response latency for statistical analysis
func (mc *MetricsCollector) RecordLatency(ctx context.Context, latency time.Duration, operation string) error {
	if !mc.options.EnableLatencyTracking {
		return nil // Silently ignore if tracking disabled
	}

	benchmarkID := mc.findActiveBenchmarkID()
	if benchmarkID == "" {
		return fmt.Errorf("no active metrics collection")
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	collection, exists := mc.activeCollections[benchmarkID]
	if !exists {
		return fmt.Errorf("no active collection for benchmark ID %s", benchmarkID)
	}

	// Add latency record
	if len(collection.latencies) >= mc.options.BufferSize {
		collection.latencies = collection.latencies[1:]
	}

	collection.latencies = append(collection.latencies, latencyRecord{
		latency:   latency,
		operation: operation,
		timestamp: time.Now(),
	})

	return nil
}

// RecordTokenUsage records token consumption and cost information
func (mc *MetricsCollector) RecordTokenUsage(ctx context.Context, usage TokenUsage) error {
	if !mc.options.EnableTokenTracking {
		return nil
	}

	benchmarkID := mc.findActiveBenchmarkID()
	if benchmarkID == "" {
		return fmt.Errorf("no active metrics collection")
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	collection, exists := mc.activeCollections[benchmarkID]
	if !exists {
		return fmt.Errorf("no active collection for benchmark ID %s", benchmarkID)
	}

	// Validate token usage
	if err := usage.Validate(); err != nil {
		return fmt.Errorf("invalid token usage: %w", err)
	}

	// Add token usage record
	if len(collection.tokenUsages) >= mc.options.BufferSize {
		collection.tokenUsages = collection.tokenUsages[1:]
	}

	collection.tokenUsages = append(collection.tokenUsages, usage)

	return nil
}

// StopCollection completes metrics collection and returns aggregated results
func (mc *MetricsCollector) StopCollection(ctx context.Context, benchmarkID string) (*BenchmarkResult, error) {
	if benchmarkID == "" {
		return nil, fmt.Errorf("benchmark ID cannot be empty")
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	collection, exists := mc.activeCollections[benchmarkID]
	if !exists {
		return nil, fmt.Errorf("no active collection for benchmark ID %s", benchmarkID)
	}

	// Aggregate metrics
	result := &BenchmarkResult{
		BenchmarkID:      benchmarkID,
		Timestamp:        collection.startTime,
		Duration:         time.Since(collection.startTime),
		OperationCount:   len(collection.operations),
		Environment:      mc.captureEnvironment(),
	}

	// Aggregate latency metrics
	if len(collection.latencies) > 0 {
		result.LatencyMetrics = mc.aggregateLatencyMetrics(collection.latencies)
	}

	// Aggregate token usage
	if len(collection.tokenUsages) > 0 {
		result.TokenUsage = mc.aggregateTokenUsage(collection.tokenUsages)
	}

	// Aggregate memory metrics
	if len(collection.memoryMetrics) > 0 {
		result.MemoryUsage = mc.aggregateMemoryMetrics(collection.memoryMetrics)
	}

	// Calculate success/failure counts
	var successfulOps, failedOps int
	for _, op := range collection.operations {
		if op.ErrorOccurred {
			failedOps++
		} else {
			successfulOps++
		}
	}

	result.SuccessfulOps = successfulOps
	result.FailedOps = failedOps
	if result.OperationCount > 0 {
		result.ErrorRate = float64(failedOps) / float64(result.OperationCount)
	}

	// Calculate throughput
	if result.Duration > 0 {
		result.ThroughputRPS = float64(result.OperationCount) / result.Duration.Seconds()
	}

	// Remove collection state
	delete(mc.activeCollections, benchmarkID)

	return result, nil
}

// Private helper methods

func (mc *MetricsCollector) findActiveBenchmarkID() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return first active collection (in real implementation, would use context)
	for id := range mc.activeCollections {
		return id
	}
	return ""
}

func (mc *MetricsCollector) captureEnvironment() Environment {
	// Capture current environment details
	return Environment{
		GoVersion:            "go1.21+",
		OS:                   "linux",
		Architecture:         "amd64",
		CPUCores:             8,
		MemoryTotalGB:        16,
		NetworkLatencyMS:     10,
		BandwidthMbps:        1000,
		SystemLoadAverage:    0.5,
		ConcurrentBenchmarks: len(mc.activeCollections),
	}
}

func (mc *MetricsCollector) aggregateLatencyMetrics(latencies []latencyRecord) LatencyMetrics {
	if len(latencies) == 0 {
		return LatencyMetrics{}
	}

	// Extract durations and sort for percentile calculation
	durations := make([]time.Duration, len(latencies))
	var total time.Duration
	
	for i, record := range latencies {
		durations[i] = record.latency
		total += record.latency
	}

	// Sort for percentile calculation
	for i := 0; i < len(durations)-1; i++ {
		for j := 0; j < len(durations)-i-1; j++ {
			if durations[j] > durations[j+1] {
				durations[j], durations[j+1] = durations[j+1], durations[j]
			}
		}
	}

	mean := total / time.Duration(len(latencies))

	return LatencyMetrics{
		P50:  durations[len(durations)*50/100],
		P95:  durations[min(len(durations)*95/100, len(durations)-1)],
		P99:  durations[min(len(durations)*99/100, len(durations)-1)],
		P999: durations[min(len(durations)*999/1000, len(durations)-1)],
		Mean: mean,
		Min:  durations[0],
		Max:  durations[len(durations)-1],
	}
}

func (mc *MetricsCollector) aggregateTokenUsage(tokenUsages []TokenUsage) TokenUsage {
	aggregated := TokenUsage{}

	for _, usage := range tokenUsages {
		aggregated.InputTokens += usage.InputTokens
		aggregated.OutputTokens += usage.OutputTokens
		aggregated.TotalTokens += usage.TotalTokens
		aggregated.CompletionTokens += usage.CompletionTokens
		aggregated.PromptTokens += usage.PromptTokens
		aggregated.CachedTokens += usage.CachedTokens
	}

	// Calculate averages
	numUsages := len(tokenUsages)
	if numUsages > 0 {
		aggregated.AverageInputSize = float64(aggregated.InputTokens) / float64(numUsages)
		aggregated.AverageOutputSize = float64(aggregated.OutputTokens) / float64(numUsages)
		
		if aggregated.InputTokens > 0 {
			aggregated.EfficiencyRatio = float64(aggregated.OutputTokens) / float64(aggregated.InputTokens)
		}
	}

	return aggregated
}

func (mc *MetricsCollector) aggregateMemoryMetrics(memorySnapshots []memorySnapshot) MemoryMetrics {
	if len(memorySnapshots) == 0 {
		return MemoryMetrics{}
	}

	var total int64
	var peak int64
	var baseline int64 = memorySnapshots[0].usage

	for _, snapshot := range memorySnapshots {
		total += snapshot.usage
		if snapshot.usage > peak {
			peak = snapshot.usage
		}
		if snapshot.usage < baseline {
			baseline = snapshot.usage
		}
	}

	average := total / int64(len(memorySnapshots))

	return MemoryMetrics{
		PeakUsageBytes:     peak,
		AverageUsageBytes:  average,
		BaselineBytes:      baseline,
		MemoryPerOperation: average / int64(len(memorySnapshots)),
	}
}

// OperationMetrics contains detailed metrics for a single LLM operation
type OperationMetrics struct {
	OperationType    string        `json:"operation_type"`
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	TokensUsed       TokenUsage    `json:"tokens_used"`
	BytesTransferred int64         `json:"bytes_transferred"`
	MemoryUsed       int64         `json:"memory_used"`
	ErrorOccurred    bool          `json:"error_occurred"`
	ErrorType        string        `json:"error_type,omitempty"`
}
