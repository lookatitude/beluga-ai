// Package session provides interfaces and implementations for Voice Session management.
// This package follows the Beluga AI Framework's design patterns for consistency,
// extensibility, configuration management, and observability.
package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session/internal"
	"go.opentelemetry.io/otel/metric"
)

// Global metrics instance - initialized once.
var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance.
func InitMetrics(meter metric.Meter) {
	metricsOnce.Do(func() {
		globalMetrics = NewMetrics(meter)
	})
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewVoiceSession creates a new VoiceSession instance based on the provided options.
// It validates the configuration and initializes all required components.
func NewVoiceSession(ctx context.Context, opts ...VoiceOption) (iface.VoiceSession, error) {
	// Initialize options with defaults
	options := &VoiceOptions{
		Config: DefaultConfig(),
	}

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	// Validate configuration
	if options.Config != nil {
		if err := options.Config.Validate(); err != nil {
			return nil, NewSessionError("NewVoiceSession", ErrCodeInvalidConfig, err)
		}
	}

	// Validate required providers
	if options.STTProvider == nil {
		return nil, NewSessionError("NewVoiceSession", ErrCodeInvalidConfig,
			errors.New("STT provider is required"))
	}
	if options.TTSProvider == nil {
		return nil, NewSessionError("NewVoiceSession", ErrCodeInvalidConfig,
			errors.New("TTS provider is required"))
	}

	// Generate session ID if not provided
	sessionID := options.Config.SessionID
	if sessionID == "" {
		sessionID = generateSessionID()
	}

	// Convert session.Config to internal.Config to avoid import cycle
	internalConfig := &internal.Config{
		SessionID:         options.Config.SessionID,
		Timeout:           options.Config.Timeout,
		AutoStart:         options.Config.AutoStart,
		EnableKeepAlive:   options.Config.EnableKeepAlive,
		KeepAliveInterval: options.Config.KeepAliveInterval,
		MaxRetries:        options.Config.MaxRetries,
		RetryDelay:        options.Config.RetryDelay,
	}

	// Convert VoiceOptions to internal.VoiceOptions
	internalOpts := &internal.VoiceOptions{
		STTProvider:       options.STTProvider,
		TTSProvider:       options.TTSProvider,
		VADProvider:       options.VADProvider,
		TurnDetector:      options.TurnDetector,
		Transport:         options.Transport,
		NoiseCancellation: options.NoiseCancellation,
		AgentCallback:     options.AgentCallback,
		OnStateChanged:    options.OnStateChanged,
		Config:            internalConfig,
	}

	// Create session implementation
	impl, err := internal.NewVoiceSessionImpl(internalConfig, internalOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create session implementation: %w", err)
	}

	return impl, nil
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	// TODO: Use a proper UUID generator (e.g., github.com/google/uuid)
	// For now, use timestamp-based ID
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}
