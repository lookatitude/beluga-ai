// Package stt provides interfaces and implementations for Speech-to-Text (STT) operations.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package stt

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
	"go.opentelemetry.io/otel/metric"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance for STT operations.
// This should be called once during application startup to enable metrics collection.
//
// Parameters:
//   - meter: OpenTelemetry meter for creating metrics instruments
//
// Example:
//
//	meter := otel.Meter("beluga.voice.stt")
//	stt.InitMetrics(meter)
//
// Example usage can be found in examples/voice/stt/main.go
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
//	metrics := stt.GetMetrics()
//	if metrics != nil {
//	    // Use metrics
//	}
//
// Example usage can be found in examples/voice/stt/main.go
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewProvider creates a new STT provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
// Supported providers include: deepgram, azure, google, openai, whisper, etc.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - providerName: Name of the STT provider to use (e.g., "deepgram", "azure")
//   - config: STT configuration (can be nil to use defaults)
//   - opts: Optional configuration functions to customize the config
//
// Returns:
//   - iface.STTProvider: A new STT provider instance ready to use
//   - error: Configuration validation errors or provider creation errors
//
// Example:
//
//	config := stt.DefaultConfig()
//	provider, err := stt.NewProvider(ctx, "deepgram", config,
//	    stt.WithAPIKey("your-api-key"),
//	    stt.WithLanguage("en-US"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	transcript, err := provider.Transcribe(ctx, audioData)
//
// Example usage can be found in examples/voice/stt/main.go
func NewProvider(ctx context.Context, providerName string, config *Config, opts ...ConfigOption) (iface.STTProvider, error) {
	// Apply options
	if config == nil {
		config = DefaultConfig()
	}
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewSTTError("NewProvider", ErrCodeInvalidConfig, err)
	}

	// Override provider name from parameter if provided
	if providerName != "" {
		config.Provider = providerName
	}

	// Get provider from registry
	registry := GetRegistry()
	provider, err := registry.GetProvider(config.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create STT provider '%s': %w", config.Provider, err)
	}

	return provider, nil
}
