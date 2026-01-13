// Package monitoring provides custom error types for the monitoring package.
package monitoring

import (
	"errors"
	"fmt"
)

// Error codes for monitoring operations.
const (
	ErrCodeInvalidConfig        = "invalid_config"
	ErrCodeProviderNotFound     = "provider_not_found"
	ErrCodeProviderError        = "provider_error"
	ErrCodeInitializationFailed = "initialization_failed"
	ErrCodeShutdownFailed       = "shutdown_failed"
	ErrCodeMetricError          = "metric_error"
	ErrCodeTraceError           = "trace_error"
	ErrCodeLogError             = "log_error"
	ErrCodeExportError          = "export_error"
	ErrCodeInvalidMetric        = "invalid_metric"
	ErrCodeInvalidTrace         = "invalid_trace"
	ErrCodeContextCanceled      = "context_canceled"
	ErrCodeContextTimeout       = "context_timeout"
)

// MonitoringError represents an error that occurred during monitoring operations.
type MonitoringError struct {
	Op      string // operation that failed
	Err     error  // underlying error
	Code    string // error code for programmatic handling
	Message string // human-readable message
}

// Error implements the error interface.
func (e *MonitoringError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("monitoring %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("monitoring %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("monitoring %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *MonitoringError) Unwrap() error {
	return e.Err
}

// NewMonitoringError creates a new MonitoringError.
func NewMonitoringError(op, code string, err error) *MonitoringError {
	return &MonitoringError{
		Op:   op,
		Code: code,
		Err:  err,
	}
}

// NewMonitoringErrorWithMessage creates a new MonitoringError with a custom message.
func NewMonitoringErrorWithMessage(op, code, message string, err error) *MonitoringError {
	return &MonitoringError{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsMonitoringError checks if an error is a MonitoringError.
func IsMonitoringError(err error) bool {
	var monitoringErr *MonitoringError
	return errors.As(err, &monitoringErr)
}

// AsMonitoringError attempts to convert an error to a MonitoringError.
func AsMonitoringError(err error) (*MonitoringError, bool) {
	var monitoringErr *MonitoringError
	if errors.As(err, &monitoringErr) {
		return monitoringErr, true
	}
	return nil, false
}
