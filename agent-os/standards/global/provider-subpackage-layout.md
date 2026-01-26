# Provider Subpackage Layout

Each provider lives in `providers/<name>/` with:

- **init.go** — `func init()` that registers with the package registry. Enables discoverability (auto-register on import) and keeps wiring in one place.

```go
func init() {
    llms.GetRegistry().Register("openai", NewOpenAIProviderFactory())
}
```

- **provider.go** — Common functionality and wiring: factory, shared setup, config handling. Keeps provider implementations consistent.

- **{name}.go** — Provider-specific implementation (e.g. `openai.go`, `anthropic.go`).

- **{name}_mock.go** — Test mocks for this provider.

**Rule:** The string passed to `Register` matches the directory name (e.g. `"openai"` for `providers/openai/`).

## Component Sub-Packages (Wrapper Packages)

For wrapper packages, sub-packages represent components (not providers):

```
pkg/voice/
├── stt/              # Speech-to-Text component
│   ├── iface/
│   ├── providers/    # STT providers (deepgram, openai, etc.)
│   ├── config.go
│   ├── metrics.go
│   ├── errors.go
│   ├── registry.go
│   └── stt.go
├── tts/              # Text-to-Speech component
│   ├── iface/
│   ├── providers/    # TTS providers (elevenlabs, google, etc.)
│   └── ...
├── vad/              # Voice Activity Detection component
└── session/          # Session management component
```

### Component Sub-Package Layout

Each component sub-package is structured like a full package:

```
pkg/voice/stt/
├── iface/
│   └── transcriber.go    # Transcriber interface
├── providers/
│   ├── deepgram/
│   │   ├── deepgram.go   # Provider implementation
│   │   └── init.go       # Auto-registration
│   └── openai/
│       ├── openai.go
│       └── init.go
├── config.go             # STT-specific config
├── metrics.go            # STT metrics
├── errors.go             # STT errors
├── registry.go           # STT registry
├── stt.go                # Main API
├── test_utils.go         # Mocks
└── advanced_test.go      # Tests
```

### Component Provider Registration

Providers within component sub-packages register with the component registry:

```go
// pkg/voice/stt/providers/deepgram/init.go
func init() {
    stt.GetRegistry().Register("deepgram", NewDeepgramTranscriber)
}
```

### Independence Requirement

Component sub-packages MUST be independently importable:

```go
// Direct import of component
import "github.com/lookatitude/beluga-ai/pkg/voice/stt"

// Use without parent
transcriber, err := stt.NewProvider(ctx, "deepgram", cfg)
```

Sub-packages MUST NOT import the parent package or sibling sub-packages directly.
