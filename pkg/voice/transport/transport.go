// Package transport provides interfaces and implementations for audio transport.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package transport

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport/iface"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance for Transport operations.
// This should be called once during application startup to enable metrics collection.
//
// Parameters:
//   - meter: OpenTelemetry meter for creating metrics instruments
//
// Example:
//
//	meter := otel.Meter("beluga.voice.transport")
//	transport.InitMetrics(meter)
//
// Example usage can be found in examples/voice/transport/main.go
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
//	metrics := transport.GetMetrics()
//	if metrics != nil {
//	    // Use metrics
//	}
//
// Example usage can be found in examples/voice/transport/main.go
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewProvider creates a new Transport provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
// Transport handles audio data transmission between components in the voice pipeline.
// Supported providers include: websocket, webrtc, etc.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the transport provider to use (e.g., "websocket", "webrtc")
//   - config: Transport configuration (can be nil to use defaults)
//   - opts: Optional configuration functions to customize the config
//
// Returns:
//   - iface.Transport: A new transport instance ready to use
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	config := transport.DefaultConfig()
//	provider, err := transport.NewProvider(ctx, "websocket", config,
//	    transport.WithBufferSize(4096),
//	    transport.WithTimeout(30*time.Second),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = provider.Send(ctx, audioData)
//
// Example usage can be found in examples/voice/transport/main.go
func NewProvider(ctx context.Context, providerName string, config *Config, opts ...ConfigOption) (iface.Transport, error) {
	// Apply options
	if config == nil {
		config = DefaultConfig()
	}
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewTransportError("NewProvider", ErrCodeInvalidConfig, err)
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get provider from registry
	registry := GetRegistry()
	provider, err := registry.GetProvider(config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Transport provider '%s': %w", config.Provider, err)
	}

	return provider, nil
}
