# Data Model: Voice Agents

**Feature**: Voice Agents Framework  
**Date**: 2025-01-27  
**Status**: Design Complete

## Overview

This document defines the core entities, their relationships, validation rules, and state transitions for the Voice Agents feature.

---

## Core Entities

### 1. VoiceSession

**Purpose**: Represents a complete voice interaction session between a user and an AI agent.

**Fields**:
- `ID` (string): Unique session identifier
- `State` (SessionState): Current session state (listening, speaking, processing, away)
- `Agent` (iface.CompositeAgent): Beluga AI agent instance
- `STTProvider` (stt.STTProvider): Speech-to-text provider
- `TTSProvider` (tts.TTSProvider): Text-to-speech provider
- `VADProvider` (vad.VADProvider): Voice activity detection provider
- `TurnDetector` (turndetection.TurnDetector): Turn detection provider
- `Transport` (transport.Transport): Audio transport (WebRTC, WebSocket, etc.)
- `Memory` (memory.Memory): Conversation memory
- `Config` (VoiceSessionConfig): Session configuration
- `CreatedAt` (time.Time): Session creation timestamp
- `UpdatedAt` (time.Time): Last update timestamp
- `Metadata` (map[string]interface{}): Additional session metadata

**Relationships**:
- Has one Agent (belongs to `pkg/agents`)
- Has one STTProvider (belongs to `pkg/voice/stt`)
- Has one TTSProvider (belongs to `pkg/voice/tts`)
- Has one VADProvider (belongs to `pkg/voice/vad`)
- Has one TurnDetector (belongs to `pkg/voice/turndetection`)
- Has one Transport (belongs to `pkg/voice/transport`)
- Has one Memory (belongs to `pkg/memory`)
- Has many AudioStreams (one-to-many)
- Has many Transcripts (one-to-many)
- Has many AgentResponses (one-to-many)

**Validation Rules**:
- ID must be non-empty
- Agent must be non-nil
- At least one of STTProvider or TTSProvider must be non-nil
- Config must be valid (see VoiceSessionConfig validation)

**State Transitions**:
```
[Initial] → [Listening] → [Processing] → [Speaking] → [Listening]
     ↓           ↓             ↓            ↓
  [Away]     [Away]       [Away]       [Away]
     ↓           ↓             ↓            ↓
  [Ended]    [Ended]      [Ended]      [Ended]
```

**State Descriptions**:
- `Initial`: Session created but not started
- `Listening`: Waiting for user speech input
- `Processing`: Processing user input through agent
- `Speaking`: Playing agent response to user
- `Away`: User inactive (timeout)
- `Ended`: Session terminated

---

### 2. AudioStream

**Purpose**: Represents real-time audio input from the user or audio output to the user.

**Fields**:
- `ID` (string): Unique stream identifier
- `SessionID` (string): Associated voice session ID
- `Direction` (StreamDirection): Input or output
- `Format` (AudioFormat): Audio format specification
- `Data` ([]byte): Audio data (for synchronous operations)
- `Stream` (chan []byte): Audio stream channel (for streaming)
- `Timestamp` (time.Time): Stream timestamp
- `SequenceNumber` (int64): Sequence number for ordering
- `Metadata` (map[string]interface{}): Additional stream metadata

**Relationships**:
- Belongs to one VoiceSession (many-to-one)
- Produces many Transcripts (one-to-many, for input streams)

**Validation Rules**:
- ID must be non-empty
- SessionID must reference valid session
- Format must be valid (sample rate > 0, channels > 0)
- Data or Stream must be non-nil (but not both)

**Stream Direction**:
- `Input`: Audio from user (microphone)
- `Output`: Audio to user (speaker)

---

### 3. Transcript

**Purpose**: Represents converted speech-to-text data from STT processing.

**Fields**:
- `ID` (string): Unique transcript identifier
- `SessionID` (string): Associated voice session ID
- `StreamID` (string): Source audio stream ID
- `Text` (string): Transcribed text
- `IsFinal` (bool): Whether transcript is final or interim
- `Language` (string): Detected language (ISO 639-1)
- `Confidence` (float64): Confidence score (0.0-1.0)
- `Words` ([]WordTimestamp): Word-level timestamps
- `StartTime` (time.Time): Transcript start time
- `EndTime` (time.Time): Transcript end time
- `CreatedAt` (time.Time): Creation timestamp
- `Error` (error): Processing error (if any)

**Relationships**:
- Belongs to one VoiceSession (many-to-one)
- Belongs to one AudioStream (many-to-one, for input streams)
- Produces one AgentResponse (one-to-one)

**Validation Rules**:
- ID must be non-empty
- SessionID must reference valid session
- Text must be non-empty if IsFinal is true
- Confidence must be between 0.0 and 1.0
- Language must be valid ISO 639-1 code if provided

---

### 4. WordTimestamp

**Purpose**: Represents word-level timing information within a transcript.

**Fields**:
- `Word` (string): The word text
- `Start` (time.Duration): Start time offset from transcript start
- `End` (time.Duration): End time offset from transcript start
- `Confidence` (float64): Word confidence score (0.0-1.0)

**Validation Rules**:
- Word must be non-empty
- Start must be >= 0
- End must be > Start
- Confidence must be between 0.0 and 1.0

---

### 5. AgentResponse

**Purpose**: Represents the AI agent's text response that will be converted to speech.

**Fields**:
- `ID` (string): Unique response identifier
- `SessionID` (string): Associated voice session ID
- `TranscriptID` (string): Source transcript ID
- `Text` (string): Response text
- `IsStreaming` (bool): Whether response is streaming
- `Stream` (chan string): Response stream channel (for streaming)
- `Metadata` (map[string]interface{}): Response metadata (tools used, etc.)
- `CreatedAt` (time.Time): Creation timestamp
- `CompletedAt` (time.Time): Completion timestamp (if completed)

**Relationships**:
- Belongs to one VoiceSession (many-to-one)
- Belongs to one Transcript (many-to-one)
- Produces one AudioStream (one-to-one, for output)

**Validation Rules**:
- ID must be non-empty
- SessionID must reference valid session
- Text must be non-empty if IsStreaming is false
- Stream must be non-nil if IsStreaming is true

---

### 6. VoiceSessionConfig

**Purpose**: Configuration for a voice agent session.

**Fields**:
- `Agent` (iface.CompositeAgent): Beluga AI agent (required)
- `STT` (stt.STTProvider): STT provider (required)
- `TTS` (tts.TTSProvider): TTS provider (required)
- `VAD` (vad.VADProvider): VAD provider (optional)
- `TurnDetector` (turndetection.TurnDetector): Turn detector (optional)
- `Transport` (transport.Transport): Transport provider (required)
- `Memory` (memory.Memory): Memory provider (optional, recommended)
- `Options` (*VoiceOptions): Voice interaction options
- `Logger` (*zap.Logger): Logger instance (optional)

**Validation Rules**:
- Agent must be non-nil
- STT must be non-nil
- TTS must be non-nil
- Transport must be non-nil
- Options must be valid if provided

---

### 7. VoiceOptions

**Purpose**: Configuration for voice interaction behavior.

**Fields**:
- `AllowInterruptions` (bool): Allow user to interrupt agent
- `DiscardAudioIfUninterruptible` (bool): Discard audio during uninterruptible responses
- `MinInterruptionDuration` (time.Duration): Minimum interruption duration
- `MinInterruptionWords` (int): Minimum words to trigger interruption
- `MinEndpointingDelay` (time.Duration): Minimum delay before endpointing
- `MaxEndpointingDelay` (time.Duration): Maximum delay before endpointing
- `MaxToolSteps` (int): Maximum tool execution steps
- `PreemptiveGeneration` (bool): Start generation on interim transcripts
- `StreamingTTS` (bool): Enable streaming TTS
- `StreamingAgent` (bool): Enable streaming agent responses
- `UserAwayTimeout` (time.Duration): Timeout for user away detection

**Validation Rules**:
- MinInterruptionDuration must be >= 0
- MinInterruptionWords must be >= 0
- MinEndpointingDelay must be >= 0
- MaxEndpointingDelay must be >= MinEndpointingDelay
- MaxToolSteps must be > 0
- UserAwayTimeout must be > 0

---

### 8. AudioFormat

**Purpose**: Represents audio format requirements.

**Fields**:
- `SampleRate` (int): Sample rate in Hz (e.g., 16000, 48000)
- `Channels` (int): Number of channels (1 for mono, 2 for stereo)
- `BitDepth` (int): Bit depth (16, 24, 32)
- `Encoding` (string): Encoding format ("pcm", "opus", "mp3", etc.)

**Validation Rules**:
- SampleRate must be > 0 (common: 8000, 16000, 48000)
- Channels must be 1 or 2
- BitDepth must be 16, 24, or 32
- Encoding must be non-empty

---

## Entity Relationships Diagram

```
VoiceSession
    │
    ├─── has one ───> Agent (pkg/agents)
    ├─── has one ───> STTProvider (pkg/voice/stt)
    ├─── has one ───> TTSProvider (pkg/voice/tts)
    ├─── has one ───> VADProvider (pkg/voice/vad)
    ├─── has one ───> TurnDetector (pkg/voice/turndetection)
    ├─── has one ───> Transport (pkg/voice/transport)
    ├─── has one ───> Memory (pkg/memory)
    │
    ├─── has many ───> AudioStream
    │       │
    │       └─── produces ───> Transcript (for input streams)
    │
    ├─── has many ───> Transcript
    │       │
    │       └─── produces ───> AgentResponse
    │
    └─── has many ───> AgentResponse
            │
            └─── produces ───> AudioStream (for output)
```

---

## State Management

### VoiceSession State Machine

```
[Initial State]
    │
    ├── Start() ───> [Listening]
    │
    └── Start() with error ───> [Error State]

[Listening State]
    │
    ├── User speaks ───> [Processing]
    │
    ├── UserAwayTimeout ───> [Away]
    │
    └── Stop() ───> [Ended]

[Processing State]
    │
    ├── Agent responds ───> [Speaking]
    │
    ├── Error ───> [Listening] (with error handling)
    │
    └── Stop() ───> [Ended]

[Speaking State]
    │
    ├── User interrupts ───> [Listening] (if AllowInterruptions)
    │
    ├── Response complete ───> [Listening]
    │
    └── Stop() ───> [Ended]

[Away State]
    │
    ├── User returns ───> [Listening]
    │
    └── Stop() ───> [Ended]

[Ended State]
    │
    └── (terminal state, no transitions)
```

---

## Data Flow

### Input Flow (User Speech → Agent Response)

```
1. AudioStream (Input) 
   └──> VADProvider.Process() [detect speech]
   └──> STTProvider.Transcribe() or StartStreaming()
   └──> Transcript (interim or final)
   └──> TurnDetector.DetectTurn() [detect turn end]
   └──> VoiceSession.ProcessTranscript()
   └──> Memory.SaveContext() [save to memory]
   └──> Agent.Run() or RunStream()
   └──> AgentResponse
   └──> TTSProvider.GenerateSpeech() or StreamGenerate()
   └──> AudioStream (Output)
```

### Streaming Flow

```
1. AudioStream (Input, streaming)
   └──> STTProvider.StartStreaming()
   └──> TranscriptCallback (interim transcripts)
   └──> [If PreemptiveGeneration] Agent.RunStream() (on interim)
   └──> TranscriptCallback (final transcript)
   └──> Agent.Run() or RunStream() (on final)
   └──> AgentResponse (streaming)
   └──> TTSProvider.StreamGenerate()
   └──> AudioStream (Output, streaming)
```

---

## Validation Summary

### Required Validations

1. **VoiceSession**:
   - All required providers must be non-nil
   - Config must be valid
   - State transitions must be valid

2. **AudioStream**:
   - Format must be valid
   - SessionID must reference existing session
   - Data or Stream must be provided (not both)

3. **Transcript**:
   - Text must be non-empty if final
   - Confidence must be in valid range
   - Language must be valid ISO code

4. **AgentResponse**:
   - Text or Stream must be provided
   - SessionID must reference existing session

5. **Config Objects**:
   - All required fields must be set
   - Numeric fields must be in valid ranges
   - Duration fields must be positive

---

## Integration Points

### With Existing Beluga AI Packages

1. **pkg/agents**: VoiceSession uses `iface.CompositeAgent`
2. **pkg/memory**: VoiceSession uses `memory.Memory` for conversation history
3. **pkg/config**: Voice providers use config package for credentials
4. **pkg/llms**: Agents use LLM interfaces (via agents package)
5. **pkg/prompts**: Agents use prompt templates (via agents package)
6. **pkg/monitoring**: All voice operations emit OTEL metrics/traces

---

## Notes

- All entities support context cancellation
- All entities are thread-safe where applicable
- All entities implement proper resource cleanup
- All entities emit observability events (metrics, traces, logs)
- All entities follow error handling patterns (Op/Err/Code)

