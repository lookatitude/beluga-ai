// Package prompts provides interfaces and implementations for creating and formatting
// prompts to be sent to language models. It follows the Beluga AI Framework design patterns
// with proper separation of concerns, dependency injection, observability, and error handling.
//
// The package supports multiple prompt formats including string templates and chat message sequences,
// with built-in validation, caching, and metrics collection.
package prompts

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
	"github.com/lookatitude/beluga-ai/pkg/prompts/internal"
	"go.opentelemetry.io/otel"
)

// PromptManager is the main entry point for the prompts package.
// It provides factory methods for creating templates and adapters with proper dependency injection.
type PromptManager struct {
	config    *Config
	metrics   iface.Metrics
	tracer    iface.Tracer
	logger    iface.Logger
	validator iface.VariableValidator
}

// NewPromptManager creates a new PromptManager with the given configuration and dependencies.
// This follows the factory pattern and dependency injection principles.
// The manager provides factory methods for creating templates and adapters.
//
// Parameters:
//   - opts: Optional configuration functions (WithConfig, WithMetrics, WithTracer, etc.)
//
// Returns:
//   - *PromptManager: A new prompt manager instance
//   - error: Configuration or initialization errors
//
// Example:
//
//	manager, err := prompts.NewPromptManager(
//	    prompts.WithConfig(prompts.DefaultConfig()),
//	    prompts.WithMetrics(metrics),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/prompts/basic/main.go.
func NewPromptManager(opts ...Option) (*PromptManager, error) {
	// Apply options
	options := &iface.Options{
		Config: DefaultConfig(),
	}
	for _, opt := range opts {
		opt(options)
	}

	// Initialize metrics if enabled
	var metrics iface.Metrics
	if options.Config.EnableMetrics {
		if options.Metrics == nil {
			// Create default metrics if not provided
			meter := otel.Meter("beluga.prompts")
			tracer := otel.Tracer("beluga.prompts")
			otelMetrics, err := NewMetrics(meter, tracer)
			if err != nil {
				return nil, fmt.Errorf("failed to create metrics: %w", err)
			}
			// Wrap OTEL metrics to implement iface.Metrics interface
			metrics = &MetricsWrapper{otelMetrics: otelMetrics}
		} else {
			metrics = options.Metrics
		}
	}

	// Initialize tracer if enabled
	var tracer iface.Tracer
	if options.Config.EnableTracing {
		if options.Tracer == nil {
			// Create a simple no-op tracer
			tracer = &iface.TracerNoOp{}
		} else {
			tracer = options.Tracer
		}
	} else {
		// Create a simple no-op tracer
		tracer = &iface.TracerNoOp{}
	}

	// Initialize logger
	var logger iface.Logger
	if options.Logger == nil {
		logger = &iface.LoggerNoOp{}
	} else {
		logger = options.Logger
	}

	manager := &PromptManager{
		config:    options.Config,
		metrics:   metrics,
		tracer:    tracer,
		logger:    logger,
		validator: options.Validator,
	}

	return manager, nil
}

// NewStringTemplate creates a new string prompt template.
// This is a factory method that properly injects dependencies.
// String templates support variable substitution using {{variable}} syntax.
//
// Parameters:
//   - name: Unique name for the template (used for caching and metrics)
//   - template: Template string with {{variable}} placeholders
//
// Returns:
//   - iface.Template: A new template instance
//   - error: Validation errors if name is empty or template is invalid
//
// Example:
//
//	template, err := manager.NewStringTemplate(
//	    "greeting",
//	    "Hello, {{name}}! Welcome to {{company}}.",
//	)
//	result, err := template.Format(map[string]any{"name": "Alice", "company": "Beluga"})
//
// Example usage can be found in examples/prompts/basic/main.go.
func (pm *PromptManager) NewStringTemplate(name, template string) (iface.Template, error) {
	if name == "" {
		return nil, NewValidationError("new_string_template", "template name cannot be empty", nil)
	}
	return internal.NewStringPromptTemplate(name, template,
		WithConfig(pm.config),
		WithMetrics(pm.metrics),
		WithTracer(pm.tracer),
		WithLogger(pm.logger),
		WithValidator(pm.validator),
	)
}

// NewDefaultAdapter creates a new default prompt adapter.
// This is a factory method that properly injects dependencies.
func (pm *PromptManager) NewDefaultAdapter(name, template string, variables []string) (iface.PromptFormatter, error) {
	if name == "" {
		return nil, NewValidationError("new_default_adapter", "adapter name cannot be empty", nil)
	}
	return internal.NewDefaultPromptAdapter(name, template, variables,
		WithConfig(pm.config),
		WithMetrics(pm.metrics),
		WithTracer(pm.tracer),
		WithLogger(pm.logger),
		WithValidator(pm.validator),
	)
}

// NewChatAdapter creates a new chat prompt adapter.
// This is a factory method that properly injects dependencies.
func (pm *PromptManager) NewChatAdapter(name, systemTemplate, userTemplate string, variables []string) (iface.PromptFormatter, error) {
	if name == "" {
		return nil, NewValidationError("new_chat_adapter", "adapter name cannot be empty", nil)
	}
	return internal.NewChatPromptAdapter(name, systemTemplate, userTemplate, variables,
		WithConfig(pm.config),
		WithMetrics(pm.metrics),
		WithTracer(pm.tracer),
		WithLogger(pm.logger),
		WithValidator(pm.validator),
	)
}

// GetConfig returns the current configuration.
func (pm *PromptManager) GetConfig() *Config {
	return pm.config
}

// GetMetrics returns the metrics collector.
func (pm *PromptManager) GetMetrics() iface.Metrics {
	return pm.metrics
}

// Check implements the HealthChecker interface.
func (pm *PromptManager) Check(ctx context.Context) error {
	// Test basic template creation
	_, err := pm.NewStringTemplate("health_check", "Hello {{.name}}")
	if err != nil {
		return NewValidationError("health_check", "failed to create test template", err)
	}

	// Test basic adapter creation
	_, err = pm.NewDefaultAdapter("health_check", "Test {{.value}}", []string{"value"})
	if err != nil {
		return NewValidationError("health_check", "failed to create test adapter", err)
	}

	return nil
}

// HealthCheck performs a health check on the prompt manager.
func (pm *PromptManager) HealthCheck(ctx context.Context) error {
	return pm.Check(ctx)
}

// Convenience functions for direct usage without PromptManager

// NewStringPromptTemplate creates a string prompt template with default configuration.
// This is a convenience function for simple use cases.
func NewStringPromptTemplate(name, template string, opts ...Option) (iface.Template, error) {
	manager, err := NewPromptManager(opts...)
	if err != nil {
		return nil, err
	}
	return manager.NewStringTemplate(name, template)
}

// NewDefaultPromptAdapter creates a default prompt adapter with default configuration.
// This is a convenience function for simple use cases.
func NewDefaultPromptAdapter(name, template string, variables []string, opts ...Option) (iface.PromptFormatter, error) {
	manager, err := NewPromptManager(opts...)
	if err != nil {
		return nil, err
	}
	return manager.NewDefaultAdapter(name, template, variables)
}

// NewChatPromptAdapter creates a chat prompt adapter with default configuration.
// This is a convenience function for simple use cases.
func NewChatPromptAdapter(name, systemTemplate, userTemplate string, variables []string, opts ...Option) (iface.PromptFormatter, error) {
	manager, err := NewPromptManager(opts...)
	if err != nil {
		return nil, err
	}
	return manager.NewChatAdapter(name, systemTemplate, userTemplate, variables)
}

// Compile-time checks to ensure implementations satisfy interfaces.
var _ iface.HealthChecker = (*PromptManager)(nil)
