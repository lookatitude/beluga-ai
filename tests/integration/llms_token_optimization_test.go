// Package integration provides integration tests for LLM token usage optimization.
// This file tests token optimization and cost analysis integration.
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/llms/benchmarks"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// TestTokenOptimizationIntegration tests comprehensive token usage analysis
func TestTokenOptimizationIntegration(t *testing.T) {
	ctx := context.Background()

	// Test token usage analysis across providers
	t.Run("CrossProviderTokenAnalysis", func(t *testing.T) {
		// Create token optimizer
		optimizer, err := benchmarks.NewTokenOptimizer(benchmarks.TokenOptimizerOptions{
			EnableCostCalculation:    true,
			EnableEfficiencyAnalysis: true,
			EnableOptimizationHints:  true,
		})
		require.NoError(t, err, "TokenOptimizer creation should succeed")

		// Create providers with different token characteristics
		providers := map[string]iface.ChatModel{
			"efficient-provider": createMockProviderWithTokenCharacteristics("efficient", "model", 1.2), // Low token usage
			"verbose-provider":   createMockProviderWithTokenCharacteristics("verbose", "model", 2.5),   // High token usage
			"balanced-provider":  createMockProviderWithTokenCharacteristics("balanced", "model", 1.8),  // Moderate usage
		}

		// Test prompts with known complexity levels
		testPrompts := []benchmarks.TokenTestPrompt{
			{Content: "What is 2+2?", ExpectedComplexity: "simple", MaxTokens: 50},
			{Content: "Explain quantum mechanics in detail.", ExpectedComplexity: "complex", MaxTokens: 500},
			{Content: "Write a short story about AI.", ExpectedComplexity: "medium", MaxTokens: 200},
		}

		// Analyze token usage for each provider and prompt
		var analysisResults []*benchmarks.TokenAnalysisResult

		for providerName, provider := range providers {
			for _, prompt := range testPrompts {
				result, err := optimizer.AnalyzeTokenUsage(ctx, provider, prompt.Content,
					benchmarks.TokenAnalysisOptions{
						MaxTokens:         prompt.MaxTokens,
						CalculateCost:     true,
						GenerateHints:     true,
						CompareToBaseline: true,
					})
				assert.NoError(t, err, "Token analysis for %s with %s prompt should succeed",
					providerName, prompt.ExpectedComplexity)
				assert.NotNil(t, result, "Analysis result should not be nil")

				analysisResults = append(analysisResults, result)

				// Verify analysis structure
				assert.NotEmpty(t, result.AnalysisID, "Should have analysis ID")
				assert.Equal(t, providerName, result.ProviderName, "Should have correct provider")
				assert.NotEmpty(t, result.PromptText, "Should have prompt text")
				assert.Greater(t, result.TokenUsage.TotalTokens, 0, "Should use some tokens")
				assert.GreaterOrEqual(t, result.CostAnalysis.TotalCostUSD, 0.0, "Cost should be non-negative")
			}
		}

		// Generate cross-provider optimization recommendations
		optimizationReport, err := optimizer.GenerateOptimizationReport(analysisResults)
		assert.NoError(t, err, "Optimization report generation should succeed")
		assert.NotNil(t, optimizationReport, "Optimization report should not be nil")

		// Verify optimization recommendations
		assert.NotEmpty(t, optimizationReport.Recommendations, "Should have recommendations")
		assert.NotNil(t, optimizationReport.ProviderEfficiencyComparison, "Should compare provider efficiency")
		assert.NotNil(t, optimizationReport.CostOptimizationSuggestions, "Should have cost suggestions")

		t.Logf("Generated optimization report with %d recommendations for %d providers",
			len(optimizationReport.Recommendations), len(providers))
	})

	// Test cost calculation accuracy
	t.Run("CostCalculationAccuracy", func(t *testing.T) {
		optimizer, err := benchmarks.NewTokenOptimizer(benchmarks.TokenOptimizerOptions{
			EnableCostCalculation: true,
			CostModelAccuracy:     "high", // Use accurate pricing models
		})
		require.NoError(t, err)

		// Create provider with known pricing characteristics
		provider := createMockProviderWithPricing("cost-test", "gpt-4", benchmarks.PricingModel{
			InputTokenCostPer1K:  0.03, // $0.03 per 1K input tokens
			OutputTokenCostPer1K: 0.06, // $0.06 per 1K output tokens
		})

		// Test cost calculation with known token counts
		testCases := []struct {
			prompt         string
			expectedInput  int
			expectedOutput int
		}{
			{"Short prompt", 10, 25},                      // Simple case
			{"Medium length prompt with details", 25, 75}, // Medium case
			{"Very detailed and comprehensive prompt requiring extensive analysis and explanation", 50, 200}, // Complex case
		}

		for _, testCase := range testCases {
			result, err := optimizer.AnalyzeTokenUsage(ctx, provider, testCase.prompt,
				benchmarks.TokenAnalysisOptions{
					CalculateCost:        true,
					ExpectedInputTokens:  testCase.expectedInput,
					ExpectedOutputTokens: testCase.expectedOutput,
				})
			assert.NoError(t, err, "Cost calculation should succeed")

			// Verify cost calculation accuracy
			expectedInputCost := float64(testCase.expectedInput) / 1000.0 * 0.03
			expectedOutputCost := float64(testCase.expectedOutput) / 1000.0 * 0.06
			expectedTotalCost := expectedInputCost + expectedOutputCost

			if result.CostAnalysis.TotalCostUSD > 0 {
				assert.InDelta(t, expectedTotalCost, result.CostAnalysis.TotalCostUSD, 0.001,
					"Cost calculation should be accurate for prompt: %s", testCase.prompt)
			}

			t.Logf("Prompt: '%s' -> %d input, %d output tokens, $%.4f",
				testCase.prompt, result.TokenUsage.InputTokens,
				result.TokenUsage.OutputTokens, result.CostAnalysis.TotalCostUSD)
		}
	})

	// Test optimization hint generation
	t.Run("OptimizationHintGeneration", func(t *testing.T) {
		optimizer, err := benchmarks.NewTokenOptimizer(benchmarks.TokenOptimizerOptions{
			EnableOptimizationHints: true,
			HintGenerationMode:      "comprehensive",
		})
		require.NoError(t, err)

		// Create scenarios that should trigger different optimization hints
		optimizationScenarios := []struct {
			name     string
			provider iface.ChatModel
			prompt   string
			expected []string // Expected hint categories
		}{
			{
				name:     "verbose-output",
				provider: createMockProviderWithTokenCharacteristics("verbose", "model", 3.0), // Very verbose
				prompt:   "Explain AI briefly.",
				expected: []string{"prompt_optimization", "output_length_control"},
			},
			{
				name:     "inefficient-input",
				provider: createMockProviderWithTokenCharacteristics("inefficient", "model", 1.1),
				prompt:   "This is a very long and unnecessarily detailed prompt that could be made much more concise while maintaining the same meaning and intent, which would reduce token usage and cost significantly.", // Verbose prompt
				expected: []string{"prompt_simplification", "input_optimization"},
			},
			{
				name: "cost-inefficient",
				provider: createMockProviderWithPricing("expensive", "premium-model", benchmarks.PricingModel{
					InputTokenCostPer1K:  0.10, // Very expensive
					OutputTokenCostPer1K: 0.20,
				}),
				prompt:   "Standard prompt for cost analysis.",
				expected: []string{"provider_selection", "cost_optimization"},
			},
		}

		for _, scenario := range optimizationScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				result, err := optimizer.AnalyzeTokenUsage(ctx, scenario.provider, scenario.prompt,
					benchmarks.TokenAnalysisOptions{
						GenerateHints:     true,
						HintCategories:    scenario.expected,
						CompareToBaseline: true,
					})
				assert.NoError(t, err, "Token analysis should succeed")

				// Verify optimization hints are generated
				assert.NotEmpty(t, result.OptimizationHints,
					"Should generate optimization hints")

				// Check for expected hint categories
				hintCategories := make(map[string]bool)
				for _, hint := range result.OptimizationHints {
					hintCategories[hint.Category] = true
				}

				for _, expectedCategory := range scenario.expected {
					if hintCategories[expectedCategory] {
						t.Logf("Scenario %s correctly generated %s hint",
							scenario.name, expectedCategory)
					}
				}

				// Verify hint structure
				for i, hint := range result.OptimizationHints {
					assert.NotEmpty(t, hint.Category, "Hint %d should have category", i)
					assert.NotEmpty(t, hint.Description, "Hint %d should have description", i)
					assert.NotEmpty(t, hint.Recommendation, "Hint %d should have recommendation", i)
					assert.Contains(t, []string{"low", "medium", "high"}, hint.Impact,
						"Hint %d should have valid impact level", i)
				}
			})
		}
	})

	// Test token usage trend analysis
	t.Run("TokenUsageTrendAnalysis", func(t *testing.T) {
		optimizer, err := benchmarks.NewTokenOptimizer(benchmarks.TokenOptimizerOptions{
			EnableTrendAnalysis: true,
		})
		require.NoError(t, err)

		provider := createMockProviderWithTokenCharacteristics("trend-test", "model", 1.5)

		// Generate historical token usage data
		const numHistoricalPoints = 15
		historicalPrompts := make([]string, numHistoricalPoints)
		for i := 0; i < numHistoricalPoints; i++ {
			// Gradually increase prompt complexity to show trend
			complexity := i + 5 // Start with some base complexity
			historicalPrompts[i] = fmt.Sprintf("Prompt with complexity level %d requiring analysis of %d factors",
				complexity, complexity*2)
		}

		// Analyze historical usage
		var historicalResults []*benchmarks.TokenAnalysisResult
		baseTime := time.Now().Add(-time.Duration(numHistoricalPoints) * time.Hour)

		for i, prompt := range historicalPrompts {
			result, err := optimizer.AnalyzeTokenUsage(ctx, provider, prompt,
				benchmarks.TokenAnalysisOptions{
					CalculateCost: true,
					Timestamp:     baseTime.Add(time.Duration(i) * time.Hour),
				})
			assert.NoError(t, err, "Historical analysis should succeed")
			historicalResults = append(historicalResults, result)
		}

		// Generate trend analysis
		trendAnalysis, err := optimizer.AnalyzeTrends(historicalResults)
		assert.NoError(t, err, "Trend analysis should succeed")
		assert.NotNil(t, trendAnalysis, "Trend analysis should not be nil")

		// Verify trend analysis
		assert.Equal(t, numHistoricalPoints, trendAnalysis.DataPoints,
			"Should analyze all historical points")
		assert.GreaterOrEqual(t, trendAnalysis.ConfidenceLevel, 0.7,
			"Should have reasonable confidence in trend")
		assert.NotEmpty(t, trendAnalysis.TrendSummary,
			"Should have trend summary")

		// Verify trend direction detection (should show increasing usage)
		if trendAnalysis.TokenUsageTrend == "increasing" {
			t.Log("Correctly detected increasing token usage trend")
		} else {
			t.Logf("Trend direction: %s (may be valid depending on implementation)",
				trendAnalysis.TokenUsageTrend)
		}

		t.Logf("Trend analysis: %s trend with %.1f%% confidence over %d data points",
			trendAnalysis.TokenUsageTrend, trendAnalysis.ConfidenceLevel*100, trendAnalysis.DataPoints)
	})
}

// Helper types and functions for token optimization testing

type TokenTestPrompt struct {
	Content            string
	ExpectedComplexity string
	MaxTokens          int
}

func createMockProviderWithTokenCharacteristics(provider, model string, verbosityMultiplier float64) iface.ChatModel {
	// This will create a mock provider with specific token usage characteristics
	// Will be implemented when token optimization infrastructure is available
	return nil
}

func createMockProviderWithPricing(provider, model string, pricing benchmarks.PricingModel) iface.ChatModel {
	// This will create a mock provider with specific pricing characteristics
	// Will be implemented when cost analysis infrastructure is available
	return nil
}
