package core

import (
	"errors"
	"testing"
)

func TestFrameworkError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *FrameworkError
		expected string
	}{
		{
			name: "error without cause",
			err: &FrameworkError{
				Type:    ErrorTypeValidation,
				Message: "Invalid input",
			},
			expected: "[validation] Invalid input",
		},
		{
			name: "error with cause",
			err: &FrameworkError{
				Type:    ErrorTypeNetwork,
				Message: "Connection failed",
				Cause:   errors.New("timeout"),
			},
			expected: "[network] Connection failed: timeout",
		},
		{
			name: "error with code",
			err: &FrameworkError{
				Type:    ErrorTypeInternal,
				Message: "System error",
				Code:    "INTERNAL_ERROR",
			},
			expected: "[internal] System error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("FrameworkError.Error() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFrameworkError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &FrameworkError{
		Type:    ErrorTypeInternal,
		Message: "Wrapped error",
		Cause:   cause,
	}

	if err.Unwrap() != cause {
		t.Errorf("FrameworkError.Unwrap() = %v, expected %v", err.Unwrap(), cause)
	}
}

func TestNewValidationError(t *testing.T) {
	cause := errors.New("field is required")
	err := NewValidationError("Input validation failed", cause)

	if err.Type != ErrorTypeValidation {
		t.Errorf("NewValidationError() Type = %v, expected %v", err.Type, ErrorTypeValidation)
	}
	if err.Message != "Input validation failed" {
		t.Errorf("NewValidationError() Message = %q, expected %q", err.Message, "Input validation failed")
	}
	if err.Cause != cause {
		t.Errorf("NewValidationError() Cause = %v, expected %v", err.Cause, cause)
	}
}

func TestNewNetworkError(t *testing.T) {
	cause := errors.New("connection refused")
	err := NewNetworkError("Network connection failed", cause)

	if err.Type != ErrorTypeNetwork {
		t.Errorf("NewNetworkError() Type = %v, expected %v", err.Type, ErrorTypeNetwork)
	}
	if err.Message != "Network connection failed" {
		t.Errorf("NewNetworkError() Message = %q, expected %q", err.Message, "Network connection failed")
	}
	if err.Cause != cause {
		t.Errorf("NewNetworkError() Cause = %v, expected %v", err.Cause, cause)
	}
}

func TestNewAuthenticationError(t *testing.T) {
	cause := errors.New("invalid token")
	err := NewAuthenticationError("Authentication failed", cause)

	if err.Type != ErrorTypeAuthentication {
		t.Errorf("NewAuthenticationError() Type = %v, expected %v", err.Type, ErrorTypeAuthentication)
	}
	if err.Message != "Authentication failed" {
		t.Errorf("NewAuthenticationError() Message = %q, expected %q", err.Message, "Authentication failed")
	}
	if err.Cause != cause {
		t.Errorf("NewAuthenticationError() Cause = %v, expected %v", err.Cause, cause)
	}
}

func TestNewInternalError(t *testing.T) {
	cause := errors.New("database connection lost")
	err := NewInternalError("Internal system error", cause)

	if err.Type != ErrorTypeInternal {
		t.Errorf("NewInternalError() Type = %v, expected %v", err.Type, ErrorTypeInternal)
	}
	if err.Message != "Internal system error" {
		t.Errorf("NewInternalError() Message = %q, expected %q", err.Message, "Internal system error")
	}
	if err.Cause != cause {
		t.Errorf("NewInternalError() Cause = %v, expected %v", err.Cause, cause)
	}
}

func TestNewConfigurationError(t *testing.T) {
	cause := errors.New("missing required config")
	err := NewConfigurationError("Configuration error", cause)

	if err.Type != ErrorTypeConfiguration {
		t.Errorf("NewConfigurationError() Type = %v, expected %v", err.Type, ErrorTypeConfiguration)
	}
	if err.Message != "Configuration error" {
		t.Errorf("NewConfigurationError() Message = %q, expected %q", err.Message, "Configuration error")
	}
	if err.Cause != cause {
		t.Errorf("NewConfigurationError() Cause = %v, expected %v", err.Cause, cause)
	}
}

func TestIsErrorType(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		errorType ErrorType
		expected  bool
	}{
		{
			name: "direct FrameworkError match",
			err: &FrameworkError{
				Type: ErrorTypeValidation,
			},
			errorType: ErrorTypeValidation,
			expected:  true,
		},
		{
			name: "direct FrameworkError no match",
			err: &FrameworkError{
				Type: ErrorTypeNetwork,
			},
			errorType: ErrorTypeValidation,
			expected:  false,
		},
		{
			name:      "non-FrameworkError",
			err:       errors.New("regular error"),
			errorType: ErrorTypeValidation,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsErrorType(tt.err, tt.errorType)
			if result != tt.expected {
				t.Errorf("IsErrorType() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAsFrameworkError(t *testing.T) {
	fwErr := &FrameworkError{
		Type:    ErrorTypeInternal,
		Message: "test error",
	}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct FrameworkError",
			err:      fwErr,
			expected: true,
		},
		{
			name:     "wrapped FrameworkError",
			err:      WrapError(fwErr, "wrapped"),
			expected: true,
		},
		{
			name:     "non-FrameworkError",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target *FrameworkError
			result := AsFrameworkError(tt.err, &target)
			if result != tt.expected {
				t.Errorf("AsFrameworkError() = %v, expected %v", result, tt.expected)
			}
			if result && target == nil {
				t.Error("AsFrameworkError() returned true but target is nil")
			}
		})
	}
}

func TestUnwrapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{
			name: "error with Unwrap method",
			err: &FrameworkError{
				Cause: errors.New("underlying"),
			},
			expected: errors.New("underlying"),
		},
		{
			name:     "error without Unwrap method",
			err:      errors.New("no unwrap"),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnwrapError(tt.err)
			if result == nil && tt.expected != nil {
				t.Errorf("UnwrapError() = nil, expected %v", tt.expected)
			}
			if result != nil && tt.expected == nil {
				t.Errorf("UnwrapError() = %v, expected nil", result)
			}
			if result != nil && tt.expected != nil && result.Error() != tt.expected.Error() {
				t.Errorf("UnwrapError() = %q, expected %q", result.Error(), tt.expected.Error())
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		message  string
		expected bool
	}{
		{
			name:     "wrap regular error",
			err:      errors.New("underlying error"),
			message:  "wrapped message",
			expected: true,
		},
		{
			name:     "wrap nil error",
			err:      nil,
			message:  "should not wrap",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.message)
			if tt.expected && result == nil {
				t.Error("WrapError() returned nil, expected FrameworkError")
			}
			if !tt.expected && result != nil {
				t.Error("WrapError() returned error, expected nil")
			}
			if tt.expected {
				fwErr, ok := result.(*FrameworkError)
				if !ok {
					t.Error("WrapError() did not return FrameworkError")
				}
				if fwErr.Message != tt.message {
					t.Errorf("WrapError() message = %q, expected %q", fwErr.Message, tt.message)
				}
				if fwErr.Type != ErrorTypeInternal {
					t.Errorf("WrapError() type = %v, expected %v", fwErr.Type, ErrorTypeInternal)
				}
			}
		})
	}
}

func TestNewFrameworkErrorWithCode(t *testing.T) {
	err := NewFrameworkErrorWithCode(ErrorTypeRateLimit, ErrorCodeRateLimited, "Rate limit exceeded", nil)

	if err.Type != ErrorTypeRateLimit {
		t.Errorf("NewFrameworkErrorWithCode() Type = %v, expected %v", err.Type, ErrorTypeRateLimit)
	}
	if err.Code != string(ErrorCodeRateLimited) {
		t.Errorf("NewFrameworkErrorWithCode() Code = %q, expected %q", err.Code, ErrorCodeRateLimited)
	}
	if err.Message != "Rate limit exceeded" {
		t.Errorf("NewFrameworkErrorWithCode() Message = %q, expected %q", err.Message, "Rate limit exceeded")
	}
	if err.Context == nil {
		t.Error("NewFrameworkErrorWithCode() Context should not be nil")
	}
}

func TestFrameworkError_AddContext(t *testing.T) {
	err := &FrameworkError{
		Type: ErrorTypeInternal,
	}

	result := err.AddContext("key1", "value1").AddContext("key2", 42)

	if result.Context["key1"] != "value1" {
		t.Errorf("AddContext() key1 = %v, expected %q", result.Context["key1"], "value1")
	}
	if result.Context["key2"] != 42 {
		t.Errorf("AddContext() key2 = %v, expected %v", result.Context["key2"], 42)
	}
}

func TestErrorConstants(t *testing.T) {
	// Test that error types are properly defined
	if ErrorTypeValidation != "validation" {
		t.Errorf("ErrorTypeValidation = %q, expected %q", ErrorTypeValidation, "validation")
	}
	if ErrorTypeNetwork != "network" {
		t.Errorf("ErrorTypeNetwork = %q, expected %q", ErrorTypeNetwork, "network")
	}
	if ErrorTypeAuthentication != "authentication" {
		t.Errorf("ErrorTypeAuthentication = %q, expected %q", ErrorTypeAuthentication, "authentication")
	}
	if ErrorTypeRateLimit != "rate_limit" {
		t.Errorf("ErrorTypeRateLimit = %q, expected %q", ErrorTypeRateLimit, "rate_limit")
	}
	if ErrorTypeInternal != "internal" {
		t.Errorf("ErrorTypeInternal = %q, expected %q", ErrorTypeInternal, "internal")
	}
	if ErrorTypeExternal != "external" {
		t.Errorf("ErrorTypeExternal = %q, expected %q", ErrorTypeExternal, "external")
	}
	if ErrorTypeConfiguration != "configuration" {
		t.Errorf("ErrorTypeConfiguration = %q, expected %q", ErrorTypeConfiguration, "configuration")
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that error codes are properly defined
	if ErrorCodeInvalidInput != "INVALID_INPUT" {
		t.Errorf("ErrorCodeInvalidInput = %q, expected %q", ErrorCodeInvalidInput, "INVALID_INPUT")
	}
	if ErrorCodeNotFound != "NOT_FOUND" {
		t.Errorf("ErrorCodeNotFound = %q, expected %q", ErrorCodeNotFound, "NOT_FOUND")
	}
	if ErrorCodeUnauthorized != "UNAUTHORIZED" {
		t.Errorf("ErrorCodeUnauthorized = %q, expected %q", ErrorCodeUnauthorized, "UNAUTHORIZED")
	}
	if ErrorCodeTimeout != "TIMEOUT" {
		t.Errorf("ErrorCodeTimeout = %q, expected %q", ErrorCodeTimeout, "TIMEOUT")
	}
	if ErrorCodeRateLimited != "RATE_LIMITED" {
		t.Errorf("ErrorCodeRateLimited = %q, expected %q", ErrorCodeRateLimited, "RATE_LIMITED")
	}
	if ErrorCodeInternalError != "INTERNAL_ERROR" {
		t.Errorf("ErrorCodeInternalError = %q, expected %q", ErrorCodeInternalError, "INTERNAL_ERROR")
	}
}
