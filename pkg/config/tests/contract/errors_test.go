package contract

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/stretchr/testify/assert"
)

// TestConfigError_Contract tests the ConfigError interface contract.
// This ensures the error implementation meets all contractual requirements.
func TestConfigError_Contract(t *testing.T) {
	// Test NewError constructor
	t.Run("NewError", func(t *testing.T) {
		op := "test.operation"
		code := config.ErrCodeLoadFailed
		provider := "test-provider"
		underlyingErr := errors.New("underlying error")

		err := config.NewError(op, code, provider, underlyingErr)
		assert.Error(t, err, "NewError should return an error")
		assert.Contains(t, err.Error(), op, "Error should contain operation")
		assert.Contains(t, err.Error(), code, "Error should contain error code")

		// Test ConfigError interface methods
		configErr, ok := config.AsConfigError(err)
		assert.True(t, ok, "Should be convertible to ConfigError")

		assert.Equal(t, op, configErr.GetOperation(), "GetOperation should return correct operation")
		assert.Equal(t, code, configErr.GetCode(), "GetCode should return correct code")
		assert.Equal(t, provider, configErr.GetProvider(), "GetProvider should return correct provider")
		assert.NotNil(t, configErr.GetContext(), "GetContext should return non-nil context")
		assert.False(t, configErr.GetTimestamp().IsZero(), "GetTimestamp should return non-zero time")

		// Test error wrapping
		unwrapped := configErr.Unwrap()
		assert.Equal(t, underlyingErr, unwrapped, "Unwrap should return underlying error")
	})

	// Test NewLoadError constructor
	t.Run("NewLoadError", func(t *testing.T) {
		op := "load.config"
		code := config.ErrCodeFileNotFound
		provider := "viper"
		format := "yaml"
		path := "/path/to/config.yaml"
		underlyingErr := errors.New("file not found")

		err := config.NewLoadError(op, code, provider, format, path, underlyingErr)
		assert.Error(t, err, "NewLoadError should return an error")

		configErr, ok := config.AsConfigError(err)
		assert.True(t, ok, "Should be convertible to ConfigError")

		assert.Equal(t, op, configErr.GetOperation(), "GetOperation should return correct operation")
		assert.Equal(t, code, configErr.GetCode(), "GetCode should return correct code")
		assert.Equal(t, provider, configErr.GetProvider(), "GetProvider should return correct provider")
		assert.Equal(t, format, configErr.GetFormat(), "GetFormat should return correct format")
		assert.Equal(t, path, configErr.GetConfigPath(), "GetConfigPath should return correct path")
	})

	// Test NewValidationError constructor
	t.Run("NewValidationError", func(t *testing.T) {
		op := "validate.config"
		code := config.ErrCodeValidationFailed
		underlyingErr := errors.New("validation failed")
		contextInfo := map[string]interface{}{
			"field":   "timeout",
			"value":   "invalid",
			"rule":    "duration",
			"message": "must be positive duration",
		}

		err := config.NewValidationError(op, code, underlyingErr, contextInfo)
		assert.Error(t, err, "NewValidationError should return an error")

		configErr, ok := config.AsConfigError(err)
		assert.True(t, ok, "Should be convertible to ConfigError")

		assert.Equal(t, op, configErr.GetOperation(), "GetOperation should return correct operation")
		assert.Equal(t, code, configErr.GetCode(), "GetCode should return correct code")

		retrievedContext := configErr.GetContext()
		assert.NotNil(t, retrievedContext, "GetContext should return non-nil context")
		assert.Equal(t, "timeout", retrievedContext["field"], "Context should contain field info")
		assert.Equal(t, "invalid", retrievedContext["value"], "Context should contain value info")
		assert.Equal(t, "duration", retrievedContext["rule"], "Context should contain rule info")
	})

	// Test error code constants
	t.Run("ErrorCodeConstants", func(t *testing.T) {
		// Test that all required error codes are defined and non-empty
		codes := []string{
			config.ErrCodeRegistryNotInitialized,
			config.ErrCodeProviderAlreadyRegistered,
			config.ErrCodeProviderNotFound,
			config.ErrCodeProviderCreationFailed,
			config.ErrCodeInvalidProviderName,
			config.ErrCodeLoadFailed,
			config.ErrCodeFileNotFound,
			config.ErrCodeFilePermissionDenied,
			config.ErrCodeParseError,
			config.ErrCodeFormatNotSupported,
			config.ErrCodeFormatDetectionFailed,
			config.ErrCodeValidationFailed,
			config.ErrCodeSchemaValidationFailed,
			config.ErrCodeRequiredFieldMissing,
			config.ErrCodeInvalidFieldValue,
			config.ErrCodeCrossFieldValidationFailed,
			config.ErrCodeProviderConfigInvalid,
			config.ErrCodeProviderOptionsInvalid,
			config.ErrCodeProviderNotSupported,
			config.ErrCodeProviderInitFailed,
			config.ErrCodeEnvVarNotFound,
			config.ErrCodeEnvVarParseError,
			config.ErrCodeEnvVarValidationFailed,
			config.ErrCodeWatchSetupFailed,
			config.ErrCodeReloadFailed,
			config.ErrCodeWatcherNotSupported,
			config.ErrCodeHealthCheckFailed,
			config.ErrCodeMetricsRecordingFailed,
			config.ErrCodeTracingSetupFailed,
			config.ErrCodeAllProvidersFailed,
			config.ErrCodeFallbackFailed,
			config.ErrCodeCompositeSetupFailed,
			config.ErrCodeInternalError,
			config.ErrCodeConfigurationCorrupted,
			config.ErrCodeResourceExhausted,
			config.ErrCodeContextCancelled,
			config.ErrCodeOperationTimeout,
		}

		for _, code := range codes {
			assert.NotEmpty(t, code, "Error code should not be empty: %s", code)
			assert.Contains(t, code, "_", "Error code should contain underscore: %s", code)
		}
	})

	// Test error wrapping utilities
	t.Run("WrapError", func(t *testing.T) {
		originalErr := errors.New("original error")
		op := "test.wrap"
		provider := "test-provider"

		wrappedErr := config.WrapError(originalErr, op, provider)
		assert.NotNil(t, wrappedErr, "WrapError should return non-nil error")

		configErr, ok := config.AsConfigError(wrappedErr)
		assert.True(t, ok, "Wrapped error should be ConfigError")

		assert.Equal(t, op, configErr.GetOperation(), "Wrapped error should have correct operation")
		assert.Equal(t, provider, configErr.GetProvider(), "Wrapped error should have correct provider")
		assert.Equal(t, originalErr, configErr.Unwrap(), "Wrapped error should unwrap to original")
	})

	t.Run("WrapError_Nil", func(t *testing.T) {
		wrappedErr := config.WrapError(nil, "test", "provider")
		assert.Nil(t, wrappedErr, "WrapError with nil should return nil")
	})

	// Test error classification and retry logic
	t.Run("RetryableErrors", func(t *testing.T) {
		testCases := []struct {
			code             string
			expectRetryable  bool
			expectRetryAfter time.Duration
		}{
			{config.ErrCodeLoadFailed, true, 2 * time.Second},
			{config.ErrCodeProviderCreationFailed, true, 3 * time.Second},
			{config.ErrCodeOperationTimeout, true, 5 * time.Second},
			{config.ErrCodeResourceExhausted, true, 10 * time.Second},
			{config.ErrCodeInternalError, true, 30 * time.Second},
			{config.ErrCodeHealthCheckFailed, true, 15 * time.Second},
			{config.ErrCodeFileNotFound, false, 0},
			{config.ErrCodeValidationFailed, false, 0},
			{config.ErrCodeProviderNotFound, false, 0},
			{config.ErrCodeFormatNotSupported, false, 0},
		}

		for _, tc := range testCases {
			err := config.NewError("test", tc.code, "test-provider", nil)
			configErr, ok := config.AsConfigError(err)
			assert.True(t, ok, "Should be ConfigError for code: %s", tc.code)

			assert.Equal(t, tc.expectRetryable, configErr.IsRetryable(),
				"Error code %s should have retryable=%t", tc.code, tc.expectRetryable)

			if tc.expectRetryable {
				assert.Equal(t, tc.expectRetryAfter, configErr.GetRetryAfter(),
					"Error code %s should have retry after %v", tc.code, tc.expectRetryAfter)
			}
		}
	})

	// Test error compatibility functions
	t.Run("ErrorCompatibilityFunctions", func(t *testing.T) {
		err := config.NewError("test", config.ErrCodeLoadFailed, "test-provider", nil)

		// Test AsConfigError
		configErr, ok := config.AsConfigError(err)
		assert.True(t, ok, "AsConfigError should succeed")
		assert.NotNil(t, configErr, "AsConfigError should return non-nil ConfigError")

		// Test IsConfigError
		assert.True(t, config.IsConfigError(err), "IsConfigError should return true")

		// Test GetErrorCode
		assert.Equal(t, config.ErrCodeLoadFailed, config.GetErrorCode(err),
			"GetErrorCode should return correct code")

		// Test IsRetryable
		assert.True(t, config.IsRetryable(err), "IsRetryable should return true for retryable error")

		// Test GetRetryAfter
		assert.Greater(t, config.GetRetryAfter(err), time.Duration(0),
			"GetRetryAfter should return positive duration for retryable error")

		// Test with non-ConfigError
		regularErr := errors.New("regular error")
		_, ok = config.AsConfigError(regularErr)
		assert.False(t, ok, "AsConfigError should fail for regular error")
		assert.False(t, config.IsConfigError(regularErr), "IsConfigError should return false for regular error")
		assert.Empty(t, config.GetErrorCode(regularErr), "GetErrorCode should return empty for regular error")
		assert.False(t, config.IsRetryable(regularErr), "IsRetryable should return false for regular error")
		assert.Equal(t, time.Duration(0), config.GetRetryAfter(regularErr),
			"GetRetryAfter should return 0 for regular error")
	})

	// Test errors.Is and errors.As compatibility
	t.Run("ErrorsPackageCompatibility", func(t *testing.T) {
		originalErr := errors.New("underlying error")
		configErr := config.NewError("test", config.ErrCodeLoadFailed, "test-provider", originalErr)

		// Test errors.Is
		assert.True(t, errors.Is(configErr, originalErr), "errors.Is should work with ConfigError")

		// Test errors.As
		var targetErr *config.ConfigError
		assert.True(t, errors.As(configErr, &targetErr), "errors.As should work with ConfigError")
		assert.NotNil(t, targetErr, "errors.As should set target")

		// Test with another ConfigError
		anotherErr := config.NewError("test", config.ErrCodeLoadFailed, "test-provider", nil)
		assert.True(t, errors.Is(anotherErr, configErr), "errors.Is should work between ConfigErrors with same code")
	})

	// Test context manipulation
	t.Run("ContextManipulation", func(t *testing.T) {
		var err error = config.NewError("test", config.ErrCodeLoadFailed, "test-provider", nil)

		// Add context
		err = config.WithContext(err, "key1", "value1")
		err = config.WithContext(err, "key2", 42)

		configErr, ok := config.AsConfigError(err)
		assert.True(t, ok, "Should be ConfigError after context addition")

		context := configErr.GetContext()
		assert.Equal(t, "value1", context["key1"], "Context should contain key1")
		assert.Equal(t, 42, context["key2"], "Context should contain key2")

		// Test WithRetryAfter
		err = config.WithRetryAfter(err, 5*time.Second)
		configErr, ok = config.AsConfigError(err)
		assert.True(t, ok, "Should be ConfigError after retry after addition")
		assert.True(t, configErr.IsRetryable(), "Error should be retryable after WithRetryAfter")
		assert.Equal(t, 5*time.Second, configErr.GetRetryAfter(), "Retry after should be set correctly")
	})

	// Test error chain preservation
	t.Run("ErrorChainPreservation", func(t *testing.T) {
		rootErr := errors.New("root cause")
		middleErr := fmt.Errorf("middle layer: %w", rootErr)
		configErr := config.WrapError(middleErr, "test", "provider")

		// Test that we can still access the root cause
		assert.True(t, errors.Is(configErr, rootErr), "Should be able to find root cause in error chain")

		// Test unwrapping
		unwrapped := configErr.Unwrap()
		assert.Equal(t, middleErr, unwrapped, "Should unwrap to the wrapped error")
	})
}

// TestConfigError_Performance tests error creation performance
func TestConfigError_Performance(t *testing.T) {
	// Test that error creation is reasonably fast
	start := time.Now()
	iterations := 10000

	for i := 0; i < iterations; i++ {
		err := config.NewError("perf.test", config.ErrCodeLoadFailed, "test-provider", errors.New("test"))
		_ = err.Error() // Force string formatting
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	// Error creation should be reasonably fast (< 100µs per error)
	assert.Less(t, avgDuration, 100*time.Microsecond,
		"Error creation should be faster than 100µs (avg: %v)", avgDuration)

	t.Logf("Created %d errors in %v (avg: %v per error)",
		iterations, duration, avgDuration)
}

// TestConfigError_ErrorConditions tests error conditions
func TestConfigError_ErrorConditions(t *testing.T) {
	t.Run("EmptyOperation", func(t *testing.T) {
		err := config.NewError("", config.ErrCodeLoadFailed, "provider", nil)
		assert.Error(t, err, "Should handle empty operation")
		assert.Contains(t, err.Error(), config.ErrCodeLoadFailed, "Should still contain error code")
	})

	t.Run("EmptyCode", func(t *testing.T) {
		err := config.NewError("test.op", "", "provider", nil)
		assert.Error(t, err, "Should handle empty error code")
		assert.Contains(t, err.Error(), "test.op", "Should still contain operation")
	})

	t.Run("NilUnderlyingError", func(t *testing.T) {
		err := config.NewError("test.op", config.ErrCodeLoadFailed, "provider", nil)
		configErr, ok := config.AsConfigError(err)
		assert.True(t, ok, "Should create ConfigError with nil underlying error")

		assert.Nil(t, configErr.Unwrap(), "Unwrap should return nil for nil underlying error")
		assert.False(t, errors.Is(err, errors.New("dummy")), "Should not match arbitrary errors")
	})

	t.Run("VeryLongStrings", func(t *testing.T) {
		longOp := string(make([]byte, 1000))
		for i := range longOp {
			longOp = longOp[:i] + "a" + longOp[i+1:]
		}

		longProvider := string(make([]byte, 500))
		for i := range longProvider {
			longProvider = longProvider[:i] + "b" + longProvider[i+1:]
		}

		err := config.NewError(longOp, config.ErrCodeLoadFailed, longProvider, nil)
		assert.Error(t, err, "Should handle very long strings")
		assert.Contains(t, err.Error(), config.ErrCodeLoadFailed, "Should still contain error code")
	})
}

// BenchmarkConfigError_NewError benchmarks error creation
func BenchmarkConfigError_NewError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.NewError("benchmark.operation", config.ErrCodeLoadFailed, "benchmark-provider", errors.New("benchmark error"))
	}
}

// BenchmarkConfigError_Error benchmarks error string formatting
func BenchmarkConfigError_Error(b *testing.B) {
	err := config.NewError("benchmark.operation", config.ErrCodeLoadFailed, "benchmark-provider", errors.New("benchmark error"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// BenchmarkConfigError_Is benchmarks error type checking
func BenchmarkConfigError_Is(b *testing.B) {
	err := config.NewError("benchmark.operation", config.ErrCodeLoadFailed, "benchmark-provider", errors.New("benchmark error"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.IsConfigError(err)
	}
}
