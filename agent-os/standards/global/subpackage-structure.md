# Sub-Package Structure Standard

## Purpose

Sub-packages are nested directories within a parent package that are treated as independent packages. This standard defines the structure, naming, and independence requirements for sub-packages.

## Naming Convention

**Pattern:** `pkg/<parent>/<provider-or-component>`

### Component Sub-Packages (Wrapper Packages)

```
pkg/voice/stt/              # Speech-to-Text component
pkg/voice/tts/              # Text-to-Speech component
pkg/voice/vad/              # Voice Activity Detection
pkg/voice/session/          # Session management
pkg/voice/transport/        # Audio transport
```

### Provider Sub-Packages (Regular Packages)

```
pkg/llms/providers/openai/      # OpenAI LLM provider
pkg/llms/providers/anthropic/   # Anthropic LLM provider
pkg/embeddings/providers/openai/ # OpenAI embeddings provider
```

## Independence Requirements

Sub-packages **MUST** be independently importable and testable.

### Direct Import

```go
// Direct import of sub-package
import "github.com/lookatitude/beluga-ai/pkg/voice/stt"

// Use without parent package
transcriber, err := stt.NewDeepgramTranscriber(cfg)
result, err := transcriber.Transcribe(ctx, audio)
```

### No Parent Dependencies

Sub-packages **MUST NOT** require imports from the parent package:

```go
// Bad: Sub-package imports parent
package stt

import "github.com/lookatitude/beluga-ai/pkg/voice"  // Don't do this

// Good: Sub-package is self-contained
package stt

import "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"  // Internal imports OK
```

### No Sibling Dependencies

Sub-packages **MUST NOT** directly import sibling sub-packages:

```go
// Bad: STT imports TTS sibling
package stt

import "github.com/lookatitude/beluga-ai/pkg/voice/tts"  // Don't do this

// Good: Coordination happens at parent level
// Parent (voice) composes stt and tts via interfaces
```

## Required Structure

Each sub-package **MUST** follow the standard package structure:

```
pkg/{parent}/{subpackage}/
├── iface/                    # Interfaces (REQUIRED)
│   └── {subpackage}.go      # Primary interface
├── internal/                 # Private implementation
├── providers/               # Providers (if multi-provider)
│   ├── {provider1}/
│   │   ├── {provider1}.go
│   │   └── init.go          # Auto-registration
│   └── {provider2}/
├── config.go                # Configuration (REQUIRED)
├── metrics.go               # OTEL metrics (REQUIRED)
├── errors.go                # Error definitions (REQUIRED)
├── registry.go              # Registry (if multi-provider)
├── {subpackage}.go          # Main API
├── test_utils.go            # Test utilities (REQUIRED)
├── advanced_test.go         # Comprehensive tests (REQUIRED)
└── README.md                # Documentation
```

## Interface Definition

Sub-packages define their own interfaces in `iface/`:

```go
// pkg/voice/stt/iface/transcriber.go
package iface

import "context"

type Transcriber interface {
    Transcribe(ctx context.Context, audio []byte) (*TranscriptionResult, error)
    StreamTranscribe(ctx context.Context, audio <-chan []byte) (<-chan *TranscriptionResult, error)
    Close() error
}

type TranscriptionResult struct {
    Text       string
    Confidence float64
    Language   string
    Segments   []Segment
}
```

## Parent-Child Relationship

### Parents Depend on Interfaces

Parents depend on sub-packages via **interfaces only**, never concrete types:

```go
// Parent package
package voice

import (
    sttiface "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
    ttsiface "github.com/lookatitude/beluga-ai/pkg/voice/tts/iface"
)

type voiceAgent struct {
    stt sttiface.Transcriber  // Interface from sub-package
    tts ttsiface.Speaker      // Interface from sub-package
}
```

### Registry Indirection

Parents use registries to discover and instantiate sub-packages:

```go
// Parent uses sub-package registry
sttProvider, err := stt.GetRegistry().GetProvider(cfg.STT.Provider, &cfg.STT)
ttsProvider, err := tts.GetRegistry().GetProvider(cfg.TTS.Provider, &cfg.TTS)
```

## Testing

Each sub-package has independent tests:

```go
// pkg/voice/stt/advanced_test.go
package stt

func TestTranscriberIndependently(t *testing.T) {
    // Test STT without voice parent
    transcriber := NewMockTranscriber()
    result, err := transcriber.Transcribe(ctx, testAudio)
    require.NoError(t, err)
}
```

## Examples

### STT Sub-Package (Component)

```
pkg/voice/stt/
├── iface/
│   └── transcriber.go       # Transcriber interface
├── providers/
│   ├── deepgram/
│   │   ├── deepgram.go
│   │   └── init.go
│   └── openai/
│       ├── openai.go
│       └── init.go
├── config.go
├── metrics.go
├── errors.go
├── registry.go
├── stt.go
├── test_utils.go
└── advanced_test.go
```

### OpenAI Provider Sub-Package

```
pkg/llms/providers/openai/
├── openai.go                # Provider implementation
├── config.go                # Provider-specific config
├── init.go                  # Auto-registration
└── openai_test.go           # Provider tests
```

## Related Standards

- [wrapper-package-pattern.md](./wrapper-package-pattern.md) - Wrapper package patterns
- [required-files.md](./required-files.md) - Required files per package
- [provider-subpackage-layout.md](./provider-subpackage-layout.md) - Provider layout
