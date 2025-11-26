// Package llms provides chat model interfaces and implementations.
// This file contains the high-level chat model interfaces and utilities
// that build on top of the core LLM functionality.
package llms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MessageGenerator defines the interface for generating messages.
// This focuses solely on message generation capabilities.
type MessageGenerator interface {
	// GenerateMessages takes a list of messages and generates a response.
	// This is the primary method for chat-based interactions.
	GenerateMessages(ctx context.Context, messages []schema.Message, options ...core.Option) ([]schema.Message, error)
}

// StreamMessageHandler defines the interface for streaming message responses.
// This allows for real-time streaming of chat responses.
type StreamMessageHandler interface {
	// StreamMessages provides streaming responses for chat interactions.
	StreamMessages(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.Message, error)
}

// ModelInfoProvider defines the interface for providing model information.
type ModelInfoProvider interface {
	// GetModelInfo returns information about the underlying model.
	GetModelInfo() ModelInfo
}

// HealthChecker defines the interface for health checking chat model components.
type HealthChecker interface {
	// CheckHealth returns the health status information.
	CheckHealth() map[string]any
}

// ChatModel defines the core interface for chat-based language models.
// It combines message generation with model information and health checking capabilities.
// This follows the Interface Segregation Principle while providing a composite interface
// for the most common use cases.
type ChatModel interface {
	MessageGenerator
	StreamMessageHandler
	ModelInfoProvider
	HealthChecker
	core.Runnable // Embed core Runnable for consistency with framework
}

// ModelInfo contains metadata about a chat model.
type ModelInfo struct {
	Name         string
	Provider     string
	Version      string
	Capabilities []string
	MaxTokens    int
}

// ChatOption represents a functional option for configuring chat models.
type ChatOption interface {
	Apply(config *map[string]any)
}

// chatOptionFunc is a helper type that allows an ordinary function to be used as a ChatOption.
type chatOptionFunc func(config *map[string]any)

// Apply calls f(config), allowing chatOptionFunc to satisfy the ChatOption interface.
func (f chatOptionFunc) Apply(config *map[string]any) {
	f(config)
}

// ChatOptionFunc creates a new ChatOption that executes the provided function.
func ChatOptionFunc(f func(config *map[string]any)) ChatOption {
	return chatOptionFunc(f)
}

// ChatOptions holds the configuration options for chat models.
type ChatOptions struct {
	SystemPrompt    string
	StopSequences   []string
	MaxTokens       int
	Timeout         time.Duration
	MaxRetries      int
	Temperature     float32
	TopP            float32
	FunctionCalling bool
	EnableMetrics   bool
	EnableTracing   bool
}

// ChatModelFactory defines the interface for creating chat model instances.
// It enables dependency injection and different chat model creation strategies.
type ChatModelFactory interface {
	// CreateChatModel creates a new chat model instance based on the provided configuration.
	CreateChatModel(ctx context.Context, config any) (ChatModel, error)
}

// WithChatTemperature sets the temperature for chat model generation.
func WithChatTemperature(temp float32) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["temperature"] = temp
	})
}

// WithChatMaxTokens sets the maximum tokens for chat model generation.
func WithChatMaxTokens(maxTokens int) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["max_tokens"] = maxTokens
	})
}

// WithChatTopP sets the top-p value for chat model generation.
func WithChatTopP(topP float32) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["top_p"] = topP
	})
}

// WithChatStopSequences sets the stop sequences for chat model generation.
func WithChatStopSequences(sequences []string) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["stop_sequences"] = sequences
	})
}

// WithChatSystemPrompt sets the system prompt for chat model generation.
func WithChatSystemPrompt(prompt string) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["system_prompt"] = prompt
	})
}

// WithChatFunctionCalling enables or disables function calling for chat models.
func WithChatFunctionCalling(enabled bool) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["function_calling"] = enabled
	})
}

// WithChatTimeout sets the timeout for chat model operations.
func WithChatTimeout(timeout time.Duration) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["timeout"] = timeout
	})
}

// WithChatMaxRetries sets the maximum retries for chat model operations.
func WithChatMaxRetries(maxRetries int) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["max_retries"] = maxRetries
	})
}

// WithChatObservability enables or disables observability features for chat models.
func WithChatObservability(metrics, tracing bool) ChatOption {
	return ChatOptionFunc(func(config *map[string]any) {
		(*config)["enable_metrics"] = metrics
		(*config)["enable_tracing"] = tracing
	})
}

// NewChatModel creates a new chat model instance with the specified model name and configuration.
// This is the main factory function for creating chat models that wraps the underlying LLM providers.
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
//	config := llms.DefaultConfig()
//	config.DefaultProvider = "openai"
//	model, err := llms.NewChatModel("gpt-4", config,
//		llms.WithChatTemperature(0.7),
//		llms.WithChatMaxTokens(1000),
//	)
func NewChatModel(model string, config *Config, opts ...ChatOption) (iface.ChatModel, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply functional options
	options := &ChatOptions{
		Temperature:     0.7,  // Default temperature
		MaxTokens:       1024, // Default max tokens
		TopP:            1.0,  // Default top-p
		StopSequences:   config.StopSequences,
		FunctionCalling: config.EnableToolCalling,
		Timeout:         config.Timeout,
		MaxRetries:      config.MaxRetries,
		EnableMetrics:   config.EnableMetrics,
		EnableTracing:   config.EnableTracing,
	}

	// Override defaults with config values if they exist
	if config.Temperature != nil {
		options.Temperature = *config.Temperature
	}
	if config.MaxTokens != nil {
		options.MaxTokens = *config.MaxTokens
	}
	if config.TopP != nil {
		options.TopP = *config.TopP
	}

	// Convert ChatOptions to map and apply options
	configMap := make(map[string]any)
	for _, opt := range opts {
		opt.Apply(&configMap)
	}

	// Apply config values to options
	if temp, ok := configMap["temperature"].(float32); ok {
		options.Temperature = temp
	}
	if maxTokens, ok := configMap["max_tokens"].(int); ok {
		options.MaxTokens = maxTokens
	}
	if topP, ok := configMap["top_p"].(float32); ok {
		options.TopP = topP
	}
	if stopSeq, ok := configMap["stop_sequences"].([]string); ok {
		options.StopSequences = stopSeq
	}
	if sysPrompt, ok := configMap["system_prompt"].(string); ok {
		// Note: SystemPrompt is set but not currently used (stub implementation)
		// This will be used once provider registration is implemented
		options.SystemPrompt = sysPrompt //nolint:govet // unusedwrite: intentional stub, will be used later
	}
	if funcCalling, ok := configMap["function_calling"].(bool); ok {
		options.FunctionCalling = funcCalling
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

	// Create underlying LLM provider
	llmOpts := []ConfigOption{}
	if options.MaxTokens > 0 {
		llmOpts = append(llmOpts, WithMaxTokensConfig(options.MaxTokens))
	}
	if options.Temperature > 0 {
		llmOpts = append(llmOpts, WithTemperatureConfig(options.Temperature))
	}
	if options.TopP > 0 {
		llmOpts = append(llmOpts, WithTopPConfig(options.TopP))
	}
	if len(options.StopSequences) > 0 {
		llmOpts = append(llmOpts, WithStopSequences(options.StopSequences))
	}
	if options.Timeout > 0 {
		llmOpts = append(llmOpts, WithTimeout(options.Timeout))
	}

	llmConfig := NewConfig(llmOpts...)
	llmConfig.ModelName = model
	llmConfig.Provider = config.Provider

	// For now, return an error indicating that provider registration is needed
	return nil, errors.New("chat model creation requires provider registration - use factory pattern to register providers first")
}

// ChatModelAdapter wraps an LLM to provide the ChatModel interface.
type ChatModelAdapter struct {
	llm     iface.LLM
	options *ChatOptions
}

// NewChatModelAdapter creates a new ChatModelAdapter.
func NewChatModelAdapter(llm iface.LLM, options *ChatOptions) *ChatModelAdapter {
	return &ChatModelAdapter{
		llm:     llm,
		options: options,
	}
}

// GenerateMessages implements the MessageGenerator interface.
func (c *ChatModelAdapter) GenerateMessages(ctx context.Context, messages []schema.Message, options ...core.Option) ([]schema.Message, error) {
	// Convert messages to a single prompt for simple LLM invocation
	prompt := c.messagesToPrompt(messages)

	// Generate response
	response, err := c.llm.Invoke(ctx, prompt, options...)
	if err != nil {
		return nil, err
	}

	// Convert response back to message format
	responseStr, ok := response.(string)
	if !ok {
		return nil, fmt.Errorf("expected string response from LLM, got %T", response)
	}
	responseMsg := schema.NewAIMessage(responseStr)
	return []schema.Message{responseMsg}, nil
}

// StreamMessages implements the StreamMessageHandler interface.
func (c *ChatModelAdapter) StreamMessages(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.Message, error) {
	// For now, just generate a single message (streaming not implemented)
	result, err := c.GenerateMessages(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	messageChan := make(chan schema.Message, 1)
	messageChan <- result[0]
	close(messageChan)

	return messageChan, nil
}

// GetModelInfo implements the ModelInfoProvider interface.
func (c *ChatModelAdapter) GetModelInfo() ModelInfo {
	return ModelInfo{
		Name:         c.llm.GetModelName(),
		Provider:     c.llm.GetProviderName(),
		Version:      "1.0",
		MaxTokens:    4096, // Default, could be made configurable
		Capabilities: []string{"chat", "generation"},
	}
}

// CheckHealth implements the HealthChecker interface.
func (c *ChatModelAdapter) CheckHealth() map[string]any {
	return map[string]any{
		"state":     "healthy",
		"model":     c.llm.GetModelName(),
		"provider":  c.llm.GetProviderName(),
		"timestamp": time.Now().Unix(),
	}
}

// Invoke implements the Runnable interface.
func (c *ChatModelAdapter) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	result, err := c.GenerateMessages(ctx, messages, options...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Batch implements the Runnable interface.
func (c *ChatModelAdapter) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := c.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

// Stream implements the Runnable interface.
func (c *ChatModelAdapter) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	messageChan, err := c.StreamMessages(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	// Convert message channel to any channel
	anyChan := make(chan any)
	go func() {
		defer close(anyChan)
		for msg := range messageChan {
			anyChan <- msg
		}
	}()

	return anyChan, nil
}

// messagesToPrompt converts a slice of messages to a single prompt string.
func (c *ChatModelAdapter) messagesToPrompt(messages []schema.Message) string {
	var prompt strings.Builder

	for i, msg := range messages {
		if i > 0 {
			_, _ = prompt.WriteString("\n\n") //nolint:errcheck // strings.Builder.WriteString rarely fails
		}

		switch m := msg.(type) {
		case *schema.ChatMessage:
			if m.GetType() == schema.RoleSystem {
				_, _ = prompt.WriteString("System: ") //nolint:errcheck // strings.Builder.WriteString rarely fails
			} else if m.GetType() == schema.RoleHuman {
				_, _ = prompt.WriteString("Human: ") //nolint:errcheck // strings.Builder.WriteString rarely fails
			} else if m.GetType() == schema.RoleAssistant {
				_, _ = prompt.WriteString("Assistant: ") //nolint:errcheck // strings.Builder.WriteString rarely fails
			}
			_, _ = prompt.WriteString(m.GetContent()) //nolint:errcheck // strings.Builder.WriteString rarely fails
		case *schema.AIMessage:
			_, _ = prompt.WriteString("Assistant: ")  //nolint:errcheck // strings.Builder.WriteString rarely fails
			_, _ = prompt.WriteString(m.GetContent()) //nolint:errcheck // strings.Builder.WriteString rarely fails
		default:
			_, _ = prompt.WriteString(msg.GetContent()) //nolint:errcheck // strings.Builder.WriteString rarely fails
		}
	}

	// Add final assistant prompt if not already there
	if !strings.HasSuffix(prompt.String(), "\n\nAssistant: ") {
		_, _ = prompt.WriteString("\n\nAssistant: ") //nolint:errcheck // strings.Builder.WriteString rarely fails
	}

	return prompt.String()
}
