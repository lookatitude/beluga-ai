// Package gemini provides an implementation of the llms.ChatModel interface
// using the Google Gemini API (Google AI Studio).
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Provider constants.
const (
	ProviderName = "gemini"
	DefaultModel = "gemini-1.5-pro"

	// Error codes specific to Gemini.
	ErrCodeInvalidAPIKey  = "gemini_invalid_api_key"
	ErrCodeRateLimit      = "gemini_rate_limit"
	ErrCodeModelNotFound  = "gemini_model_not_found"
	ErrCodeInvalidRequest = "gemini_invalid_request"
	ErrCodeQuotaExceeded  = "gemini_quota_exceeded"
)

// GeminiProvider implements the ChatModel interface for Google Gemini models.
type GeminiProvider struct {
	metrics     llms.MetricsRecorder
	config      *llms.Config
	httpClient  *http.Client
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
	modelName   string
	baseURL     string
	apiKey      string
	tools       []tools.Tool
}

// Gemini API request/response structures.
type geminiGenerateRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools            []geminiTool            `json:"tools,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiGenerationConfig struct {
	Temperature     *float32 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	TopP            *float32 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type geminiGenerateResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// NewGeminiProvider creates a new Gemini provider instance.
func NewGeminiProvider(config *llms.Config) (*GeminiProvider, error) {
	// Validate configuration
	if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
		return nil, fmt.Errorf("invalid Gemini configuration: %w", err)
	}

	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// Set base URL
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	provider := &GeminiProvider{
		config:     config,
		httpClient: httpClient,
		modelName:  modelName,
		baseURL:    baseURL,
		apiKey:     config.APIKey,
		metrics:    llms.GetMetrics(),
		tracing:    common.NewTracingHelper(),
		retryConfig: &common.RetryConfig{
			MaxRetries: config.MaxRetries,
			Delay:      config.RetryDelay,
			Backoff:    config.RetryBackoff,
		},
	}

	return provider, nil
}

// Generate implements the ChatModel interface.
func (g *GeminiProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = g.tracing.StartOperation(ctx, "gemini.generate", ProviderName, g.modelName)

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

	retryErr := common.RetryWithBackoff(ctx, g.retryConfig, "gemini.generate", func() error {
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
func (g *GeminiProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	ctx = g.tracing.StartOperation(ctx, "gemini.stream", ProviderName, g.modelName)

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
func (g *GeminiProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	newProvider := *g // Create a copy
	newProvider.tools = make([]tools.Tool, len(toolsToBind))
	copy(newProvider.tools, toolsToBind)
	return &newProvider
}

// GetModelName implements the ChatModel interface.
func (g *GeminiProvider) GetModelName() string {
	return g.modelName
}

// GetProviderName returns the provider name.
func (g *GeminiProvider) GetProviderName() string {
	return ProviderName
}

// Invoke implements the Runnable interface.
func (g *GeminiProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return g.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface.
func (g *GeminiProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
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
func (g *GeminiProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
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
func (g *GeminiProvider) generateInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (schema.Message, error) {
	// Convert messages to Gemini format
	contents, err := g.convertMessages(messages)
	if err != nil {
		return nil, llms.NewLLMError("generateInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Build request
	req := &geminiGenerateRequest{
		Contents:         contents,
		GenerationConfig: &geminiGenerationConfig{},
	}

	// Apply call options
	if opts.MaxTokens != nil {
		req.GenerationConfig.MaxOutputTokens = opts.MaxTokens
	}
	if opts.Temperature != nil {
		req.GenerationConfig.Temperature = opts.Temperature
	}
	if opts.TopP != nil {
		req.GenerationConfig.TopP = opts.TopP
	}
	if len(opts.StopSequences) > 0 {
		req.GenerationConfig.StopSequences = opts.StopSequences
	}

	// Add tools if bound
	if len(g.tools) > 0 {
		req.Tools = g.convertTools(g.tools)
	}

	// Make API call
	resp, err := g.makeAPIRequest(ctx, "generateContent", req)
	if err != nil {
		return nil, g.handleGeminiError("generateInternal", err)
	}

	// Convert response to schema.Message
	return g.convertGeminiResponse(resp)
}

// streamInternal performs the actual streaming logic.
func (g *GeminiProvider) streamInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (<-chan iface.AIMessageChunk, error) {
	// Convert messages to Gemini format
	contents, err := g.convertMessages(messages)
	if err != nil {
		return nil, llms.NewLLMError("streamInternal", llms.ErrCodeInvalidRequest, err)
	}

	// Build request
	req := &geminiGenerateRequest{
		Contents:         contents,
		GenerationConfig: &geminiGenerationConfig{},
	}

	// Apply call options
	if opts.MaxTokens != nil {
		req.GenerationConfig.MaxOutputTokens = opts.MaxTokens
	}
	if opts.Temperature != nil {
		req.GenerationConfig.Temperature = opts.Temperature
	}
	if opts.TopP != nil {
		req.GenerationConfig.TopP = opts.TopP
	}
	if len(opts.StopSequences) > 0 {
		req.GenerationConfig.StopSequences = opts.StopSequences
	}

	// Add tools if bound
	if len(g.tools) > 0 {
		req.Tools = g.convertTools(g.tools)
	}

	outputChan := make(chan iface.AIMessageChunk)

	go func() {
		defer close(outputChan)

		// Make streaming API request
		url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s", g.baseURL, g.modelName, g.apiKey)

		reqBody, err := json.Marshal(req)
		if err != nil {
			outputChan <- iface.AIMessageChunk{
				Err: llms.WrapError("gemini.stream", err),
			}
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			outputChan <- iface.AIMessageChunk{
				Err: llms.WrapError("gemini.stream", err),
			}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := g.httpClient.Do(httpReq)
		if err != nil {
			outputChan <- iface.AIMessageChunk{
				Err: g.handleGeminiError("streamInternal", err),
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			outputChan <- iface.AIMessageChunk{
				Err: g.handleGeminiError("streamInternal", err),
			}
			return
		}

		// Parse streaming response (SSE format)
		decoder := json.NewDecoder(resp.Body)
		for {
			var streamResp struct {
				Candidates []geminiCandidate `json:"candidates"`
			}

			if err := decoder.Decode(&streamResp); err != nil {
				if err == io.EOF {
					break
				}
				outputChan <- iface.AIMessageChunk{
					Err: llms.WrapError("gemini.stream.decode", err),
				}
				return
			}

			if len(streamResp.Candidates) > 0 {
				candidate := streamResp.Candidates[0]
				if len(candidate.Content.Parts) > 0 {
					chunk := iface.AIMessageChunk{
						Content: candidate.Content.Parts[0].Text,
					}
					if candidate.FinishReason != "" {
						chunk.AdditionalArgs = map[string]any{
							"finish_reason": candidate.FinishReason,
						}
					}
					select {
					case outputChan <- chunk:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return outputChan, nil
}

// convertMessages converts schema messages to Gemini format.
func (g *GeminiProvider) convertMessages(messages []schema.Message) ([]geminiContent, error) {
	contents := make([]geminiContent, 0, len(messages))

	for _, msg := range messages {
		var role string
		var text string

		switch m := msg.(type) {
		case *schema.ChatMessage:
			switch m.GetType() {
			case schema.RoleSystem:
				role = "user" // Gemini doesn't have system role, use user
				text = m.GetContent()
			case schema.RoleHuman:
				role = "user"
				text = m.GetContent()
			case schema.RoleAssistant:
				role = "model"
				text = m.GetContent()
			default:
				continue
			}
		case *schema.AIMessage:
			role = "model"
			text = m.GetContent()
		default:
			continue
		}

		contents = append(contents, geminiContent{
			Role: role,
			Parts: []geminiPart{
				{Text: text},
			},
		})
	}

	if len(contents) == 0 {
		return nil, errors.New("no valid messages provided for Gemini conversion")
	}

	return contents, nil
}

// convertTools converts tools to Gemini format.
func (g *GeminiProvider) convertTools(tools []tools.Tool) []geminiTool {
	if len(tools) == 0 {
		return nil
	}

	declarations := make([]geminiFunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		def := tool.Definition()
		decl := geminiFunctionDeclaration{
			Name:        def.Name,
			Description: def.Description,
		}

		// Add parameters schema
		if def.InputSchema != nil {
			if schemaStr, ok := def.InputSchema.(string); ok && schemaStr != "" {
				var params map[string]any
				if err := json.Unmarshal([]byte(schemaStr), &params); err == nil {
					decl.Parameters = params
				}
			}
		}

		declarations = append(declarations, decl)
	}

	return []geminiTool{
		{FunctionDeclarations: declarations},
	}
}

// makeAPIRequest makes an HTTP request to the Gemini API.
func (g *GeminiProvider) makeAPIRequest(ctx context.Context, method string, reqBody interface{}) (*geminiGenerateResponse, error) {
	url := fmt.Sprintf("%s/models/%s:%s?key=%s", g.baseURL, g.modelName, method, g.apiKey)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiGenerateResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &geminiResp, nil
}

// convertGeminiResponse converts Gemini response to schema.Message.
func (g *GeminiProvider) convertGeminiResponse(resp *geminiGenerateResponse) (schema.Message, error) {
	if len(resp.Candidates) == 0 {
		return nil, errors.New("empty response from Gemini")
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, errors.New("empty content in Gemini response")
	}

	responseText := candidate.Content.Parts[0].Text
	aiMsg := schema.NewAIMessage(responseText)

	// Add usage information
	if resp.UsageMetadata != nil {
		args := aiMsg.AdditionalArgs()
		args["usage"] = map[string]int{
			"input_tokens":  resp.UsageMetadata.PromptTokenCount,
			"output_tokens": resp.UsageMetadata.CandidatesTokenCount,
			"total_tokens":  resp.UsageMetadata.TotalTokenCount,
		}
	}

	return aiMsg, nil
}

// buildCallOptions merges configuration options with call-specific options.
func (g *GeminiProvider) buildCallOptions(options ...core.Option) *llms.CallOptions {
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
	if len(g.config.StopSequences) > 0 {
		callOpts.StopSequences = g.config.StopSequences
	}

	// Apply call-specific options
	for _, opt := range options {
		callOpts.ApplyCallOption(opt)
	}

	return callOpts
}

// handleGeminiError converts Gemini errors to LLM errors.
func (g *GeminiProvider) handleGeminiError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeRateLimit
		message = "Gemini API rate limit exceeded"
	} else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		errorCode = ErrCodeInvalidAPIKey
		message = "Gemini API authentication failed"
	} else if strings.Contains(errStr, "model") || strings.Contains(errStr, "404") {
		errorCode = ErrCodeModelNotFound
		message = "Gemini model not found"
	} else if strings.Contains(errStr, "quota") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeQuotaExceeded
		message = "Gemini API quota exceeded"
	} else {
		errorCode = ErrCodeInvalidRequest
		message = "Gemini API request failed"
	}

	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// CheckHealth implements the HealthChecker interface.
func (g *GeminiProvider) CheckHealth() map[string]any {
	return map[string]any{
		"state":       "healthy",
		"provider":    "gemini",
		"model":       g.modelName,
		"timestamp":   time.Now().Unix(),
		"api_key_set": g.apiKey != "",
		"tools_count": len(g.tools),
	}
}

// Factory function for creating Gemini providers.
func NewGeminiProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewGeminiProvider(config)
	}
}
