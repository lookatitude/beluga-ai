package iface

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrorSeverity represents the severity level of an error.
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// String returns the string representation of the error severity.
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// MonitoringError represents errors specific to monitoring operations.
// It provides structured error information for programmatic error handling.
type MonitoringError struct {
	Timestamp time.Time       `json:"timestamp"`
	Cause     error           `json:"-"`
	Context   context.Context `json:"-"`
	Metadata  map[string]any  `json:"metadata"`
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Operation string          `json:"operation"`
	Component string          `json:"component"`
	Severity  ErrorSeverity   `json:"severity"`
}

// Error implements the error interface.
func (e *MonitoringError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for error wrapping compatibility.
func (e *MonitoringError) Unwrap() error {
	return e.Cause
}

// NewMonitoringError creates a new MonitoringError with the given code and message.
func NewMonitoringError(code, message string, args ...any) *MonitoringError {
	return &MonitoringError{
		Code:      code,
		Message:   fmt.Sprintf(message, args...),
		Timestamp: time.Now(),
		Severity:  SeverityMedium,
		Metadata:  make(map[string]any),
	}
}

// NewMonitoringErrorWithContext creates a new MonitoringError with context information.
func NewMonitoringErrorWithContext(ctx context.Context, code, message string, args ...any) *MonitoringError {
	err := NewMonitoringError(code, message, args...)
	err.Context = ctx
	return err
}

// WrapError wraps an existing error with monitoring context.
func WrapError(cause error, code, message string, args ...any) *MonitoringError {
	return &MonitoringError{
		Code:      code,
		Message:   fmt.Sprintf(message, args...),
		Cause:     cause,
		Timestamp: time.Now(),
		Severity:  SeverityMedium,
		Metadata:  make(map[string]any),
	}
}

// WrapErrorWithContext wraps an existing error with monitoring context and operation details.
func WrapErrorWithContext(ctx context.Context, cause error, code, message string, args ...any) *MonitoringError {
	return &MonitoringError{
		Code:      code,
		Message:   fmt.Sprintf(message, args...),
		Cause:     cause,
		Timestamp: time.Now(),
		Context:   ctx,
		Severity:  SeverityMedium,
		Metadata:  make(map[string]any),
	}
}

// Common error codes.
const (
	ErrCodeInvalidConfig        = "invalid_config"
	ErrCodeProviderNotFound     = "provider_not_found"
	ErrCodeProviderDisabled     = "provider_disabled"
	ErrCodeConnectionFailed     = "connection_failed"
	ErrCodeMetricRecording      = "metric_recording_failed"
	ErrCodeTracingFailed        = "tracing_failed"
	ErrCodeLoggingFailed        = "logging_failed"
	ErrCodeHealthCheckFailed    = "health_check_failed"
	ErrCodeSafetyViolation      = "safety_violation"
	ErrCodeEthicalViolation     = "ethical_violation"
	ErrCodeInvalidParameters    = "invalid_parameters"
	ErrCodeTimeout              = "timeout"
	ErrCodeRateLimitExceeded    = "rate_limit_exceeded"
	ErrCodeResourceExhausted    = "resource_exhausted"
	ErrCodePermissionDenied     = "permission_denied"
	ErrCodeAuthenticationFailed = "authentication_failed"
	ErrCodeValidationFailed     = "validation_failed"
	ErrCodeInternalError        = "internal_error"
	ErrCodeServiceUnavailable   = "service_unavailable"
)

// IsMonitoringError checks if an error is a MonitoringError with the given code.
func IsMonitoringError(err error, code string) bool {
	var monErr *MonitoringError
	if !AsMonitoringError(err, &monErr) {
		return false
	}
	return monErr.Code == code
}

// AsMonitoringError attempts to cast an error to MonitoringError.
func AsMonitoringError(err error, target **MonitoringError) bool {
	for err != nil {
		monErr := &MonitoringError{}
		if errors.As(err, &monErr) {
			*target = monErr
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}

// WithOperation sets the operation that failed.
func (e *MonitoringError) WithOperation(operation string) *MonitoringError {
	e.Operation = operation
	return e
}

// WithComponent sets the component where the error occurred.
func (e *MonitoringError) WithComponent(component string) *MonitoringError {
	e.Component = component
	return e
}

// WithSeverity sets the error severity.
func (e *MonitoringError) WithSeverity(severity ErrorSeverity) *MonitoringError {
	e.Severity = severity
	return e
}

// WithMetadata adds metadata to the error.
func (e *MonitoringError) WithMetadata(key string, value any) *MonitoringError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// WithMetadataMap adds multiple metadata entries to the error.
func (e *MonitoringError) WithMetadataMap(metadata map[string]any) *MonitoringError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	for k, v := range metadata {
		e.Metadata[k] = v
	}
	return e
}

// GetOperation returns the operation that failed.
func (e *MonitoringError) GetOperation() string {
	return e.Operation
}

// GetComponent returns the component where the error occurred.
func (e *MonitoringError) GetComponent() string {
	return e.Component
}

// GetSeverity returns the error severity.
func (e *MonitoringError) GetSeverity() ErrorSeverity {
	return e.Severity
}

// GetMetadata returns the error metadata.
func (e *MonitoringError) GetMetadata() map[string]any {
	if e.Metadata == nil {
		return make(map[string]any)
	}
	result := make(map[string]any)
	for k, v := range e.Metadata {
		result[k] = v
	}
	return result
}

// IsCritical returns true if the error is critical.
func (e *MonitoringError) IsCritical() bool {
	return e.Severity == SeverityCritical
}

// IsHighSeverity returns true if the error is high severity or above.
func (e *MonitoringError) IsHighSeverity() bool {
	return e.Severity >= SeverityHigh
}

// ShouldRetry returns true if the operation should be retried based on the error.
func (e *MonitoringError) ShouldRetry() bool {
	switch e.Code {
	case ErrCodeTimeout, ErrCodeConnectionFailed, ErrCodeServiceUnavailable, ErrCodeRateLimitExceeded:
		return true
	default:
		return false
	}
}

// GetRetryDelay returns the recommended retry delay for the error.
func (e *MonitoringError) GetRetryDelay() time.Duration {
	switch e.Code {
	case ErrCodeRateLimitExceeded:
		return 60 * time.Second
	case ErrCodeTimeout, ErrCodeConnectionFailed:
		return 5 * time.Second
	case ErrCodeServiceUnavailable:
		return 10 * time.Second
	default:
		return 0
	}
}
