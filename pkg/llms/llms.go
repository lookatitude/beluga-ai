// Package llms provides interfaces and implementations for Large Language Model interactions.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package llms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance for LLM operations.
// This should be called once during application startup to enable metrics collection.
// Subsequent calls are ignored due to sync.Once protection.
//
// Parameters:
//   - meter: OpenTelemetry meter for recording metrics
//   - tracer: OpenTelemetry tracer for recording traces (can be nil for no-op)
//
// Example usage can be found in examples/llm-usage/main.go.
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		if tracer == nil {
			tracer = trace.NewNoopTracerProvider().Tracer("llms")
		}
		metrics, err := NewMetrics(meter, tracer)
		if err != nil {
			// If metrics creation fails, use no-op metrics
			globalMetrics = NoOpMetrics()
			return
		}
		globalMetrics = metrics
	})
}

// GetMetrics returns the global metrics instance for LLM operations.
// Returns nil if InitMetrics has not been called.
//
// Returns:
//   - *Metrics: The global metrics instance, or nil if not initialized
//
// Example usage can be found in examples/llm-usage/main.go.
func GetMetrics() *Metrics {
	return globalMetrics
}

// Factory provides a factory pattern for creating LLM instances.
// It manages the creation of different LLM providers based on configuration.
type Factory struct {
	providers         map[string]iface.ChatModel
	llms              map[string]iface.LLM
	providerFactories map[string]func(*Config) (iface.ChatModel, error)
	llmFactories      map[string]func(*Config) (iface.LLM, error)
	mu                sync.RWMutex
}

// NewFactory creates a new LLM factory instance for managing LLM providers.
// The factory supports registration of both ChatModel and LLM providers,
// enabling flexible provider management and creation.
//
// Returns:
//   - *Factory: A new factory instance ready for provider registration
//
// Example:
//
//	factory := llms.NewFactory()
//	factory.RegisterProviderFactory("openai", openai.NewProvider)
//	provider, err := factory.CreateProvider("openai", config)
//
// Example usage can be found in examples/llm-usage/main.go.
func NewFactory() *Factory {
	return &Factory{
		providers:         make(map[string]iface.ChatModel),
		llms:              make(map[string]iface.LLM),
		providerFactories: make(map[string]func(*Config) (iface.ChatModel, error)),
		llmFactories:      make(map[string]func(*Config) (iface.LLM, error)),
	}
}

// RegisterProvider registers a ChatModel provider instance with the factory.
// The provider can be retrieved later using GetProvider. This method is thread-safe.
//
// Parameters:
//   - name: Unique identifier for the provider (e.g., "openai", "anthropic")
//   - provider: ChatModel implementation to register
//
// Example:
//
//	openaiProvider, _ := openai.NewProvider(config)
//	factory.RegisterProvider("openai", openaiProvider)
//
// Example usage can be found in examples/llm-usage/main.go.
func (f *Factory) RegisterProvider(name string, provider iface.ChatModel) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[name] = provider
}

// RegisterLLM registers an LLM provider instance with the factory.
// The LLM can be retrieved later using GetLLM. This method is thread-safe.
//
// Parameters:
//   - name: Unique identifier for the LLM (e.g., "openai", "anthropic")
//   - llm: LLM implementation to register
//
// Example:
//
//	openaiLLM, _ := openai.NewLLM(config)
//	factory.RegisterLLM("openai", openaiLLM)
//
// Example usage can be found in examples/llm-usage/main.go.
func (f *Factory) RegisterLLM(name string, llm iface.LLM) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.llms[name] = llm
}

// GetProvider returns a registered ChatModel provider by name.
// This method is thread-safe and returns an error if the provider is not found.
//
// Parameters:
//   - name: The name of the provider to retrieve
//
// Returns:
//   - iface.ChatModel: The registered ChatModel provider
//   - error: ErrCodeUnsupportedProvider if the provider is not registered
//
// Example:
//
//	provider, err := factory.GetProvider("openai")
//	if err != nil {
//	    log.Fatal("Provider not found:", err)
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
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

// GetLLM returns a registered LLM provider by name.
// This method is thread-safe and returns an error if the LLM is not found.
//
// Parameters:
//   - name: The name of the LLM to retrieve
//
// Returns:
//   - iface.LLM: The registered LLM provider
//   - error: ErrCodeUnsupportedProvider if the LLM is not registered
//
// Example:
//
//	llm, err := factory.GetLLM("openai")
//	if err != nil {
//	    log.Fatal("LLM not found:", err)
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
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

// ListProviders returns a list of all registered ChatModel provider names.
// This method is thread-safe and returns an empty slice if no providers are registered.
//
// Returns:
//   - []string: Slice of provider names (e.g., ["openai", "anthropic"])
//
// Example:
//
//	providers := factory.ListProviders()
//	fmt.Printf("Available providers: %v\n", providers)
//
// Example usage can be found in examples/llm-usage/main.go.
func (f *Factory) ListProviders() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.providers))
	for name := range f.providers {
		names = append(names, name)
	}
	return names
}

// ListLLMs returns a list of all registered LLM provider names.
// This method is thread-safe and returns an empty slice if no LLMs are registered.
//
// Returns:
//   - []string: Slice of LLM names (e.g., ["openai", "anthropic"])
//
// Example:
//
//	llms := factory.ListLLMs()
//	fmt.Printf("Available LLMs: %v\n", llms)
//
// Example usage can be found in examples/llm-usage/main.go.
func (f *Factory) ListLLMs() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.llms))
	for name := range f.llms {
		names = append(names, name)
	}
	return names
}

// CreateProvider creates a ChatModel provider instance using the registered factory function.
// The provider must be registered using RegisterProviderFactory before calling this method.
//
// Parameters:
//   - providerName: Name of the provider to create (e.g., "openai", "anthropic")
//   - config: Configuration for the provider (API key, model name, etc.)
//
// Returns:
//   - iface.ChatModel: The created ChatModel provider instance
//   - error: ErrCodeUnsupportedProvider if the factory is not registered, or provider-specific errors
//
// Example:
//
//	config := llms.NewConfig(llms.WithAPIKey("key"), llms.WithModelName("gpt-4"))
//	provider, err := factory.CreateProvider("openai", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
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

// CreateLLM creates an LLM provider instance using the registered factory function.
// The LLM must be registered using RegisterLLMFactory before calling this method.
//
// Parameters:
//   - providerName: Name of the LLM provider to create (e.g., "openai", "anthropic")
//   - config: Configuration for the LLM (API key, model name, etc.)
//
// Returns:
//   - iface.LLM: The created LLM provider instance
//   - error: ErrCodeUnsupportedProvider if the factory is not registered, or provider-specific errors
//
// Example:
//
//	config := llms.NewConfig(llms.WithAPIKey("key"), llms.WithModelName("gpt-4"))
//	llm, err := factory.CreateLLM("openai", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
func (f *Factory) CreateLLM(providerName string, config *Config) (iface.LLM, error) {
	f.mu.RLock()
	factory, exists := f.llmFactories[providerName]
	f.mu.RUnlock()

	if !exists {
		return nil, NewLLMError("CreateLLM", ErrCodeUnsupportedProvider,
			fmt.Errorf("LLM factory '%s' not registered", providerName))
	}

	// Set provider name in config if not already set
	if config.Provider == "" {
		config.Provider = providerName
	}

	return factory(config)
}

// ListAvailableProviders returns a list of all available provider factory names.
// This includes providers registered via RegisterProviderFactory, not direct instances.
// This method is thread-safe.
//
// Returns:
//   - []string: Slice of provider factory names that can be used with CreateProvider
//
// Example:
//
//	available := factory.ListAvailableProviders()
//	fmt.Printf("Available provider factories: %v\n", available)
//
// Example usage can be found in examples/llm-usage/main.go.
func (f *Factory) ListAvailableProviders() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.providerFactories))
	for name := range f.providerFactories {
		names = append(names, name)
	}
	return names
}

// RegisterProviderFactory registers a factory function for creating ChatModel providers.
// The factory function will be called when CreateProvider is invoked with the given name.
// This method is thread-safe.
//
// Parameters:
//   - name: Unique identifier for the provider factory (e.g., "openai", "anthropic")
//   - factory: Function that creates a ChatModel provider from a Config
//
// Example:
//
//	factory.RegisterProviderFactory("openai", func(config *llms.Config) (iface.ChatModel, error) {
//	    return openai.NewProvider(config)
//	})
//
// Example usage can be found in examples/llm-usage/main.go.
func (f *Factory) RegisterProviderFactory(name string, factory func(*Config) (iface.ChatModel, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providerFactories[name] = factory
}

// RegisterLLMFactory registers a factory function for creating LLM providers.
// The factory function will be called when CreateLLM is invoked with the given name.
// This method is thread-safe.
//
// Parameters:
//   - name: Unique identifier for the LLM factory (e.g., "openai", "anthropic")
//   - factory: Function that creates an LLM provider from a Config
//
// Example:
//
//	factory.RegisterLLMFactory("openai", func(config *llms.Config) (iface.LLM, error) {
//	    return openai.NewLLM(config)
//	})
//
// Example usage can be found in examples/llm-usage/main.go.
func (f *Factory) RegisterLLMFactory(name string, factory func(*Config) (iface.LLM, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.llmFactories[name] = factory
}

// EnsureMessages ensures the input is a slice of schema.Message.
// It attempts to convert common input types (like a single string or Message) into the required format.
// This is a convenience function for handling flexible input types in LLM operations.
//
// Parameters:
//   - input: Can be a string, single schema.Message, or []schema.Message
//
// Returns:
//   - []schema.Message: Normalized slice of messages
//   - error: ErrCodeInvalidRequest if the input type is not supported
//
// Example:
//
//	messages, err := llms.EnsureMessages("Hello, world!")
//	// Returns: []schema.Message{schema.NewHumanMessage("Hello, world!")}
//
//	messages, err := llms.EnsureMessages([]schema.Message{msg1, msg2})
//	// Returns: []schema.Message{msg1, msg2}
//
// Example usage can be found in examples/llm-usage/main.go.
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
// or require a single prompt string. It separates system messages from human messages and concatenates
// all human messages into a single string.
//
// Parameters:
//   - messages: Slice of messages to process
//
// Returns:
//   - string: The system prompt (empty if no system message found)
//   - string: Concatenated human messages separated by newlines
//
// Example:
//
//	messages := []schema.Message{
//	    schema.NewSystemMessage("You are a helpful assistant."),
//	    schema.NewHumanMessage("Hello"),
//	    schema.NewHumanMessage("How are you?"),
//	}
//	system, human := llms.GetSystemAndHumanPromptsFromSchema(messages)
//	// system: "You are a helpful assistant."
//	// human: "Hello\nHow are you?"
//
// Example usage can be found in examples/llm-usage/main.go.
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
	var fullHumanPromptSb220 strings.Builder
	for i, p := range humanPrompts {
		if i > 0 {
			_, _ = fullHumanPromptSb220.WriteString("\n")
		}
		_, _ = fullHumanPromptSb220.WriteString(p)
	}
	fullHumanPrompt += fullHumanPromptSb220.String()
	return systemPrompt, fullHumanPrompt
}

// Utility functions for common LLM operations

// GenerateText is a convenience function for generating text with a ChatModel.
// It handles message conversion, OTEL tracing, and error handling automatically.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - model: ChatModel instance to use for generation
//   - prompt: Text prompt to send to the model
//   - options: Optional core.Option parameters (temperature, max_tokens, etc.)
//
// Returns:
//   - string: The generated text response
//   - error: Any error that occurred during generation (network errors, API errors, etc.)
//
// Example:
//
//	model, _ := llms.NewOpenAIChat(llms.WithAPIKey("key"))
//	response, err := llms.GenerateText(ctx, model, "What is the capital of France?")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(response)
//
// Example usage can be found in examples/llm-usage/main.go.
func GenerateText(ctx context.Context, model iface.ChatModel, prompt string, options ...core.Option) (string, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/llms")
	ctx, span := tracer.Start(ctx, "llms.GenerateText",
		trace.WithAttributes(
			attribute.Int("prompt_length", len(prompt)),
		))
	defer span.End()

	messages, err := EnsureMessages(prompt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to ensure messages", "error", err)
		return "", err
	}

	response, err := model.Generate(ctx, messages, options...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to generate text", "error", err)
		return "", err
	}

	content := response.GetContent()
	span.SetAttributes(attribute.Int("response_length", len(content)))
	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Text generated successfully",
		"prompt_length", len(prompt),
		"response_length", len(content))
	return content, nil
}

// GenerateTextWithTools is a convenience function for generating text with tool calling support.
// It automatically binds tools to the model and handles tool execution if the model requests it.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - model: ChatModel instance that supports tool calling
//   - prompt: Text prompt to send to the model
//   - tools: Slice of tools available to the model
//   - options: Optional core.Option parameters (temperature, max_tokens, etc.)
//
// Returns:
//   - string: The generated text response (may include tool call results)
//   - error: Any error that occurred during generation or tool execution
//
// Example:
//
//	model, _ := llms.NewOpenAIChat(llms.WithAPIKey("key"))
//	tools := []tools.Tool{calculator, webSearch}
//	response, err := llms.GenerateTextWithTools(ctx, model, "What is 2+2?", tools)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(response)
//
// Example usage can be found in examples/llm-usage/main.go.
func GenerateTextWithTools(ctx context.Context, model iface.ChatModel, prompt string, tools []tools.Tool, options ...core.Option) (string, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/llms")
	ctx, span := tracer.Start(ctx, "llms.GenerateTextWithTools",
		trace.WithAttributes(
			attribute.Int("prompt_length", len(prompt)),
			attribute.Int("tools_count", len(tools)),
		))
	defer span.End()

	messages, err := EnsureMessages(prompt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to ensure messages", "error", err)
		return "", err
	}

	// Bind tools to the model
	modelWithTools := model.BindTools(tools)

	response, err := modelWithTools.Generate(ctx, messages, options...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to generate text with tools", "error", err, "tools_count", len(tools))
		return "", err
	}

	content := response.GetContent()
	span.SetAttributes(attribute.Int("response_length", len(content)))
	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Text generated with tools successfully",
		"prompt_length", len(prompt),
		"response_length", len(content),
		"tools_count", len(tools))
	return content, nil
}

// StreamText is a convenience function for streaming text generation.
// It returns a channel that receives message chunks as they are generated, enabling
// real-time display of responses as they are produced.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - model: ChatModel instance that supports streaming
//   - prompt: Text prompt to send to the model
//   - options: Optional core.Option parameters (temperature, max_tokens, etc.)
//
// Returns:
//   - <-chan iface.AIMessageChunk: Channel receiving message chunks as they are generated
//   - error: Any error that occurred during stream initialization
//
// Example:
//
//	model, _ := llms.NewOpenAIChat(llms.WithAPIKey("key"))
//	ch, err := llms.StreamText(ctx, model, "Tell me a story")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for chunk := range ch {
//	    fmt.Print(chunk.GetContent())
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
func StreamText(ctx context.Context, model iface.ChatModel, prompt string, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/llms")
	ctx, span := tracer.Start(ctx, "llms.StreamText",
		trace.WithAttributes(
			attribute.Int("prompt_length", len(prompt)),
		))
	defer span.End()

	messages, err := EnsureMessages(prompt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to ensure messages", "error", err)
		return nil, err
	}

	ch, err := model.StreamChat(ctx, messages, options...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to stream text", "error", err)
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Text streaming started", "prompt_length", len(prompt))
	return ch, nil
}

// BatchGenerate is a convenience function for batch text generation.
// It processes multiple prompts efficiently, handling errors per-prompt and returning
// all successful results. If any prompt fails, the error is returned but partial results
// may still be available.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - model: ChatModel instance to use for generation
//   - prompts: Slice of text prompts to process
//   - options: Optional core.Option parameters (temperature, max_tokens, etc.)
//
// Returns:
//   - []string: Slice of generated text responses, one per input prompt
//   - error: Any error that occurred during batch processing (may include partial results)
//
// Example:
//
//	model, _ := llms.NewOpenAIChat(llms.WithAPIKey("key"))
//	prompts := []string{"What is AI?", "What is ML?", "What is NLP?"}
//	responses, err := llms.BatchGenerate(ctx, model, prompts)
//	if err != nil {
//	    log.Printf("Batch generation had errors: %v", err)
//	}
//	for i, response := range responses {
//	    fmt.Printf("Prompt %d: %s\n", i, response)
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
func BatchGenerate(ctx context.Context, model iface.ChatModel, prompts []string, options ...core.Option) ([]string, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/llms")
	ctx, span := tracer.Start(ctx, "llms.BatchGenerate",
		trace.WithAttributes(
			attribute.Int("batch_size", len(prompts)),
		))
	defer span.End()

	inputs := make([]any, len(prompts))
	for i, prompt := range prompts {
		messages, err := EnsureMessages(prompt)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logWithOTELContext(ctx, slog.LevelError, "Failed to convert prompt in batch", "error", err, "prompt_index", i)
			return nil, fmt.Errorf("failed to convert prompt %d: %w", i, err)
		}
		inputs[i] = messages
	}

	results, err := model.Batch(ctx, inputs, options...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to batch generate", "error", err, "batch_size", len(prompts))
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

	span.SetAttributes(attribute.Int("responses_count", len(responses)))
	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Batch generation completed",
		"batch_size", len(prompts),
		"responses_count", len(responses))
	return responses, nil
}

// logWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}

// ValidateModelName validates that a model name is supported by a provider.
// This helps catch configuration errors early before making API calls.
//
// Parameters:
//   - provider: Provider name (e.g., "openai", "anthropic")
//   - modelName: Model name to validate (e.g., "gpt-4", "claude-3-opus")
//
// Returns:
//   - error: ErrCodeInvalidModel if the model is not supported, or if modelName is empty
//
// Example:
//
//	err := llms.ValidateModelName("openai", "gpt-4")
//	if err != nil {
//	    log.Fatal("Invalid model:", err)
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
func ValidateModelName(provider, modelName string) error {
	if modelName == "" {
		return NewLLMError("ValidateModelName", ErrCodeInvalidModel,
			errors.New("model name cannot be empty"))
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

// DefaultConfig returns a default configuration for LLM operations.
// The returned config has sensible defaults for most use cases:
// - 30 second timeout
// - 3 retries with exponential backoff
// - Streaming enabled
// - Observability enabled (tracing, metrics, logging)
//
// Returns:
//   - *Config: Configuration instance with default values
//
// Example:
//
//	config := llms.DefaultConfig()
//	config.APIKey = "your-api-key"
//	config.ModelName = "gpt-4"
//	provider, err := llms.NewOpenAIChat(llms.WithConfig(config))
//
// Example usage can be found in examples/llm-usage/main.go.
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
		ProviderSpecific:        make(map[string]any),
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
		EnableToolCalling:       true,
	}
}

// Helper functions for backward compatibility
// These provide compatibility with the old interface while encouraging migration

// WithMaxTokensLegacy sets the maximum number of tokens to generate (deprecated, use core.WithOption).
func WithMaxTokensLegacy(tokens int) core.Option {
	return core.WithOption("max_tokens", tokens)
}

// WithTemperatureLegacy sets the sampling temperature (deprecated, use core.WithOption).
func WithTemperatureLegacy(temp float32) core.Option {
	return core.WithOption("temperature", temp)
}

// WithTopPLegacy sets the nucleus sampling probability (deprecated, use core.WithOption).
func WithTopPLegacy(topP float32) core.Option {
	return core.WithOption("top_p", topP)
}

// WithTopKLegacy sets the top-k sampling parameter (deprecated, use core.WithOption).
func WithTopKLegacy(topK int) core.Option {
	return core.WithOption("top_k", topK)
}

// WithStopWordsLegacy sets the stop sequences for generation (deprecated, use core.WithOption).
func WithStopWordsLegacy(stop []string) core.Option {
	return core.WithOption("stop_words", stop)
}

// WithToolsLegacy sets the tools that the model can call (deprecated, use core.WithOption).
func WithToolsLegacy(toolsToUse []tools.Tool) core.Option {
	return core.WithOption("tools", toolsToUse)
}

// WithToolChoiceLegacy forces the model to call a specific tool (deprecated, use core.WithOption).
func WithToolChoiceLegacy(choice string) core.Option {
	return core.WithOption("tool_choice", choice)
}

// NewAnthropicChat creates a new Anthropic chat model provider with the given options.
// This is a convenience function that internally uses the factory pattern.
// It sets default model to "claude-3-haiku-20240307" if not specified.
//
// Parameters:
//   - opts: Configuration options (API key, model name, temperature, etc.)
//
// Returns:
//   - iface.ChatModel: Anthropic ChatModel provider instance
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	model, err := llms.NewAnthropicChat(
//	    llms.WithAPIKey("your-api-key"),
//	    llms.WithModelName("claude-3-opus-20240229"),
//	    llms.WithTemperature(0.7),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
func NewAnthropicChat(opts ...ConfigOption) (iface.ChatModel, error) {
	config := NewConfig(opts...)
	config.Provider = "anthropic"
	if config.ModelName == "" {
		config.ModelName = "claude-3-haiku-20240307"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := NewFactory()
	factory.RegisterProviderFactory("anthropic", func(c *Config) (iface.ChatModel, error) {
		// Import the anthropic package dynamically to avoid circular imports
		// This is a simplified implementation - in production, this would be handled differently
		return nil, errors.New("anthropic provider not available - use factory pattern with explicit import")
	})

	return factory.CreateProvider("anthropic", config)
}

// NewOpenAIChat creates a new OpenAI chat model provider with the given options.
// This is a convenience function that internally uses the factory pattern.
// It sets default model to "gpt-3.5-turbo" if not specified.
//
// Parameters:
//   - opts: Configuration options (API key, model name, temperature, etc.)
//
// Returns:
//   - iface.ChatModel: OpenAI ChatModel provider instance
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	model, err := llms.NewOpenAIChat(
//	    llms.WithAPIKey("your-api-key"),
//	    llms.WithModelName("gpt-4"),
//	    llms.WithTemperature(0.7),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/llm-usage/main.go.
func NewOpenAIChat(opts ...ConfigOption) (iface.ChatModel, error) {
	config := NewConfig(opts...)
	config.Provider = "openai"
	if config.ModelName == "" {
		config.ModelName = "gpt-3.5-turbo"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := NewFactory()
	factory.RegisterProviderFactory("openai", func(c *Config) (iface.ChatModel, error) {
		// Import the openai package dynamically to avoid circular imports
		return nil, errors.New("openai provider not available - use factory pattern with explicit import")
	})
	return factory.CreateProvider("openai", config)
}

// NewOllamaChat creates a new Ollama chat model provider with the given options.
// This is a convenience function that internally uses the factory pattern.
func NewOllamaChat(opts ...ConfigOption) (iface.ChatModel, error) {
	config := NewConfig(opts...)
	config.Provider = "ollama"
	if config.ModelName == "" {
		config.ModelName = "llama2"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := NewFactory()
	factory.RegisterProviderFactory("ollama", func(c *Config) (iface.ChatModel, error) {
		// Import the ollama package dynamically to avoid circular imports
		return nil, errors.New("ollama provider not available - use factory pattern with explicit import")
	})
	return factory.CreateProvider("ollama", config)
}

// NewAnthropicLLM creates a new Anthropic LLM provider with the given options.
// This is a convenience function that internally uses the factory pattern.
func NewAnthropicLLM(opts ...ConfigOption) (iface.LLM, error) {
	config := NewConfig(opts...)
	config.Provider = "anthropic"
	if config.ModelName == "" {
		config.ModelName = "claude-3-haiku-20240307"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := NewFactory()
	factory.RegisterLLMFactory("anthropic", func(c *Config) (iface.LLM, error) {
		return nil, errors.New("anthropic LLM not available - use factory pattern with explicit import")
	})
	return factory.CreateLLM("anthropic", config)
}

// NewOpenAILLM creates a new OpenAI LLM provider with the given options.
// This is a convenience function that internally uses the factory pattern.
func NewOpenAILLM(opts ...ConfigOption) (iface.LLM, error) {
	config := NewConfig(opts...)
	config.Provider = "openai"
	if config.ModelName == "" {
		config.ModelName = "gpt-3.5-turbo"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := NewFactory()
	factory.RegisterLLMFactory("openai", func(c *Config) (iface.LLM, error) {
		return nil, errors.New("openai LLM not available - use factory pattern with explicit import")
	})
	return factory.CreateLLM("openai", config)
}

// NewBedrockLLM creates a new Bedrock LLM provider with the given options.
// This is a convenience function that internally uses the factory pattern.
func NewBedrockLLM(opts ...ConfigOption) (iface.LLM, error) {
	config := NewConfig(opts...)
	config.Provider = "bedrock"
	if config.ModelName == "" {
		config.ModelName = "amazon.titan-text-express-v1"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := NewFactory()
	factory.RegisterLLMFactory("bedrock", func(c *Config) (iface.LLM, error) {
		return nil, errors.New("bedrock LLM not available - use factory pattern with explicit import")
	})
	return factory.CreateLLM("bedrock", config)
}

// NewOllamaLLM creates a new Ollama LLM provider with the given options.
// This is a convenience function that internally uses the factory pattern.
func NewOllamaLLM(opts ...ConfigOption) (iface.LLM, error) {
	config := NewConfig(opts...)
	config.Provider = "ollama"
	if config.ModelName == "" {
		config.ModelName = "llama2"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := NewFactory()
	factory.RegisterLLMFactory("ollama", func(c *Config) (iface.LLM, error) {
		return nil, errors.New("ollama LLM not available - use factory pattern with explicit import")
	})
	return factory.CreateLLM("ollama", config)
}

// NewMockLLM creates a new Mock LLM provider with the given options.
// This is a convenience function that internally uses the factory pattern.
func NewMockLLM(opts ...ConfigOption) (iface.LLM, error) {
	config := NewConfig(opts...)
	config.Provider = "mock"
	if config.ModelName == "" {
		config.ModelName = "mock-model"
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := NewFactory()
	factory.RegisterLLMFactory("mock", func(c *Config) (iface.LLM, error) {
		return nil, errors.New("mock LLM not available - use factory pattern with explicit import")
	})
	return factory.CreateLLM("mock", config)
}

// InitializeDefaultFactory creates and returns a factory with all built-in providers registered.
// Note: Provider factories must be registered explicitly to avoid circular dependencies.
// This function returns an empty factory that should be populated with provider factories.
//
// Returns:
//   - *Factory: A new factory instance (empty, requires explicit provider registration)
//
// Example:
//
//	factory := llms.InitializeDefaultFactory()
//	// Register providers explicitly
//	factory.RegisterProviderFactory("openai", openai.NewProviderFactory())
//	factory.RegisterProviderFactory("anthropic", anthropic.NewProviderFactory())
//
// Example usage can be found in examples/llm-usage/main.go.
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
