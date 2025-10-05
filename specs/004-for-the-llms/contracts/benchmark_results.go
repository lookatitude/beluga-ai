// Package contracts defines benchmark result data structures and validation contracts
// for the LLMs package performance testing system.
package contracts

import (
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

// PerformanceAnalysis provides comprehensive analysis of benchmark results
// with insights and optimization recommendations.
type PerformanceAnalysis struct {
	// Analysis Metadata
	AnalysisID      string    `json:"analysis_id" validate:"required"`
	CreatedAt       time.Time `json:"created_at" validate:"required"`
	ResultsAnalyzed int       `json:"results_analyzed" validate:"min=1"`

	// Performance Summary
	OverallScore        float64 `json:"overall_score" validate:"min=0,max=100"`
	LatencyScore        float64 `json:"latency_score" validate:"min=0,max=100"`
	ThroughputScore     float64 `json:"throughput_score" validate:"min=0,max=100"`
	CostEfficiencyScore float64 `json:"cost_efficiency_score" validate:"min=0,max=100"`
	ReliabilityScore    float64 `json:"reliability_score" validate:"min=0,max=100"`

	// Performance Insights
	KeyInsights       []string                     `json:"key_insights" validate:"required"`
	PerformanceIssues []PerformanceIssue           `json:"performance_issues" validate:"required"`
	Recommendations   []OptimizationRecommendation `json:"recommendations" validate:"required"`

	// Comparative Analysis
	BenchmarkComparison map[string]float64 `json:"benchmark_comparison,omitempty"`
	IndustryComparison  IndustryComparison `json:"industry_comparison,omitempty"`
}

// PerformanceIssue identifies specific performance problems discovered during analysis.
type PerformanceIssue struct {
	Severity    string   `json:"severity" validate:"required,oneof=low medium high critical"`
	Category    string   `json:"category" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Impact      string   `json:"impact" validate:"required"`
	Evidence    []string `json:"evidence" validate:"required"`
}

// OptimizationRecommendation provides actionable advice for performance improvement.
type OptimizationRecommendation struct {
	Priority        string `json:"priority" validate:"required,oneof=low medium high"`
	Category        string `json:"category" validate:"required"`
	Title           string `json:"title" validate:"required"`
	Description     string `json:"description" validate:"required"`
	Implementation  string `json:"implementation" validate:"required"`
	ExpectedImpact  string `json:"expected_impact" validate:"required"`
	EstimatedEffort string `json:"estimated_effort" validate:"required"`
}

// IndustryComparison provides context for performance results relative to industry standards.
type IndustryComparison struct {
	Percentile     int     `json:"percentile" validate:"min=0,max=100"`
	AboveAverage   bool    `json:"above_average"`
	ComparisonNote string  `json:"comparison_note"`
	IndustryMedian float64 `json:"industry_median" validate:"min=0"`
}

// Validation interfaces for contract enforcement

// BenchmarkResultValidator defines validation rules for benchmark results.
type BenchmarkResultValidator interface {
	ValidateResult(result *BenchmarkResult) error
	ValidateLatencyMetrics(metrics *LatencyMetrics) error
	ValidateTokenUsage(usage *TokenUsage) error
	ValidateEnvironment(env *Environment) error
}

// PerformanceAnalysisValidator defines validation rules for performance analysis.
type PerformanceAnalysisValidator interface {
	ValidateAnalysis(analysis *PerformanceAnalysis) error
	ValidateRecommendations(recommendations []OptimizationRecommendation) error
	ValidateScores(scores map[string]float64) error
}
