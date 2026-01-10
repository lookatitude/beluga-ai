// Package chatmodels provides chat-based language model implementations following the Beluga AI Framework design patterns.
//
// This package implements chat models that can handle conversation-like interactions with various providers.
// It follows SOLID principles with dependency inversion, interface segregation, and composition over inheritance.
//
// Key Features:
//   - Multiple provider support (OpenAI, Anthropic, local models)
//   - Streaming and non-streaming message generation
//   - Comprehensive error handling with custom error types
//   - Observability with OpenTelemetry tracing and metrics
//   - Configurable generation with retry logic and timeouts
//   - Health checking and model information
//
// Basic Usage:
//
//	// Create a chat model
//	config := chatmodels.DefaultConfig()
//	config.DefaultProvider = "openai"
//	model, err := chatmodels.NewChatModel("gpt-4", config)
//
//	// Generate messages
//	:= []schema.Message{
//		{Role: "user", Content: "Hello, how are you?"},
//	}
//	response, err := model.GenerateMessages(ctx, messages)
//
// Advanced Usage:
//
//	// Create with custom configuration
//	model, err := chatmodels.NewChatModel("gpt-4", config,
//		chatmodels.WithTemperature(0.8),
//		chatmodels.WithMaxTokens(2000),
//		chatmodels.WithFunctionCalling(true),
//	)
//
//	// Use streaming
//	stream, err := model.StreamMessages(ctx, messages)
//	for msg := range stream {
//		fmt.Println("Received:", msg.Content)
//	}
package chatmodels

import (
	"context"
	"errors"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/internal/mock"
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// NewChatModel creates a new chat model instance with the specified model name and configuration.
// This is the main factory function for creating chat models.
//
// Parameters:
//   - model: The model name/identifier (e.g., "gpt-4", "claude-3")
//   - config: Configuration instance (use DefaultConfig() for defaults)
//   - opts: Optional configuration functions
//
// Returns:
//   - Chat model instance implementing the ChatModel interface
//   - Error if initialization fails
//
// Example:
//
//	config := chatmodels.DefaultConfig()
//	model, err := chatmodels.NewChatModel("gpt-4", config,
//		chatmodels.WithTemperature(0.7),
//		chatmodels.WithMaxTokens(1000),
//	)
func NewChatModel(model string, config *Config, opts ...iface.Option) (iface.ChatModel, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply functional options
	options := &iface.Options{
		Temperature:     config.DefaultTemperature,
		MaxTokens:       config.DefaultMaxTokens,
		TopP:            config.DefaultTopP,
		StopSequences:   config.DefaultStopSequences,
		SystemPrompt:    config.DefaultSystemPrompt,
		FunctionCalling: config.DefaultFunctionCalling,
		Timeout:         config.DefaultTimeout,
		MaxRetries:      config.DefaultMaxRetries,
		EnableMetrics:   config.EnableMetrics,
		EnableTracing:   config.EnableTracing,
	}

	for _, opt := range opts {
		configMap := make(map[string]any)
		opt.Apply(&configMap)
		// Apply the config values to options
		if temp, ok := configMap["temperature"].(float32); ok {
			options.Temperature = temp
		}
		if maxTokens, ok := configMap["max_tokens"].(int); ok {
			options.MaxTokens = maxTokens
		}
		if topP, ok := configMap["top_p"].(float32); ok {
			options.TopP = topP
		}
		if stopSequences, ok := configMap["stop_sequences"].([]string); ok {
			options.StopSequences = stopSequences
		}
		if systemPrompt, ok := configMap["system_prompt"].(string); ok {
			options.SystemPrompt = systemPrompt
		}
		if functionCalling, ok := configMap["function_calling"].(bool); ok {
			options.FunctionCalling = functionCalling
		}
		if timeout, ok := configMap["timeout"].(time.Duration); ok {
			options.Timeout = timeout
		}
		if maxRetries, ok := configMap["max_retries"].(int); ok {
			options.MaxRetries = maxRetries
		}
		if enableMetrics, ok := configMap["enable_metrics"].(bool); ok {
			options.EnableMetrics = enableMetrics
		}
		if enableTracing, ok := configMap["enable_tracing"].(bool); ok {
			options.EnableTracing = enableTracing
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewChatModelError("creation", model, config.DefaultProvider, ErrCodeConfigInvalid, err)
	}

	// Try to get provider from registry first
	registry := GetRegistry()
	if registry.IsRegistered(config.DefaultProvider) {
		return registry.CreateProvider(model, config, options)
	}

	// Fallback to switch statement for backward compatibility
	switch config.DefaultProvider {
	case "openai":
		return openai.NewOpenAIChatModel(model, config, options)
	case "mock":
		return mock.NewMockChatModel(model, config, options)
	default:
		return nil, NewChatModelError("creation", model, config.DefaultProvider, ErrCodeProviderNotSupported,
			errors.New("unsupported provider: "+config.DefaultProvider))
	}
}

// NewOpenAIChatModel creates a new OpenAI chat model instance.
// This is a convenience function for creating OpenAI-specific models.
//
// Parameters:
//   - model: OpenAI model name (e.g., "gpt-4", "gpt-3.5-turbo")
//   - apiKey: OpenAI API key
//   - opts: Optional configuration functions
//
// Returns:
//   - OpenAI chat model instance
//   - Error if initialization fails
//
// Example:
//
//	model, err := chatmodels.NewOpenAIChatModel("gpt-4", "your-api-key",
//		chatmodels.WithTemperature(0.7),
//	)
func NewOpenAIChatModel(model, apiKey string, opts ...iface.Option) (iface.ChatModel, error) {
	config := DefaultConfig()
	config.DefaultProvider = "openai"

	// Set API key in provider config
	config.Providers["openai"] = &ProviderConfig{
		APIKey:     apiKey,
		Timeout:    config.DefaultTimeout,
		MaxRetries: config.DefaultMaxRetries,
	}

	return NewChatModel(model, config, opts...)
}

// NewMockChatModel creates a new mock chat model for testing.
// This creates a chat model that returns predetermined responses.
//
// Parameters:
//   - model: Model name for the mock
//   - opts: Optional configuration functions
//
// Returns:
//   - Mock chat model instance
//   - Error if initialization fails
//
// Example:
//
//	model, err := chatmodels.NewMockChatModel("mock-gpt-4",
//		chatmodels.WithTemperature(0.5),
//	)
func NewMockChatModel(model string, opts ...iface.Option) (iface.ChatModel, error) {
	config := DefaultConfig()
	config.DefaultProvider = "mock"

	return NewChatModel(model, config, opts...)
}

// NewDefaultConfig creates a new configuration instance with default values.
// This provides sensible defaults for most use cases while allowing customization.
//
// Returns:
//   - Configuration instance with defaults
//
// Example:
//
//	config := chatmodels.NewDefaultConfig()
//	config.DefaultTemperature = 0.8
//	model, err := chatmodels.NewChatModel("gpt-4", config)
func NewDefaultConfig() *Config {
	return DefaultConfig()
}

// ValidateConfig validates a chat model configuration.
// This ensures the configuration is complete and contains valid values.
//
// Parameters:
//   - config: Configuration to validate
//
// Returns:
//   - Error if validation fails, nil otherwise
//
// Example:
//
//	config := chatmodels.DefaultConfig()
//	if err := chatmodels.ValidateConfig(config); err != nil {
//		log.Fatal("Invalid config:", err)
//	}
func ValidateConfig(config *Config) error {
	return config.Validate()
}

// GetSupportedProviders returns a list of supported chat model providers.
//
// Returns:
//   - Slice of supported provider names
//
// Example:
//
//	providers := chatmodels.GetSupportedProviders()
//	fmt.Printf("Supported providers: %v\n", providers)
func GetSupportedProviders() []string {
	return []string{"openai", "mock"}
}

// GetSupportedModels returns a list of supported models for a given provider.
//
// Parameters:
//   - provider: Provider name
//
// Returns:
//   - Slice of supported model names
//
// Example:
//
//	models := chatmodels.GetSupportedModels("openai")
//	fmt.Printf("OpenAI models: %v\n", models)
func GetSupportedModels(provider string) []string {
	switch provider {
	case "openai":
		return []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "gpt-4o", "gpt-4o-mini"}
	case "mock":
		return []string{"mock-gpt-4", "mock-claude", "mock-general"}
	default:
		return []string{}
	}
}

// HealthCheck performs a health check on a chat model.
// This can be used for monitoring and ensuring model availability.
//
// Parameters:
//   - model: Chat model to check
//
// Returns:
//   - Health status information
//
// Example:
//
//	status := chatmodels.HealthCheck(model)
//	if status["state"] == "error" {
//		log.Warn("Model is in error state")
//	}
func HealthCheck(model iface.ChatModel) map[string]any {
	return model.CheckHealth()
}

// GetModelInfo retrieves model information from a chat model.
//
// Parameters:
//   - model: Chat model instance
//
// Returns:
//   - Model information struct
//
// Example:
//
//	info := chatmodels.GetModelInfo(model)
//	fmt.Printf("Model: %s, Provider: %s\n", info.Name, info.Provider)
func GetModelInfo(model iface.ChatModel) iface.ModelInfo {
	return model.GetModelInfo()
}

// GenerateMessages is a convenience function for generating messages with a chat model.
// This wraps the model's GenerateMessages method for easier usage.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - model: Chat model instance
//   - messages: Input messages
//   - opts: Optional generation options
//
// Returns:
//   - Generated response messages
//   - Error if generation fails
//
// Example:
//
//	messages := []schema.Message{
//		schema.NewHumanMessage("Hello!"),
//	}
//	response, err := chatmodels.GenerateMessages(ctx, model, messages)
func GenerateMessages(ctx context.Context, model iface.ChatModel, messages []schema.Message, opts ...iface.Option) ([]schema.Message, error) {
	// Convert iface.Option to core.Option
	coreOpts := make([]core.Option, len(opts))
	for i, opt := range opts {
		coreOpts[i] = opt
	}
	return model.GenerateMessages(ctx, messages, coreOpts...)
}

// StreamMessages is a convenience function for streaming messages with a chat model.
// This wraps the model's StreamMessages method for easier usage.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - model: Chat model instance
//   - messages: Input messages
//   - opts: Optional streaming options
//
// Returns:
//   - Channel of streaming messages
//   - Error if streaming fails
//
// Example:
//
//	stream, err := chatmodels.StreamMessages(ctx, model, messages)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for msg := range stream {
//		fmt.Println("Received:", msg.GetContent())
//	}
func StreamMessages(ctx context.Context, model iface.ChatModel, messages []schema.Message, opts ...iface.Option) (<-chan schema.Message, error) {
	// Convert iface.Option to core.Option
	coreOpts := make([]core.Option, len(opts))
	for i, opt := range opts {
		coreOpts[i] = opt
	}
	return model.StreamMessages(ctx, messages, coreOpts...)
}

// Compile-time checks to ensure implementations satisfy interfaces
// These are checked at build time to ensure proper interface implementation.
var (
	_ iface.ChatModel            = (*openai.OpenAIChatModel)(nil)
	_ iface.ChatModel            = (*mock.MockChatModel)(nil)
	_ iface.MessageGenerator     = (*openai.OpenAIChatModel)(nil)
	_ iface.MessageGenerator     = (*mock.MockChatModel)(nil)
	_ iface.StreamMessageHandler = (*openai.OpenAIChatModel)(nil)
	_ iface.StreamMessageHandler = (*mock.MockChatModel)(nil)
	_ iface.ModelInfoProvider    = (*openai.OpenAIChatModel)(nil)
	_ iface.ModelInfoProvider    = (*mock.MockChatModel)(nil)
	_ iface.HealthChecker        = (*openai.OpenAIChatModel)(nil)
	_ iface.HealthChecker        = (*mock.MockChatModel)(nil)
)
