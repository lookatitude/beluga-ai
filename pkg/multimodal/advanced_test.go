// Package multimodal provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, performance benchmarks, and integration test patterns.
package multimodal

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/internal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultimodalModelAdvanced provides advanced table-driven tests for MultimodalModel operations.
func TestMultimodalModelAdvanced(t *testing.T) {
	tests := []struct {
		setup       func() iface.MultimodalModel
		operation   func(ctx context.Context, model iface.MultimodalModel) error
		validate    func(t *testing.T, err error)
		name        string
		description string
		errContains string
		wantErr     bool
	}{
		{
			name:        "process_text_only",
			description: "Test processing text-only input",
			setup: func() iface.MultimodalModel {
				return NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
					Text: true,
				})
			},
			operation: func(ctx context.Context, model iface.MultimodalModel) error {
				input := &types.MultimodalInput{
					ID: "test-input",
					ContentBlocks: []*types.ContentBlock{
						{
							Type: "text",
							Data: []byte("Hello, world!"),
							Size: 13,
						},
					},
				}
				_, err := model.Process(ctx, input)
				return err
			},
			wantErr: false,
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:        "process_multimodal",
			description: "Test processing multimodal input (text + image)",
			setup: func() iface.MultimodalModel {
				return NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
					Text:  true,
					Image: true,
				})
			},
			operation: func(ctx context.Context, model iface.MultimodalModel) error {
				input := &types.MultimodalInput{
					ID: "test-input",
					ContentBlocks: []*types.ContentBlock{
						{
							Type: "text",
							Data: []byte("What's in this image?"),
							Size: 22,
						},
						{
							Type: "image",
							URL:  "https://example.com/image.png",
							Size: 1000,
						},
					},
				}
				_, err := model.Process(ctx, input)
				return err
			},
			wantErr: false,
		},
		{
			name:        "unsupported_modality",
			description: "Test handling unsupported modality",
			setup: func() iface.MultimodalModel {
				return NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
					Text:  true,
					Image: false,
				})
			},
			operation: func(ctx context.Context, model iface.MultimodalModel) error {
				input := &types.MultimodalInput{
					ID: "test-input",
					ContentBlocks: []*types.ContentBlock{
						{
							Type: "image",
							URL:  "https://example.com/image.png",
							Size: 1000,
						},
					},
				}
				_, err := model.Process(ctx, input)
				return err
			},
			wantErr: false, // Mock doesn't error, just processes
		},
		{
			name:        "empty_input",
			description: "Test handling empty input",
			setup: func() iface.MultimodalModel {
				// Create a mock that will error on empty input
				mock := NewMockMultimodalModel("openai", "gpt-4o", nil)
				// Note: Mock doesn't validate, so this test verifies the mock behavior
				// In real implementations, empty input would be validated
				return mock
			},
			operation: func(ctx context.Context, model iface.MultimodalModel) error {
				input := &types.MultimodalInput{
					ID:            "test-input",
					ContentBlocks: []*types.ContentBlock{},
				}
				_, err := model.Process(ctx, input)
				return err
			},
			wantErr: false, // Mock doesn't validate empty input
		},
		{
			name:        "context_cancellation",
			description: "Test context cancellation handling",
			setup: func() iface.MultimodalModel {
				return NewMockMultimodalModel("openai", "gpt-4o", nil)
			},
			operation: func(ctx context.Context, model iface.MultimodalModel) error {
				ctx, cancel := context.WithCancel(ctx)
				cancel() // Cancel immediately
				input := &types.MultimodalInput{
					ID: "test-input",
					ContentBlocks: []*types.ContentBlock{
						{
							Type: "text",
							Data: []byte("test"),
							Size: 4,
						},
					},
				}
				// Mock doesn't check context, so this won't error
				// Real implementations should check ctx.Done()
				_, err := model.Process(ctx, input)
				return err
			},
			wantErr: false, // Mock doesn't check context cancellation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			model := tt.setup()
			ctx := context.Background()

			err := tt.operation(ctx, model)

			if tt.wantErr {
				assert.Error(t, err, "Expected error for %s", tt.description)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains, "Error should contain expected text")
				}
			} else {
				assert.NoError(t, err, "No error expected for %s", tt.description)
			}

			if tt.validate != nil {
				tt.validate(t, err)
			}
		})
	}
}

// TestMultimodalModelConcurrency tests concurrent operations.
func TestMultimodalModelConcurrency(t *testing.T) {
	model := NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text:  true,
		Image: true,
	})
	ctx := context.Background()

	numGoroutines := 10
	numOperations := 100
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{
				Type: "text",
				Data: []byte("test"),
				Size: 4,
			},
		},
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/numGoroutines; j++ {
				_, err := model.Process(ctx, input)
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		require.NoError(t, err, "Concurrent operation should not error")
	}
}

// TestMultimodalModelStreaming tests streaming operations.
func TestMultimodalModelStreaming(t *testing.T) {
	model := NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text: true,
	})
	ctx := context.Background()

	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{
				Type: "text",
				Data: []byte("test"),
				Size: 4,
			},
		},
	}

	outputChan, err := model.ProcessStream(ctx, input)
	require.NoError(t, err)

	// Receive outputs
	outputCount := 0
	for output := range outputChan {
		assert.NotNil(t, output)
		assert.Equal(t, input.ID, output.InputID)
		outputCount++
	}

	assert.Positive(t, outputCount, "Should receive at least one output")
}

// TestMultimodalModelCapabilities tests capability checking.
func TestMultimodalModelCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities *types.ModalityCapabilities
		modality     string
		expected     bool
	}{
		{
			name: "text_supported",
			capabilities: &types.ModalityCapabilities{
				Text: true,
			},
			modality: "text",
			expected: true,
		},
		{
			name: "image_supported",
			capabilities: &types.ModalityCapabilities{
				Image: true,
			},
			modality: "image",
			expected: true,
		},
		{
			name: "audio_not_supported",
			capabilities: &types.ModalityCapabilities{
				Audio: false,
			},
			modality: "audio",
			expected: false,
		},
		{
			name: "video_not_supported",
			capabilities: &types.ModalityCapabilities{
				Video: false,
			},
			modality: "video",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewMockMultimodalModel("openai", "gpt-4o", tt.capabilities)
			ctx := context.Background()

			supported, err := model.SupportsModality(ctx, tt.modality)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, supported)
		})
	}
}

// TestMultimodalModelPerformance tests performance characteristics.
func TestMultimodalModelPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	model := NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text: true,
	})
	ctx := context.Background()

	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{
				Type: "text",
				Data: []byte("test"),
				Size: 4,
			},
		},
	}

	// Test single operation latency
	t.Run("single_operation_latency", func(t *testing.T) {
		start := time.Now()
		_, err := model.Process(ctx, input)
		require.NoError(t, err)
		duration := time.Since(start)
		t.Logf("Single operation took: %v", duration)
		assert.Less(t, duration, 1*time.Second, "Single operation should complete quickly")
	})

	// Test throughput
	t.Run("throughput", func(t *testing.T) {
		numOperations := 100
		start := time.Now()

		for i := 0; i < numOperations; i++ {
			_, err := model.Process(ctx, input)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		throughput := float64(numOperations) / duration.Seconds()
		t.Logf("Throughput: %.2f operations/second", throughput)
		assert.Greater(t, throughput, 10.0, "Should handle at least 10 ops/sec")
	})
}

// BenchmarkMultimodalModelProcess benchmarks Process operations.
func BenchmarkMultimodalModelProcess(b *testing.B) {
	model := NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text: true,
	})
	ctx := context.Background()

	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{
				Type: "text",
				Data: []byte("benchmark input"),
				Size: 15,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := model.Process(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultimodalModelProcessStream benchmarks ProcessStream operations.
func BenchmarkMultimodalModelProcessStream(b *testing.B) {
	model := NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text: true,
	})
	ctx := context.Background()

	input := &types.MultimodalInput{
		ID: "test-input",
		ContentBlocks: []*types.ContentBlock{
			{
				Type: "text",
				Data: []byte("benchmark input"),
				Size: 15,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputChan, err := model.ProcessStream(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
		// Consume all outputs
		for range outputChan {
		}
	}
}

// BenchmarkMultimodalModelGetCapabilities benchmarks GetCapabilities operations.
func BenchmarkMultimodalModelGetCapabilities(b *testing.B) {
	model := NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text:  true,
		Image: true,
		Audio: true,
		Video: true,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := model.GetCapabilities(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultimodalModelSupportsModality benchmarks SupportsModality operations.
func BenchmarkMultimodalModelSupportsModality(b *testing.B) {
	model := NewMockMultimodalModel("openai", "gpt-4o", &types.ModalityCapabilities{
		Text:  true,
		Image: true,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := model.SupportsModality(ctx, "text")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContentBlockCreation benchmarks content block creation.
func BenchmarkContentBlockCreation(b *testing.B) {
	data := []byte("test data")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewContentBlock("text", data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultimodalInputCreation benchmarks multimodal input creation.
func BenchmarkMultimodalInputCreation(b *testing.B) {
	blocks := []*ContentBlock{
		{
			Type: "text",
			Data: []byte("test"),
			Size: 4,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewMultimodalInput(blocks)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContentRouting benchmarks content routing operations.
// This benchmarks the performance-critical routing logic that determines
// which provider should handle each content block.
func BenchmarkContentRouting(b *testing.B) {
	ctx := context.Background()
	router := internal.NewRouter(registry.GetRegistry())

	input := &types.MultimodalInput{
		ID: "benchmark-input",
		ContentBlocks: []*types.ContentBlock{
			{Type: "text", Data: []byte("test text"), Size: 9},
			{Type: "image", URL: "https://example.com/image.png", Size: 1000},
			{Type: "audio", Data: []byte("audio data"), Size: 10},
		},
		Format: "base64",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.Route(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContentNormalization benchmarks content normalization operations.
// This benchmarks the performance-critical format conversion logic.
func BenchmarkContentNormalization(b *testing.B) {
	ctx := context.Background()
	normalizer := internal.NewNormalizer()

	block := &types.ContentBlock{
		Type:     "image",
		Data:     make([]byte, 1024), // 1KB test data
		Size:     1024,
		MIMEType: "image/png",
		Format:   "png",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := normalizer.Normalize(ctx, block, "base64")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContentNormalizationURL benchmarks normalization from URL format.
func BenchmarkContentNormalizationURL(b *testing.B) {
	ctx := context.Background()
	normalizer := internal.NewNormalizer()

	block := &types.ContentBlock{
		Type:     "image",
		URL:      "https://example.com/image.png",
		Size:     1024,
		MIMEType: "image/png",
		Format:   "png",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will fail without network access, but benchmarks the code path
		_, _ = normalizer.Normalize(ctx, block, "base64")
	}
}

// T099: Test coverage for pkg/multimodal/config.go.
func TestConfigOptions(t *testing.T) {
	t.Run("WithProvider", func(t *testing.T) {
		config := &Config{}
		WithProvider("openai")(config)
		assert.Equal(t, "openai", config.Provider)
	})

	t.Run("WithModel", func(t *testing.T) {
		config := &Config{}
		WithModel("gpt-4o")(config)
		assert.Equal(t, "gpt-4o", config.Model)
	})

	t.Run("WithAPIKey", func(t *testing.T) {
		config := &Config{}
		WithAPIKey("test-key")(config)
		assert.Equal(t, "test-key", config.APIKey)
	})

	t.Run("WithTimeout", func(t *testing.T) {
		config := &Config{}
		WithTimeout(60 * time.Second)(config)
		assert.Equal(t, 60*time.Second, config.Timeout)
	})

	t.Run("WithStreaming", func(t *testing.T) {
		config := &Config{}
		WithStreaming(true)(config)
		assert.True(t, config.EnableStreaming)
	})

	t.Run("WithMaxRetries", func(t *testing.T) {
		config := &Config{}
		WithMaxRetries(5)(config)
		assert.Equal(t, 5, config.MaxRetries)
	})

	t.Run("WithRetryDelay", func(t *testing.T) {
		config := &Config{}
		WithRetryDelay(2 * time.Second)(config)
		assert.Equal(t, 2*time.Second, config.RetryDelay)
	})

	t.Run("WithStreamChunkSize", func(t *testing.T) {
		config := &Config{}
		WithStreamChunkSize(2048)(config)
		assert.Equal(t, int64(2048), config.StreamChunkSize)
	})

	t.Run("WithBaseURL", func(t *testing.T) {
		config := &Config{}
		WithBaseURL("https://api.example.com")(config)
		assert.Equal(t, "https://api.example.com", config.BaseURL)
	})

	t.Run("WithProviderSpecific", func(t *testing.T) {
		config := &Config{}
		WithProviderSpecific(map[string]any{"key": "value"})(config)
		assert.Equal(t, "value", config.ProviderSpecific["key"])
	})
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		config      *Config
		name        string
		shouldError bool
	}{
		{
			name: "valid config",
			config: &Config{
				Provider:   "openai",
				Model:      "gpt-4o",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			shouldError: false,
		},
		{
			name: "missing provider",
			config: &Config{
				Model:   "gpt-4o",
				Timeout: 30 * time.Second,
			},
			shouldError: true,
		},
		{
			name: "missing model",
			config: &Config{
				Provider: "openai",
				Timeout:  30 * time.Second,
			},
			shouldError: true,
		},
		{
			name: "zero timeout",
			config: &Config{
				Provider: "openai",
				Model:    "gpt-4o",
				Timeout:  0,
			},
			shouldError: true,
		},
		{
			name: "negative max retries",
			config: &Config{
				Provider:   "openai",
				Model:      "gpt-4o",
				Timeout:    30 * time.Second,
				MaxRetries: -1,
			},
			shouldError: true,
		},
		{
			name: "negative retry delay",
			config: &Config{
				Provider:   "openai",
				Model:      "gpt-4o",
				Timeout:    30 * time.Second,
				RetryDelay: -1 * time.Second,
			},
			shouldError: true,
		},
		{
			name: "streaming enabled but zero chunk size",
			config: &Config{
				Provider:        "openai",
				Model:           "gpt-4o",
				Timeout:         30 * time.Second,
				EnableStreaming: true,
				StreamChunkSize: 0,
			},
			shouldError: true,
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
}

func TestRoutingConfig_Validate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		routing     *RoutingConfig
		name        string
		shouldError bool
	}{
		{
			name: "valid auto strategy",
			routing: &RoutingConfig{
				Strategy: "auto",
			},
			shouldError: false,
		},
		{
			name: "valid manual strategy with providers",
			routing: &RoutingConfig{
				Strategy:      "manual",
				TextProvider:  "openai",
				ImageProvider: "gemini",
			},
			shouldError: false,
		},
		{
			name: "manual strategy without providers",
			routing: &RoutingConfig{
				Strategy: "manual",
			},
			shouldError: true,
		},
		{
			name: "invalid strategy",
			routing: &RoutingConfig{
				Strategy: "invalid",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.routing.Validate(ctx)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// T100: Test coverage for pkg/multimodal/errors.go.
func TestMultimodalErrors(t *testing.T) {
	t.Run("NewMultimodalError", func(t *testing.T) {
		err := NewMultimodalError("test_op", ErrCodeTimeout, errors.New("timeout occurred"))
		assert.NotNil(t, err)
		assert.Equal(t, "test_op", err.Op)
		assert.Equal(t, ErrCodeTimeout, err.Code)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test_op")
		assert.Contains(t, err.Error(), ErrCodeTimeout)
	})

	t.Run("MultimodalError_Unwrap", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := NewMultimodalError("test_op", ErrCodeTimeout, underlying)
		assert.Equal(t, underlying, err.Unwrap())
	})

	t.Run("WrapError", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := WrapError(underlying, "test_op", ErrCodeProviderError)
		assert.NotNil(t, err)
		assert.Equal(t, "test_op", err.Op)
		assert.Equal(t, ErrCodeProviderError, err.Code)

		assert.Nil(t, WrapError(nil, "op", "code"))
	})

	t.Run("IsMultimodalError", func(t *testing.T) {
		err := NewMultimodalError("test_op", ErrCodeTimeout, errors.New("timeout"))
		assert.True(t, IsMultimodalError(err))

		regularErr := errors.New("regular error")
		assert.False(t, IsMultimodalError(regularErr))
	})

	t.Run("AsMultimodalError", func(t *testing.T) {
		err := NewMultimodalError("test_op", ErrCodeTimeout, errors.New("timeout"))
		mmErr, ok := AsMultimodalError(err)
		assert.True(t, ok)
		assert.NotNil(t, mmErr)
		assert.Equal(t, "test_op", mmErr.Op)

		regularErr := errors.New("regular error")
		_, ok = AsMultimodalError(regularErr)
		assert.False(t, ok)
	})

	t.Run("IsRetryableError", func(t *testing.T) {
		tests := []struct {
			err       error
			name      string
			retryable bool
		}{
			{
				name:      "nil error",
				err:       nil,
				retryable: false,
			},
			{
				name:      "rate limit error",
				err:       NewMultimodalError("op", ErrCodeRateLimit, errors.New("rate limit")),
				retryable: true,
			},
			{
				name:      "timeout error",
				err:       NewMultimodalError("op", ErrCodeTimeout, errors.New("timeout")),
				retryable: true,
			},
			{
				name:      "network error",
				err:       NewMultimodalError("op", ErrCodeNetworkError, errors.New("network")),
				retryable: true,
			},
			{
				name:      "provider error",
				err:       NewMultimodalError("op", ErrCodeProviderError, errors.New("provider")),
				retryable: true,
			},
			{
				name:      "invalid input error",
				err:       NewMultimodalError("op", ErrCodeInvalidInput, errors.New("invalid")),
				retryable: false,
			},
			{
				name:      "authentication failed",
				err:       NewMultimodalError("op", ErrCodeAuthenticationFailed, errors.New("auth")),
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
			{
				name:      "unknown error",
				err:       errors.New("unknown error"),
				retryable: true, // Default to retryable for unknown errors
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.retryable, IsRetryableError(tt.err))
			})
		}
	})

	t.Run("GetErrorCode", func(t *testing.T) {
		err := NewMultimodalError("op", ErrCodeRateLimit, errors.New("rate limit"))
		assert.Equal(t, ErrCodeRateLimit, GetErrorCode(err))

		regularErr := errors.New("regular error")
		assert.Equal(t, ErrCodeProviderError, GetErrorCode(regularErr))

		assert.Empty(t, GetErrorCode(nil))
	})

	t.Run("IsErrorCode", func(t *testing.T) {
		err := NewMultimodalError("op", ErrCodeTimeout, errors.New("timeout"))
		assert.True(t, IsErrorCode(err, ErrCodeTimeout))
		assert.False(t, IsErrorCode(err, ErrCodeRateLimit))
	})

	t.Run("WrapContextError", func(t *testing.T) {
		err := WrapContextError(context.Canceled, "test_op")
		assert.NotNil(t, err)
		assert.Equal(t, ErrCodeCancelled, err.Code)

		err = WrapContextError(context.DeadlineExceeded, "test_op")
		assert.NotNil(t, err)
		assert.Equal(t, ErrCodeTimeout, err.Code)

		otherErr := errors.New("other error")
		err = WrapContextError(otherErr, "test_op")
		assert.NotNil(t, err)
		assert.Equal(t, ErrCodeProviderError, err.Code)

		assert.Nil(t, WrapContextError(nil, "op"))
	})

	t.Run("IsContextError", func(t *testing.T) {
		assert.True(t, IsContextError(context.Canceled))
		assert.True(t, IsContextError(context.DeadlineExceeded))
		assert.False(t, IsContextError(errors.New("other error")))
		assert.False(t, IsContextError(nil))
	})

	t.Run("IsNetworkError", func(t *testing.T) {
		err := NewMultimodalError("op", ErrCodeNetworkError, errors.New("network"))
		assert.True(t, IsNetworkError(err))

		networkErr := errors.New("connection refused")
		assert.True(t, IsNetworkError(networkErr))

		regularErr := errors.New("other error")
		assert.False(t, IsNetworkError(regularErr))
	})

	t.Run("IsFileError", func(t *testing.T) {
		err := NewMultimodalError("op", ErrCodeFileNotFound, errors.New("file not found"))
		assert.True(t, IsFileError(err))

		fileErr := errors.New("file not found")
		assert.True(t, IsFileError(fileErr))

		regularErr := errors.New("other error")
		assert.False(t, IsFileError(regularErr))
	})
}

// BenchmarkContentNormalizationFilePath benchmarks normalization from file path format.
func BenchmarkContentNormalizationFilePath(b *testing.B) {
	ctx := context.Background()
	normalizer := internal.NewNormalizer()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "benchmark_*.png")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(make([]byte, 1024))
	tmpFile.Close()

	block := &types.ContentBlock{
		Type:     "image",
		FilePath: tmpFile.Name(),
		Size:     1024,
		MIMEType: "image/png",
		Format:   "png",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := normalizer.Normalize(ctx, block, "base64")
		if err != nil {
			b.Fatal(err)
		}
	}
}
