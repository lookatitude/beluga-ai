package iface

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/codes"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// PromptFormatter defines the interface for formatting prompts into various output formats.
// This follows the Interface Segregation Principle by focusing solely on formatting concerns.
type PromptFormatter interface {
	// Format takes input variables and returns a formatted prompt value
	Format(ctx context.Context, inputs map[string]interface{}) (interface{}, error)

	// GetInputVariables returns the list of expected input variable names
	GetInputVariables() []string
}

// TemplateEngine defines the interface for template processing engines.
// This allows for different template engines (Go templates, Jinja2, etc.) to be used.
type TemplateEngine interface {
	// Parse parses a template string and returns a parsed template
	Parse(name, template string) (ParsedTemplate, error)

	// ExtractVariables extracts variable names from a template string
	ExtractVariables(template string) ([]string, error)
}

// ParsedTemplate represents a parsed template that can be executed
type ParsedTemplate interface {
	// Execute executes the template with the given data and writes to the writer
	Execute(data interface{}) (string, error)
}

// VariableValidator defines the interface for validating template variables
type VariableValidator interface {
	// Validate checks if the provided variables meet the requirements
	Validate(required []string, provided map[string]interface{}) error

	// ValidateTypes validates the types of provided variables
	ValidateTypes(variables map[string]interface{}) error
}

// PromptValue represents the formatted output of a prompt template.
// It serves as an intermediate representation that can be easily converted
// into either a raw string or a structured list of messages.
type PromptValue interface {
	// ToString returns the prompt as a single string
	ToString() string

	// ToMessages returns the prompt as a slice of schema.Message objects
	ToMessages() []schema.Message
}

// TemplateManager defines the interface for managing prompt templates
type TemplateManager interface {
	// CreateTemplate creates a new template from a string
	CreateTemplate(name, templateStr string) (Template, error)

	// GetTemplate retrieves a cached template by name
	GetTemplate(name string) (Template, bool)

	// ListTemplates returns a list of available template names
	ListTemplates() []string

	// DeleteTemplate removes a template from cache
	DeleteTemplate(name string) error
}

// Template defines the interface for individual templates
type Template interface {
	PromptFormatter

	// Name returns the template name
	Name() string

	// Validate validates the template structure
	Validate() error
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	// Check performs a health check and returns an error if unhealthy
	Check(ctx context.Context) error
}

// Options holds internal configuration options
type Options struct {
	Config         *Config
	Validator      VariableValidator
	TemplateEngine TemplateEngine
	Metrics        Metrics
	Tracer         Tracer
	Logger         Logger
	HealthChecker  HealthChecker
}

// Config holds configuration for the prompts package
type Config struct {
	// Template configuration
	DefaultTemplateTimeout Duration `mapstructure:"default_template_timeout" yaml:"default_template_timeout" env:"PROMPTS_DEFAULT_TEMPLATE_TIMEOUT" default:"30s"`

	// Validation configuration
	MaxTemplateSize     int  `mapstructure:"max_template_size" yaml:"max_template_size" env:"PROMPTS_MAX_TEMPLATE_SIZE" default:"1048576"` // 1MB
	ValidateVariables   bool `mapstructure:"validate_variables" yaml:"validate_variables" env:"PROMPTS_VALIDATE_VARIABLES" default:"true"`
	StrictVariableCheck bool `mapstructure:"strict_variable_check" yaml:"strict_variable_check" env:"PROMPTS_STRICT_VARIABLE_CHECK" default:"false"`

	// Caching configuration
	EnableTemplateCache bool     `mapstructure:"enable_template_cache" yaml:"enable_template_cache" env:"PROMPTS_ENABLE_TEMPLATE_CACHE" default:"true"`
	CacheTTL            Duration `mapstructure:"cache_ttl" yaml:"cache_ttl" env:"PROMPTS_CACHE_TTL" default:"5m"`
	MaxCacheSize        int      `mapstructure:"max_cache_size" yaml:"max_cache_size" env:"PROMPTS_MAX_CACHE_SIZE" default:"100"`

	// Observability configuration
	EnableMetrics bool `mapstructure:"enable_metrics" yaml:"enable_metrics" env:"PROMPTS_ENABLE_METRICS" default:"true"`
	EnableTracing bool `mapstructure:"enable_tracing" yaml:"enable_tracing" env:"PROMPTS_ENABLE_TRACING" default:"true"`

	// Adapter configuration
	DefaultAdapterType string `mapstructure:"default_adapter_type" yaml:"default_adapter_type" env:"PROMPTS_DEFAULT_ADAPTER_TYPE" default:"default"`
}

// Metrics interface for collecting observability metrics
type Metrics interface {
	// Template metrics
	RecordTemplateCreated(templateType string)
	RecordTemplateExecuted(templateName string, duration float64)
	RecordTemplateError(templateName string, errorType string)

	// Formatting metrics
	RecordFormattingRequest(adapterType string, duration float64)
	RecordFormattingError(adapterType string, errorType string)

	// Variable validation metrics
	RecordValidationRequest()
	RecordValidationError(errorType string)

	// Cache metrics
	RecordCacheHit()
	RecordCacheMiss()
	RecordCacheSize(size int64)

	// Adapter metrics
	RecordAdapterRequest(adapterType string)
	RecordAdapterError(adapterType string, errorType string)
}

// Tracer interface for distributed tracing
type Tracer interface {
	Start(ctx context.Context, spanName string, opts ...interface{}) (context.Context, Span)
}

// Span interface for tracing spans
type Span interface {
	End()
	RecordError(err error)
	SetStatus(code codes.Code, msg string)
	SetAttributes(kv ...interface{})
}

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// Additional interfaces for OpenTelemetry compatibility
type Int64Counter interface {
	Add(ctx context.Context, incr int64, options ...interface{})
}

type Float64Histogram interface {
	Record(ctx context.Context, incr float64, options ...interface{})
}

type Int64UpDownCounter interface {
	Add(ctx context.Context, incr int64, options ...interface{})
}

type Meter interface {
	// Placeholder for OpenTelemetry meter interface
	// In practice, this would be metric.Meter from go.opentelemetry.io/otel/metric
}

type Duration interface {
	String() string
}

// Option represents a functional option
type Option func(*Options)

// TracerNoOp provides a no-op implementation of the Tracer interface
type TracerNoOp struct{}

func (t *TracerNoOp) Start(ctx context.Context, spanName string, opts ...interface{}) (context.Context, Span) {
	return ctx, &SpanNoOp{}
}

// SpanNoOp provides a no-op implementation of the Span interface
type SpanNoOp struct{}

func (s *SpanNoOp) End()                                  {}
func (s *SpanNoOp) RecordError(err error)                 {}
func (s *SpanNoOp) SetStatus(code codes.Code, msg string) {}
func (s *SpanNoOp) SetAttributes(kv ...interface{})       {}

// LoggerNoOp provides a no-op implementation of the Logger interface
type LoggerNoOp struct{}

func (l *LoggerNoOp) Debug(msg string, keysAndValues ...interface{}) {}
func (l *LoggerNoOp) Info(msg string, keysAndValues ...interface{})  {}
func (l *LoggerNoOp) Warn(msg string, keysAndValues ...interface{})  {}
func (l *LoggerNoOp) Error(msg string, keysAndValues ...interface{}) {}

// Error codes for the prompts package
const (
	ErrCodeTemplateParse      = "template_parse_error"
	ErrCodeTemplateExecute    = "template_execute_error"
	ErrCodeVariableMissing    = "variable_missing"
	ErrCodeVariableInvalid    = "variable_invalid"
	ErrCodeValidationFailed   = "validation_failed"
	ErrCodeCacheError         = "cache_error"
	ErrCodeAdapterError       = "adapter_error"
	ErrCodeConfigurationError = "configuration_error"
	ErrCodeTimeout            = "timeout"
)

// PromptError represents a custom error type for the prompts package
type PromptError struct {
	Code    string                 // Error code for programmatic handling
	Message string                 // Human-readable error message
	Op      string                 // Operation that failed
	Err     error                  // Underlying error (if any)
	Context map[string]interface{} // Additional context information
}

// Error implements the error interface
func (e *PromptError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("prompts %s: %s (%s)", e.Op, e.Message, e.Err.Error())
	}
	return fmt.Sprintf("prompts %s: %s", e.Op, e.Message)
}

// Unwrap returns the underlying error
func (e *PromptError) Unwrap() error {
	return e.Err
}

// NewTemplateParseError creates a new template parse error
func NewTemplateParseError(op string, templateName string, err error) *PromptError {
	return &PromptError{
		Code:    ErrCodeTemplateParse,
		Message: fmt.Sprintf("failed to parse template '%s'", templateName),
		Op:      op,
		Err:     err,
		Context: map[string]interface{}{
			"template_name": templateName,
		},
	}
}

// NewTemplateExecuteError creates a new template execution error
func NewTemplateExecuteError(op string, templateName string, err error) *PromptError {
	return &PromptError{
		Code:    ErrCodeTemplateExecute,
		Message: fmt.Sprintf("failed to execute template '%s'", templateName),
		Op:      op,
		Err:     err,
		Context: map[string]interface{}{
			"template_name": templateName,
		},
	}
}

// NewVariableMissingError creates a new variable missing error
func NewVariableMissingError(op string, variableName string, templateName string) *PromptError {
	return &PromptError{
		Code:    ErrCodeVariableMissing,
		Message: fmt.Sprintf("required variable '%s' is missing", variableName),
		Op:      op,
		Context: map[string]interface{}{
			"variable_name": variableName,
			"template_name": templateName,
		},
	}
}

// NewVariableInvalidError creates a new variable invalid error
func NewVariableInvalidError(op string, variableName string, expectedType string, actualType string) *PromptError {
	return &PromptError{
		Code:    ErrCodeVariableInvalid,
		Message: fmt.Sprintf("variable '%s' has invalid type: expected %s, got %s", variableName, expectedType, actualType),
		Op:      op,
		Context: map[string]interface{}{
			"variable_name": variableName,
			"expected_type": expectedType,
			"actual_type":   actualType,
		},
	}
}

// NewValidationError creates a new validation error
func NewValidationError(op string, details string, err error) *PromptError {
	return &PromptError{
		Code:    ErrCodeValidationFailed,
		Message: fmt.Sprintf("validation failed: %s", details),
		Op:      op,
		Err:     err,
		Context: map[string]interface{}{
			"validation_details": details,
		},
	}
}

// NewCacheError creates a new cache error
func NewCacheError(op string, details string, err error) *PromptError {
	return &PromptError{
		Code:    ErrCodeCacheError,
		Message: fmt.Sprintf("cache operation failed: %s", details),
		Op:      op,
		Err:     err,
		Context: map[string]interface{}{
			"cache_operation": details,
		},
	}
}

// NewAdapterError creates a new adapter error
func NewAdapterError(op string, adapterType string, err error) *PromptError {
	return &PromptError{
		Code:    ErrCodeAdapterError,
		Message: fmt.Sprintf("adapter '%s' operation failed", adapterType),
		Op:      op,
		Err:     err,
		Context: map[string]interface{}{
			"adapter_type": adapterType,
		},
	}
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(op string, details string, err error) *PromptError {
	return &PromptError{
		Code:    ErrCodeConfigurationError,
		Message: fmt.Sprintf("configuration error: %s", details),
		Op:      op,
		Err:     err,
		Context: map[string]interface{}{
			"config_details": details,
		},
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(op string, timeout string) *PromptError {
	return &PromptError{
		Code:    ErrCodeTimeout,
		Message: fmt.Sprintf("operation timed out after %s", timeout),
		Op:      op,
		Context: map[string]interface{}{
			"timeout": timeout,
		},
	}
}
