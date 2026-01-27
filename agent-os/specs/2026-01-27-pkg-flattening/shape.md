# Shape: Package Flattening Refactor

## Problem Statement

The current `pkg/voice/` package structure nests subpackages deeply, which:
1. Creates long import paths (`github.com/lookatitude/beluga-ai/pkg/voice/stt`)
2. Makes it harder to use individual components independently
3. Doesn't follow Go standard library patterns (e.g., `net/http` vs `net/http/httptransport`)
4. The `pkg/embeddings/registry/` subdirectory is an anti-pattern when other packages use `registry.go` at root

## Shaping Questions & Decisions

### Q1: Should voice subpackages be promoted to top-level?
**Decision**: Yes, promote to `pkg/` level
**Rationale**: Go stdlib pattern promotes independence. Users importing only STT shouldn't need to understand the voice package hierarchy.

### Q2: What about naming conflicts with stdlib?
**Decision**: Prefix domain-specific packages
- `transport` → `audiotransport` (avoids confusion with net/http transport)
- `noise` → `noisereduction` (more descriptive, avoids generic name)
- `backend` → `voicebackend`
- `session` → `voicesession`

### Q3: Where should shared voice interfaces live?
**Decision**: New `pkg/voiceutils/` package
**Contents**:
- `iface/` - Interfaces currently in `pkg/voice/iface/`
- `audio/` - Audio utilities from `pkg/voice/internal/audio/`
- `bufferpool/` - Buffer pool from `pkg/voice/buffer_pool.go`

### Q4: How to handle backward compatibility?
**Decision**: Deprecation shims in `pkg/voice/`
- Type aliases pointing to new locations
- Variable aliases for factory functions
- `// Deprecated:` comments with migration instructions
- Remove in v2.0

### Q5: What about the embeddings registry?
**Decision**: Move to package root
- `pkg/embeddings/registry/registry.go` → `pkg/embeddings/registry.go`
- Merge `registry/iface/` into `pkg/embeddings/iface/`
- This matches the pattern used in other packages

## Migration Strategy

1. Create new packages first (additive)
2. Update internal imports
3. Add deprecation shims
4. Update examples and tests
5. Update documentation
6. Verify build and tests pass

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Breaking external consumers | Deprecation shims provide migration period |
| Circular imports | voiceutils created specifically to break cycles |
| Missing imports during refactor | Comprehensive grep verification step |
| Documentation drift | Bulk search-and-replace with verification |
