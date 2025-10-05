package benchmarks

import (
	"fmt"
	"time"
)

// BenchmarkResult represents the complete outcome of a performance benchmark execution
// with comprehensive metrics and statistical analysis.
type BenchmarkResult struct {
	// Identification
	BenchmarkID  string    `json:"benchmark_id" validate:"required"`
	TestName     string    `json:"test_name" validate:"required"`
	ProviderName string    `json:"provider_name" validate:"required"`
	ModelName    string    `json:"model_name" validate:"required"`
	Timestamp    time.Time `json:"timestamp" validate:"required"`

	// Performance Metrics
	Duration       time.Duration  `json:"duration" validate:"min=0"`
	LatencyMetrics LatencyMetrics `json:"latency_metrics" validate:"required"`
	ThroughputRPS  float64        `json:"throughput_rps" validate:"min=0"`
	ErrorRate      float64        `json:"error_rate" validate:"min=0,max=1"`
	MemoryUsage    MemoryMetrics  `json:"memory_usage" validate:"required"`

	// Token and Cost Analysis
	TokenUsage   TokenUsage   `json:"token_usage" validate:"required"`
	CostAnalysis CostAnalysis `json:"cost_analysis" validate:"required"`

	// Operational Details
	OperationCount   int `json:"operation_count" validate:"min=1"`
	ConcurrencyLevel int `json:"concurrency_level" validate:"min=1"`
	SuccessfulOps    int `json:"successful_ops" validate:"min=0"`
	FailedOps        int `json:"failed_ops" validate:"min=0"`

	// Additional Context
	ScenarioName      string      `json:"scenario_name" validate:"required"`
	ConfigurationHash string      `json:"configuration_hash"`
	Environment       Environment `json:"environment" validate:"required"`

	// Metadata
	AdditionalMetrics map[string]interface{} `json:"additional_metrics,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
}

// LatencyMetrics provides statistical analysis of response times
// with percentile-based performance characteristics.
type LatencyMetrics struct {
	// Percentile Analysis
	P50  time.Duration `json:"p50" validate:"min=0"`
	P95  time.Duration `json:"p95" validate:"min=0"`
	P99  time.Duration `json:"p99" validate:"min=0"`
	P999 time.Duration `json:"p999" validate:"min=0"`

	// Statistical Measures
	Mean              time.Duration `json:"mean" validate:"min=0"`
	Min               time.Duration `json:"min" validate:"min=0"`
	Max               time.Duration `json:"max" validate:"min=0"`
	StandardDeviation time.Duration `json:"standard_deviation" validate:"min=0"`

	// Specialized Metrics
	TimeToFirstToken time.Duration `json:"time_to_first_token" validate:"min=0"`
	TimeToLastToken  time.Duration `json:"time_to_last_token" validate:"min=0"`
	StreamingLatency time.Duration `json:"streaming_latency" validate:"min=0"`
}

// TokenUsage provides detailed breakdown of token consumption
// and efficiency metrics for cost optimization.
type TokenUsage struct {
	// Basic Token Counts
	InputTokens  int `json:"input_tokens" validate:"min=0"`
	OutputTokens int `json:"output_tokens" validate:"min=0"`
	TotalTokens  int `json:"total_tokens" validate:"min=0"`

	// Efficiency Metrics
	EfficiencyRatio float64 `json:"efficiency_ratio" validate:"min=0"`
	TokensPerSecond float64 `json:"tokens_per_second" validate:"min=0"`

	// Provider-Specific Tokens
	CompletionTokens int `json:"completion_tokens,omitempty" validate:"min=0"`
	PromptTokens     int `json:"prompt_tokens,omitempty" validate:"min=0"`
	CachedTokens     int `json:"cached_tokens,omitempty" validate:"min=0"`

	// Token Distribution
	AverageInputSize  float64 `json:"average_input_size" validate:"min=0"`
	AverageOutputSize float64 `json:"average_output_size" validate:"min=0"`
}

// CostAnalysis provides cost calculation and optimization insights
// based on provider pricing models.
type CostAnalysis struct {
	// Direct Costs
	InputCostUSD  float64 `json:"input_cost_usd" validate:"min=0"`
	OutputCostUSD float64 `json:"output_cost_usd" validate:"min=0"`
	TotalCostUSD  float64 `json:"total_cost_usd" validate:"min=0"`

	// Cost Efficiency
	CostPerOperation float64 `json:"cost_per_operation" validate:"min=0"`
	CostPerToken     float64 `json:"cost_per_token" validate:"min=0"`
	CostPerSecond    float64 `json:"cost_per_second" validate:"min=0"`

	// Optimization Metrics
	EstimatedMonthlyCost float64 `json:"estimated_monthly_cost" validate:"min=0"`
	CostEfficiencyScore  float64 `json:"cost_efficiency_score" validate:"min=0,max=100"`
}

// MemoryMetrics tracks memory usage patterns during benchmark execution
// for resource optimization analysis.
type MemoryMetrics struct {
	// Memory Usage
	PeakUsageBytes    int64 `json:"peak_usage_bytes" validate:"min=0"`
	AverageUsageBytes int64 `json:"average_usage_bytes" validate:"min=0"`
	BaselineBytes     int64 `json:"baseline_bytes" validate:"min=0"`

	// Memory Efficiency
	MemoryPerOperation int64 `json:"memory_per_operation" validate:"min=0"`
	MemoryPerToken     int64 `json:"memory_per_token" validate:"min=0"`
	GCCount            int   `json:"gc_count" validate:"min=0"`
	GCPauseTotalNS     int64 `json:"gc_pause_total_ns" validate:"min=0"`
}

// Environment captures the execution environment for benchmark reproducibility
// and result comparison across different conditions.
type Environment struct {
	// Runtime Environment
	GoVersion     string `json:"go_version" validate:"required"`
	OS            string `json:"os" validate:"required"`
	Architecture  string `json:"architecture" validate:"required"`
	CPUCores      int    `json:"cpu_cores" validate:"min=1"`
	MemoryTotalGB int    `json:"memory_total_gb" validate:"min=1"`

	// Network Conditions
	NetworkLatencyMS int `json:"network_latency_ms" validate:"min=0"`
	BandwidthMbps    int `json:"bandwidth_mbps" validate:"min=0"`

	// Load Conditions
	SystemLoadAverage    float64 `json:"system_load_average" validate:"min=0"`
	ConcurrentBenchmarks int     `json:"concurrent_benchmarks" validate:"min=1"`
}

// LoadTestResult represents the outcome of sustained load testing
// with stress testing and performance degradation analysis.
type LoadTestResult struct {
	// Test Configuration
	TestID         string        `json:"test_id" validate:"required"`
	Duration       time.Duration `json:"duration" validate:"min=0"`
	TargetRPS      int           `json:"target_rps" validate:"min=1"`
	ActualRPS      float64       `json:"actual_rps" validate:"min=0"`
	MaxConcurrency int           `json:"max_concurrency" validate:"min=1"`

	// Performance Results
	TotalOperations int `json:"total_operations" validate:"min=0"`
	SuccessfulOps   int `json:"successful_ops" validate:"min=0"`
	FailedOps       int `json:"failed_ops" validate:"min=0"`
	TimeoutOps      int `json:"timeout_ops" validate:"min=0"`

	// Performance Metrics
	LatencyMetrics    LatencyMetrics    `json:"latency_metrics" validate:"required"`
	ThroughputCurve   []ThroughputPoint `json:"throughput_curve" validate:"required"`
	ErrorRateOverTime []ErrorRatePoint  `json:"error_rate_over_time" validate:"required"`

	// Resource Impact
	MemoryMetrics   MemoryMetrics `json:"memory_metrics" validate:"required"`
	CPUUsagePercent float64       `json:"cpu_usage_percent" validate:"min=0,max=100"`

	// Cost Impact
	TotalCostUSD        float64 `json:"total_cost_usd" validate:"min=0"`
	CostPerSuccessfulOp float64 `json:"cost_per_successful_op" validate:"min=0"`
}

// ThroughputPoint represents throughput measurement at a specific point in time
// during load testing for performance curve analysis.
type ThroughputPoint struct {
	Timestamp   time.Time     `json:"timestamp" validate:"required"`
	RPS         float64       `json:"rps" validate:"min=0"`
	Concurrency int           `json:"concurrency" validate:"min=0"`
	AvgLatency  time.Duration `json:"avg_latency" validate:"min=0"`
}

// ErrorRatePoint represents error rate measurement at a specific point in time
// for identifying performance degradation patterns.
type ErrorRatePoint struct {
	Timestamp  time.Time `json:"timestamp" validate:"required"`
	ErrorRate  float64   `json:"error_rate" validate:"min=0,max=1"`
	ErrorCount int       `json:"error_count" validate:"min=0"`
	ErrorTypes []string  `json:"error_types,omitempty"`
}

// PerformanceProfile represents comprehensive performance analysis for a provider/model
type PerformanceProfile struct {
	ProfileID        string             `json:"profile_id" validate:"required"`
	ProviderName     string             `json:"provider_name" validate:"required"`
	ModelName        string             `json:"model_name" validate:"required"`
	CreatedAt        time.Time          `json:"created_at" validate:"required"`
	UpdatedAt        time.Time          `json:"updated_at" validate:"required"`
	BenchmarkResults []*BenchmarkResult `json:"benchmark_results"`
	TrendAnalysis    *TrendAnalysis     `json:"trend_analysis,omitempty"`
}

// TrendAnalysis provides temporal analysis of performance characteristics
type TrendAnalysis struct {
	TrendID         string  `json:"trend_id" validate:"required"`
	DataPoints      int     `json:"data_points" validate:"min=3"`
	ConfidenceLevel float64 `json:"confidence_level" validate:"min=0,max=1"`
	LatencyTrend    string  `json:"latency_trend"` // "improving", "degrading", "stable"
	ThroughputTrend string  `json:"throughput_trend"`
	CostTrend       string  `json:"cost_trend"`
	TrendSummary    string  `json:"trend_summary"`
	TokenUsageTrend string  `json:"token_usage_trend"`
}

// Configuration types for benchmark components

// BenchmarkRunnerOptions configures the benchmark runner behavior
type BenchmarkRunnerOptions struct {
	EnableMetrics  bool          `json:"enable_metrics"`
	MaxConcurrency int           `json:"max_concurrency" validate:"min=1,max=1000"`
	Timeout        time.Duration `json:"timeout" validate:"min=1s"`
}

// PerformanceAnalyzerOptions configures the performance analyzer
type PerformanceAnalyzerOptions struct {
	EnableTrendAnalysis bool    `json:"enable_trend_analysis"`
	MinSampleSize       int     `json:"min_sample_size" validate:"min=1"`
	ConfidenceLevel     float64 `json:"confidence_level" validate:"min=0.5,max=0.99"`
}

// MetricsCollectorOptions configures the metrics collector
type MetricsCollectorOptions struct {
	EnableLatencyTracking bool `json:"enable_latency_tracking"`
	EnableTokenTracking   bool `json:"enable_token_tracking"`
	EnableMemoryTracking  bool `json:"enable_memory_tracking"`
	BufferSize            int  `json:"buffer_size" validate:"min=100"`
}

// ProfileManagerOptions configures the profile manager
type ProfileManagerOptions struct {
	StorageType  string        `json:"storage_type" validate:"oneof=memory file"`
	MaxProfiles  int           `json:"max_profiles" validate:"min=1"`
	ArchiveAfter time.Duration `json:"archive_after" validate:"min=1h"`
	EnableTrends bool          `json:"enable_trends"`
	MinTrendData int           `json:"min_trend_data" validate:"min=1"`
}

// MockConfiguratorOptions configures the mock configurator
type MockConfiguratorOptions struct {
	EnableLatencySimulation bool `json:"enable_latency_simulation"`
	EnableErrorInjection    bool `json:"enable_error_injection"`
	EnableMemorySimulation  bool `json:"enable_memory_simulation"`
}

// LoadTesterOptions configures the load tester
type LoadTesterOptions struct {
	MaxConcurrency   int           `json:"max_concurrency" validate:"min=1,max=1000"`
	DefaultTimeout   time.Duration `json:"default_timeout" validate:"min=1s"`
	EnableMetrics    bool          `json:"enable_metrics"`
	EnableStressMode bool          `json:"enable_stress_mode"`
}

// StreamingAnalyzerOptions configures the streaming analyzer
type StreamingAnalyzerOptions struct {
	EnableTTFTTracking         bool          `json:"enable_ttft_tracking"`
	EnableThroughputTracking   bool          `json:"enable_throughput_tracking"`
	EnableMemoryTracking       bool          `json:"enable_memory_tracking"`
	EnableBackpressureTracking bool          `json:"enable_backpressure_tracking"`
	SampleRate                 float64       `json:"sample_rate" validate:"min=0,max=1"`
	ThroughputWindowSize       time.Duration `json:"throughput_window_size"`
}

// TokenOptimizerOptions configures the token optimizer
type TokenOptimizerOptions struct {
	EnableCostCalculation    bool   `json:"enable_cost_calculation"`
	EnableEfficiencyAnalysis bool   `json:"enable_efficiency_analysis"`
	EnableOptimizationHints  bool   `json:"enable_optimization_hints"`
	EnableTrendAnalysis      bool   `json:"enable_trend_analysis"`
	CostModelAccuracy        string `json:"cost_model_accuracy" validate:"oneof=low medium high"`
	HintGenerationMode       string `json:"hint_generation_mode" validate:"oneof=basic comprehensive"`
}

// Scenario configuration types

// ScenarioConfig contains execution parameters for benchmark scenarios
type ScenarioConfig struct {
	OperationCount    int           `json:"operation_count" validate:"min=1"`
	ConcurrencyLevel  int           `json:"concurrency_level" validate:"min=1,max=100"`
	TimeoutDuration   time.Duration `json:"timeout_duration" validate:"min=1s"`
	RequiresTools     bool          `json:"requires_tools"`
	RequiresStreaming bool          `json:"requires_streaming"`
}

// LoadTestConfig defines parameters for sustained load testing
type LoadTestConfig struct {
	Duration       time.Duration `json:"duration" validate:"min=1s"`
	TargetRPS      int           `json:"target_rps" validate:"min=1"`
	MaxConcurrency int           `json:"max_concurrency" validate:"min=1"`
	RampUpDuration time.Duration `json:"ramp_up_duration" validate:"min=0s"`
	ScenarioName   string        `json:"scenario_name" validate:"required"`
}

// StandardScenarioConfig configures standard benchmark scenarios
type StandardScenarioConfig struct {
	TestPrompts       []string      `json:"test_prompts" validate:"required,min=1"`
	OperationCount    int           `json:"operation_count" validate:"min=1"`
	ConcurrencyLevel  int           `json:"concurrency_level" validate:"min=1"`
	TimeoutDuration   time.Duration `json:"timeout_duration" validate:"min=1s"`
	RequiresTools     bool          `json:"requires_tools"`
	RequiresStreaming bool          `json:"requires_streaming"`
}

// Analysis result types

// PerformanceAnalysis provides comprehensive analysis of benchmark results
type PerformanceAnalysis struct {
	AnalysisID      string    `json:"analysis_id" validate:"required"`
	CreatedAt       time.Time `json:"created_at" validate:"required"`
	ResultsAnalyzed int       `json:"results_analyzed" validate:"min=1"`

	// Performance Summary
	OverallScore        float64 `json:"overall_score" validate:"min=0,max=100"`
	LatencyScore        float64 `json:"latency_score" validate:"min=0,max=100"`
	ThroughputScore     float64 `json:"throughput_score" validate:"min=0,max=100"`
	CostEfficiencyScore float64 `json:"cost_efficiency_score" validate:"min=0,max=100"`
	ReliabilityScore    float64 `json:"reliability_score" validate:"min=0,max=100"`

	// Analysis Components
	LatencyAnalysis    *LatencyMetrics              `json:"latency_analysis,omitempty"`
	StatisticalSummary *StatisticalSummary          `json:"statistical_summary,omitempty"`
	KeyInsights        []string                     `json:"key_insights"`
	PerformanceIssues  []PerformanceIssue           `json:"performance_issues"`
	Recommendations    []OptimizationRecommendation `json:"recommendations"`
}

// StatisticalSummary provides statistical confidence information
type StatisticalSummary struct {
	SampleSize         int                `json:"sample_size" validate:"min=1"`
	ConfidenceLevel    float64            `json:"confidence_level" validate:"min=0.5,max=0.99"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
}

// ConfidenceInterval represents statistical confidence bounds
type ConfidenceInterval struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

// ProviderComparison provides detailed comparison between providers
type ProviderComparison struct {
	ComparisonID      string                     `json:"comparison_id" validate:"required"`
	CreatedAt         time.Time                  `json:"created_at" validate:"required"`
	ProviderRankings  map[string]ProviderRanking `json:"provider_rankings" validate:"required"`
	PerformanceMatrix *PerformanceMatrix         `json:"performance_matrix"`
	WinnerByCategory  map[string]string          `json:"winner_by_category"`
	OverallWinner     string                     `json:"overall_winner"`
}

// ProviderRanking represents a provider's ranking in comparison
type ProviderRanking struct {
	Rank            int     `json:"rank" validate:"min=1"`
	OverallScore    float64 `json:"overall_score" validate:"min=0,max=100"`
	LatencyRank     int     `json:"latency_rank" validate:"min=1"`
	CostRank        int     `json:"cost_rank" validate:"min=1"`
	ReliabilityRank int     `json:"reliability_rank" validate:"min=1"`
}

// PerformanceMatrix provides detailed performance comparison data
type PerformanceMatrix struct {
	LatencyComparison     map[string]float64 `json:"latency_comparison"`
	ThroughputComparison  map[string]float64 `json:"throughput_comparison"`
	CostComparison        map[string]float64 `json:"cost_comparison"`
	ReliabilityComparison map[string]float64 `json:"reliability_comparison"`
}

// Issue and recommendation types

// PerformanceIssue identifies specific performance problems
type PerformanceIssue struct {
	Severity    string   `json:"severity" validate:"required,oneof=low medium high critical"`
	Category    string   `json:"category" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Impact      string   `json:"impact" validate:"required"`
	Evidence    []string `json:"evidence" validate:"required"`
}

// OptimizationRecommendation provides actionable advice for performance improvement
type OptimizationRecommendation struct {
	Priority        string `json:"priority" validate:"required,oneof=low medium high"`
	Category        string `json:"category" validate:"required"`
	Title           string `json:"title" validate:"required"`
	Description     string `json:"description" validate:"required"`
	Implementation  string `json:"implementation" validate:"required"`
	ExpectedImpact  string `json:"expected_impact" validate:"required"`
	EstimatedEffort string `json:"estimated_effort" validate:"required"`
}

// Token optimization specific types

// TokenAnalysisResult represents the outcome of token usage analysis
type TokenAnalysisResult struct {
	AnalysisID        string                  `json:"analysis_id" validate:"required"`
	ProviderName      string                  `json:"provider_name" validate:"required"`
	ModelName         string                  `json:"model_name" validate:"required"`
	PromptText        string                  `json:"prompt_text" validate:"required"`
	Timestamp         time.Time               `json:"timestamp" validate:"required"`
	TokenUsage        TokenUsage              `json:"token_usage" validate:"required"`
	CostAnalysis      CostAnalysis            `json:"cost_analysis" validate:"required"`
	EfficiencyScore   float64                 `json:"efficiency_score" validate:"min=0,max=100"`
	OptimizationHints []TokenOptimizationHint `json:"optimization_hints"`
}

// TokenOptimizationHint provides specific advice for token usage optimization
type TokenOptimizationHint struct {
	Category         string  `json:"category" validate:"required"`
	Description      string  `json:"description" validate:"required"`
	Recommendation   string  `json:"recommendation" validate:"required"`
	Impact           string  `json:"impact" validate:"oneof=low medium high"`
	PotentialSavings float64 `json:"potential_savings" validate:"min=0,max=100"` // Percentage
}

// OptimizationReport provides comprehensive token usage optimization analysis
type OptimizationReport struct {
	ReportID                     string                       `json:"report_id" validate:"required"`
	CreatedAt                    time.Time                    `json:"created_at" validate:"required"`
	Recommendations              []TokenOptimizationHint      `json:"recommendations" validate:"required"`
	ProviderEfficiencyComparison map[string]float64           `json:"provider_efficiency_comparison" validate:"required"`
	CostOptimizationSuggestions  []CostOptimizationSuggestion `json:"cost_optimization_suggestions"`
}

// CostOptimizationSuggestion provides cost reduction recommendations
type CostOptimizationSuggestion struct {
	Type               string   `json:"type" validate:"required"`
	Description        string   `json:"description" validate:"required"`
	EstimatedSavings   float64  `json:"estimated_savings" validate:"min=0"` // USD amount
	ImplementationTips []string `json:"implementation_tips"`
}

// Streaming analysis specific types

// TTFTResult represents Time-To-First-Token analysis results
type TTFTResult struct {
	ProviderName      string        `json:"provider_name" validate:"required"`
	ModelName         string        `json:"model_name" validate:"required"`
	TimeToFirstToken  time.Duration `json:"time_to_first_token" validate:"min=0"`
	TotalStreamTime   time.Duration `json:"total_stream_time" validate:"min=0"`
	FirstTokenLatency time.Duration `json:"first_token_latency" validate:"min=0"`
	Timestamp         time.Time     `json:"timestamp" validate:"required"`
}

// StreamingThroughputResult represents streaming throughput analysis
type StreamingThroughputResult struct {
	ProviderName        string            `json:"provider_name" validate:"required"`
	ModelName           string            `json:"model_name" validate:"required"`
	TokensPerSecond     float64           `json:"tokens_per_second" validate:"min=0"`
	TotalStreamingTime  time.Duration     `json:"total_streaming_time" validate:"min=0"`
	TotalTokensStreamed int               `json:"total_tokens_streamed" validate:"min=0"`
	ThroughputCurve     []ThroughputPoint `json:"throughput_curve"`
	Timestamp           time.Time         `json:"timestamp" validate:"required"`
}

// BackpressureResult represents backpressure handling analysis
type BackpressureResult struct {
	ProviderName       string        `json:"provider_name" validate:"required"`
	ModelName          string        `json:"model_name" validate:"required"`
	BackpressureEvents int           `json:"backpressure_events" validate:"min=0"`
	MaxBufferSize      int64         `json:"max_buffer_size" validate:"min=0"`
	RecoveryTime       time.Duration `json:"recovery_time" validate:"min=0"`
	BufferUtilization  float64       `json:"buffer_utilization" validate:"min=0,max=1"`
	Timestamp          time.Time     `json:"timestamp" validate:"required"`
}

// StreamingMemoryResult represents memory analysis during streaming
type StreamingMemoryResult struct {
	ProviderName          string    `json:"provider_name" validate:"required"`
	ModelName             string    `json:"model_name" validate:"required"`
	PeakMemoryUsage       int64     `json:"peak_memory_usage" validate:"min=0"`
	AverageMemoryUsage    int64     `json:"average_memory_usage" validate:"min=0"`
	MemoryGrowthRate      float64   `json:"memory_growth_rate"`
	MemoryEfficiencyScore float64   `json:"memory_efficiency_score" validate:"min=0,max=100"`
	GCPressure            float64   `json:"gc_pressure" validate:"min=0,max=1"`
	Timestamp             time.Time `json:"timestamp" validate:"required"`
}

// Pricing and cost analysis types

// PricingModel represents provider pricing structure
type PricingModel struct {
	InputTokenCostPer1K  float64   `json:"input_token_cost_per_1k" validate:"min=0"`
	OutputTokenCostPer1K float64   `json:"output_token_cost_per_1k" validate:"min=0"`
	ModelTier            string    `json:"model_tier"`
	EffectiveDate        time.Time `json:"effective_date"`
}

// TokenAnalysisOptions configures token usage analysis
type TokenAnalysisOptions struct {
	MaxTokens            int       `json:"max_tokens" validate:"min=1"`
	CalculateCost        bool      `json:"calculate_cost"`
	GenerateHints        bool      `json:"generate_hints"`
	CompareToBaseline    bool      `json:"compare_to_baseline"`
	HintCategories       []string  `json:"hint_categories,omitempty"`
	ExpectedInputTokens  int       `json:"expected_input_tokens,omitempty"`
	ExpectedOutputTokens int       `json:"expected_output_tokens,omitempty"`
	Timestamp            time.Time `json:"timestamp,omitempty"`
}

// Filtering and query types

// ProfileFilter defines filtering criteria for performance profile queries
type ProfileFilter struct {
	ProviderNames []string  `json:"provider_names,omitempty"`
	ModelNames    []string  `json:"model_names,omitempty"`
	CreatedAfter  time.Time `json:"created_after,omitempty"`
	UpdatedAfter  time.Time `json:"updated_after,omitempty"`
}

// Validation methods

// Validate validates the BenchmarkResult structure
func (br *BenchmarkResult) Validate() error {
	if br.TestName == "" {
		return fmt.Errorf("test name is required")
	}
	if br.ProviderName == "" {
		return fmt.Errorf("provider name is required")
	}
	if br.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	if br.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if br.OperationCount <= 0 {
		return fmt.Errorf("operation count must be positive")
	}
	if br.SuccessfulOps+br.FailedOps != br.OperationCount {
		return fmt.Errorf("successful + failed operations must equal total operations")
	}
	if br.ErrorRate < 0 || br.ErrorRate > 1 {
		return fmt.Errorf("error rate must be between 0 and 1")
	}
	return nil
}

// Validate validates the TokenUsage structure
func (tu *TokenUsage) Validate() error {
	if tu.InputTokens < 0 || tu.OutputTokens < 0 {
		return fmt.Errorf("token counts must be non-negative")
	}
	if tu.TotalTokens != tu.InputTokens+tu.OutputTokens {
		return fmt.Errorf("total tokens must equal input + output tokens")
	}
	if tu.TokensPerSecond < 0 {
		return fmt.Errorf("tokens per second must be non-negative")
	}
	return nil
}

// Validate validates the LoadTestConfig structure
func (ltc *LoadTestConfig) Validate() error {
	if ltc.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if ltc.TargetRPS <= 0 {
		return fmt.Errorf("target RPS must be positive")
	}
	if ltc.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive")
	}
	if ltc.ScenarioName == "" {
		return fmt.Errorf("scenario name is required")
	}
	return nil
}
