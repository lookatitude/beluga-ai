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

// Global metrics instance - initialized once
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance
func InitMetrics(meter metric.Meter) {
	metricsOnce.Do(func() {
		globalMetrics = NewMetrics(meter)
	})
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewProvider creates a new TTS provider instance based on the configuration.
// It uses the global registry to find and instantiate the appropriate provider.
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
