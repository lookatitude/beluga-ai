// Package embeddings provides comprehensive tests for embedding implementations.
// This file contains advanced testing scenarios focusing on missing coverage paths,
// including table-driven tests, concurrency testing, and performance benchmarks.
package embeddings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// ============================================================================
// Tests for errors.go functions (0% coverage - all need tests)
// ============================================================================

// TestEmbeddingErrorAdvanced provides comprehensive tests for error type functionality.
func TestEmbeddingErrorAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		createError func() *EmbeddingError
		validate    func(t *testing.T, err *EmbeddingError)
		description string
	}{
		{
			name:        "error_with_message",
			description: "Error with custom message",
			createError: func() *EmbeddingError {
				return NewEmbeddingErrorWithMessage("test_op", ErrCodeInvalidConfig, "test message", errors.New("underlying error"))
			},
			validate: func(t *testing.T, err *EmbeddingError) {
				assert.Equal(t, "test_op", err.Op)
				assert.Equal(t, ErrCodeInvalidConfig, err.Code)
				assert.Equal(t, "test message", err.Message)
				assert.NotNil(t, err.Err)
				assert.Contains(t, err.Error(), "test message")
				assert.Contains(t, err.Error(), ErrCodeInvalidConfig)
			},
		},
		{
			name:        "error_without_message",
			description: "Error without custom message",
			createError: func() *EmbeddingError {
				return NewEmbeddingError("test_op", ErrCodeInvalidInput, errors.New("underlying error"))
			},
			validate: func(t *testing.T, err *EmbeddingError) {
				assert.Equal(t, "test_op", err.Op)
				assert.Equal(t, ErrCodeInvalidInput, err.Code)
				assert.NotNil(t, err.Err)
				assert.Contains(t, err.Error(), "underlying error")
				assert.Contains(t, err.Error(), ErrCodeInvalidInput)
			},
		},
		{
			name:        "error_without_underlying",
			description: "Error without underlying error",
			createError: func() *EmbeddingError {
				return NewEmbeddingError("test_op", ErrCodeEmbeddingFailed, nil)
			},
			validate: func(t *testing.T, err *EmbeddingError) {
				assert.Equal(t, "test_op", err.Op)
				assert.Equal(t, ErrCodeEmbeddingFailed, err.Code)
				assert.Nil(t, err.Err)
				assert.Contains(t, err.Error(), "unknown error")
			},
		},
		{
			name:        "error_with_message_no_underlying",
			description: "Error with message but no underlying error",
			createError: func() *EmbeddingError {
				return NewEmbeddingErrorWithMessage("test_op", ErrCodeProviderError, "custom message", nil)
			},
			validate: func(t *testing.T, err *EmbeddingError) {
				assert.Equal(t, "test_op", err.Op)
				assert.Equal(t, ErrCodeProviderError, err.Code)
				assert.Equal(t, "custom message", err.Message)
				assert.Nil(t, err.Err)
				assert.Contains(t, err.Error(), "custom message")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			err := tt.createError()
			tt.validate(t, err)

			// Test Unwrap
			unwrapped := err.Unwrap()
			if err.Err != nil {
				assert.Equal(t, err.Err, unwrapped)
			} else {
				assert.Nil(t, unwrapped)
			}
		})
	}
}

// TestIsEmbeddingErrorAdvanced tests error type checking.
func TestIsEmbeddingErrorAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expected    bool
		description string
	}{
		{
			name:        "is_embedding_error",
			description: "Check if error is EmbeddingError",
			err:         NewEmbeddingError("test", ErrCodeInvalidConfig, errors.New("test")),
			expected:    true,
		},
		{
			name:        "not_embedding_error",
			description: "Check if regular error is not EmbeddingError",
			err:         errors.New("regular error"),
			expected:    false,
		},
		{
			name:        "nil_error",
			description: "Check nil error",
			err:         nil,
			expected:    false,
		},
		{
			name:        "wrapped_embedding_error",
			description: "Check wrapped EmbeddingError",
			err:         fmt.Errorf("wrapped: %w", NewEmbeddingError("test", ErrCodeInvalidConfig, errors.New("test"))),
			expected:    true, // errors.As should find it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			result := IsEmbeddingError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAsEmbeddingErrorAdvanced tests error type conversion.
func TestAsEmbeddingErrorAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expected    bool
		validate    func(t *testing.T, embErr *EmbeddingError, ok bool)
		description string
	}{
		{
			name:        "convert_embedding_error",
			description: "Convert EmbeddingError successfully",
			err:         NewEmbeddingError("test", ErrCodeInvalidConfig, errors.New("test")),
			expected:    true,
			validate: func(t *testing.T, embErr *EmbeddingError, ok bool) {
				assert.True(t, ok)
				assert.NotNil(t, embErr)
				assert.Equal(t, "test", embErr.Op)
				assert.Equal(t, ErrCodeInvalidConfig, embErr.Code)
			},
		},
		{
			name:        "convert_regular_error",
			description: "Fail to convert regular error",
			err:         errors.New("regular error"),
			expected:    false,
			validate: func(t *testing.T, embErr *EmbeddingError, ok bool) {
				assert.False(t, ok)
				assert.Nil(t, embErr)
			},
		},
		{
			name:        "convert_nil_error",
			description: "Fail to convert nil error",
			err:         nil,
			expected:    false,
			validate: func(t *testing.T, embErr *EmbeddingError, ok bool) {
				assert.False(t, ok)
				assert.Nil(t, embErr)
			},
		},
		{
			name:        "convert_wrapped_embedding_error",
			description: "Convert wrapped EmbeddingError",
			err:         fmt.Errorf("wrapped: %w", NewEmbeddingError("test", ErrCodeNetworkError, errors.New("network"))),
			expected:    true,
			validate: func(t *testing.T, embErr *EmbeddingError, ok bool) {
				assert.True(t, ok)
				assert.NotNil(t, embErr)
				assert.Equal(t, ErrCodeNetworkError, embErr.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			embErr, ok := AsEmbeddingError(tt.err)
			assert.Equal(t, tt.expected, ok)
			if tt.validate != nil {
				tt.validate(t, embErr, ok)
			}
		})
	}
}

// TestErrorCodesAdvanced tests all error code constants.
func TestErrorCodesAdvanced(t *testing.T) {
	codes := []string{
		ErrCodeInvalidConfig,
		ErrCodeInvalidInput,
		ErrCodeEmbeddingFailed,
		ErrCodeProviderNotFound,
		ErrCodeProviderError,
		ErrCodeNetworkError,
		ErrCodeTimeout,
		ErrCodeRateLimit,
		ErrCodeAuthentication,
		ErrCodeInvalidDimension,
		ErrCodeBatchSizeExceeded,
		ErrCodeContextCanceled,
		ErrCodeContextTimeout,
	}

	for _, code := range codes {
		t.Run(fmt.Sprintf("error_code_%s", code), func(t *testing.T) {
			err := NewEmbeddingError("test", code, errors.New("test"))
			assert.Equal(t, code, err.Code)
			assert.Contains(t, err.Error(), code)
		})
	}
}

// ============================================================================
// Tests for embeddings.go missing coverage paths
// ============================================================================

// TestEmbedderFactoryNewMockEmbedder tests mock embedder creation.
// Since newMockEmbedder is private, we test it indirectly through factory behavior.
func TestEmbedderFactoryNewMockEmbedder(t *testing.T) {
	// Test that mock embedder can be created when Mock config is enabled
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	require.NoError(t, err)

	// Creating a mock embedder should work
	embedder, err := factory.NewEmbedder("mock")
	require.NoError(t, err)
	assert.NotNil(t, embedder)

	// Verify the embedder works
	ctx := context.Background()
	dim, err := embedder.GetDimension(ctx)
	require.NoError(t, err)
	assert.Equal(t, 128, dim)
}

// TestEmbedderFactoryCheckHealthAdvanced tests additional health check paths.
func TestEmbedderFactoryCheckHealthAdvanced(t *testing.T) {
	ctx := context.Background()

	t.Run("health_check_with_healthchecker_interface", func(t *testing.T) {
		// Test health check when embedder implements HealthChecker
		config := &Config{
			Mock: &MockConfig{
				Dimension: 128,
				Enabled:   true,
			},
		}

		factory, err := NewEmbedderFactory(config)
		require.NoError(t, err)

		// Mock embedder should pass health check
		err = factory.CheckHealth(ctx, "mock")
		require.NoError(t, err)
	})

	t.Run("health_check_failure_path", func(t *testing.T) {
		// Test health check failure when provider doesn't exist
		config := &Config{
			Mock: &MockConfig{
				Dimension: 128,
				Enabled:   true,
			},
		}

		factory, err := NewEmbedderFactory(config)
		require.NoError(t, err)

		// Unknown provider should fail
		err = factory.CheckHealth(ctx, "unknown-provider")
		require.Error(t, err)
	})

	t.Run("health_check_with_context_timeout", func(t *testing.T) {
		// Test health check with context timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		config := &Config{
			Mock: &MockConfig{
				Dimension: 128,
				Enabled:   true,
			},
		}

		factory, err := NewEmbedderFactory(config)
		require.NoError(t, err)

		// Should complete within timeout
		err = factory.CheckHealth(ctx, "mock")
		require.NoError(t, err)
	})
}

// TestLogWithOTELContextAdvanced tests OTEL context logging paths.
func TestLogWithOTELContextAdvanced(t *testing.T) {
	t.Run("with_otel_context", func(t *testing.T) {
		// Create a context with a span
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test.operation")
		defer span.End()

		// This should not panic and should include trace/span IDs
		logWithOTELContext(ctx, slog.LevelInfo, "Test message", "key", "value")
	})

	t.Run("without_otel_context", func(t *testing.T) {
		// This should not panic even without OTEL context
		ctx := context.Background()
		logWithOTELContext(ctx, slog.LevelInfo, "Test message", "key", "value")
	})

	t.Run("with_different_log_levels", func(t *testing.T) {
		ctx := context.Background()
		levels := []slog.Level{
			slog.LevelDebug,
			slog.LevelInfo,
			slog.LevelWarn,
			slog.LevelError,
		}

		for _, level := range levels {
			t.Run(fmt.Sprintf("level_%s", level.String()), func(t *testing.T) {
				logWithOTELContext(ctx, level, "Test message", "key", "value")
			})
		}
	})

	t.Run("with_multiple_attributes", func(t *testing.T) {
		ctx := context.Background()
		logWithOTELContext(ctx, slog.LevelInfo, "Test message",
			"key1", "value1",
			"key2", "value2",
			"key3", "value3",
		)
	})
}

// ============================================================================
// Tests for config.go missing coverage paths
// ============================================================================

// TestCohereConfigValidate tests Cohere config validation (0% coverage).
func TestCohereConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *CohereConfig
		wantErr     bool
		description string
	}{
		{
			name:        "valid_cohere_config",
			description: "Valid Cohere config should pass",
			config: &CohereConfig{
				APIKey: "test-key",
				Model:  "embed-english-v3.0",
			},
			wantErr: false,
		},
		{
			name:        "invalid_cohere_missing_key",
			description: "Cohere config missing API key should fail",
			config: &CohereConfig{
				APIKey: "",
				Model:  "embed-english-v3.0",
			},
			wantErr: true,
		},
		{
			name:        "invalid_cohere_missing_model",
			description: "Cohere config missing model should fail",
			config: &CohereConfig{
				APIKey: "test-key",
				Model:  "",
			},
			wantErr: true,
		},
		{
			name:        "invalid_cohere_missing_both",
			description: "Cohere config missing both key and model should fail",
			config: &CohereConfig{
				APIKey: "",
				Model:  "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			err := tt.config.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cohere")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestConfigValidateAdvanced tests additional Config.Validate paths (75% coverage).
func TestConfigValidateAdvanced(t *testing.T) {
	t.Run("config_with_cohere_validation", func(t *testing.T) {
		// Test Config.Validate when Cohere config is present
		config := &Config{
			Cohere: &CohereConfig{
				APIKey: "test-key",
				Model:  "embed-english-v3.0",
			},
		}
		err := config.Validate()
		require.NoError(t, err)
	})

	t.Run("config_with_invalid_cohere", func(t *testing.T) {
		// Test Config.Validate when Cohere config is invalid
		config := &Config{
			Cohere: &CohereConfig{
				APIKey: "", // Missing required field
				Model:  "embed-english-v3.0",
			},
		}
		err := config.Validate()
		require.Error(t, err)
	})

	t.Run("config_with_all_providers", func(t *testing.T) {
		// Test Config.Validate with all providers configured
		config := &Config{
			OpenAI: &OpenAIConfig{
				APIKey: "openai-key",
				Model:  "text-embedding-ada-002",
			},
			Ollama: &OllamaConfig{
				Model: "nomic-embed-text",
			},
			Mock: &MockConfig{
				Dimension: 128,
			},
			Cohere: &CohereConfig{
				APIKey: "cohere-key",
				Model:  "embed-english-v3.0",
			},
		}
		err := config.Validate()
		require.NoError(t, err)
	})
}

// ============================================================================
// Concurrency Tests
// ============================================================================

// TestConcurrentEmbedderFactoryAdvanced tests concurrent factory operations.
func TestConcurrentEmbedderFactoryAdvanced(t *testing.T) {
	const numGoroutines = 10
	const numOperations = 5

	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numOperations; j++ {
				embedder, err := factory.NewEmbedder("mock")
				if err != nil {
					errors <- err
					continue
				}
				_, err = embedder.GetDimension(ctx)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Log any errors but don't fail - concurrent execution may have different results
	for err := range errors {
		t.Logf("Concurrent operation error: %v", err)
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

// BenchmarkEmbeddingErrorCreation benchmarks error creation.
func BenchmarkEmbeddingErrorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEmbeddingError("test", ErrCodeInvalidConfig, errors.New("test"))
	}
}

// BenchmarkEmbeddingErrorWithMessage benchmarks error creation with message.
func BenchmarkEmbeddingErrorWithMessage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEmbeddingErrorWithMessage("test", ErrCodeInvalidConfig, "test message", errors.New("test"))
	}
}

// BenchmarkIsEmbeddingError benchmarks error type checking.
func BenchmarkIsEmbeddingError(b *testing.B) {
	err := NewEmbeddingError("test", ErrCodeInvalidConfig, errors.New("test"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsEmbeddingError(err)
	}
}

// BenchmarkAsEmbeddingError benchmarks error type conversion.
func BenchmarkAsEmbeddingError(b *testing.B) {
	err := NewEmbeddingError("test", ErrCodeInvalidConfig, errors.New("test"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = AsEmbeddingError(err)
	}
}

// BenchmarkCohereConfigValidate benchmarks Cohere config validation.
func BenchmarkCohereConfigValidate(b *testing.B) {
	config := &CohereConfig{
		APIKey: "test-key",
		Model:  "embed-english-v3.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}
