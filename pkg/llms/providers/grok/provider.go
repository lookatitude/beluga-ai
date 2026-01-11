// Package grok provides an implementation of the llms.ChatModel interface
// using the xAI Grok API (OpenAI-compatible).
package grok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	openaiClient "github.com/sashabaranov/go-openai"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Provider constants.
const (
	ProviderName = "grok"
	DefaultModel = "grok-beta"

	// Error codes specific to Grok.
	ErrCodeInvalidAPIKey  = "grok_invalid_api_key"
	ErrCodeRateLimit      = "grok_rate_limit"
	ErrCodeModelNotFound  = "grok_model_not_found"
	ErrCodeInvalidRequest = "grok_invalid_request"
	ErrCodeQuotaExceeded  = "grok_quota_exceeded"
)

// GrokProvider implements the ChatModel interface for xAI Grok models.
type GrokProvider struct {
	metrics     llms.MetricsRecorder
	config      *llms.Config
	client      *openaiClient.Client
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
	modelName   string
	tools       []tools.Tool
}

// NewGrokProvider creates a new Grok provider instance.
func NewGrokProvider(config *llms.Config) (*GrokProvider, error) {
	// Validate configuration
	if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
		return nil, fmt.Errorf("invalid Grok configuration: %w", err)
	}

	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// Build client configuration
	// Grok uses OpenAI-compatible API, so we can use the same client
	grokConfig := openaiClient.DefaultConfig(config.APIKey)
	
	// Set base URL (default to xAI API endpoint)
	if config.BaseURL != "" {
		grokConfig.BaseURL = config.BaseURL
	} else {
		grokConfig.BaseURL = "https://api.x.ai/v1"
	}

	// Add organization if specified in provider-specific config
	if orgID, ok := config.ProviderSpecific["organization"].(string); ok && orgID != "" {
		grokConfig.OrgID = orgID
	}

	client := openaiClient.NewClientWithConfig(grokConfig)

	provider := &GrokProvider{
		config:    config,
		client:    client,
		modelName: modelName,
		metrics:   llms.GetMetrics(), // Get global metrics instance
		tracing:   common.NewTracingHelper(),
		retryConfig: &common.RetryConfig{
			MaxRetries: config.MaxRetries,
			Delay:      config.RetryDelay,
			Backoff:    config.RetryBackoff,
		},
	}

	return provider, nil
}

// Generate implements the ChatModel interface.
func (g *GrokProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = g.tracing.StartOperation(ctx, "grok.generate", ProviderName, g.modelName)

	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	g.tracing.AddSpanAttributes(ctx, map[string]any{"input_size": inputSize})

	start := time.Now()

	// Record request metrics
	g.metrics.IncrementActiveRequests(ctx, ProviderName, g.modelName)
	defer g.metrics.DecrementActiveRequests(ctx, ProviderName, g.modelName)

	// Apply options and merge with defaults
	callOpts := g.buildCallOptions(options...)

	// Execute with retry logic
	var result schema.Message
	var err error

	retryErr := common.RetryWithBackoff(ctx, g.retryConfig, "grok.generate", func() error {
		result, err = g.generateInternal(ctx, messages, callOpts)
		return err
	})

	if retryErr != nil {
		duration := time.Since(start)
		g.metrics.RecordError(ctx, ProviderName, g.modelName, llms.GetLLMErrorCode(retryErr), duration)
		g.tracing.RecordError(ctx, retryErr)
		return nil, retryErr
	}

	duration := time.Since(start)
	g.metrics.RecordRequest(ctx, ProviderName, g.modelName, duration)

	return result, nil
}

// StreamChat implements the ChatModel interface.
func (g *GrokProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	ctx = g.tracing.StartOperation(ctx, "grok.stream", ProviderName, g.modelName)

	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	g.tracing.AddSpanAttributes(ctx, map[string]any{"input_size": inputSize})

	// Apply options and merge with defaults
	callOpts := g.buildCallOptions(options...)

	// Execute streaming request
	return g.streamInternal(ctx, messages, callOpts)
}

// BindTools implements the ChatModel interface.
func (g *GrokProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	newProvider := *g // Create a copy
	newProvider.tools = make([]tools.Tool, len(toolsToBind))
	copy(newProvider.tools, toolsToBind)
	return &newProvider
}

// GetModelName implements the ChatModel interface.
func (g *GrokProvider) GetModelName() string {
	return g.modelName
}

// GetProviderName returns the provider name.
func (g *GrokProvider) GetProviderName() string {
	return ProviderName
}

// Invoke implements the Runnable interface.
func (g *GrokProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return g.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface.
func (g *GrokProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	errors := make([]error, len(inputs))

	// Use semaphore for concurrency control
	sem := make(chan struct{}, g.config.MaxConcurrentBatches)

	for i, input := range inputs {
		sem <- struct{}{} // Acquire semaphore

		go func(index int, currentInput any) {
			defer func() { <-sem }() // Release semaphore

			result, err := g.Invoke(ctx, currentInput, options...)
			results[index] = result
			errors[index] = err
		}(i, input)
	}

	// Wait for all goroutines to complete
	for i := 0; i < g.config.MaxConcurrentBatches; i++ {
		sem <- struct{}{}
	}

	// Check for errors
	var combinedErr error
	for _, err := range errors {
		if err != nil {
			if combinedErr == nil {
				combinedErr = err
			} else {
				combinedErr = fmt.Errorf("%w; %w", combinedErr, err)
			}
		}
	}

	return results, combinedErr
}

// Stream implements the Runnable interface.
func (g *GrokProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := g.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	// Convert AIMessageChunk channel to any channel
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

// generateInternal performs the actual generation logic.
func (g *GrokProvider) generateInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (schema.Message, error) {
	// Convert messages to OpenAI format (Grok uses OpenAI-compatible API)
	openaiMessages, err := g.convertMessages(messages)
	if err != nil {
		return nil, llms.NewLLMError("generateInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Build request parameters
	req := g.buildGrokRequest(openaiMessages, opts)

	// Make API call
	resp, err := g.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, g.handleGrokError("generateInternal", err)
	}

	// Convert response to schema.Message
	return g.convertGrokResponse(&resp)
}

// streamInternal performs the actual streaming logic.
func (g *GrokProvider) streamInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (<-chan iface.AIMessageChunk, error) {
	// Convert messages to OpenAI format
	openaiMessages, err := g.convertMessages(messages)
	if err != nil {
		return nil, llms.NewLLMError("streamInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Build request parameters
	req := g.buildGrokRequest(openaiMessages, opts)
	req.Stream = true

	// Create streaming response
	stream, err := g.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, g.handleGrokError("streamInternal", err)
	}

	outputChan := make(chan iface.AIMessageChunk)

	go func() {
		defer close(outputChan)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "stream closed" {
					break
				}
				finalChunk := iface.AIMessageChunk{
					Err: llms.WrapError("grok.stream", err),
				}
				select {
				case outputChan <- finalChunk:
				case <-ctx.Done():
				}
				return
			}

			// Convert stream response to AIMessageChunk
			chunk, err := g.convertGrokStreamResponse(&response)
			if err != nil {
				finalChunk := iface.AIMessageChunk{
					Err: llms.WrapError("grok.stream.convert", err),
				}
				select {
				case outputChan <- finalChunk:
				case <-ctx.Done():
				}
				return
			}

			if chunk != nil {
				select {
				case outputChan <- *chunk:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outputChan, nil
}

// convertMessages converts schema messages to OpenAI format.
func (g *GrokProvider) convertMessages(messages []schema.Message) ([]openaiClient.ChatCompletionMessage, error) {
	openaiMessages := make([]openaiClient.ChatCompletionMessage, 0, len(messages))

	for _, msg := range messages {
		var openaiMsg openaiClient.ChatCompletionMessage

		switch m := msg.(type) {
		case *schema.ChatMessage:
			switch m.GetType() {
			case schema.RoleSystem:
				openaiMsg.Role = openaiClient.ChatMessageRoleSystem
				openaiMsg.Content = m.GetContent()
			case schema.RoleHuman:
				openaiMsg.Role = openaiClient.ChatMessageRoleUser
				openaiMsg.Content = m.GetContent()
			case schema.RoleAssistant:
				openaiMsg.Role = openaiClient.ChatMessageRoleAssistant
				openaiMsg.Content = m.GetContent()
			default:
				continue // Skip unknown roles
			}
		case *schema.AIMessage:
			openaiMsg.Role = openaiClient.ChatMessageRoleAssistant
			openaiMsg.Content = m.GetContent()

			// Add tool calls if present
			if len(m.ToolCalls()) > 0 {
				openaiMsg.ToolCalls = make([]openaiClient.ToolCall, len(m.ToolCalls()))
				for i, tc := range m.ToolCalls() {
					openaiMsg.ToolCalls[i] = openaiClient.ToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: openaiClient.FunctionCall{
							Name:      tc.Name,
							Arguments: tc.Arguments,
						},
					}
				}
			}
		default:
			continue // Skip unknown message types
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	if len(openaiMessages) == 0 {
		return nil, errors.New("no valid messages provided for Grok conversion")
	}

	return openaiMessages, nil
}

// buildGrokRequest builds the Grok API request (OpenAI-compatible format).
func (g *GrokProvider) buildGrokRequest(messages []openaiClient.ChatCompletionMessage, opts *llms.CallOptions) openaiClient.ChatCompletionRequest {
	req := openaiClient.ChatCompletionRequest{
		Model:    g.modelName,
		Messages: messages,
	}

	// Apply call options
	if opts.MaxTokens != nil {
		req.MaxTokens = *opts.MaxTokens
	}
	if opts.Temperature != nil {
		req.Temperature = *opts.Temperature
	}
	if opts.TopP != nil {
		req.TopP = *opts.TopP
	}
	if len(opts.StopSequences) > 0 {
		req.Stop = opts.StopSequences
	}
	if opts.FrequencyPenalty != nil {
		req.FrequencyPenalty = *opts.FrequencyPenalty
	}
	if opts.PresencePenalty != nil {
		req.PresencePenalty = *opts.PresencePenalty
	}

	// Add tools if bound
	if len(g.tools) > 0 {
		req.Tools = g.convertTools(g.tools)
	}

	return req
}

// convertTools converts tools to OpenAI format.
func (g *GrokProvider) convertTools(tools []tools.Tool) []openaiClient.Tool {
	if len(tools) == 0 {
		return nil
	}

	openaiTools := make([]openaiClient.Tool, 0, len(tools))
	for _, tool := range tools {
		def := tool.Definition()

		openaiTool := openaiClient.Tool{
			Type: "function",
			Function: &openaiClient.FunctionDefinition{
				Name:        def.Name,
				Description: def.Description,
			},
		}

		// Add parameters schema
		if def.InputSchema != nil {
			if schemaStr, ok := def.InputSchema.(string); ok && schemaStr != "" {
				var params map[string]any
				if err := json.Unmarshal([]byte(schemaStr), &params); err == nil {
					openaiTool.Function.Parameters = params
				}
			}
		}

		openaiTools = append(openaiTools, openaiTool)
	}

	return openaiTools
}

// convertGrokResponse converts Grok response to schema.Message.
func (g *GrokProvider) convertGrokResponse(resp *openaiClient.ChatCompletionResponse) (schema.Message, error) {
	if len(resp.Choices) == 0 {
		return nil, errors.New("empty response from Grok")
	}

	choice := resp.Choices[0]
	responseText := choice.Message.Content

	aiMsg := schema.NewAIMessage(responseText)

	// Add tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]schema.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			toolCalls[i] = schema.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}

		if aiMsgInternal, ok := aiMsg.(*schema.AIMessage); ok {
			aiMsgInternal.ToolCalls_ = toolCalls
		}
	}

	// Add usage information
	if resp.Usage.PromptTokens > 0 || resp.Usage.CompletionTokens > 0 {
		args := aiMsg.AdditionalArgs()
		args["usage"] = map[string]int{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
			"total_tokens":  resp.Usage.TotalTokens,
		}
	}

	return aiMsg, nil
}

// convertGrokStreamResponse converts Grok stream response to AIMessageChunk.
func (g *GrokProvider) convertGrokStreamResponse(resp *openaiClient.ChatCompletionStreamResponse) (*iface.AIMessageChunk, error) {
	if len(resp.Choices) == 0 {
		return nil, nil
	}

	choice := resp.Choices[0]

	chunk := &iface.AIMessageChunk{
		Content:        choice.Delta.Content,
		AdditionalArgs: make(map[string]any),
	}

	// Add finish reason if present
	if choice.FinishReason != "" {
		chunk.AdditionalArgs["finish_reason"] = choice.FinishReason
	}

	return chunk, nil
}

// buildCallOptions merges configuration options with call-specific options.
func (g *GrokProvider) buildCallOptions(options ...core.Option) *llms.CallOptions {
	callOpts := llms.NewCallOptions()

	// Apply default configuration
	if g.config.MaxTokens != nil {
		callOpts.MaxTokens = g.config.MaxTokens
	}
	if g.config.Temperature != nil {
		temp := float32(*g.config.Temperature)
		callOpts.Temperature = &temp
	}
	if g.config.TopP != nil {
		topP := float32(*g.config.TopP)
		callOpts.TopP = &topP
	}
	if g.config.FrequencyPenalty != nil {
		freq := float32(*g.config.FrequencyPenalty)
		callOpts.FrequencyPenalty = &freq
	}
	if g.config.PresencePenalty != nil {
		pres := float32(*g.config.PresencePenalty)
		callOpts.PresencePenalty = &pres
	}
	if len(g.config.StopSequences) > 0 {
		callOpts.StopSequences = g.config.StopSequences
	}

	// Apply call-specific options
	for _, opt := range options {
		callOpts.ApplyCallOption(opt)
	}

	return callOpts
}

// handleGrokError converts Grok errors to LLM errors.
func (g *GrokProvider) handleGrokError(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Try to extract more specific error information from Grok error
	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeRateLimit
		message = "Grok API rate limit exceeded"
	} else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") {
		errorCode = ErrCodeInvalidAPIKey
		message = "Grok API authentication failed"
	} else if strings.Contains(errStr, "model") || strings.Contains(errStr, "404") {
		errorCode = ErrCodeModelNotFound
		message = "Grok model not found"
	} else if strings.Contains(errStr, "quota") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeQuotaExceeded
		message = "Grok API quota exceeded"
	} else {
		errorCode = ErrCodeInvalidRequest
		message = "Grok API request failed"
	}

	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// CheckHealth implements the HealthChecker interface.
func (g *GrokProvider) CheckHealth() map[string]any {
	return map[string]any{
		"state":       "healthy",
		"provider":    "grok",
		"model":       g.modelName,
		"timestamp":   time.Now().Unix(),
		"api_key_set": g.config.APIKey != "",
		"tools_count": len(g.tools),
	}
}

// Factory function for creating Grok providers.
func NewGrokProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewGrokProvider(config)
	}
}
