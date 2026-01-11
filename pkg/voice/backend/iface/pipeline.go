package iface

import (
	"context"
	"time"
)

// PipelineType represents the type of audio processing pipeline.
type PipelineType string

const (
	// PipelineTypeSTTTTS represents a traditional STT/TTS pipeline.
	PipelineTypeSTTTTS PipelineType = "stt_tts"
	// PipelineTypeS2S represents a speech-to-speech pipeline.
	PipelineTypeS2S PipelineType = "s2s"
)

// ConnectionState represents the connection state of a voice backend.
type ConnectionState string

const (
	// ConnectionStateDisconnected indicates the backend is not connected.
	ConnectionStateDisconnected ConnectionState = "disconnected"
	// ConnectionStateConnecting indicates the backend is connecting.
	ConnectionStateConnecting ConnectionState = "connecting"
	// ConnectionStateConnected indicates the backend is connected and ready.
	ConnectionStateConnected ConnectionState = "connected"
	// ConnectionStateReconnecting indicates the backend is reconnecting after an error.
	ConnectionStateReconnecting ConnectionState = "reconnecting"
	// ConnectionStateError indicates the backend is in an error state.
	ConnectionStateError ConnectionState = "error"
)

// PipelineState represents the state of a voice session pipeline.
type PipelineState string

const (
	// PipelineStateIdle indicates the session is idle, not processing.
	PipelineStateIdle PipelineState = "idle"
	// PipelineStateListening indicates the session is listening for user speech.
	PipelineStateListening PipelineState = "listening"
	// PipelineStateProcessing indicates the session is processing audio/transcript.
	PipelineStateProcessing PipelineState = "processing"
	// PipelineStateSpeaking indicates the session is playing agent response.
	PipelineStateSpeaking PipelineState = "speaking"
	// PipelineStateError indicates the session is in an error state.
	PipelineStateError PipelineState = "error"
)

// PersistenceStatus represents the persistence status of a voice session.
type PersistenceStatus string

const (
	// PersistenceStatusActive indicates the session is active and should persist.
	PersistenceStatusActive PersistenceStatus = "active"
	// PersistenceStatusCompleted indicates the session is completed and is ephemeral.
	PersistenceStatusCompleted PersistenceStatus = "completed"
)

// HealthStatus represents the health status of a voice backend.
type HealthStatus struct {
	Status    string            `json:"status"`    // "healthy", "degraded", "unhealthy"
	Details   map[string]any    `json:"details"`    // Additional health details
	LastCheck time.Time         `json:"last_check"` // Last health check timestamp
}

// ProviderCapabilities represents the capabilities of a voice backend provider.
type ProviderCapabilities struct {
	// S2SSupport indicates if the provider supports speech-to-speech processing.
	S2SSupport bool `json:"s2s_support"`
	// MultiUserSupport indicates if the provider supports concurrent multi-user conversations.
	MultiUserSupport bool `json:"multi_user_support"`
	// SessionPersistence indicates if the provider supports session state persistence.
	SessionPersistence bool `json:"session_persistence"`
	// CustomAuth indicates if the provider supports custom authentication.
	CustomAuth bool `json:"custom_auth"`
	// CustomRateLimiting indicates if the provider supports custom rate limiting.
	CustomRateLimiting bool `json:"custom_rate_limiting"`
	// MaxConcurrentSessions is the maximum concurrent sessions (0 = unlimited).
	MaxConcurrentSessions int `json:"max_concurrent_sessions"`
	// MinLatency is the minimum achievable latency.
	MinLatency time.Duration `json:"min_latency"`
	// SupportedCodecs is a list of supported audio codecs (e.g., ["opus", "pcm"]).
	SupportedCodecs []string `json:"supported_codecs"`
}

// PipelineConfiguration represents the audio processing pipeline setup.
type PipelineConfiguration struct {
	// Type is the pipeline type (STT_TTS or S2S).
	Type PipelineType `json:"type"`
	// STTProvider is the STT provider name (if STT_TTS pipeline).
	STTProvider string `json:"stt_provider,omitempty"`
	// TTSProvider is the TTS provider name (if STT_TTS pipeline).
	TTSProvider string `json:"tts_provider,omitempty"`
	// S2SProvider is the S2S provider name (if S2S pipeline).
	S2SProvider string `json:"s2s_provider,omitempty"`
	// VADProvider is the VAD provider name (optional).
	VADProvider string `json:"vad_provider,omitempty"`
	// TurnDetectionProvider is the turn detection provider name (optional).
	TurnDetectionProvider string `json:"turn_detection_provider,omitempty"`
	// NoiseCancellationProvider is the noise cancellation provider name (optional).
	NoiseCancellationProvider string `json:"noise_cancellation_provider,omitempty"`
	// ProcessingOrder is the order of processing components.
	ProcessingOrder []string `json:"processing_order,omitempty"`
	// LatencyTarget is the target latency (e.g., 500ms).
	LatencyTarget time.Duration `json:"latency_target"`
	// CustomProcessors is a list of custom audio processors (extensibility hooks).
	CustomProcessors []CustomProcessor `json:"custom_processors,omitempty"`
}

// CustomProcessor represents a custom audio processor for extensibility.
type CustomProcessor interface {
	// Process processes audio data.
	Process(ctx context.Context, audio []byte, metadata map[string]any) ([]byte, error)
	// GetName returns the processor name.
	GetName() string
	// GetOrder returns the processing order (lower = earlier).
	GetOrder() int
}
