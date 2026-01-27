# References: Twilio Provider Migration

## Reference Implementations Studied

### 1. LiveKit Provider (pkg/voicebackend/providers/livekit/)

The LiveKit provider has already been migrated and serves as the canonical reference.

**Key files:**
- `init.go` - Registration pattern
- `provider.go` - Provider implementation
- `backend.go` - Backend implementation
- `config.go` - Configuration struct

**Registration Pattern (init.go):**
```go
package livekit

import (
	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
)

func init() {
	// Register LiveKit provider with the global registry
	provider := NewLiveKitProvider()
	voicebackend.GetRegistry().Register("livekit", provider.CreateBackend)
}
```

### 2. VoiceBackend Registry (pkg/voicebackend/registry.go)

The registry manages provider registration and creation.

**Key interfaces:**
- `GetRegistry()` - Returns global registry singleton
- `Register(name, creator)` - Registers a provider
- `Create(ctx, name, config)` - Creates backend instance
- `ListProviders()` - Lists registered providers

**Creator function signature:**
```go
func(context.Context, *vbiface.Config) (vbiface.VoiceBackend, error)
```

### 3. VoiceBackend Interface (pkg/voicebackend/iface/)

The interface package defines the contracts that providers must implement.

**Key interfaces:**
- `VoiceBackend` - Main backend interface
- `Config` - Configuration struct
- Various session and handler interfaces

## Import Mapping Table

| Old Import Path | New Import Path | Alias |
|-----------------|-----------------|-------|
| `github.com/lookatitude/beluga-ai/pkg/voice/backend` | `github.com/lookatitude/beluga-ai/pkg/voicebackend` | - |
| `github.com/lookatitude/beluga-ai/pkg/voice/backend/iface` | `github.com/lookatitude/beluga-ai/pkg/voicebackend/iface` | `vbiface` |
| `github.com/lookatitude/beluga-ai/pkg/voice/iface` | `github.com/lookatitude/beluga-ai/pkg/voiceutils/iface` | `viface` |
| `github.com/lookatitude/beluga-ai/pkg/voice/stt` | `github.com/lookatitude/beluga-ai/pkg/stt` | - |
| `github.com/lookatitude/beluga-ai/pkg/voice/tts` | `github.com/lookatitude/beluga-ai/pkg/tts` | - |
| `github.com/lookatitude/beluga-ai/pkg/voice/vad` | `github.com/lookatitude/beluga-ai/pkg/vad` | - |
| `github.com/lookatitude/beluga-ai/pkg/voice/noise` | `github.com/lookatitude/beluga-ai/pkg/noisereduction` | - |
| `github.com/lookatitude/beluga-ai/pkg/voice/turndetection` | `github.com/lookatitude/beluga-ai/pkg/turndetection` | - |
| `github.com/lookatitude/beluga-ai/pkg/voice/s2s` | `github.com/lookatitude/beluga-ai/pkg/s2s` | - |
| `github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface` | `github.com/lookatitude/beluga-ai/pkg/s2s/iface` | `s2siface` |
| `github.com/lookatitude/beluga-ai/pkg/voice/session` | `github.com/lookatitude/beluga-ai/pkg/voicesession` | - |
| `github.com/lookatitude/beluga-ai/pkg/voice/session/iface` | `github.com/lookatitude/beluga-ai/pkg/voicesession/iface` | `vsiface` |
| `github.com/lookatitude/beluga-ai/pkg/voice/transport/iface` | `github.com/lookatitude/beluga-ai/pkg/voiceutils/iface` | `viface` |

## Files to Migrate

### Source Files (20 total)

**Core files (no external voice imports):**
1. `errors.go` - Error definitions
2. `metrics.go` - OTEL metrics
3. `streaming.go` - Streaming utilities

**Config/Provider files:**
4. `config.go` - Configuration struct
5. `provider.go` - Provider implementation

**Backend implementation:**
6. `backend.go` - Main backend
7. `webhook.go` - Webhook handling
8. `webhook_handlers.go` - Webhook handlers
9. `orchestration.go` - Orchestration logic
10. `transcription.go` - Transcription handling

**Session files (most complex):**
11. `session.go` - Session management
12. `session_adapter.go` - Session adapter (highest complexity)

**Registration:**
13. `init.go` - Provider registration

**Test files:**
14. `test_utils.go` - Test utilities
15. `advanced_test.go` - Advanced tests
16. `provider_test.go` - Provider tests
17. `session_adapter_test.go` - Session adapter tests
18. `webhook_test.go` - Webhook tests
19. `transcription_test.go` - Transcription tests
20. `streaming_test.go` - Streaming tests

## Related Documentation

- [Package Design Patterns](../../../docs/package_design_patterns.md)
- [VoiceBackend README](../../../pkg/voicebackend/README.md)
- [LiveKit Provider](../../../pkg/voicebackend/providers/livekit/)
