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
