// Package common provides shared utilities and helpers for LLM implementations.
// These utilities help maintain consistency across different LLM providers.
package common

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxRetries int
	Delay      time.Duration
	Backoff    float64
}

// DefaultRetryConfig returns default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		Delay:      time.Second,
		Backoff:    2.0,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic.
func RetryWithBackoff(ctx context.Context, config *RetryConfig, operation string, fn func() error) error {
	var lastErr error
	delay := config.Delay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * config.Backoff)
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if the error is not retryable
		if !llms.IsRetryableError(err) {
			break
		}
	}

	return llms.WrapError(operation, lastErr)
}

// MessageConverter provides utilities for converting between different message formats.
type MessageConverter struct{}

// NewMessageConverter creates a new MessageConverter.
func NewMessageConverter() *MessageConverter {
	return &MessageConverter{}
}

// ConvertToSchemaMessages converts messages (placeholder for future conversion).
func (mc *MessageConverter) ConvertToSchemaMessages(messages []schema.Message) ([]schema.Message, error) {
	if len(messages) == 0 {
		return nil, errors.New("no messages provided")
	}

	// For now, just return the messages as-is since they're already schema messages
	return messages, nil
}

// ExtractSystemMessage extracts and removes the system message from a message slice.
func (mc *MessageConverter) ExtractSystemMessage(messages []schema.Message) (*string, []schema.Message) {
	if len(messages) == 0 {
		return nil, messages
	}

	if chatMsg, ok := messages[0].(*schema.ChatMessage); ok && chatMsg.GetType() == schema.RoleSystem {
		content := chatMsg.GetContent()
		return &content, messages[1:]
	}

	return nil, messages
}

// ToolCallConverter provides utilities for converting tool calls between formats.
type ToolCallConverter struct{}

// NewToolCallConverter creates a new ToolCallConverter.
func NewToolCallConverter() *ToolCallConverter {
	return &ToolCallConverter{}
}

// ConvertToolCallsToSchema converts tool calls (placeholder).
func (tcc *ToolCallConverter) ConvertToolCallsToSchema(toolCalls []schema.ToolCall) []schema.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	// For now, just return the tool calls as-is since they're already schema tool calls
	return toolCalls
}

// TracingHelper provides utilities for tracing with OpenTelemetry.
type TracingHelper struct{}

// NewTracingHelper creates a new TracingHelper.
func NewTracingHelper() *TracingHelper {
	return &TracingHelper{}
}

// StartOperation starts a new trace span for an LLM operation.
func (th *TracingHelper) StartOperation(ctx context.Context, operation, provider, model string) context.Context {
	// This is a placeholder - the actual tracing is handled in the main package
	// to avoid circular dependencies. The main TracingHelper is in the llms package.
	return ctx
}

// RecordError records an error on the current span.
func (th *TracingHelper) RecordError(ctx context.Context, err error) {
	// This is a placeholder - the actual tracing is handled in the main package
	// to avoid circular dependencies. The main TracingHelper is in the llms package.
}

// AddSpanAttributes adds attributes to the current span.
func (th *TracingHelper) AddSpanAttributes(ctx context.Context, attrs map[string]any) {
	// This is a placeholder - the actual tracing is handled in the main package
	// to avoid circular dependencies. The main TracingHelper is in the llms package.
}

// EndSpan ends the current span.
func (th *TracingHelper) EndSpan(ctx context.Context) {
	// This is a placeholder - the actual tracing is handled in the main package
	// to avoid circular dependencies. The main TracingHelper is in the llms package.
}

// MetricsHelper provides utilities for metrics recording.
type MetricsHelper struct {
	metrics *llms.Metrics
}

// NewMetricsHelper creates a new MetricsHelper.
func NewMetricsHelper(metrics *llms.Metrics) *MetricsHelper {
	return &MetricsHelper{metrics: metrics}
}

// RecordRequest records request metrics.
func (mh *MetricsHelper) RecordRequest(ctx context.Context, provider, model string, duration time.Duration) {
	if mh.metrics != nil {
		mh.metrics.RecordRequest(ctx, provider, model, duration)
	}
}

// RecordError records error metrics.
func (mh *MetricsHelper) RecordError(ctx context.Context, provider, model, errorCode string, duration time.Duration) {
	if mh.metrics != nil {
		mh.metrics.RecordError(ctx, provider, model, errorCode, duration)
	}
}

// RecordTokenUsage records token usage metrics.
func (mh *MetricsHelper) RecordTokenUsage(ctx context.Context, provider, model string, inputTokens, outputTokens int) {
	if mh.metrics != nil {
		mh.metrics.RecordTokenUsage(ctx, provider, model, inputTokens, outputTokens)
	}
}

// PromptBuilder provides utilities for building prompts from messages.
type PromptBuilder struct{}

// NewPromptBuilder creates a new PromptBuilder.
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildChatPrompt builds a chat-style prompt from messages.
func (pb *PromptBuilder) BuildChatPrompt(messages []schema.Message, systemPrompt *string) string {
	var prompt strings.Builder

	// Add system prompt if provided
	if systemPrompt != nil && *systemPrompt != "" {
		_, _ = prompt.WriteString("System: ")    //nolint:errcheck // strings.Builder.WriteString rarely fails
		_, _ = prompt.WriteString(*systemPrompt) //nolint:errcheck // strings.Builder.WriteString rarely fails
		_, _ = prompt.WriteString("\n\n")        //nolint:errcheck // strings.Builder.WriteString rarely fails
	}

	// Add conversation messages
	for _, msg := range messages {
		switch m := msg.(type) {
		case *schema.ChatMessage:
			switch m.GetType() {
			case schema.RoleSystem:
				if systemPrompt == nil || *systemPrompt == "" {
					_, _ = prompt.WriteString("System: ")     //nolint:errcheck // strings.Builder.WriteString rarely fails
					_, _ = prompt.WriteString(m.GetContent()) //nolint:errcheck // strings.Builder.WriteString rarely fails
					_, _ = prompt.WriteString("\n\n")         //nolint:errcheck // strings.Builder.WriteString rarely fails
				}
			case schema.RoleHuman:
				_, _ = prompt.WriteString("Human: ")      //nolint:errcheck // strings.Builder.WriteString rarely fails
				_, _ = prompt.WriteString(m.GetContent()) //nolint:errcheck // strings.Builder.WriteString rarely fails
				_, _ = prompt.WriteString("\n\nAssistant: ")
			case schema.RoleAssistant:
				_, _ = prompt.WriteString(m.GetContent())
				_, _ = prompt.WriteString("\n\n")
			}
		case *schema.AIMessage:
			_, _ = prompt.WriteString(m.GetContent())
			_, _ = prompt.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(prompt.String())
}

// BuildCompletionPrompt builds a simple completion-style prompt.
func (pb *PromptBuilder) BuildCompletionPrompt(prompt string, systemPrompt *string) string {
	var result strings.Builder

	if systemPrompt != nil && *systemPrompt != "" {
		_, _ = result.WriteString(*systemPrompt)
		_, _ = result.WriteString("\n\n")
	}

	_, _ = result.WriteString(prompt)
	return result.String()
}

// ValidationHelper provides utilities for input validation.
type ValidationHelper struct{}

// NewValidationHelper creates a new ValidationHelper.
func NewValidationHelper() *ValidationHelper {
	return &ValidationHelper{}
}

// ValidateMessages validates a slice of messages.
func (vh *ValidationHelper) ValidateMessages(messages []schema.Message) error {
	if len(messages) == 0 {
		return errors.New("messages cannot be empty")
	}

	for i, msg := range messages {
		if msg == nil {
			return fmt.Errorf("message at index %d is nil", i)
		}

		content := msg.GetContent()
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("message at index %d has empty content", i)
		}
	}

	return nil
}

// ValidatePrompt validates a prompt string.
func (vh *ValidationHelper) ValidatePrompt(prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return errors.New("prompt cannot be empty")
	}
	return nil
}

// ValidateConfig validates basic configuration.
func (vh *ValidationHelper) ValidateConfig(config *llms.Config) error {
	if config == nil {
		return errors.New("configuration cannot be nil")
	}

	if config.Provider == "" {
		return errors.New("provider cannot be empty")
	}

	if config.ModelName == "" {
		return errors.New("model name cannot be empty")
	}

	// Provider-specific validation
	switch config.Provider {
	case "openai", "anthropic":
		if config.APIKey == "" {
			return fmt.Errorf("API key is required for provider %s", config.Provider)
		}
	case "mock":
		// No API key required for mock
	default:
		// Allow other providers
	}

	return nil
}
