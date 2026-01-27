# Voice Agent Package

The voiceagent package provides a simplified API for creating voice-enabled agents. It reduces the boilerplate typically required to set up STT, TTS, VAD, and LLM integration for conversational voice applications.

## Features

- **Fluent Builder Pattern**: Chain configuration methods for easy voice agent setup
- **Modular Components**: STT, TTS, VAD, and LLM are independently configurable
- **Session Management**: Create and manage voice conversation sessions
- **Memory Integration**: Built-in conversation memory support
- **Callback Support**: Hooks for transcript, response, and error events
- **OpenTelemetry Integration**: Full observability with metrics and tracing
- **Structured Errors**: Op/Err/Code error pattern for clear error handling

## Installation

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/voiceagent"
```

## Quick Start

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/convenience/voiceagent"
)

// Create STT and TTS providers
sttProvider := createSTTProvider()
ttsProvider := createTTSProvider()

// Build the voice agent
agent, err := voiceagent.NewBuilder().
    WithSTTInstance(sttProvider).
    WithTTSInstance(ttsProvider).
    WithSystemPrompt("You are a helpful voice assistant.").
    Build(ctx)
if err != nil {
    log.Fatal(err)
}
defer agent.Shutdown()

// Process audio
responseAudio, err := agent.ProcessAudio(ctx, audioData)
```

## Builder API

### Creating a Builder

```go
builder := voiceagent.NewBuilder()
```

### Configuration Methods

#### STT Configuration (Required)

```go
// Use an STT provider instance
builder.WithSTTInstance(sttProvider)

// Use provider-based resolution (not yet implemented)
builder.WithSTT("deepgram")
```

#### TTS Configuration (Required)

```go
// Use a TTS provider instance
builder.WithTTSInstance(ttsProvider)

// Use provider-based resolution (not yet implemented)
builder.WithTTS("elevenlabs")
```

#### VAD Configuration (Optional)

```go
// Use a VAD provider instance
builder.WithVADInstance(vadProvider)

// Use provider-based resolution (not yet implemented)
builder.WithVAD("silero")
```

#### LLM Configuration (Optional)

```go
// Use an LLM instance for intelligent responses
builder.WithLLMInstance(chatModel)

// Use provider-based resolution (not yet implemented)
builder.WithLLM("openai")
```

#### Memory Configuration

```go
// Enable memory with size
builder.WithMemory(true)
builder.WithMemorySize(100)  // Number of messages to retain (default: 50)

// Use a pre-configured memory instance
builder.WithMemoryInstance(customMemory)
```

#### System Prompt

```go
builder.WithSystemPrompt("You are a helpful voice assistant.")
```

#### Timeout

```go
builder.WithTimeout(60 * time.Second)  // Default: 30s
```

#### Callbacks

```go
// Called when transcript is received
builder.WithOnTranscript(func(text string, isFinal bool) {
    if isFinal {
        fmt.Println("User said:", text)
    }
})

// Called when response is generated
builder.WithOnResponse(func(text string) {
    fmt.Println("Assistant:", text)
})

// Called on errors
builder.WithOnError(func(err error) {
    log.Println("Error:", err)
})
```

#### Metrics Configuration

```go
builder.WithMetrics(customMetrics)
```

### Building the Agent

```go
agent, err := builder.Build(ctx)
if err != nil {
    // Handle error
}
defer agent.Shutdown()
```

## VoiceAgent Interface

The built agent implements the `VoiceAgent` interface:

```go
type VoiceAgent interface {
    StartSession(ctx context.Context) (Session, error)
    ProcessAudio(ctx context.Context, audio []byte) ([]byte, error)
    ProcessText(ctx context.Context, text string) (string, error)
    GetSTT() voiceutilsiface.STTProvider
    GetTTS() voiceutilsiface.TTSProvider
    GetVAD() voiceutilsiface.VADProvider
    Shutdown() error
}
```

### Processing Audio

```go
// Process audio and get audio response
responseAudio, err := agent.ProcessAudio(ctx, audioData)
if err != nil {
    log.Fatal(err)
}
playAudio(responseAudio)
```

### Processing Text

```go
// Process text and get text response (bypasses STT)
response, err := agent.ProcessText(ctx, "Hello, how are you?")
fmt.Println("Response:", response)
```

### Session Management

```go
// Start a new session
session, err := agent.StartSession(ctx)
if err != nil {
    log.Fatal(err)
}
defer session.Stop()

// Start the session
err = session.Start(ctx)

// Send audio to the session
err = session.SendAudio(ctx, audioChunk)

// Receive audio responses
for audio := range session.ReceiveAudio() {
    playAudio(audio)
}

// Get the transcript
transcript := session.GetTranscript()
```

### Accessing Components

```go
stt := agent.GetSTT()
tts := agent.GetTTS()
vad := agent.GetVAD()  // May be nil if not configured
```

## Session Interface

```go
type Session interface {
    ID() string
    Start(ctx context.Context) error
    Stop() error
    SendAudio(ctx context.Context, audio []byte) error
    ReceiveAudio() <-chan []byte
    GetTranscript() string
    IsActive() bool
}
```

## Error Handling

The package uses structured errors with Op/Err/Code pattern:

```go
agent, err := builder.Build(ctx)
if err != nil {
    var vaErr *voiceagent.Error
    if errors.As(err, &vaErr) {
        switch vaErr.Code {
        case voiceagent.ErrCodeMissingSTT:
            // No STT provider configured
        case voiceagent.ErrCodeMissingTTS:
            // No TTS provider configured
        case voiceagent.ErrCodeSTTCreation:
            // Failed to create STT from provider name
        case voiceagent.ErrCodeTTSCreation:
            // Failed to create TTS from provider name
        }
    }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `missing_stt` | No STT provider configured |
| `missing_tts` | No TTS provider configured |
| `stt_creation_failed` | Failed to create STT from provider name |
| `tts_creation_failed` | Failed to create TTS from provider name |
| `vad_creation_failed` | Failed to create VAD from provider name |
| `agent_creation_failed` | Failed to create underlying agent |
| `memory_creation_failed` | Failed to create memory |
| `session_creation_failed` | Failed to create session |
| `transcription_failed` | STT transcription failed |
| `synthesis_failed` | TTS synthesis failed |

## Observability

The package includes OpenTelemetry metrics and tracing:

```go
// Get global metrics instance
metrics := voiceagent.GetMetrics()

// Create custom metrics
metrics, err := voiceagent.NewMetrics("custom-prefix")

// Use no-op metrics (for testing)
metrics := voiceagent.NoOpMetrics()
```

### Metrics Recorded

- `voiceagent_builds_total` - Counter for build operations
- `voiceagent_build_duration_seconds` - Histogram for build duration
- `voiceagent_sessions_total` - Counter for session operations
- `voiceagent_session_duration_seconds` - Histogram for session duration
- `voiceagent_transcriptions_total` - Counter for transcriptions
- `voiceagent_transcription_duration_seconds` - Histogram for transcription duration
- `voiceagent_syntheses_total` - Counter for TTS syntheses
- `voiceagent_synthesis_duration_seconds` - Histogram for synthesis duration
- `voiceagent_errors_total` - Counter for errors by type

## Examples

### Basic Voice Agent

```go
stt := createSTTProvider()
tts := createTTSProvider()

agent, err := voiceagent.NewBuilder().
    WithSTTInstance(stt).
    WithTTSInstance(tts).
    Build(ctx)

// Process audio directly
response, _ := agent.ProcessAudio(ctx, audioInput)
```

### Voice Agent with LLM

```go
stt := createSTTProvider()
tts := createTTSProvider()
llm := createChatModel()

agent, err := voiceagent.NewBuilder().
    WithSTTInstance(stt).
    WithTTSInstance(tts).
    WithLLMInstance(llm).
    WithSystemPrompt("You are a helpful assistant. Be concise in voice responses.").
    WithMemory(true).
    WithMemorySize(20).
    Build(ctx)

// Now ProcessAudio will use the LLM to generate intelligent responses
response, _ := agent.ProcessAudio(ctx, audioInput)
```

### Voice Agent with VAD

```go
stt := createSTTProvider()
tts := createTTSProvider()
vad := createVADProvider()

agent, err := voiceagent.NewBuilder().
    WithSTTInstance(stt).
    WithTTSInstance(tts).
    WithVADInstance(vad).
    Build(ctx)
```

### Voice Agent with Callbacks

```go
agent, err := voiceagent.NewBuilder().
    WithSTTInstance(stt).
    WithTTSInstance(tts).
    WithOnTranscript(func(text string, isFinal bool) {
        if isFinal {
            log.Printf("Transcription: %s", text)
        }
    }).
    WithOnResponse(func(text string) {
        log.Printf("Response: %s", text)
    }).
    WithOnError(func(err error) {
        log.Printf("Error: %v", err)
    }).
    Build(ctx)
```

### Interactive Session

```go
agent, _ := voiceagent.NewBuilder().
    WithSTTInstance(stt).
    WithTTSInstance(tts).
    WithLLMInstance(llm).
    Build(ctx)

session, _ := agent.StartSession(ctx)
defer session.Stop()

// Start receiving responses in background
go func() {
    for audio := range session.ReceiveAudio() {
        playAudio(audio)
    }
}()

// Start the session
session.Start(ctx)

// Send audio chunks as they arrive
for chunk := range audioInputStream {
    session.SendAudio(ctx, chunk)
}
```

## Default Values

| Option | Default |
|--------|---------|
| Memory Size | 50 |
| Memory Enabled | false |
| Timeout | 30 seconds |

## Thread Safety

The built agent is safe for concurrent use. Sessions should be used by a single goroutine, but the agent can create multiple sessions concurrently.

## Supported Providers

### STT Providers
- Deepgram
- OpenAI Whisper
- Google Speech
- Azure Speech

### TTS Providers
- ElevenLabs
- OpenAI TTS
- Google TTS
- Azure Speech

### VAD Providers
- Silero
- WebRTC VAD

### LLM Providers
- OpenAI
- Anthropic
- Ollama
- Google Gemini

## See Also

- [pkg/stt](../../stt/) - Speech-to-text providers
- [pkg/tts](../../tts/) - Text-to-speech providers
- [pkg/vad](../../vad/) - Voice activity detection
- [pkg/llms](../../llms/) - LLM providers
- [pkg/voicesession](../../voicesession/) - Voice session management
- [pkg/memory](../../memory/) - Memory implementations
