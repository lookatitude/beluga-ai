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
	LastCheck time.Time      `json:"last_check"`
	Details   map[string]any `json:"details"`
	Status    string         `json:"status"`
}

// ProviderCapabilities represents the capabilities of a voice backend provider.
type ProviderCapabilities struct {
	SupportedCodecs       []string      `json:"supported_codecs"`
	MaxConcurrentSessions int           `json:"max_concurrent_sessions"`
	MinLatency            time.Duration `json:"min_latency"`
	S2SSupport            bool          `json:"s2s_support"`
	MultiUserSupport      bool          `json:"multi_user_support"`
	SessionPersistence    bool          `json:"session_persistence"`
	CustomAuth            bool          `json:"custom_auth"`
	CustomRateLimiting    bool          `json:"custom_rate_limiting"`
}

// PipelineConfiguration represents the audio processing pipeline setup.
type PipelineConfiguration struct {
	Type                      PipelineType      `json:"type"`
	STTProvider               string            `json:"stt_provider,omitempty"`
	TTSProvider               string            `json:"tts_provider,omitempty"`
	S2SProvider               string            `json:"s2s_provider,omitempty"`
	VADProvider               string            `json:"vad_provider,omitempty"`
	TurnDetectionProvider     string            `json:"turn_detection_provider,omitempty"`
	NoiseCancellationProvider string            `json:"noise_cancellation_provider,omitempty"`
	ProcessingOrder           []string          `json:"processing_order,omitempty"`
	CustomProcessors          []CustomProcessor `json:"custom_processors,omitempty"`
	LatencyTarget             time.Duration     `json:"latency_target"`
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
