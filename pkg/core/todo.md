# Core Package Redesign TODO

This TODO list details the necessary changes to redesign the core package in the Beluga AI Framework. The goal is to make the core package more consistent, easy to use and configure, and extensible, while addressing gaps in observability and error standardization. All changes should adhere to the framework's core principles: ISP, DIP, SRP, and composition over inheritance. Incorporate observability (OTEL traces/metrics, structured logging), custom error handling, and proper testing/documentation as per the design patterns.

## 1. Add Tracing to Runnable
- Implement automatic OTEL tracing spans for key methods in the Runnable interface, such as Invoke, Batch, and Stream.
- Ensure spans include relevant attributes (e.g., input/output sizes, runnable type) and handle errors appropriately.
- Use composition to embed tracing functionality without altering existing implementations, allowing for easy extension.
- Make tracing configurable via functional options in the Runnable factory (e.g., WithTracer).

## 2. Standardize Errors Across the Framework
- Move utils/errors.go to either the schema or core package (evaluate based on usage; prefer core if it's foundational).
- Define custom error types with operation (Op), underlying error (Err), and error codes (e.g., constants like ErrCodeInvalidInput).
- Update all core methods to return standardized errors, wrapping underlying errors and respecting context cancellation.
- Ensure errors are extensible by allowing packages to embed and extend core error types.
- Add validation for error codes in config structs where applicable.

## 3. Enhance DI Container with Monitoring Integration
- Add OTEL tracing for dependency resolutions in the DI container (e.g., trace Resolve and Provide methods).
- Integrate structured logging with context and trace IDs for DI operations.
- Make monitoring configurable (e.g., via options like WithTracer, WithLogger) to allow users to enable/disable or customize.
- Ensure the DI container remains extensible by using small interfaces and constructor injection for its own dependencies.

## 4. Add Metrics for Runnable Executions
- Define metrics in a new metrics.go file, using OTEL (e.g., counter for runnable_invocations_total with labels for runnable type, success/failure).
- Add histograms for execution duration (e.g., runnable_execution_duration_seconds).
- Instrument public methods of Runnable to record these metrics automatically.
- Make metrics collection optional and configurable via functional options in factories.

## 5. Ensure All Packages Implement core.Runnable Where Appropriate
- Review all framework packages (e.g., llms, embeddings) and implement core.Runnable interface where it fits for orchestration purposes.
- Use embedding for compatibility and to avoid breaking changes.
- Provide factory functions in each package that return Runnable-compatible instances.
- Document implementation guidelines in each package's README to ensure consistency and extensibility.

## 6. Add Health Checks to Core Components
- Implement the HealthChecker interface (from observability standards) for core components like the DI container.
- Add health check methods that verify dependencies, configurations, and basic functionality (e.g., DI container can resolve a test dependency).
- Make health checks configurable and integrable with monitoring tools.
- Ensure health checks are non-intrusive and can be extended by embedding.

## 7. Update Tests for Tracing and Metrics
- Add table-driven tests in core_test.go for new tracing and metrics functionality.
- Use mocks (in internal/mock/) for dependencies like tracers and metrics collectors.
- Include benchmarks for performance-critical parts (e.g., invocation with tracing).
- Test error standardization, health checks, and configurability scenarios.
- Ensure tests cover edge cases and validate extensibility (e.g., custom options).

## 8. Document Core as the "Glue" Layer
- Update README.md in the core package to describe it as the foundational "glue" layer that orchestrates components, provides DI, and ensures observability.
- Include usage examples, configuration guides, and extension points.
- Add package-level comments and function docs with examples for all new/updated APIs.
- Provide migration guides if changes affect existing users, following SemVer.

## Overall Considerations
- **Consistency**: Follow standard package layout (e.g., config.go, metrics.go, internal/ for private impl).
- **Ease of Use/Configuration**: Use config structs with validation tags and functional options throughout.
- **Extensibility**: Favor small interfaces, embedding, and factories to allow easy overrides and additions.
- **Validation**: Ensure all new configs are validated at creation time.
- **No Breaking Changes**: Deprecate old APIs if needed, with notices and migration paths.
- **Code Generation**: Use tools for generating mocks, validation, and metrics where possible.

Track progress by checking off items as they are completed. Prioritize observability additions first, followed by error standardization and testing.
