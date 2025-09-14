// Package ollama provides an implementation of the llms.ChatModel interface
// using the Ollama API for local LLM models.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Provider constants
const (
	ProviderName = "ollama"
	DefaultModel = "llama2"

	// Error codes specific to Ollama
	ErrCodeConnectionFailed = "ollama_connection_failed"
	ErrCodeModelNotFound    = "ollama_model_not_found"
	ErrCodeInvalidRequest   = "ollama_invalid_request"
)

// OllamaProvider implements the ChatModel interface for Ollama models
type OllamaProvider struct {
	config      *llms.Config
	baseURL     string
	modelName   string
	tools       []tools.Tool
	metrics     llms.MetricsRecorder
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
	client      *http.Client
}

// NewOllamaProvider creates a new Ollama provider instance
func NewOllamaProvider(config *llms.Config) (*OllamaProvider, error) {
	// Validate configuration
	if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
		return nil, fmt.Errorf("invalid Ollama configuration: %w", err)
	}

	// Get base URL from provider-specific config or default
	baseURL := "http://localhost:11434" // Default Ollama server URL
	if url, ok := config.ProviderSpecific["base_url"].(string); ok && url != "" {
		baseURL = url
	}

	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	provider := &OllamaProvider{
		config:    config,
		baseURL:   baseURL,
		modelName: modelName,
		metrics:   llms.GetMetrics(), // Get global metrics instance
		tracing:   common.NewTracingHelper(),
		retryConfig: &common.RetryConfig{
			MaxRetries: config.MaxRetries,
			Delay:      config.RetryDelay,
			Backoff:    config.RetryBackoff,
		},
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}

	return provider, nil
}

// Generate implements the ChatModel interface
func (o *OllamaProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = o.tracing.StartOperation(ctx, "ollama.generate", ProviderName, o.modelName)
	defer o.tracing.EndSpan(ctx)

	// Record request metrics
	o.metrics.IncrementActiveRequests(ctx, ProviderName, o.modelName)
	defer o.metrics.DecrementActiveRequests(ctx, ProviderName, o.modelName)

	// Apply options and merge with defaults
	callOpts := o.buildCallOptions(options...)

	// Execute with retry logic
	var result schema.Message
	var err error

	retryErr := common.RetryWithBackoff(ctx, o.retryConfig, "ollama.generate", func() error {
		result, err = o.generateInternal(ctx, messages, callOpts)
		return err
	})

	if retryErr != nil {
		o.metrics.RecordError(ctx, ProviderName, o.modelName, llms.GetLLMErrorCode(retryErr))
		o.tracing.RecordError(ctx, retryErr)
		return nil, retryErr
	}

	// Record success metrics
	o.metrics.RecordRequest(ctx, ProviderName, o.modelName, 0) // Duration will be recorded by caller

	return result, nil
}

// StreamChat implements the ChatModel interface
func (o *OllamaProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	ctx = o.tracing.StartOperation(ctx, "ollama.stream", ProviderName, o.modelName)
	defer o.tracing.EndSpan(ctx)

	// Apply options and merge with defaults
	callOpts := o.buildCallOptions(options...)

	// Execute streaming request
	return o.streamInternal(ctx, messages, callOpts)
}

// BindTools implements the ChatModel interface
func (o *OllamaProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	newProvider := *o // Create a copy
	newProvider.tools = make([]tools.Tool, len(toolsToBind))
	copy(newProvider.tools, toolsToBind)
	return &newProvider
}

// GetModelName implements the ChatModel interface
func (o *OllamaProvider) GetModelName() string {
	return o.modelName
}

// Invoke implements the Runnable interface
func (o *OllamaProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return o.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface
func (o *OllamaProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
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
func (o *OllamaProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
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
func (o *OllamaProvider) generateInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (schema.Message, error) {
	// Convert messages to Ollama format
	ollamaRequest, err := o.convertMessages(messages, opts, false)
	if err != nil {
		return nil, llms.NewLLMError("generateInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Make API call
	resp, err := o.makeRequest(ctx, "POST", "/api/generate", ollamaRequest)
	if err != nil {
		return nil, o.handleOllamaError("generateInternal", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, o.handleHTTPError("generateInternal", resp)
	}

	// Parse response
	var ollamaResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, llms.NewLLMError("generateInternal", llms.ErrCodeInvalidResponse,
			fmt.Errorf("failed to decode Ollama response: %w", err))
	}

	// Convert response to schema.Message
	return o.convertOllamaResponse(ollamaResp)
}

// streamInternal performs the actual streaming logic
func (o *OllamaProvider) streamInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (<-chan iface.AIMessageChunk, error) {
	// Convert messages to Ollama format with streaming enabled
	ollamaRequest, err := o.convertMessages(messages, opts, true)
	if err != nil {
		return nil, llms.NewLLMError("streamInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Make streaming API call
	resp, err := o.makeRequest(ctx, "POST", "/api/generate", ollamaRequest)
	if err != nil {
		return nil, o.handleOllamaError("streamInternal", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, o.handleHTTPError("streamInternal", resp)
	}

	outputChan := make(chan iface.AIMessageChunk)

	go func() {
		defer close(outputChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var ollamaResp map[string]interface{}
			if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
				finalChunk := iface.AIMessageChunk{
					Err: llms.WrapError("ollama.stream.parse", err),
				}
				select {
				case outputChan <- finalChunk:
				case <-ctx.Done():
				}
				return
			}

			// Convert Ollama response to AIMessageChunk
			chunk, err := o.convertOllamaStreamResponse(ollamaResp)
			if err != nil {
				finalChunk := iface.AIMessageChunk{
					Err: llms.WrapError("ollama.stream.convert", err),
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

			// Check if this is the final response
			if done, ok := ollamaResp["done"].(bool); ok && done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			finalChunk := iface.AIMessageChunk{
				Err: llms.WrapError("ollama.stream.scan", err),
			}
			select {
			case outputChan <- finalChunk:
			case <-ctx.Done():
			}
		}
	}()

	return outputChan, nil
}

// convertMessages converts schema messages to Ollama format
func (o *OllamaProvider) convertMessages(messages []schema.Message, opts *llms.CallOptions, stream bool) (map[string]interface{}, error) {
	// Ollama uses a simple prompt format, not structured messages like ChatGPT
	var prompt strings.Builder

	// Convert conversation to a single prompt
	for i, msg := range messages {
		if i > 0 {
			prompt.WriteString("\n\n")
		}

		if chatMsg, ok := msg.(*schema.ChatMessage); ok {
			if chatMsg.GetType() == schema.RoleSystem {
				prompt.WriteString("System: ")
			} else if chatMsg.GetType() == schema.RoleHuman {
				prompt.WriteString("Human: ")
			} else if chatMsg.GetType() == schema.RoleAssistant {
				prompt.WriteString("Assistant: ")
			}
			prompt.WriteString(chatMsg.GetContent())
		} else if aiMsg, ok := msg.(*schema.AIMessage); ok {
			prompt.WriteString("Assistant: ")
			prompt.WriteString(aiMsg.GetContent())
		} else {
			prompt.WriteString(msg.GetContent())
		}
	}

	// Add final assistant prompt
	prompt.WriteString("\n\nAssistant: ")

	// Build Ollama request
	request := map[string]interface{}{
		"model":  o.modelName,
		"prompt": prompt.String(),
		"stream": stream,
	}

	// Apply call options
	if opts.MaxTokens != nil {
		request["num_predict"] = *opts.MaxTokens
	}
	if opts.Temperature != nil {
		request["temperature"] = *opts.Temperature
	}
	if opts.TopP != nil {
		request["top_p"] = *opts.TopP
	}
	if opts.TopK != nil {
		request["top_k"] = *opts.TopK
	}

	return request, nil
}

// convertOllamaResponse converts Ollama response to schema.Message
func (o *OllamaProvider) convertOllamaResponse(ollamaResp map[string]interface{}) (schema.Message, error) {
	response, ok := ollamaResp["response"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid Ollama response format: missing response field")
	}

	aiMsg := schema.NewAIMessage(response)

	// Add usage information if available
	if evalCount, ok := ollamaResp["eval_count"].(float64); ok {
		args := aiMsg.AdditionalArgs()
		args["usage"] = map[string]interface{}{
			"eval_count":     int(evalCount),
			"eval_duration":  ollamaResp["eval_duration"],
			"total_duration": ollamaResp["total_duration"],
		}
	}

	return aiMsg, nil
}

// convertOllamaStreamResponse converts Ollama streaming response to AIMessageChunk
func (o *OllamaProvider) convertOllamaStreamResponse(ollamaResp map[string]interface{}) (*iface.AIMessageChunk, error) {
	response, ok := ollamaResp["response"].(string)
	if !ok {
		// Check if this is the final message with done=true
		if done, ok := ollamaResp["done"].(bool); ok && done {
			chunk := &iface.AIMessageChunk{
				AdditionalArgs: map[string]interface{}{
					"finish_reason": "stop",
				},
			}
			return chunk, nil
		}
		return nil, nil
	}

	chunk := &iface.AIMessageChunk{
		Content:        response,
		AdditionalArgs: make(map[string]interface{}),
	}

	// Check if this is the final chunk
	if done, ok := ollamaResp["done"].(bool); ok && done {
		chunk.AdditionalArgs["finish_reason"] = "stop"

		// Add usage information if available
		if evalCount, ok := ollamaResp["eval_count"].(float64); ok {
			chunk.AdditionalArgs["usage"] = map[string]interface{}{
				"eval_count":     int(evalCount),
				"eval_duration":  ollamaResp["eval_duration"],
				"total_duration": ollamaResp["total_duration"],
			}
		}
	}

	return chunk, nil
}

// makeRequest makes an HTTP request to the Ollama server
func (o *OllamaProvider) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, o.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return o.client.Do(req)
}

// handleHTTPError handles HTTP errors from Ollama
func (o *OllamaProvider) handleHTTPError(operation string, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	var errorCode string
	var message string

	switch resp.StatusCode {
	case http.StatusNotFound:
		errorCode = ErrCodeModelNotFound
		message = "Ollama model not found"
	case http.StatusBadRequest:
		errorCode = ErrCodeInvalidRequest
		message = "Invalid Ollama request"
	case http.StatusInternalServerError:
		errorCode = ErrCodeConnectionFailed
		message = "Ollama server error"
	default:
		errorCode = ErrCodeConnectionFailed
		message = "Ollama request failed"
	}

	err := fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyStr)
	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// handleOllamaError converts Ollama errors to LLM errors
func (o *OllamaProvider) handleOllamaError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		errorCode = ErrCodeConnectionFailed
		message = "Failed to connect to Ollama server"
	} else if strings.Contains(errStr, "timeout") {
		errorCode = llms.ErrCodeTimeout
		message = "Ollama request timeout"
	} else {
		errorCode = ErrCodeInvalidRequest
		message = "Ollama request failed"
	}

	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// buildCallOptions merges configuration options with call-specific options
func (o *OllamaProvider) buildCallOptions(options ...core.Option) *llms.CallOptions {
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
	if o.config.TopK != nil {
		topK := *o.config.TopK
		callOpts.TopK = &topK
	}

	// Apply call-specific options
	for _, opt := range options {
		callOpts.ApplyCallOption(opt)
	}

	return callOpts
}

// Factory function for creating Ollama providers
func NewOllamaProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewOllamaProvider(config)
	}
}
