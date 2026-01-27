# Shape: Twilio Provider Migration

## Problem Statement

The `pkg/voice/` package is being deprecated as part of a broader effort to flatten the package structure and improve organization. The Twilio provider currently lives in `pkg/voice/providers/twilio/` and needs to be migrated to `pkg/voicebackend/providers/twilio/` to align with the new package structure.

## Context

### Current State
- Twilio provider is in `pkg/voice/providers/twilio/` (20 files, ~11,000 lines)
- LiveKit provider already migrated to `pkg/voicebackend/providers/livekit/`
- Voice-related packages have been split into:
  - `pkg/stt/` - Speech-to-Text
  - `pkg/tts/` - Text-to-Speech
  - `pkg/vad/` - Voice Activity Detection
  - `pkg/noisereduction/` - Noise cancellation
  - `pkg/turndetection/` - Turn detection
  - `pkg/s2s/` - Speech-to-Speech
  - `pkg/voicesession/` - Voice session management
  - `pkg/voicebackend/` - Voice backend implementations
  - `pkg/voiceutils/` - Shared voice utilities

### Target State
- Twilio provider in `pkg/voicebackend/providers/twilio/`
- All imports updated to use new package paths
- Original location has deprecation notices
- Provider registered with `voicebackend.GetRegistry()`

## Design Decisions

### Decision 1: Follow LiveKit Pattern
The LiveKit provider has already been migrated and serves as the reference implementation. Key patterns:
- `init.go` registers provider with global registry
- Provider implements `CreateBackend` method
- Uses `pkg/voicebackend/iface` for interfaces

### Decision 2: Import Mapping Strategy
All old `pkg/voice/*` imports map to new standalone packages:
- `pkg/voice/backend` → `pkg/voicebackend`
- `pkg/voice/iface` → `pkg/voiceutils/iface`
- `pkg/voice/stt` → `pkg/stt`
- `pkg/voice/tts` → `pkg/tts`
- etc.

### Decision 3: Deprecation vs Deletion
Phase 1 only adds deprecation notices - original files are NOT deleted yet. This allows:
- Gradual migration of dependent code
- Rollback if issues are discovered
- Clear deprecation path for external consumers

### Decision 4: Minimal Code Changes
Only import paths should change. No functional changes to the provider implementation during migration.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Import path typos | Thorough testing with `go build ./...` |
| Missing interface compatibility | Compare with LiveKit provider pattern |
| Breaking external consumers | Deprecation notices in original location |
| Test failures | Run full test suite after migration |

## Out of Scope

- Functional changes to Twilio provider
- Deletion of original files (Phase 3)
- Migration of other pkg/voice components
- Performance optimizations
