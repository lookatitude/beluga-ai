// Package turndetection provides interfaces and implementations for Turn Detection.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package turndetection

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection/iface"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance for Turn Detection operations.
// This should be called once during application startup to enable metrics collection.
//
// Parameters:
//   - meter: OpenTelemetry meter for creating metrics instruments
//
// Example:
//
//	meter := otel.Meter("beluga.voice.turndetection")
//	turndetection.InitMetrics(meter)
//
// Example usage can be found in examples/voice/turndetection/main.go.
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
//	metrics := turndetection.GetMetrics()
//	if metrics != nil {
//	    // Use metrics
//	}
//
// Example usage can be found in examples/voice/turndetection/main.go.
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewProvider creates a new Turn Detection provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
// Turn Detection identifies when a speaker has finished speaking and it's time for a response.
// Supported providers include: silence-based, energy-based, etc.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the turn detection provider to use (e.g., "silence", "energy")
//   - config: Turn detection configuration (can be nil to use defaults)
//   - opts: Optional configuration functions to customize the config
//
// Returns:
//   - iface.TurnDetector: A new turn detector instance ready to use
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	config := turndetection.DefaultConfig()
//	provider, err := turndetection.NewProvider(ctx, "silence", config,
//	    turndetection.WithSilenceDuration(500*time.Millisecond),
//	    turndetection.WithMinSpeechDuration(200*time.Millisecond),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	isTurnComplete, err := provider.DetectTurn(ctx, audioStream)
//
// Example usage can be found in examples/voice/turndetection/main.go.
func NewProvider(ctx context.Context, providerName string, config *Config, opts ...ConfigOption) (iface.TurnDetector, error) {
	// Apply options
	if config == nil {
		config = DefaultConfig()
	}
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewTurnDetectionError("NewProvider", ErrCodeInvalidConfig, err)
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get provider from registry
	registry := GetRegistry()
	provider, err := registry.GetProvider(config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Turn Detection provider '%s': %w", config.Provider, err)
	}

	return provider, nil
}
