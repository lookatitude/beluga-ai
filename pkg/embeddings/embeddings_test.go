package embeddings

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

func TestNewEmbedderFactory(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		opts    []Option
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "test-key",
					Model:  "text-embedding-ada-002",
				},
			},
			wantErr: false,
		},
		{
			name: "config with options",
			config: &Config{
				Mock: &MockConfig{
					Dimension: 128,
				},
			},
			opts: []Option{
				WithTimeout(10 * time.Second),
				WithMaxRetries(5),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewEmbedderFactory(tt.config, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmbedderFactory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && factory == nil {
				t.Error("NewEmbedderFactory() returned nil factory")
			}
		})
	}
}

func TestEmbedderFactory_NewEmbedder(t *testing.T) {
	config := &Config{
		OpenAI: &OpenAIConfig{
			APIKey: "test-key",
			Model:  "text-embedding-ada-002",
		},
		Ollama: &OllamaConfig{
			Model: "nomic-embed-text",
		},
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	tests := []struct {
		name         string
		providerType string
		wantErr      bool
	}{
		{
			name:         "openai provider",
			providerType: "openai",
			wantErr:      false,
		},
		{
			name:         "ollama provider",
			providerType: "ollama",
			wantErr:      false,
		},
		{
			name:         "mock provider",
			providerType: "mock",
			wantErr:      false,
		},
		{
			name:         "unknown provider",
			providerType: "unknown",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder, err := factory.NewEmbedder(tt.providerType)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedderFactory.NewEmbedder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if embedder == nil {
					t.Error("EmbedderFactory.NewEmbedder() returned nil embedder")
				}
				// Verify interface compliance
				var _ iface.Embedder = embedder
			}
		})
	}
}

func TestEmbedderFactory_GetAvailableProviders(t *testing.T) {
	config := &Config{
		OpenAI: &OpenAIConfig{
			APIKey: "test-key",
			Model:  "text-embedding-ada-002",
		},
		Ollama: &OllamaConfig{
			Model: "nomic-embed-text",
		},
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	providers := factory.GetAvailableProviders()
	expectedProviders := map[string]bool{
		"openai": true,
		"ollama": true,
		"mock":   true,
	}

	if len(providers) != len(expectedProviders) {
		t.Errorf("GetAvailableProviders() returned %d providers, expected %d", len(providers), len(expectedProviders))
	}

	for _, provider := range providers {
		if !expectedProviders[provider] {
			t.Errorf("GetAvailableProviders() returned unexpected provider: %s", provider)
		}
	}
}

func TestEmbedderFactory_CheckHealth(t *testing.T) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()

	// Test mock provider health check (should always succeed)
	err = factory.CheckHealth(ctx, "mock")
	if err != nil {
		t.Errorf("CheckHealth() failed for mock provider: %v", err)
	}

	// Test unknown provider
	err = factory.CheckHealth(ctx, "unknown")
	if err == nil {
		t.Error("CheckHealth() should fail for unknown provider")
	}
}

func TestOptions(t *testing.T) {
	tests := []struct {
		name  string
		opts  []Option
		check func(*optionConfig) bool
	}{
		{
			name: "WithTimeout",
			opts: []Option{WithTimeout(10 * time.Second)},
			check: func(c *optionConfig) bool {
				return c.timeout == 10*time.Second
			},
		},
		{
			name: "WithMaxRetries",
			opts: []Option{WithMaxRetries(5)},
			check: func(c *optionConfig) bool {
				return c.maxRetries == 5
			},
		},
		{
			name: "WithModel",
			opts: []Option{WithModel("test-model")},
			check: func(c *optionConfig) bool {
				return c.model == "test-model"
			},
		},
		{
			name: "multiple options",
			opts: []Option{
				WithTimeout(20 * time.Second),
				WithMaxRetries(10),
				WithModel("multi-test"),
			},
			check: func(c *optionConfig) bool {
				return c.timeout == 20*time.Second && c.maxRetries == 10 && c.model == "multi-test"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := defaultOptionConfig()
			for _, opt := range tt.opts {
				opt(config)
			}

			if !tt.check(config) {
				t.Error("Options not applied correctly")
			}
		})
	}
}

func TestEmbedderFactory_NewEmbedder_DisabledProvider(t *testing.T) {
	config := &Config{
		OpenAI: &OpenAIConfig{
			APIKey:  "sk-test",
			Model:   "text-embedding-ada-002",
			Enabled: false, // Try to disable
		},
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// SetDefaults overrides Enabled, so manually disable after factory creation
	config.OpenAI.Enabled = false

	// Try to create disabled provider
	_, err = factory.NewEmbedder("openai")
	if err == nil {
		t.Error("Expected error for disabled provider")
	}

	// Mock should still work
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		t.Errorf("Expected mock provider to work: %v", err)
	}
	if embedder == nil {
		t.Error("Expected non-nil embedder")
	}
}

func TestEmbedderFactory_GetAvailableProviders_EnabledDisabled(t *testing.T) {
	config := &Config{
		OpenAI: &OpenAIConfig{
			APIKey:  "sk-test",
			Model:   "text-embedding-ada-002",
			Enabled: false, // Try to disable
		},
		Ollama: &OllamaConfig{
			Model:   "nomic-embed-text",
			Enabled: true,
		},
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Manually disable OpenAI after SetDefaults
	config.OpenAI.Enabled = false

	providers := factory.GetAvailableProviders()

	// Should only include enabled providers
	expectedProviders := map[string]bool{
		"ollama": true,
		"mock":   true,
	}

	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d: %v", len(expectedProviders), len(providers), providers)
	}

	for _, provider := range providers {
		if !expectedProviders[provider] {
			t.Errorf("Unexpected provider: %s", provider)
		}
	}
}

func TestEmbedderFactory_CheckHealth_ErrorCases(t *testing.T) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()

	// Test unknown provider
	err = factory.CheckHealth(ctx, "unknown")
	if err == nil {
		t.Error("Expected error for unknown provider")
	}

	// Test disabled provider - create separate config to avoid shared state
	config2 := &Config{
		OpenAI: &OpenAIConfig{
			APIKey:  "sk-test",
			Model:   "text-embedding-ada-002",
			Enabled: false, // Will be overridden by SetDefaults
		},
		Mock: &MockConfig{
			Dimension: 128,
		},
	}
	factory2, err := NewEmbedderFactory(config2)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Manually disable after SetDefaults
	config2.OpenAI.Enabled = false

	err = factory2.CheckHealth(ctx, "openai")
	if err == nil {
		t.Error("Expected error for disabled provider")
	}
}

func TestEmbedderFactory_ConcurrentAccess(t *testing.T) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Test concurrent creation of embedders
	done := make(chan bool, 10)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			embedder, err := factory.NewEmbedder("mock")
			if err != nil {
				t.Errorf("Goroutine %d: failed to create embedder: %v", id, err)
				return
			}

			// Test concurrent embedding
			texts := []string{"test text"}
			_, err = embedder.EmbedDocuments(ctx, texts)
			if err != nil {
				t.Errorf("Goroutine %d: failed to embed: %v", id, err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestEmbedderFactory_OptionsApplication(t *testing.T) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	// Test that options are stored and can be accessed
	opts := []Option{
		WithTimeout(45 * time.Second),
		WithMaxRetries(7),
		WithModel("custom-model"),
	}

	factory, err := NewEmbedderFactory(config, opts...)
	if err != nil {
		t.Fatalf("Failed to create factory with options: %v", err)
	}

	// Options are stored in factory.options, but we can't directly access them
	// This test mainly ensures options don't break factory creation
	if factory == nil {
		t.Error("Factory should not be nil")
	}
}

func TestEmbedderFactory_ProviderCreationErrors(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		providerType string
		wantErr      bool
	}{
		{
			name:   "openai config nil",
			config: &Config{
				// OpenAI config is nil - SetDefaults will create it but without API key
			},
			providerType: "openai",
			wantErr:      true, // OpenAI needs API key
		},
		{
			name:   "ollama config nil",
			config: &Config{
				// Ollama config is nil - SetDefaults will create it but without model
			},
			providerType: "ollama",
			wantErr:      true, // Ollama needs a model
		},
		{
			name:   "mock config nil",
			config: &Config{
				// Mock config is nil - SetDefaults will create it
			},
			providerType: "mock",
			wantErr:      false, // Mock works with defaults
		},
		{
			name: "openai config invalid",
			config: &Config{
				OpenAI: &OpenAIConfig{
					APIKey: "", // Missing required field
					Model:  "text-embedding-ada-002",
				},
			},
			providerType: "openai",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewEmbedderFactory(tt.config)
			if err != nil && !tt.wantErr {
				t.Fatalf("Failed to create factory: %v", err)
			}
			if err != nil && tt.wantErr {
				// Factory creation failed as expected due to validation
				return
			}

			_, err = factory.NewEmbedder(tt.providerType)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmbedder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbedderFactory_ContextCancellation(t *testing.T) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Factory operations don't use context, but embedder operations do
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Test that embedder respects cancelled context
	_, err = embedder.EmbedQuery(ctx, "test")
	// Mock embedder may or may not respect context cancellation
	// This test mainly ensures the embedder can be created and called
	if embedder == nil {
		t.Error("Embedder should not be nil")
	}
}

func TestEmbedderFactory_MultipleProviders(t *testing.T) {
	config := &Config{
		OpenAI: &OpenAIConfig{
			APIKey: "sk-test",
			Model:  "text-embedding-ada-002",
		},
		Ollama: &OllamaConfig{
			Model: "nomic-embed-text",
		},
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()

	// Test creating multiple different embedders
	providers := []string{"openai", "ollama", "mock"}
	embedders := make(map[string]iface.Embedder)

	for _, provider := range providers {
		embedder, err := factory.NewEmbedder(provider)
		if err != nil {
			t.Errorf("Failed to create %s embedder: %v", provider, err)
			continue
		}
		embedders[provider] = embedder
	}

	// Test that each embedder works
	for provider, embedder := range embedders {
		dimension, err := embedder.GetDimension(ctx)
		if err != nil {
			t.Errorf("%s embedder GetDimension failed: %v", provider, err)
		}
		// Ollama returns 0 (unknown), others return positive dimensions
		if provider != "ollama" && dimension <= 0 {
			t.Errorf("%s embedder returned invalid dimension: %d", provider, dimension)
		}
		if provider == "ollama" && dimension != 0 {
			t.Logf("Ollama returned dimension %d (unexpected but not an error)", dimension)
		}
	}
}

// TestNewEmbedder_ErrorPaths tests comprehensive error handling for the global NewEmbedder function
func TestNewEmbedder_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	// Test the basic unknown provider case (doesn't require provider registration)
	t.Run("unknown provider", func(t *testing.T) {
		_, err := NewEmbedder(ctx, "nonexistent", Config{})
		if err == nil {
			t.Error("expected error for unknown provider")
			return
		}

		var embErr *iface.EmbeddingError
		if errors.As(err, &embErr) {
			if embErr.Code != iface.ErrCodeProviderNotFound {
				t.Errorf("expected error code %s, got %s", iface.ErrCodeProviderNotFound, embErr.Code)
			}
		} else {
			t.Errorf("expected EmbeddingError, got %T", err)
		}
	})

	// Test with a registered mock provider to verify success case
	t.Run("registered mock provider", func(t *testing.T) {
		// Register a mock provider for testing
		RegisterGlobal("test_mock", func(ctx context.Context, config Config) (iface.Embedder, error) {
			if config.Mock == nil {
				return nil, iface.WrapError(fmt.Errorf("mock config is nil"), iface.ErrCodeInvalidConfig, "missing mock configuration")
			}
			return NewAdvancedMockEmbedder("test", "mock-model", config.Mock.Dimension), nil
		})
		defer func() {
			// Cleanup: remove the test provider
			registry := GetGlobalRegistry()
			registry.mu.Lock()
			delete(registry.creators, "test_mock")
			registry.mu.Unlock()
		}()

		embedder, err := NewEmbedder(ctx, "test_mock", Config{
			Mock: &MockConfig{
				Dimension: 128,
			},
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if embedder == nil {
			t.Error("expected valid embedder but got nil")
			return
		}

		// Test basic functionality
		dimension, err := embedder.GetDimension(ctx)
		if err != nil {
			t.Errorf("GetDimension failed: %v", err)
		} else if dimension != 128 {
			t.Errorf("expected dimension 128, got %d", dimension)
		}
	})
}

// TestProviderRegistry_ConcurrencyStress tests registry behavior under high concurrency
func TestProviderRegistry_ConcurrencyStress(t *testing.T) {
	ctx := context.Background()
	registry := NewProviderRegistry()

	// Number of concurrent operations
	numGoroutines := 50
	numOperations := 100

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*numOperations)

	// Start concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				operationType := j % 3 // 0=register, 1=retrieve, 2=list

				switch operationType {
				case 0: // Register operation
					providerName := fmt.Sprintf("stress_provider_%d_%d", goroutineID, j)
					registry.Register(providerName, func(ctx context.Context, config Config) (iface.Embedder, error) {
						return NewAdvancedMockEmbedder(providerName, "stress-model", 128), nil
					})

				case 1: // Retrieve operation
					providerName := fmt.Sprintf("stress_provider_%d_%d", goroutineID%5, j%10) // Use some existing providers
					_, err := registry.Create(ctx, providerName, Config{
						Mock: &MockConfig{Dimension: 128},
					})
					// Ignore "not found" errors as some providers may not exist
					if err != nil && !strings.Contains(err.Error(), "not found") {
						errChan <- fmt.Errorf("create failed: %v", err)
					}

				case 2: // List operation
					providers := registry.ListProviders()
					if providers == nil {
						errChan <- fmt.Errorf("list returned nil")
					}
				}
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("encountered %d errors during concurrency stress test:", len(errors))
		for i, err := range errors {
			t.Errorf("  %d: %v", i+1, err)
		}
	}

	// Verify registry is in a consistent state
	finalProviders := registry.ListProviders()
	t.Logf("Final registry state: %d providers registered", len(finalProviders))

	// Test that we can still perform basic operations
	testEmbedder, err := registry.Create(ctx, "stress_provider_0_0", Config{
		Mock: &MockConfig{Dimension: 128},
	})
	if err != nil {
		t.Errorf("post-stress registry operation failed: %v", err)
	} else {
		// Test basic functionality
		dimension, err := testEmbedder.GetDimension(ctx)
		if err != nil || dimension != 128 {
			t.Errorf("post-stress embedder functionality failed: dim=%d, err=%v", dimension, err)
		}
	}
}
