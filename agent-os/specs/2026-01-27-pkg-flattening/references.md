# References: Go Standard Library Patterns

## Go stdlib Package Structure Examples

### Independent Subpackages
```
net/
├── http/           # Standalone, most users import this directly
├── mail/           # Standalone
├── smtp/           # Standalone
├── url/            # Standalone
└── internal/       # Shared internals not for external use
```

### Flat vs Nested
**Flat (preferred for independent use)**:
- `encoding/json` - not `encoding/format/json`
- `crypto/aes` - not `crypto/symmetric/aes`
- `database/sql` - not `database/driver/sql`

**Nested (when hierarchy adds meaning)**:
- `net/http/httputil` - utilities specifically for http
- `go/ast` + `go/parser` + `go/token` - related but independent

## Beluga Voice Package Analysis

### Current Structure (nested)
```
pkg/voice/
├── stt/           # Could be independent
├── tts/           # Could be independent
├── vad/           # Could be independent
├── s2s/           # Could be independent
├── transport/     # Could be independent (renamed)
├── noise/         # Could be independent (renamed)
├── turndetection/ # Could be independent
├── backend/       # Could be independent (renamed)
├── session/       # Depends on others, renamed
└── iface/         # Shared interfaces → voiceutils
```

### After Flattening (stdlib-like)
```
pkg/
├── stt/            # like encoding/json
├── tts/            # like encoding/xml
├── vad/            # like crypto/aes
├── s2s/            # like crypto/rsa
├── audiotransport/ # like net/http (domain-prefixed)
├── noisereduction/ # like compress/gzip (descriptive)
├── turndetection/  # like text/scanner
├── voicebackend/   # like database/sql (domain-prefixed)
├── voicesession/   # like net/http (domain-prefixed)
└── voiceutils/     # like internal shared types
```

## Registry Pattern Reference

### Standard Pattern (used by llms, chatmodels, vectorstores)
```go
// pkg/llms/registry.go
package llms

var globalRegistry = &Registry{
    providers: make(map[string]ProviderFactory),
}

func Register(name string, factory ProviderFactory) {
    globalRegistry.Register(name, factory)
}

func New(ctx context.Context, name string, cfg Config) (LLM, error) {
    return globalRegistry.New(ctx, name, cfg)
}
```

### Anti-pattern (current embeddings)
```go
// pkg/embeddings/registry/registry.go  ← subdirectory
package registry

// This creates import like:
// import "github.com/lookatitude/beluga-ai/pkg/embeddings/registry"
// Instead of just using pkg/embeddings
```

## Deprecation Pattern Reference

### Go stdlib deprecation style
```go
// Deprecated: Use errors.Is instead.
func (e *Error) Temporary() bool { ... }
```

### Beluga deprecation shim pattern
```go
// pkg/voice/deprecated.go
package voice

import (
    "github.com/lookatitude/beluga-ai/pkg/stt"
    "github.com/lookatitude/beluga-ai/pkg/tts"
)

// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/stt instead.
type STTConfig = stt.Config

// Deprecated: Use stt.NewProvider instead.
var NewSTTProvider = stt.NewProvider
```
