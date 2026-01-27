# voiceutils

Package voiceutils provides shared utilities and interfaces for voice processing packages in the Beluga AI Framework.

## Overview

This package contains:
- **iface/**: Core interfaces for voice processing (STT, TTS, VAD, etc.)
- **audio/**: Audio format utilities and codecs
- **bufferpool.go**: Efficient buffer pooling for audio processing

## Usage

### Interfaces

Import interfaces for implementing providers:

```go
import "github.com/lookatitude/beluga-ai/pkg/voiceutils/iface"

type MySTTProvider struct{}

func (p *MySTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
    // Implementation
}

func (p *MySTTProvider) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
    // Implementation
}
```

### Audio Utilities

```go
import "github.com/lookatitude/beluga-ai/pkg/voiceutils/audio"

// Create default audio format
format := audio.DefaultAudioFormat()

// Validate format
if err := format.Validate(); err != nil {
    log.Fatal(err)
}

// Check codec support
codec := audio.NewCodec()
if codec.IsSupported("opus") {
    // Use opus codec
}
```

### Buffer Pool

```go
import "github.com/lookatitude/beluga-ai/pkg/voiceutils"

// Get global buffer pool
pool := voiceutils.GetGlobalBufferPool()

// Get a buffer
buf := pool.Get(4096)
defer pool.Put(buf)

// Use buffer for audio processing
// ...
```

## Interfaces

| Interface | Description |
|-----------|-------------|
| `STTProvider` | Speech-to-text conversion |
| `TTSProvider` | Text-to-speech conversion |
| `VADProvider` | Voice activity detection |
| `Transport` | Audio transport (WebRTC, WebSocket) |
| `NoiseCancellation` | Noise reduction |
| `TurnDetector` | Conversation turn detection |
| `VoiceSession` | Voice session management |

## Related Packages

- `pkg/stt` - Speech-to-text implementations
- `pkg/tts` - Text-to-speech implementations
- `pkg/vad` - Voice activity detection implementations
- `pkg/audiotransport` - Audio transport implementations
- `pkg/noisereduction` - Noise cancellation implementations
- `pkg/turndetection` - Turn detection implementations
- `pkg/voicesession` - Voice session management
- `pkg/voicebackend` - Voice backend integrations
