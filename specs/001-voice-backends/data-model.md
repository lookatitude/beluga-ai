# Data Model: Voice Backends

**Date**: 2026-01-11  
**Feature**: Voice Backends (001-voice-backends)

## Overview

This document defines the core entities, their attributes, relationships, and validation rules for the Voice Backends package. The data model supports multiple backend providers, session management, pipeline orchestration, and extensibility hooks.

## Core Entities

### VoiceBackend

Represents a voice backend instance that manages real-time voice pipelines.

**Attributes**:
- `ID` (string): Unique identifier for the backend instance
- `Provider` (string): Backend provider name (e.g., "livekit", "pipecat")
- `Config` (Config): Backend configuration
- `ActiveSessions` (map[string]*VoiceSession): Active voice sessions keyed by session ID
- `ConnectionState` (ConnectionState): Current connection state (connecting, connected, disconnected, error)
- `HealthStatus` (HealthStatus): Health check status (healthy, degraded, unhealthy)
- `CreatedAt` (time.Time): Backend creation timestamp
- `LastHealthCheck` (time.Time): Last health check timestamp
- `Metrics` (*Metrics): OTEL metrics recorder
- `mu` (sync.RWMutex): Mutex for thread-safe access

**Relationships**:
- Has many `VoiceSession` instances (1:N)
- Belongs to one `BackendProvider` (N:1)
- Uses one `PipelineConfiguration` (1:1)

**Validation Rules**:
- `ID` must be non-empty
- `Provider` must be registered in `BackendRegistry`
- `Config` must pass validation (per FR-016)
- `ActiveSessions` must be thread-safe accessed

**State Transitions**:
```
[Initial] → [Connecting] → [Connected] → [Disconnected]
                              ↓
                          [Error] → [Reconnecting] → [Connected]
```

### BackendProvider

Represents a voice backend provider implementation (e.g., LiveKit, Pipecat).

**Attributes**:
- `Name` (string): Provider name (e.g., "livekit", "pipecat")
- `Capabilities` (*ProviderCapabilities): Provider capabilities
- `ConfigSchema` (ConfigSchema): Configuration schema for validation
- `Factory` (BackendFactory): Factory function for creating backend instances
- `Metadata` (map[string]any): Provider metadata (version, description, etc.)

**Relationships**:
- Has many `VoiceBackend` instances (1:N)
- Registered in `BackendRegistry` (N:1)

**Validation Rules**:
- `Name` must be unique in registry
- `Factory` must not be nil
- `ConfigSchema` must be valid

**ProviderCapabilities**:
- `S2SSupport` (bool): Supports speech-to-speech processing
- `MultiUserSupport` (bool): Supports concurrent multi-user conversations
- `SessionPersistence` (bool): Supports session state persistence
- `CustomAuth` (bool): Supports custom authentication
- `CustomRateLimiting` (bool): Supports custom rate limiting
- `MaxConcurrentSessions` (int): Maximum concurrent sessions (0 = unlimited)
- `MinLatency` (time.Duration): Minimum achievable latency
- `SupportedCodecs` ([]string): Supported audio codecs (e.g., ["opus", "pcm"])

### VoiceSession

Represents an active voice conversation session within a backend.

**Attributes**:
- `ID` (string): Unique session identifier
- `BackendID` (string): Parent backend instance ID
- `UserConnection` (*UserConnection): User connection details
- `PipelineState` (PipelineState): Current pipeline state (idle, listening, processing, speaking, error)
- `AgentIntegration` (*AgentIntegration): Agent integration configuration
- `AudioBuffers` (*AudioBuffers): Audio input/output buffers
- `TurnDetectionState` (*TurnDetectionState): Turn detection state
- `PersistenceStatus` (PersistenceStatus): Persistence status (active=persist, completed=ephemeral)
- `CreatedAt` (time.Time): Session creation timestamp
- `LastActivity` (time.Time): Last activity timestamp
- `Metadata` (map[string]any): Session metadata
- `mu` (sync.RWMutex): Mutex for thread-safe access

**Relationships**:
- Belongs to one `VoiceBackend` (N:1)
- Uses one `PipelineConfiguration` (1:1)
- Has one `AgentIntegration` (1:1, optional)

**Validation Rules**:
- `ID` must be unique within backend
- `BackendID` must reference existing backend
- `PersistenceStatus` determines if session state is saved (active sessions persist per FR-025)

**State Transitions**:
```
[Idle] → [Listening] → [Processing] → [Speaking] → [Listening]
   ↓         ↓              ↓             ↓
[Error]  [Error]       [Error]       [Error]
```

**Persistence Rules**:
- `PersistenceStatus == Active`: Session state persisted (survives restarts)
- `PersistenceStatus == Completed`: Session ephemeral (no persistence, cleaned up)

### PipelineConfiguration

Represents the audio processing pipeline setup.

**Attributes**:
- `Type` (PipelineType): Pipeline type (STT_TTS, S2S)
- `STTProvider` (string): STT provider name (if STT_TTS pipeline)
- `TTSProvider` (string): TTS provider name (if STT_TTS pipeline)
- `S2SProvider` (string): S2S provider name (if S2S pipeline)
- `VADProvider` (string): VAD provider name (optional)
- `TurnDetectionProvider` (string): Turn detection provider name (optional)
- `NoiseCancellationProvider` (string): Noise cancellation provider name (optional)
- `ProcessingOrder` ([]string): Order of processing components
- `LatencyTarget` (time.Duration): Target latency (e.g., 500ms)
- `CustomProcessors` ([]CustomProcessor): Custom audio processors (extensibility hooks)

**Relationships**:
- Used by one `VoiceBackend` (1:1)
- Used by one `VoiceSession` (1:1)

**Validation Rules**:
- `Type` must be valid (STT_TTS or S2S)
- If `Type == STT_TTS`: `STTProvider` and `TTSProvider` must be non-empty
- If `Type == S2S`: `S2SProvider` must be non-empty
- `ProcessingOrder` must include all enabled components
- `LatencyTarget` must be > 0

**PipelineType**:
- `STT_TTS`: Traditional pipeline (STT → Agent → TTS)
- `S2S`: Speech-to-speech pipeline (bypasses text transcription)

### BackendRegistry

Represents the global registry for backend providers.

**Attributes**:
- `providers` (map[string]*BackendProvider): Registered providers keyed by name
- `mu` (sync.RWMutex): Mutex for thread-safe access

**Relationships**:
- Has many `BackendProvider` instances (1:N)

**Validation Rules**:
- Provider names must be unique
- Providers must be registered before use

### UserConnection

Represents a user's connection to a voice session.

**Attributes**:
- `UserID` (string): User identifier
- `ConnectionID` (string): Connection identifier
- `Transport` (string): Transport type (webrtc, websocket)
- `RemoteAddress` (string): Remote address
- `ConnectedAt` (time.Time): Connection timestamp
- `LastPing` (time.Time): Last ping timestamp
- `AuthToken` (string): Authentication token (provider-specific)
- `Metadata` (map[string]any): Connection metadata

**Relationships**:
- Belongs to one `VoiceSession` (1:1)

**Validation Rules**:
- `UserID` must be non-empty
- `ConnectionID` must be unique
- `Transport` must be valid (webrtc, websocket)

### AgentIntegration

Represents agent integration configuration for a voice session.

**Attributes**:
- `AgentInstance` (agents.Agent): Agent instance (optional, can use callback instead)
- `AgentCallback` (func(context.Context, string) (string, error)): Agent callback function (optional)
- `StreamingMode` (bool): Enable streaming agent responses
- `InterruptionHandling` (bool): Enable interruption detection
- `PreemptiveGeneration` (bool): Enable preemptive response generation

**Relationships**:
- Belongs to one `VoiceSession` (1:1)

**Validation Rules**:
- Either `AgentInstance` or `AgentCallback` must be set
- `AgentInstance` must implement `agents.Agent` interface

### AudioBuffers

Represents audio input/output buffers for a session.

**Attributes**:
- `InputBuffer` (chan []byte): Input audio buffer
- `OutputBuffer` (chan []byte): Output audio buffer
- `BufferSize` (int): Buffer size in bytes
- `Format` (AudioFormat): Audio format (sample rate, channels, codec)

**Relationships**:
- Belongs to one `VoiceSession` (1:1)

**Validation Rules**:
- `BufferSize` must be > 0
- `Format` must be valid

### TurnDetectionState

Represents turn detection state for a session.

**Attributes**:
- `IsListening` (bool): Currently listening for user speech
- `LastSpeechTime` (time.Time): Last detected speech timestamp
- `SilenceTimeout` (time.Duration): Silence timeout for turn end detection
- `TurnEndDetected` (bool): Turn end detected flag

**Relationships**:
- Belongs to one `VoiceSession` (1:1)

**Validation Rules**:
- `SilenceTimeout` must be > 0

### CustomProcessor

Represents a custom audio processor for extensibility.

**Attributes**:
- `Name` (string): Processor name
- `Processor` (AudioProcessor): Processor implementation
- `Order` (int): Processing order (lower = earlier)
- `Enabled` (bool): Whether processor is enabled

**Relationships**:
- Used by `PipelineConfiguration` (N:1)

**Validation Rules**:
- `Name` must be unique within pipeline
- `Processor` must implement `AudioProcessor` interface
- `Order` must be >= 0

## Interfaces

### VoiceBackend Interface

```go
type VoiceBackend interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Session Management
    CreateSession(ctx context.Context, config *SessionConfig) (*VoiceSession, error)
    GetSession(ctx context.Context, sessionID string) (*VoiceSession, error)
    ListSessions(ctx context.Context) ([]*VoiceSession, error)
    CloseSession(ctx context.Context, sessionID string) error
    
    // Health & Status
    HealthCheck(ctx context.Context) (*HealthStatus, error)
    GetConnectionState() ConnectionState
    GetActiveSessionCount() int
    
    // Configuration
    GetConfig() *Config
    UpdateConfig(ctx context.Context, config *Config) error
}
```

### BackendProvider Interface

```go
type BackendProvider interface {
    // Provider Info
    GetName() string
    GetCapabilities(ctx context.Context) (*ProviderCapabilities, error)
    
    // Backend Creation
    CreateBackend(ctx context.Context, config *Config) (VoiceBackend, error)
    
    // Configuration
    ValidateConfig(ctx context.Context, config *Config) error
    GetConfigSchema() *ConfigSchema
}
```

### VoiceSession Interface

```go
type VoiceSession interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Audio Processing
    ProcessAudio(ctx context.Context, audio []byte) error
    SendAudio(ctx context.Context, audio []byte) error
    ReceiveAudio() <-chan []byte
    
    // Agent Integration
    SetAgentCallback(callback func(context.Context, string) (string, error)) error
    SetAgentInstance(agent agents.Agent) error
    
    // State Management
    GetState() PipelineState
    GetPersistenceStatus() PersistenceStatus
    UpdateMetadata(metadata map[string]any) error
}
```

## Data Flow

### Session Creation Flow

```
1. User requests session creation
2. Backend validates configuration (FR-016)
3. Backend creates VoiceSession instance
4. Session initializes pipeline (STT/TTS or S2S)
5. Session establishes user connection (WebRTC/WebSocket)
6. Session starts listening for audio
7. Session state persisted if active (FR-025)
```

### Audio Processing Flow (STT/TTS)

```
1. Audio received from user connection
2. Audio buffered in AudioBuffers
3. VAD detects voice activity (if enabled)
4. Turn detection identifies speech boundaries
5. Audio sent to STT provider
6. Transcript received from STT
7. Transcript routed to agent (via callback or instance)
8. Agent generates response
9. Response sent to TTS provider
10. Audio generated from TTS
11. Audio sent to user connection
```

### Audio Processing Flow (S2S)

```
1. Audio received from user connection
2. Audio buffered in AudioBuffers
3. Audio routed to agent (if agent integration enabled)
4. Agent processes audio (optional reasoning)
5. Audio sent to S2S provider
6. Audio output received from S2S
7. Audio sent to user connection
```

## State Management

### Backend Connection States

- `Disconnected`: Backend not connected
- `Connecting`: Backend connecting to provider service
- `Connected`: Backend connected and ready
- `Reconnecting`: Backend reconnecting after error
- `Error`: Backend in error state

### Session Pipeline States

- `Idle`: Session idle, not processing
- `Listening`: Listening for user speech
- `Processing`: Processing audio/transcript
- `Speaking`: Playing agent response
- `Error`: Session in error state

### Persistence Status

- `Active`: Session is active, state should persist (FR-025)
- `Completed`: Session completed, ephemeral (FR-026)

## Validation Rules Summary

1. **Backend Configuration**: Must pass provider-specific validation (FR-016)
2. **Session IDs**: Must be unique within backend
3. **Pipeline Configuration**: Must have valid provider names for enabled components
4. **Agent Integration**: Must have either agent instance or callback
5. **Audio Format**: Must match provider requirements (FR-019)
6. **Concurrent Sessions**: Must not exceed provider's `MaxConcurrentSessions` (FR-010)

## Relationships Diagram

```
BackendRegistry
    ├── BackendProvider (1:N)
    │       └── VoiceBackend (1:N)
    │               ├── PipelineConfiguration (1:1)
    │               └── VoiceSession (1:N)
    │                       ├── UserConnection (1:1)
    │                       ├── AgentIntegration (1:1)
    │                       ├── AudioBuffers (1:1)
    │                       └── TurnDetectionState (1:1)
    │
    └── CustomProcessor (N:1) → PipelineConfiguration
```
