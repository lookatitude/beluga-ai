package iface

import "fmt"

// MonitoringError represents errors specific to monitoring operations.
// It provides structured error information for programmatic error handling.
type MonitoringError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Cause   error  // Underlying error that caused this error
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
func NewMonitoringError(code, message string, args ...interface{}) *MonitoringError {
	return &MonitoringError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with monitoring context.
func WrapError(cause error, code, message string, args ...interface{}) *MonitoringError {
	return &MonitoringError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeInvalidConfig     = "invalid_config"
	ErrCodeProviderNotFound  = "provider_not_found"
	ErrCodeProviderDisabled  = "provider_disabled"
	ErrCodeConnectionFailed  = "connection_failed"
	ErrCodeMetricRecording   = "metric_recording_failed"
	ErrCodeTracingFailed     = "tracing_failed"
	ErrCodeLoggingFailed     = "logging_failed"
	ErrCodeHealthCheckFailed = "health_check_failed"
	ErrCodeSafetyViolation   = "safety_violation"
	ErrCodeEthicalViolation  = "ethical_violation"
	ErrCodeInvalidParameters = "invalid_parameters"
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
		if monErr, ok := err.(*MonitoringError); ok {
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
