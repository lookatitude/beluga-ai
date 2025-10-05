# LLMs Package Benchmark Enhancement - Quickstart Guide

## Overview

This guide demonstrates how to use the enhanced benchmarking capabilities in the LLMs package to perform comprehensive performance analysis, provider comparison, and optimization of LLM operations.

## Prerequisites

- Beluga AI Framework installed and configured
- LLMs package with benchmark enhancements
- Provider API keys (OpenAI, Anthropic, etc.) for real testing
- Go 1.21+ for running benchmarks

## Quick Start Examples

### 1. Basic Provider Benchmarking

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/benchmarks"
)

func main() {
    ctx := context.Background()
    
    // Create provider
    provider, err := llms.NewOpenAIChat(
        llms.WithAPIKey("your-openai-key"),
        llms.WithModelName("gpt-4"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Create benchmark runner
    runner := benchmarks.NewRunner()
    
    // Define a simple benchmark scenario
    scenario := benchmarks.NewSimpleScenario(
        "basic-completion",
        []string{
            "What is the capital of France?",
            "Explain quantum computing in simple terms.",
            "Write a haiku about technology.",
        },
        benchmarks.Config{
            OperationCount:   50,
            ConcurrencyLevel: 5,
            TimeoutDuration:  30 * time.Second,
        },
    )
    
    // Run benchmark
    result, err := runner.RunBenchmark(ctx, provider, scenario)
    if err != nil {
        log.Fatal(err)
    }
    
    // Display results
    fmt.Printf("Benchmark Results for %s:\n", result.ProviderName)
    fmt.Printf("  Average Latency: %v\n", result.LatencyMetrics.Mean)
    fmt.Printf("  P95 Latency: %v\n", result.LatencyMetrics.P95)
    fmt.Printf("  Throughput: %.2f RPS\n", result.ThroughputRPS)
    fmt.Printf("  Error Rate: %.2f%%\n", result.ErrorRate*100)
    fmt.Printf("  Total Cost: $%.4f\n", result.CostAnalysis.TotalCostUSD)
}
```

### 2. Multi-Provider Comparison

```go
func compareProviders() {
    ctx := context.Background()
    
    // Create multiple providers
    providers := map[string]llms.ChatModel{
        "openai-gpt4": createOpenAIProvider(),
        "anthropic-claude": createAnthropicProvider(), 
        "bedrock-claude": createBedrockProvider(),
    }
    
    // Create comparison scenario
    scenario := benchmarks.NewComparisonScenario(
        "provider-comparison",
        []string{
            "Summarize the key benefits of renewable energy.",
            "Write a Python function to sort a list.",
            "Explain the concept of machine learning.",
        },
        benchmarks.Config{
            OperationCount:   100,
            ConcurrencyLevel: 10,
            TimeoutDuration:  60 * time.Second,
        },
    )
    
    // Run comparison benchmark
    runner := benchmarks.NewRunner()
    results, err := runner.RunComparisonBenchmark(ctx, providers, scenario)
    if err != nil {
        log.Fatal(err)
    }
    
    // Analyze and compare results
    analyzer := benchmarks.NewAnalyzer()
    comparison, err := analyzer.CompareProviders(results)
    if err != nil {
        log.Fatal(err)
    }
    
    // Display comparison
    fmt.Println("Provider Comparison Results:")
    for provider, score := range comparison.OverallScores {
        fmt.Printf("  %s: %.2f/100\n", provider, score)
    }
    
    // Show best performer by category
    fmt.Printf("\nBest Performers:\n")
    fmt.Printf("  Fastest: %s (%.2fms avg)\n", 
        comparison.FastestProvider, comparison.BestLatency.Milliseconds())
    fmt.Printf("  Most Cost-Effective: %s ($%.6f per token)\n", 
        comparison.MostCostEffective, comparison.BestCostPerToken)
    fmt.Printf("  Most Reliable: %s (%.2f%% success rate)\n", 
        comparison.MostReliable, comparison.BestSuccessRate*100)
}
```

### 3. Load Testing and Stress Analysis

```go
func runLoadTest() {
    ctx := context.Background()
    
    // Create provider for load testing
    provider := createOpenAIProvider()
    
    // Configure load test
    loadConfig := benchmarks.LoadTestConfig{
        Duration:         5 * time.Minute,
        TargetRPS:        20,
        MaxConcurrency:   50,
        RampUpDuration:   30 * time.Second,
        ScenarioName:     "load-test-scenario",
    }
    
    // Run load test
    runner := benchmarks.NewRunner()
    loadResult, err := runner.RunLoadTest(ctx, provider, loadConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // Analyze load test results
    fmt.Printf("Load Test Results:\n")
    fmt.Printf("  Target RPS: %d, Achieved RPS: %.2f\n", 
        loadResult.TargetRPS, loadResult.ActualRPS)
    fmt.Printf("  Total Operations: %d\n", loadResult.TotalOperations)
    fmt.Printf("  Success Rate: %.2f%%\n", 
        float64(loadResult.SuccessfulOps)/float64(loadResult.TotalOperations)*100)
    fmt.Printf("  P99 Latency: %v\n", loadResult.LatencyMetrics.P99)
    fmt.Printf("  Total Cost: $%.2f\n", loadResult.TotalCostUSD)
    
    // Check for performance degradation
    if loadResult.ActualRPS < float64(loadConfig.TargetRPS)*0.9 {
        fmt.Println("⚠️  Warning: Target RPS not achieved - possible performance bottleneck")
    }
    
    // Analyze error patterns
    for _, errorPoint := range loadResult.ErrorRateOverTime {
        if errorPoint.ErrorRate > 0.05 { // 5% error rate threshold
            fmt.Printf("⚠️  High error rate at %v: %.2f%% (Types: %v)\n", 
                errorPoint.Timestamp, errorPoint.ErrorRate*100, errorPoint.ErrorTypes)
        }
    }
}
```

### 4. Performance Profiling and Analysis

```go
func profilePerformance() {
    ctx := context.Background()
    
    // Create provider
    provider := createAnthropicProvider()
    
    // Create profiling scenario focused on token efficiency
    scenario := benchmarks.NewProfilingScenario(
        "token-efficiency-analysis",
        []string{
            "Write a detailed explanation of blockchain technology.",
            "Create a comprehensive business plan for a tech startup.",
            "Develop a marketing strategy for a new product launch.",
        },
        benchmarks.ProfilingConfig{
            EnableCPUProfiling:    true,
            EnableMemoryProfiling: true,
            EnableNetworkTracing:  true,
            SampleDuration:        2 * time.Minute,
        },
    )
    
    // Run profiling benchmark
    runner := benchmarks.NewProfilingRunner()
    result, profile, err := runner.RunProfilingBenchmark(ctx, provider, scenario)
    if err != nil {
        log.Fatal(err)
    }
    
    // Analyze performance profile
    analyzer := benchmarks.NewProfileAnalyzer()
    analysis, err := analyzer.AnalyzeProfile(profile)
    if err != nil {
        log.Fatal(err)
    }
    
    // Display profiling insights
    fmt.Printf("Performance Profile Analysis:\n")
    fmt.Printf("  CPU Usage: %.2f%% average\n", analysis.CPUUsagePercent)
    fmt.Printf("  Memory Efficiency: %.2f MB/operation\n", 
        float64(result.MemoryMetrics.MemoryPerOperation)/1024/1024)
    fmt.Printf("  Token Efficiency: %.2f tokens/second\n", result.TokenUsage.TokensPerSecond)
    fmt.Printf("  Network Utilization: %.2f Mbps\n", analysis.NetworkUtilization)
    
    // Show optimization recommendations
    recommendations, err := analyzer.GenerateOptimizationRecommendations(analysis)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("\nOptimization Recommendations:")
    for _, rec := range recommendations {
        fmt.Printf("  [%s] %s: %s\n", rec.Priority, rec.Title, rec.Description)
    }
}
```

### 5. Mock Provider Testing for Development

```go
func testWithMockProvider() {
    ctx := context.Background()
    
    // Create enhanced mock provider with realistic behavior
    mockConfig := llms.MockConfiguration{
        SimulatedLatency:       100 * time.Millisecond,
        ErrorInjectionRate:     0.02, // 2% error rate
        TokenGenerationRate:    50,   // 50 tokens/second
        MemoryUsageSimulation:  10 * 1024 * 1024, // 10MB
        ResponseVariability:    0.3,  // 30% response time variation
    }
    
    mockProvider := llms.NewEnhancedMockProvider(mockConfig)
    
    // Run development benchmarks without API costs
    scenario := benchmarks.NewDevelopmentScenario(
        "development-testing",
        []string{"Test prompt for development"},
        benchmarks.Config{
            OperationCount:   1000,
            ConcurrencyLevel: 20,
            TimeoutDuration:  2 * time.Minute,
        },
    )
    
    runner := benchmarks.NewRunner()
    result, err := runner.RunBenchmark(ctx, mockProvider, scenario)
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate development expectations
    fmt.Printf("Development Benchmark Results:\n")
    fmt.Printf("  Simulated Performance: %.2f RPS\n", result.ThroughputRPS)
    fmt.Printf("  Error Handling: %.2f%% error rate\n", result.ErrorRate*100)
    fmt.Printf("  Resource Usage: %.2f MB peak memory\n", 
        float64(result.MemoryMetrics.PeakUsageBytes)/1024/1024)
        
    // Test performance under different mock conditions
    scenarios := []llms.MockConfiguration{
        {SimulatedLatency: 50 * time.Millisecond},   // Fast network
        {SimulatedLatency: 300 * time.Millisecond},  // Slow network
        {ErrorInjectionRate: 0.1},                  // High error rate
    }
    
    for i, config := range scenarios {
        mock := llms.NewEnhancedMockProvider(config)
        result, _ := runner.RunBenchmark(ctx, mock, scenario)
        fmt.Printf("  Scenario %d Performance: %.2f RPS, %.2f%% errors\n", 
            i+1, result.ThroughputRPS, result.ErrorRate*100)
    }
}
```

## Running Benchmarks via Command Line

### Built-in Benchmark Commands

```bash
# Run standard benchmark suite against all configured providers
go test ./pkg/llms/... -bench=BenchmarkProviderComparison -benchtime=60s

# Run specific provider benchmarks
go test ./pkg/llms/... -bench=BenchmarkOpenAI -benchmem

# Run load testing benchmarks
go test ./pkg/llms/... -bench=BenchmarkLoadTest -timeout=10m

# Run with race detection for concurrency validation
go test ./pkg/llms/... -bench=. -race

# Generate CPU profile during benchmarking
go test ./pkg/llms/... -bench=BenchmarkDetailed -cpuprofile=cpu.prof

# Generate memory profile during benchmarking  
go test ./pkg/llms/... -bench=BenchmarkDetailed -memprofile=mem.prof
```

### Custom Benchmark Execution

```bash
# Run custom benchmark scenario from file
go run cmd/benchmark/main.go -scenario=scenarios/production-test.json -providers=openai,anthropic

# Generate benchmark comparison report
go run cmd/benchmark/main.go -compare -output=report.html -duration=5m

# Run continuous performance monitoring
go run cmd/benchmark/main.go -monitor -interval=1h -alert-threshold=20%
```

## Interpreting Results

### Key Metrics to Monitor

1. **Latency Metrics**: P50, P95, P99 response times
2. **Throughput**: Requests per second capacity
3. **Error Rates**: Failure percentages under load
4. **Token Efficiency**: Tokens per second and cost per token
5. **Memory Usage**: Peak memory consumption patterns
6. **Cost Analysis**: Total costs and cost per operation

### Performance Thresholds

```go
// Example performance validation
func validatePerformance(result *BenchmarkResult) []string {
    var issues []string
    
    if result.LatencyMetrics.P95 > 2*time.Second {
        issues = append(issues, "P95 latency exceeds 2 seconds")
    }
    
    if result.ErrorRate > 0.01 { // 1%
        issues = append(issues, "Error rate exceeds 1%")
    }
    
    if result.CostAnalysis.CostPerToken > 0.0001 {
        issues = append(issues, "Cost per token exceeds budget threshold")
    }
    
    if result.MemoryMetrics.PeakUsageBytes > 100*1024*1024 { // 100MB
        issues = append(issues, "Memory usage exceeds 100MB")
    }
    
    return issues
}
```

### Optimization Guidelines

1. **Latency Optimization**: Focus on P95/P99 rather than averages
2. **Cost Optimization**: Monitor token efficiency and prompt optimization
3. **Reliability**: Maintain error rates below 1% under normal load
4. **Scalability**: Ensure performance degrades gracefully under load
5. **Resource Efficiency**: Monitor memory usage and CPU utilization

## Advanced Usage

### Custom Scenario Development

```go
// Implement custom benchmark scenario
type CustomScenario struct {
    name        string
    prompts     []string
    config      ScenarioConfig
    validator   func(response string) bool
}

func (s *CustomScenario) GetName() string { return s.name }
func (s *CustomScenario) GetPrompts() []string { return s.prompts }
func (s *CustomScenario) ValidateProvider(provider ChatModel) error {
    // Custom provider validation logic
    return nil
}
```

### Integration with CI/CD

```yaml
# Example GitHub Actions integration
name: LLM Performance Benchmarks
on:
  pull_request:
    paths: ['pkg/llms/**']

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Run Benchmarks
        run: |
          go test ./pkg/llms/... -bench=. -benchmem > benchmark-results.txt
          go run cmd/benchmark-analyzer/main.go benchmark-results.txt
      - name: Check Performance Regressions
        run: |
          go run cmd/regression-check/main.go \
            --baseline=main-branch-results.json \
            --current=benchmark-results.txt \
            --threshold=10%
```

This quickstart guide provides comprehensive examples for using the enhanced benchmarking capabilities while maintaining the existing LLMs package architecture and framework compliance.
