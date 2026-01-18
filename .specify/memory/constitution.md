<!--
Sync Impact Report:
- Version change: 1.0.0 → 1.1.0
- Summary: Updated constitution to reflect condensed framework rules and remove redundant details.
- Modified Principles:
  - Consolidates Package Structure, Interface Design, Provider Registry, OTEL, Error Handling, Configuration, Testing, and Backward Compatibility into core mandatory principles.
  - Adds a Framework Overview section summarizing philosophy and extensibility.
  - Updates Governance section to reflect current active technologies and standards.
- Templates Validated:
  - ✅ plan-template.md (Constitution Check logic remains valid)
  - ✅ spec-template.md (Standard requirement sections align)
  - ✅ tasks-template.md (Task categorization aligns)
-->
# Beluga AI Framework Constitution

## Framework Overview

**Beluga AI** is an enterprise-grade, standardized Go framework designed for building scalable AI agents and RAG applications. It emphasizes consistency, observability, and modularity.

**Core Philosophy**:
- **Go-Native**: Built for concurrency and performance.
- **Production-Ready**: Enterprise patterns, comprehensive testing, and full observability.
- **Extensible & Modular**: Interface-driven design for easy component swapping.

## Core Principles

### I. Package Structure (MUST)
All packages MUST follow the standard structure to ensure consistency:
- `pkg/{package}/iface/` - Interfaces and public types (REQUIRED)
- `pkg/{package}/internal/` - Private implementation details
- `pkg/{package}/providers/` - Implementations for multi-provider packages
- `pkg/{package}/config.go` - Configuration structs with validation (REQUIRED)
- `pkg/{package}/metrics.go` - OTEL metrics implementation (REQUIRED)
- `pkg/{package}/errors.go` - Custom error types with Op/Err/Code (REQUIRED)
- `pkg/{package}/{package}.go` - Main interfaces and factory functions
- `pkg/{package}/factory.go` or `registry.go` - Global registry for providers
- `pkg/{package}/test_utils.go` - Advanced mocks and test helpers (REQUIRED)
- `pkg/{package}/advanced_test.go` - Comprehensive test suites (REQUIRED)
- `pkg/{package}/README.md` - Package documentation (REQUIRED)

### II. Interface Design (MUST)
All packages MUST adhere to interface design best practices:
- **ISP**: Define small, focused interfaces (e.g., `Embedder`, `Retriever`).
- **DIP**: Depend on abstractions, not implementations; use constructor injection.
- **Composition**: Embed interfaces and structs; prefer composition over inheritance.
- **Naming**: Use `-er` suffix for single-method interfaces, noun-based for multi-method.

### III. Provider Registry (MUST)
Multi-provider packages (e.g., `llms`, `embeddings`) MUST implement the Global Registry pattern:
- Thread-safe `ProviderRegistry` with `sync.RWMutex`.
- `RegisterGlobal(name, creator)` function.
- `NewProvider(ctx, name, config)` factory function.
- Providers reside in `providers/{name}/` and register via `init()`.

### IV. Observability (MUST)
All packages MUST use OpenTelemetry (OTEL) as the **ONLY** observability solution:
- **Metrics**: Standardized counters/histograms in `metrics.go` (`{pkg}_operations_total`).
- **Tracing**: Spans for all public methods with attributes and error recording.
- **Logging**: Structured logs containing `trace_id` and `span_id`.
- **NO** custom metric implementations allowed.

### V. Error Handling (MUST)
All packages MUST implement standardized error handling:
- Custom error structs with `Op` (operation), `Err` (underlying), and `Code` (programmatic).
- Standard error codes constants (e.g., `ErrCodeRateLimit`, `ErrCodeTimeout`).
- Context-aware: Respect context cancellation and timeouts.
- Helper functions: `NewError`, `WrapError`.

### VI. Configuration (MUST)
All packages MUST use standardized configuration management:
- Config structs in `config.go` with tags (`mapstructure`, `yaml`, `env`, `validate`).
- Validation at creation time using `go-playground/validator/v10`.
- Functional options pattern for runtime configuration (e.g., `WithTimeout`).
- Support for environment variables and configuration files (Viper).

### VII. Testing (MUST)
All packages MUST implement comprehensive testing strategies:
- **Unit**: Table-driven tests in `advanced_test.go`.
- **Mocks**: Advanced mock implementations in `test_utils.go`.
- **Concurrency**: `ConcurrentTestRunner` to verify thread safety.
- **Integration**: Cross-package tests in `tests/integration/`.
- **Coverage**: Aim for 100% coverage on critical paths; minimum 80%.

### VIII. Knowledge Base (MUST)
All development work MUST integrate with the project Knowledge Base:
- **Query**: Check `.project-kb` context before starting work.
- **Update**: Add new patterns, files, and features to the KB after changes.
- **Link**: Establish metadata relationships between packages in KB entries.

## Governance

**Constitution Authority**: This constitution supersedes all other guidelines. All features and code MUST comply with these principles.

**Compliance Verification**:
- PRs must verify alignment with `beluga-design-patterns.mdc`.
- Architecture changes require updates to `beluga-core-architecture.mdc`.
- Quality checks (`make lint`, `make test`) are mandatory.

**Version**: 1.1.0 | **Ratified**: 2025-01-27 | **Last Amended**: 2026-01-18
