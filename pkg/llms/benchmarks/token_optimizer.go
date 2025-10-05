package benchmarks

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// TokenOptimizer implements token usage analysis and cost optimization
type TokenOptimizer struct {
	options TokenOptimizerOptions
}

// NewTokenOptimizer creates a new token optimizer with the specified options
func NewTokenOptimizer(options TokenOptimizerOptions) (*TokenOptimizer, error) {
	if options.CostModelAccuracy == "" {
		options.CostModelAccuracy = "medium"
	}
	if options.HintGenerationMode == "" {
		options.HintGenerationMode = "basic"
	}

	return &TokenOptimizer{
		options: options,
	}, nil
}

// AnalyzeTokenUsage performs comprehensive token usage analysis
func (to *TokenOptimizer) AnalyzeTokenUsage(ctx context.Context, provider iface.ChatModel, prompt string, options TokenAnalysisOptions) (*TokenAnalysisResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	result := &TokenAnalysisResult{
		AnalysisID:   fmt.Sprintf("token-analysis-%d", time.Now().UnixNano()),
		ProviderName: to.extractProviderName(provider),
		ModelName:    to.extractModelName(provider),
		PromptText:   prompt,
		Timestamp:    time.Now(),
	}

	// Simulate token analysis (would integrate with actual provider response)
	tokenUsage := to.simulateTokenUsage(prompt, options)
	result.TokenUsage = tokenUsage

	// Calculate cost if enabled
	if options.CalculateCost {
		result.CostAnalysis = to.calculateCost(tokenUsage, to.extractProviderName(provider))
	}

	// Calculate efficiency score
	result.EfficiencyScore = to.calculateEfficiencyScore(tokenUsage, len(prompt))

	// Generate optimization hints if enabled
	if options.GenerateHints {
		result.OptimizationHints = to.generateOptimizationHints(prompt, tokenUsage, result.CostAnalysis)
	}

	return result, nil
}

// GenerateOptimizationReport creates comprehensive optimization analysis
func (to *TokenOptimizer) GenerateOptimizationReport(results []*TokenAnalysisResult) (*OptimizationReport, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to analyze")
	}

	report := &OptimizationReport{
		ReportID:                     fmt.Sprintf("optimization-%d", time.Now().UnixNano()),
		CreatedAt:                    time.Now(),
		ProviderEfficiencyComparison: make(map[string]float64),
	}

	// Calculate provider efficiency comparison
	providerStats := make(map[string][]float64)
	for _, result := range results {
		providerStats[result.ProviderName] = append(providerStats[result.ProviderName], result.EfficiencyScore)
	}

	for provider, scores := range providerStats {
		var total float64
		for _, score := range scores {
			total += score
		}
		avgScore := total / float64(len(scores))
		report.ProviderEfficiencyComparison[provider] = avgScore
	}

	// Generate aggregated recommendations
	report.Recommendations = to.aggregateOptimizationHints(results)

	// Generate cost optimization suggestions
	report.CostOptimizationSuggestions = to.generateCostOptimizationSuggestions(results)

	return report, nil
}

// AnalyzeTrends analyzes token usage trends over time
func (to *TokenOptimizer) AnalyzeTrends(historicalResults []*TokenAnalysisResult) (*TrendAnalysis, error) {
	if len(historicalResults) < 3 {
		return nil, fmt.Errorf("need at least 3 data points for trend analysis")
	}

	// Sort by timestamp
	sortedResults := make([]*TokenAnalysisResult, len(historicalResults))
	copy(sortedResults, historicalResults)
	
	for i := 0; i < len(sortedResults)-1; i++ {
		for j := 0; j < len(sortedResults)-i-1; j++ {
			if sortedResults[j].Timestamp.After(sortedResults[j+1].Timestamp) {
				sortedResults[j], sortedResults[j+1] = sortedResults[j+1], sortedResults[j]
			}
		}
	}

	trends := &TrendAnalysis{
		TrendID:         fmt.Sprintf("token-trend-%d", time.Now().UnixNano()),
		DataPoints:      len(historicalResults),
		ConfidenceLevel: 0.85,
	}

	// Analyze token usage trend
	trends.TokenUsageTrend = to.analyzeTokenTrend(sortedResults)

	// Analyze cost trend
	trends.CostTrend = to.analyzeCostTrend(sortedResults)

	// Generate trend summary
	trends.TrendSummary = to.generateTokenTrendSummary(trends)

	return trends, nil
}

// Private helper methods

func (to *TokenOptimizer) simulateTokenUsage(prompt string, options TokenAnalysisOptions) TokenUsage {
	// Simulate token usage based on prompt length and complexity
	promptLength := len(prompt)
	
	// Simple heuristic for token estimation (real implementation would use tokenizer)
	estimatedInputTokens := promptLength / 4 // ~4 characters per token average
	
	// Use expected tokens if provided, otherwise estimate
	inputTokens := estimatedInputTokens
	if options.ExpectedInputTokens > 0 {
		inputTokens = options.ExpectedInputTokens
	}

	outputTokens := inputTokens * 2 // Simulate 2:1 output ratio
	if options.ExpectedOutputTokens > 0 {
		outputTokens = options.ExpectedOutputTokens
	}

	// Apply max tokens limit
	if options.MaxTokens > 0 && outputTokens > options.MaxTokens {
		outputTokens = options.MaxTokens
	}

	totalTokens := inputTokens + outputTokens

	return TokenUsage{
		InputTokens:       inputTokens,
		OutputTokens:      outputTokens,
		TotalTokens:       totalTokens,
		EfficiencyRatio:   float64(outputTokens) / float64(inputTokens),
		TokensPerSecond:   40.0, // Default simulation
		AverageInputSize:  float64(inputTokens),
		AverageOutputSize: float64(outputTokens),
	}
}

func (to *TokenOptimizer) calculateCost(tokenUsage TokenUsage, provider string) CostAnalysis {
	// Pricing models for different providers (simplified)
	pricingModels := map[string]PricingModel{
		"openai": {
			InputTokenCostPer1K:  0.03,
			OutputTokenCostPer1K: 0.06,
		},
		"anthropic": {
			InputTokenCostPer1K:  0.025,
			OutputTokenCostPer1K: 0.05,
		},
		"bedrock": {
			InputTokenCostPer1K:  0.02,
			OutputTokenCostPer1K: 0.04,
		},
	}

	pricing, exists := pricingModels[provider]
	if !exists {
		// Default pricing
		pricing = PricingModel{
			InputTokenCostPer1K:  0.03,
			OutputTokenCostPer1K: 0.06,
		}
	}

	inputCost := float64(tokenUsage.InputTokens) / 1000.0 * pricing.InputTokenCostPer1K
	outputCost := float64(tokenUsage.OutputTokens) / 1000.0 * pricing.OutputTokenCostPer1K
	totalCost := inputCost + outputCost

	return CostAnalysis{
		InputCostUSD:         inputCost,
		OutputCostUSD:        outputCost,
		TotalCostUSD:         totalCost,
		CostPerOperation:     totalCost,
		CostPerToken:         totalCost / float64(tokenUsage.TotalTokens),
		EstimatedMonthlyCost: totalCost * 30 * 24 * 10, // Simulate 10 ops/hour
		CostEfficiencyScore:  to.calculateCostEfficiency(totalCost, tokenUsage.TotalTokens),
	}
}

func (to *TokenOptimizer) calculateEfficiencyScore(tokenUsage TokenUsage, promptLength int) float64 {
	// Calculate efficiency based on output/input ratio and prompt utilization
	if tokenUsage.InputTokens == 0 {
		return 0
	}

	outputRatio := float64(tokenUsage.OutputTokens) / float64(tokenUsage.InputTokens)
	promptUtilization := float64(tokenUsage.InputTokens) / float64(promptLength/4) // ~4 chars per token

	// Combine factors into efficiency score
	efficiency := (outputRatio * 0.6 + promptUtilization * 0.4) * 100
	return math.Min(100, math.Max(0, efficiency))
}

func (to *TokenOptimizer) calculateCostEfficiency(totalCost float64, totalTokens int) float64 {
	if totalCost <= 0 || totalTokens <= 0 {
		return 0
	}

	// Lower cost per token = higher efficiency
	costPerToken := totalCost / float64(totalTokens)
	
	// Normalize to 0-100 scale (assuming $0.0001 per token is baseline)
	baselineCostPerToken := 0.0001
	efficiency := (1 - (costPerToken / baselineCostPerToken)) * 100
	
	return math.Min(100, math.Max(0, efficiency))
}

func (to *TokenOptimizer) generateOptimizationHints(prompt string, tokenUsage TokenUsage, costAnalysis CostAnalysis) []TokenOptimizationHint {
	var hints []TokenOptimizationHint

	// Prompt length optimization
	if len(prompt) > 500 {
		hints = append(hints, TokenOptimizationHint{
			Category:         "prompt_optimization",
			Description:      "Prompt is quite long and could potentially be shortened",
			Recommendation:   "Review prompt for redundant phrases and unnecessary details",
			Impact:           "medium",
			PotentialSavings: 15.0,
		})
	}

	// Output length optimization
	if tokenUsage.OutputTokens > tokenUsage.InputTokens*3 {
		hints = append(hints, TokenOptimizationHint{
			Category:         "output_length_control",
			Description:      "Output is much longer than input, consider length constraints",
			Recommendation:   "Add specific length requirements to prompt or use max_tokens parameter",
			Impact:           "high",
			PotentialSavings: 25.0,
		})
	}

	// Cost optimization
	if costAnalysis.TotalCostUSD > 0.01 { // If cost > 1 cent
		hints = append(hints, TokenOptimizationHint{
			Category:         "cost_optimization",
			Description:      "Operation cost is above typical threshold",
			Recommendation:   "Consider prompt optimization, alternative providers, or caching",
			Impact:           "high",
			PotentialSavings: 30.0,
		})
	}

	// Efficiency optimization
	if tokenUsage.EfficiencyRatio < 1.0 {
		hints = append(hints, TokenOptimizationHint{
			Category:         "efficiency_optimization",
			Description:      "Output is shorter than input, which may indicate inefficient prompt design",
			Recommendation:   "Review prompt clarity and specificity to encourage more detailed responses",
			Impact:           "medium",
			PotentialSavings: 10.0,
		})
	}

	return hints
}

func (to *TokenOptimizer) aggregateOptimizationHints(results []*TokenAnalysisResult) []TokenOptimizationHint {
	hintCounts := make(map[string]int)
	hintExamples := make(map[string]TokenOptimizationHint)

	// Count hint frequency and collect examples
	for _, result := range results {
		for _, hint := range result.OptimizationHints {
			hintCounts[hint.Category]++
			hintExamples[hint.Category] = hint
		}
	}

	var aggregatedHints []TokenOptimizationHint

	// Create aggregated hints for frequent issues
	for category, count := range hintCounts {
		if count > len(results)/3 { // If more than 1/3 of results have this hint
			example := hintExamples[category]
			aggregatedHint := TokenOptimizationHint{
				Category:         category,
				Description:      fmt.Sprintf("Frequent issue: %s (appears in %d/%d analyses)", example.Description, count, len(results)),
				Recommendation:   example.Recommendation,
				Impact:           example.Impact,
				PotentialSavings: example.PotentialSavings,
			}
			aggregatedHints = append(aggregatedHints, aggregatedHint)
		}
	}

	return aggregatedHints
}

func (to *TokenOptimizer) generateCostOptimizationSuggestions(results []*TokenAnalysisResult) []CostOptimizationSuggestion {
	var suggestions []CostOptimizationSuggestion

	// Analyze cost patterns
	totalCost := 0.0
	highCostOperations := 0

	for _, result := range results {
		totalCost += result.CostAnalysis.TotalCostUSD
		if result.CostAnalysis.TotalCostUSD > 0.005 { // > 0.5 cent
			highCostOperations++
		}
	}

	avgCost := totalCost / float64(len(results))

	// Generate suggestions based on patterns
	if highCostOperations > len(results)/4 {
		suggestions = append(suggestions, CostOptimizationSuggestion{
			Type:             "provider_selection",
			Description:      fmt.Sprintf("%.1f%% of operations are high-cost", float64(highCostOperations)/float64(len(results))*100),
			EstimatedSavings: avgCost * 0.3,
			ImplementationTips: []string{
				"Consider switching to more cost-effective providers for routine operations",
				"Use premium providers only for complex tasks requiring highest quality",
				"Implement provider selection based on task complexity",
			},
		})
	}

	if avgCost > 0.003 { // Average cost > 0.3 cent
		suggestions = append(suggestions, CostOptimizationSuggestion{
			Type:             "prompt_optimization",
			Description:      fmt.Sprintf("Average cost per operation ($%.4f) is above recommended threshold", avgCost),
			EstimatedSavings: avgCost * 0.2,
			ImplementationTips: []string{
				"Optimize prompts for conciseness while maintaining clarity",
				"Use system messages to reduce repetitive context in prompts",
				"Implement prompt templates to standardize efficient patterns",
			},
		})
	}

	// Token caching suggestion
	suggestions = append(suggestions, CostOptimizationSuggestion{
		Type:             "caching",
		Description:      "Implement response caching for frequently repeated queries",
		EstimatedSavings: avgCost * 0.4,
		ImplementationTips: []string{
			"Cache responses for identical or similar prompts",
			"Implement semantic similarity caching for related queries",
			"Use TTL-based caching to balance freshness and cost savings",
		},
	})

	return suggestions
}

// AnalyzeTrends analyzes token usage trends over time
func (to *TokenOptimizer) AnalyzeTrends(historicalResults []*TokenAnalysisResult) (*TrendAnalysis, error) {
	if len(historicalResults) < 3 {
		return nil, fmt.Errorf("need at least 3 data points for trend analysis")
	}

	// Sort by timestamp
	sortedResults := make([]*TokenAnalysisResult, len(historicalResults))
	copy(sortedResults, historicalResults)
	
	for i := 0; i < len(sortedResults)-1; i++ {
		for j := 0; j < len(sortedResults)-i-1; j++ {
			if sortedResults[j].Timestamp.After(sortedResults[j+1].Timestamp) {
				sortedResults[j], sortedResults[j+1] = sortedResults[j+1], sortedResults[j]
			}
		}
	}

	trends := &TrendAnalysis{
		TrendID:         fmt.Sprintf("token-trend-%d", time.Now().UnixNano()),
		DataPoints:      len(historicalResults),
		ConfidenceLevel: 0.8,
	}

	// Analyze token usage trend
	trends.TokenUsageTrend = to.analyzeTokenUsageTrendHelper(sortedResults)

	// Analyze cost trend
	trends.CostTrend = to.analyzeCostTrend(sortedResults)

	// Generate summary
	trends.TrendSummary = to.generateTokenTrendSummary(trends)

	return trends, nil
}

// Helper methods

func (to *TokenOptimizer) extractProviderName(provider iface.ChatModel) string {
	return "token-provider" // Default for testing
}

func (to *TokenOptimizer) extractModelName(provider iface.ChatModel) string {
	return "token-model" // Default for testing
}

func (to *TokenOptimizer) analyzeTokenUsageTrendHelper(results []*TokenAnalysisResult) string {
	if len(results) < 3 {
		return "insufficient_data"
	}

	// Calculate average token usage for first and last thirds
	firstThirdEnd := len(results) / 3
	lastThirdStart := len(results) * 2 / 3

	var firstThirdAvg, lastThirdAvg float64
	
	for i := 0; i < firstThirdEnd; i++ {
		firstThirdAvg += float64(results[i].TokenUsage.TotalTokens)
	}
	firstThirdAvg /= float64(firstThirdEnd)

	for i := lastThirdStart; i < len(results); i++ {
		lastThirdAvg += float64(results[i].TokenUsage.TotalTokens)
	}
	lastThirdAvg /= float64(len(results) - lastThirdStart)

	// Determine trend direction
	threshold := 0.1 // 10% change threshold
	
	if lastThirdAvg > firstThirdAvg*(1+threshold) {
		return "increasing"
	} else if lastThirdAvg < firstThirdAvg*(1-threshold) {
		return "decreasing"
	} else {
		return "stable"
	}
}

func (to *TokenOptimizer) analyzeCostTrend(results []*TokenAnalysisResult) string {
	if len(results) < 3 {
		return "insufficient_data"
	}

	// Similar trend analysis for cost
	firstThirdEnd := len(results) / 3
	lastThirdStart := len(results) * 2 / 3

	var firstThirdAvg, lastThirdAvg float64
	
	for i := 0; i < firstThirdEnd; i++ {
		firstThirdAvg += results[i].CostAnalysis.TotalCostUSD
	}
	firstThirdAvg /= float64(firstThirdEnd)

	for i := lastThirdStart; i < len(results); i++ {
		lastThirdAvg += results[i].CostAnalysis.TotalCostUSD
	}
	lastThirdAvg /= float64(len(results) - lastThirdStart)

	threshold := 0.1 // 10% change threshold
	
	if lastThirdAvg > firstThirdAvg*(1+threshold) {
		return "increasing"
	} else if lastThirdAvg < firstThirdAvg*(1-threshold) {
		return "decreasing"  
	} else {
		return "stable"
	}
}

func (to *TokenOptimizer) generateTokenTrendSummary(trends *TrendAnalysis) string {
	tokenTrend := trends.TokenUsageTrend
	costTrend := trends.CostTrend

	if tokenTrend == "increasing" && costTrend == "increasing" {
		return "Token usage and costs are both increasing - optimization recommended"
	} else if tokenTrend == "decreasing" && costTrend == "decreasing" {
		return "Token usage and costs are both decreasing - optimization efforts showing results"
	} else if tokenTrend == "stable" && costTrend == "stable" {
		return "Token usage and costs are stable - consistent usage patterns"
	} else {
		return fmt.Sprintf("Mixed trends: token usage %s, cost %s", tokenTrend, costTrend)
	}
}
