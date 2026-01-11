// Package google provides an implementation of the llms.ChatModel interface
// using the Google Vertex AI API (Gemini models).
package google

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	schemaiface "github.com/lookatitude/beluga-ai/pkg/schema/iface"
)

// Provider constants.
const (
	ProviderName = "google"
	DefaultModel = "gemini-1.5-pro"

	// Error codes specific to Google Vertex AI.
	ErrCodeInvalidAPIKey  = "google_invalid_api_key"
	ErrCodeRateLimit      = "google_rate_limit"
	ErrCodeModelNotFound  = "google_model_not_found"
	ErrCodeInvalidRequest = "google_invalid_request"
	ErrCodeQuotaExceeded  = "google_quota_exceeded"
)

// GoogleProvider implements the ChatModel interface for Google Vertex AI Gemini models.
type GoogleProvider struct {
	metrics     llms.MetricsRecorder
	config      *llms.Config
	client      GoogleClient // Interface for Google API client
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
	modelName   string
	projectID   string
	location    string
	tools       []tools.Tool
}

// GoogleClient defines the interface for Google Vertex AI API client.
// This allows for dependency injection and testing.
type GoogleClient interface {
	GenerateContent(ctx context.Context, req *GoogleGenerateRequest) (*GoogleGenerateResponse, error)
	StreamGenerateContent(ctx context.Context, req *GoogleGenerateRequest) (<-chan *GoogleGenerateResponse, error)
}

// GoogleGenerateRequest represents a request to Google Vertex AI.
type GoogleGenerateRequest struct {
	Model       string
	Contents    []GoogleContent
	Temperature float32
	MaxTokens   int
	Tools       []GoogleTool
}

// GoogleGenerateResponse represents a response from Google Vertex AI.
type GoogleGenerateResponse struct {
	Candidates []GoogleCandidate
	Usage      GoogleUsage
}

// GoogleContent represents content in a Google Vertex AI request.
type GoogleContent struct {
	Role  string
	Parts []GooglePart
}

// GooglePart represents a part of content (text or function call).
type GooglePart struct {
	Text       string
	FunctionCall *GoogleFunctionCall
}

// GoogleCandidate represents a candidate response from Google Vertex AI.
type GoogleCandidate struct {
	Content      GoogleContent
	FinishReason string
}

// GoogleUsage represents token usage information.
type GoogleUsage struct {
	PromptTokens     int
	CandidatesTokens int
	TotalTokens      int
}

// GoogleTool represents a tool definition for Google Vertex AI.
type GoogleTool struct {
	FunctionDeclarations []GoogleFunctionDeclaration
}

// GoogleFunctionDeclaration represents a function declaration for tool calling.
type GoogleFunctionDeclaration struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// GoogleFunctionCall represents a function call in a response.
type GoogleFunctionCall struct {
	Name      string
	Arguments map[string]any
}

// NewGoogleProvider creates a new Google Vertex AI provider instance.
func NewGoogleProvider(config *llms.Config) (*GoogleProvider, error) {
	// Validate configuration
	if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
		return nil, fmt.Errorf("invalid Google configuration: %w", err)
	}

	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// Extract project ID and location from provider-specific config
	projectID, _ := config.ProviderSpecific["project_id"].(string)
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required in provider_specific config for Google Vertex AI")
	}

	location, _ := config.ProviderSpecific["location"].(string)
	if location == "" {
		location = "us-central1" // Default location
	}

	// TODO: Initialize Google Vertex AI client
	// This will involve:
	// 1. Creating a Vertex AI client with credentials
	// 2. Setting up the model endpoint
	// 3. Configuring retry and timeout settings
	var client GoogleClient
	// client = googleai.NewClient(...) // Placeholder

	provider := &GoogleProvider{
		config:    config,
		client:    client,
		modelName: modelName,
		projectID: projectID,
		location:  location,
		metrics:   llms.GetMetrics(),
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
func (g *GoogleProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = g.tracing.StartOperation(ctx, "google.generate", ProviderName, g.modelName)

	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	g.tracing.AddSpanAttributes(ctx, map[string]any{"input_size": inputSize})

	start := time.Now()

	// Record metrics
	g.metrics.IncrementActiveRequests(ctx, ProviderName, g.modelName)
	defer g.metrics.DecrementActiveRequests(ctx, ProviderName, g.modelName)

	// Convert messages to Google format
	contents := g.convertMessagesToGoogleContents(messages)

	// Prepare request
	temperature := float32(0.7) // Default temperature
	if g.config.Temperature != nil {
		temperature = *g.config.Temperature
	}
	maxTokens := 1024 // Default max tokens
	if g.config.MaxTokens != nil {
		maxTokens = *g.config.MaxTokens
	}
	req := &GoogleGenerateRequest{
		Model:       g.modelName,
		Contents:    contents,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	// Add tools if bound
	if len(g.tools) > 0 {
		req.Tools = g.convertToolsToGoogleTools(g.tools)
	}

	// Check that client is initialized (not yet implemented)
	if g.client == nil {
		err := llms.NewLLMErrorWithMessage("Generate", llms.ErrCodeInternalError,
			"google provider is not yet implemented: client initialization is required", nil)
		g.metrics.RecordError(ctx, ProviderName, g.modelName, llms.ErrCodeInternalError, time.Since(start))
		g.tracing.RecordError(ctx, err)
		return nil, err
	}

	// Execute with retry
	var resp *GoogleGenerateResponse
	var err error

	retryErr := common.RetryWithBackoff(ctx, g.retryConfig, "google.generate", func() error {
		resp, err = g.client.GenerateContent(ctx, req)
		return err
	})

	duration := time.Since(start)

	if retryErr != nil {
		g.metrics.RecordError(ctx, ProviderName, g.modelName, llms.GetLLMErrorCode(retryErr), duration)
		g.tracing.RecordError(ctx, retryErr)
		return nil, g.handleGoogleError("Generate", retryErr)
	}

	// Convert response to schema.Message
	if len(resp.Candidates) == 0 {
		return nil, llms.NewLLMError("Generate", ErrCodeInvalidRequest,
			errors.New("no candidates in response"))
	}

	candidate := resp.Candidates[0]
	aiMessage := g.convertGoogleContentToMessage(candidate.Content)

	// Record metrics
	g.metrics.RecordRequest(ctx, ProviderName, g.modelName, duration)
	g.metrics.RecordTokenUsage(ctx, ProviderName, g.modelName, resp.Usage.PromptTokens, resp.Usage.CandidatesTokens)

	return aiMessage, nil
}

// StreamChat implements the ChatModel interface.
func (g *GoogleProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	ctx = g.tracing.StartOperation(ctx, "google.stream_chat", ProviderName, g.modelName)

	ch := make(chan iface.AIMessageChunk, 10)

	// Convert messages to Google format
	contents := g.convertMessagesToGoogleContents(messages)

	// Prepare request
	temperature := float32(0.7) // Default temperature
	if g.config.Temperature != nil {
		temperature = *g.config.Temperature
	}
	maxTokens := 1024 // Default max tokens
	if g.config.MaxTokens != nil {
		maxTokens = *g.config.MaxTokens
	}
	req := &GoogleGenerateRequest{
		Model:       g.modelName,
		Contents:    contents,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	// Add tools if bound
	if len(g.tools) > 0 {
		req.Tools = g.convertToolsToGoogleTools(g.tools)
	}

	// Check that client is initialized (not yet implemented)
	if g.client == nil {
		err := llms.NewLLMErrorWithMessage("StreamChat", llms.ErrCodeInternalError,
			"google provider is not yet implemented: client initialization is required", nil)
		g.tracing.RecordError(ctx, err)
		go func() {
			defer close(ch)
			ch <- iface.AIMessageChunk{Err: err}
		}()
		return ch, nil
	}

	// Start streaming in goroutine
	go func() {
		defer close(ch)

		stream, err := g.client.StreamGenerateContent(ctx, req)
		if err != nil {
			ch <- iface.AIMessageChunk{
				Err: g.handleGoogleError("StreamChat", err),
			}
			return
		}

		for resp := range stream {
			if resp == nil {
				continue
			}

			if len(resp.Candidates) > 0 {
				candidate := resp.Candidates[0]
				chunk := iface.AIMessageChunk{
					Content: g.extractTextFromContent(candidate.Content),
				}
				ch <- chunk
			}
		}
	}()

	return ch, nil
}

// BindTools implements the ChatModel interface.
func (g *GoogleProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	// Create a new provider instance with tools bound
	newProvider := *g
	newProvider.tools = toolsToBind
	return &newProvider
}

// GetModelName implements the ChatModel interface.
func (g *GoogleProvider) GetModelName() string {
	return g.modelName
}

// GetProviderName implements the LLM interface.
func (g *GoogleProvider) GetProviderName() string {
	return ProviderName
}

// Invoke implements the core.Runnable interface.
func (g *GoogleProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	result, err := g.Generate(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	return result.GetContent(), nil
}

// Batch implements the core.Runnable interface.
func (g *GoogleProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := g.Invoke(ctx, input, options...)
		if err != nil {
			return results[:i], err
		}
		results[i] = result
	}
	return results, nil
}

// Stream implements the core.Runnable interface.
func (g *GoogleProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunks, err := g.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	outputCh := make(chan any, 10)
	go func() {
		defer close(outputCh)
		for chunk := range chunks {
			if chunk.Err != nil {
				outputCh <- chunk.Err
				return
			}
			outputCh <- chunk.Content
		}
	}()

	return outputCh, nil
}

// CheckHealth implements the HealthChecker interface.
func (g *GoogleProvider) CheckHealth() map[string]any {
	return map[string]any{
		"state":       "healthy",
		"provider":    "google",
		"model":       g.modelName,
		"timestamp":   time.Now().Unix(),
		"api_key_set": g.config.APIKey != "",
		"project_id":  g.projectID,
		"location":    g.location,
		"tools_count": len(g.tools),
	}
}

// Helper methods

func (g *GoogleProvider) convertMessagesToGoogleContents(messages []schema.Message) []GoogleContent {
	contents := make([]GoogleContent, 0, len(messages))
	for _, msg := range messages {
		role := "user"
		switch msg.GetType() {
		case schemaiface.RoleSystem:
			role = "system"
		case schemaiface.RoleAssistant:
			role = "model"
		case schemaiface.RoleHuman:
			role = "user"
		}

		content := GoogleContent{
			Role: role,
			Parts: []GooglePart{
				{Text: msg.GetContent()},
			},
		}
		contents = append(contents, content)
	}
	return contents
}

func (g *GoogleProvider) convertGoogleContentToMessage(content GoogleContent) schema.Message {
	text := g.extractTextFromContent(content)
	return schema.NewAIMessage(text)
}

func (g *GoogleProvider) extractTextFromContent(content GoogleContent) string {
	var parts []string
	for _, part := range content.Parts {
		if part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "")
}

func (g *GoogleProvider) convertToolsToGoogleTools(tools []tools.Tool) []GoogleTool {
	declarations := make([]GoogleFunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		inputSchema := tool.Definition().InputSchema
		var schemaMap map[string]any
		if inputSchema != nil {
			if m, ok := inputSchema.(map[string]any); ok {
				schemaMap = m
			} else {
				// Convert to map if needed
				schemaMap = make(map[string]any)
			}
		} else {
			schemaMap = make(map[string]any)
		}
		decl := GoogleFunctionDeclaration{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  g.convertToolSchema(schemaMap),
		}
		declarations = append(declarations, decl)
	}
	return []GoogleTool{
		{FunctionDeclarations: declarations},
	}
}

func (g *GoogleProvider) convertToolSchema(schema map[string]any) map[string]any {
	// Convert tool schema to Google Function Declaration format
	// This is a simplified conversion - full implementation would handle
	// JSON Schema to Google Function Declaration format conversion
	return schema
}

func (g *GoogleProvider) handleGoogleError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeRateLimit
		message = "Google Vertex AI API rate limit exceeded"
	} else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		errorCode = ErrCodeInvalidAPIKey
		message = "Google Vertex AI API authentication failed"
	} else if strings.Contains(errStr, "quota") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeQuotaExceeded
		message = "Google Vertex AI API quota exceeded"
	} else {
		errorCode = ErrCodeInvalidRequest
		message = "Google Vertex AI API request failed"
	}

	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// Factory function for creating Google providers.
func NewGoogleProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewGoogleProvider(config)
	}
}
