package iface

import (
	"errors"
	"fmt"
	"testing"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewConfigError(t *testing.T) {
	err := NewConfigError("test_code", "test message %s", "arg")

	if err.Code != "test_code" {
		t.Errorf("expected code 'test_code', got %s", err.Code)
	}

	expectedMsg := "test message arg"
	if err.Message != expectedMsg {
		t.Errorf("expected message '%s', got '%s'", expectedMsg, err.Message)
	}

	if err.Cause != nil {
		t.Error("expected no cause for new error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestWrapError(t *testing.T) {
	cause := errors.New("original error")
	err := WrapError(cause, "test_code", "wrapped message %s", "arg")

	if err.Code != "test_code" {
		t.Errorf("expected code 'test_code', got %s", err.Code)
	}

	expectedMsg := "wrapped message arg"
	if err.Message != expectedMsg {
		t.Errorf("expected message '%s', got '%s'", expectedMsg, err.Message)
	}

	if err.Cause != cause {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Error("expected cause to be preserved")
	}
}

func TestConfigError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigError
		expected string
	}{
		{
			name:     "no cause",
			err:      NewConfigError("code", "message"),
			expected: "message",
		},
		{
			name:     "with cause",
			err:      WrapError(errors.New("cause"), "code", "message"),
			expected: "message: cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConfigError_Unwrap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	cause := errors.New("original error")
	wrapped := WrapError(cause, "code", "message")

	if wrapped.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestIsConfigError(t *testing.T) {
	configErr := NewConfigError("test_code", "message")
	regularErr := errors.New("regular error")

	tests := []struct {
		name     string
		err      error
		code     string
		expected bool
	}{
		{"config error with matching code", configErr, "test_code", true},
		{"config error with non-matching code", configErr, "other_code", false},
		{"regular error", regularErr, "test_code", false},
		{"nil error", nil, "test_code", false},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConfigError(tt.err, tt.code)
			if result != tt.expected {
				t.Errorf("IsConfigError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAsConfigError(t *testing.T) {
	configErr := NewConfigError("test_code", "message")
	regularErr := errors.New("regular error")

	tests := []struct {
		name        string
		err         error
		expectFound bool
	}{
		{"config error", configErr, true},
		{"wrapped config error", WrapError(configErr, "outer", "outer message"), true},
		{"regular error", regularErr, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target *ConfigError
			found := AsConfigError(tt.err, &target)

			if found != tt.expectFound {
				t.Errorf("AsConfigError() found = %v, want %v", found, tt.expectFound)
			}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

			if tt.expectFound && target == nil {
				t.Error("expected target to be set when found is true")
			}

			if !tt.expectFound && target != nil {
				t.Error("expected target to be nil when found is false")
			}
		})
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that all error codes are defined and non-empty
	codes := []string{
		ErrCodeInvalidConfig,
		ErrCodeValidationFailed,
		ErrCodeFileNotFound,
		ErrCodeParseFailed,
		ErrCodeUnsupportedFormat,
		ErrCodeMissingRequired,
		ErrCodeInvalidProvider,
		ErrCodeInvalidParameters,
		ErrCodeLoadFailed,
		ErrCodeSaveFailed,
		ErrCodeProviderUnavailable,
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		ErrCodeRemoteLoadTimeout,
		ErrCodeAllProvidersFailed,
		ErrCodeConfigNotFound,
		ErrCodeKeyNotFound,
		ErrCodeInvalidFormat,
	}

	for _, code := range codes {
		if code == "" {
			t.Error("error code should not be empty")
		}
	}
}

func TestIsConfigError_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     string
		expected bool
	}{
		{"nil error", nil, "test_code", false},
		{"regular error", errors.New("regular"), "test_code", false},
		{"config error with matching code", NewConfigError("test_code", "msg"), "test_code", true},
		{"config error with non-matching code", NewConfigError("other_code", "msg"), "test_code", false},
		{"wrapped config error", WrapError(NewConfigError("test_code", "msg"), "wrapper", "wrapper msg"), "test_code", true},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		{"deeply wrapped config error", func() error {
			return WrapError(WrapError(NewConfigError("test_code", "msg"), "middle", "middle"), "outer", "outer")
		}(), "test_code", true},
		{"empty code", NewConfigError("", "msg"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConfigError(tt.err, tt.code)
			if result != tt.expected {
				t.Errorf("IsConfigError(%v, %s) = %v, want %v", tt.err, tt.code, result, tt.expected)
			}
		})
	}
}

func TestAsConfigError_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectFound bool
	}{
		{"nil error", nil, false},
		{"regular error", errors.New("regular"), false},
		{"config error", NewConfigError("code", "msg"), true},
		{"wrapped config error", WrapError(NewConfigError("code", "msg"), "wrapper", "msg"), true},
		{"deeply wrapped config error", func() error {
			return WrapError(WrapError(NewConfigError("code", "msg"), "middle", "msg"), "outer", "msg")
		}(), true},
		{"config error wrapped in regular error", func() error {
			// Actually wrap a ConfigError in a regular error using fmt.Errorf
			configErr := NewConfigError("inner", "inner msg")
			return fmt.Errorf("regular error: %w", configErr)
		}(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			var target *ConfigError
			found := AsConfigError(tt.err, &target)

			if found != tt.expectFound {
				t.Errorf("AsConfigError() found = %v, want %v", found, tt.expectFound)
			}

			if tt.expectFound && target == nil {
				t.Error("expected target to be set when found is true")
			}

			if !tt.expectFound && target != nil {
				t.Error("expected target to be nil when found is false")
			}
		})
	}
}

func TestNewConfigError_Formatting(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		message  string
		args     []interface{}
		expected string
	}{
		{
			name:     "no args",
			code:     "test_code",
			message:  "simple message",
			args:     nil,
			expected: "simple message",
		},
		{
			name:     "with args",
			code:     "test_code",
			message:  "message with %s and %d",
			args:     []interface{}{"string", 42},
			expected: "message with string and 42",
		},
		{
			name:     "empty message",
			code:     "test_code",
			message:  "",
			args:     nil,
			expected: "",
		},
		{
			name:     "message with %",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			code:     "test_code",
			message:  "message with %% percent",
			args:     nil,
			expected: "message with % percent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewConfigError(tt.code, tt.message, tt.args...)
			if err.Message != tt.expected {
				t.Errorf("NewConfigError() message = %q, want %q", err.Message, tt.expected)
			}
			if err.Code != tt.code {
				t.Errorf("NewConfigError() code = %q, want %q", err.Code, tt.code)
			}
		})
	}
}

func TestWrapError_Formatting(t *testing.T) {
	cause := errors.New("original cause")

	tests := []struct {
		name     string
		cause    error
		code     string
		message  string
		args     []interface{}
		expected string
	}{
		{
			name:     "wrap with no args",
			cause:    cause,
			code:     "test_code",
			message:  "wrapped message",
			args:     nil,
			expected: "wrapped message: original cause",
		},
		{
			name:     "wrap with args",
			cause:    cause,
			code:     "test_code",
			message:  "wrapped %s with %d",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			args:     []interface{}{"message", 42},
			expected: "wrapped message with 42: original cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapError(tt.cause, tt.code, tt.message, tt.args...)
			// Test the Error() method which combines message and cause
			if err.Error() != tt.expected {
				t.Errorf("WrapError() Error() = %q, want %q", err.Error(), tt.expected)
			}
			if err.Code != tt.code {
				t.Errorf("WrapError() code = %q, want %q", err.Code, tt.code)
			}
			if err.Cause != tt.cause {
				t.Errorf("WrapError() cause = %v, want %v", err.Cause, tt.cause)
			}
		})
	}
}

func TestConfigError_Error_Formatting(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigError
		expected string
	}{
		{
			name:     "no cause",
			err:      NewConfigError("code", "message"),
			expected: "message",
		},
		{
			name:     "with cause",
			err:      WrapError(errors.New("cause"), "code", "message"),
			expected: "message: cause",
		},
		{
			name:     "nil cause",
			err:      &ConfigError{Code: "code", Message: "message", Cause: nil},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			expected: "message",
		},
		{
			name:     "empty message",
			err:      NewConfigError("code", ""),
			expected: "",
		},
		{
			name:     "empty message with cause",
			err:      WrapError(errors.New("cause"), "code", ""),
			expected: ": cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("ConfigError.Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestErrorHandling_Integration(t *testing.T) {
	// Test error wrapping and unwrapping chain
	originalErr := errors.New("original error")

	// Wrap multiple times
	wrapped1 := WrapError(originalErr, "level1", "level 1 message")
	wrapped2 := WrapError(wrapped1, "level2", "level 2 message: %s", "detail")

	// Test IsConfigError at different levels
	if !IsConfigError(wrapped2, "level2") {
		t.Error("should find level2 error code")
	}
	if !IsConfigError(wrapped2, "level1") {
		t.Error("should find level1 error code in wrapped error")
	}
	if IsConfigError(wrapped2, "nonexistent") {
		t.Error("should not find nonexistent error code")
	}

	// Test AsConfigError
	var cfgErr *ConfigError
	if !AsConfigError(wrapped2, &cfgErr) {
		t.Error("should be able to cast to ConfigError")
	}
	if cfgErr.Code != "level2" {
		t.Errorf("expected code 'level2', got %s", cfgErr.Code)
	}

	// Test unwrapping chain
	unwrapped := cfgErr.Unwrap()
	if unwrapped == nil {
		t.Fatal("expected unwrapped error")
	}

	var cfgErr2 *ConfigError
	if !AsConfigError(unwrapped, &cfgErr2) {
		t.Error("unwrapped error should also be ConfigError")
	}
	if cfgErr2.Code != "level1" {
		t.Errorf("expected unwrapped code 'level1', got %s", cfgErr2.Code)
	}
}
