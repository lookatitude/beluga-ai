# Phase 1: Migrate Twilio Provider to pkg/voicebackend

## Summary

Migrate the Twilio voice backend provider from `pkg/voice/providers/twilio/` to `pkg/voicebackend/providers/twilio/` as part of the pkg/voice/ deprecation effort. This is Phase 1 of a 3-phase deprecation plan.

## Scope

- **Type**: Provider migration (part of pkg/voice deprecation)
- **Files**: 20 Go files (~11,000 lines)
- **Risk**: Medium-High (many import path changes, interface compatibility)

## Tasks

### Task 1: Save Spec Documentation

Create `agent-os/specs/2026-01-27-twilio-provider-migration/` with:
- `plan.md` - This plan
- `shape.md` - Shaping notes and decisions
- `references.md` - Reference implementations studied

### Task 2: Create Target Directory

Create `pkg/voicebackend/providers/twilio/` directory.

### Task 3: Migrate Core Files (No External Dependencies)

Copy and update these files first (minimal import changes):

| File | Import Changes |
|------|----------------|
| `errors.go` | None (self-contained) |
| `metrics.go` | None (OTEL only) |
| `streaming.go` | None (uses local errors.go) |

### Task 4: Migrate Config and Provider Files

| File | Import Changes |
|------|----------------|
| `config.go` | `pkg/voice/backend/iface` → `pkg/voicebackend/iface` |
| `provider.go` | `pkg/voice/backend` → `pkg/voicebackend`, `pkg/voice/backend/iface` → `pkg/voicebackend/iface` |

### Task 5: Migrate Backend Implementation

| File | Import Changes |
|------|----------------|
| `backend.go` | `pkg/voice/backend/iface` → `pkg/voicebackend/iface` |
| `webhook.go` | `pkg/voice/backend/iface` → `pkg/voicebackend/iface` |
| `webhook_handlers.go` | `pkg/voice/backend/iface` → `pkg/voicebackend/iface` |
| `orchestration.go` | `pkg/voice/backend/iface` → `pkg/voicebackend/iface` |
| `transcription.go` | `pkg/voice/backend/iface` → `pkg/voicebackend/iface` |

### Task 6: Migrate Session Files (Most Complex)

These have the most import changes:

**session.go**:
- `pkg/voice/backend/iface` → `pkg/voicebackend/iface`
- `pkg/voice/iface` → `pkg/voiceutils/iface`
- `pkg/voice/stt` → `pkg/stt`
- `pkg/voice/tts` → `pkg/tts`

**session_adapter.go** (highest complexity):
- `pkg/voice/backend/iface` → `pkg/voicebackend/iface`
- `pkg/voice/iface` → `pkg/voiceutils/iface`
- `pkg/voice/noise` → `pkg/noisereduction`
- `pkg/voice/s2s` → `pkg/s2s`
- `pkg/voice/s2s/iface` → `pkg/s2s/iface`
- `pkg/voice/session` → `pkg/voicesession`
- `pkg/voice/session/iface` → `pkg/voicesession/iface`
- `pkg/voice/stt` → `pkg/stt`
- `pkg/voice/transport/iface` → `pkg/voiceutils/iface`
- `pkg/voice/tts` → `pkg/tts`
- `pkg/voice/turndetection` → `pkg/turndetection`
- `pkg/voice/vad` → `pkg/vad`

### Task 7: Migrate init.go (Registry Integration)

Update to match livekit pattern:

```go
package twilio

import (
    "github.com/lookatitude/beluga-ai/pkg/voicebackend"
)

func init() {
    provider := NewTwilioProvider()
    voicebackend.GetRegistry().Register("twilio", provider.CreateBackend)
}
```

### Task 8: Migrate Test Files

| File | Notes |
|------|-------|
| `test_utils.go` | Update vbiface imports |
| `advanced_test.go` | Update all imports |
| `provider_test.go` | Update all imports |
| `session_adapter_test.go` | Update all imports |
| `webhook_test.go` | Update all imports |
| `transcription_test.go` | Update all imports |
| `streaming_test.go` | Update all imports |

### Task 9: Add Deprecation Notice

Add deprecation notice to original `pkg/voice/providers/twilio/` pointing to new location.

### Task 10: Update Documentation

Update references in:
- `docs/providers/twilio.md`
- `docs/package-catalog.md`

## Critical Files

### Source (to migrate)
- `pkg/voice/providers/twilio/*.go` (20 files)

### Target (to create)
- `pkg/voicebackend/providers/twilio/*.go`

### Reference Implementation
- `pkg/voicebackend/providers/livekit/init.go` - Registration pattern
- `pkg/voicebackend/registry.go` - Registry API

### Import Mapping Reference
| Old Import | New Import |
|------------|------------|
| `pkg/voice/backend` | `pkg/voicebackend` |
| `pkg/voice/backend/iface` | `pkg/voicebackend/iface` |
| `pkg/voice/iface` | `pkg/voiceutils/iface` |
| `pkg/voice/stt` | `pkg/stt` |
| `pkg/voice/tts` | `pkg/tts` |
| `pkg/voice/vad` | `pkg/vad` |
| `pkg/voice/noise` | `pkg/noisereduction` |
| `pkg/voice/turndetection` | `pkg/turndetection` |
| `pkg/voice/s2s` | `pkg/s2s` |
| `pkg/voice/s2s/iface` | `pkg/s2s/iface` |
| `pkg/voice/session` | `pkg/voicesession` |
| `pkg/voice/session/iface` | `pkg/voicesession/iface` |
| `pkg/voice/transport/iface` | `pkg/voiceutils/iface` |

## Verification

After implementation:

```bash
# 1. Run Twilio provider tests
go test -v ./pkg/voicebackend/providers/twilio/...

# 2. Run full voicebackend package tests
go test -v ./pkg/voicebackend/...

# 3. Verify no import errors
go build ./...

# 4. Run linting
make lint

# 5. Verify provider registration
# (In test or manually check voicebackend.GetRegistry().ListProviders() includes "twilio")

# 6. Run full test suite
make test
```

## Notes

- This is Phase 1 of 3 for pkg/voice deprecation
- Phase 2: Extract utilities to pkg/voiceutils
- Phase 3: Delete pkg/voice directory
- Do NOT delete original files yet - add deprecation notices only
