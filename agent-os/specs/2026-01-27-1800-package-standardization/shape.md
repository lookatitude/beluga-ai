# Shaping Notes: Package Standardization

## Problem Statement

The `pkg/retrievers` and `pkg/memory` packages do not fully follow the standardized package structure convention established in other packages like `pkg/embeddings` and `pkg/chatmodels`.

### pkg/retrievers Issues
1. No registry infrastructure for provider management
2. Provider implementations (vectorstore, multiquery) live in root package instead of `providers/` subdirectory
3. Mock provider exists in `providers/mock/` but doesn't auto-register

### pkg/memory Issues
1. Provider implementations live in `internal/` instead of `providers/`
2. Base history implementations live in `providers/` instead of `internal/base/`
3. Inconsistent with other packages that use `providers/` for swappable implementations

## Design Decisions

### Decision 1: Registry Pattern for Retrievers

**Chosen Approach:** Implement the same registry pattern used in `pkg/embeddings`

**Rationale:**
- Consistency across the framework
- Enables dynamic provider registration
- Supports plugin-style extensibility
- Allows users to add custom retrievers without modifying framework code

**Alternative Considered:** Keep using direct constructor functions
- Rejected because it doesn't allow runtime provider discovery

### Decision 2: Provider Directory Structure

**Chosen Approach:** Move implementations to `providers/{name}/` subdirectories

**Rationale:**
- Matches pattern in `pkg/embeddings/providers/openai/`, `pkg/vectorstores/providers/inmemory/`
- Each provider is self-contained with its own config.go, init.go
- Clean separation between interface (iface/) and implementations (providers/)

### Decision 3: Memory Internal vs Providers

**Chosen Approach:**
- Move actual memory providers (buffer, summary, window, vectorstore, redis) to `providers/`
- Move base/shared implementations to `internal/base/`

**Rationale:**
- `internal/` should contain truly internal implementation details
- `providers/` should contain swappable provider implementations
- `internal/base/` is for shared base classes used by providers

### Decision 4: Clean Break vs Backward Compatibility

**Chosen Approach:** Clean break with no type aliases

**Rationale:**
- This is v2.0.0-beta, breaking changes are expected
- Type aliases add complexity and maintenance burden
- Clear migration path: update imports to new paths

## Constraints

1. **Must maintain public API compatibility** - Existing factory functions like `NewVectorStoreRetriever()` must continue to work
2. **Must pass all existing tests** - No functionality regression
3. **Must follow OTEL integration patterns** - Metrics and tracing remain functional
4. **Must not introduce import cycles** - Provider packages can import iface/ but not parent package

## Risks

1. **Import path changes may break external users** - Mitigated by v2.0.0-beta status
2. **Circular dependencies** - Careful interface design in iface/ package prevents this
3. **Test coverage gaps** - Run full test suite after changes

## Success Criteria

1. All tests pass (`make test-unit`)
2. Linter passes (`make lint`)
3. Providers auto-register via init()
4. ListProviders() returns expected providers
5. Create() works with registered provider names
