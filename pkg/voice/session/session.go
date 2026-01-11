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

// InitMetrics initializes the global metrics instance for Voice Session operations.
// This should be called once during application startup to enable metrics collection.
//
// Parameters:
//   - meter: OpenTelemetry meter for creating metrics instruments
//
// Example:
//
//	meter := otel.Meter("beluga.voice.session")
//	session.InitMetrics(meter)
//
// Example usage can be found in examples/voice/session/main.go
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
//	metrics := session.GetMetrics()
//	if metrics != nil {
//	    // Use metrics
//	}
//
// Example usage can be found in examples/voice/session/main.go
func GetMetrics() *Metrics {
	return globalMetrics
}

// NewVoiceSession creates a new VoiceSession instance based on the provided options.
// It validates the configuration and initializes all required components.
// A voice session manages the complete lifecycle of a voice interaction, including
// audio input/output, transcription, synthesis, and agent communication.
//
// The session can operate in two modes:
//   - Traditional mode: Requires STT and TTS providers
//   - S2S mode: Uses a single Speech-to-Speech provider
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - opts: Optional configuration functions (WithSTTProvider, WithTTSProvider, WithS2SProvider, etc.)
//
// Returns:
//   - iface.VoiceSession: A new voice session instance ready to use
//   - error: Configuration validation errors or initialization errors
//
// Example:
//
//	session, err := session.NewVoiceSession(ctx,
//	    session.WithSTTProvider(sttProvider),
//	    session.WithTTSProvider(ttsProvider),
//	    session.WithVADProvider(vadProvider),
//	    session.WithAgent(agent),
//	    session.WithConfig(session.DefaultConfig()),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = session.Start(ctx)
//
// Example usage can be found in examples/voice/session/main.go
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

	// Validate required providers - either STT+TTS or S2S
	if options.S2SProvider != nil {
		// S2S mode: S2S provider is sufficient
		if options.STTProvider != nil || options.TTSProvider != nil {
			return nil, NewSessionError("NewVoiceSession", ErrCodeInvalidConfig,
				errors.New("cannot specify both S2S provider and STT/TTS providers"))
		}
	} else {
		// Traditional mode: STT and TTS providers are required
		if options.STTProvider == nil {
			return nil, NewSessionError("NewVoiceSession", ErrCodeInvalidConfig,
				errors.New("STT provider is required (or use S2S provider)"))
		}
		if options.TTSProvider == nil {
			return nil, NewSessionError("NewVoiceSession", ErrCodeInvalidConfig,
				errors.New("TTS provider is required (or use S2S provider)"))
		}
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
		S2SProvider:       options.S2SProvider,
		VADProvider:       options.VADProvider,
		TurnDetector:      options.TurnDetector,
		Transport:         options.Transport,
		NoiseCancellation: options.NoiseCancellation,
		AgentCallback:     options.AgentCallback,
		OnStateChanged:    options.OnStateChanged,
		Config:            internalConfig,
		AgentInstance:     options.AgentInstance,
		AgentConfig:       options.AgentConfig,
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
