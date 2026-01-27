# Voice Agent Package

The voiceagent package provides a simplified API for creating voice-enabled agents. It reduces the boilerplate typically required to set up STT, TTS, VAD, and LLM integration for conversational voice applications.

> **Note**: This package is a work in progress. For production use, please use the individual packages directly (stt, tts, vad, llms, voicesession).

## Features

- **Fluent Builder Pattern**: Chain configuration methods for easy voice agent setup
- **Provider Configuration**: Configure STT, TTS, VAD, and LLM providers by name
- **Memory Management**: Enable and configure conversation memory
- **System Prompt**: Set the system prompt for the voice agent

## Usage

### Basic Builder Usage

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/voiceagent"

// Create a new voice agent builder
builder := voiceagent.NewBuilder().
    WithSTT("deepgram").
    WithTTS("elevenlabs").
    WithVAD("silero").
    WithLLM("openai").
    WithMemory(true).
    WithMemorySize(100).
    WithSystemPrompt("You are a helpful voice assistant.")

// Access configuration
fmt.Println(builder.GetSTTProvider())    // "deepgram"
fmt.Println(builder.GetTTSProvider())    // "elevenlabs"
fmt.Println(builder.GetVADProvider())    // "silero"
fmt.Println(builder.GetLLMProvider())    // "openai"
fmt.Println(builder.IsMemoryEnabled())   // true
fmt.Println(builder.GetMemorySize())     // 100
fmt.Println(builder.GetSystemPrompt())   // "You are a helpful voice assistant."
```

## Configuration Options

### STT Provider
```go
builder.WithSTT("deepgram")  // Speech-to-text provider name
```

### TTS Provider
```go
builder.WithTTS("elevenlabs")  // Text-to-speech provider name
```

### VAD Provider
```go
builder.WithVAD("silero")  // Voice activity detection provider name
```

### LLM Provider
```go
builder.WithLLM("openai")  // Language model provider name
```

### Memory
```go
builder.WithMemory(true)       // Enable conversation memory
builder.WithMemorySize(100)    // Number of messages to retain (default: 50)
```

### System Prompt
```go
builder.WithSystemPrompt("You are a helpful assistant.")
```

## Default Values

- **Memory Size**: 50
- **Memory Enabled**: false

## Future API (Planned)

The intended future API will look like:

```go
agent, err := voiceagent.NewBuilder().
    WithSTT("deepgram").
    WithTTS("elevenlabs").
    WithVAD("silero").
    WithLLM("openai").
    WithMemory(true).
    Build(ctx)

session, err := agent.StartSession(ctx)
```

## Production Usage

For production use, compose the individual packages:

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/stt"
    "github.com/lookatitude/beluga-ai/pkg/tts"
    "github.com/lookatitude/beluga-ai/pkg/vad"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/voicesession"
)

// Create individual providers
sttProvider, err := stt.NewProvider(ctx, "deepgram", sttConfig)
ttsProvider, err := tts.NewProvider(ctx, "elevenlabs", ttsConfig)
vadProvider, err := vad.NewProvider(ctx, "silero", vadConfig)
llmProvider, err := llms.NewProvider(ctx, "openai", llmConfig)

// Create voice session
session, err := voicesession.New(
    voicesession.WithSTT(sttProvider),
    voicesession.WithTTS(ttsProvider),
    voicesession.WithVAD(vadProvider),
    voicesession.WithLLM(llmProvider),
)
```

## Supported Providers

### STT Providers
- Deepgram
- OpenAI Whisper
- Google Speech

### TTS Providers
- ElevenLabs
- OpenAI TTS
- Google TTS

### VAD Providers
- Silero
- WebRTC VAD

### LLM Providers
- OpenAI
- Anthropic
- Ollama
