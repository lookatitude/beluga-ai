# LLMs Package Redesign TODO

This TODO list outlines the changes needed to redesign the LLMs package. The goal is to unify it with chatmodels, improve composition, add necessary features for observability, error handling, testing, and documentation, while adhering to Beluga AI Framework principles (ISP, DIP, SRP, composition over inheritance). Changes will ensure the package is consistent, easy to use and configure, and extensible. No implementation changes will be made until this plan is reviewed.

## 1. Unification with ChatModels
- Analyze current LLMs and ChatModels structures to identify overlaps and differences.
- Redesign ChatModel to compose LLM (e.g., embed LLM interface in ChatModel for better composition).
- Ensure backward compatibility by embedding interfaces where possible.

## 2. Factories for Providers
- Add factory functions in llms.go for all providers (e.g., NewOpenAILLM, NewAnthropicLLM, etc.).
- Factories should accept Config structs and functional options for flexibility.
- Include validation in factories using the validator library.

## 3. Enhance Metrics
- Update metrics.go to include new metrics for token usage (e.g., counters for input/output tokens).
- Add metrics for generations (e.g., histograms for generation latency, counters for successful/failed generations).
- Ensure metrics are OTEL-compatible with appropriate labels (e.g., provider, model).

## 4. Integration with Schema
- Integrate schema package for handling tool calls and structured messages.
- Update generate/stream methods to support schema-based inputs/outputs (e.g., add parameters for tool schemas).
- Ensure providers handle schema integrations consistently.

## 5. Add Tracing
- Implement OTEL tracing for generate and stream calls in all providers.
- Add spans with attributes (e.g., provider, model, input size) and handle errors in traces.
- Include trace IDs in structured logging.

## 6. Standardize Errors
- Define custom error types and codes in a new errors.go file (e.g., ErrCodeRateLimit, ErrCodeInvalidInput).
- Update all providers to use standardized error wrapping and codes.
- Ensure errors respect context cancellation and timeouts.

## 7. Implement HealthChecker
- Ensure all providers implement the HealthChecker interface.
- Add health check methods that verify API connectivity or basic functionality.
- Integrate health checks into factories or config validation where appropriate.

## 8. Update Tests
- Review and update existing tests for cross-provider consistency (e.g., table-driven tests covering all providers).
- Add new tests for unification, metrics, tracing, errors, and schema integration.
- Generate or update mocks in internal/mock/ for new interfaces or changes.
- Include benchmarks for performance-critical paths like generate/stream.

## 9. Documentation
- Create or update README.md to document the package as the core AI interaction layer.
- Include usage examples, configuration guides, provider setup, and extension points.
- Add package-level comments and function docs with examples for all public APIs.

## Additional Consistency and Extensibility Tasks
- Review package structure against standard layout: ensure config.go, metrics.go, providers/ (if multi-backend), etc.
- Group imports and inject dependencies via constructors.
- Use SemVer for versioning; plan deprecations if needed with migration guides.
- Automate code generation for mocks, validation, and metrics where possible.

Once this TODO is complete, proceed to implement changes in phases, starting with interface updates and unification.
