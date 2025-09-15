package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms"
)

// AdvancedExample demonstrates advanced usage patterns of the LLM package
func AdvancedExample() {
	fmt.Println("\n🚀 Advanced LLM Usage Examples")
	fmt.Println("================================")

	ctx := context.Background()

	// Example 1: Provider Registration Pattern
	fmt.Println("\n📝 Example A1: Provider Registration Pattern")
	providerRegistrationExample(ctx)

	// Example 2: Configuration from Environment
	fmt.Println("\n🌍 Example A2: Configuration from Environment")
	environmentConfigExample(ctx)

	// Example 3: Advanced Error Handling with Retry
	fmt.Println("\n🔄 Example A3: Advanced Error Handling")
	advancedErrorHandlingExample(ctx)

	// Example 4: Performance Monitoring
	fmt.Println("\n📊 Example A4: Performance Monitoring")
	performanceMonitoringExample(ctx)

	// Example 5: Provider Switching Strategy
	fmt.Println("\n🔀 Example A5: Provider Switching Strategy")
	providerSwitchingExample(ctx)
}

func providerRegistrationExample(ctx context.Context) {
	fmt.Println("Demonstrating provider registration and management...")

	// Create factory
	factory := llms.NewFactory()

	// In a real application, you would register actual provider factories:
	// factory.RegisterProviderFactory("anthropic", anthropic.NewAnthropicProviderFactory())
	// factory.RegisterProviderFactory("openai", openai.NewOpenAIProviderFactory())

	fmt.Printf("✅ Factory created\n")
	fmt.Printf("📋 Available provider factories: %v\n", factory.ListAvailableProviders())

	// Demonstrate provider creation pattern
	fmt.Printf("💡 Provider creation pattern:\n")
	fmt.Printf("   config := llms.NewConfig(llms.WithProvider(\"anthropic\"), ...)\n")
	fmt.Printf("   provider, err := factory.CreateProvider(\"anthropic\", config)\n")
	fmt.Printf("   if err != nil {\n")
	fmt.Printf("       log.Printf(\"Failed to create provider: %%v\", err)\n")
	fmt.Printf("       return\n")
	fmt.Printf("   }\n")

	// Show how you would use the created provider
	fmt.Printf("💡 Provider usage pattern:\n")
	fmt.Printf("   messages := []schema.Message{schema.NewHumanMessage(\"Hello!\")}\n")
	fmt.Printf("   response, err := provider.Generate(ctx, messages)\n")
	fmt.Printf("   if err != nil {\n")
	fmt.Printf("       // Handle error with proper error codes\n")
	fmt.Printf("       return\n")
	fmt.Printf("   }\n")
	fmt.Printf("   fmt.Printf(\"Response: %%s\", response.GetContent())\n")
}

func environmentConfigExample(ctx context.Context) {
	fmt.Println("Demonstrating configuration from environment variables...")

	// Example environment variable mapping
	envVars := map[string]string{
		"ANTHROPIC_API_KEY":  "your-anthropic-key",
		"OPENAI_API_KEY":     "your-openai-key",
		"ANTHROPIC_MODEL":    "claude-3-sonnet-20240229",
		"OPENAI_MODEL":       "gpt-4",
		"LLM_TEMPERATURE":    "0.7",
		"LLM_MAX_TOKENS":     "2048",
		"LLM_MAX_RETRIES":    "3",
		"LLM_RETRY_DELAY":    "1s",
		"LLM_ENABLE_TRACING": "true",
		"LLM_ENABLE_METRICS": "true",
	}

	// Set environment variables for demonstration
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	fmt.Printf("✅ Environment variables set for demonstration\n")

	// Create configurations using environment variables
	configs := map[string]*llms.Config{
		"Anthropic": llms.NewConfig(
			llms.WithProvider("anthropic"),
			llms.WithModelName(os.Getenv("ANTHROPIC_MODEL")),
			llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
			llms.WithTemperatureConfig(0.7),
			llms.WithMaxTokensConfig(2048),
			llms.WithRetryConfig(3, time.Second, 2.0),
			llms.WithObservability(true, true, true),
		),
		"OpenAI": llms.NewConfig(
			llms.WithProvider("openai"),
			llms.WithModelName(os.Getenv("OPENAI_MODEL")),
			llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
			llms.WithTemperatureConfig(0.7),
			llms.WithMaxTokensConfig(2048),
			llms.WithRetryConfig(3, time.Second, 2.0),
			llms.WithObservability(true, true, true),
		),
	}

	// Validate configurations
	for name, config := range configs {
		if err := llms.ValidateProviderConfig(ctx, config); err != nil {
			if name == "Anthropic" || name == "OpenAI" {
				fmt.Printf("⚠️  %s config would fail without real API keys: %v\n", name, err)
			}
		} else {
			fmt.Printf("✅ %s configuration validated\n", name)
		}
	}

	fmt.Printf("💡 Environment-based configuration benefits:\n")
	fmt.Printf("   - No hardcoded secrets in code\n")
	fmt.Printf("   - Easy environment switching (dev/staging/prod)\n")
	fmt.Printf("   - Configurable via deployment tools (Docker, Kubernetes)\n")
	fmt.Printf("   - Sensitive data management\n")
}

func advancedErrorHandlingExample(ctx context.Context) {
	fmt.Println("Demonstrating advanced error handling patterns...")

	// Simulate different error scenarios
	errorScenarios := []struct {
		name        string
		err         error
		description string
	}{
		{
			name:        "Rate Limit Error",
			err:         llms.NewLLMError("generate", llms.ErrCodeRateLimit, fmt.Errorf("rate limit exceeded")),
			description: "Should trigger exponential backoff retry",
		},
		{
			name:        "Authentication Error",
			err:         llms.NewLLMError("generate", llms.ErrCodeAuthentication, fmt.Errorf("invalid API key")),
			description: "Should not retry, requires user intervention",
		},
		{
			name:        "Network Error",
			err:         llms.NewLLMError("generate", llms.ErrCodeNetworkError, fmt.Errorf("connection timeout")),
			description: "Should retry with shorter backoff",
		},
		{
			name:        "Invalid Request",
			err:         llms.NewLLMError("generate", llms.ErrCodeInvalidRequest, fmt.Errorf("malformed prompt")),
			description: "Should not retry, requires code fix",
		},
	}

	for _, scenario := range errorScenarios {
		fmt.Printf("🔍 %s:\n", scenario.name)
		fmt.Printf("   Error: %v\n", scenario.err)
		fmt.Printf("   Code: %s\n", llms.GetLLMErrorCode(scenario.err))
		fmt.Printf("   Retryable: %t\n", llms.IsRetryableError(scenario.err))
		fmt.Printf("   Description: %s\n", scenario.description)
		fmt.Println()
	}

	// Demonstrate retry logic pattern
	fmt.Printf("💡 Retry logic pattern:\n")
	fmt.Printf("   retryErr := common.RetryWithBackoff(ctx, retryConfig, \"operation\", func() error {\n")
	fmt.Printf("       return performOperation()\n")
	fmt.Printf("   })\n")
	fmt.Printf("   if retryErr != nil {\n")
	fmt.Printf("       return handleFinalError(retryErr)\n")
	fmt.Printf("   }\n")
}

func performanceMonitoringExample(ctx context.Context) {
	fmt.Println("Demonstrating performance monitoring...")

	// Get metrics instance
	metrics := llms.GetMetrics()

	// Reset metrics for clean demonstration
	metrics.Reset()

	// Simulate some activity
	fmt.Println("Simulating LLM activity...")

	providers := []string{"anthropic", "openai", "bedrock"}
	models := []string{"claude-3-sonnet", "gpt-4", "titan-text-express"}

	for i := 0; i < 10; i++ {
		provider := providers[i%len(providers)]
		model := models[i%len(models)]

		// Simulate request
		duration := time.Duration(100+i*50) * time.Millisecond
		metrics.RecordRequest(ctx, provider, model, duration)

		// Simulate token usage
		inputTokens := 100 + i*20
		outputTokens := 50 + i*10
		metrics.RecordTokenUsage(ctx, provider, model, inputTokens, outputTokens)

		// Simulate some errors (20% error rate)
		if i%5 == 0 {
			metrics.RecordError(ctx, provider, model, llms.ErrCodeNetworkError)
		}

		// Simulate active requests
		metrics.IncrementActiveRequests(ctx, provider, model)
		time.Sleep(10 * time.Millisecond) // Simulate processing time
		metrics.DecrementActiveRequests(ctx, provider, model)
	}

	// Display metrics
	fmt.Printf("📈 Performance Metrics Summary:\n")
	fmt.Printf("   Total Requests: %d\n", metrics.GetRequestsTotal())
	fmt.Printf("   Total Errors: %d\n", metrics.GetErrorsTotal())
	fmt.Printf("   Total Token Usage: %d\n", metrics.GetTokenUsageTotal())
	fmt.Printf("   Current Active Requests: %d\n", metrics.GetActiveRequests())

	// Calculate error rate
	if metrics.GetRequestsTotal() > 0 {
		errorRate := float64(metrics.GetErrorsTotal()) / float64(metrics.GetRequestsTotal()) * 100
		fmt.Printf("   Error Rate: %.1f%%\n", errorRate)
	}

	fmt.Printf("💡 Metrics benefits:\n")
	fmt.Printf("   - Real-time performance monitoring\n")
	fmt.Printf("   - Error rate tracking\n")
	fmt.Printf("   - Token usage optimization\n")
	fmt.Printf("   - Provider performance comparison\n")
	fmt.Printf("   - Alerting and anomaly detection\n")
}

func providerSwitchingExample(ctx context.Context) {
	fmt.Println("Demonstrating provider switching strategies...")

	// Simulate provider health status
	providerHealth := map[string]bool{
		"anthropic": true,
		"openai":    false, // Simulating outage
		"bedrock":   true,
		"mock":      true,
	}

	// Provider priority order
	providerPriority := []string{"anthropic", "openai", "bedrock", "mock"}

	fmt.Printf("🏥 Provider Health Status:\n")
	for provider, healthy := range providerHealth {
		status := "✅ Healthy"
		if !healthy {
			status = "❌ Unhealthy"
		}
		fmt.Printf("   %s: %s\n", provider, status)
	}

	// Find first healthy provider
	var selectedProvider string
	for _, provider := range providerPriority {
		if providerHealth[provider] {
			selectedProvider = provider
			break
		}
	}

	fmt.Printf("🎯 Selected Provider: %s\n", selectedProvider)

	// Simulate provider switching logic
	fmt.Printf("💡 Provider switching patterns:\n")
	fmt.Printf("   1. Health Check Pattern:\n")
	fmt.Printf("      func getHealthyProvider() (string, error) {\n")
	fmt.Printf("          for _, provider := range priorityOrder {\n")
	fmt.Printf("              if isHealthy(provider) {\n")
	fmt.Printf("                  return provider, nil\n")
	fmt.Printf("              }\n")
	fmt.Printf("          }\n")
	fmt.Printf("          return \"\", fmt.Errorf(\"no healthy providers\")\n")
	fmt.Printf("      }\n")
	fmt.Printf("\n")
	fmt.Printf("   2. Fallback Pattern:\n")
	fmt.Printf("      provider, err := factory.CreateProvider(primaryProvider, config)\n")
	fmt.Printf("      if err != nil {\n")
	fmt.Printf("          log.Printf(\"Primary provider failed, trying fallback...\")\n")
	fmt.Printf("          provider, err = factory.CreateProvider(fallbackProvider, config)\n")
	fmt.Printf("      }\n")
	fmt.Printf("\n")
	fmt.Printf("   3. Load Balancing Pattern:\n")
	fmt.Printf("      provider := getLeastLoadedProvider(availableProviders)\n")
	fmt.Printf("      // Route request to least loaded provider\n")

	fmt.Printf("✅ Provider switching strategy demonstrated\n")
}

// Helper function to demonstrate configuration validation patterns
func demonstrateConfigValidation() {
	fmt.Println("\n🔍 Example A6: Configuration Validation Patterns")

	// Test various configuration scenarios
	testConfigs := []struct {
		name        string
		config      *llms.Config
		shouldError bool
		errorReason string
	}{
		{
			name: "Valid Anthropic Config",
			config: llms.NewConfig(
				llms.WithProvider("anthropic"),
				llms.WithModelName("claude-3-sonnet-20240229"),
				llms.WithAPIKey("test-key"),
			),
			shouldError: false,
		},
		{
			name: "Missing Provider",
			config: llms.NewConfig(
				llms.WithModelName("claude-3-sonnet"),
				llms.WithAPIKey("test-key"),
			),
			shouldError: true,
			errorReason: "provider is required",
		},
		{
			name: "Missing Model",
			config: llms.NewConfig(
				llms.WithProvider("anthropic"),
				llms.WithAPIKey("test-key"),
			),
			shouldError: true,
			errorReason: "model name is required",
		},
		{
			name: "Valid Mock Config",
			config: llms.NewConfig(
				llms.WithProvider("mock"),
				llms.WithModelName("test-model"),
			),
			shouldError: false,
		},
		{
			name: "Invalid Temperature",
			config: llms.NewConfig(
				llms.WithProvider("anthropic"),
				llms.WithModelName("claude-3-sonnet"),
				llms.WithAPIKey("test-key"),
				llms.WithTemperatureConfig(3.0), // Invalid: > 2.0
			),
			shouldError: true,
			errorReason: "temperature out of range",
		},
	}

	fmt.Printf("Testing %d configuration scenarios:\n", len(testConfigs))

	for _, test := range testConfigs {
		err := test.config.Validate()
		if test.shouldError {
			if err != nil {
				fmt.Printf("✅ %s: Correctly failed validation\n", test.name)
			} else {
				fmt.Printf("❌ %s: Should have failed but didn't\n", test.name)
			}
		} else {
			if err != nil {
				fmt.Printf("❌ %s: Should have passed but failed: %v\n", test.name, err)
			} else {
				fmt.Printf("✅ %s: Correctly passed validation\n", test.name)
			}
		}
	}

	fmt.Printf("💡 Configuration validation benefits:\n")
	fmt.Printf("   - Early error detection\n")
	fmt.Printf("   - Consistent configuration format\n")
	fmt.Printf("   - Developer experience improvement\n")
	fmt.Printf("   - Runtime stability\n")
}

// Integration with the main example
func init() {
	// Add advanced examples to the main execution flow
	go func() {
		time.Sleep(200 * time.Millisecond)
		AdvancedExample()
		demonstrateConfigValidation()
	}()
}
