// Package embeddings provides interfaces and implementations for text embedding generation.
// This package follows the Beluga AI Framework design patterns with clean separation
// of interfaces, implementations, and configuration management.
//
// Key features:
// - Focused Embedder interface following Interface Segregation Principle (ISP)
// - Functional options pattern for flexible configuration
// - Comprehensive error handling with custom error types
// - OpenTelemetry tracing and metrics integration
// - Multiple providers: OpenAI, Ollama, and mock for testing
// - Extensive test coverage with table-driven tests and benchmarks
package embeddings

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/providers/ollama"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// EmbedderFactory provides factory methods for creating embedder instances
type EmbedderFactory struct {
	config  *Config
	metrics *Metrics
	tracer  trace.Tracer
	options *optionConfig
}

// NewEmbedderFactory creates a new embedder factory with the given configuration
func NewEmbedderFactory(config *Config, opts ...Option) (*EmbedderFactory, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	config.SetDefaults()

	// Apply functional options
	optionCfg := defaultOptionConfig()
	for _, opt := range opts {
		opt(optionCfg)
	}

	// Initialize metrics (assuming global meter is available)
	meter := otel.Meter("github.com/lookatitude/beluga-ai/pkg/embeddings")
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
	metrics, err := NewMetrics(meter, tracer)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	factory := &EmbedderFactory{
		config:  config,
		metrics: metrics,
		tracer:  tracer,
	}

	// Store options for later use
	factory.options = optionCfg

	return factory, nil
}

// NewEmbedder creates an embedder instance based on the provider type
func (f *EmbedderFactory) NewEmbedder(providerType string) (iface.Embedder, error) {
	switch providerType {
	case "openai":
		return f.newOpenAIEmbedder()
	case "ollama":
		return f.newOllamaEmbedder()
	case "mock":
		return f.newMockEmbedder()
	default:
		return nil, fmt.Errorf("unknown embedder provider: %s", providerType)
	}
}

// newOpenAIEmbedder creates an OpenAI embedder instance
func (f *EmbedderFactory) newOpenAIEmbedder() (iface.Embedder, error) {
	if f.config.OpenAI == nil || !f.config.OpenAI.Enabled {
		return nil, fmt.Errorf("OpenAI provider is not configured or disabled")
	}

	if err := f.config.OpenAI.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OpenAI configuration: %w", err)
	}

	openaiConfig := &openai.Config{
		APIKey:     f.config.OpenAI.APIKey,
		Model:      f.config.OpenAI.Model,
		BaseURL:    f.config.OpenAI.BaseURL,
		APIVersion: f.config.OpenAI.APIVersion,
		Timeout:    f.config.OpenAI.Timeout,
		MaxRetries: f.config.OpenAI.MaxRetries,
		Enabled:    f.config.OpenAI.Enabled,
	}

	return openai.NewOpenAIEmbedder(openaiConfig, f.tracer)
}

// newOllamaEmbedder creates an Ollama embedder instance
func (f *EmbedderFactory) newOllamaEmbedder() (iface.Embedder, error) {
	if f.config.Ollama == nil || !f.config.Ollama.Enabled {
		return nil, fmt.Errorf("ollama provider is not configured or disabled")
	}

	if err := f.config.Ollama.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Ollama configuration: %w", err)
	}

	ollamaConfig := &ollama.Config{
		ServerURL:  f.config.Ollama.ServerURL,
		Model:      f.config.Ollama.Model,
		Timeout:    f.config.Ollama.Timeout,
		MaxRetries: f.config.Ollama.MaxRetries,
		KeepAlive:  f.config.Ollama.KeepAlive,
		Enabled:    f.config.Ollama.Enabled,
	}

	return ollama.NewOllamaEmbedder(ollamaConfig, f.tracer)
}

// newMockEmbedder creates a mock embedder instance
func (f *EmbedderFactory) newMockEmbedder() (iface.Embedder, error) {
	if f.config.Mock == nil || !f.config.Mock.Enabled {
		return nil, fmt.Errorf("mock provider is not configured or disabled")
	}

	if err := f.config.Mock.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Mock configuration: %w", err)
	}

	mockConfig := &mock.Config{
		Dimension:    f.config.Mock.Dimension,
		Seed:         f.config.Mock.Seed,
		RandomizeNil: f.config.Mock.RandomizeNil,
		Enabled:      f.config.Mock.Enabled,
	}

	return mock.NewMockEmbedder(mockConfig, f.tracer)
}

// GetAvailableProviders returns a list of available provider types
func (f *EmbedderFactory) GetAvailableProviders() []string {
	providers := []string{}

	if f.config.OpenAI != nil && f.config.OpenAI.Enabled {
		providers = append(providers, "openai")
	}
	if f.config.Ollama != nil && f.config.Ollama.Enabled {
		providers = append(providers, "ollama")
	}
	if f.config.Mock != nil && f.config.Mock.Enabled {
		providers = append(providers, "mock")
	}

	return providers
}

// Health checks

// HealthChecker interface for embedder health checks
type HealthChecker interface {
	Check(ctx context.Context) error
}

// CheckHealth performs a health check on the embedder
func (f *EmbedderFactory) CheckHealth(ctx context.Context, providerType string) error {
	embedder, err := f.NewEmbedder(providerType)
	if err != nil {
		return fmt.Errorf("failed to create embedder for health check: %w", err)
	}

	if checker, ok := embedder.(HealthChecker); ok {
		return checker.Check(ctx)
	}

	// Default health check: try to get dimension
	_, err = embedder.GetDimension(ctx)
	return err
}
