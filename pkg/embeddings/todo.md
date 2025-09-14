# Embeddings Package Redesign TODO

This TODO outlines the necessary changes to redesign the embeddings package for consistency, ease of use, configuration, and extensibility. Changes will align with Beluga AI Framework principles: ISP, DIP, SRP, composition over inheritance, standard package structure, observability, error handling, etc. No implementation changes will be made until this TODO is reviewed and approved.

## 1. Standardize Package Structure
- Adopt the standard package layout:
  ```
  pkg/embeddings/
  ├── iface/           # Interfaces (e.g., embedder.go)
  ├── internal/        # Private implementation details
  │   ├── openai/
  │   ├── ollama/
  │   └── mock/
  ├── providers/       # Provider registrations and factories (if multi-backend)
  ├── config.go        # Config structs with validation
  ├── metrics.go       # OTEL metrics definitions
  ├── embeddings.go    # Main interfaces, central factory
  ├── embeddings_test.go # Tests
  └── README.md        # Documentation
  ```
- Move existing provider implementations (openai, ollama, mock) to internal/ if not already.
- Use providers/ only if supporting multiple backends; otherwise, keep in internal/.

## 2. Interface Design
- Define a focused Embedder interface in iface/embedder.go (or embeddings.go if simple):
  - Methods: Embed(ctx context.Context, text string) ([]float64, error), EmbedBatch(ctx context.Context, texts []string) ([][]float64, error), GetDimension() int.
  - Make GetDimension consistent and required across all providers.
- Use composition: Allow embedding interfaces for extensions without breaking changes.
- Ensure interfaces are small and follow ISP.

## 3. Central Factory
- Add a central factory in embeddings.go, similar to llms package:
  ```go
  // ... existing code ...
  func NewEmbedder(caller EmbedderCaller, opts ...Option) (Embedder, error) { /* to be implemented */ }
  ```
- Support functional options pattern for flexibility.
- Factory should handle provider selection based on config (e.g., "openai", "ollama", "mock").

## 4. Configuration Management
- Standardize config in config.go:
  - Struct with tags: mapstructure, yaml, env, validate.
  - Include provider-specific fields (e.g., APIKey for openai, Model for ollama).
  - Add validation using validator library at creation time.
- Ensure all providers (openai, ollama, mock) use this config struct.
- Implement functional options for runtime configuration.

## 5. Observability and Monitoring
- Add metrics.go with OTEL metrics:
  - Counters: embeddings_generated (labels: provider, success).
  - Histograms: embedding_duration_seconds (labels: provider, batch_size).
- Integrate tracing: Add OTEL spans to public methods (Embed, EmbedBatch) with attributes (e.g., provider, text_length).
- Add structured logging with context/trace IDs.
- If applicable, implement HealthChecker interface for providers.

## 6. Error Handling
- Define custom error types in embeddings.go or errors.go:
  - Struct: type Error struct { Op string, Err error, Code string }
  - Codes: ErrCodeInvalidConfig, ErrCodeRateLimit, ErrEmbeddingDimensionMismatch, etc.
- Wrap errors and use codes for common failures.
- Respect context cancellation in all operations.

## 7. Provider Consistency
- Ensure openai, ollama, mock providers:
  - Use config validation.
  - Implement the Embedder interface fully (including GetDimension).
  - Use options pattern in their constructors.
- Make mock provider configurable for testing (e.g., simulate dimensions, errors).

## 8. Testing Updates
- Update embeddings_test.go:
  - Table-driven tests for factories, config validation, and each provider.
  - Add tests for metrics emission and tracing.
  - Use mocks from internal/mock/ if needed.
  - Include benchmarks for performance-critical parts (e.g., EmbedBatch).

## 9. Documentation
- Update README.md:
  - Explain package purpose, usage, and configuration.
  - Detail how to extend with new providers (e.g., implementing Embedder, registering in factory).
  - Include examples for creating embedders with different providers.
  - Cover error codes and observability integration.

## Additional Considerations
- Dependency Management: Group imports, inject via constructors, specify versions.
- Code Generation: Generate mocks and metrics where possible.
- Evolution: Ensure changes are backward-compatible; deprecate old factories if needed with migration notes.
- Review: Verify config validation, health checks, docs, no breaking changes before implementation.

Once this TODO is complete, proceed to implement changes in phases, starting with structure and interfaces.
