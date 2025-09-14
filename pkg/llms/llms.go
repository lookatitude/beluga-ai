// Package llms provides interfaces and implementations for Large Language Model interactions.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package llms

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel/metric"
)

// Global metrics instance - initialized once
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance
func InitMetrics(meter metric.Meter) {
	metricsOnce.Do(func() {
		globalMetrics = NewMetrics()
	})
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	return globalMetrics
}

// Factory provides a factory pattern for creating LLM instances.
// It manages the creation of different LLM providers based on configuration.
type Factory struct {
	providers         map[string]iface.ChatModel
	llms              map[string]iface.LLM
	providerFactories map[string]func(*Config) (iface.ChatModel, error)
	mu                sync.RWMutex
}

// NewFactory creates a new LLM factory
func NewFactory() *Factory {
	return &Factory{
		providers:         make(map[string]iface.ChatModel),
		llms:              make(map[string]iface.LLM),
		providerFactories: make(map[string]func(*Config) (iface.ChatModel, error)),
	}
}

// RegisterProvider registers a ChatModel provider with the factory
func (f *Factory) RegisterProvider(name string, provider iface.ChatModel) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[name] = provider
}

// RegisterLLM registers an LLM provider with the factory
func (f *Factory) RegisterLLM(name string, llm iface.LLM) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.llms[name] = llm
}

// GetProvider returns a registered ChatModel provider
func (f *Factory) GetProvider(name string) (iface.ChatModel, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	provider, exists := f.providers[name]
	if !exists {
		return nil, NewLLMError("GetProvider", ErrCodeUnsupportedProvider,
			fmt.Errorf("provider '%s' not registered", name))
	}
	return provider, nil
}

// GetLLM returns a registered LLM provider
func (f *Factory) GetLLM(name string) (iface.LLM, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	llm, exists := f.llms[name]
	if !exists {
		return nil, NewLLMError("GetLLM", ErrCodeUnsupportedProvider,
			fmt.Errorf("LLM '%s' not registered", name))
	}
	return llm, nil
}

// ListProviders returns a list of all registered provider names
func (f *Factory) ListProviders() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.providers))
	for name := range f.providers {
		names = append(names, name)
	}
	return names
}

// ListLLMs returns a list of all registered LLM names
func (f *Factory) ListLLMs() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.llms))
	for name := range f.llms {
		names = append(names, name)
	}
	return names
}

// CreateProvider creates a provider instance using the registered factory
func (f *Factory) CreateProvider(providerName string, config *Config) (iface.ChatModel, error) {
	f.mu.RLock()
	factory, exists := f.providerFactories[providerName]
	f.mu.RUnlock()

	if !exists {
		return nil, NewLLMError("CreateProvider", ErrCodeUnsupportedProvider,
			fmt.Errorf("provider factory '%s' not registered", providerName))
	}

	// Set provider name in config if not already set
	if config.Provider == "" {
		config.Provider = providerName
	}

	return factory(config)
}

// ListAvailableProviders returns a list of all available provider names
func (f *Factory) ListAvailableProviders() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.providerFactories))
	for name := range f.providerFactories {
		names = append(names, name)
	}
	return names
}

// RegisterProviderFactory registers a provider factory function
func (f *Factory) RegisterProviderFactory(name string, factory func(*Config) (iface.ChatModel, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providerFactories[name] = factory
}

// EnsureMessages ensures the input is a slice of schema.Message.
// It attempts to convert common input types (like a single string or Message) into the required format.
func EnsureMessages(input any) ([]schema.Message, error) {
	switch v := input.(type) {
	case string:
		return []schema.Message{schema.NewHumanMessage(v)}, nil
	case schema.Message:
		return []schema.Message{v}, nil
	case []schema.Message:
		return v, nil
	default:
		return nil, NewLLMError("EnsureMessages", ErrCodeInvalidRequest,
			fmt.Errorf("invalid input type for messages: %T", input))
	}
}

// EnsureMessagesFromSchema ensures the input is a slice of schema.Message.
// It attempts to convert common input types (like a single string or Message) into the required format.
// Deprecated: Use EnsureMessages instead.
func EnsureMessagesFromSchema(input any) ([]schema.Message, error) {
	return EnsureMessages(input)
}

// GetSystemAndHumanPromptsFromSchema extracts the system prompt and concatenates human messages.
// This is a utility function that might be useful for models that don't support distinct system messages
// or require a single prompt string.
func GetSystemAndHumanPromptsFromSchema(messages []schema.Message) (string, string) {
	var systemPrompt string
	var humanPrompts []string
	for _, msg := range messages {
		if msg.GetType() == schema.RoleSystem {
			systemPrompt = msg.GetContent()
		} else if msg.GetType() == schema.RoleHuman {
			humanPrompts = append(humanPrompts, msg.GetContent())
		}
	}
	fullHumanPrompt := ""
	for i, p := range humanPrompts {
		if i > 0 {
			fullHumanPrompt += "\n"
		}
		fullHumanPrompt += p
	}
	return systemPrompt, fullHumanPrompt
}

// Utility functions for common LLM operations

// GenerateText is a convenience function for generating text with a ChatModel
func GenerateText(ctx context.Context, model iface.ChatModel, prompt string, options ...core.Option) (string, error) {
	messages, err := EnsureMessages(prompt)
	if err != nil {
		return "", err
	}

	response, err := model.Generate(ctx, messages, options...)
	if err != nil {
		return "", err
	}

	return response.GetContent(), nil
}

// GenerateTextWithTools is a convenience function for generating text with tool calling
func GenerateTextWithTools(ctx context.Context, model iface.ChatModel, prompt string, tools []tools.Tool, options ...core.Option) (string, error) {
	messages, err := EnsureMessages(prompt)
	if err != nil {
		return "", err
	}

	// Bind tools to the model
	modelWithTools := model.BindTools(tools)

	response, err := modelWithTools.Generate(ctx, messages, options...)
	if err != nil {
		return "", err
	}

	return response.GetContent(), nil
}

// StreamText is a convenience function for streaming text generation
func StreamText(ctx context.Context, model iface.ChatModel, prompt string, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	messages, err := EnsureMessages(prompt)
	if err != nil {
		return nil, err
	}

	return model.StreamChat(ctx, messages, options...)
}

// BatchGenerate is a convenience function for batch text generation
func BatchGenerate(ctx context.Context, model iface.ChatModel, prompts []string, options ...core.Option) ([]string, error) {
	inputs := make([]any, len(prompts))
	for i, prompt := range prompts {
		messages, err := EnsureMessages(prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to convert prompt %d: %w", i, err)
		}
		inputs[i] = messages
	}

	results, err := model.Batch(ctx, inputs, options...)
	if err != nil {
		return nil, err
	}

	responses := make([]string, len(results))
	for i, result := range results {
		if msg, ok := result.(schema.Message); ok {
			responses[i] = msg.GetContent()
		} else {
			responses[i] = fmt.Sprintf("%v", result)
		}
	}

	return responses, nil
}

// ValidateModelName validates that a model name is supported by a provider
func ValidateModelName(provider, modelName string) error {
	if modelName == "" {
		return NewLLMError("ValidateModelName", ErrCodeInvalidModel,
			fmt.Errorf("model name cannot be empty"))
	}

	// Provider-specific validation could be added here
	switch provider {
	case "openai":
		validModels := []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"}
		for _, valid := range validModels {
			if modelName == valid {
				return nil
			}
		}
		return NewLLMError("ValidateModelName", ErrCodeInvalidModel,
			fmt.Errorf("unsupported OpenAI model: %s", modelName))
	case "anthropic":
		validModels := []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}
		for _, valid := range validModels {
			if modelName == valid {
				return nil
			}
		}
		return NewLLMError("ValidateModelName", ErrCodeInvalidModel,
			fmt.Errorf("unsupported Anthropic model: %s", modelName))
	default:
		// For unknown providers, just check that model name is not empty
		return nil
	}
}

// DefaultConfig returns a default configuration for LLM operations
func DefaultConfig() *Config {
	return &Config{
		Provider:                "",
		ModelName:               "",
		APIKey:                  "",
		BaseURL:                 "",
		Timeout:                 30000000000, // 30 seconds
		Temperature:             nil,
		TopP:                    nil,
		TopK:                    nil,
		MaxTokens:               nil,
		StopSequences:           nil,
		EnableStreaming:         true,
		MaxConcurrentBatches:    5,
		MaxRetries:              3,
		RetryDelay:              1000000000, // 1 second
		RetryBackoff:            2.0,
		ProviderSpecific:        make(map[string]interface{}),
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
		EnableToolCalling:       true,
	}
}

// Helper functions for backward compatibility
// These provide compatibility with the old interface while encouraging migration

// WithMaxTokensLegacy sets the maximum number of tokens to generate (deprecated, use core.WithOption)
func WithMaxTokensLegacy(tokens int) core.Option {
	return core.WithOption("max_tokens", tokens)
}

// WithTemperatureLegacy sets the sampling temperature (deprecated, use core.WithOption)
func WithTemperatureLegacy(temp float32) core.Option {
	return core.WithOption("temperature", temp)
}

// WithTopPLegacy sets the nucleus sampling probability (deprecated, use core.WithOption)
func WithTopPLegacy(topP float32) core.Option {
	return core.WithOption("top_p", topP)
}

// WithTopKLegacy sets the top-k sampling parameter (deprecated, use core.WithOption)
func WithTopKLegacy(topK int) core.Option {
	return core.WithOption("top_k", topK)
}

// WithStopWordsLegacy sets the stop sequences for generation (deprecated, use core.WithOption)
func WithStopWordsLegacy(stop []string) core.Option {
	return core.WithOption("stop_words", stop)
}

// WithToolsLegacy sets the tools that the model can call (deprecated, use core.WithOption)
func WithToolsLegacy(toolsToUse []tools.Tool) core.Option {
	return core.WithOption("tools", toolsToUse)
}

// WithToolChoiceLegacy forces the model to call a specific tool (deprecated, use core.WithOption)
func WithToolChoiceLegacy(choice string) core.Option {
	return core.WithOption("tool_choice", choice)
}

// InitializeDefaultFactory creates and returns a factory with all built-in providers registered
func InitializeDefaultFactory() *Factory {
	factory := NewFactory()

	// Register built-in provider factories
	// Note: These imports would cause circular dependencies, so they are commented out
	// In practice, users should register providers explicitly:
	//
	// import (
	//     "github.com/lookatitude/beluga-ai/pkg/llms/providers/anthropic"
	//     "github.com/lookatitude/beluga-ai/pkg/llms/providers/openai"
	//     "github.com/lookatitude/beluga-ai/pkg/llms/providers/bedrock"
	//     "github.com/lookatitude/beluga-ai/pkg/llms/providers/ollama"
	//     "github.com/lookatitude/beluga-ai/pkg/llms/providers/mock"
	// )
	//
	// factory.RegisterProviderFactory("anthropic", anthropic.NewAnthropicProviderFactory())
	// factory.RegisterProviderFactory("openai", openai.NewOpenAIProviderFactory())
	// factory.RegisterProviderFactory("bedrock", bedrock.NewBedrockProviderFactory())
	// factory.RegisterProviderFactory("ollama", ollama.NewOllamaProviderFactory())
	// factory.RegisterProviderFactory("mock", mock.NewMockProviderFactory())

	return factory
}

// Note: Provider creation functions have been moved to their respective packages
// to avoid circular dependencies. Use the factory pattern instead:
//
// factory := llms.NewFactory()
// factory.RegisterProviderFactory("anthropic", anthropic.NewAnthropicProviderFactory())
// factory.RegisterProviderFactory("openai", openai.NewOpenAIProviderFactory())
// factory.RegisterProviderFactory("bedrock", bedrock.NewBedrockProviderFactory())
// factory.RegisterProviderFactory("ollama", ollama.NewOllamaProviderFactory())
// factory.RegisterProviderFactory("mock", mock.NewMockProviderFactory())
//
// provider, err := factory.CreateProvider("anthropic", config)
//
// Or use the convenience functions:
// anthropicProvider, err := llms.NewAnthropicChat(llms.WithAPIKey("key"), ...)
// openaiProvider, err := llms.NewOpenAIChat(llms.WithAPIKey("key"), ...)
// bedrockProvider, err := llms.NewBedrockLLM(context.Background(), modelName, ...)
// ollamaProvider, err := llms.NewOllamaChat(llms.WithModelName("llama2"), ...)
//
// Or use the high-level chat model interface:
// chatModel, err := llms.NewChatModel("gpt-4", config,
//     llms.WithChatTemperature(0.7),
//     llms.WithChatMaxTokens(1000),
// )
