// Package twilio provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
package twilio

import (
	"context"
	"errors"
	"testing"
	"time"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/stretchr/testify/assert"
)

func TestAdvancedTwilioVoice(t *testing.T) {
	tests := []struct {
		component     *AdvancedMockTwilioVoice
		operations    func(ctx context.Context, comp *AdvancedMockTwilioVoice) error
		name          string
		expectedCalls int
		expectedError bool
	}{
		{
			name:      "successful start",
			component: NewAdvancedMockTwilioVoice(),
			operations: func(ctx context.Context, comp *AdvancedMockTwilioVoice) error {
				return comp.Start(ctx)
			},
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name: "error handling",
			component: NewAdvancedMockTwilioVoice(
				WithMockError(true, errors.New("test error")),
			),
			operations: func(ctx context.Context, comp *AdvancedMockTwilioVoice) error {
				return comp.Start(ctx)
			},
			expectedError: true,
			expectedCalls: 1,
		},
		{
			name:      "create session",
			component: NewAdvancedMockTwilioVoice(),
			operations: func(ctx context.Context, comp *AdvancedMockTwilioVoice) error {
				_, err := comp.CreateSession(ctx, &vbiface.SessionConfig{})
				return err
			},
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name:      "get active session count",
			component: NewAdvancedMockTwilioVoice(),
			operations: func(ctx context.Context, comp *AdvancedMockTwilioVoice) error {
				count := comp.GetActiveSessionCount()
				if count < 0 {
					return errors.New("invalid count")
				}
				return nil
			},
			expectedError: false,
			expectedCalls: 0, // GetActiveSessionCount doesn't increment callCount
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

			if tt.expectedCalls > 0 {
				assert.GreaterOrEqual(t, tt.component.GetCallCount(), tt.expectedCalls)
			}
		})
	}
}

func TestConcurrencyAdvancedTwilio(t *testing.T) {
	component := NewAdvancedMockTwilioVoice()
	numGoroutines := 100 // SC-003: 100 concurrent calls
	duration := 5 * time.Second

	runner := NewConcurrentTestRunner(numGoroutines, duration, func() error {
		ctx := context.Background()
		_, err := component.CreateSession(ctx, &vbiface.SessionConfig{})
		return err
	})

	err := runner.Run()
	assert.NoError(t, err)
}

func TestTwilioErrorHandling(t *testing.T) {
	tests := []struct {
		setupError  error
		operation   func() error
		name        string
		errorCode   string
		expectError bool
	}{
		{
			name:       "rate limit error",
			setupError: NewTwilioError("operation", ErrCodeTwilioRateLimit, errors.New("rate limit")),
			operation: func() error {
				return NewTwilioError("operation", ErrCodeTwilioRateLimit, errors.New("rate limit"))
			},
			expectError: true,
			errorCode:   ErrCodeTwilioRateLimit,
		},
		{
			name:       "timeout error",
			setupError: NewTwilioError("operation", ErrCodeTwilioTimeout, errors.New("timeout")),
			operation: func() error {
				return NewTwilioError("operation", ErrCodeTwilioTimeout, errors.New("timeout"))
			},
			expectError: true,
			errorCode:   ErrCodeTwilioTimeout,
		},
		{
			name:       "auth error",
			setupError: NewTwilioError("operation", ErrCodeTwilioAuthError, errors.New("auth failed")),
			operation: func() error {
				return NewTwilioError("operation", ErrCodeTwilioAuthError, errors.New("auth failed"))
			},
			expectError: true,
			errorCode:   ErrCodeTwilioAuthError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if tt.expectError {
				assert.Error(t, err)
				twilioErr := &TwilioError{}
				ok := errors.As(err, &twilioErr)
				if ok {
					assert.Equal(t, tt.errorCode, twilioErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTwilioConfigValidation(t *testing.T) {
	tests := []struct {
		config      *TwilioConfig
		name        string
		expectError bool
	}{
		{
			name: "valid config",
			config: &TwilioConfig{
				Config:      &vbiface.Config{Provider: "twilio"},
				AccountSID:  "AC1234567890abcdef",
				AuthToken:   "auth_token_123",
				PhoneNumber: "+15551234567",
			},
			expectError: false,
		},
		{
			name: "missing account_sid",
			config: &TwilioConfig{
				Config:      &vbiface.Config{Provider: "twilio"},
				AuthToken:   "auth_token_123",
				PhoneNumber: "+15551234567",
			},
			expectError: true,
		},
		{
			name: "missing auth_token",
			config: &TwilioConfig{
				Config:      &vbiface.Config{Provider: "twilio"},
				AccountSID:  "AC1234567890abcdef",
				PhoneNumber: "+15551234567",
			},
			expectError: true,
		},
		{
			name: "missing phone_number",
			config: &TwilioConfig{
				Config:     &vbiface.Config{Provider: "twilio"},
				AccountSID: "AC1234567890abcdef",
				AuthToken:  "auth_token_123",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkTwilioVoiceOperations(b *testing.B) {
	component := NewAdvancedMockTwilioVoice()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := component.CreateSession(ctx, &vbiface.SessionConfig{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTwilioVoiceConcurrent(b *testing.B) {
	component := NewAdvancedMockTwilioVoice()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := component.CreateSession(ctx, &vbiface.SessionConfig{})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestTwilioConfigMultiAccount(t *testing.T) {
	// Test multi-account configuration support (FR-041, T020a)
	config1 := &TwilioConfig{
		Config:      &vbiface.Config{Provider: "twilio"},
		AccountSID:  "AC1111111111111111",
		AuthToken:   "token1",
		PhoneNumber: "+15551111111",
		AccountName: "account1",
	}

	config2 := &TwilioConfig{
		Config:      &vbiface.Config{Provider: "twilio"},
		AccountSID:  "AC2222222222222222",
		AuthToken:   "token2",
		PhoneNumber: "+15552222222",
		AccountName: "account2",
	}

	// Both configs should be valid
	err1 := config1.Validate()
	err2 := config2.Validate()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, config1.AccountSID, config2.AccountSID)
	assert.NotEqual(t, config1.AccountName, config2.AccountName)
}
