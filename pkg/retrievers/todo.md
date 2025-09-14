# Retrievers Package Redesign TODO

## Overview
The current retrievers package is basic and requires alignment with the vectorstores package and Beluga AI Framework design patterns. This includes enhancing structure, configuration, observability, and extensibility while maintaining consistency, ease of use, and configurability. No implementation changes should be made yet; this TODO outlines the planned updates.

## Alignment with Framework Principles
- Adhere to ISP (small interfaces), DIP (depend on abstractions), SRP (one responsibility), and composition over inheritance.
- Ensure backward compatibility through interface embedding.
- Use functional options for configuration.
- Incorporate observability (OTEL traces/metrics, structured logging) in all public methods.

## Structural Changes
- Verify and adjust package layout to match standard:
  - iface/ for interfaces (already present).
  - internal/ for private details (already present).
  - providers/ for multi-backend implementations (e.g., add more like keyword or hybrid retrievers if needed; currently has mock).
  - config.go for configs (already present; enhance as needed).
  - metrics.go for metrics (already present; add more).
  - errors.go for custom errors (add if missing).
  - retrievers.go for main factories/interfaces (already present).
  - retrievers_test.go for tests (already present; update).

## Configuration Enhancements
- Expand config.go to support various retriever types.
- Add structs with tags for mapstructure, yaml, env, validate.
- Implement validation at creation time.
- Add functional options for runtime configuration.

## Factory Implementations
- Enhance or add factories like NewRetriever, NewVectorStoreRetriever with options pattern.
- Ensure factories support multiple providers via registration (similar to vectorstores).

## Metrics and Observability
- Add specific metrics for retrievals, e.g., retrieval_duration_seconds histogram, retrieval_success_count counter.
- Integrate monitoring: OTEL spans for methods, structured logging with context.
- Add health checks if applicable.

## Error Handling
- Add custom error types with codes, e.g., ErrNoDocumentsFound, ErrRetrievalFailed.
- Ensure errors are wrapped and respect context cancellation.

## Composition and Integration
- Ensure seamless composition with embeddings and vectorstores.
- Implement or enhance AsRetriever methods or similar for easy integration.

## Testing Updates
- Update tests to use table-driven patterns.
- Add mocks in internal/mock/ if not present.
- Include benchmarks for performance-critical parts.

## Documentation
- Update README.md with detailed usage, configuration examples, extensibility guide, and migration notes.

## Additional Goals
- Make the package more extensible for new retriever types.
- Ensure ease of use and configuration.
- Follow SemVer for evolution; deprecate thoughtfully.

## Next Steps
- Review current implementation against this TODO.
- Plan implementation in phases to avoid breaking changes.
- Generate any necessary code (mocks, validation) using tools.
