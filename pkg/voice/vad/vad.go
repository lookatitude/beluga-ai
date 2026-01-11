// Package vad provides interfaces and implementations for Voice Activity Detection (VAD).
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package vad

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
	"go.opentelemetry.io/otel/metric"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance for VAD operations.
// This should be called once during application startup to enable metrics collection.
//
// Parameters:
//   - meter: OpenTelemetry meter for creating metrics instruments
//
// Example:
//
//	meter := otel.Meter("beluga.voice.vad")
//	vad.InitMetrics(meter)
//
// Example usage can be found in examples/voice/vad/main.go
func InitMetrics(meter metric.Meter) {
	metricsOnce.Do(func() {
		globalMetrics = NewMetrics(meter)
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
//	metrics := vad.GetMetrics()
//	if metrics != nil {
//	    // Use metrics
//	}
//
// Example usage can be found in examples/voice/vad/main.go
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewProvider creates a new VAD provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
// VAD (Voice Activity Detection) identifies when speech is present in audio streams.
// Supported providers include: webrtc, silero, etc.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the VAD provider to use (e.g., "webrtc", "silero")
//   - config: VAD configuration (can be nil to use defaults)
//   - opts: Optional configuration functions to customize the config
//
// Returns:
//   - iface.VADProvider: A new VAD provider instance ready to use
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	config := vad.DefaultConfig()
//	provider, err := vad.NewProvider(ctx, "webrtc", config,
//	    vad.WithFrameSize(30*time.Millisecond),
//	    vad.WithSilenceThreshold(0.5),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	isSpeech, err := provider.Detect(ctx, audioFrame)
//
// Example usage can be found in examples/voice/vad/main.go
func NewProvider(ctx context.Context, providerName string, config *Config, opts ...ConfigOption) (iface.VADProvider, error) {
	// Apply options
	if config == nil {
		config = DefaultConfig()
	}
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewVADError("NewProvider", ErrCodeInvalidConfig, err)
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get provider from registry
	registry := GetRegistry()
	provider, err := registry.GetProvider(config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create VAD provider '%s': %w", config.Provider, err)
	}

	return provider, nil
}
