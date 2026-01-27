# Package Standardization - Relevant Standards

## Standards Updated

### global/naming (agent-os/standards/index.yml)

**Before**:
```yaml
naming:
  description: Package and folder naming — lowercase, singular, iface/, allowed abbreviations (llms, stt, tts, vad, rag)
```

**After**:
```yaml
naming:
  description: Package and folder naming — lowercase, plural forms preferred (agents, llms, embeddings), iface/, allowed abbreviations (stt, tts, vad, rag)
```

**Rationale**: Aligns documentation with actual codebase where all packages use plural forms.

### global/required-files (agent-os/standards/index.yml)

No changes needed - already correctly documents optional vs required files.

### backend/registry-shape (agent-os/standards/index.yml)

No changes needed - correctly documents `registry.go at root` pattern.

## Standards Referenced

The following standards were referenced during this work:

### Package Structure Standards
- `global/required-files` - Defines required files per package
- `global/subpackage-structure` - Defines sub-package layout
- `global/internal-vs-providers` - What goes in internal/ vs providers/
- `global/wrapper-package-pattern` - Wrapper package facade pattern

### Registry Standards
- `backend/registry-shape` - registry.go location and API
- `backend/registry-public-api` - Required registry methods
- `backend/registry-provider-naming` - Provider naming conventions

### Testing Standards
- `testing/test-utils` - Mocks and helpers in test_utils.go
- `testing/advanced-test` - advanced_test.go requirements

## Package Structure Convention (Updated)

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

**Key clarifications**:
1. `internal/` is OPTIONAL - only use when genuinely needed
2. `registry.go` at root preferred; `registry/` subdirectory acceptable for import cycle avoidance
3. Package names use plural forms: `agents`, `llms`, `embeddings`

## Naming Conventions (Updated)

### Package Names
- **Lowercase**: Always use lowercase letters
- **Plural forms**: Use plural for multi-provider packages (`agents`, `llms`, `embeddings`)
- **Descriptive**: Name should clearly indicate purpose
- **Abbreviations allowed**: `llms`, `stt`, `tts`, `vad`, `rag`

### Examples
```
pkg/agents/          # Agent framework (plural - multiple agent types)
pkg/llms/            # LLM providers (plural - multiple providers)
pkg/embeddings/      # Embedding providers (plural - multiple providers)
pkg/vectorstores/    # Vector store providers (plural - multiple providers)
pkg/config/          # Configuration (singular - single responsibility)
pkg/schema/          # Schema definitions (singular - single responsibility)
```

### When to Use Singular vs Plural
- **Plural**: Multi-provider packages with registry pattern
- **Singular**: Single-responsibility utility packages

In practice, most packages in this framework use plural because they support multiple provider implementations.
