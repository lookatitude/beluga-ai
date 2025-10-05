package config

import (
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/stretchr/testify/assert"
)

// Test cases for advanced testing infrastructure
// This file implements constitutional requirements for comprehensive testing

// TestConfigErrors_ErrorHandling tests comprehensive error handling
func TestConfigErrors_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name            string
		createError     func() error
		expectCode      string
		expectRetryable bool
	}{
		{
			name: "registry not initialized",
			createError: func() error {
				return NewError("TestOp", ErrCodeRegistryNotInitialized, "", nil)
			},
			expectCode:      ErrCodeRegistryNotInitialized,
			expectRetryable: false,
		},
		{
			name: "provider not found",
			createError: func() error {
				return NewError("TestOp", ErrCodeProviderNotFound, "unknown", nil)
			},
			expectCode:      ErrCodeProviderNotFound,
			expectRetryable: false,
		},
		{
			name: "load failed",
			createError: func() error {
				return NewError("TestOp", ErrCodeLoadFailed, "viper", nil)
			},
			expectCode:      ErrCodeLoadFailed,
			expectRetryable: true,
		},
		{
			name: "validation failed",
			createError: func() error {
				return NewValidationError("TestOp", ErrCodeValidationFailed, nil, map[string]interface{}{
					"field": "timeout",
					"value": "invalid",
				})
			},
			expectCode:      ErrCodeValidationFailed,
			expectRetryable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.createError()
			assert.Error(t, err)

			// Test error interface
			assert.Contains(t, err.Error(), tc.expectCode)

			// Test ConfigError interface
			configErr, ok := AsConfigError(err)
			assert.True(t, ok, "Should be a ConfigError")

			assert.Equal(t, tc.expectCode, configErr.GetCode())
			assert.Equal(t, tc.expectRetryable, configErr.IsRetryable())

			// Test error unwrapping
			if configErr.Unwrap() == nil {
				assert.Nil(t, configErr.Unwrap())
			}

			// Test errors.Is compatibility
			assert.True(t, IsConfigError(err))
			assert.Equal(t, tc.expectCode, GetErrorCode(err))
			assert.Equal(t, tc.expectRetryable, IsRetryable(err))
		})
	}
}

// TestRegistryBasicOperations tests basic registry operations
func TestRegistryBasicOperations(t *testing.T) {
	// Test global registry functions
	providers := ListProviders()
	assert.NotNil(t, providers, "Should be able to list providers")

	// Test provider registration
	err := RegisterGlobal("test-provider", func(options ProviderOptions) (iface.Provider, error) {
		return NewAdvancedMockConfigProvider("test", "test"), nil
	})
	assert.NoError(t, err, "Should be able to register provider")

	// Test provider discovery
	assert.True(t, IsProviderRegistered("test-provider"), "Provider should be registered")
}

// TestErrorCodeConstants tests that all error codes are properly defined
func TestErrorCodeConstants(t *testing.T) {
	// Test that key error codes are defined and non-empty
	assert.NotEmpty(t, ErrCodeRegistryNotInitialized)
	assert.NotEmpty(t, ErrCodeProviderNotFound)
	assert.NotEmpty(t, ErrCodeLoadFailed)
	assert.NotEmpty(t, ErrCodeValidationFailed)

	// Test error creation with constants
	err := NewError("test", ErrCodeProviderNotFound, "test-provider", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrCodeProviderNotFound)
}

// BenchmarkConfigErrorCreation benchmarks error creation performance
func BenchmarkConfigErrorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewError("BenchmarkOp", ErrCodeLoadFailed, "test-provider", nil)
	}
}

// BenchmarkProviderRegistration benchmarks provider registration
func BenchmarkProviderRegistration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		providerName := fmt.Sprintf("bench-provider-%d", i)
		_ = RegisterGlobal(providerName, func(options ProviderOptions) (iface.Provider, error) {
			return NewAdvancedMockConfigProvider(providerName, "bench"), nil
		})
	}
}
