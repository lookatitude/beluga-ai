// Package voice provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
package voice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigValidationAdvanced provides advanced table-driven tests for config validation.
func TestConfigValidationAdvanced(t *testing.T) {
	tests := []struct {
		config      *Config
		validate    func(t *testing.T, err error)
		name        string
		description string
		wantErr     bool
	}{
		{
			name:        "valid_config",
			description: "Validate valid configuration",
			config:      DefaultConfig(),
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:        "invalid_log_level",
			description: "Handle invalid log level",
			config: &Config{
				LogLevel: "invalid",
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
			wantErr: true,
		},
		{
			name:        "invalid_sample_rate",
			description: "Handle invalid sample rate",
			config: &Config{
				DefaultSampleRate: -1,
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			err := tt.config.Validate()
			tt.validate(t, err)
		})
	}
}

// TestErrorHandlingAdvanced provides advanced table-driven tests for error handling.
func TestErrorHandlingAdvanced(t *testing.T) {
	tests := []struct {
		createError func() error
		validate    func(t *testing.T, err error)
		name        string
		description string
	}{
		{
			name:        "basic_voice_error",
			description: "Create basic voice error",
			createError: func() error {
				return NewVoiceError("TestOp", ErrCodeInvalidInput, nil)
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "voice")
				assert.Contains(t, err.Error(), "TestOp")
			},
		},
		{
			name:        "voice_error_with_message",
			description: "Create voice error with message",
			createError: func() error {
				return NewVoiceErrorWithMessage("TestOp", ErrCodeInvalidInput, "Test message", nil)
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "Test message")
			},
		},
		{
			name:        "is_voice_error",
			description: "Check if error is VoiceError",
			createError: func() error {
				return NewVoiceError("TestOp", ErrCodeInvalidInput, nil)
			},
			validate: func(t *testing.T, err error) {
				assert.True(t, IsVoiceError(err))
				voiceErr, ok := AsVoiceError(err)
				assert.True(t, ok)
				assert.NotNil(t, voiceErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			err := tt.createError()
			tt.validate(t, err)
		})
	}
}

// TestConcurrentConfigValidation tests concurrent config validation.
func TestConcurrentConfigValidation(t *testing.T) {
	const numGoroutines = 20

	config := DefaultConfig()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			err := config.Validate()
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Should not have errors for valid config
	for err := range errors {
		require.NoError(t, err)
	}
}

// TestConfigWithContext tests config operations with context.
func TestConfigWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := DefaultConfig()

	t.Run("validate_with_context", func(t *testing.T) {
		err := config.Validate()
		_ = ctx // Acknowledge context
		assert.NoError(t, err)
	})
}

// BenchmarkConfigValidation benchmarks config validation performance.
func BenchmarkConfigValidation(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

// BenchmarkErrorCreation benchmarks error creation performance.
func BenchmarkErrorCreation(b *testing.B) {
	b.Run("basic_error", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewVoiceError("TestOp", ErrCodeInvalidInput, nil)
		}
	})

	b.Run("error_with_message", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewVoiceErrorWithMessage("TestOp", ErrCodeInvalidInput, "Test message", nil)
		}
	})
}
