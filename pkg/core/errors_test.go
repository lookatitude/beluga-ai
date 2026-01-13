package core

import (
	"errors"
	"testing"
)

// Static errors for error tests.
var (
	errTimeout                = errors.New("timeout")
	errUnderlyingError        = errors.New("underlying error")
	errFieldRequired          = errors.New("field is required")
	errConnectionRefused      = errors.New("connection refused")
	errInvalidToken           = errors.New("invalid token")
	errDatabaseConnectionLost = errors.New("database connection lost")
	errMissingRequiredConfig  = errors.New("missing required config")
	errRegularError           = errors.New("regular error")
	errUnderlying             = errors.New("underlying")
	errNoUnwrap               = errors.New("no unwrap")
	errRegular                = errors.New("regular")
	errErrorTestCause         = errors.New("test cause")
	errCause                  = errors.New("cause")
)

func TestFrameworkError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *FrameworkError
		expected string
	}{
		{
			name: "error without err",
			err: &FrameworkError{
				Op:      "validate",
				Message: "Invalid input",
				Code:    string(ErrorCodeInvalidInput),
			},
			expected: "core validate: Invalid input (code: invalid_input)",
		},
		{
			name: "error with err",
			err: &FrameworkError{
				Op:      "connect",
				Message: "Connection failed",
				Err:     errTimeout,
				Code:    string(ErrorCodeTimeout),
			},
			expected: "core connect: Connection failed (code: timeout)",
		},
		{
			name: "error with code only",
			err: &FrameworkError{
				Op:      "process",
				Message: "System error",
				Code:    string(ErrorCodeInternalError),
			},
			expected: "core process: System error (code: internal_error)",
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
	cause := errUnderlyingError
	err := &FrameworkError{
		Op:      "process",
		Message: "Wrapped error",
		Err:     cause,
		Code:    string(ErrorCodeInternalError),
	}

	if !errors.Is(err.Unwrap(), cause) {
		t.Errorf("FrameworkError.Unwrap() = %v, expected %v", err.Unwrap(), cause)
	}
}

func TestNewValidationError(t *testing.T) {
	cause := errFieldRequired
	err := NewValidationError("validate", "Input validation failed", cause)

	if err.Op != "validate" {
		t.Errorf("NewValidationError() Op = %q, expected %q", err.Op, "validate")
	}
	if err.Message != "Input validation failed" {
		t.Errorf("NewValidationError() Message = %q, expected %q", err.Message, "Input validation failed")
	}
	if err.Code != string(ErrorCodeInvalidInput) {
		t.Errorf("NewValidationError() Code = %q, expected %q", err.Code, ErrorCodeInvalidInput)
	}
	if !errors.Is(err.Err, cause) {
		t.Errorf("NewValidationError() Err = %v, expected %v", err.Err, cause)
	}
}

func TestNewNetworkError(t *testing.T) {
	cause := errConnectionRefused
	err := NewNetworkError("connect", "Network connection failed", cause)

	if err.Op != "connect" {
		t.Errorf("NewNetworkError() Op = %q, expected %q", err.Op, "connect")
	}
	if err.Message != "Network connection failed" {
		t.Errorf("NewNetworkError() Message = %q, expected %q", err.Message, "Network connection failed")
	}
	if err.Code != string(ErrorCodeTimeout) {
		t.Errorf("NewNetworkError() Code = %q, expected %q", err.Code, ErrorCodeTimeout)
	}
	if !errors.Is(err.Err, cause) {
		t.Errorf("NewNetworkError() Err = %v, expected %v", err.Err, cause)
	}
}

func TestNewAuthenticationError(t *testing.T) {
	cause := errInvalidToken
	err := NewAuthenticationError("authenticate", "Authentication failed", cause)

	if err.Op != "authenticate" {
		t.Errorf("NewAuthenticationError() Op = %q, expected %q", err.Op, "authenticate")
	}
	if err.Message != "Authentication failed" {
		t.Errorf("NewAuthenticationError() Message = %q, expected %q", err.Message, "Authentication failed")
	}
	if err.Code != string(ErrorCodeUnauthorized) {
		t.Errorf("NewAuthenticationError() Code = %q, expected %q", err.Code, ErrorCodeUnauthorized)
	}
	if !errors.Is(err.Err, cause) {
		t.Errorf("NewAuthenticationError() Err = %v, expected %v", err.Err, cause)
	}
}

func TestNewInternalError(t *testing.T) {
	cause := errDatabaseConnectionLost
	err := NewInternalError("process", "Internal system error", cause)

	if err.Op != "process" {
		t.Errorf("NewInternalError() Op = %q, expected %q", err.Op, "process")
	}
	if err.Message != "Internal system error" {
		t.Errorf("NewInternalError() Message = %q, expected %q", err.Message, "Internal system error")
	}
	if err.Code != string(ErrorCodeInternalError) {
		t.Errorf("NewInternalError() Code = %q, expected %q", err.Code, ErrorCodeInternalError)
	}
	if !errors.Is(err.Err, cause) {
		t.Errorf("NewInternalError() Err = %v, expected %v", err.Err, cause)
	}
}

func TestNewConfigurationError(t *testing.T) {
	cause := errMissingRequiredConfig
	err := NewConfigurationError("load_config", "Configuration error", cause)

	if err.Op != "load_config" {
		t.Errorf("NewConfigurationError() Op = %q, expected %q", err.Op, "load_config")
	}
	if err.Message != "Configuration error" {
		t.Errorf("NewConfigurationError() Message = %q, expected %q", err.Message, "Configuration error")
	}
	if err.Code != string(ErrorCodeInvalidInput) {
		t.Errorf("NewConfigurationError() Code = %q, expected %q", err.Code, ErrorCodeInvalidInput)
	}
	if !errors.Is(err.Err, cause) {
		t.Errorf("NewConfigurationError() Err = %v, expected %v", err.Err, cause)
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
				Op:   "validate",
				Code: string(ErrorCodeInvalidInput),
			},
			errorType: ErrorTypeValidation,
			expected:  true,
		},
		{
			name: "direct FrameworkError no match",
			err: &FrameworkError{
				Op:   "connect",
				Code: string(ErrorCodeTimeout),
			},
			errorType: ErrorTypeValidation,
			expected:  false,
		},
		{
			name:      "non-FrameworkError",
			err:       errRegularError,
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
		Op:      "process",
		Message: "test error",
		Code:    string(ErrorCodeInternalError),
	}

	tests := []struct {
		err      error
		name     string
		expected bool
	}{
		{
			name:     "direct FrameworkError",
			err:      fwErr,
			expected: true,
		},
		{
			name:     "wrapped FrameworkError",
			err:      WrapError(fwErr, "wrap", "wrapped"),
			expected: true,
		},
		{
			name:     "non-FrameworkError",
			err:      errRegularError,
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
		err      error
		expected error
		name     string
	}{
		{
			name: "error with Unwrap method",
			err: &FrameworkError{
				Op:  "process",
				Err: errUnderlying,
			},
			expected: errUnderlying,
		},
		{
			name:     "error without Unwrap method",
			err:      errNoUnwrap,
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
		op       string
		message  string
		expected bool
	}{
		{
			name:     "wrap regular error",
			err:      errUnderlyingError,
			op:       "wrap",
			message:  "wrapped message",
			expected: true,
		},
		{
			name:     "wrap nil error",
			err:      nil,
			op:       "wrap",
			message:  "should not wrap",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.op, tt.message)
			if tt.expected && result == nil {
				t.Error("WrapError() returned nil, expected FrameworkError")
			}
			if !tt.expected && result != nil {
				t.Error("WrapError() returned error, expected nil")
			}
			if tt.expected {
				fwErr := &FrameworkError{}
				ok := errors.As(result, &fwErr)
				if !ok {
					t.Error("WrapError() did not return FrameworkError")
				}
				if fwErr.Message != tt.message {
					t.Errorf("WrapError() message = %q, expected %q", fwErr.Message, tt.message)
				}
				if fwErr.Op != tt.op {
					t.Errorf("WrapError() op = %q, expected %q", fwErr.Op, tt.op)
				}
				if fwErr.Code != string(ErrorCodeInternalError) {
					t.Errorf("WrapError() code = %q, expected %q", fwErr.Code, ErrorCodeInternalError)
				}
			}
		})
	}
}

func TestNewFrameworkErrorWithCode(t *testing.T) {
	err := NewFrameworkError("rate_limit", ErrorCodeRateLimited, "Rate limit exceeded", nil)

	if err.Op != "rate_limit" {
		t.Errorf("NewFrameworkError() Op = %q, expected %q", err.Op, "rate_limit")
	}
	if err.Code != string(ErrorCodeRateLimited) {
		t.Errorf("NewFrameworkError() Code = %q, expected %q", err.Code, ErrorCodeRateLimited)
	}
	if err.Message != "Rate limit exceeded" {
		t.Errorf("NewFrameworkError() Message = %q, expected %q", err.Message, "Rate limit exceeded")
	}
	if err.Context == nil {
		t.Error("NewFrameworkError() Context should not be nil")
	}
}

func TestFrameworkError_AddContext(t *testing.T) {
	err := &FrameworkError{
		Op:   "process",
		Code: string(ErrorCodeInternalError),
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
	// Test that error codes are properly defined (now using snake_case)
	if ErrorCodeInvalidInput != "invalid_input" {
		t.Errorf("ErrorCodeInvalidInput = %q, expected %q", ErrorCodeInvalidInput, "invalid_input")
	}
	if ErrorCodeNotFound != "not_found" {
		t.Errorf("ErrorCodeNotFound = %q, expected %q", ErrorCodeNotFound, "not_found")
	}
	if ErrorCodeUnauthorized != "unauthorized" {
		t.Errorf("ErrorCodeUnauthorized = %q, expected %q", ErrorCodeUnauthorized, "unauthorized")
	}
	if ErrorCodeTimeout != "timeout" {
		t.Errorf("ErrorCodeTimeout = %q, expected %q", ErrorCodeTimeout, "timeout")
	}
	if ErrorCodeRateLimited != "rate_limited" {
		t.Errorf("ErrorCodeRateLimited = %q, expected %q", ErrorCodeRateLimited, "rate_limited")
	}
	if ErrorCodeInternalError != "internal_error" {
		t.Errorf("ErrorCodeInternalError = %q, expected %q", ErrorCodeInternalError, "internal_error")
	}
}

// Edge case and error scenario tests

func TestFrameworkError_NilErr(t *testing.T) {
	err := &FrameworkError{
		Op:      "validate",
		Message: "test message",
		Code:    string(ErrorCodeInvalidInput),
		Err:     nil,
	}

	errorMsg := err.Error()
	expected := "core validate: test message (code: invalid_input)"
	if errorMsg != expected {
		t.Errorf("FrameworkError.Error() = %q, expected %q", errorMsg, expected)
	}

	if err.Unwrap() != nil {
		t.Error("FrameworkError.Unwrap() should return nil for nil err")
	}
}

func TestFrameworkError_EmptyFields(t *testing.T) {
	err := &FrameworkError{}

	errorMsg := err.Error()
	expected := "core : unknown error (code: )"
	if errorMsg != expected {
		t.Errorf("FrameworkError.Error() = %q, expected %q", errorMsg, expected)
	}
}

func TestNewFrameworkError_EdgeCases(t *testing.T) {
	// Test with empty code
	err := NewFrameworkError("process", "", "message", nil)
	if err.Code != "" {
		t.Errorf("Expected empty code, got %q", err.Code)
	}

	// Test with nil err
	err = NewFrameworkError("connect", ErrorCodeTimeout, "message", nil)
	if err.Err != nil {
		t.Errorf("Expected nil err, got %v", err.Err)
	}

	// Test with context
	err = NewFrameworkError("validate", ErrorCodeInvalidInput, "message", errCause)
	if err.Context == nil {
		t.Error("Context should be initialized")
	}
}

func TestIsErrorType_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		errorType ErrorType
		expected  bool
	}{
		{
			name:      "nil error",
			err:       nil,
			errorType: ErrorTypeValidation,
			expected:  false,
		},
		{
			name:      "non-FrameworkError",
			err:       errRegularError,
			errorType: ErrorTypeValidation,
			expected:  false,
		},
		{
			name: "FrameworkError with empty Code",
			err: &FrameworkError{
				Op:      "validate",
				Code:    "",
				Message: "test",
			},
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

func TestAsFrameworkError_EdgeCases(t *testing.T) {
	tests := []struct {
		err      error
		name     string
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errRegularError,
			expected: false,
		},
		{
			name: "FrameworkError",
			err: &FrameworkError{
				Op:      "process",
				Code:    string(ErrorCodeInternalError),
				Message: "test",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target *FrameworkError
			result := AsFrameworkError(tt.err, &target)
			if result != tt.expected {
				t.Errorf("AsFrameworkError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestUnwrapError_EdgeCases(t *testing.T) {
	tests := []struct {
		err      error
		expected error
		name     string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: nil,
		},
		{
			name:     "error without Unwrap method",
			err:      errNoUnwrap,
			expected: nil,
		},
		{
			name: "FrameworkError with nil err",
			err: &FrameworkError{
				Op:      "process",
				Message: "test",
				Code:    string(ErrorCodeInternalError),
				Err:     nil,
			},
			expected: nil,
		},
		{
			name:     "FrameworkError with err",
			err:      &FrameworkError{Op: "process", Err: errUnderlying},
			expected: errUnderlying,
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

func TestWrapError_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		op       string
		message  string
		expected bool // true if FrameworkError expected
	}{
		{
			name:     "nil error",
			err:      nil,
			op:       "wrap",
			message:  "should not wrap",
			expected: false,
		},
		{
			name:     "regular error",
			err:      errRegular,
			op:       "wrap",
			message:  "wrapped",
			expected: true,
		},
		{
			name:     "empty message",
			err:      errRegular,
			op:       "wrap",
			message:  "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.op, tt.message)
			if tt.expected && result == nil {
				t.Error("WrapError() returned nil, expected FrameworkError")
			}
			if !tt.expected && result != nil {
				t.Error("WrapError() returned error, expected nil")
			}
			if tt.expected {
				fwErr := &FrameworkError{}
				ok := errors.As(result, &fwErr)
				if !ok {
					t.Error("WrapError() did not return FrameworkError")
				} else {
					if fwErr.Message != tt.message {
						t.Errorf("WrapError() message = %q, expected %q", fwErr.Message, tt.message)
					}
					if fwErr.Op != tt.op {
						t.Errorf("WrapError() op = %q, expected %q", fwErr.Op, tt.op)
					}
					if fwErr.Code != string(ErrorCodeInternalError) {
						t.Errorf("WrapError() code = %q, expected %q", fwErr.Code, ErrorCodeInternalError)
					}
					if !errors.Is(fwErr.Err, tt.err) {
						t.Errorf("WrapError() err = %v, expected %v", fwErr.Err, tt.err)
					}
				}
			}
		})
	}
}

func TestFrameworkError_AddContext_EdgeCases(t *testing.T) {
	err := &FrameworkError{
		Op:      "process",
		Code:    string(ErrorCodeInternalError),
		Message: "test",
	}

	// Test adding to nil context
	err = err.AddContext("key1", "value1")
	if err.Context == nil {
		t.Error("Context should be initialized after AddContext")
	}

	// Test overwriting existing key
	err = err.AddContext("key1", "new_value")
	if err.Context["key1"] != "new_value" {
		t.Errorf("AddContext() should overwrite existing key, got %v", err.Context["key1"])
	}

	// Test adding nil value
	err = err.AddContext("nil_key", nil)
	if err.Context["nil_key"] != nil {
		t.Errorf("AddContext() should handle nil values, got %v", err.Context["nil_key"])
	}
}

func TestErrorConstructors_EdgeCases(t *testing.T) {
	cause := errErrorTestCause

	// Test with nil err
	err1 := NewValidationError("validate", "message", nil)
	if err1.Err != nil {
		t.Error("NewValidationError() with nil err should have nil Err")
	}

	// Test with empty message
	err2 := NewValidationError("validate", "", cause)
	if err2.Message != "" {
		t.Error("NewValidationError() should preserve empty message")
	}

	// Test all constructor functions
	constructors := []func(string, string, error) *FrameworkError{
		NewValidationError,
		NewNetworkError,
		NewAuthenticationError,
		NewInternalError,
		NewConfigurationError,
	}

	expectedOps := []string{
		"validate",
		"connect",
		"authenticate",
		"process",
		"load_config",
	}

	expectedCodes := []ErrorCode{
		ErrorCodeInvalidInput,
		ErrorCodeTimeout,
		ErrorCodeUnauthorized,
		ErrorCodeInternalError,
		ErrorCodeInvalidInput,
	}

	for i, constructor := range constructors {
		err := constructor(expectedOps[i], "test message", cause)
		if err.Op != expectedOps[i] {
			t.Errorf("Constructor %d: Op = %q, expected %q", i, err.Op, expectedOps[i])
		}
		if err.Message != "test message" {
			t.Errorf("Constructor %d: Message = %q, expected %q", i, err.Message, "test message")
		}
		if err.Code != string(expectedCodes[i]) {
			t.Errorf("Constructor %d: Code = %q, expected %q", i, err.Code, expectedCodes[i])
		}
		if !errors.Is(err.Err, cause) {
			t.Errorf("Constructor %d: Err = %v, expected %v", i, err.Err, cause)
		}
	}
}

func TestErrorTypeStringValues(t *testing.T) {
	// Test that error type constants have expected string values
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrorTypeValidation, "validation"},
		{ErrorTypeNetwork, "network"},
		{ErrorTypeAuthentication, "authentication"},
		{ErrorTypeRateLimit, "rate_limit"},
		{ErrorTypeInternal, "internal"},
		{ErrorTypeExternal, "external"},
		{ErrorTypeConfiguration, "configuration"},
	}

	for _, tt := range tests {
		if string(tt.errorType) != tt.expected {
			t.Errorf("ErrorType %v = %q, expected %q", tt.errorType, string(tt.errorType), tt.expected)
		}
	}
}

func TestErrorCodeStringValues(t *testing.T) {
	// Test that error code constants have expected string values (now snake_case)
	tests := []struct {
		errorCode ErrorCode
		expected  string
	}{
		{ErrorCodeInvalidInput, "invalid_input"},
		{ErrorCodeNotFound, "not_found"},
		{ErrorCodeUnauthorized, "unauthorized"},
		{ErrorCodeTimeout, "timeout"},
		{ErrorCodeRateLimited, "rate_limited"},
		{ErrorCodeInternalError, "internal_error"},
	}

	for _, tt := range tests {
		if string(tt.errorCode) != tt.expected {
			t.Errorf("ErrorCode %v = %q, expected %q", tt.errorCode, string(tt.errorCode), tt.expected)
		}
	}
}
