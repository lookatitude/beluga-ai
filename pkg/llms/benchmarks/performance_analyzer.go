package benchmarks

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// PerformanceAnalyzer implements comprehensive analysis of benchmark results
type PerformanceAnalyzer struct {
	options PerformanceAnalyzerOptions
}

// NewPerformanceAnalyzer creates a new performance analyzer with the specified options
func NewPerformanceAnalyzer(options PerformanceAnalyzerOptions) (*PerformanceAnalyzer, error) {
	if options.MinSampleSize == 0 {
		options.MinSampleSize = 3
	}
	if options.ConfidenceLevel == 0 {
		options.ConfidenceLevel = 0.95
	}

	return &PerformanceAnalyzer{
		options: options,
	}, nil
}

// AnalyzeResults processes benchmark results and generates performance insights
func (pa *PerformanceAnalyzer) AnalyzeResults(results []*BenchmarkResult) (*PerformanceAnalysis, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to analyze")
	}

	analysis := &PerformanceAnalysis{
		AnalysisID:      fmt.Sprintf("analysis-%d", time.Now().UnixNano()),
		CreatedAt:       time.Now(),
		ResultsAnalyzed: len(results),
	}

	// Calculate performance scores
	analysis.LatencyScore = pa.calculateLatencyScore(results)
	analysis.ThroughputScore = pa.calculateThroughputScore(results)
	analysis.CostEfficiencyScore = pa.calculateCostEfficiencyScore(results)
	analysis.ReliabilityScore = pa.calculateReliabilityScore(results)
	analysis.OverallScore = pa.calculateOverallScore(analysis)

	// Generate insights and recommendations
	analysis.KeyInsights = pa.generateKeyInsights(results)
	analysis.PerformanceIssues = pa.identifyPerformanceIssues(results)
	analysis.Recommendations = pa.generateRecommendations(results)

	// Calculate statistical summary if enough data
	if len(results) >= pa.options.MinSampleSize {
		analysis.StatisticalSummary = pa.calculateStatisticalSummary(results)
		analysis.LatencyAnalysis = pa.aggregateLatencyMetrics(results)
	}

	return analysis, nil
}

// CompareProviders generates a detailed comparison report between providers
func (pa *PerformanceAnalyzer) CompareProviders(results map[string]*BenchmarkResult) (*ProviderComparison, error) {
	if len(results) < 2 {
		return nil, fmt.Errorf("need at least 2 providers for comparison")
	}

	comparison := &ProviderComparison{
		ComparisonID:     fmt.Sprintf("comparison-%d", time.Now().UnixNano()),
		CreatedAt:        time.Now(),
		ProviderRankings: make(map[string]ProviderRanking),
		WinnerByCategory: make(map[string]string),
	}

	// Calculate performance matrix
	comparison.PerformanceMatrix = pa.buildPerformanceMatrix(results)

	// Rank providers by different criteria
	latencyRankings := pa.rankByLatency(results)
	costRankings := pa.rankByCost(results)
	reliabilityRankings := pa.rankByReliability(results)
	overallRankings := pa.rankOverall(results)

	// Build provider rankings
	for providerName := range results {
		ranking := ProviderRanking{
			Rank:            overallRankings[providerName],
			LatencyRank:     latencyRankings[providerName],
			CostRank:        costRankings[providerName],
			ReliabilityRank: reliabilityRankings[providerName],
			OverallScore:    pa.calculateProviderScore(results[providerName]),
		}
		comparison.ProviderRankings[providerName] = ranking
	}

	// Determine category winners
	comparison.WinnerByCategory["latency"] = pa.findBestByLatency(results)
	comparison.WinnerByCategory["cost"] = pa.findBestByCost(results)
	comparison.WinnerByCategory["reliability"] = pa.findBestByReliability(results)
	comparison.OverallWinner = pa.findOverallWinner(results)

	return comparison, nil
}

// CalculateTrends analyzes performance trends over time
func (pa *PerformanceAnalyzer) CalculateTrends(historicalResults []*BenchmarkResult) (*TrendAnalysis, error) {
	if len(historicalResults) < 3 {
		return nil, fmt.Errorf("need at least 3 data points for trend analysis")
	}

	// Sort by timestamp
	sort.Slice(historicalResults, func(i, j int) bool {
		return historicalResults[i].Timestamp.Before(historicalResults[j].Timestamp)
	})

	trends := &TrendAnalysis{
		TrendID:         fmt.Sprintf("trend-%d", time.Now().UnixNano()),
		DataPoints:      len(historicalResults),
		ConfidenceLevel: pa.options.ConfidenceLevel,
	}

	// Analyze latency trend
	trends.LatencyTrend = pa.analyzeTrend(historicalResults, func(r *BenchmarkResult) float64 {
		return float64(r.LatencyMetrics.Mean.Nanoseconds())
	})

	// Analyze throughput trend
	trends.ThroughputTrend = pa.analyzeTrend(historicalResults, func(r *BenchmarkResult) float64 {
		return r.ThroughputRPS
	})

	// Analyze cost trend
	trends.CostTrend = pa.analyzeTrend(historicalResults, func(r *BenchmarkResult) float64 {
		return r.CostAnalysis.TotalCostUSD
	})

	// Generate summary
	trends.TrendSummary = pa.generateTrendSummary(trends)

	return trends, nil
}

// GenerateOptimizationRecommendations provides actionable recommendations
func (pa *PerformanceAnalyzer) GenerateOptimizationRecommendations(analysis *PerformanceAnalysis) ([]OptimizationRecommendation, error) {
	var recommendations []OptimizationRecommendation

	// Latency optimization recommendations
	if analysis.LatencyScore < 70 {
		recommendations = append(recommendations, OptimizationRecommendation{
			Priority:        "high",
			Category:        "latency",
			Title:           "Improve Response Latency", 
			Description:     "High latency detected in benchmark results",
			Implementation:  "Consider provider optimization, request batching, or caching",
			ExpectedImpact:  "20-40% latency reduction",
			EstimatedEffort: "medium",
		})
	}

	// Throughput optimization recommendations
	if analysis.ThroughputScore < 70 {
		recommendations = append(recommendations, OptimizationRecommendation{
			Priority:        "high",
			Category:        "throughput",
			Title:           "Increase Request Throughput",
			Description:     "Low throughput detected in benchmark results",
			Implementation:  "Increase concurrency, optimize request handling, consider connection pooling",
			ExpectedImpact:  "30-60% throughput increase",
			EstimatedEffort: "medium",
		})
	}

	// Cost optimization recommendations
	if analysis.CostEfficiencyScore < 60 {
		recommendations = append(recommendations, OptimizationRecommendation{
			Priority:        "medium",
			Category:        "cost",
			Title:           "Optimize Token Usage and Cost",
			Description:     "High cost per operation detected",
			Implementation:  "Optimize prompts, consider alternative providers, implement token caching",
			ExpectedImpact:  "15-30% cost reduction",
			EstimatedEffort: "low",
		})
	}

	// Reliability recommendations
	if analysis.ReliabilityScore < 80 {
		recommendations = append(recommendations, OptimizationRecommendation{
			Priority:        "high",
			Category:        "reliability",
			Title:           "Improve Error Handling and Reliability",
			Description:     "High error rate detected in benchmark results",
			Implementation:  "Add retry logic, improve error handling, implement circuit breakers",
			ExpectedImpact:  "50-80% error rate reduction",
			EstimatedEffort: "high",
		})
	}

	return recommendations, nil
}

// Private helper methods

func (pa *PerformanceAnalyzer) calculateLatencyScore(results []*BenchmarkResult) float64 {
	if len(results) == 0 {
		return 0
	}

	var totalLatency time.Duration
	for _, result := range results {
		totalLatency += result.LatencyMetrics.Mean
	}
	avgLatency := totalLatency / time.Duration(len(results))

	// Convert to score (lower latency = higher score)
	// 100ms = 100 points, 1s = 50 points, 2s+ = 0 points
	latencyMs := float64(avgLatency.Milliseconds())
	score := math.Max(0, 100-(latencyMs-100)*0.5)
	return math.Min(100, math.Max(0, score))
}

func (pa *PerformanceAnalyzer) calculateThroughputScore(results []*BenchmarkResult) float64 {
	if len(results) == 0 {
		return 0
	}

	var totalThroughput float64
	for _, result := range results {
		totalThroughput += result.ThroughputRPS
	}
	avgThroughput := totalThroughput / float64(len(results))

	// Convert to score (higher throughput = higher score)
	// 1 RPS = 10 points, 10 RPS = 100 points
	score := avgThroughput * 10
	return math.Min(100, math.Max(0, score))
}

func (pa *PerformanceAnalyzer) calculateCostEfficiencyScore(results []*BenchmarkResult) float64 {
	if len(results) == 0 {
		return 0
	}

	var totalEfficiency float64
	validResults := 0

	for _, result := range results {
		if result.CostAnalysis.CostPerOperation > 0 {
			// Lower cost per operation = higher efficiency
			efficiency := 1.0 / result.CostAnalysis.CostPerOperation * 100
			totalEfficiency += efficiency
			validResults++
		}
	}

	if validResults == 0 {
		return 50 // Default score if no cost data
	}

	avgEfficiency := totalEfficiency / float64(validResults)
	return math.Min(100, math.Max(0, avgEfficiency))
}

func (pa *PerformanceAnalyzer) calculateReliabilityScore(results []*BenchmarkResult) float64 {
	if len(results) == 0 {
		return 0
	}

	var totalErrorRate float64
	for _, result := range results {
		totalErrorRate += result.ErrorRate
	}
	avgErrorRate := totalErrorRate / float64(len(results))

	// Convert error rate to reliability score
	reliabilityScore := (1 - avgErrorRate) * 100
	return math.Min(100, math.Max(0, reliabilityScore))
}

func (pa *PerformanceAnalyzer) calculateOverallScore(analysis *PerformanceAnalysis) float64 {
	// Weighted average of different scores
	weights := map[string]float64{
		"latency":     0.3,
		"throughput":  0.25,
		"cost":        0.2,
		"reliability": 0.25,
	}

	overallScore := analysis.LatencyScore*weights["latency"] +
		analysis.ThroughputScore*weights["throughput"] +
		analysis.CostEfficiencyScore*weights["cost"] +
		analysis.ReliabilityScore*weights["reliability"]

	return math.Min(100, math.Max(0, overallScore))
}

func (pa *PerformanceAnalyzer) generateKeyInsights(results []*BenchmarkResult) []string {
	var insights []string

	// Analyze patterns in results
	if len(results) > 1 {
		latencyVariation := pa.calculateLatencyVariation(results)
		if latencyVariation > 0.3 {
			insights = append(insights, "High latency variation detected - consider optimization")
		}

		avgErrorRate := pa.calculateAverageErrorRate(results)
		if avgErrorRate > 0.05 {
			insights = append(insights, fmt.Sprintf("Error rate %.1f%% above recommended threshold", avgErrorRate*100))
		}

		avgThroughput := pa.calculateAverageThroughput(results)
		if avgThroughput < 5.0 {
			insights = append(insights, "Low throughput detected - consider concurrency optimization")
		}
	}

	if len(insights) == 0 {
		insights = append(insights, "Performance within expected parameters")
	}

	return insights
}

func (pa *PerformanceAnalyzer) identifyPerformanceIssues(results []*BenchmarkResult) []PerformanceIssue {
	var issues []PerformanceIssue

	for _, result := range results {
		// Check for high latency
		if result.LatencyMetrics.P95 > 2*time.Second {
			issues = append(issues, PerformanceIssue{
				Severity:    "high",
				Category:    "latency",
				Description: fmt.Sprintf("High P95 latency: %v", result.LatencyMetrics.P95),
				Impact:      "Degraded user experience",
				Evidence:    []string{fmt.Sprintf("P95 latency %v exceeds 2s threshold", result.LatencyMetrics.P95)},
			})
		}

		// Check for high error rate
		if result.ErrorRate > 0.1 {
			issues = append(issues, PerformanceIssue{
				Severity:    "critical",
				Category:    "reliability",
				Description: fmt.Sprintf("High error rate: %.1f%%", result.ErrorRate*100),
				Impact:      "Service reliability compromised",
				Evidence:    []string{fmt.Sprintf("Error rate %.1f%% exceeds 10%% threshold", result.ErrorRate*100)},
			})
		}

		// Check for memory issues
		if result.MemoryUsage.PeakUsageBytes > 100*1024*1024 { // 100MB
			issues = append(issues, PerformanceIssue{
				Severity:    "medium",
				Category:    "memory",
				Description: fmt.Sprintf("High memory usage: %d MB", result.MemoryUsage.PeakUsageBytes/1024/1024),
				Impact:      "Increased resource costs",
				Evidence:    []string{fmt.Sprintf("Peak memory %d bytes exceeds threshold", result.MemoryUsage.PeakUsageBytes)},
			})
		}
	}

	return issues
}

func (pa *PerformanceAnalyzer) generateRecommendations(results []*BenchmarkResult) []OptimizationRecommendation {
	var recommendations []OptimizationRecommendation

	// Generic recommendations based on common patterns
	avgLatency := pa.calculateAverageLatency(results)
	if avgLatency > 500*time.Millisecond {
		recommendations = append(recommendations, OptimizationRecommendation{
			Priority:        "high",
			Category:        "performance",
			Title:           "Optimize API Request Latency",
			Description:     "Average latency exceeds recommended thresholds",
			Implementation:  "Implement request caching, optimize network calls, consider provider alternatives",
			ExpectedImpact:  "20-40% latency reduction",
			EstimatedEffort: "medium",
		})
	}

	avgThroughput := pa.calculateAverageThroughput(results)
	if avgThroughput < 10.0 {
		recommendations = append(recommendations, OptimizationRecommendation{
			Priority:        "medium",
			Category:        "scaling",
			Title:           "Increase Request Throughput",
			Description:     "Current throughput may not meet high-load requirements",
			Implementation:  "Increase concurrency limits, implement connection pooling, optimize request handling",
			ExpectedImpact:  "50-100% throughput increase",
			EstimatedEffort: "medium",
		})
	}

	return recommendations
}

func (pa *PerformanceAnalyzer) calculateStatisticalSummary(results []*BenchmarkResult) *StatisticalSummary {
	latencies := make([]float64, len(results))
	for i, result := range results {
		latencies[i] = float64(result.LatencyMetrics.Mean.Nanoseconds())
	}

	mean := pa.calculateMean(latencies)
	stdDev := pa.calculateStandardDeviation(latencies, mean)
	
	// Calculate confidence interval
	marginOfError := pa.calculateMarginOfError(stdDev, len(latencies), pa.options.ConfidenceLevel)
	
	return &StatisticalSummary{
		SampleSize:      len(results),
		ConfidenceLevel: pa.options.ConfidenceLevel,
		ConfidenceInterval: ConfidenceInterval{
			Lower: mean - marginOfError,
			Upper: mean + marginOfError,
		},
	}
}

func (pa *PerformanceAnalyzer) aggregateLatencyMetrics(results []*BenchmarkResult) *LatencyMetrics {
	if len(results) == 0 {
		return nil
	}

	var allLatencies []time.Duration
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration

	for i, result := range results {
		latency := result.LatencyMetrics.Mean
		allLatencies = append(allLatencies, latency)
		totalLatency += latency
		
		if i == 0 {
			minLatency = latency
			maxLatency = latency
		} else {
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}
	}

	// Sort for percentile calculation
	sort.Slice(allLatencies, func(i, j int) bool {
		return allLatencies[i] < allLatencies[j]
	})

	meanLatency := totalLatency / time.Duration(len(results))
	
	return &LatencyMetrics{
		P50:  allLatencies[len(allLatencies)*50/100],
		P95:  allLatencies[len(allLatencies)*95/100],
		P99:  allLatencies[min(len(allLatencies)*99/100, len(allLatencies)-1)],
		Mean: meanLatency,
		Min:  minLatency,
		Max:  maxLatency,
	}
}

// Statistical calculation helpers

func (pa *PerformanceAnalyzer) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (pa *PerformanceAnalyzer) calculateStandardDeviation(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	
	variance := sumSquares / float64(len(values)-1)
	return math.Sqrt(variance)
}

func (pa *PerformanceAnalyzer) calculateMarginOfError(stdDev float64, sampleSize int, confidence float64) float64 {
	// Simplified margin of error calculation
	// In practice, would use t-distribution for small samples
	zScore := 1.96 // For 95% confidence
	if confidence < 0.9 {
		zScore = 1.65 // For ~90% confidence
	} else if confidence > 0.98 {
		zScore = 2.33 // For 98% confidence
	}
	
	return zScore * stdDev / math.Sqrt(float64(sampleSize))
}

func (pa *PerformanceAnalyzer) analyzeTrend(results []*BenchmarkResult, extractor func(*BenchmarkResult) float64) string {
	if len(results) < 3 {
		return "insufficient_data"
	}

	values := make([]float64, len(results))
	for i, result := range results {
		values[i] = extractor(result)
	}

	// Simple linear trend detection
	firstThird := pa.calculateMean(values[:len(values)/3])
	lastThird := pa.calculateMean(values[len(values)*2/3:])

	improvementThreshold := 0.05 // 5% improvement threshold
	
	if lastThird > firstThird*(1+improvementThreshold) {
		return "improving"
	} else if lastThird < firstThird*(1-improvementThreshold) {
		return "degrading"
	} else {
		return "stable"
	}
}

func (pa *PerformanceAnalyzer) generateTrendSummary(trends *TrendAnalysis) string {
	improving := 0
	degrading := 0
	stable := 0

	trends_map := map[string]string{
		"latency":     trends.LatencyTrend,
		"throughput":  trends.ThroughputTrend,
		"cost":        trends.CostTrend,
	}

	for _, trend := range trends_map {
		switch trend {
		case "improving":
			improving++
		case "degrading":
			degrading++
		case "stable":
			stable++
		}
	}

	if degrading > improving {
		return fmt.Sprintf("Performance degrading: %d metrics declining, %d improving", degrading, improving)
	} else if improving > degrading {
		return fmt.Sprintf("Performance improving: %d metrics improving, %d declining", improving, degrading)
	} else {
		return fmt.Sprintf("Performance stable: %d metrics stable, %d improving, %d declining", stable, improving, degrading)
	}
}

// Provider comparison helpers

func (pa *PerformanceAnalyzer) buildPerformanceMatrix(results map[string]*BenchmarkResult) *PerformanceMatrix {
	matrix := &PerformanceMatrix{
		LatencyComparison:     make(map[string]float64),
		ThroughputComparison:  make(map[string]float64),
		CostComparison:        make(map[string]float64),
		ReliabilityComparison: make(map[string]float64),
	}

	for providerName, result := range results {
		matrix.LatencyComparison[providerName] = float64(result.LatencyMetrics.Mean.Milliseconds())
		matrix.ThroughputComparison[providerName] = result.ThroughputRPS
		matrix.CostComparison[providerName] = result.CostAnalysis.CostPerOperation
		matrix.ReliabilityComparison[providerName] = (1 - result.ErrorRate) * 100
	}

	return matrix
}

func (pa *PerformanceAnalyzer) rankByLatency(results map[string]*BenchmarkResult) map[string]int {
	// Create slice of providers sorted by latency
	type providerLatency struct {
		name    string
		latency time.Duration
	}

	var providers []providerLatency
	for name, result := range results {
		providers = append(providers, providerLatency{
			name:    name,
			latency: result.LatencyMetrics.Mean,
		})
	}

	// Sort by latency (lower is better)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].latency < providers[j].latency
	})

	rankings := make(map[string]int)
	for i, provider := range providers {
		rankings[provider.name] = i + 1
	}

	return rankings
}

func (pa *PerformanceAnalyzer) rankByCost(results map[string]*BenchmarkResult) map[string]int {
	type providerCost struct {
		name string
		cost float64
	}

	var providers []providerCost
	for name, result := range results {
		providers = append(providers, providerCost{
			name: name,
			cost: result.CostAnalysis.CostPerOperation,
		})
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].cost < providers[j].cost
	})

	rankings := make(map[string]int)
	for i, provider := range providers {
		rankings[provider.name] = i + 1
	}

	return rankings
}

func (pa *PerformanceAnalyzer) rankByReliability(results map[string]*BenchmarkResult) map[string]int {
	type providerReliability struct {
		name       string
		errorRate  float64
	}

	var providers []providerReliability
	for name, result := range results {
		providers = append(providers, providerReliability{
			name:      name,
			errorRate: result.ErrorRate,
		})
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].errorRate < providers[j].errorRate
	})

	rankings := make(map[string]int)
	for i, provider := range providers {
		rankings[provider.name] = i + 1
	}

	return rankings
}

func (pa *PerformanceAnalyzer) rankOverall(results map[string]*BenchmarkResult) map[string]int {
	type providerScore struct {
		name  string
		score float64
	}

	var providers []providerScore
	for name, result := range results {
		score := pa.calculateProviderScore(result)
		providers = append(providers, providerScore{
			name:  name,
			score: score,
		})
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].score > providers[j].score // Higher score is better
	})

	rankings := make(map[string]int)
	for i, provider := range providers {
		rankings[provider.name] = i + 1
	}

	return rankings
}

func (pa *PerformanceAnalyzer) calculateProviderScore(result *BenchmarkResult) float64 {
	// Calculate composite score for a single provider result
	latencyScore := math.Max(0, 100-float64(result.LatencyMetrics.Mean.Milliseconds())/10)
	throughputScore := math.Min(100, result.ThroughputRPS*10)
	reliabilityScore := (1 - result.ErrorRate) * 100
	
	// Weighted average
	return latencyScore*0.4 + throughputScore*0.3 + reliabilityScore*0.3
}

func (pa *PerformanceAnalyzer) findBestByLatency(results map[string]*BenchmarkResult) string {
	var bestProvider string
	var bestLatency time.Duration

	for name, result := range results {
		if bestProvider == "" || result.LatencyMetrics.Mean < bestLatency {
			bestProvider = name
			bestLatency = result.LatencyMetrics.Mean
		}
	}

	return bestProvider
}

func (pa *PerformanceAnalyzer) findBestByCost(results map[string]*BenchmarkResult) string {
	var bestProvider string
	var bestCost float64

	for name, result := range results {
		cost := result.CostAnalysis.CostPerOperation
		if bestProvider == "" || cost < bestCost {
			bestProvider = name
			bestCost = cost
		}
	}

	return bestProvider
}

func (pa *PerformanceAnalyzer) findBestByReliability(results map[string]*BenchmarkResult) string {
	var bestProvider string
	var bestErrorRate float64 = 2.0 // Start with impossible value

	for name, result := range results {
		if bestProvider == "" || result.ErrorRate < bestErrorRate {
			bestProvider = name
			bestErrorRate = result.ErrorRate
		}
	}

	return bestProvider
}

func (pa *PerformanceAnalyzer) findOverallWinner(results map[string]*BenchmarkResult) string {
	var bestProvider string
	var bestScore float64

	for name, result := range results {
		score := pa.calculateProviderScore(result)
		if bestProvider == "" || score > bestScore {
			bestProvider = name
			bestScore = score
		}
	}

	return bestProvider
}

// Calculation helpers

func (pa *PerformanceAnalyzer) calculateLatencyVariation(results []*BenchmarkResult) float64 {
	if len(results) <= 1 {
		return 0
	}

	latencies := make([]float64, len(results))
	for i, result := range results {
		latencies[i] = float64(result.LatencyMetrics.Mean.Milliseconds())
	}

	mean := pa.calculateMean(latencies)
	if mean == 0 {
		return 0
	}

	stdDev := pa.calculateStandardDeviation(latencies, mean)
	return stdDev / mean // Coefficient of variation
}

func (pa *PerformanceAnalyzer) calculateAverageErrorRate(results []*BenchmarkResult) float64 {
	if len(results) == 0 {
		return 0
	}

	var totalErrorRate float64
	for _, result := range results {
		totalErrorRate += result.ErrorRate
	}
	
	return totalErrorRate / float64(len(results))
}

func (pa *PerformanceAnalyzer) calculateAverageThroughput(results []*BenchmarkResult) float64 {
	if len(results) == 0 {
		return 0
	}

	var totalThroughput float64
	for _, result := range results {
		totalThroughput += result.ThroughputRPS
	}
	
	return totalThroughput / float64(len(results))
}

func (pa *PerformanceAnalyzer) calculateAverageLatency(results []*BenchmarkResult) time.Duration {
	if len(results) == 0 {
		return 0
	}

	var totalLatency time.Duration
	for _, result := range results {
		totalLatency += result.LatencyMetrics.Mean
	}

	return totalLatency / time.Duration(len(results))
}
