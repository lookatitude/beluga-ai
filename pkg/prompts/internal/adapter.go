package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Note: options struct is defined in the main prompts package

// BaseAdapter provides common functionality for prompt adapters.
type BaseAdapter struct {
	metrics   iface.Metrics
	tracer    iface.Tracer
	logger    iface.Logger
	config    *iface.Config
	name      string
	variables []string
}

// Name returns the adapter name.
func (b *BaseAdapter) Name() string {
	return b.name
}

// GetInputVariables returns the list of expected input variables.
func (b *BaseAdapter) GetInputVariables() []string {
	return b.variables
}

// DefaultPromptAdapter is a basic implementation that handles simple string formatting.
// It can be used for models that expect a single string prompt.
type DefaultPromptAdapter struct {
	Template string
	BaseAdapter
}

// NewDefaultPromptAdapter creates a new DefaultPromptAdapter.
func NewDefaultPromptAdapter(name, template string, inputVariables []string, opts ...iface.Option) (*DefaultPromptAdapter, error) {
	if name == "" {
		return nil, iface.NewValidationError("new_adapter", "adapter name cannot be empty", nil)
	}
	if template == "" {
		return nil, iface.NewValidationError("new_adapter", "template cannot be empty", nil)
	}

	// Apply options
	options := &iface.Options{}
	for _, opt := range opts {
		opt(options)
	}

	adapter := &DefaultPromptAdapter{
		BaseAdapter: BaseAdapter{
			name:      name,
			variables: inputVariables,
			metrics:   options.Metrics,
			tracer:    options.Tracer,
			logger:    options.Logger,
			config:    options.Config,
		},
		Template: template,
	}

	// Record metrics
	if adapter.metrics != nil {
		adapter.metrics.RecordAdapterRequest("default")
	}

	return adapter, nil
}

// Format interpolates the input variables into the template string.
// For this default adapter, it returns a single string.
func (dpa *DefaultPromptAdapter) Format(ctx context.Context, inputs map[string]any) (any, error) {
	ctx, span := dpa.tracer.Start(ctx, "default_adapter.format")
	defer span.End()

	start := time.Now()

	// Validate inputs if enabled
	if dpa.config != nil && dpa.config.ValidateVariables {
		if err := dpa.validateInputs(inputs); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if dpa.metrics != nil {
				dpa.metrics.RecordValidationError("input_validation")
			}
			return nil, err
		}
	}

	formattedPrompt := dpa.Template
	for _, key := range dpa.variables {
		value, ok := inputs[key]
		if !ok {
			err := iface.NewVariableMissingError("format", key, dpa.name)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if dpa.metrics != nil {
				dpa.metrics.RecordAdapterError("default", "missing_variable")
			}
			return "", err
		}

		valStr, ok := value.(string)
		if !ok {
			err := iface.NewVariableInvalidError("format", key, "string", fmt.Sprintf("%T", value))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if dpa.metrics != nil {
				dpa.metrics.RecordAdapterError("default", "invalid_type")
			}
			return "", err
		}

		// Simple placeholder replacement
		placeholder := fmt.Sprintf("{{.%s}}", key)
		formattedPrompt = strings.ReplaceAll(formattedPrompt, placeholder, valStr)
	}

	duration := time.Since(start).Seconds()
	span.SetAttributes(attribute.Float64("duration_seconds", duration))

	if dpa.metrics != nil {
		dpa.metrics.RecordFormattingRequest("default", duration)
	}

	if dpa.logger != nil {
		dpa.logger.Info("formatted prompt",
			"adapter", dpa.name,
			"template_length", len(dpa.Template),
			"result_length", len(formattedPrompt))
	}

	return formattedPrompt, nil
}

// validateInputs validates that all required inputs are present and have correct types.
func (dpa *DefaultPromptAdapter) validateInputs(inputs map[string]any) error {
	for _, key := range dpa.variables {
		value, ok := inputs[key]
		if !ok {
			return iface.NewVariableMissingError("validate_inputs", key, dpa.name)
		}

		if _, ok := value.(string); !ok {
			return iface.NewVariableInvalidError("validate_inputs", key, "string", fmt.Sprintf("%T", value))
		}
	}

	return nil
}

// ChatPromptAdapter for models that use a list of messages (e.g., OpenAI Chat, Anthropic Claude).
type ChatPromptAdapter struct {
	SystemMessageTemplate string
	UserMessageTemplate   string
	HistoryKey            string
	BaseAdapter
}

// NewChatPromptAdapter creates a new ChatPromptAdapter.
func NewChatPromptAdapter(name, systemTemplate, userTemplate string, inputVariables []string, opts ...iface.Option) (*ChatPromptAdapter, error) {
	if name == "" {
		return nil, iface.NewValidationError("new_chat_adapter", "adapter name cannot be empty", nil)
	}
	if userTemplate == "" {
		return nil, iface.NewValidationError("new_chat_adapter", "user template cannot be empty", nil)
	}

	// Apply options
	options := &iface.Options{}
	for _, opt := range opts {
		opt(options)
	}

	adapter := &ChatPromptAdapter{
		BaseAdapter: BaseAdapter{
			name:      name,
			variables: inputVariables,
			metrics:   options.Metrics,
			tracer:    options.Tracer,
			logger:    options.Logger,
			config:    options.Config,
		},
		SystemMessageTemplate: systemTemplate,
		UserMessageTemplate:   userTemplate,
		HistoryKey:            "history",
	}

	// Record metrics
	if adapter.metrics != nil {
		adapter.metrics.RecordAdapterRequest("chat")
	}

	return adapter, nil
}

// Format formats inputs into a list of messages for chat models.
func (cpa *ChatPromptAdapter) Format(ctx context.Context, inputs map[string]any) (any, error) {
	ctx, span := cpa.tracer.Start(ctx, "chat_adapter.format",
		trace.WithAttributes(
			attribute.String("adapter.name", cpa.name),
			attribute.Int("inputs.count", len(inputs)),
		))
	defer span.End()

	start := time.Now()

	// Validate inputs if enabled
	if cpa.config != nil && cpa.config.ValidateVariables {
		if err := cpa.validateInputs(inputs); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if cpa.metrics != nil {
				cpa.metrics.RecordValidationError("input_validation")
			}
			return nil, err
		}
	}

	var messages []schema.Message

	// Add system message if template provided
	if cpa.SystemMessageTemplate != "" {
		systemContent, err := cpa.formatTemplate(cpa.SystemMessageTemplate, inputs)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if cpa.metrics != nil {
				cpa.metrics.RecordAdapterError("chat", "system_format")
			}
			return nil, err
		}
		messages = append(messages, NewChatMessage("system", systemContent))
	}

	// Add chat history if provided
	if history, ok := inputs[cpa.HistoryKey]; ok {
		if historyMsgs, ok := history.([]schema.Message); ok {
			messages = append(messages, historyMsgs...)
		}
	}

	// Add user message
	userContent, err := cpa.formatTemplate(cpa.UserMessageTemplate, inputs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if cpa.metrics != nil {
			cpa.metrics.RecordAdapterError("chat", "user_format")
		}
		return nil, err
	}
	messages = append(messages, NewChatMessage("user", userContent))

	duration := time.Since(start).Seconds()
	span.SetAttributes(attribute.Float64("duration_seconds", duration))

	if cpa.metrics != nil {
		cpa.metrics.RecordFormattingRequest("chat", duration)
	}

	if cpa.logger != nil {
		cpa.logger.Info("formatted chat messages",
			"adapter", cpa.name,
			"message_count", len(messages))
	}

	return messages, nil
}

// formatTemplate formats a single template string with inputs.
func (cpa *ChatPromptAdapter) formatTemplate(template string, inputs map[string]any) (string, error) {
	formatted := template
	for _, key := range cpa.variables {
		value, ok := inputs[key]
		if !ok {
			return "", iface.NewVariableMissingError("format_template", key, cpa.name)
		}

		valStr, ok := value.(string)
		if !ok {
			return "", iface.NewVariableInvalidError("format_template", key, "string", fmt.Sprintf("%T", value))
		}

		placeholder := fmt.Sprintf("{{.%s}}", key)
		formatted = strings.ReplaceAll(formatted, placeholder, valStr)
	}
	return formatted, nil
}

// validateInputs validates that all required inputs are present and have correct types.
func (cpa *ChatPromptAdapter) validateInputs(inputs map[string]any) error {
	for _, key := range cpa.variables {
		value, ok := inputs[key]
		if !ok {
			return iface.NewVariableMissingError("validate_inputs", key, cpa.name)
		}

		if _, ok := value.(string); !ok {
			return iface.NewVariableInvalidError("validate_inputs", key, "string", fmt.Sprintf("%T", value))
		}
	}

	return nil
}

// Helper function to create a schema.Message (useful for chat model adapters).
func NewChatMessage(role schema.MessageType, content string) schema.Message {
	return schema.NewChatMessage(role, content)
}

// Compile-time checks to ensure implementations satisfy interfaces.
var (
	_ iface.PromptFormatter = (*DefaultPromptAdapter)(nil)
	_ iface.PromptFormatter = (*ChatPromptAdapter)(nil)
)
