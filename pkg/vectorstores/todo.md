# Vectorstores Package Redesign TODO

## Goals
- Ensure the package is consistent with Beluga AI Framework principles (ISP, DIP, SRP, Composition over Inheritance).
- Make it easy to use, configure, and extensible.
- Address inconsistency in providers while leveraging the existing good iface/.

## Tasks

1. **Standardize Providers**
   - Move or organize providers into a consistent structure under providers/ if multiple backends, or internal/ if single.
   - Ensure all providers implement the interfaces from iface/.

2. **Add Central Factory for Stores**
   - Create a factory function (e.g., NewVectorStore) in vectorstores.go that takes a config and options to instantiate the appropriate provider.
   - Use functional options for flexibility.

3. **Ensure Config Validation for All Providers**
   - Define or update Config struct in config.go with appropriate tags (mapstructure, yaml, env, validate).
   - Implement validation using the validator library at creation time in the factory.

4. **Add Metrics for Add/Retrieve Operations**
   - Create metrics.go if not present.
   - Define OTEL counters/histograms for operations like add, retrieve, with relevant labels.
   - Instrument public methods with these metrics.

5. **Integrate with Embeddings and Retrievers**
   - Ensure VectorStore interfaces compose well with Embedder and Retriever interfaces.
   - Add any necessary adapter or integration logic.
   - Embed interfaces for backward compatibility if extending.

6. **Add Custom Error Codes**
   - Define custom error types in a errors.go file if needed, with codes like ErrCodeIndexNotFound, ErrCodeRateLimit, etc.
   - Use them in provider implementations for common failures.
   - Ensure errors are wrapped and context cancellation is respected.

7. **Update Tests**
   - Update existing tests to cover new factory, configs, and metrics.
   - Add table-driven tests for each provider.
   - Generate or update mocks in internal/mock/ for dependencies.
   - Include benchmarks for performance-critical operations.

8. **Document Providers in README.md**
   - Create or update README.md in the package directory.
   - Include sections on usage, configuration examples, available providers, and integration with other packages.
   - Add package-level comments and function docs with examples.

## Additional Considerations
- Follow SemVer for any changes; deprecate old APIs if necessary with migration guides.
- Use code generation for mocks, validation, and metrics where applicable.
- Ensure observability: Add tracing and structured logging to public methods.
- Verify health checks if the package supports them.
