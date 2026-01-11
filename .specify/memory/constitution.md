# Beluga AI Framework Constitution

## Core Principles

### I. Package Structure (MUST)
All packages MUST follow the standard v2 structure:
- `pkg/{package_name}/iface/` - Interfaces and types (REQUIRED)
- `pkg/{package_name}/internal/` - Private implementation details
- `pkg/{package_name}/providers/` - Provider implementations (for multi-provider packages)
- `pkg/{package_name}/config.go` - Configuration structs and validation (REQUIRED)
- `pkg/{package_name}/metrics.go` - OTEL metrics implementation (REQUIRED)
- `pkg/{package_name}/errors.go` - Custom error types with Op/Err/Code pattern (REQUIRED)
- `pkg/{package_name}/{package_name}.go` - Main interfaces and factory functions
- `pkg/{package_name}/factory.go` OR `registry.go` - Global factory/registry for multi-provider packages
- `pkg/{package_name}/test_utils.go` - Advanced testing utilities and mocks (REQUIRED)
- `pkg/{package_name}/advanced_test.go` - Comprehensive test suites (REQUIRED)
- `pkg/{package_name}/README.md` - Package documentation (REQUIRED)

### II. Interface Design (MUST)
All packages MUST follow Interface Segregation Principle (ISP):
- Define small, focused interfaces that serve specific purposes
- Avoid "god interfaces" that force implementations to depend on unused methods
- Prefer multiple small interfaces over one large interface
- Use "er" suffix pattern where appropriate (e.g., Embedder, Caller)
- Follow Dependency Inversion Principle (DIP): depend on abstractions, use constructor injection

### III. Provider Registry Pattern (MUST)
Multi-provider packages MUST use global registry pattern:
- `GetRegistry()` function for global registry access
- `Register()` method for provider registration
- `Create()` method for provider instantiation
- Provider registration in `providers/*/init.go` files
- Match existing patterns from `pkg/llms/` and `pkg/embeddings/`

### IV. OTEL Observability (MUST)
All packages MUST include comprehensive observability:
- OTEL metrics in `metrics.go` (counters, histograms for latency, throughput)
- OTEL tracing for all public methods with span attributes
- Structured logging with OTEL context (trace IDs, span IDs)
- Use `logWithOTELContext` helper function following framework patterns
- Record errors with `span.RecordError()` and set status codes

### V. Error Handling (MUST)
All packages MUST use custom error types:
- Error struct with Op, Err, Code, Message fields
- Error codes for common failures (provider_not_found, invalid_config, etc.)
- Context cancellation support
- Error helper functions: `NewError`, `WrapError`, `IsError`, `AsError`

### VI. Configuration (MUST)
All packages MUST use structured configuration:
- Config struct with mapstructure, yaml, env, validate tags
- Functional options for runtime configuration
- Validation at creation time using validator library
- Default values where appropriate

### VII. Testing (MUST)
All packages MUST include comprehensive tests:
- Table-driven tests in `advanced_test.go`
- Mocks in `internal/mock/` or `test_utils.go`
- Benchmarks for performance-critical operations
- Integration tests for cross-package compatibility
- Test coverage >80% for all packages

### VIII. Backward Compatibility (MUST)
All new features MUST maintain backward compatibility:
- No breaking changes to existing APIs
- New functionality is opt-in where possible
- Deprecation notices with migration guides for any planned breaking changes

## Framework Design Patterns

### Core Principles
- **Interface Segregation Principle (ISP)**: Small, focused interfaces
- **Dependency Inversion Principle (DIP)**: Depend on abstractions; use constructor injection
- **Single Responsibility Principle (SRP)**: One responsibility per package/struct
- **Composition over Inheritance**: Embed interfaces; use functional options

### Package Naming Conventions
- Use lowercase, descriptive names: `llms`, `vectorstores`, `embeddings`
- Avoid abbreviations unless widely understood (e.g., `llms` is acceptable)
- Use singular forms: `agent` not `agents`, `tool` not `tools`

## Governance

**Constitution Authority**: This constitution supersedes all other practices and guidelines. All feature specifications, implementation plans, and tasks MUST comply with these principles.

**Compliance Verification**: 
- All PRs/reviews must verify compliance with framework patterns
- Complexity must be justified if deviating from standard patterns
- Package design patterns are documented in `docs/package_design_patterns.md`

**Amendments**: 
- Constitution changes require documentation, approval, and migration plan
- Feature-specific workarounds must be documented in feature specifications
- See `specs/{feature}/plan.md` "Constitution Check" section for feature-specific compliance

**Version**: 1.0.0 | **Ratified**: 2025-01-27 | **Last Amended**: 2025-01-27
