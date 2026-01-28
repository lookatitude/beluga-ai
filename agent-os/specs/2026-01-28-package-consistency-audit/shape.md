# Package Structure Consistency Audit - Shape Document

## Audit Scope

Comprehensive review of all 29 packages in `pkg/` directory to verify compliance with the standardized package structure convention.

## Package Structure Convention (Reference)

Every package **MUST** follow this standardized structure:

```
pkg/{package_name}/
├── iface/                    # Public interfaces and types (REQUIRED)
├── internal/                 # Private implementation details (OPTIONAL)
├── providers/                # Provider implementations (multi-provider packages)
├── config.go                 # Configuration structs with validation
├── metrics.go                # OTEL metrics implementation
├── errors.go                 # Custom errors (Op/Err/Code pattern)
├── {package_name}.go         # Main API and factory functions
├── registry.go               # Global registry (multi-provider packages)
├── test_utils.go             # Test helpers and mock factories
├── advanced_test.go          # Comprehensive test suite
└── README.md                 # Package documentation
```

## Audit Findings

### Category 1: Fully Compliant (26 packages)

The following packages fully comply with the naming conventions:

| Package | Main File | Has Registry | Has Providers | Status |
|---------|-----------|--------------|---------------|--------|
| agents | agents.go | Yes | Yes | Compliant |
| audiotransport | audiotransport.go | Yes | Yes | Compliant |
| chatmodels | chatmodels.go | Yes | Yes | Compliant |
| config | config.go | No* | No | Compliant |
| core | N/A* | No | No | Compliant |
| documentloaders | documentloaders.go | Yes | Yes | Compliant |
| embeddings | embeddings.go | Yes | Yes | Compliant |
| llms | llms.go | Yes | Yes | Compliant |
| memory | memory.go | Yes | Yes | Compliant |
| messaging | messaging.go | Yes | Yes | Compliant |
| monitoring | monitoring.go | Yes | No | Compliant |
| multimodal | multimodal.go | Yes | Yes | Compliant |
| retrievers | retrievers.go | Yes | Yes | Compliant |
| s2s | s2s.go | Yes | Yes | Compliant |
| schema | schema.go | No* | No | Compliant |
| server | server.go | Yes | No | Compliant |
| stt | stt.go | Yes | Yes | Compliant |
| textsplitters | textsplitters.go | Yes | Yes | Compliant |
| tools | tools.go | Yes | Yes | Compliant |
| tts | tts.go | Yes | Yes | Compliant |
| turndetection | turndetection.go | Yes | Yes | Compliant |
| vad | vad.go | Yes | Yes | Compliant |
| vectorstores | vectorstores.go | Yes | Yes | Compliant |
| voicebackend | voicebackend.go | Yes | Yes | Compliant |
| voicesession | voicesession.go | No* | No | Compliant |
| voiceutils | N/A* | No | No | Compliant |

*Intentional deviations documented in standards.md

### Category 2: Fixed - Naming Issues (5 packages) ✅

The following naming issues were identified and fixed:

| Package | Issue | Previous File | Fixed File | Status |
|---------|-------|---------------|------------|--------|
| noisereduction | Main file misnamed | noise.go | noisereduction.go | ✅ Fixed |
| orchestration | Main file misnamed | orchestrator.go | orchestration.go | ✅ Fixed |
| audiotransport | Main file misnamed | transport.go | audiotransport.go | ✅ Fixed |
| voicebackend | Main file misnamed | backend.go | voicebackend.go | ✅ Fixed |
| voicesession | Main file misnamed | session.go | voicesession.go | ✅ Fixed |

### Category 3: Fixed - Missing Registry (1 package) ✅

| Package | Issue | Fix Applied | Status |
|---------|-------|-------------|--------|
| prompts | Missing registry.go | Added registry.go and iface/registry.go | ✅ Fixed |

## Decisions

### Fix Decisions

1. **noisereduction/noise.go** → Rename to `noisereduction.go`
   - Follows convention: main file matches package name
   - No code changes required, only file rename

2. **orchestration/orchestrator.go** → Rename to `orchestration.go`
   - Follows convention: main file matches package name
   - No code changes required, only file rename

3. **prompts/registry.go** → Create new file
   - Package has `providers/` directory with mock.go
   - Should have registry for template engine providers
   - Follow embeddings/registry.go pattern

### Intentional Deviations (No Action Needed)

1. **config** - No registry
   - Config is loaded from files/env, not created from providers
   - Uses factory functions directly

2. **core** - No {package_name}.go main file
   - Utility package with multiple entry points
   - Contains: di.go, runnable.go, errors.go
   - No single "main" concept

3. **schema** - No registry
   - Pure data structure definitions
   - No providers or factory pattern needed

4. **voicesession** - No registry
   - Single implementation package
   - No multiple providers pattern

5. **voiceutils** - No main API file
   - Shared interfaces and utility types only
   - Imported by other voice packages

6. **convenience** - Namespace package with no root-level code
   - Aggregation package grouping convenience sub-packages
   - Sub-packages (agent, rag, voiceagent, mock, context, provider) follow standard structure
   - Root directory contains only README.md

## Impact Assessment

### Risk: Low
- File renames are git-tracked with full history preserved
- No code changes to existing implementations
- No breaking API changes

### Testing Required
- Build verification after each rename
- Unit test execution for affected packages
- Lint check to ensure no issues introduced
