# Config Package Redesign TODO

This TODO list outlines the necessary changes to redesign the config package for better extensibility, integration, and adherence to Beluga AI Framework principles (ISP, DIP, SRP, Composition over Inheritance). The redesign aims to make the package more consistent, easy to use and configure, while enhancing observability and error handling. Implementation will follow the framework's standard package structure, configuration management, observability, error handling, testing, and documentation standards.

## 1. Add Schema-Based Validation
- Integrate with the schema package to define config structs as schemas.
- Update config structs in config.go to use schema definitions for validation.
- Modify validation logic at creation time to leverage schema-based validation instead of or in addition to existing validator library tags (e.g., validate:"required").
- Ensure validation respects functional options and defaults.

## 2. Enhance Providers
- Extend existing providers (e.g., viper) to support additional formats: JSON, TOML (in addition to YAML if not already supported).
- Add support for remote loading providers, such as etcd, Consul, or similar key-value stores.
- Place provider implementations in a providers/ subdirectory if multiple backends are introduced (following multi-provider package structure).
- Use composition to allow easy addition of new providers without modifying core logic.

## 3. Add Metrics for Config Loading
- Define new metrics in metrics.go, including:
  - load_duration_seconds (histogram for measuring config load times).
  - Other relevant metrics like config_load_success_total (counter), config_load_errors_total (counter with labels for error types).
- Instrument public methods (e.g., Load functions) to record these metrics using OTEL.

## 4. Integrate with Monitoring for Tracing
- Add OTEL tracing spans to config loading methods, including attributes for provider type, format, and success/failure.
- Ensure traces include context propagation and error handling.
- Integrate structured logging with trace IDs during config operations.

## 5. Make Loader More Composable
- Redesign the Loader interface to support chaining multiple providers (e.g., fallback from remote to local).
- Use composition (e.g., decorator pattern or functional options) to chain loaders.
- Update factory functions (e.g., NewLoader) to accept options for configuring provider chains.
- Ensure the design follows DIP by depending on abstractions for providers.

## 6. Add Error Types for Config-Specific Failures
- Define custom error types in a new errors.go file or within config.go, such as:
  - ErrConfigNotFound (for missing config files or keys).
  - ErrValidationFailed (with details from schema validation).
  - ErrProviderUnavailable (for remote loading failures).
  - Other codes like ErrCodeInvalidFormat, ErrCodeRemoteLoadTimeout.
- Use error wrapping and respect context cancellation in all operations.

## 7. Update Tests
- Add table-driven tests in config_test.go for new schema-based validations, covering valid/invalid cases and edge scenarios.
- Include tests for new providers (formats and remote loading), using mocks in internal/mock/ if needed.
- Add tests for metrics emission and tracing integration.
- Incorporate benchmarks for performance-critical parts like loading large configs.
- Ensure tests cover composable loaders and error handling scenarios.

## 8. Documentation Updates
- Update package-level comments to reflect new features and usage.
- Add function docs with examples for new APIs (e.g., chaining loaders, schema validation).
- Create or update README.md in pkg/config/ to document advanced usage, including:
  - Configuring remote configs (e.g., etcd setup).
  - Examples of provider chaining.
  - Schema definition and validation usage.
  - Metrics and tracing integration.

## Additional Considerations
- Maintain backward compatibility: Embed interfaces for extensions, deprecate old methods if needed with migration notes.
- Follow SemVer for versioning; document any breaking changes.
- Use code generation where applicable (e.g., for mocks, validation helpers).
- Review all changes against core principles: Ensure small interfaces, dependency injection, single responsibilities, and composition.

Once this TODO is complete, the config package will be more extensible for framework-wide use, with improved validation, observability, and composability.
