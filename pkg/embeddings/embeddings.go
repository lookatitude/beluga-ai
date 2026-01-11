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
	"errors"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// EmbedderFactory provides factory methods for creating embedder instances.
type EmbedderFactory struct {
	config  *Config
	metrics *Metrics
	tracer  trace.Tracer
	options *optionConfig
}

// NewEmbedderFactory creates a new embedder factory with the given configuration.
func NewEmbedderFactory(config *Config, opts ...Option) (*EmbedderFactory, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
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
	metrics := NewMetrics(meter)

	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")

	factory := &EmbedderFactory{
		config:  config,
		metrics: metrics,
		tracer:  tracer,
	}

	// Store options for later use
	factory.options = optionCfg

	return factory, nil
}

// NewEmbedder creates an embedder instance based on the provider type.
// Uses the registry to avoid import cycles.
func (f *EmbedderFactory) NewEmbedder(providerType string) (iface.Embedder, error) {
	// Use registry to create embedder to avoid import cycles
	// Provider init() functions will register themselves when imported elsewhere
	ctx := context.Background()
	return GetRegistry().Create(ctx, providerType, *f.config)
}

// newMockEmbedder creates a mock embedder instance using the registry.
func (f *EmbedderFactory) newMockEmbedder() (iface.Embedder, error) {
	if f.config.Mock == nil || !f.config.Mock.Enabled {
		return nil, errors.New("mock provider is not configured or disabled")
	}

	if err := f.config.Mock.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Mock configuration: %w", err)
	}

	// Use registry to create mock embedder to avoid import cycle
	// The mock provider's init() function will register itself when imported elsewhere
	ctx := context.Background()
	return GetRegistry().Create(ctx, "mock", *f.config)
}

// GetAvailableProviders returns a list of available provider types.
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

// HealthChecker interface for embedder health checks.
type HealthChecker interface {
	Check(ctx context.Context) error
}

// CheckHealth performs a health check on the embedder.
func (f *EmbedderFactory) CheckHealth(ctx context.Context, providerType string) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
	ctx, span := tracer.Start(ctx, "embeddings.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider_type", providerType),
		))
	defer span.End()

	embedder, err := f.NewEmbedder(providerType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to create embedder for health check", "error", err, "provider_type", providerType)
		return fmt.Errorf("failed to create embedder for health check: %w", err)
	}

	if checker, ok := embedder.(HealthChecker); ok {
		err = checker.Check(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logWithOTELContext(ctx, slog.LevelError, "Health check failed", "error", err, "provider_type", providerType)
			return err
		}
		span.SetStatus(codes.Ok, "")
		logWithOTELContext(ctx, slog.LevelInfo, "Health check passed", "provider_type", providerType)
		return nil
	}

	// Default health check: try to get dimension
	_, err = embedder.GetDimension(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Health check failed (dimension check)", "error", err, "provider_type", providerType)
		return err
	}
	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Health check passed (dimension check)", "provider_type", providerType)
	return nil
}

// logWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}
