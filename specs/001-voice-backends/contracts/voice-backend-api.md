# Voice Backend API Contract

**Date**: 2026-01-11  
**Feature**: Voice Backends (001-voice-backends)  
**Version**: 1.0.0

## Overview

This document defines the API contract for the Voice Backends package. The API provides interfaces for creating backend instances, managing sessions, processing audio, and integrating with agents.

## Core Interfaces

### VoiceBackend Interface

Main interface for voice backend instances.

```go
package backend

import (
    "context"
    "time"
)

// VoiceBackend represents a voice backend instance that manages real-time voice pipelines.
type VoiceBackend interface {
    // Lifecycle Management
    
    // Start initializes and starts the backend instance.
    // Returns error if backend cannot be started (invalid config, provider unavailable, etc.).
    Start(ctx context.Context) error
    
    // Stop gracefully shuts down the backend, completing in-flight conversations.
    // Per FR-020: Completes in-flight conversations before termination.
    Stop(ctx context.Context) error
    
    // Session Management
    
    // CreateSession creates a new voice session with the given configuration.
    // Returns error if session cannot be created (invalid config, capacity exceeded, etc.).
    // Per FR-002: Creates sessions from registered providers using configuration.
    CreateSession(ctx context.Context, config *SessionConfig) (VoiceSession, error)
    
    // GetSession retrieves an existing session by ID.
    // Returns error if session not found.
    GetSession(ctx context.Context, sessionID string) (VoiceSession, error)
    
    // ListSessions returns all active sessions for this backend.
    // Per FR-010: Supports concurrent multi-user conversations.
    ListSessions(ctx context.Context) ([]VoiceSession, error)
    
    // CloseSession closes a session and cleans up resources.
    // Per FR-026: Completed sessions are ephemeral (no persistence after close).
    CloseSession(ctx context.Context, sessionID string) error
    
    // Health & Status
    
    // HealthCheck performs a health check on the backend instance.
    // Per FR-018: Provides health check endpoints or methods.
    HealthCheck(ctx context.Context) (*HealthStatus, error)
    
    // GetConnectionState returns the current connection state.
    GetConnectionState() ConnectionState
    
    // GetActiveSessionCount returns the number of active sessions.
    GetActiveSessionCount() int
    
    // Configuration
    
    // GetConfig returns the backend configuration.
    GetConfig() *Config
    
    // UpdateConfig updates the backend configuration.
    // Returns error if new config is invalid.
    UpdateConfig(ctx context.Context, config *Config) error
}
```

### BackendProvider Interface

Interface for backend provider implementations.

```go
// BackendProvider represents a voice backend provider implementation.
type BackendProvider interface {
    // GetName returns the provider name (e.g., "livekit", "pipecat").
    GetName() string
    
    // GetCapabilities returns the capabilities of this provider.
    // Per FR-002: Providers expose capabilities (S2S support, multi-user support).
    GetCapabilities(ctx context.Context) (*ProviderCapabilities, error)
    
    // CreateBackend creates a new backend instance with the given configuration.
    // Per FR-002: Creates backends from registered providers using configuration.
    // Per FR-016: Validates configuration before instance creation.
    CreateBackend(ctx context.Context, config *Config) (VoiceBackend, error)
    
    // ValidateConfig validates the provider-specific configuration.
    // Per FR-016: Validates backend provider configuration before instance creation.
    ValidateConfig(ctx context.Context, config *Config) error
    
    // GetConfigSchema returns the configuration schema for this provider.
    GetConfigSchema() *ConfigSchema
}
```

### VoiceSession Interface

Interface for voice conversation sessions.

```go
// VoiceSession represents an active voice conversation session within a backend.
type VoiceSession interface {
    // Lifecycle
    
    // Start starts the voice session and begins listening for audio.
    Start(ctx context.Context) error
    
    // Stop stops the voice session and cleans up resources.
    // Per FR-026: Completed sessions are ephemeral.
    Stop(ctx context.Context) error
    
    // Audio Processing
    
    // ProcessAudio processes incoming audio through the configured pipeline.
    // Per FR-004: Processes audio through configurable pipeline (STT/TTS or S2S).
    // Per FR-005: Integrates with voice package components for orchestration.
    ProcessAudio(ctx context.Context, audio []byte) error
    
    // SendAudio sends audio to the user connection.
    SendAudio(ctx context.Context, audio []byte) error
    
    // ReceiveAudio returns a channel for receiving audio from the user.
    ReceiveAudio() <-chan []byte
    
    // Agent Integration
    
    // SetAgentCallback sets the agent callback function for processing transcripts.
    // Per FR-009: Integrates with agent framework to route transcripts and receive responses.
    SetAgentCallback(callback func(context.Context, string) (string, error)) error
    
    // SetAgentInstance sets the agent instance for processing transcripts.
    // Per FR-009: Integrates with agent framework.
    SetAgentInstance(agent agents.Agent) error
    
    // State Management
    
    // GetState returns the current pipeline state.
    GetState() PipelineState
    
    // GetPersistenceStatus returns the persistence status (active=persist, completed=ephemeral).
    // Per FR-025: Active sessions persist.
    // Per FR-026: Completed sessions are ephemeral.
    GetPersistenceStatus() PersistenceStatus
    
    // UpdateMetadata updates session metadata.
    UpdateMetadata(metadata map[string]any) error
    
    // GetID returns the session ID.
    GetID() string
}
```

## Registry Interface

### BackendRegistry Interface

Global registry for backend providers.

```go
// BackendRegistry manages backend provider registration and retrieval.
type BackendRegistry interface {
    // Register registers a new backend provider with the registry.
    // Per FR-001: Provides registry for voice backend providers.
    Register(name string, provider BackendProvider)
    
    // Create creates a new backend instance using the registered provider.
    // Per FR-002: Creates backends from registered providers using configuration.
    Create(ctx context.Context, name string, config *Config) (VoiceBackend, error)
    
    // ListProviders returns a list of all registered provider names.
    // Per User Story 4: Developers can query available providers.
    ListProviders() []string
    
    // IsRegistered checks if a provider is registered.
    IsRegistered(name string) bool
    
    // GetProvider returns a provider by name.
    GetProvider(name string) (BackendProvider, error)
}
```

## Configuration Types

### Config

Backend configuration structure.

```go
// Config represents the configuration for a voice backend instance.
type Config struct {
    // Provider Configuration
    Provider        string                 `mapstructure:"provider" yaml:"provider" validate:"required"`
    ProviderConfig  map[string]any          `mapstructure:"provider_config" yaml:"provider_config"`
    
    // Pipeline Configuration
    PipelineType    PipelineType            `mapstructure:"pipeline_type" yaml:"pipeline_type" validate:"required,oneof=stt_tts s2s"`
    STTProvider     string                  `mapstructure:"stt_provider" yaml:"stt_provider" validate:"required_if=PipelineType stt_tts"`
    TTSProvider     string                  `mapstructure:"tts_provider" yaml:"tts_provider" validate:"required_if=PipelineType stt_tts"`
    S2SProvider     string                  `mapstructure:"s2s_provider" yaml:"s2s_provider" validate:"required_if=PipelineType s2s"`
    VADProvider      string                 `mapstructure:"vad_provider" yaml:"vad_provider"`
    TurnDetectionProvider string            `mapstructure:"turn_detection_provider" yaml:"turn_detection_provider"`
    NoiseCancellationProvider string        `mapstructure:"noise_cancellation_provider" yaml:"noise_cancellation_provider"`
    
    // Performance Configuration
    LatencyTarget   time.Duration           `mapstructure:"latency_target" yaml:"latency_target" default:"500ms" validate:"min=100ms,max=5s"`
    Timeout         time.Duration           `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`
    MaxRetries      int                     `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
    RetryDelay      time.Duration           `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
    
    // Scalability Configuration
    MaxConcurrentSessions int                `mapstructure:"max_concurrent_sessions" yaml:"max_concurrent_sessions" default:"100" validate:"gte=1"`
    
    // Observability Configuration
    EnableTracing   bool                    `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
    EnableMetrics   bool                    `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
    EnableStructuredLogging bool            `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
    
    // Extensibility Hooks
    AuthHook        AuthHook                `mapstructure:"-" yaml:"-"` // Per FR-021, FR-022
    RateLimiter     RateLimiter             `mapstructure:"-" yaml:"-"` // Per FR-023, FR-024
    DataRetentionHook DataRetentionHook     `mapstructure:"-" yaml:"-"` // Per FR-027, FR-028
    CustomProcessors []CustomProcessor      `mapstructure:"custom_processors" yaml:"custom_processors"` // Per FR-015
}
```

### SessionConfig

Session configuration structure.

```go
// SessionConfig represents configuration for creating a voice session.
type SessionConfig struct {
    // User Connection
    UserID          string                  `mapstructure:"user_id" yaml:"user_id" validate:"required"`
    Transport       string                  `mapstructure:"transport" yaml:"transport" validate:"required,oneof=webrtc websocket"`
    ConnectionURL   string                  `mapstructure:"connection_url" yaml:"connection_url" validate:"required"`
    
    // Agent Integration
    AgentCallback   func(context.Context, string) (string, error) `mapstructure:"-" yaml:"-"`
    AgentInstance   agents.Agent            `mapstructure:"-" yaml:"-"`
    
    // Pipeline Overrides (optional, uses backend defaults if not set)
    PipelineType    PipelineType            `mapstructure:"pipeline_type" yaml:"pipeline_type"`
    
    // Session Metadata
    Metadata        map[string]any          `mapstructure:"metadata" yaml:"metadata"`
}
```

## Extensibility Hooks

### AuthHook Interface

```go
// AuthHook provides authentication hooks for backend providers.
// Per FR-021, FR-022: Backend-agnostic authentication with framework hooks.
type AuthHook interface {
    // Authenticate authenticates a user connection.
    Authenticate(ctx context.Context, token string, metadata map[string]any) (*AuthResult, error)
    
    // Authorize authorizes an operation for a user.
    Authorize(ctx context.Context, userID string, operation string) (bool, error)
}
```

### RateLimiter Interface

```go
// RateLimiter provides rate limiting for backend operations.
// Per FR-023, FR-024: Provider-specific with framework fallback.
type RateLimiter interface {
    // Allow checks if a request is allowed under rate limits.
    Allow(ctx context.Context, key string) (bool, error)
    
    // Wait waits until a request is allowed.
    Wait(ctx context.Context, key string) error
}
```

### DataRetentionHook Interface

```go
// DataRetentionHook provides data privacy and retention hooks.
// Per FR-027, FR-028: Provider-controlled with framework hooks.
type DataRetentionHook interface {
    // ShouldRetain determines if data should be retained.
    ShouldRetain(ctx context.Context, dataType string, metadata map[string]any) (bool, error)
    
    // GetRetentionPeriod returns the retention period for data.
    GetRetentionPeriod(ctx context.Context, dataType string) (time.Duration, error)
}
```

### AudioProcessor Interface

```go
// AudioProcessor provides custom audio processing hooks.
// Per FR-015: Extensibility hooks for custom audio processors.
type AudioProcessor interface {
    // Process processes audio data.
    Process(ctx context.Context, audio []byte, metadata map[string]any) ([]byte, error)
    
    // GetName returns the processor name.
    GetName() string
    
    // GetOrder returns the processing order (lower = earlier).
    GetOrder() int
}
```

## Error Types

### BackendError

```go
// BackendError represents an error that occurred during backend operations.
type BackendError struct {
    Op      string
    Code    string
    Err     error
    Message string
    Details map[string]any
}

// Error codes
const (
    ErrCodeInvalidConfig        = "invalid_config"
    ErrCodeProviderNotFound     = "provider_not_found"
    ErrCodeConnectionFailed     = "connection_failed"
    ErrCodeConnectionTimeout    = "connection_timeout"
    ErrCodeSessionNotFound      = "session_not_found"
    ErrCodeSessionLimitExceeded = "session_limit_exceeded"
    ErrCodeRateLimitExceeded    = "rate_limit_exceeded"
    ErrCodeAuthenticationFailed = "authentication_failed"
    ErrCodeAuthorizationFailed  = "authorization_failed"
    ErrCodePipelineError        = "pipeline_error"
    ErrCodeAgentError           = "agent_error"
    ErrCodeTimeout              = "timeout"
    ErrCodeContextCanceled      = "context_canceled"
)
```

## Factory Functions

### NewBackend

```go
// NewBackend creates a new voice backend instance using the global registry.
// Per FR-002: Creates backends from registered providers using configuration.
func NewBackend(ctx context.Context, providerName string, config *Config) (VoiceBackend, error)
```

### GetRegistry

```go
// GetRegistry returns the global backend provider registry.
// Per FR-001: Provides registry for voice backend providers.
func GetRegistry() BackendRegistry
```

## Usage Example

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

// Get registry and create backend
registry := backend.GetRegistry()
backend, err := registry.Create(ctx, "livekit", &backend.Config{
    Provider: "livekit",
    PipelineType: backend.PipelineTypeSTTTTS,
    STTProvider: "deepgram",
    TTSProvider: "elevenlabs",
    LatencyTarget: 500 * time.Millisecond,
    MaxConcurrentSessions: 100,
})
if err != nil {
    log.Fatal(err)
}

// Start backend
err = backend.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// Create session
session, err := backend.CreateSession(ctx, &backend.SessionConfig{
    UserID: "user-123",
    Transport: "webrtc",
    ConnectionURL: "wss://example.com/signaling",
    AgentCallback: func(ctx context.Context, transcript string) (string, error) {
        // Process transcript and return response
        return "Hello! How can I help you?", nil
    },
})
if err != nil {
    log.Fatal(err)
}

// Start session
err = session.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// Process audio
audio := []byte{...} // Audio data
err = session.ProcessAudio(ctx, audio)
if err != nil {
    log.Fatal(err)
}
```
