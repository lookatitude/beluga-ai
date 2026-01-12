// Package main demonstrates how to create and register a custom LLM provider
// with the Beluga AI framework. This example shows the complete pattern for
// implementing the ChatModel interface, adding OTEL instrumentation, and
// registering with the global provider registry.
//
// This is a production-ready example that you can use as a template for
// integrating any LLM API with Beluga AI.
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// =============================================================================
// Provider Constants
// =============================================================================

// ProviderName is the unique identifier for this provider in the registry.
// We use this when registering and when looking up the provider.
const ProviderName = "example-custom"

// DefaultModel is used when no model is specified in the configuration.
// For your custom provider, this would be whatever default model your API supports.
const DefaultModel = "example-model-v1"

// Error codes specific to this provider. Using provider-prefixed codes
// makes debugging easier when errors occur.
const (
	ErrCodeInvalidAPIKey  = "example_invalid_api_key"
	ErrCodeRateLimit      = "example_rate_limit"
	ErrCodeModelNotFound  = "example_model_not_found"
	ErrCodeInvalidRequest = "example_invalid_request"
)

// =============================================================================
// Provider Implementation
// =============================================================================

// CustomProvider implements the iface.ChatModel interface for a custom LLM.
// This struct holds all the state needed to communicate with your LLM API.
//
// Key design decisions:
// - We store the config for reference but don't modify it after creation
// - Metrics and tracing are injected for observability
// - Tools are stored separately to allow immutable BindTools behavior
type CustomProvider struct {
	config      *llms.Config
	metrics     *CustomMetrics
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
	modelName   string
	tools       []tools.Tool

	// In a real implementation, you'd have your API client here:
	// client *customapi.Client

	// For this example, we use simulated responses
	responses []string
	respIndex int
	mu        sync.Mutex
}

// CustomMetrics wraps OTEL metrics for this provider.
// Having provider-specific metrics gives you fine-grained observability.
type CustomMetrics struct {
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	errorCounter    metric.Int64Counter
	activeRequests  metric.Int64UpDownCounter
}

// NewCustomMetrics creates metrics for the custom provider.
// We follow the naming convention: beluga.{component}.{metric_name}
func NewCustomMetrics(meter metric.Meter) (*CustomMetrics, error) {
	requestCounter, err := meter.Int64Counter(
		"beluga.custom_provider.requests_total",
		metric.WithDescription("Total number of requests to custom LLM provider"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request counter: %w", err)
	}

	requestDuration, err := meter.Float64Histogram(
		"beluga.custom_provider.request_duration_seconds",
		metric.WithDescription("Request duration in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create duration histogram: %w", err)
	}

	errorCounter, err := meter.Int64Counter(
		"beluga.custom_provider.errors_total",
		metric.WithDescription("Total number of errors from custom LLM provider"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create error counter: %w", err)
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"beluga.custom_provider.active_requests",
		metric.WithDescription("Number of currently active requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active requests counter: %w", err)
	}

	return &CustomMetrics{
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		errorCounter:    errorCounter,
		activeRequests:  activeRequests,
	}, nil
}

// =============================================================================
// Provider Options
// =============================================================================

// ProviderOption is a functional option for configuring the provider.
// Using functional options gives us a clean, extensible API.
type ProviderOption func(*CustomProvider)

// WithSimulatedResponses sets canned responses for testing.
// In a real provider, you wouldn't have this - it's just for demonstration.
func WithSimulatedResponses(responses []string) ProviderOption {
	return func(p *CustomProvider) {
		p.responses = responses
	}
}

// =============================================================================
// Constructor
// =============================================================================

// NewCustomProvider creates a new custom LLM provider instance.
// This validates configuration and sets up all the components needed
// for the provider to function.
//
// Parameters:
//   - config: LLM configuration with API key, model name, etc.
//   - opts: Optional functional options for customization
//
// Returns:
//   - *CustomProvider: Ready-to-use provider instance
//   - error: Configuration validation errors
//
// Example:
//
//	config := &llms.Config{
//	    Provider:  "example-custom",
//	    ModelName: "example-model-v1",
//	    APIKey:    "your-api-key",
//	}
//	provider, err := NewCustomProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewCustomProvider(config *llms.Config, opts ...ProviderOption) (*CustomProvider, error) {
	// Validate configuration first - fail fast if something's wrong
	if config == nil {
		return nil, errors.New("configuration cannot be nil")
	}

	// For this example, we do minimal validation
	// In production, you'd validate API key, model name, etc.
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// Create OTEL meter for metrics
	meter := otel.Meter("github.com/lookatitude/beluga-ai/examples/llms/custom_provider")
	metrics, err := NewCustomMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	provider := &CustomProvider{
		config:    config,
		modelName: modelName,
		metrics:   metrics,
		tracing:   common.NewTracingHelper(),
		retryConfig: &common.RetryConfig{
			MaxRetries: config.MaxRetries,
			Delay:      config.RetryDelay,
			Backoff:    config.RetryBackoff,
		},
		// Default simulated responses
		responses: []string{
			"This is a response from the custom LLM provider.",
			"I'm a simulated response for demonstration purposes.",
			"Custom provider working correctly!",
		},
	}

	// Apply any functional options
	for _, opt := range opts {
		opt(provider)
	}

	return provider, nil
}

// =============================================================================
// ChatModel Interface Implementation
// =============================================================================

// Generate sends messages to the LLM and returns the response.
// This method includes:
// - OTEL tracing for visibility
// - Metrics recording for monitoring
// - Retry logic for resilience
// - Proper error handling
func (c *CustomProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start a tracing span to track this operation
	tracer := otel.Tracer("custom_provider")
	ctx, span := tracer.Start(ctx, "custom.Generate",
		trace.WithAttributes(
			attribute.String("provider", ProviderName),
			attribute.String("model", c.modelName),
			attribute.Int("message_count", len(messages)),
		),
	)
	defer span.End()

	// Calculate input size for metrics
	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	span.SetAttributes(attribute.Int("input_size", inputSize))

	start := time.Now()

	// Track active requests for load monitoring
	c.metrics.activeRequests.Add(ctx, 1)
	defer c.metrics.activeRequests.Add(ctx, -1)

	// Build call options from defaults and overrides
	callOpts := c.buildCallOptions(options...)

	// Execute with retry logic for resilience
	var result schema.Message
	var genErr error

	retryErr := common.RetryWithBackoff(ctx, c.retryConfig, "custom.generate", func() error {
		result, genErr = c.generateInternal(ctx, messages, callOpts)
		return genErr
	})

	duration := time.Since(start)

	if retryErr != nil {
		// Record error metrics
		c.metrics.errorCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("provider", ProviderName),
				attribute.String("model", c.modelName),
				attribute.String("error_code", getErrorCode(retryErr)),
			),
		)
		span.RecordError(retryErr)
		span.SetStatus(codes.Error, retryErr.Error())
		return nil, retryErr
	}

	// Record success metrics
	c.metrics.requestCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("provider", ProviderName),
			attribute.String("model", c.modelName),
			attribute.String("status", "success"),
		),
	)
	c.metrics.requestDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("provider", ProviderName),
			attribute.String("model", c.modelName),
		),
	)

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// generateInternal performs the actual generation logic.
// Separating this from Generate allows the retry wrapper to work cleanly.
func (c *CustomProvider) generateInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (schema.Message, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate we have messages
	if len(messages) == 0 {
		return nil, llms.NewLLMError("generateInternal", ErrCodeInvalidRequest,
			errors.New("no messages provided"))
	}

	// In a real implementation, you would:
	// 1. Convert messages to your API's format
	// 2. Make the API call
	// 3. Convert the response back

	// For this example, we return simulated responses
	c.mu.Lock()
	response := c.responses[c.respIndex%len(c.responses)]
	c.respIndex++
	c.mu.Unlock()

	// If tools are bound, simulate a tool call decision
	if len(c.tools) > 0 && shouldCallTool(messages) {
		return c.simulateToolCall(messages)
	}

	return schema.NewAIMessage(response), nil
}

// simulateToolCall creates a response with tool calls for demonstration.
func (c *CustomProvider) simulateToolCall(messages []schema.Message) (schema.Message, error) {
	if len(c.tools) == 0 {
		return schema.NewAIMessage("No tools available"), nil
	}

	// Get the first tool
	tool := c.tools[0]
	def := tool.Definition()

	// Create an AI message with a tool call
	aiMsg := schema.NewAIMessage("")
	if aiMsgTyped, ok := aiMsg.(*schema.AIMessage); ok {
		aiMsgTyped.ToolCalls_ = []schema.ToolCall{
			{
				ID:        "call_example_123",
				Name:      def.Name,
				Arguments: `{"example": "value"}`,
			},
		}
	}

	return aiMsg, nil
}

// shouldCallTool determines if the message should trigger a tool call.
// This is a simplified heuristic for demonstration.
func shouldCallTool(messages []schema.Message) bool {
	for _, msg := range messages {
		content := strings.ToLower(msg.GetContent())
		if strings.Contains(content, "calculate") ||
			strings.Contains(content, "search") ||
			strings.Contains(content, "look up") {
			return true
		}
	}
	return false
}

// StreamChat implements streaming responses.
// Each chunk is sent through the channel as it becomes available.
func (c *CustomProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	tracer := otel.Tracer("custom_provider")
	ctx, span := tracer.Start(ctx, "custom.StreamChat",
		trace.WithAttributes(
			attribute.String("provider", ProviderName),
			attribute.String("model", c.modelName),
		),
	)

	// Validate messages
	if len(messages) == 0 {
		span.End()
		return nil, llms.NewLLMError("StreamChat", ErrCodeInvalidRequest,
			errors.New("no messages provided"))
	}

	outputChan := make(chan iface.AIMessageChunk)

	go func() {
		defer close(outputChan)
		defer span.End()

		// Simulate streaming by breaking response into words
		c.mu.Lock()
		response := c.responses[c.respIndex%len(c.responses)]
		c.respIndex++
		c.mu.Unlock()

		words := strings.Fields(response)
		for i, word := range words {
			// Check for cancellation
			select {
			case <-ctx.Done():
				outputChan <- iface.AIMessageChunk{Err: ctx.Err()}
				return
			default:
			}

			// Add space before word (except first)
			content := word
			if i > 0 {
				content = " " + word
			}

			chunk := iface.AIMessageChunk{
				Content:        content,
				AdditionalArgs: make(map[string]any),
			}

			select {
			case outputChan <- chunk:
			case <-ctx.Done():
				return
			}

			// Simulate streaming delay
			time.Sleep(50 * time.Millisecond)
		}

		// Send final chunk with finish reason
		finalChunk := iface.AIMessageChunk{
			AdditionalArgs: map[string]any{
				"finish_reason": "stop",
			},
		}
		select {
		case outputChan <- finalChunk:
		case <-ctx.Done():
		}
	}()

	return outputChan, nil
}

// BindTools returns a new provider instance with the given tools attached.
// We return a copy to avoid mutating the original provider, which ensures
// thread safety and allows the same provider to be used with different tool sets.
func (c *CustomProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	// Create a shallow copy
	newProvider := *c

	// Deep copy the tools slice
	newProvider.tools = make([]tools.Tool, len(toolsToBind))
	copy(newProvider.tools, toolsToBind)

	return &newProvider
}

// GetModelName returns the model identifier being used.
func (c *CustomProvider) GetModelName() string {
	return c.modelName
}

// GetProviderName returns the provider identifier.
func (c *CustomProvider) GetProviderName() string {
	return ProviderName
}

// =============================================================================
// Runnable Interface Implementation
// =============================================================================

// Invoke implements the core.Runnable interface.
// It converts input to messages and calls Generate.
func (c *CustomProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return c.Generate(ctx, messages, options...)
}

// Batch implements the core.Runnable interface for batch processing.
// It processes multiple inputs concurrently with controlled parallelism.
func (c *CustomProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	errs := make([]error, len(inputs))

	// Use semaphore for concurrency control
	maxConcurrent := c.config.MaxConcurrentBatches
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	for i, input := range inputs {
		wg.Add(1)

		go func(index int, currentInput any) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := c.Invoke(ctx, currentInput, options...)
			results[index] = result
			errs[index] = err
		}(i, input)
	}

	wg.Wait()

	// Combine errors
	var combinedErr error
	for _, err := range errs {
		if err != nil {
			if combinedErr == nil {
				combinedErr = err
			} else {
				combinedErr = fmt.Errorf("%w; %v", combinedErr, err)
			}
		}
	}

	return results, combinedErr
}

// Stream implements the core.Runnable interface.
// It converts the typed chunk channel to a generic channel.
func (c *CustomProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := c.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	// Convert typed channel to generic channel
	outputChan := make(chan any)
	go func() {
		defer close(outputChan)
		for chunk := range chunkChan {
			select {
			case outputChan <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return outputChan, nil
}

// =============================================================================
// Health Check
// =============================================================================

// CheckHealth returns health status information for monitoring.
// This is useful for load balancers and health check endpoints.
func (c *CustomProvider) CheckHealth() map[string]any {
	return map[string]any{
		"state":       "healthy",
		"provider":    ProviderName,
		"model":       c.modelName,
		"timestamp":   time.Now().Unix(),
		"api_key_set": c.config.APIKey != "",
		"tools_count": len(c.tools),
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// buildCallOptions merges configuration defaults with call-specific options.
func (c *CustomProvider) buildCallOptions(options ...core.Option) *llms.CallOptions {
	callOpts := llms.NewCallOptions()

	// Apply configuration defaults
	if c.config.MaxTokens != nil {
		callOpts.MaxTokens = c.config.MaxTokens
	}
	if c.config.Temperature != nil {
		temp := float32(*c.config.Temperature)
		callOpts.Temperature = &temp
	}
	if c.config.TopP != nil {
		topP := float32(*c.config.TopP)
		callOpts.TopP = &topP
	}

	// Apply call-specific options (these override defaults)
	for _, opt := range options {
		callOpts.ApplyCallOption(opt)
	}

	return callOpts
}

// getErrorCode extracts an error code from an error.
func getErrorCode(err error) string {
	if llmErr, ok := err.(*llms.LLMError); ok {
		return llmErr.Code
	}
	return "unknown"
}

// =============================================================================
// Factory and Registration
// =============================================================================

// NewCustomProviderFactory returns a factory function for the registry.
// This is the function that gets registered and called when creating providers.
func NewCustomProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewCustomProvider(config)
	}
}

// RegisterCustomProvider registers the provider with the global registry.
// Call this during application initialization.
func RegisterCustomProvider() {
	llms.GetRegistry().Register(ProviderName, NewCustomProviderFactory())
}

// =============================================================================
// Main - Usage Example
// =============================================================================

func main() {
	ctx := context.Background()

	// Register our custom provider with the global registry
	RegisterCustomProvider()

	// Verify registration
	registry := llms.GetRegistry()
	fmt.Printf("Registered providers: %v\n", registry.ListProviders())

	// Create configuration
	config := llms.NewConfig(
		llms.WithProvider(ProviderName),
		llms.WithModelName("example-model-v1"),
		llms.WithAPIKey("example-api-key"),
	)

	// Get provider from registry
	provider, err := registry.GetProvider(ProviderName, config)
	if err != nil {
		fmt.Printf("Failed to get provider: %v\n", err)
		return
	}

	// Use the provider
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("Hello! What can you do?"),
	}

	// Generate a response
	fmt.Println("\n--- Generate Example ---")
	response, err := provider.Generate(ctx, messages)
	if err != nil {
		fmt.Printf("Generate error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", response.GetContent())

	// Streaming example
	fmt.Println("\n--- Streaming Example ---")
	fmt.Print("Streaming: ")
	streamChan, err := provider.StreamChat(ctx, messages)
	if err != nil {
		fmt.Printf("Stream error: %v\n", err)
		return
	}
	for chunk := range streamChan {
		if chunk.Err != nil {
			fmt.Printf("\nStream error: %v\n", chunk.Err)
			break
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println()

	// Tool binding example
	fmt.Println("\n--- Tool Binding Example ---")
	exampleTool := &ExampleTool{name: "calculator", description: "Performs calculations"}
	providerWithTools := provider.BindTools([]tools.Tool{exampleTool})

	toolMessages := []schema.Message{
		schema.NewHumanMessage("Please calculate 2 + 2"),
	}
	toolResponse, err := providerWithTools.Generate(ctx, toolMessages)
	if err != nil {
		fmt.Printf("Tool call error: %v\n", err)
		return
	}

	if aiMsg, ok := toolResponse.(*schema.AIMessage); ok && len(aiMsg.ToolCalls()) > 0 {
		fmt.Printf("Tool calls: %+v\n", aiMsg.ToolCalls())
	} else {
		fmt.Printf("Response: %s\n", toolResponse.GetContent())
	}

	// Health check
	fmt.Println("\n--- Health Check ---")
	health := provider.CheckHealth()
	fmt.Printf("Health: %+v\n", health)

	fmt.Println("\nCustom provider example completed successfully!")
}

// =============================================================================
// Example Tool for Demonstration
// =============================================================================

// ExampleTool is a simple tool implementation for testing.
type ExampleTool struct {
	name        string
	description string
}

func (t *ExampleTool) Definition() tools.Definition {
	return tools.Definition{
		Name:        t.name,
		Description: t.description,
		InputSchema: `{"type": "object", "properties": {"expression": {"type": "string"}}}`,
	}
}

func (t *ExampleTool) Execute(ctx context.Context, input string) (string, error) {
	return fmt.Sprintf("Executed %s with input: %s", t.name, input), nil
}
