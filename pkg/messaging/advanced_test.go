// Package messaging provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedMessaging(t *testing.T) {
	tests := []struct {
		name           string
		component      *AdvancedMockMessaging
		operations     func(ctx context.Context, comp *AdvancedMockMessaging) error
		expectedError  bool
		expectedCalls  int
		validateResult func(t *testing.T, result interface{})
	}{
		{
			name:      "successful start",
			component: NewAdvancedMockMessaging(),
			operations: func(ctx context.Context, comp *AdvancedMockMessaging) error {
				return comp.Start(ctx)
			},
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name: "error handling",
			component: NewAdvancedMockMessaging(
				WithMockError(true, errors.New("test error")),
			),
			operations: func(ctx context.Context, comp *AdvancedMockMessaging) error {
				return comp.Start(ctx)
			},
			expectedError: true,
			expectedCalls: 1,
		},
		{
			name:      "create conversation",
			component: NewAdvancedMockMessaging(),
			operations: func(ctx context.Context, comp *AdvancedMockMessaging) error {
				_, err := comp.CreateConversation(ctx, &iface.ConversationConfig{
					FriendlyName: "Test Conversation",
				})
				return err
			},
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name:      "send message",
			component: NewAdvancedMockMessaging(),
			operations: func(ctx context.Context, comp *AdvancedMockMessaging) error {
				conv, err := comp.CreateConversation(ctx, &iface.ConversationConfig{
					FriendlyName: "Test",
				})
				if err != nil {
					return err
				}
				return comp.SendMessage(ctx, conv.ConversationSID, &iface.Message{
					Body: "Hello",
				})
			},
			expectedError: false,
			expectedCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.operations(ctx, tt.component)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedCalls, tt.component.GetCallCount())

			if tt.validateResult != nil {
				tt.validateResult(t, nil)
			}
		})
	}
}

func TestConcurrencyAdvanced(t *testing.T) {
	component := NewAdvancedMockMessaging()
	numGoroutines := 10
	duration := 1 * time.Second

	runner := NewConcurrentTestRunner(numGoroutines, duration, func() error {
		ctx := context.Background()
		_, err := component.CreateConversation(ctx, &iface.ConversationConfig{
			FriendlyName: "Concurrent Test",
		})
		return err
	})

	err := runner.Run()
	assert.NoError(t, err)
}

func TestMessagingErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupError  error
		operation   func() error
		expectError bool
		errorCode   string
	}{
		{
			name:       "rate limit error",
			setupError: NewMessagingError("operation", ErrCodeRateLimit, errors.New("rate limit")),
			operation: func() error {
				return NewMessagingError("operation", ErrCodeRateLimit, errors.New("rate limit"))
			},
			expectError: true,
			errorCode:   ErrCodeRateLimit,
		},
		{
			name:       "timeout error",
			setupError: NewMessagingError("operation", ErrCodeTimeout, errors.New("timeout")),
			operation: func() error {
				return NewMessagingError("operation", ErrCodeTimeout, errors.New("timeout"))
			},
			expectError: true,
			errorCode:   ErrCodeTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorCode, GetMessagingErrorCode(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProviderRegistry(t *testing.T) {
	registry := GetRegistry()

	// Test registration
	registry.Register("test", func(ctx context.Context, config *Config) (iface.ConversationalBackend, error) {
		return NewAdvancedMockMessaging(), nil
	})

	// Test listing
	providers := registry.ListProviders()
	assert.Contains(t, providers, "test")

	// Test creation
	ctx := context.Background()
	config := DefaultConfig()
	config.Provider = "test"
	backend, err := registry.Create(ctx, "test", config)
	require.NoError(t, err)
	assert.NotNil(t, backend)

	// Test unknown provider
	_, err = registry.Create(ctx, "unknown", config)
	assert.Error(t, err)
	assert.Equal(t, ErrCodeProviderNotFound, GetMessagingErrorCode(err))
}

// T082: Test coverage for pkg/messaging/messaging.go
func TestNewBackend(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		providerName string
		config       *Config
		shouldError  bool
		setup        func()
		teardown     func()
	}{
		{
			name:         "create backend with nil config uses default",
			providerName: "test",
			config:       nil,
			shouldError:  false,
			setup: func() {
				registry := GetRegistry()
				registry.Register("test", func(ctx context.Context, config *Config) (iface.ConversationalBackend, error) {
					return NewAdvancedMockMessaging(), nil
				})
			},
		},
		{
			name:         "create backend with config",
			providerName: "test",
			config:       DefaultConfig(),
			shouldError:  false,
			setup: func() {
				registry := GetRegistry()
				registry.Register("test", func(ctx context.Context, config *Config) (iface.ConversationalBackend, error) {
					return NewAdvancedMockMessaging(), nil
				})
			},
		},
		{
			name:         "create backend with provider name override",
			providerName: "test",
			config: func() *Config {
				c := DefaultConfig()
				c.Provider = "wrong"
				return c
			}(),
			shouldError: false,
			setup: func() {
				registry := GetRegistry()
				registry.Register("test", func(ctx context.Context, config *Config) (iface.ConversationalBackend, error) {
					return NewAdvancedMockMessaging(), nil
				})
			},
		},
		{
			name:         "create backend with empty provider name uses config",
			providerName: "",
			config: func() *Config {
				c := DefaultConfig()
				c.Provider = "test"
				return c
			}(),
			shouldError: false,
			setup: func() {
				registry := GetRegistry()
				registry.Register("test", func(ctx context.Context, config *Config) (iface.ConversationalBackend, error) {
					return NewAdvancedMockMessaging(), nil
				})
			},
		},
		{
			name:         "create backend with unknown provider",
			providerName: "unknown",
			config:       DefaultConfig(),
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.teardown != nil {
				defer tt.teardown()
			}

			backend, err := NewBackend(ctx, tt.providerName, tt.config)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, backend)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, backend)
			}
		})
	}
}

// T083: Test coverage for pkg/messaging/config.go
func TestConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		assert.NotNil(t, config)
		assert.Equal(t, 30*time.Second, config.Timeout)
		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, time.Second, config.RetryDelay)
		assert.Equal(t, 2.0, config.RetryBackoff)
		assert.True(t, config.EnableTracing)
		assert.True(t, config.EnableMetrics)
		assert.True(t, config.EnableStructuredLogging)
		assert.NotNil(t, config.ProviderSpecific)
	})

	t.Run("Config_Validate", func(t *testing.T) {
		tests := []struct {
			name        string
			config      *Config
			shouldError bool
		}{
			{
				name: "valid config",
				config: &Config{
					Provider:     "test",
					Timeout:      30 * time.Second,
					RetryDelay:   time.Second,
					RetryBackoff: 2.0,
				},
				shouldError: false,
			},
			{
				name:        "nil config",
				config:      nil,
				shouldError: true,
			},
			{
				name:        "missing provider",
				config:      &Config{Timeout: 30 * time.Second},
				shouldError: true,
			},
			{
				name:        "timeout too short",
				config:      &Config{Provider: "test", Timeout: 500 * time.Millisecond},
				shouldError: true,
			},
			{
				name:        "timeout too long",
				config:      &Config{Provider: "test", Timeout: 10 * time.Minute},
				shouldError: true,
			},
			{
				name: "valid timeout at minimum",
				config: &Config{
					Provider:     "test",
					Timeout:      time.Second,
					RetryDelay:   time.Second,
					RetryBackoff: 2.0,
				},
				shouldError: false,
			},
			{
				name: "valid timeout at maximum",
				config: &Config{
					Provider:     "test",
					Timeout:      5 * time.Minute,
					RetryDelay:   time.Second,
					RetryBackoff: 2.0,
				},
				shouldError: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.config.Validate()
				if tt.shouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ConfigOption_WithProvider", func(t *testing.T) {
		config := DefaultConfig()
		WithProvider("test")(config)
		assert.Equal(t, "test", config.Provider)
	})

	t.Run("ConfigOption_WithTimeout", func(t *testing.T) {
		config := DefaultConfig()
		WithTimeout(60 * time.Second)(config)
		assert.Equal(t, 60*time.Second, config.Timeout)
	})

	t.Run("ConfigOption_WithRetryConfig", func(t *testing.T) {
		config := DefaultConfig()
		WithRetryConfig(5, 2*time.Second, 3.0)(config)
		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 2*time.Second, config.RetryDelay)
		assert.Equal(t, 3.0, config.RetryBackoff)
	})

	t.Run("ConfigOption_WithObservability", func(t *testing.T) {
		config := DefaultConfig()
		WithObservability(false, false, false)(config)
		assert.False(t, config.EnableTracing)
		assert.False(t, config.EnableMetrics)
		assert.False(t, config.EnableStructuredLogging)
	})

	t.Run("ConfigOption_WithProviderSpecific", func(t *testing.T) {
		config := DefaultConfig()
		WithProviderSpecific("key1", "value1")(config)
		assert.Equal(t, "value1", config.ProviderSpecific["key1"])

		WithProviderSpecific("key2", 42)(config)
		assert.Equal(t, 42, config.ProviderSpecific["key2"])
	})

	t.Run("Config_MergeOptions", func(t *testing.T) {
		config := DefaultConfig()
		config.MergeOptions(
			WithProvider("test"),
			WithTimeout(60*time.Second),
		)
		assert.Equal(t, "test", config.Provider)
		assert.Equal(t, 60*time.Second, config.Timeout)
	})

	t.Run("NewConfig", func(t *testing.T) {
		config := NewConfig(
			WithProvider("test"),
			WithTimeout(60*time.Second),
		)
		assert.Equal(t, "test", config.Provider)
		assert.Equal(t, 60*time.Second, config.Timeout)
	})

	t.Run("ValidateConfig", func(t *testing.T) {
		ctx := context.Background()

		tests := []struct {
			name        string
			config      *Config
			shouldError bool
		}{
			{
				name:        "nil config",
				config:      nil,
				shouldError: true,
			},
			{
				name: "valid config",
				config: &Config{
					Provider:     "twilio",
					Timeout:      30 * time.Second,
					RetryDelay:   time.Second,
					RetryBackoff: 2.0,
				},
				shouldError: false,
			},
			{
				name: "valid mock provider",
				config: &Config{
					Provider:     "mock",
					Timeout:      30 * time.Second,
					RetryDelay:   time.Second,
					RetryBackoff: 2.0,
				},
				shouldError: false,
			},
			{
				name:        "empty provider",
				config:      &Config{Provider: "", Timeout: 30 * time.Second},
				shouldError: true,
			},
			{
				name:        "invalid config validation",
				config:      &Config{Timeout: 500 * time.Millisecond},
				shouldError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateConfig(ctx, tt.config)
				if tt.shouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

// T084: Test coverage for pkg/messaging/errors.go
func TestMessagingErrors(t *testing.T) {
	t.Run("NewMessagingError", func(t *testing.T) {
		err := NewMessagingError("test_op", ErrCodeTimeout, errors.New("timeout occurred"))
		assert.NotNil(t, err)
		assert.Equal(t, "test_op", err.Op)
		assert.Equal(t, ErrCodeTimeout, err.Code)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test_op")
		assert.Contains(t, err.Error(), ErrCodeTimeout)
	})

	t.Run("NewMessagingErrorWithMessage", func(t *testing.T) {
		err := NewMessagingErrorWithMessage("test_op", ErrCodeRateLimit, "rate limit exceeded", errors.New("underlying"))
		assert.NotNil(t, err)
		assert.Equal(t, "test_op", err.Op)
		assert.Equal(t, ErrCodeRateLimit, err.Code)
		assert.Equal(t, "rate limit exceeded", err.Message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit exceeded")
	})

	t.Run("NewMessagingErrorWithDetails", func(t *testing.T) {
		details := map[string]any{"key": "value"}
		err := NewMessagingErrorWithDetails("test_op", ErrCodeNetworkError, "network failed", errors.New("underlying"), details)
		assert.NotNil(t, err)
		assert.Equal(t, details, err.Details)
	})

	t.Run("MessagingError_Unwrap", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := NewMessagingError("test_op", ErrCodeTimeout, underlying)
		assert.Equal(t, underlying, err.Unwrap())
	})

	t.Run("IsMessagingError", func(t *testing.T) {
		err := NewMessagingError("test_op", ErrCodeTimeout, errors.New("timeout"))
		assert.True(t, IsMessagingError(err))

		regularErr := errors.New("regular error")
		assert.False(t, IsMessagingError(regularErr))
	})

	t.Run("GetMessagingError", func(t *testing.T) {
		err := NewMessagingError("test_op", ErrCodeTimeout, errors.New("timeout"))
		msgErr := GetMessagingError(err)
		assert.NotNil(t, msgErr)
		assert.Equal(t, "test_op", msgErr.Op)

		regularErr := errors.New("regular error")
		assert.Nil(t, GetMessagingError(regularErr))
	})

	t.Run("GetMessagingErrorCode", func(t *testing.T) {
		err := NewMessagingError("test_op", ErrCodeRateLimit, errors.New("rate limit"))
		assert.Equal(t, ErrCodeRateLimit, GetMessagingErrorCode(err))

		regularErr := errors.New("regular error")
		assert.Equal(t, "", GetMessagingErrorCode(regularErr))
	})

	t.Run("IsRetryableError", func(t *testing.T) {
		tests := []struct {
			name      string
			err       error
			retryable bool
		}{
			{
				name:      "nil error",
				err:       nil,
				retryable: false,
			},
			{
				name:      "rate limit error",
				err:       NewMessagingError("op", ErrCodeRateLimit, errors.New("rate limit")),
				retryable: true,
			},
			{
				name:      "network error",
				err:       NewMessagingError("op", ErrCodeNetworkError, errors.New("network")),
				retryable: true,
			},
			{
				name:      "timeout error",
				err:       NewMessagingError("op", ErrCodeTimeout, errors.New("timeout")),
				retryable: true,
			},
			{
				name:      "internal error",
				err:       NewMessagingError("op", ErrCodeInternalError, errors.New("internal")),
				retryable: true,
			},
			{
				name:      "authentication error",
				err:       NewMessagingError("op", ErrCodeAuthentication, errors.New("auth")),
				retryable: false,
			},
			{
				name:      "invalid config error",
				err:       NewMessagingError("op", ErrCodeInvalidConfig, errors.New("config")),
				retryable: false,
			},
			{
				name:      "context canceled",
				err:       context.Canceled,
				retryable: false,
			},
			{
				name:      "context deadline exceeded",
				err:       context.DeadlineExceeded,
				retryable: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.retryable, IsRetryableError(tt.err))
			})
		}
	})

	t.Run("WrapError", func(t *testing.T) {
		err := WrapError("test_op", errors.New("original error"))
		assert.Error(t, err)
		assert.True(t, IsMessagingError(err))

		messagingErr := NewMessagingError("old_op", ErrCodeTimeout, errors.New("timeout"))
		wrapped := WrapError("new_op", messagingErr)
		msgErr := GetMessagingError(wrapped)
		assert.Equal(t, "new_op", msgErr.Op)

		assert.Nil(t, WrapError("op", nil))
	})

	t.Run("MapHTTPError", func(t *testing.T) {
		tests := []struct {
			name       string
			statusCode int
			code       string
		}{
			{"unauthorized", 401, ErrCodeAuthentication},
			{"forbidden", 403, ErrCodeAuthorization},
			{"not found", 404, ErrCodeNotFound},
			{"too many requests", 429, ErrCodeRateLimit},
			{"bad request", 400, ErrCodeInvalidInput},
			{"internal server error", 500, ErrCodeInternalError},
			{"bad gateway", 502, ErrCodeInternalError},
			{"service unavailable", 503, ErrCodeInternalError},
			{"gateway timeout", 504, ErrCodeInternalError},
			{"unknown", 418, ErrCodeNetworkError},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := MapHTTPError("test_op", tt.statusCode, errors.New("http error"))
				assert.Equal(t, tt.code, err.Code)
			})
		}
	})
}

func BenchmarkMessagingOperations(b *testing.B) {
	component := NewAdvancedMockMessaging()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := component.CreateConversation(ctx, &iface.ConversationConfig{
			FriendlyName: "Benchmark",
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessagingConcurrent(b *testing.B) {
	component := NewAdvancedMockMessaging()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := component.CreateConversation(ctx, &iface.ConversationConfig{
				FriendlyName: "Concurrent Benchmark",
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
