---
title: "Voice TTS"
description: "Text-to-speech interface and providers: ElevenLabs, Cartesia, PlayHT, Fish, Groq, LMNT, Smallest"
---

## tts

```go
import "github.com/lookatitude/beluga-ai/voice/tts"
```

Package tts provides the text-to-speech (TTS) interface and provider registry
for the Beluga AI voice pipeline. Providers implement the `TTS` interface and
register themselves via init() for discovery.

## Core Interface

The `TTS` interface supports both batch and streaming synthesis:

```go
type TTS interface {
    Synthesize(ctx context.Context, text string, opts ...Option) ([]byte, error)
    SynthesizeStream(ctx context.Context, textStream iter.Seq2[string, error], opts ...Option) iter.Seq2[[]byte, error]
}
```

## Audio Formats

Supported output formats are defined as `AudioFormat` constants:
`FormatPCM`, `FormatOpus`, `FormatMP3`, and `FormatWAV`.

## Registry Pattern

Providers register via `Register` in their init() function and are created
with `New`. Use `List` to discover available providers.

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"

engine, err := tts.New("elevenlabs", tts.Config{Voice: "rachel"})
audio, err := engine.Synthesize(ctx, "Hello, world!")

// Streaming:
for chunk, err := range engine.SynthesizeStream(ctx, textStream) {
    if err != nil { break }
    transport.Send(chunk)
}
```

## Frame Processor Integration

Use `AsFrameProcessor` to wrap a TTS engine as a voice.FrameProcessor for
integration with the cascading pipeline:

```go
processor := tts.AsFrameProcessor(engine, 24000)
```

## Configuration

The `Config` struct supports voice, model, sample rate, format, speed, pitch,
and provider-specific extras. Use functional options like `WithVoice`,
`WithModel`, `WithSampleRate`, `WithFormat`, `WithSpeed`, and `WithPitch`
to configure individual operations.

## Hooks

The `Hooks` struct provides callbacks: BeforeSynthesize, OnAudioChunk, and
OnError. Use `ComposeHooks` to merge multiple hooks.

## Available Providers

- elevenlabs — ElevenLabs (voice/tts/providers/elevenlabs)
- cartesia — Cartesia Sonic (voice/tts/providers/cartesia)
- playht — PlayHT (voice/tts/providers/playht)
- lmnt — LMNT (voice/tts/providers/lmnt)
- fish — Fish Audio (voice/tts/providers/fish)
- groq — Groq TTS (voice/tts/providers/groq)
- smallest — Smallest.ai (voice/tts/providers/smallest)

---

## cartesia

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/cartesia"
```

Package cartesia provides the Cartesia TTS provider for the Beluga AI voice
pipeline. It uses the Cartesia Text-to-Speech API via direct HTTP for batch
synthesis and streaming.

## Registration

This package registers itself as "cartesia" with the tts registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/cartesia"
```

## Usage

```go
engine, err := tts.New("cartesia", tts.Config{
    Voice: "a0e99841-438c-4a64-b679-ae501e7d6091",
    Extra: map[string]any{"api_key": "sk-..."},
})
audio, err := engine.Synthesize(ctx, "Hello, world!")
```

## Configuration

Required configuration in Config.Extra:

- api_key — Cartesia API key (required)
- base_url — Custom API base URL (optional)

The default model is "sonic-2" with raw PCM output at 24000 Hz. Voice is
specified as a Cartesia voice ID.

## Exported Types

- [Engine] — implements tts.TTS using Cartesia
- [New] — constructor accepting tts.Config

---

## elevenlabs

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
```

Package elevenlabs provides the ElevenLabs TTS provider for the Beluga AI
voice pipeline. It uses the ElevenLabs Text-to-Speech API for high-quality
voice synthesis.

## Registration

This package registers itself as "elevenlabs" with the tts registry. Import
it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
```

## Usage

```go
engine, err := tts.New("elevenlabs", tts.Config{
    Voice: "rachel",
    Extra: map[string]any{"api_key": "xi-..."},
})
audio, err := engine.Synthesize(ctx, "Hello, world!")
```

## Configuration

Required configuration in Config.Extra:

- api_key — ElevenLabs API key (required)
- base_url — Custom API base URL (optional)

The default voice is "Rachel" (21m00Tcm4TlvDq8ikWAM) and the default model
is "eleven_monolingual_v1". Output format defaults to audio/mpeg.

## Exported Types

- [Engine] — implements tts.TTS using ElevenLabs
- [New] — constructor accepting tts.Config

---

## fish

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/fish"
```

Package fish provides the Fish Audio TTS provider for the Beluga AI voice
pipeline. It uses the Fish Audio Text-to-Speech API for voice synthesis.

## Registration

This package registers itself as "fish" with the tts registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/fish"
```

## Usage

```go
engine, err := tts.New("fish", tts.Config{
    Voice: "default",
    Extra: map[string]any{"api_key": "..."},
})
audio, err := engine.Synthesize(ctx, "Hello!")
```

## Configuration

Required configuration in Config.Extra:

- api_key — Fish Audio API key (required)
- base_url — Custom API base URL (optional)

The default voice is "default". Voice is used as the reference_id in the
Fish Audio API.

## Exported Types

- [Engine] — implements tts.TTS using Fish Audio
- [New] — constructor accepting tts.Config

---

## groq

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/groq"
```

Package groq provides the Groq TTS provider for the Beluga AI voice pipeline.
It uses the Groq TTS endpoint (OpenAI-compatible API) for voice synthesis.

## Registration

This package registers itself as "groq" with the tts registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/groq"
```

## Usage

```go
engine, err := tts.New("groq", tts.Config{
    Voice: "aura-asteria-en",
    Extra: map[string]any{"api_key": "gsk-..."},
})
audio, err := engine.Synthesize(ctx, "Hello!")
```

## Configuration

Required configuration in Config.Extra:

- api_key — Groq API key (required)
- base_url — Custom API base URL (optional)

The default voice is "aura-asteria-en" and the default model is "playai-tts".
Speed and output format are configurable through [tts.Config].

## Exported Types

- [Engine] — implements tts.TTS using Groq
- [New] — constructor accepting tts.Config

---

## lmnt

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/lmnt"
```

Package lmnt provides the LMNT TTS provider for the Beluga AI voice pipeline.
It uses the LMNT Text-to-Speech API for low-latency voice synthesis.

## Registration

This package registers itself as "lmnt" with the tts registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/lmnt"
```

## Usage

```go
engine, err := tts.New("lmnt", tts.Config{
    Voice: "lily",
    Extra: map[string]any{"api_key": "..."},
})
audio, err := engine.Synthesize(ctx, "Hello!")
```

## Configuration

Required configuration in Config.Extra:

- api_key — LMNT API key (required)
- base_url — Custom API base URL (optional)

The default voice is "lily". Speed and output format are configurable
through [tts.Config].

## Exported Types

- [Engine] — implements tts.TTS using LMNT
- [New] — constructor accepting tts.Config

---

## playht

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/playht"
```

Package playht provides the PlayHT TTS provider for the Beluga AI voice
pipeline. It uses the PlayHT Text-to-Speech API for voice synthesis.

## Registration

This package registers itself as "playht" with the tts registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/playht"
```

## Usage

```go
engine, err := tts.New("playht", tts.Config{
    Voice: "s3://voice-cloning-zero-shot/...",
    Extra: map[string]any{"api_key": "...", "user_id": "..."},
})
audio, err := engine.Synthesize(ctx, "Hello!")
```

## Configuration

Required configuration in Config.Extra:

- api_key — PlayHT API key (required)
- user_id — PlayHT user ID (required)
- base_url — Custom API base URL (optional)

Speed and output format are configurable through [tts.Config]. Default output
format is MP3.

## Exported Types

- [Engine] — implements tts.TTS using PlayHT
- [New] — constructor accepting tts.Config

---

## smallest

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/smallest"
```

Package smallest provides the Smallest.ai TTS provider for the Beluga AI
voice pipeline. It uses the Smallest.ai Text-to-Speech API for low-latency
voice synthesis.

## Registration

This package registers itself as "smallest" with the tts registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/smallest"
```

## Usage

```go
engine, err := tts.New("smallest", tts.Config{
    Voice: "emily",
    Extra: map[string]any{"api_key": "..."},
})
audio, err := engine.Synthesize(ctx, "Hello!")
```

## Configuration

Required configuration in Config.Extra:

- api_key — Smallest.ai API key (required)
- base_url — Custom API base URL (optional)

The default voice is "emily" and the default model is "lightning". Speed is
configurable through [tts.Config].

## Exported Types

- [Engine] — implements tts.TTS using Smallest.ai
- [New] — constructor accepting tts.Config
