---
title: "Voice S2S"
description: "Speech-to-speech interface and providers: OpenAI Realtime, Gemini Live, Nova S2S"
---

## s2s

```go
import "github.com/lookatitude/beluga-ai/voice/s2s"
```

Package s2s provides the speech-to-speech (S2S) interface and provider
registry for the Beluga AI voice pipeline. S2S providers handle native
audio-in/audio-out via their own transport (WebRTC, WebSocket), bypassing
the STT → LLM → TTS cascade for lower latency.

## Core Interface

The `S2S` interface provides a single method to start a bidirectional audio
session:

```go
type S2S interface {
    Start(ctx context.Context, opts ...Option) (Session, error)
}
```

The `Session` interface represents an active bidirectional audio connection:

```go
type Session interface {
    SendAudio(ctx context.Context, audio []byte) error
    SendText(ctx context.Context, text string) error
    SendToolResult(ctx context.Context, result schema.ToolResult) error
    Recv() <-chan SessionEvent
    Interrupt(ctx context.Context) error
    Close() error
}
```

## Session Events

Events received from the session channel are typed by `SessionEventType`:

- [EventAudioOutput] — model-generated audio
- [EventTextOutput] — model-generated text
- [EventTranscript] — user speech transcript
- [EventToolCall] — tool invocation request
- [EventTurnEnd] — end of conversational turn
- [EventError] — error occurred

## Registry Pattern

Providers register via `Register` in their init() function and are created
with `New`. Use `List` to discover available providers.

```go
import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"

engine, err := s2s.New("openai_realtime", s2s.Config{Voice: "alloy"})
session, err := engine.Start(ctx)
defer session.Close()

session.SendAudio(ctx, audioChunk)
for event := range session.Recv() {
    switch event.Type {
    case s2s.EventAudioOutput:
        playAudio(event.Audio)
    case s2s.EventToolCall:
        handleToolCall(event.ToolCall)
    }
}
```

## Frame Processor Integration

Use `AsFrameProcessor` to wrap an S2S engine as a voice.FrameProcessor for
integration with the cascading or hybrid pipeline.

## Hooks

The `Hooks` struct provides callbacks for S2S-specific events: OnTurn,
OnInterrupt, OnToolCall, and OnError. Use `ComposeHooks` to merge hooks.

## Available Providers

- openai_realtime — OpenAI Realtime API (voice/s2s/providers/openai)
- gemini_live — Google Gemini Live API (voice/s2s/providers/gemini)
- nova — Amazon Nova Sonic via Bedrock (voice/s2s/providers/nova)

---

## gemini

```go
import "github.com/lookatitude/beluga-ai/voice/s2s/providers/gemini"
```

Package gemini provides the Gemini Live S2S provider for the Beluga AI voice
pipeline. It uses the Google Gemini Live API via WebSocket for bidirectional
audio streaming with support for text, audio, and tool call events.

## Registration

This package registers itself as "gemini_live" with the s2s registry. Import
it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/gemini"
```

## Usage

```go
engine, err := s2s.New("gemini_live", s2s.Config{
    Model: "gemini-2.0-flash-exp",
    Extra: map[string]any{"api_key": "..."},
})
session, err := engine.Start(ctx)
defer session.Close()
```

## Configuration

Required configuration in Config.Extra:

- api_key — Google AI API key (required)
- base_url — Custom WebSocket endpoint (optional, defaults to Gemini Live production URL)

The default model is "gemini-2.0-flash-exp". Voice, instructions, and tools
are passed through [s2s.Config] fields.

## Exported Types

- [Engine] — implements s2s.S2S using Gemini Live
- [New] — constructor accepting s2s.Config

---

## nova

```go
import "github.com/lookatitude/beluga-ai/voice/s2s/providers/nova"
```

Package nova provides the Amazon Nova S2S provider for the Beluga AI voice
pipeline. It uses the AWS Bedrock Runtime API for bidirectional audio
streaming with Amazon Nova Sonic.

## Registration

This package registers itself as "nova" with the s2s registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/nova"
```

## Usage

```go
engine, err := s2s.New("nova", s2s.Config{
    Model: "amazon.nova-sonic-v1:0",
    Extra: map[string]any{"region": "us-east-1"},
})
session, err := engine.Start(ctx)
defer session.Close()
```

## Configuration

Configuration in Config.Extra:

- region — AWS region (optional, defaults to "us-east-1")
- base_url — Custom WebSocket endpoint (optional, defaults to Bedrock Runtime URL)

The default model is "amazon.nova-sonic-v1:0". Instructions and tools are
passed through [s2s.Config] fields.

## Exported Types

- [Engine] — implements s2s.S2S using Amazon Nova via Bedrock
- [New] — constructor accepting s2s.Config

---

## openai

```go
import "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
```

Package openai provides the OpenAI Realtime S2S provider for the Beluga AI
voice pipeline. It uses the OpenAI Realtime API via WebSocket for
bidirectional audio streaming with support for text, audio, tool calls,
and server-side VAD.

## Registration

This package registers itself as "openai_realtime" with the s2s registry.
Import it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
```

## Usage

```go
engine, err := s2s.New("openai_realtime", s2s.Config{
    Voice: "alloy",
    Model: "gpt-4o-realtime-preview",
    Extra: map[string]any{"api_key": "sk-..."},
})
session, err := engine.Start(ctx)
defer session.Close()
```

## Configuration

Required configuration in Config.Extra:

- api_key — OpenAI API key (required)
- base_url — Custom WebSocket endpoint (optional, defaults to wss://api.openai.com/v1/realtime)

The default model is "gpt-4o-realtime-preview" and the default voice is
"alloy". Instructions and tools are passed through [s2s.Config] fields.
Audio uses PCM16 format with server-side VAD for turn detection.

## Exported Types

- [Engine] — implements s2s.S2S using OpenAI Realtime
- [New] — constructor accepting s2s.Config
