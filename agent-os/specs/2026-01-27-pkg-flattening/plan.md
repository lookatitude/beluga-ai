# Package Flattening Refactor

## Summary

Flatten the pkg/ structure to maximize reusability following Go standard library patterns. The primary change is promoting voice subpackages (stt, tts, vad, etc.) to top-level packages and fixing the embeddings registry anti-pattern.

## Decisions Made

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Voice subpackages | Promote to pkg/ level | Go stdlib pattern, independent imports |
| Naming conflicts | Prefix with domain (audiotransport, noisereduction) | Avoid stdlib conflicts |
| Shared voice interfaces | New voiceutils package | Central location for shared types |
| Backward compatibility | Deprecation shims in pkg/voice | Allow migration period |
| embeddings/registry/ | Move to embeddings/registry.go | Match standard pattern |

## Standards Applied

- **global/naming** — Plural forms, lowercase
- **global/subpackage-structure** — Sub-package independence
- **backend/registry-shape** — registry.go at root
- **global/internal-vs-providers** — internal/ only when needed
- **global/wrapper-package-pattern** — Facade patterns

## Tasks

### Task 1: Save Spec Documentation
Create `agent-os/specs/2026-01-27-pkg-flattening/` with plan.md, shape.md, standards.md, references.md

### Task 2: Create voiceutils Foundation Package
Create `pkg/voiceutils/` with shared voice interfaces and utilities

### Task 3-6: Flatten Voice Subpackages
- STT: `pkg/voice/stt/` → `pkg/stt/`
- TTS: `pkg/voice/tts/` → `pkg/tts/`
- VAD: `pkg/voice/vad/` → `pkg/vad/`
- S2S: `pkg/voice/s2s/` → `pkg/s2s/`

### Task 7-11: Flatten Renamed Packages
- Transport: `pkg/voice/transport/` → `pkg/audiotransport/`
- Noise: `pkg/voice/noise/` → `pkg/noisereduction/`
- Turndetection: `pkg/voice/turndetection/` → `pkg/turndetection/`
- Backend: `pkg/voice/backend/` → `pkg/voicebackend/`
- Session: `pkg/voice/session/` → `pkg/voicesession/`

### Task 12: Fix Embeddings Registry
Move `pkg/embeddings/registry/` to `pkg/embeddings/registry.go`

### Task 13: Create Deprecation Shims
Backward-compatible shims in `pkg/voice/deprecated.go`

### Task 14: Flatten agents/tools/
Flatten nested structure within `pkg/agents/tools/`

### Task 15-17: Documentation and Verification
Update all documentation and verify build/tests pass

## New Package Structure

```
pkg/
├── agents/
├── audiotransport/      # NEW (from voice/transport)
├── chatmodels/
├── config/
├── core/
├── documentloaders/
├── embeddings/          # FIXED registry
├── llms/
├── memory/
├── messaging/
├── monitoring/
├── multimodal/
├── noisereduction/      # NEW (from voice/noise)
├── orchestration/
├── prompts/
├── retrievers/
├── s2s/                 # NEW (from voice/s2s)
├── safety/
├── schema/
├── server/
├── stt/                 # NEW (from voice/stt)
├── textsplitters/
├── tts/                 # NEW (from voice/tts)
├── turndetection/       # NEW (from voice/turndetection)
├── vad/                 # NEW (from voice/vad)
├── vectorstores/
├── voice/               # DEPRECATED (shims only)
├── voicebackend/        # NEW (from voice/backend)
├── voicesession/        # NEW (from voice/session)
└── voiceutils/          # NEW (shared interfaces)
```

## Breaking Changes

| Old Import | New Import |
|------------|------------|
| `pkg/voice/stt` | `pkg/stt` |
| `pkg/voice/tts` | `pkg/tts` |
| `pkg/voice/vad` | `pkg/vad` |
| `pkg/voice/s2s` | `pkg/s2s` |
| `pkg/voice/transport` | `pkg/audiotransport` |
| `pkg/voice/noise` | `pkg/noisereduction` |
| `pkg/voice/turndetection` | `pkg/turndetection` |
| `pkg/voice/backend` | `pkg/voicebackend` |
| `pkg/voice/session` | `pkg/voicesession` |
| `pkg/voice/iface` | `pkg/voiceutils/iface` |
| `pkg/embeddings/registry` | `pkg/embeddings` |

Deprecation shims provided for v1.x, removed in v2.0.
