// Package anthropic provides an implementation of the llms.ChatModel interface
// using the Anthropic API (Claude models).
package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Provider constants
const (
	ProviderName = "anthropic"
	DefaultModel = "claude-3-haiku-20240307"

	// Error codes specific to Anthropic
	ErrCodeInvalidAPIKey  = "anthropic_invalid_api_key"
	ErrCodeRateLimit      = "anthropic_rate_limit"
	ErrCodeModelNotFound  = "anthropic_model_not_found"
	ErrCodeInvalidRequest = "anthropic_invalid_request"
)

// AnthropicProvider implements the ChatModel interface for Anthropic Claude models
type AnthropicProvider struct {
	config      *llms.Config
	client      *anthropic.Client
	modelName   string
	tools       []tools.Tool
	metrics     llms.MetricsRecorder
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider(config *llms.Config) (*AnthropicProvider, error) {
	// Validate configuration
	if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
		return nil, fmt.Errorf("invalid Anthropic configuration: %w", err)
	}

	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// Build client options
	clientOpts := []option.RequestOption{}
	if config.APIKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(config.APIKey))
	}
	if config.BaseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(config.BaseURL))
	}

	// Add API version if specified in provider-specific config
	if apiVersion, ok := config.ProviderSpecific["api_version"].(string); ok && apiVersion != "" {
		clientOpts = append(clientOpts, option.WithHeader("anthropic-version", apiVersion))
	}

	client := anthropic.NewClient(clientOpts...)

	provider := &AnthropicProvider{
		config:    config,
		client:    &client,
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

// Generate implements the ChatModel interface
func (a *AnthropicProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = a.tracing.StartOperation(ctx, "anthropic.generate", ProviderName, a.modelName)

	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	a.tracing.AddSpanAttributes(ctx, map[string]interface{}{"input_size": inputSize})

	start := time.Now()

	// Record request metrics
	a.metrics.IncrementActiveRequests(ctx, ProviderName, a.modelName)
	defer a.metrics.DecrementActiveRequests(ctx, ProviderName, a.modelName)

	// Apply options and merge with defaults
	callOpts := a.buildCallOptions(options...)

	// Execute with retry logic
	var result schema.Message
	var err error

	retryErr := common.RetryWithBackoff(ctx, a.retryConfig, "anthropic.generate", func() error {
		result, err = a.generateInternal(ctx, messages, callOpts)
		return err
	})

	if retryErr != nil {
		duration := time.Since(start)
		a.metrics.RecordError(ctx, ProviderName, a.modelName, llms.GetLLMErrorCode(retryErr), duration)
		a.tracing.RecordError(ctx, retryErr)
		return nil, retryErr
	}

	duration := time.Since(start)
	a.metrics.RecordRequest(ctx, ProviderName, a.modelName, duration)

	return result, nil
}

// StreamChat implements the ChatModel interface
func (a *AnthropicProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	ctx = a.tracing.StartOperation(ctx, "anthropic.stream", ProviderName, a.modelName)

	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	a.tracing.AddSpanAttributes(ctx, map[string]interface{}{"input_size": inputSize})

	start := time.Now()

	// Apply options and merge with defaults
	callOpts := a.buildCallOptions(options...)

	// Execute streaming request
	streamChan, err := a.streamInternal(ctx, messages, callOpts)
	if err != nil {
		duration := time.Since(start)
		a.metrics.RecordError(ctx, ProviderName, a.modelName, llms.GetLLMErrorCode(err), duration)
		a.tracing.EndSpan(ctx)
		return nil, err
	}

	// Create a wrapper channel to handle span ending
	wrappedChan := make(chan iface.AIMessageChunk)
	go func() {
		defer close(wrappedChan)
		defer a.tracing.EndSpan(ctx)

		for chunk := range streamChan {
			select {
			case wrappedChan <- chunk:
			case <-ctx.Done():
				return
			}
		}

		duration := time.Since(start)
		a.metrics.RecordStream(ctx, ProviderName, a.modelName, duration)
	}()

	return wrappedChan, nil
}

// BindTools implements the ChatModel interface
func (a *AnthropicProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	newProvider := *a // Create a copy
	newProvider.tools = make([]tools.Tool, len(toolsToBind))
	copy(newProvider.tools, toolsToBind)
	return &newProvider
}

// GetModelName implements the ChatModel interface
func (a *AnthropicProvider) GetModelName() string {
	return a.modelName
}

func (a *AnthropicProvider) GetProviderName() string {
	return ProviderName
}

// Invoke implements the Runnable interface
func (a *AnthropicProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return a.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface
func (a *AnthropicProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	errors := make([]error, len(inputs))

	// Use semaphore for concurrency control
	sem := make(chan struct{}, a.config.MaxConcurrentBatches)

	for i, input := range inputs {
		sem <- struct{}{} // Acquire semaphore

		go func(index int, currentInput any) {
			defer func() { <-sem }() // Release semaphore

			result, err := a.Invoke(ctx, currentInput, options...)
			results[index] = result
			errors[index] = err
		}(i, input)
	}

	// Wait for all goroutines to complete
	for i := 0; i < a.config.MaxConcurrentBatches; i++ {
		sem <- struct{}{}
	}

	// Check for errors
	var combinedErr error
	for _, err := range errors {
		if err != nil {
			if combinedErr == nil {
				combinedErr = err
			} else {
				combinedErr = fmt.Errorf("%v; %v", combinedErr, err)
			}
		}
	}

	return results, combinedErr
}

// Stream implements the Runnable interface
func (a *AnthropicProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := a.StreamChat(ctx, messages, options...)
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

// generateInternal performs the actual generation logic
func (a *AnthropicProvider) generateInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (schema.Message, error) {
	// Convert messages to Anthropic format
	systemPrompt, anthropicMessages, err := a.convertMessages(messages)
	if err != nil {
		return nil, llms.NewLLMError("generateInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Build request parameters
	req := a.buildAnthropicRequest(systemPrompt, anthropicMessages, opts)

	// Make API call
	resp, err := a.client.Beta.Messages.New(ctx, req)
	if err != nil {
		return nil, a.handleAnthropicError("generateInternal", err)
	}

	// Convert response to schema.Message
	return a.convertAnthropicResponse(resp)
}

// streamInternal performs the actual streaming logic
func (a *AnthropicProvider) streamInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (<-chan iface.AIMessageChunk, error) {
	// Convert messages to Anthropic format
	systemPrompt, anthropicMessages, err := a.convertMessages(messages)
	if err != nil {
		return nil, llms.NewLLMError("streamInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Build request parameters
	req := a.buildAnthropicRequest(systemPrompt, anthropicMessages, opts)

	// Create streaming response
	streamResp := a.client.Beta.Messages.NewStreaming(ctx, req)
	if err := streamResp.Err(); err != nil {
		return nil, a.handleAnthropicError("streamInternal", err)
	}

	stream := streamResp
	outputChan := make(chan iface.AIMessageChunk)

	go func() {
		defer close(outputChan)
		defer stream.Close()

		for {
			event := stream.Next()
			if stream.Err() != nil {
				if stream.Err() == io.EOF {
					break
				}
				finalChunk := iface.AIMessageChunk{
					Err: llms.WrapError("anthropic.stream", stream.Err()),
				}
				select {
				case outputChan <- finalChunk:
				case <-ctx.Done():
				}
				return
			}

			// Convert Anthropic event to AIMessageChunk
			chunk, err := a.convertAnthropicStreamEvent(event)
			if err != nil {
				finalChunk := iface.AIMessageChunk{
					Err: llms.WrapError("anthropic.stream.convert", err),
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

// convertMessages converts schema messages to Anthropic format
func (a *AnthropicProvider) convertMessages(messages []schema.Message) (*string, []anthropic.BetaMessageParam, error) {
	var systemPrompt *string
	var anthropicMsgs []anthropic.BetaMessageParam

	processedMessages := messages

	// Extract system message if present
	if len(messages) > 0 {
		if chatMsg, ok := messages[0].(*schema.ChatMessage); ok && chatMsg.GetType() == schema.RoleSystem {
			if chatMsg.GetContent() != "" {
				content := chatMsg.GetContent()
				systemPrompt = &content
			}
			processedMessages = messages[1:]
		}
	}

	// Convert remaining messages
	anthropicMsgs = make([]anthropic.BetaMessageParam, 0, len(processedMessages))
	for _, msg := range processedMessages {
		var contentBlocks []anthropic.BetaContentBlockParamUnion
		var role string

		switch m := msg.(type) {
		case *schema.ChatMessage:
			if m.GetType() == schema.RoleHuman {
				role = "user"
				contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestTextBlock(m.GetContent()))
			} else if m.GetType() == schema.RoleAssistant {
				role = "assistant"
				contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestTextBlock(m.GetContent()))
			}
		case *schema.AIMessage:
			role = "assistant"
			contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestTextBlock(m.GetContent()))
		default:
			log.Printf("Warning: Skipping unsupported message type for Anthropic: %T", msg)
			continue
		}

		if len(contentBlocks) > 0 {
			anthropicMsgs = append(anthropicMsgs, anthropic.BetaMessageParam{
				Role:    anthropic.BetaMessageParamRole(role),
				Content: contentBlocks,
			})
		}
	}

	if len(anthropicMsgs) == 0 && systemPrompt == nil {
		return nil, nil, fmt.Errorf("no valid messages provided for Anthropic conversion")
	}

	return systemPrompt, anthropicMsgs, nil
}

// buildAnthropicRequest builds the Anthropic API request
func (a *AnthropicProvider) buildAnthropicRequest(systemPrompt *string, messages []anthropic.BetaMessageParam, opts *llms.CallOptions) anthropic.BetaMessageNewParams {
	req := anthropic.BetaMessageNewParams{
		Model:     a.modelName,
		MaxTokens: 1024, // Default, will be overridden by options
		Messages:  messages,
	}

	// Set system prompt
	if systemPrompt != nil {
		req.System = []anthropic.BetaTextBlockParam{
			{Text: *systemPrompt},
		}
	}

	// Apply call options
	if opts.MaxTokens != nil {
		req.MaxTokens = int64(*opts.MaxTokens)
	}
	if opts.Temperature != nil {
		req.Temperature = param.NewOpt(float64(*opts.Temperature))
	}
	if opts.TopP != nil {
		req.TopP = param.NewOpt(float64(*opts.TopP))
	}
	if opts.TopK != nil {
		req.TopK = param.NewOpt(int64(*opts.TopK))
	}
	if len(opts.StopSequences) > 0 {
		req.StopSequences = opts.StopSequences
	}

	// Add tools if bound
	if len(a.tools) > 0 {
		req.Tools = a.convertTools(a.tools)
	}

	return req
}

// convertTools converts tools to Anthropic format
func (a *AnthropicProvider) convertTools(tools []tools.Tool) []anthropic.BetaToolUnionParam {
	if len(tools) == 0 {
		return nil
	}

	anthropicTools := make([]anthropic.BetaToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		def := tool.Definition()

		// Create tool schema
		toolMap := map[string]interface{}{
			"name":        def.Name,
			"description": def.Description,
		}

		// Add parameters schema
		if def.InputSchema != nil {
			if schemaStr, ok := def.InputSchema.(string); ok && schemaStr != "" {
				var params map[string]interface{}
				if err := json.Unmarshal([]byte(schemaStr), &params); err == nil {
					toolMap["input_schema"] = params
				}
			}
		}

		// Convert to Anthropic format
		toolBytes, _ := json.Marshal(toolMap)
		var toolUnion anthropic.BetaToolUnionParam
		if err := json.Unmarshal(toolBytes, &toolUnion); err == nil {
			anthropicTools = append(anthropicTools, toolUnion)
		}
	}

	return anthropicTools
}

// convertAnthropicResponse converts Anthropic response to schema.Message
func (a *AnthropicProvider) convertAnthropicResponse(resp *anthropic.BetaMessage) (schema.Message, error) {
	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Anthropic")
	}

	var responseText string
	var toolCalls []schema.ToolCall

	for _, blockUnion := range resp.Content {
		switch content := blockUnion.AsAny().(type) {
		case anthropic.BetaTextBlock:
			responseText += content.Text
		case anthropic.BetaToolUseBlock:
			argsBytes, _ := json.Marshal(content.Input)
			toolCall := schema.ToolCall{
				ID:        content.ID,
				Name:      content.Name,
				Arguments: string(argsBytes),
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	aiMsg := schema.NewAIMessage(responseText)
	if len(toolCalls) > 0 {
		if aiMsgInternal, ok := aiMsg.(*schema.AIMessage); ok {
			aiMsgInternal.ToolCalls_ = toolCalls
		}
	}

	// Add usage information
	args := aiMsg.AdditionalArgs()
	args["usage"] = map[string]int{
		"input_tokens":  int(resp.Usage.InputTokens),
		"output_tokens": int(resp.Usage.OutputTokens),
		"total_tokens":  int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
	}

	return aiMsg, nil
}

// convertAnthropicStreamEvent converts Anthropic stream events to AIMessageChunk
func (a *AnthropicProvider) convertAnthropicStreamEvent(event any) (*iface.AIMessageChunk, error) {
	// This is a simplified implementation
	// In a real implementation, you'd handle all the different event types
	_ = event // Mark event as used to avoid linting error

	// For now, return nil to indicate no meaningful chunk
	// In a full implementation, you'd parse the event and extract content
	return nil, nil
}

// buildCallOptions merges configuration options with call-specific options
func (a *AnthropicProvider) buildCallOptions(options ...core.Option) *llms.CallOptions {
	callOpts := llms.NewCallOptions()

	// Apply default configuration
	if a.config.MaxTokens != nil {
		callOpts.MaxTokens = a.config.MaxTokens
	}
	if a.config.Temperature != nil {
		temp := float32(*a.config.Temperature)
		callOpts.Temperature = &temp
	}
	if a.config.TopP != nil {
		topP := float32(*a.config.TopP)
		callOpts.TopP = &topP
	}
	if a.config.TopK != nil {
		topK := *a.config.TopK
		callOpts.TopK = &topK
	}
	if len(a.config.StopSequences) > 0 {
		callOpts.StopSequences = a.config.StopSequences
	}

	// Apply call-specific options
	for _, opt := range options {
		callOpts.ApplyCallOption(opt)
	}

	return callOpts
}

// handleAnthropicError converts Anthropic errors to LLM errors
func (a *AnthropicProvider) handleAnthropicError(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Try to extract more specific error information
	// This is a simplified implementation
	var errorCode string
	var message string

	if strings.Contains(err.Error(), "rate limit") {
		errorCode = ErrCodeRateLimit
		message = "Anthropic API rate limit exceeded"
	} else if strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), "unauthorized") {
		errorCode = ErrCodeInvalidAPIKey
		message = "Anthropic API authentication failed"
	} else if strings.Contains(err.Error(), "model") {
		errorCode = ErrCodeModelNotFound
		message = "Anthropic model not found"
	} else {
		errorCode = ErrCodeInvalidRequest
		message = "Anthropic API request failed"
	}

	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// CheckHealth implements the HealthChecker interface
func (a *AnthropicProvider) CheckHealth() map[string]interface{} {
	return map[string]interface{}{
		"state":       "healthy",
		"provider":    "anthropic",
		"model":       a.modelName,
		"timestamp":   time.Now().Unix(),
		"api_key_set": a.config.APIKey != "",
		"tools_count": len(a.tools),
	}
}

// Factory function for creating Anthropic providers
func NewAnthropicProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewAnthropicProvider(config)
	}
}
