// Package retrievers provides tests for the retrievers package.
package retrievers

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
)

func TestMain(m *testing.M) {
	// Set up test logger to discard logs during tests
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	slog.SetDefault(logger)

	os.Exit(m.Run())
}

func TestNewVectorStoreRetriever(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "valid configuration with defaults",
			options: []Option{},
			wantErr: false,
		},
		{
			name:    "invalid k value too low",
			options: []Option{WithDefaultK(0)},
			wantErr: true,
		},
		{
			name:    "invalid k value too high",
			options: []Option{WithDefaultK(101)},
			wantErr: true,
		},
		{
			name: "valid custom configuration",
			options: []Option{
				WithDefaultK(5),
				WithTimeout(10 * time.Second),
				WithTracing(false),
				WithMetrics(false),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For this test, we'll use nil for vectorStore since we're just testing config validation
			_, err := NewVectorStoreRetriever(nil, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVectorStoreRetriever() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewVectorStoreRetrieverFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  VectorStoreRetrieverConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: VectorStoreRetrieverConfig{
				K:              5,
				ScoreThreshold: 0.7,
				Timeout:        30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid k value",
			config: VectorStoreRetrieverConfig{
				K:              0,
				ScoreThreshold: 0.7,
				Timeout:        30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid score threshold",
			config: VectorStoreRetrieverConfig{
				K:              5,
				ScoreThreshold: 1.5,
				Timeout:        30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: VectorStoreRetrieverConfig{
				K:              5,
				ScoreThreshold: 0.7,
				Timeout:        500 * time.Millisecond,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVectorStoreRetrieverFromConfig(nil, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVectorStoreRetrieverFromConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetrieverOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		expected *RetrieverOptions
	}{
		{
			name:    "default options",
			options: []Option{},
			expected: &RetrieverOptions{
				DefaultK:       4,
				ScoreThreshold: 0.0,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				EnableTracing:  true,
				EnableMetrics:  true,
			},
		},
		{
			name: "custom options",
			options: []Option{
				WithDefaultK(10),
				WithMaxRetries(5),
				WithTimeout(60 * time.Second),
				WithTracing(false),
				WithMetrics(false),
			},
			expected: &RetrieverOptions{
				DefaultK:       10,
				ScoreThreshold: 0.0, // Default value
				MaxRetries:     5,
				Timeout:        60 * time.Second,
				EnableTracing:  false,
				EnableMetrics:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &RetrieverOptions{
				DefaultK:       4,
				ScoreThreshold: 0.0,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				EnableTracing:  true,
				EnableMetrics:  true,
			}

			for _, option := range tt.options {
				option(opts)
			}

			if opts.DefaultK != tt.expected.DefaultK {
				t.Errorf("DefaultK = %v, want %v", opts.DefaultK, tt.expected.DefaultK)
			}
			if opts.ScoreThreshold != tt.expected.ScoreThreshold {
				t.Errorf("ScoreThreshold = %v, want %v", opts.ScoreThreshold, tt.expected.ScoreThreshold)
			}
			if opts.MaxRetries != tt.expected.MaxRetries {
				t.Errorf("MaxRetries = %v, want %v", opts.MaxRetries, tt.expected.MaxRetries)
			}
			if opts.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout = %v, want %v", opts.Timeout, tt.expected.Timeout)
			}
			if opts.EnableTracing != tt.expected.EnableTracing {
				t.Errorf("EnableTracing = %v, want %v", opts.EnableTracing, tt.expected.EnableTracing)
			}
			if opts.EnableMetrics != tt.expected.EnableMetrics {
				t.Errorf("EnableMetrics = %v, want %v", opts.EnableMetrics, tt.expected.EnableMetrics)
			}
		})
	}
}

// TestMockRetriever is removed due to import cycle issues.
// MockRetriever tests are located in the providers package.

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				EnableTracing:  true,
				EnableMetrics:  true,
			},
			wantErr: false,
		},
		{
			name: "invalid DefaultK too low",
			config: Config{
				DefaultK:       0,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid DefaultK too high",
			config: Config{
				DefaultK:       101,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid ScoreThreshold too low",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: -0.1,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid ScoreThreshold too high",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: 1.1,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid Timeout too short",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        500 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "invalid Timeout too long",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        6 * time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustomErrorTypes(t *testing.T) {
	t.Run("RetrieverError", func(t *testing.T) {
		originalErr := NewRetrieverError("test operation", nil, ErrCodeInvalidInput)
		if originalErr.Op != "test operation" {
			t.Errorf("RetrieverError.Op = %v, want 'test operation'", originalErr.Op)
		}
		if originalErr.Code != ErrCodeInvalidInput {
			t.Errorf("RetrieverError.Code = %v, want %v", originalErr.Code, ErrCodeInvalidInput)
		}
	})

	t.Run("RetrieverErrorWithMessage", func(t *testing.T) {
		customMsg := "custom error message"
		err := NewRetrieverErrorWithMessage("test operation", nil, ErrCodeInvalidInput, customMsg)
		if err.Message != customMsg {
			t.Errorf("RetrieverError.Message = %v, want %v", err.Message, customMsg)
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		field := "TestField"
		value := "invalid_value"
		msg := "must be valid"
		err := &ValidationError{
			Field: field,
			Value: value,
			Msg:   msg,
		}

		expectedMsg := "validation failed for field 'TestField' with value 'invalid_value': must be valid"
		if err.Error() != expectedMsg {
			t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expectedMsg)
		}
	})

	t.Run("TimeoutError", func(t *testing.T) {
		timeout := 30 * time.Second
		underlyingErr := NewRetrieverError("underlying", nil, ErrCodeTimeout)
		err := NewTimeoutError("test operation", timeout, underlyingErr)

		if err.Op != "test operation" {
			t.Errorf("TimeoutError.Op = %v, want 'test operation'", err.Op)
		}
		if err.Timeout != timeout {
			t.Errorf("TimeoutError.Timeout = %v, want %v", err.Timeout, timeout)
		}
	})
}

func TestMetrics(t *testing.T) {
	// Create a no-op meter for testing
	meter := noop.NewMeterProvider().Meter("test")

	metrics, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("NewMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Error("NewMetrics() returned nil")
	}

	// Test recording retrieval metrics
	ctx := context.Background()
	duration := 100 * time.Millisecond

	metrics.RecordRetrieval(ctx, "test_retriever", duration, 5, 0.8, nil)

	// Test recording vector store metrics
	metrics.RecordVectorStoreOperation(ctx, "similarity_search", duration, 5, nil)

	// Test recording batch metrics
	metrics.RecordBatchOperation(ctx, "test_operation", 10, duration)
}

func TestGetRetrieverTypes(t *testing.T) {
	types := GetRetrieverTypes()

	if len(types) == 0 {
		t.Error("GetRetrieverTypes() returned empty slice")
	}

	// Check that "vector_store" is included
	found := false
	for _, typ := range types {
		if typ == "vector_store" {
			found = true
			break
		}
	}

	if !found {
		t.Error("GetRetrieverTypes() should include 'vector_store'")
	}
}

func TestValidateRetrieverConfig(t *testing.T) {
	validConfig := DefaultConfig()
	err := ValidateRetrieverConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateRetrieverConfig() with valid config returned error: %v", err)
	}

	invalidConfig := DefaultConfig()
	invalidConfig.DefaultK = 0 // Invalid value
	err = ValidateRetrieverConfig(invalidConfig)
	if err == nil {
		t.Error("ValidateRetrieverConfig() with invalid config should return error")
	}
}

// Benchmark is removed due to import cycle issues.
// Benchmarks are located in the providers package.
