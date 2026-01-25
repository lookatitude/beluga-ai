// Package noise provides interfaces and implementations for Noise Cancellation.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package noise

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/noise/iface"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance for Noise Cancellation operations.
// This should be called once during application startup to enable metrics collection.
//
// Parameters:
//   - meter: OpenTelemetry meter for creating metrics instruments
//
// Example:
//
//	meter := otel.Meter("beluga.voice.noise")
//	noise.InitMetrics(meter)
//
// Example usage can be found in examples/voice/noise/main.go.
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
//	metrics := noise.GetMetrics()
//	if metrics != nil {
//	    // Use metrics
//	}
//
// Example usage can be found in examples/voice/noise/main.go.
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewProvider creates a new Noise Cancellation provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
// Noise Cancellation removes background noise from audio streams to improve speech quality.
// Supported providers include: rnnoise, etc.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the noise cancellation provider to use (e.g., "rnnoise")
//   - config: Noise cancellation configuration (can be nil to use defaults)
//   - opts: Optional configuration functions to customize the config
//
// Returns:
//   - iface.NoiseCancellation: A new noise cancellation instance ready to use
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	config := noise.DefaultConfig()
//	provider, err := noise.NewProvider(ctx, "rnnoise", config,
//	    noise.WithAggressiveness(0.5),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	cleanedAudio, err := provider.CancelNoise(ctx, audioData)
//
// Example usage can be found in examples/voice/noise/main.go.
func NewProvider(ctx context.Context, providerName string, config *Config, opts ...ConfigOption) (iface.NoiseCancellation, error) {
	// Apply options
	if config == nil {
		config = DefaultConfig()
	}
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewNoiseCancellationError("NewProvider", ErrCodeInvalidConfig, err)
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get provider from registry
	registry := GetRegistry()
	provider, err := registry.GetProvider(config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Noise Cancellation provider '%s': %w", config.Provider, err)
	}

	return provider, nil
}
