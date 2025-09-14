package embeddings

import (
	"context"
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
		name string
		opts []Option
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

// Benchmark tests
func BenchmarkNewEmbedderFactory(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewEmbedderFactory(config)
	}
}

func BenchmarkEmbedderFactory_NewEmbedder(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.NewEmbedder("mock")
	}
}
