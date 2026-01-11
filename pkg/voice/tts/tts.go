// Package tts provides interfaces and implementations for Text-to-Speech (TTS) operations.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package tts

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts/iface"
	"go.opentelemetry.io/otel/metric"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance for TTS operations.
// This should be called once during application startup to enable metrics collection.
//
// Parameters:
//   - meter: OpenTelemetry meter for creating metrics instruments
//
// Example:
//
//	meter := otel.Meter("beluga.voice.tts")
//	tts.InitMetrics(meter)
//
// Example usage can be found in examples/voice/tts/main.go
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
//	metrics := tts.GetMetrics()
//	if metrics != nil {
//	    // Use metrics
//	}
//
// Example usage can be found in examples/voice/tts/main.go
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewProvider creates a new TTS provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
// Supported providers include: deepgram, azure, google, openai, elevenlabs, etc.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the TTS provider to use (e.g., "deepgram", "azure")
//   - config: TTS configuration (can be nil to use defaults)
//   - opts: Optional configuration functions to customize the config
//
// Returns:
//   - iface.TTSProvider: A new TTS provider instance ready to use
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	config := tts.DefaultConfig()
//	provider, err := tts.NewProvider(ctx, "deepgram", config,
//	    tts.WithAPIKey("your-api-key"),
//	    tts.WithVoice("nova"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	audio, err := provider.Synthesize(ctx, "Hello, world!")
//
// Example usage can be found in examples/voice/tts/main.go
func NewProvider(ctx context.Context, providerName string, config *Config, opts ...ConfigOption) (iface.TTSProvider, error) {
	// Apply options
	if config == nil {
		config = DefaultConfig()
	}
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewTTSError("NewProvider", ErrCodeInvalidConfig, err)
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get provider from registry
	registry := GetRegistry()
	provider, err := registry.GetProvider(config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS provider '%s': %w", config.Provider, err)
	}

	return provider, nil
}
