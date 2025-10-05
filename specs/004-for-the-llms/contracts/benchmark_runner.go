// Package contracts defines the API contracts for LLMs package benchmark operations.
// These interfaces enable comprehensive performance testing and provider comparison.
package contracts

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// BenchmarkRunner defines the interface for executing performance benchmarks
// against LLM providers with comprehensive metrics collection.
type BenchmarkRunner interface {
	// RunBenchmark executes a benchmark scenario against the specified provider
	// and returns detailed performance metrics.
	RunBenchmark(ctx context.Context, provider iface.ChatModel, scenario BenchmarkScenario) (*BenchmarkResult, error)

	// RunComparisonBenchmark executes the same scenario across multiple providers
	// for direct performance comparison.
	RunComparisonBenchmark(ctx context.Context, providers map[string]iface.ChatModel, scenario BenchmarkScenario) (map[string]*BenchmarkResult, error)

	// RunLoadTest executes sustained load testing with the specified concurrency
	// and duration parameters.
	RunLoadTest(ctx context.Context, provider iface.ChatModel, config LoadTestConfig) (*LoadTestResult, error)

	// GetSupportedMetrics returns the list of metrics that this runner can collect
	// during benchmark execution.
	GetSupportedMetrics() []string
}

// BenchmarkScenario defines a standardized test scenario for consistent
// performance evaluation across different providers.
type BenchmarkScenario interface {
	// GetName returns the unique identifier for this benchmark scenario.
	GetName() string

	// GetDescription returns a human-readable description of what this scenario tests.
	GetDescription() string

	// GetTestPrompts returns the set of prompts to use for testing.
	// Prompts should be designed to exercise specific capabilities or load patterns.
	GetTestPrompts() []string

	// GetConfiguration returns the execution configuration for this scenario.
	GetConfiguration() ScenarioConfig

	// ValidateProvider verifies that the provider supports all features
	// required by this scenario (e.g., tool calling, streaming).
	ValidateProvider(provider iface.ChatModel) error
}

// PerformanceAnalyzer defines the interface for analyzing benchmark results
// and generating insights for performance optimization.
type PerformanceAnalyzer interface {
	// AnalyzeResults processes benchmark results and generates performance insights.
	AnalyzeResults(results []*BenchmarkResult) (*PerformanceAnalysis, error)

	// CompareProviders generates a detailed comparison report between different providers.
	CompareProviders(results map[string]*BenchmarkResult) (*ProviderComparison, error)

	// CalculateTrends analyzes performance trends over time using historical data.
	CalculateTrends(historicalResults []*BenchmarkResult) (*TrendAnalysis, error)

	// GenerateOptimizationRecommendations provides actionable recommendations
	// for improving performance based on benchmark results.
	GenerateOptimizationRecommendations(analysis *PerformanceAnalysis) ([]OptimizationRecommendation, error)
}

// MetricsCollector defines the interface for collecting detailed metrics
// during benchmark execution.
type MetricsCollector interface {
	// StartCollection begins metrics collection for a benchmark run.
	StartCollection(ctx context.Context, benchmarkID string) error

	// RecordOperation records metrics for a single LLM operation.
	RecordOperation(ctx context.Context, operation OperationMetrics) error

	// RecordLatency records response latency for statistical analysis.
	RecordLatency(ctx context.Context, latency time.Duration, operation string) error

	// RecordTokenUsage records token consumption and cost information.
	RecordTokenUsage(ctx context.Context, usage TokenUsage) error

	// StopCollection completes metrics collection and returns aggregated results.
	StopCollection(ctx context.Context, benchmarkID string) (*BenchmarkResult, error)
}

// ProfileManager defines the interface for managing performance profiles
// and historical benchmark data.
type ProfileManager interface {
	// CreateProfile creates a new performance profile for a provider/model combination.
	CreateProfile(ctx context.Context, providerName, modelName string) (*PerformanceProfile, error)

	// UpdateProfile adds new benchmark results to an existing performance profile.
	UpdateProfile(ctx context.Context, profileID string, result *BenchmarkResult) error

	// GetProfile retrieves a performance profile by provider and model.
	GetProfile(ctx context.Context, providerName, modelName string) (*PerformanceProfile, error)

	// ListProfiles returns all available performance profiles with optional filtering.
	ListProfiles(ctx context.Context, filter ProfileFilter) ([]*PerformanceProfile, error)

	// ArchiveOldResults removes or archives old benchmark results to manage storage.
	ArchiveOldResults(ctx context.Context, olderThan time.Time) error
}

// MockConfigurator defines the interface for configuring realistic mock
// provider behavior during testing.
type MockConfigurator interface {
	// ConfigureLatency sets the simulated network latency for mock responses.
	ConfigureLatency(latency time.Duration, variability float64) error

	// ConfigureErrorInjection sets the error injection rate and types.
	ConfigureErrorInjection(rate float64, errorTypes []string) error

	// ConfigureTokenGeneration sets the simulated token generation characteristics.
	ConfigureTokenGeneration(tokensPerSecond int, variability float64) error

	// ConfigureMemoryUsage sets the simulated memory usage patterns.
	ConfigureMemoryUsage(baseUsage int64, streamingMultiplier float64) error

	// ResetToDefaults resets all mock configuration to default realistic values.
	ResetToDefaults() error
}

// Data structures referenced by interfaces

// ScenarioConfig contains execution parameters for benchmark scenarios.
type ScenarioConfig struct {
	OperationCount    int           // Number of operations to execute
	ConcurrencyLevel  int           // Number of concurrent requests
	TimeoutDuration   time.Duration // Maximum time allowed for scenario
	RequiresTools     bool          // Whether scenario needs tool calling
	RequiresStreaming bool          // Whether scenario tests streaming
}

// LoadTestConfig defines parameters for sustained load testing.
type LoadTestConfig struct {
	Duration       time.Duration // How long to run the load test
	TargetRPS      int           // Target requests per second
	MaxConcurrency int           // Maximum concurrent requests
	RampUpDuration time.Duration // Time to reach target RPS
	ScenarioName   string        // Name of scenario to use for load test
}

// OperationMetrics contains detailed metrics for a single LLM operation.
type OperationMetrics struct {
	OperationType    string     // Type of operation (generate, stream, etc.)
	StartTime        time.Time  // Operation start timestamp
	EndTime          time.Time  // Operation completion timestamp
	TokensUsed       TokenUsage // Token consumption details
	BytesTransferred int64      // Network bytes transferred
	MemoryUsed       int64      // Peak memory usage during operation
	ErrorOccurred    bool       // Whether an error occurred
	ErrorType        string     // Type of error if one occurred
}

// ProfileFilter defines filtering criteria for performance profile queries.
type ProfileFilter struct {
	ProviderNames []string  // Filter by provider names
	ModelNames    []string  // Filter by model names
	CreatedAfter  time.Time // Include profiles created after this time
	UpdatedAfter  time.Time // Include profiles updated after this time
}
