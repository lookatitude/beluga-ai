// Package groq provides an implementation of the llms.ChatModel interface
// using the Groq API (fast inference with open models).
package groq

import (
	"context"
	"encoding/json"
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
)

// Provider constants.
const (
	ProviderName = "groq"
	DefaultModel = "llama-3.1-70b-versatile"

	// Error codes specific to Groq.
	ErrCodeInvalidAPIKey  = "groq_invalid_api_key"
	ErrCodeRateLimit      = "groq_rate_limit"
	ErrCodeModelNotFound  = "groq_model_not_found"
	ErrCodeInvalidRequest = "groq_invalid_request"
	ErrCodeQuotaExceeded  = "groq_quota_exceeded"
)

// GroqProvider implements the ChatModel interface for Groq models.
type GroqProvider struct {
	metrics     llms.MetricsRecorder
	config      *llms.Config
	client      GroqClient // Interface for Groq API client
	tracing     *common.TracingHelper
	retryConfig *common.RetryConfig
	modelName   string
	tools       []tools.Tool
}

// GroqClient defines the interface for Groq API client.
// This allows for dependency injection and testing.
type GroqClient interface {
	CreateChatCompletion(ctx context.Context, req *GroqChatRequest) (*GroqChatResponse, error)
	CreateChatCompletionStream(ctx context.Context, req *GroqChatRequest) (<-chan *GroqChatResponse, error)
}

// GroqChatRequest represents a request to Groq API.
type GroqChatRequest struct {
	Model       string
	Messages    []GroqMessage
	Temperature float32
	MaxTokens   int
	Tools       []GroqTool
	ToolChoice  string
}

// GroqChatResponse represents a response from Groq API.
type GroqChatResponse struct {
	ID      string
	Choices []GroqChoice
	Usage   GroqUsage
}

// GroqMessage represents a message in Groq format.
type GroqMessage struct {
	Role    string
	Content string
}

// GroqChoice represents a choice in Groq response.
type GroqChoice struct {
	Message      GroqMessage
	FinishReason string
}

// GroqUsage represents token usage information.
type GroqUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// GroqTool represents a tool definition for Groq.
type GroqTool struct {
	Type     string
	Function GroqFunction
}

// GroqFunction represents a function definition for tool calling.
type GroqFunction struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// NewGroqProvider creates a new Groq provider instance.
func NewGroqProvider(config *llms.Config) (*GroqProvider, error) {
	// Validate configuration
	if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
		return nil, fmt.Errorf("invalid Groq configuration: %w", err)
	}

	// Set default model if not specified
	modelName := config.ModelName
	if modelName == "" {
		modelName = DefaultModel
	}

	// TODO: Initialize Groq client
	// This will involve:
	// 1. Creating a Groq client with API key
	// 2. Setting up the base URL (https://api.groq.com/openai/v1)
	// 3. Configuring retry and timeout settings
	var client GroqClient
	// client = groq.NewClient(config.APIKey) // Placeholder

	provider := &GroqProvider{
		config:    config,
		client:    client,
		modelName: modelName,
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
func (g *GroqProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	// Start tracing
	ctx = g.tracing.StartOperation(ctx, "groq.generate", ProviderName, g.modelName)

	inputSize := 0
	for _, m := range messages {
		inputSize += len(m.GetContent())
	}
	g.tracing.AddSpanAttributes(ctx, map[string]any{"input_size": inputSize})

	start := time.Now()

	// Record metrics
	g.metrics.IncrementActiveRequests(ctx, ProviderName, g.modelName)
	defer g.metrics.DecrementActiveRequests(ctx, ProviderName, g.modelName)

	// Convert messages to Groq format
	groqMessages := g.convertMessagesToGroqMessages(messages)

	// Prepare request
	req := &GroqChatRequest{
		Model:       g.modelName,
		Messages:    groqMessages,
		Temperature: g.config.Temperature,
		MaxTokens:   g.config.MaxTokens,
	}

	// Add tools if bound
	if len(g.tools) > 0 {
		req.Tools = g.convertToolsToGroqTools(g.tools)
		req.ToolChoice = "auto"
	}

	// Execute with retry
	var resp *GroqChatResponse
	var err error

	retryErr := common.RetryWithBackoff(ctx, g.retryConfig, "groq.generate", func() error {
		resp, err = g.client.CreateChatCompletion(ctx, req)
		return err
	})

	duration := time.Since(start)

	if retryErr != nil {
		g.metrics.RecordError(ctx, ProviderName, g.modelName, llms.GetErrorCode(retryErr), duration)
		g.tracing.RecordError(ctx, retryErr)
		return nil, g.handleGroqError("Generate", retryErr)
	}

	// Convert response to schema.Message
	if len(resp.Choices) == 0 {
		return nil, llms.NewLLMError("Generate", ErrCodeInvalidRequest,
			errors.New("no choices in response"))
	}

	choice := resp.Choices[0]
	aiMessage := schema.NewAIMessage(choice.Message.Content)

	// Record metrics
	g.metrics.RecordRequest(ctx, ProviderName, g.modelName, duration)
	g.metrics.RecordTokens(ctx, ProviderName, g.modelName, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return aiMessage, nil
}

// StreamChat implements the ChatModel interface.
func (g *GroqProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
	// Start tracing
	ctx = g.tracing.StartOperation(ctx, "groq.stream_chat", ProviderName, g.modelName)

	ch := make(chan iface.AIMessageChunk, 10)

	// Convert messages to Groq format
	groqMessages := g.convertMessagesToGroqMessages(messages)

	// Prepare request
	req := &GroqChatRequest{
		Model:       g.modelName,
		Messages:    groqMessages,
		Temperature: g.config.Temperature,
		MaxTokens:   g.config.MaxTokens,
	}

	// Add tools if bound
	if len(g.tools) > 0 {
		req.Tools = g.convertToolsToGroqTools(g.tools)
		req.ToolChoice = "auto"
	}

	// Start streaming in goroutine
	go func() {
		defer close(ch)

		stream, err := g.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			ch <- iface.AIMessageChunk{
				Err: g.handleGroqError("StreamChat", err),
			}
			return
		}

		for resp := range stream {
			if resp == nil {
				continue
			}

			if len(resp.Choices) > 0 {
				choice := resp.Choices[0]
				chunk := iface.AIMessageChunk{
					Content: choice.Message.Content,
				}
				ch <- chunk
			}
		}
	}()

	return ch, nil
}

// BindTools implements the ChatModel interface.
func (g *GroqProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	// Create a new provider instance with tools bound
	newProvider := *g
	newProvider.tools = toolsToBind
	return &newProvider
}

// GetModelName implements the ChatModel interface.
func (g *GroqProvider) GetModelName() string {
	return g.modelName
}

// GetProviderName implements the LLM interface.
func (g *GroqProvider) GetProviderName() string {
	return ProviderName
}

// Invoke implements the core.Runnable interface.
func (g *GroqProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
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
func (g *GroqProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
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
func (g *GroqProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
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
func (g *GroqProvider) CheckHealth() map[string]any {
	return map[string]any{
		"state":       "healthy",
		"provider":    "groq",
		"model":       g.modelName,
		"timestamp":   time.Now().Unix(),
		"api_key_set": g.config.APIKey != "",
		"tools_count": len(g.tools),
	}
}

// Helper methods

func (g *GroqProvider) convertMessagesToGroqMessages(messages []schema.Message) []GroqMessage {
	groqMessages := make([]GroqMessage, 0, len(messages))
	for _, msg := range messages {
		role := "user"
		switch msg.(type) {
		case *schema.SystemMessage:
			role = "system"
		case *schema.AIMessage:
			role = "assistant"
		case *schema.HumanMessage:
			role = "user"
		}

		groqMessages = append(groqMessages, GroqMessage{
			Role:    role,
			Content: msg.GetContent(),
		})
	}
	return groqMessages
}

func (g *GroqProvider) convertToolsToGroqTools(tools []tools.Tool) []GroqTool {
	groqTools := make([]GroqTool, 0, len(tools))
	for _, tool := range tools {
		groqTool := GroqTool{
			Type: "function",
			Function: GroqFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  g.convertToolSchema(tool.Schema()),
			},
		}
		groqTools = append(groqTools, groqTool)
	}
	return groqTools
}

func (g *GroqProvider) convertToolSchema(schema map[string]any) map[string]any {
	// Convert tool schema to Groq function parameters format
	// This is a simplified conversion - full implementation would handle
	// JSON Schema to Groq function format conversion
	return schema
}

func (g *GroqProvider) handleGroqError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	var message string

	errStr := err.Error()
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeRateLimit
		message = "Groq API rate limit exceeded"
	} else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		errorCode = ErrCodeInvalidAPIKey
		message = "Groq API authentication failed"
	} else if strings.Contains(errStr, "quota") || strings.Contains(errStr, "429") {
		errorCode = ErrCodeQuotaExceeded
		message = "Groq API quota exceeded"
	} else {
		errorCode = ErrCodeInvalidRequest
		message = "Groq API request failed"
	}

	return llms.NewLLMErrorWithMessage(operation, errorCode, message, err)
}

// Factory function for creating Groq providers.
func NewGroqProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
	return func(config *llms.Config) (iface.ChatModel, error) {
		return NewGroqProvider(config)
	}
}
