// Package openai provides an implementation of the llms.ChatModel interface
// using the OpenAI API (GPT models).
package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Provider constants
const (
	ProviderName = "openai"
	DefaultModel = "gpt-3.5-turbo"

	// Error codes specific to OpenAI
	ErrCodeInvalidAPIKey  = "openai_invalid_api_key"
	ErrCodeRateLimit      = "openai_rate_limit"
	ErrCodeModelNotFound  = "openai_model_not_found"
	ErrCodeInvalidRequest = "openai_invalid_request"
	ErrCodeQuotaExceeded  = "openai_quota_exceeded"
)

// OpenAIProvider implements the ChatModel interface for OpenAI GPT models
type OpenAIProvider struct {
	config      *llms.Config
	client      *openai.Client
	modelName   string
	tools       []tools.Tool
	metrics     llms.MetricsRecorder
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(config *llms.Config) (*OpenAIProvider, error) {
	// Validate configuration
	if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
		return nil, fmt.Errorf("invalid OpenAI configuration: %w", err)
	}

	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// Build client configuration
	openaiConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		openaiConfig.BaseURL = config.BaseURL
	}

	// Add organization if specified in provider-specific config
	if orgID, ok := config.ProviderSpecific["organization"].(string); ok && orgID != "" {
		openaiConfig.OrgID = orgID
	}

	client := openai.NewClientWithConfig(openaiConfig)

	provider := &OpenAIProvider{
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

// Generate implements the ChatModel interface
func (o *OpenAIProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = o.tracing.StartOperation(ctx, "openai.generate", ProviderName, o.modelName)

	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	o.tracing.AddSpanAttributes(ctx, map[string]interface{}{"input_size": inputSize})

	start := time.Now()

	// Record request metrics
	o.metrics.IncrementActiveRequests(ctx, ProviderName, o.modelName)
	defer o.metrics.DecrementActiveRequests(ctx, ProviderName, o.modelName)

	// Apply options and merge with defaults
	callOpts := o.buildCallOptions(options...)

	// Execute with retry logic
	var result schema.Message
	var err error

	retryErr := common.RetryWithBackoff(ctx, o.retryConfig, "openai.generate", func() error {
		result, err = o.generateInternal(ctx, messages, callOpts)
		return err
	})

	if retryErr != nil {
		duration := time.Since(start)
		o.metrics.RecordError(ctx, ProviderName, o.modelName, llms.GetLLMErrorCode(retryErr), duration)
		o.tracing.RecordError(ctx, retryErr)
		return nil, retryErr
	}

	duration := time.Since(start)
	o.metrics.RecordRequest(ctx, ProviderName, o.modelName, duration)

	return result, nil
}

// StreamChat implements the ChatModel interface
func (o *OpenAIProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	ctx = o.tracing.StartOperation(ctx, "openai.stream", ProviderName, o.modelName)

	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	o.tracing.AddSpanAttributes(ctx, map[string]interface{}{"input_size": inputSize})

	// Apply options and merge with defaults
	callOpts := o.buildCallOptions(options...)

	// Execute streaming request
	return o.streamInternal(ctx, messages, callOpts)
}

// BindTools implements the ChatModel interface
func (o *OpenAIProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	newProvider := *o // Create a copy
	newProvider.tools = make([]tools.Tool, len(toolsToBind))
	copy(newProvider.tools, toolsToBind)
	return &newProvider
}

// GetModelName implements the ChatModel interface
func (o *OpenAIProvider) GetModelName() string {
	return o.modelName
}

func (o *OpenAIProvider) GetProviderName() string {
	return ProviderName
}

// Invoke implements the Runnable interface
func (o *OpenAIProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return o.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface
func (o *OpenAIProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	errors := make([]error, len(inputs))

	// Use semaphore for concurrency control
	sem := make(chan struct{}, o.config.MaxConcurrentBatches)

	for i, input := range inputs {
		sem <- struct{}{} // Acquire semaphore

		go func(index int, currentInput any) {
			defer func() { <-sem }() // Release semaphore

			result, err := o.Invoke(ctx, currentInput, options...)
			results[index] = result
			errors[index] = err
		}(i, input)
	}

	// Wait for all goroutines to complete
	for i := 0; i < o.config.MaxConcurrentBatches; i++ {
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
func (o *OpenAIProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := o.StreamChat(ctx, messages, options...)
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
func (o *OpenAIProvider) generateInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (schema.Message, error) {
	// Convert messages to OpenAI format
	openaiMessages, err := o.convertMessages(messages)
	if err != nil {
		return nil, llms.NewLLMError("generateInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Build request parameters
	req := o.buildOpenAIRequest(openaiMessages, opts)

	// Make API call
	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, o.handleOpenAIError("generateInternal", err)
	}

	// Convert response to schema.Message
	return o.convertOpenAIResponse(&resp)
}

// streamInternal performs the actual streaming logic
func (o *OpenAIProvider) streamInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (<-chan iface.AIMessageChunk, error) {
	// Convert messages to OpenAI format
	openaiMessages, err := o.convertMessages(messages)
	if err != nil {
		return nil, llms.NewLLMError("streamInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Build request parameters
	req := o.buildOpenAIRequest(openaiMessages, opts)
	req.Stream = true

	// Create streaming response
	stream, err := o.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, o.handleOpenAIError("streamInternal", err)
	}

	outputChan := make(chan iface.AIMessageChunk)

	go func() {
		defer close(outputChan)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err != nil && err.Error() == "stream closed" {
					break
				}
				finalChunk := iface.AIMessageChunk{
					Err: llms.WrapError("openai.stream", err),
				}
				select {
				case outputChan <- finalChunk:
				case <-ctx.Done():
				}
				return
			}

			// Convert OpenAI stream response to AIMessageChunk
			chunk, err := o.convertOpenAIStreamResponse(&response)
			if err != nil {
				finalChunk := iface.AIMessageChunk{
					Err: llms.WrapError("openai.stream.convert", err),
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

// convertMessages converts schema messages to OpenAI format
func (o *OpenAIProvider) convertMessages(messages []schema.Message) ([]openai.ChatCompletionMessage, error) {
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages))

	for _, msg := range messages {
		var openaiMsg openai.ChatCompletionMessage

		switch m := msg.(type) {
		case *schema.ChatMessage:
			switch m.GetType() {
			case schema.RoleSystem:
				openaiMsg.Role = openai.ChatMessageRoleSystem
				openaiMsg.Content = m.GetContent()
			case schema.RoleHuman:
				openaiMsg.Role = openai.ChatMessageRoleUser
				openaiMsg.Content = m.GetContent()
			case schema.RoleAssistant:
				openaiMsg.Role = openai.ChatMessageRoleAssistant
				openaiMsg.Content = m.GetContent()
			default:
				continue // Skip unknown roles
			}
		case *schema.AIMessage:
			openaiMsg.Role = openai.ChatMessageRoleAssistant
			openaiMsg.Content = m.GetContent()

			// Add tool calls if present
			if len(m.ToolCalls()) > 0 {
				openaiMsg.ToolCalls = make([]openai.ToolCall, len(m.ToolCalls()))
				for i, tc := range m.ToolCalls() {
					openaiMsg.ToolCalls[i] = openai.ToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: openai.FunctionCall{
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
		return nil, fmt.Errorf("no valid messages provided for OpenAI conversion")
	}

	return openaiMessages, nil
}

// buildOpenAIRequest builds the OpenAI API request
func (o *OpenAIProvider) buildOpenAIRequest(messages []openai.ChatCompletionMessage, opts *llms.CallOptions) openai.ChatCompletionRequest {
	req := openai.ChatCompletionRequest{
		Model:    o.modelName,
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
	if len(o.tools) > 0 {
		req.Tools = o.convertTools(o.tools)
	}

	return req
}

// convertTools converts tools to OpenAI format
func (o *OpenAIProvider) convertTools(tools []tools.Tool) []openai.Tool {
	if len(tools) == 0 {
		return nil
	}

	openaiTools := make([]openai.Tool, 0, len(tools))
	for _, tool := range tools {
		def := tool.Definition()

		openaiTool := openai.Tool{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        def.Name,
				Description: def.Description,
			},
		}

		// Add parameters schema
		if def.InputSchema != nil {
			if schemaStr, ok := def.InputSchema.(string); ok && schemaStr != "" {
				var params map[string]interface{}
				if err := json.Unmarshal([]byte(schemaStr), &params); err == nil {
					openaiTool.Function.Parameters = params
				}
			}
		}

		openaiTools = append(openaiTools, openaiTool)
	}

	return openaiTools
}

// convertOpenAIResponse converts OpenAI response to schema.Message
func (o *OpenAIProvider) convertOpenAIResponse(resp *openai.ChatCompletionResponse) (schema.Message, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from OpenAI")
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

// convertOpenAIStreamResponse converts OpenAI stream response to AIMessageChunk
func (o *OpenAIProvider) convertOpenAIStreamResponse(resp *openai.ChatCompletionStreamResponse) (*iface.AIMessageChunk, error) {
	if len(resp.Choices) == 0 {
		return nil, nil
	}

	choice := resp.Choices[0]
	chunk := &iface.AIMessageChunk{
		Content:        choice.Delta.Content,
		AdditionalArgs: make(map[string]interface{}),
	}

	// Add finish reason if present
	if choice.FinishReason != "" {
		chunk.AdditionalArgs["finish_reason"] = choice.FinishReason
	}

	return chunk, nil
}

// buildCallOptions merges configuration options with call-specific options
func (o *OpenAIProvider) buildCallOptions(options ...core.Option) *llms.CallOptions {
	callOpts := llms.NewCallOptions()

	// Apply default configuration
	if o.config.MaxTokens != nil {
		callOpts.MaxTokens = o.config.MaxTokens
	}
	if o.config.Temperature != nil {
		temp := float32(*o.config.Temperature)
		callOpts.Temperature = &temp
	}
	if o.config.TopP != nil {
		topP := float32(*o.config.TopP)
		callOpts.TopP = &topP
	}
	if o.config.FrequencyPenalty != nil {
		freq := float32(*o.config.FrequencyPenalty)
		callOpts.FrequencyPenalty = &freq
	}
	if o.config.PresencePenalty != nil {
		pres := float32(*o.config.PresencePenalty)
		callOpts.PresencePenalty = &pres
	}
	if len(o.config.StopSequences) > 0 {
		callOpts.StopSequences = o.config.StopSequences
	}

	// Apply call-specific options
	for _, opt := range options {
		callOpts.ApplyCallOption(opt)
	}

	return callOpts
}

// handleOpenAIError converts OpenAI errors to LLM errors
func (o *OpenAIProvider) handleOpenAIError(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Try to extract more specific error information from OpenAI error
	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeRateLimit
		message = "OpenAI API rate limit exceeded"
	} else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") {
		errorCode = ErrCodeInvalidAPIKey
		message = "OpenAI API authentication failed"
	} else if strings.Contains(errStr, "model") || strings.Contains(errStr, "404") {
		errorCode = ErrCodeModelNotFound
		message = "OpenAI model not found"
	} else if strings.Contains(errStr, "quota") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeQuotaExceeded
		message = "OpenAI API quota exceeded"
	} else {
		errorCode = ErrCodeInvalidRequest
		message = "OpenAI API request failed"
	}

	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// CheckHealth implements the HealthChecker interface
func (o *OpenAIProvider) CheckHealth() map[string]interface{} {
	return map[string]interface{}{
		"state":       "healthy",
		"provider":    "openai",
		"model":       o.modelName,
		"timestamp":   time.Now().Unix(),
		"api_key_set": o.config.APIKey != "",
		"tools_count": len(o.tools),
	}
}

// Factory function for creating OpenAI providers
func NewOpenAIProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewOpenAIProvider(config)
	}
}
