package chatmodels

import (
	"errors"
	"fmt"
)

// ChatModelError represents a custom error type for chat model operations.
// It includes context about the operation that failed and wraps the underlying error.
type ChatModelError struct {
	Err      error
	Fields   map[string]any
	Op       string
	Model    string
	Provider string
	Code     string
}

// Error implements the error interface.
func (e *ChatModelError) Error() string {
	if e.Provider != "" && e.Model != "" {
		return fmt.Sprintf("chatmodel %s %s (provider: %s): %v", e.Model, e.Op, e.Provider, e.Err)
	}
	if e.Model != "" {
		return fmt.Sprintf("chatmodel %s %s: %v", e.Model, e.Op, e.Err)
	}
	return fmt.Sprintf("chatmodel %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for error wrapping.
func (e *ChatModelError) Unwrap() error {
	return e.Err
}

// Error codes for different types of chat model errors.
const (
	ErrCodeConfigInvalid        = "config_invalid"
	ErrCodeInitialization       = "initialization_failed"
	ErrCodeGeneration           = "generation_failed"
	ErrCodeStreaming            = "streaming_failed"
	ErrCodeRateLimit            = "rate_limit"
	ErrCodeInvalidInput         = "invalid_input"
	ErrCodeNetworkError         = "network_error"
	ErrCodeTimeout              = "timeout"
	ErrCodeMaxRetries           = "max_retries_exceeded"
	ErrCodeInvalidResponse      = "invalid_response"
	ErrCodeModelNotFound        = "model_not_found"
	ErrCodeProviderNotSupported = "provider_not_supported"
	ErrCodeAuthentication       = "authentication_failed"
	ErrCodeQuotaExceeded        = "quota_exceeded"
	ErrCodeResourceExhausted    = "resource_exhausted"
)

// NewChatModelError creates a new ChatModelError.
func NewChatModelError(op, model, provider, code string, err error) *ChatModelError {
	return &ChatModelError{
		Op:       op,
		Model:    model,
		Provider: provider,
		Code:     code,
		Err:      err,
		Fields:   make(map[string]any),
	}
}

// WithField adds a context field to the error.
func (e *ChatModelError) WithField(key string, value any) *ChatModelError {
	if e.Fields == nil {
		e.Fields = make(map[string]any)
	}
	e.Fields[key] = value
	return e
}

// ValidationError represents configuration validation errors.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ProviderError represents errors that occur with specific providers.
type ProviderError struct {
	Err       error
	Provider  string
	Operation string
}

// Error implements the error interface.
func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider '%s' error during %s: %v", e.Provider, e.Operation, e.Err)
}

// Unwrap returns the underlying error.
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError.
func NewProviderError(provider, operation string, err error) *ProviderError {
	return &ProviderError{
		Provider:  provider,
		Operation: operation,
		Err:       err,
	}
}

// GenerationError represents errors that occur during message generation.
type GenerationError struct {
	Err        error
	Model      string
	Suggestion string
	Messages   int
	Tokens     int
}

// Error implements the error interface.
func (e *GenerationError) Error() string {
	msg := fmt.Sprintf("generation error for model '%s' with %d messages", e.Model, e.Messages)
	if e.Tokens > 0 {
		msg += fmt.Sprintf(" (%d tokens)", e.Tokens)
	}
	msg += fmt.Sprintf(": %v", e.Err)
	if e.Suggestion != "" {
		msg += ". Suggestion: " + e.Suggestion
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *GenerationError) Unwrap() error {
	return e.Err
}

// NewGenerationError creates a new GenerationError.
func NewGenerationError(model string, messages int, err error) *GenerationError {
	return &GenerationError{
		Model:    model,
		Messages: messages,
		Err:      err,
	}
}

// WithTokenCount adds token count information to the error.
func (e *GenerationError) WithTokenCount(tokens int) *GenerationError {
	e.Tokens = tokens
	return e
}

// WithSuggestion adds a suggestion to help resolve the generation error.
func (e *GenerationError) WithSuggestion(suggestion string) *GenerationError {
	e.Suggestion = suggestion
	return e
}

// StreamingError represents errors that occur during streaming operations.
type StreamingError struct {
	Err      error
	Model    string
	Duration string
}

// Error implements the error interface.
func (e *StreamingError) Error() string {
	msg := fmt.Sprintf("streaming error for model '%s'", e.Model)
	if e.Duration != "" {
		msg += " after " + e.Duration
	}
	msg += fmt.Sprintf(": %v", e.Err)
	return msg
}

// Unwrap returns the underlying error.
func (e *StreamingError) Unwrap() error {
	return e.Err
}

// NewStreamingError creates a new StreamingError.
func NewStreamingError(model string, err error) *StreamingError {
	return &StreamingError{
		Model: model,
		Err:   err,
	}
}

// WithDuration adds duration information to the streaming error.
func (e *StreamingError) WithDuration(duration string) *StreamingError {
	e.Duration = duration
	return e
}

// Common error variables for frequently used errors.
var (
	ErrChatModelNotFound    = errors.New("chat model not found")
	ErrInvalidConfig        = errors.New("invalid configuration")
	ErrProviderNotAvailable = errors.New("provider not available")
	ErrModelNotSupported    = errors.New("model not supported")
	ErrMaxRetriesExceeded   = errors.New("maximum retries exceeded")
	ErrContextCancelled     = errors.New("context canceled")
	ErrTimeout              = errors.New("operation timed out")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrQuotaExceeded        = errors.New("quota exceeded")
	ErrResourceExhausted    = errors.New("resource exhausted")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrNetworkError         = errors.New("network error")
)

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	var chatErr *ChatModelError
	if errors.As(err, &chatErr) {
		switch chatErr.Code {
		case ErrCodeRateLimit, ErrCodeNetworkError, ErrCodeTimeout, ErrCodeResourceExhausted:
			return true
		}
	}

	// Check for common retryable error conditions
	if errors.Is(err, ErrTimeout) || errors.Is(err, ErrRateLimitExceeded) ||
		errors.Is(err, ErrResourceExhausted) || errors.Is(err, ErrNetworkError) {
		return true
	}

	return false
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
}

// IsProviderError checks if an error is a provider error.
func IsProviderError(err error) bool {
	var provErr *ProviderError
	return errors.As(err, &provErr)
}

// IsGenerationError checks if an error is a generation error.
func IsGenerationError(err error) bool {
	var genErr *GenerationError
	return errors.As(err, &genErr)
}

// IsStreamingError checks if an error is a streaming error.
func IsStreamingError(err error) bool {
	var streamErr *StreamingError
	return errors.As(err, &streamErr)
}

// IsAuthenticationError checks if an error is an authentication error.
func IsAuthenticationError(err error) bool {
	var chatErr *ChatModelError
	if errors.As(err, &chatErr) {
		return chatErr.Code == ErrCodeAuthentication
	}
	return errors.Is(err, ErrAuthenticationFailed)
}

// IsQuotaError checks if an error is a quota-related error.
func IsQuotaError(err error) bool {
	var chatErr *ChatModelError
	if errors.As(err, &chatErr) {
		return chatErr.Code == ErrCodeQuotaExceeded
	}
	return errors.Is(err, ErrQuotaExceeded)
}
