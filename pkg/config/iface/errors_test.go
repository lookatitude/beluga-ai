package iface

import (
	"errors"
	"testing"
)

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
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConfigError_Unwrap(t *testing.T) {
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
