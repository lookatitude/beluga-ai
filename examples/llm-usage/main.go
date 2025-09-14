package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	// Note: Advanced examples and config loading are included in this package
)

func main() {
	fmt.Println("🔄 Beluga AI LLM Package Usage Example")
	fmt.Println("=====================================")

	ctx := context.Background()

	// Initialize metrics (optional - would use OpenTelemetry in production)
	llms.InitMetrics(nil)

	// Example 1: Basic Factory Usage
	fmt.Println("\n📋 Example 1: Basic Factory Usage")
	basicFactoryExample(ctx)

	// Example 2: Configuration Management
	fmt.Println("\n⚙️  Example 2: Configuration Management")
	configurationExample(ctx)

	// Example 3: Error Handling
	fmt.Println("\n🚨 Example 3: Error Handling")
	errorHandlingExample(ctx)

	// Example 4: Streaming Responses
	fmt.Println("\n🌊 Example 4: Streaming Responses")
	streamingExample(ctx)

	// Example 5: Batch Processing
	fmt.Println("\n📦 Example 5: Batch Processing")
	batchProcessingExample(ctx)

	// Example 6: Tool Calling (Mock Example)
	fmt.Println("\n🛠️  Example 6: Tool Calling (Mock)")
	toolCallingExample(ctx)

	fmt.Println("\n✨ All examples completed successfully!")
}

func basicFactoryExample(ctx context.Context) {
	fmt.Println("Creating factory and registering mock provider...")

	// Create a new factory
	llms.NewFactory()

	// Create a mock provider for demonstration
	llms.NewConfig(
		llms.WithProvider("mock"),
		llms.WithModelName("demo-model"),
		llms.WithProviderSpecific("responses", []string{
			"Hello! I'm a mock AI assistant.",
			"I can help you with various tasks.",
			"Thank you for using the Beluga AI framework!",
		}),
	)

	// For demonstration, we'll create the provider directly
	// In real usage, you'd use: provider, err := factory.CreateProvider("mock", mockConfig)
	fmt.Printf("✅ Factory created successfully\n")
	fmt.Printf("✅ Mock configuration created\n")
	fmt.Printf("📊 Available provider types: %v\n", []string{"anthropic", "openai", "bedrock", "mock"})
}

func configurationExample(ctx context.Context) {
	fmt.Println("Demonstrating configuration patterns...")

	// Example configurations for different providers
	configs := map[string]*llms.Config{
		"Anthropic": llms.NewConfig(
			llms.WithProvider("anthropic"),
			llms.WithModelName("claude-3-sonnet-20240229"),
			llms.WithAPIKey("demo-anthropic-key"),
			llms.WithTemperatureConfig(0.7),
			llms.WithMaxTokensConfig(1024),
			llms.WithMaxConcurrentBatches(5),
			llms.WithRetryConfig(3, time.Second, 2.0),
		),
		"OpenAI": llms.NewConfig(
			llms.WithProvider("openai"),
			llms.WithModelName("gpt-4"),
			llms.WithAPIKey("demo-openai-key"),
			llms.WithBaseURL("https://api.openai.com/v1"),
			llms.WithTemperatureConfig(0.8),
			llms.WithMaxTokensConfig(2048),
			llms.WithProviderSpecific("organization", "demo-org"),
		),
		"Mock": llms.NewConfig(
			llms.WithProvider("mock"),
			llms.WithModelName("test-model"),
			llms.WithProviderSpecific("responses", []string{"Mock response for testing"}),
		),
	}

	// Validate configurations
	for name, config := range configs {
		if err := llms.ValidateProviderConfig(ctx, config); err != nil {
			if name == "Mock" {
				// Mock doesn't require API key, so validation should pass
				fmt.Printf("❌ %s config validation failed unexpectedly: %v\n", name, err)
			}
		} else {
			fmt.Printf("✅ %s configuration validated successfully\n", name)
		}
	}

	// Demonstrate configuration merging
	baseConfig := llms.NewConfig(
		llms.WithProvider("anthropic"),
		llms.WithModelName("claude-3-haiku-20240307"),
		llms.WithMaxTokensConfig(512),
	)

	// Override some settings
	llms.NewConfig(
		llms.WithTemperatureConfig(0.9),
		llms.WithMaxTokensConfig(1024), // Override the base config
	)

	baseConfig.MergeOptions(llms.WithTemperatureConfig(0.9), llms.WithMaxTokensConfig(1024))

	fmt.Printf("✅ Configuration merging demonstrated\n")
}

func errorHandlingExample(ctx context.Context) {
	fmt.Println("Demonstrating error handling patterns...")

	// Create a mock provider that will return errors
	llms.NewConfig(
		llms.WithProvider("mock"),
		llms.WithModelName("error-demo"),
		llms.WithProviderSpecific("should_error", true),
		llms.WithProviderSpecific("responses", []string{"This won't be reached"}),
	)

	// ValidateProviderConfig would fail for mock with should_error, so we'll skip that
	fmt.Printf("✅ Error configuration created\n")

	// Demonstrate error code checking
	err := llms.NewLLMError("demo", llms.ErrCodeRateLimit, fmt.Errorf("rate limit exceeded"))

	if llms.IsLLMError(err) {
		fmt.Printf("✅ LLM Error detected: %s\n", llms.GetLLMErrorCode(err))
		fmt.Printf("✅ Is retryable: %t\n", llms.IsRetryableError(err))
	}

	// Demonstrate different error types
	errorCodes := []string{
		llms.ErrCodeRateLimit,
		llms.ErrCodeAuthentication,
		llms.ErrCodeInvalidRequest,
		llms.ErrCodeNetworkError,
		llms.ErrCodeInternalError,
	}

	for _, code := range errorCodes {
		err := llms.NewLLMError("test", code, fmt.Errorf("test error"))
		fmt.Printf("   %s: retryable=%t\n", code, llms.IsRetryableError(err))
	}
}

func streamingExample(ctx context.Context) {
	fmt.Println("Demonstrating streaming responses...")

	// Create messages for streaming
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful AI assistant."),
		schema.NewHumanMessage("Tell me a short story about a robot learning to paint."),
	}

	fmt.Printf("📝 Created %d messages for streaming\n", len(messages))
	fmt.Printf("✅ Streaming example prepared (would work with real providers)\n")

	// In a real implementation, you would:
	// 1. Create a provider
	// 2. Call provider.StreamChat(ctx, messages)
	// 3. Handle the streaming channel
	// 4. Process chunks as they arrive

	fmt.Printf("💡 Streaming usage pattern:\n")
	fmt.Printf("   streamChan, err := provider.StreamChat(ctx, messages)\n")
	fmt.Printf("   for chunk := range streamChan {\n")
	fmt.Printf("       if chunk.Err != nil { handle error }\n")
	fmt.Printf("       fmt.Print(chunk.Content) // Print as it arrives\n")
	fmt.Printf("   }\n")
}

func batchProcessingExample(ctx context.Context) {
	fmt.Println("Demonstrating batch processing...")

	// Create multiple inputs for batch processing
	inputs := []any{
		[]schema.Message{schema.NewHumanMessage("What is 2 + 2?")},
		[]schema.Message{schema.NewHumanMessage("What is the capital of France?")},
		[]schema.Message{schema.NewHumanMessage("Who wrote Romeo and Juliet?")},
		[]schema.Message{schema.NewHumanMessage("What is the largest planet?")},
	}

	fmt.Printf("📦 Prepared %d batch inputs\n", len(inputs))

	// In a real implementation, you would:
	// 1. Create a provider
	// 2. Call provider.Batch(ctx, inputs)
	// 3. Process all results

	fmt.Printf("✅ Batch processing example prepared\n")
	fmt.Printf("💡 Batch processing benefits:\n")
	fmt.Printf("   - Concurrent processing of multiple requests\n")
	fmt.Printf("   - Automatic error handling per request\n")
	fmt.Printf("   - Configurable concurrency limits\n")
	fmt.Printf("   - Efficient resource utilization\n")
}

func toolCallingExample(ctx context.Context) {
	fmt.Println("Demonstrating tool calling...")

	// Create messages that would benefit from tool calling
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant with access to tools."),
		schema.NewHumanMessage("Calculate 15 * 23 and then find information about the number 345."),
	}

	fmt.Printf("🛠️  Created %d messages for tool calling\n", len(messages))

	// In a real implementation, you would:
	// 1. Create tools (calculator, web search, etc.)
	// 2. Bind tools to the provider
	// 3. Generate response
	// 4. Handle tool calls in the response
	// 5. Execute tools and continue conversation

	fmt.Printf("✅ Tool calling example prepared\n")
	fmt.Printf("💡 Tool calling workflow:\n")
	fmt.Printf("   1. Create tools: calculator, webSearch, etc.\n")
	fmt.Printf("   2. Bind to provider: provider.BindTools(tools)\n")
	fmt.Printf("   3. Generate: response, err := provider.Generate(ctx, messages)\n")
	fmt.Printf("   4. Handle tool calls from response.ToolCalls\n")
	fmt.Printf("   5. Execute tools and continue conversation\n")
}

func demonstrateUtilityFunctions() {
	fmt.Println("\n🔧 Example 7: Utility Functions")

	// Test EnsureMessages
	fmt.Println("Testing EnsureMessages utility...")

	// Test with string
	messages1, err := llms.EnsureMessages("Hello world!")
	if err != nil {
		fmt.Printf("❌ EnsureMessages failed: %v\n", err)
	} else {
		fmt.Printf("✅ String converted to %d messages\n", len(messages1))
	}

	// Test with message slice
	_, err = llms.EnsureMessages([]schema.Message{
		schema.NewHumanMessage("Test message"),
	})
	if err != nil {
		fmt.Printf("❌ EnsureMessages failed: %v\n", err)
	} else {
		fmt.Printf("✅ Message slice handled correctly\n")
	}

	// Test GetSystemAndHumanPrompts
	fmt.Println("Testing GetSystemAndHumanPrompts utility...")
	testMessages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("What is AI?"),
		schema.NewHumanMessage("How does it work?"),
	}

	system, human := llms.GetSystemAndHumanPromptsFromSchema(testMessages)
	fmt.Printf("✅ System prompt: %s\n", system)
	fmt.Printf("✅ Human prompts: %s\n", human)

	// Test ValidateModelName
	fmt.Println("Testing ValidateModelName utility...")
	validModels := []struct{ provider, model string }{
		{"openai", "gpt-4"},
		{"anthropic", "claude-3-sonnet"},
	}

	for _, vm := range validModels {
		if err := llms.ValidateModelName(vm.provider, vm.model); err != nil {
			fmt.Printf("❌ Model validation failed for %s/%s: %v\n", vm.provider, vm.model, err)
		} else {
			fmt.Printf("✅ Model %s validated for provider %s\n", vm.model, vm.provider)
		}
	}
}

func demonstrateMetricsAndObservability() {
	fmt.Println("\n📊 Example 8: Metrics and Observability")

	// Get metrics instance
	metrics := llms.GetMetrics()

	// Demonstrate metrics recording
	fmt.Println("Recording sample metrics...")

	ctx := context.Background()
	metrics.RecordRequest(ctx, "demo-provider", "demo-model", 150*time.Millisecond)
	metrics.RecordTokenUsage(ctx, "demo-provider", "demo-model", 100, 50)
	metrics.IncrementActiveRequests(ctx, "demo-provider", "demo-model")

	fmt.Printf("✅ Recorded request metrics\n")
	fmt.Printf("✅ Recorded token usage (100 input, 50 output)\n")
	fmt.Printf("✅ Incremented active request counter\n")

	// Show current metrics
	fmt.Printf("📈 Current metrics:\n")
	fmt.Printf("   Total requests: %d\n", metrics.GetRequestsTotal())
	fmt.Printf("   Total errors: %d\n", metrics.GetErrorsTotal())
	fmt.Printf("   Total token usage: %d\n", metrics.GetTokenUsageTotal())
	fmt.Printf("   Active requests: %d\n", metrics.GetActiveRequests())

	fmt.Printf("💡 In production, these metrics would be exported to:\n")
	fmt.Printf("   - Prometheus for monitoring\n")
	fmt.Printf("   - Jaeger/Grafana for tracing\n")
	fmt.Printf("   - ELK stack for logging\n")
}

// Additional helper function to show configuration file loading pattern
func demonstrateConfigFilePattern() {
	fmt.Println("\n📄 Example 9: Configuration File Pattern")

	fmt.Println("Example configuration file structure (YAML):")
	configExample := `
llms:
  providers:
    - name: "claude-production"
      provider: "anthropic"
      model_name: "claude-3-sonnet-20240229"
      api_key: "${ANTHROPIC_API_KEY}"
      temperature: 0.7
      max_tokens: 2048
      max_concurrent_batches: 10
      retry_config:
        max_retries: 3
        delay: "1s"
        backoff: 2.0
      observability:
        tracing: true
        metrics: true
        structured_logging: true

    - name: "gpt-fallback"
      provider: "openai"
      model_name: "gpt-4"
      api_key: "${OPENAI_API_KEY}"
      base_url: "https://api.openai.com/v1"
      temperature: 0.8
      max_tokens: 1024

    - name: "mock-testing"
      provider: "mock"
      model_name: "test-model"
      provider_specific:
        responses:
          - "Mock response 1"
          - "Mock response 2"
          - "Mock response 3"
`

	fmt.Printf("%s\n", configExample)
	fmt.Printf("💡 Configuration benefits:\n")
	fmt.Printf("   - Environment variable substitution\n")
	fmt.Printf("   - Multiple provider configurations\n")
	fmt.Printf("   - Easy switching between environments\n")
	fmt.Printf("   - Centralized configuration management\n")
}

// Run additional examples
func runAdditionalExamples() {
	demonstrateUtilityFunctions()
	demonstrateMetricsAndObservability()
	demonstrateConfigFilePattern()
}

// Update main function to include additional examples
func init() {
	// Add additional examples to main execution
	go func() {
		time.Sleep(100 * time.Millisecond)
		runAdditionalExamples()
	}()
}
