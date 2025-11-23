# Voice Session API Contract

**Feature**: Voice Agents  
**Date**: 2025-01-27  
**Type**: Go Interface Contract

## Overview

This document defines the API contract for the Voice Session package, following Beluga AI framework patterns.

---

## Core Interface: VoiceSession

```go
// VoiceSession manages a complete voice interaction session
type VoiceSession interface {
    // Start starts the voice session
    Start(ctx context.Context) error
    
    // Stop stops the voice session gracefully
    Stop(ctx context.Context) error
    
    // Say converts text to speech and plays it
    Say(ctx context.Context, text string) (*SayHandle, error)
    
    // SayWithOptions converts text to speech with options
    SayWithOptions(ctx context.Context, text string, options SayOptions) (*SayHandle, error)
    
    // ProcessAudio processes incoming audio data
    ProcessAudio(ctx context.Context, audio []byte) error
    
    // GetState returns the current session state
    GetState() SessionState
    
    // OnStateChanged sets callback for state changes
    OnStateChanged(callback func(SessionState))
    
    // GetSessionID returns the session identifier
    GetSessionID() string
}
```

---

## Supporting Types

### SessionState

```go
type SessionState string

const (
    SessionStateInitial    SessionState = "initial"
    SessionStateListening  SessionState = "listening"
    SessionStateProcessing SessionState = "processing"
    SessionStateSpeaking   SessionState = "speaking"
    SessionStateAway       SessionState = "away"
    SessionStateEnded      SessionState = "ended"
)
```

### SayOptions

```go
type SayOptions struct {
    AllowInterruptions bool
    Voice              string
    Speed              float64  // 0.5-2.0, default: 1.0
    Volume             float64  // 0.0-1.0, default: 1.0
}
```

### SayHandle

```go
type SayHandle interface {
    // WaitForPlayout waits for audio to finish playing
    WaitForPlayout(ctx context.Context) error
    
    // Cancel cancels the Say operation
    Cancel() error
}
```

---

## Factory Function

```go
// NewVoiceSession creates a new voice session with the given configuration
func NewVoiceSession(ctx context.Context, config VoiceSessionConfig) (VoiceSession, error)
```

### VoiceSessionConfig

```go
type VoiceSessionConfig struct {
    Agent        iface.CompositeAgent
    STT          stt.STTProvider
    TTS          tts.TTSProvider
    VAD          vad.VADProvider          // Optional
    TurnDetector turndetection.TurnDetector // Optional
    Transport    transport.Transport
    Memory       memory.Memory            // Optional, recommended
    Options      *VoiceOptions            // Optional
    Logger       *zap.Logger              // Optional
}
```

---

## Contract Requirements

### Start(ctx context.Context) error

**Preconditions**:
- Session must be in `Initial` state
- All required providers (Agent, STT, TTS, Transport) must be non-nil
- Config must be valid

**Postconditions**:
- Session state transitions to `Listening`
- Audio transport is established
- VAD and STT providers are ready
- OnStateChanged callback is invoked with `SessionStateListening`

**Error Conditions**:
- `ErrCodeInvalidState`: Session not in Initial state
- `ErrCodeInvalidConfig`: Configuration is invalid
- `ErrCodeTransportFailed`: Failed to establish transport connection
- `ErrCodeProviderInitFailed`: Provider initialization failed

**Observability**:
- Emit metric: `voice.session.start` (counter)
- Emit trace: `voice.session.start` span
- Log: Info level with session ID

---

### Stop(ctx context.Context) error

**Preconditions**:
- Session must be in any state except `Ended`

**Postconditions**:
- Session state transitions to `Ended`
- All audio streams are closed
- All providers are cleaned up
- Resources are released
- OnStateChanged callback is invoked with `SessionStateEnded`

**Error Conditions**:
- `ErrCodeInvalidState`: Session already ended
- `ErrCodeCleanupFailed`: Resource cleanup failed

**Observability**:
- Emit metric: `voice.session.stop` (counter)
- Emit trace: `voice.session.stop` span
- Log: Info level with session ID and duration

---

### Say(ctx context.Context, text string) (*SayHandle, error)

**Preconditions**:
- Session must be in `Listening` or `Speaking` state
- Text must be non-empty

**Postconditions**:
- If in `Listening`: State transitions to `Speaking`
- Text is converted to speech via TTS provider
- Audio is played via transport
- SayHandle is returned for control

**Error Conditions**:
- `ErrCodeInvalidState`: Session not in valid state
- `ErrCodeEmptyText`: Text is empty
- `ErrCodeTTSFailed`: TTS generation failed
- `ErrCodeTransportFailed`: Audio playback failed

**Observability**:
- Emit metric: `voice.session.say` (counter, histogram for duration)
- Emit trace: `voice.session.say` span with text length
- Log: Debug level with text preview

---

### SayWithOptions(ctx context.Context, text string, options SayOptions) (*SayHandle, error)

**Preconditions**:
- Same as `Say()`
- Options must be valid (Speed 0.5-2.0, Volume 0.0-1.0)

**Postconditions**:
- Same as `Say()` but with options applied

**Error Conditions**:
- Same as `Say()`
- `ErrCodeInvalidOptions`: Options are invalid

---

### ProcessAudio(ctx context.Context, audio []byte) error

**Preconditions**:
- Session must be in `Listening` or `Processing` state
- Audio must be non-empty
- Audio format must match session configuration

**Postconditions**:
- Audio is processed by VAD (if configured)
- If speech detected: Audio is sent to STT provider
- If turn detected: Transcript is processed by agent
- State may transition to `Processing` if agent is invoked

**Error Conditions**:
- `ErrCodeInvalidState`: Session not in valid state
- `ErrCodeInvalidAudio`: Audio format is invalid
- `ErrCodeSTTFailed`: STT processing failed
- `ErrCodeAgentFailed`: Agent processing failed

**Observability**:
- Emit metric: `voice.session.process_audio` (counter, histogram for audio size)
- Emit trace: `voice.session.process_audio` span
- Log: Debug level with audio size

---

### GetState() SessionState

**Preconditions**: None

**Postconditions**: Returns current session state

**Error Conditions**: None (always succeeds)

---

### OnStateChanged(callback func(SessionState))

**Preconditions**: None

**Postconditions**: Callback is registered for state change events

**Error Conditions**: None (always succeeds)

**Note**: Callback is invoked synchronously on state transitions

---

## Error Codes

```go
const (
    ErrCodeInvalidState        = "invalid_state"
    ErrCodeInvalidConfig       = "invalid_config"
    ErrCodeTransportFailed     = "transport_failed"
    ErrCodeProviderInitFailed  = "provider_init_failed"
    ErrCodeCleanupFailed       = "cleanup_failed"
    ErrCodeEmptyText           = "empty_text"
    ErrCodeTTSFailed           = "tts_failed"
    ErrCodeInvalidOptions      = "invalid_options"
    ErrCodeInvalidAudio        = "invalid_audio"
    ErrCodeSTTFailed           = "stt_failed"
    ErrCodeAgentFailed         = "agent_failed"
)
```

---

## Testing Contracts

### Unit Test Requirements

1. **Start() Tests**:
   - Valid start from Initial state
   - Invalid state error
   - Invalid config error
   - Transport failure error
   - Provider init failure error

2. **Stop() Tests**:
   - Valid stop from any state
   - Already ended error
   - Cleanup failure error

3. **Say() Tests**:
   - Valid say from Listening state
   - Valid say from Speaking state (interruption)
   - Invalid state error
   - Empty text error
   - TTS failure error

4. **ProcessAudio() Tests**:
   - Valid audio processing
   - Invalid state error
   - Invalid audio format error
   - STT failure error
   - Agent failure error

5. **State Machine Tests**:
   - All valid state transitions
   - Invalid state transitions
   - State change callbacks

---

## Integration Contracts

### With pkg/agents

- VoiceSession uses `iface.CompositeAgent` interface
- Agent.Run() or RunStream() is called with transcript text
- Agent responses are converted to speech

### With pkg/memory

- VoiceSession uses `memory.Memory` interface
- Transcripts are saved via Memory.SaveContext()
- Agent context is retrieved via Memory.GetContext()

### With pkg/voice/stt

- VoiceSession uses `stt.STTProvider` interface
- STT.Transcribe() or StartStreaming() is called
- Transcripts are received via callbacks

### With pkg/voice/tts

- VoiceSession uses `tts.TTSProvider` interface
- TTS.GenerateSpeech() or StreamGenerate() is called
- Audio chunks are received for playback

### With pkg/voice/transport

- VoiceSession uses `transport.Transport` interface
- Transport.SendAudio() is called for output
- Transport.OnAudioReceived() callback receives input

---

## Performance Contracts

- **Start()**: Must complete within 1 second
- **Stop()**: Must complete within 500ms
- **Say()**: Must start playback within 100ms of call
- **ProcessAudio()**: Must process within 50ms (not including STT/agent time)
- **State transitions**: Must complete within 10ms

---

## Observability Contracts

All operations MUST emit:
- **Metrics**: Counter and/or histogram with operation name
- **Traces**: Span with operation name and relevant attributes
- **Logs**: Structured log with session ID, operation, and result

Metric names follow pattern: `voice.session.{operation}`
Trace names follow pattern: `voice.session.{operation}`
Log fields: `session_id`, `operation`, `state`, `error` (if any)

