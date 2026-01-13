// Package s2s provides interfaces and implementations for Speech-to-Speech (S2S) operations.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package s2s

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance for S2S operations.
// This should be called once during application startup to enable metrics collection.
//
// Parameters:
//   - meter: OpenTelemetry meter for creating metrics instruments
//
// Example:
//
//	meter := otel.Meter("beluga.voice.s2s")
//	s2s.InitMetrics(meter)
//
// Example usage can be found in examples/voice/s2s/main.go
func InitMetrics(meter metric.Meter, tracer trace.Tracer) {
	metricsOnce.Do(func() {
		globalMetrics = NewMetrics(meter, tracer)
	})
}

// GetMetrics returns the global metrics instance.
// Returns nil if InitMetrics has not been called.
//
// Returns:
//   - *Metrics: The global metrics instance, or nil if not initialized
//
// Example:
//
//	metrics := s2s.GetMetrics()
//	if metrics != nil {
//	    // Use metrics
//	}
//
// Example usage can be found in examples/voice/s2s/main.go
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewProvider creates a new S2S provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
// S2S (Speech-to-Speech) providers convert speech directly to speech without
// intermediate text representation, enabling more natural and faster conversations.
// Supported providers include: deepgram, azure, etc.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the S2S provider to use (e.g., "deepgram", "azure")
//   - config: S2S configuration (can be nil to use defaults)
//   - opts: Optional configuration functions to customize the config
//
// Returns:
//   - iface.S2SProvider: A new S2S provider instance ready to use
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	config := s2s.DefaultConfig()
//	provider, err := s2s.NewProvider(ctx, "deepgram", config,
//	    s2s.WithAPIKey("your-api-key"),
//	    s2s.WithLanguage("en-US"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	responseAudio, err := provider.Process(ctx, inputAudio)
//
// Example usage can be found in examples/voice/s2s/main.go
func NewProvider(ctx context.Context, providerName string, config *Config, opts ...ConfigOption) (iface.S2SProvider, error) {
	// Apply options
	if config == nil {
		config = DefaultConfig()
	}
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewS2SError("NewProvider", ErrCodeInvalidConfig, err)
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get provider from registry
	registry := GetRegistry()
	provider, err := registry.GetProvider(config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create S2S provider '%s': %w", config.Provider, err)
	}

	return provider, nil
}
