package iface

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMonitoringError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *MonitoringError
		expected string
	}{
		{
			name: "error without cause",
			err: &MonitoringError{
				Code:    "test_error",
				Message: "Test error message",
			},
			expected: "Test error message",
		},
		{
			name: "error with cause",
			err: &MonitoringError{
				Code:    "test_error",
				Message: "Test error message",
				Cause:   errors.New("underlying error"),
			},
			expected: "Test error message: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestMonitoringError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &MonitoringError{
		Code:    "test_error",
		Message: "Test error message",
		Cause:   underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())
}

func TestNewMonitoringError(t *testing.T) {
	err := NewMonitoringError("test_code", "Test message %s", "arg1")

	assert.Equal(t, "test_code", err.Code)
	assert.Equal(t, "Test message arg1", err.Message)
	assert.NotZero(t, err.Timestamp)
	assert.Equal(t, SeverityMedium, err.Severity)
	assert.NotNil(t, err.Metadata)
}

func TestNewMonitoringErrorWithContext(t *testing.T) {
	ctx := context.Background()
	err := NewMonitoringErrorWithContext(ctx, "test_code", "Test message")

	assert.Equal(t, "test_code", err.Code)
	assert.Equal(t, ctx, err.Context)
}

func TestWrapError(t *testing.T) {
	underlying := errors.New("underlying error")
	err := WrapError(underlying, "test_code", "Wrapped message")

	assert.Equal(t, "test_code", err.Code)
	assert.Equal(t, "Wrapped message", err.Message)
	assert.Equal(t, underlying, err.Cause)
}

func TestWrapErrorWithContext(t *testing.T) {
	ctx := context.Background()
	underlying := errors.New("underlying error")
	err := WrapErrorWithContext(ctx, underlying, "test_code", "Wrapped message")

	assert.Equal(t, "test_code", err.Code)
	assert.Equal(t, ctx, err.Context)
	assert.Equal(t, underlying, err.Cause)
}

func TestMonitoringError_WithOperation(t *testing.T) {
	err := NewMonitoringError("test_code", "Test message")
	result := err.WithOperation("test_operation")

	assert.Equal(t, err, result) // Should return self for chaining
	assert.Equal(t, "test_operation", err.Operation)
}

func TestMonitoringError_WithComponent(t *testing.T) {
	err := NewMonitoringError("test_code", "Test message")
	result := err.WithComponent("test_component")

	assert.Equal(t, err, result)
	assert.Equal(t, "test_component", err.Component)
}

func TestMonitoringError_WithSeverity(t *testing.T) {
	err := NewMonitoringError("test_code", "Test message")
	result := err.WithSeverity(SeverityHigh)

	assert.Equal(t, err, result)
	assert.Equal(t, SeverityHigh, err.Severity)
}

func TestMonitoringError_WithMetadata(t *testing.T) {
	err := NewMonitoringError("test_code", "Test message")
	result := err.WithMetadata("key1", "value1")

	assert.Equal(t, err, result)
	assert.Equal(t, "value1", err.Metadata["key1"])
}

func TestMonitoringError_WithMetadataMap(t *testing.T) {
	err := NewMonitoringError("test_code", "Test message")
	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	result := err.WithMetadataMap(metadata)

	assert.Equal(t, err, result)
	assert.Equal(t, "value1", err.Metadata["key1"])
	assert.Equal(t, 42, err.Metadata["key2"])
}

func TestMonitoringError_Getters(t *testing.T) {
	err := NewMonitoringError("test_code", "Test message").
		WithOperation("test_op").
		WithComponent("test_comp").
		WithSeverity(SeverityCritical).
		WithMetadata("key", "value")

	assert.Equal(t, "test_op", err.GetOperation())
	assert.Equal(t, "test_comp", err.GetComponent())
	assert.Equal(t, SeverityCritical, err.GetSeverity())

	metadata := err.GetMetadata()
	assert.Equal(t, "value", metadata["key"])
}

func TestMonitoringError_IsCritical(t *testing.T) {
	criticalErr := NewMonitoringError("test", "msg").WithSeverity(SeverityCritical)
	highErr := NewMonitoringError("test", "msg").WithSeverity(SeverityHigh)
	mediumErr := NewMonitoringError("test", "msg").WithSeverity(SeverityMedium)

	assert.True(t, criticalErr.IsCritical())
	assert.False(t, highErr.IsCritical())
	assert.False(t, mediumErr.IsCritical())
}

func TestMonitoringError_IsHighSeverity(t *testing.T) {
	criticalErr := NewMonitoringError("test", "msg").WithSeverity(SeverityCritical)
	highErr := NewMonitoringError("test", "msg").WithSeverity(SeverityHigh)
	mediumErr := NewMonitoringError("test", "msg").WithSeverity(SeverityMedium)
	lowErr := NewMonitoringError("test", "msg").WithSeverity(SeverityLow)

	assert.True(t, criticalErr.IsHighSeverity())
	assert.True(t, highErr.IsHighSeverity())
	assert.False(t, mediumErr.IsHighSeverity())
	assert.False(t, lowErr.IsHighSeverity())
}

func TestMonitoringError_ShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"timeout should retry", ErrCodeTimeout, true},
		{"connection failed should retry", ErrCodeConnectionFailed, true},
		{"service unavailable should retry", ErrCodeServiceUnavailable, true},
		{"rate limit should retry", ErrCodeRateLimitExceeded, true},
		{"invalid config should not retry", ErrCodeInvalidConfig, false},
		{"internal error should not retry", ErrCodeInternalError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMonitoringError(tt.code, "test message")
			assert.Equal(t, tt.expected, err.ShouldRetry())
		})
	}
}

func TestMonitoringError_GetRetryDelay(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected time.Duration
	}{
		{"rate limit delay", ErrCodeRateLimitExceeded, 60 * time.Second},
		{"timeout delay", ErrCodeTimeout, 5 * time.Second},
		{"connection failed delay", ErrCodeConnectionFailed, 5 * time.Second},
		{"service unavailable delay", ErrCodeServiceUnavailable, 10 * time.Second},
		{"no retry delay", ErrCodeInvalidConfig, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMonitoringError(tt.code, "test message")
			assert.Equal(t, tt.expected, err.GetRetryDelay())
		})
	}
}

func TestIsMonitoringError(t *testing.T) {
	monErr := NewMonitoringError("test_code", "test message")
	regularErr := errors.New("regular error")

	assert.True(t, IsMonitoringError(monErr, "test_code"))
	assert.False(t, IsMonitoringError(monErr, "wrong_code"))
	assert.False(t, IsMonitoringError(regularErr, "test_code"))
}

func TestAsMonitoringError(t *testing.T) {
	monErr := NewMonitoringError("test_code", "test message")
	var target *MonitoringError

	// Test successful cast
	assert.True(t, AsMonitoringError(monErr, &target))
	assert.Equal(t, monErr, target)

	// Test wrapped error - should return the wrapped error, not the original
	wrappedErr := WrapError(monErr, "wrap_code", "wrapped message")
	assert.True(t, AsMonitoringError(wrappedErr, &target))
	assert.Equal(t, wrappedErr, target) // Should be the wrapped error

	// Test regular error
	regularErr := errors.New("regular error")
	assert.False(t, AsMonitoringError(regularErr, &target))
}

func TestErrorSeverity_String(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		expected string
	}{
		{SeverityLow, "low"},
		{SeverityMedium, "medium"},
		{SeverityHigh, "high"},
		{SeverityCritical, "critical"},
		{ErrorSeverity(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.String())
		})
	}
}

func BenchmarkNewMonitoringError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMonitoringError("test_code", "Test message %d", i)
	}
}

func BenchmarkWrapError(b *testing.B) {
	underlying := errors.New("underlying error")
	for i := 0; i < b.N; i++ {
		WrapError(underlying, "test_code", "Wrapped message %d", i)
	}
}

func BenchmarkMonitoringError_Error(b *testing.B) {
	err := NewMonitoringError("test_code", "Test message").WithCause(errors.New("underlying"))
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// WithCause is a helper method for testing
func (e *MonitoringError) WithCause(cause error) *MonitoringError {
	e.Cause = cause
	return e
}
