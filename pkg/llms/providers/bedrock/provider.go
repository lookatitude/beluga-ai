// Package bedrock provides an implementation of the llms.ChatModel interface
// using the AWS Bedrock API.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Provider constants
const (
	ProviderName = "bedrock"
	DefaultModel = "anthropic.claude-3-haiku-20240307-v1:0"

	// Error codes specific to AWS Bedrock
	ErrCodeInvalidCredentials = "bedrock_invalid_credentials"
	ErrCodeRegionNotFound     = "bedrock_region_not_found"
	ErrCodeModelNotFound      = "bedrock_model_not_found"
	ErrCodeInvalidRequest     = "bedrock_invalid_request"
	ErrCodeThrottling         = "bedrock_throttling"
)

// BedrockProvider implements the ChatModel interface for AWS Bedrock models
type BedrockProvider struct {
	config      *llms.Config
	client      *bedrockruntime.Client
	modelName   string
	tools       []tools.Tool
	metrics     llms.MetricsRecorder
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
	region      string
}

// NewBedrockProvider creates a new AWS Bedrock provider instance
func NewBedrockProvider(ctx context.Context, config *llms.Config) (*BedrockProvider, error) {
	// Validate configuration
	if err := llms.ValidateProviderConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("invalid Bedrock configuration: %w", err)
	}

	// Get region from provider-specific config or default
	region := "us-east-1" // Default region
	if regionStr, ok := config.ProviderSpecific["region"].(string); ok && regionStr != "" {
		region = regionStr
	}

	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Bedrock Runtime client
	client := bedrockruntime.NewFromConfig(awsCfg)

	provider := &BedrockProvider{
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
		region: region,
	}

	return provider, nil
}

// NewBedrockLLM creates a new Bedrock provider (legacy compatibility function)
func NewBedrockLLM(ctx context.Context, modelName string, opts ...llms.ConfigOption) (iface.ChatModel, error) {
	config := llms.NewConfig(opts...)
	config.ModelName = modelName
	config.Provider = ProviderName

	return NewBedrockProvider(ctx, config)
}

// Generate implements the ChatModel interface
func (b *BedrockProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = b.tracing.StartOperation(ctx, "bedrock.generate", ProviderName, b.modelName)
	defer b.tracing.EndSpan(ctx)

	// Record request metrics
	b.metrics.IncrementActiveRequests(ctx, ProviderName, b.modelName)
	defer b.metrics.DecrementActiveRequests(ctx, ProviderName, b.modelName)

	// Apply options and merge with defaults
	callOpts := b.buildCallOptions(options...)

	// Execute with retry logic
	var result schema.Message
	var err error

	retryErr := common.RetryWithBackoff(ctx, b.retryConfig, "bedrock.generate", func() error {
		result, err = b.generateInternal(ctx, messages, callOpts)
		return err
	})

	if retryErr != nil {
		b.metrics.RecordError(ctx, ProviderName, b.modelName, llms.GetLLMErrorCode(retryErr))
		b.tracing.RecordError(ctx, retryErr)
		return nil, retryErr
	}

	// Record success metrics
	b.metrics.RecordRequest(ctx, ProviderName, b.modelName, 0) // Duration will be recorded by caller

	return result, nil
}

// StreamChat implements the ChatModel interface
func (b *BedrockProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// TODO: Implement streaming for Bedrock
	// For now, return an error
	return nil, fmt.Errorf("streaming not yet implemented for Bedrock provider")
}

// BindTools implements the ChatModel interface
func (b *BedrockProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	newProvider := *b // Create a copy
	newProvider.tools = make([]tools.Tool, len(toolsToBind))
	copy(newProvider.tools, toolsToBind)
	return &newProvider
}

// GetModelName implements the ChatModel interface
func (b *BedrockProvider) GetModelName() string {
	return b.modelName
}

// Invoke implements the Runnable interface
func (b *BedrockProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return b.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface
func (b *BedrockProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	errors := make([]error, len(inputs))

	// Use semaphore for concurrency control
	sem := make(chan struct{}, b.config.MaxConcurrentBatches)

	for i, input := range inputs {
		sem <- struct{}{} // Acquire semaphore

		go func(index int, currentInput any) {
			defer func() { <-sem }() // Release semaphore

			result, err := b.Invoke(ctx, currentInput, options...)
			results[index] = result
			errors[index] = err
		}(i, input)
	}

	// Wait for all goroutines to complete
	for i := 0; i < b.config.MaxConcurrentBatches; i++ {
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
func (b *BedrockProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := b.StreamChat(ctx, messages, options...)
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
func (b *BedrockProvider) generateInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (schema.Message, error) {
	// Convert messages to Bedrock format
	bedrockRequest, err := b.convertMessages(messages, opts)
	if err != nil {
		return nil, llms.NewLLMError("generateInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Make API call
	resp, err := b.client.InvokeModel(ctx, bedrockRequest)
	if err != nil {
		return nil, b.handleBedrockError("generateInternal", err)
	}

	// Convert response to schema.Message
	return b.convertBedrockResponse(resp)
}

// streamInternal performs the actual streaming logic
func (b *BedrockProvider) streamInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (<-chan iface.AIMessageChunk, error) {
	// TODO: Implement streaming for Bedrock
	return nil, fmt.Errorf("streaming not yet implemented for Bedrock provider")
}

// convertMessages converts schema messages to Bedrock format
func (b *BedrockProvider) convertMessages(messages []schema.Message, opts *llms.CallOptions) (*bedrockruntime.InvokeModelInput, error) {
	// This is a simplified implementation - in practice, you'd need to handle
	// different model formats (Anthropic, Titan, etc.) based on the model name

	// For Anthropic models (most common)
	if strings.Contains(b.modelName, "anthropic") {
		return b.convertAnthropicMessages(messages, opts)
	}

	// For Titan models
	if strings.Contains(b.modelName, "titan") {
		return b.convertTitanMessages(messages, opts)
	}

	return nil, fmt.Errorf("unsupported model type for Bedrock: %s", b.modelName)
}

// convertAnthropicMessages converts messages for Anthropic models on Bedrock
func (b *BedrockProvider) convertAnthropicMessages(messages []schema.Message, opts *llms.CallOptions) (*bedrockruntime.InvokeModelInput, error) {
	var systemPrompt *string
	var anthropicMessages []map[string]interface{}

	// Extract system message and convert others
	for _, msg := range messages {
		if chatMsg, ok := msg.(*schema.ChatMessage); ok {
			if chatMsg.GetType() == schema.RoleSystem {
				content := chatMsg.GetContent()
				systemPrompt = &content
			} else {
				role := "user"
				if chatMsg.GetType() == schema.RoleAssistant {
					role = "assistant"
				}

				anthropicMessages = append(anthropicMessages, map[string]interface{}{
					"role":    role,
					"content": chatMsg.GetContent(),
				})
			}
		} else if aiMsg, ok := msg.(*schema.AIMessage); ok {
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    "assistant",
				"content": aiMsg.GetContent(),
			})
		}
	}

	// Build Anthropic request body
	requestBody := map[string]interface{}{
		"messages":   anthropicMessages,
		"max_tokens": 1024, // Default
	}

	if systemPrompt != nil {
		requestBody["system"] = *systemPrompt
	}

	// Apply call options
	if opts.MaxTokens != nil {
		requestBody["max_tokens"] = *opts.MaxTokens
	}
	if opts.Temperature != nil {
		requestBody["temperature"] = *opts.Temperature
	}
	if opts.TopP != nil {
		requestBody["top_p"] = *opts.TopP
	}
	if opts.TopK != nil {
		requestBody["top_k"] = *opts.TopK
	}

	// Serialize request body
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Anthropic request: %w", err)
	}

	return &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(b.modelName),
		Body:        bodyBytes,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	}, nil
}

// convertTitanMessages converts messages for Titan models on Bedrock
func (b *BedrockProvider) convertTitanMessages(messages []schema.Message, opts *llms.CallOptions) (*bedrockruntime.InvokeModelInput, error) {
	// Simplified Titan message conversion
	var inputText string

	for _, msg := range messages {
		if chatMsg, ok := msg.(*schema.ChatMessage); ok {
			if chatMsg.GetType() == schema.RoleSystem {
				inputText += "System: " + chatMsg.GetContent() + "\n\n"
			} else {
				role := "User"
				if chatMsg.GetType() == schema.RoleAssistant {
					role = "Assistant"
				}
				inputText += role + ": " + chatMsg.GetContent() + "\n\n"
			}
		} else if aiMsg, ok := msg.(*schema.AIMessage); ok {
			inputText += "Assistant: " + aiMsg.GetContent() + "\n\n"
		}
	}

	// Build Titan request body
	requestBody := map[string]interface{}{
		"inputText": inputText,
		"textGenerationConfig": map[string]interface{}{
			"maxTokenCount": 1024, // Default
		},
	}

	// Apply call options
	if opts.MaxTokens != nil {
		requestBody["textGenerationConfig"].(map[string]interface{})["maxTokenCount"] = *opts.MaxTokens
	}
	if opts.Temperature != nil {
		requestBody["textGenerationConfig"].(map[string]interface{})["temperature"] = *opts.Temperature
	}
	if opts.TopP != nil {
		requestBody["textGenerationConfig"].(map[string]interface{})["topP"] = *opts.TopP
	}

	// Serialize request body
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Titan request: %w", err)
	}

	return &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(b.modelName),
		Body:        bodyBytes,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	}, nil
}

// convertBedrockResponse converts Bedrock response to schema.Message
func (b *BedrockProvider) convertBedrockResponse(resp *bedrockruntime.InvokeModelOutput) (schema.Message, error) {
	// Parse response based on model type
	if strings.Contains(b.modelName, "anthropic") {
		return b.convertAnthropicResponse(resp)
	}
	if strings.Contains(b.modelName, "titan") {
		return b.convertTitanResponse(resp)
	}

	return nil, fmt.Errorf("unsupported model type for response conversion: %s", b.modelName)
}

// convertAnthropicResponse converts Anthropic model response
func (b *BedrockProvider) convertAnthropicResponse(resp *bedrockruntime.InvokeModelOutput) (schema.Message, error) {
	var anthropicResp map[string]interface{}
	if err := json.Unmarshal(resp.Body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Anthropic response: %w", err)
	}

	content, ok := anthropicResp["content"].([]interface{})
	if !ok || len(content) == 0 {
		return nil, fmt.Errorf("invalid Anthropic response format")
	}

	textContent := ""
	for _, item := range content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, ok := itemMap["type"].(string); ok && itemType == "text" {
				if text, ok := itemMap["text"].(string); ok {
					textContent += text
				}
			}
		}
	}

	aiMsg := schema.NewAIMessage(textContent)

	// Add usage information if available
	if usage, ok := anthropicResp["usage"].(map[string]interface{}); ok {
		args := aiMsg.AdditionalArgs()
		args["usage"] = usage
	}

	return aiMsg, nil
}

// convertTitanResponse converts Titan model response
func (b *BedrockProvider) convertTitanResponse(resp *bedrockruntime.InvokeModelOutput) (schema.Message, error) {
	var titanResp map[string]interface{}
	if err := json.Unmarshal(resp.Body, &titanResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Titan response: %w", err)
	}

	results, ok := titanResp["results"].([]interface{})
	if !ok || len(results) == 0 {
		return nil, fmt.Errorf("invalid Titan response format")
	}

	result, ok := results[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Titan result format")
	}

	outputText, ok := result["outputText"].(string)
	if !ok {
		return nil, fmt.Errorf("missing output text in Titan response")
	}

	aiMsg := schema.NewAIMessage(outputText)

	// Add usage information if available
	if tokenCount, ok := result["tokenCount"].(float64); ok {
		args := aiMsg.AdditionalArgs()
		args["usage"] = map[string]interface{}{
			"output_tokens": int(tokenCount),
		}
	}

	return aiMsg, nil
}

// convertBedrockStreamEvent converts Bedrock stream events to AIMessageChunk
func (b *BedrockProvider) convertBedrockStreamEvent(event any) (*iface.AIMessageChunk, error) {
	// This is a simplified implementation - in practice, you'd need to handle
	// the specific event types for different Bedrock models
	return nil, nil
}

// buildCallOptions merges configuration options with call-specific options
func (b *BedrockProvider) buildCallOptions(options ...core.Option) *llms.CallOptions {
	callOpts := llms.NewCallOptions()

	// Apply default configuration
	if b.config.MaxTokens != nil {
		callOpts.MaxTokens = b.config.MaxTokens
	}
	if b.config.Temperature != nil {
		temp := float32(*b.config.Temperature)
		callOpts.Temperature = &temp
	}
	if b.config.TopP != nil {
		topP := float32(*b.config.TopP)
		callOpts.TopP = &topP
	}
	if b.config.TopK != nil {
		topK := *b.config.TopK
		callOpts.TopK = &topK
	}

	// Apply call-specific options
	for _, opt := range options {
		callOpts.ApplyCallOption(opt)
	}

	return callOpts
}

// handleBedrockError converts Bedrock errors to LLM errors
func (b *BedrockProvider) handleBedrockError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "throttling") || strings.Contains(errStr, "TooManyRequests") {
		errorCode = ErrCodeThrottling
		message = "Bedrock API throttling limit exceeded"
	} else if strings.Contains(errStr, "credentials") || strings.Contains(errStr, "Unauthorized") {
		errorCode = ErrCodeInvalidCredentials
		message = "Bedrock API credentials invalid"
	} else if strings.Contains(errStr, "region") {
		errorCode = ErrCodeRegionNotFound
		message = "Bedrock region not found"
	} else if strings.Contains(errStr, "model") || strings.Contains(errStr, "NotFound") {
		errorCode = ErrCodeModelNotFound
		message = "Bedrock model not found"
	} else {
		errorCode = ErrCodeInvalidRequest
		message = "Bedrock API request failed"
	}

	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// Factory function for creating Bedrock providers
func NewBedrockProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewBedrockProvider(context.Background(), config)
	}
}
