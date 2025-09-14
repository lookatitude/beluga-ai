# Chatmodels Package Redesign TODO

This TODO lists the necessary changes to redesign the chatmodels package for consistency, ease of use, configurability, and extensibility. It should adhere to Beluga AI Framework principles: ISP (small interfaces), DIP (depend on abstractions), SRP (one responsibility), and composition over inheritance. The package should compose with llms for shared logic where appropriate.

## Structural Changes
- **Add Central Factory**: Implement a central factory in chatmodels.go (e.g., NewChatModel) that dispatches to specific providers based on configuration. Use functional options for flexibility.
- **Provider Structure**: Ensure providers (e.g., openai, mock) are organized under providers/ if multiple backends exist, or internal/ for single-provider. Add factories for each provider with consistent options.
- **Interface Composition**: Merge redundant interfaces with llms package. For example, have ChatModel embed llms.LLM for composition and shared logic, ensuring backward compatibility.

## Configuration and Options
- **Enhance Config**: Update config.go to include structs with tags (mapstructure, yaml, env, validate). Ensure validation occurs at creation time.
- **Functional Options**: Make options functional and consistent across providers (e.g., WithTemperature, WithMaxTokens) that apply universally. Use these in factories.

## Observability and Monitoring
- **Enhance Metrics**: In metrics.go, add OTEL histograms for response times and counters for message generations. Include labels for providers and error types.
- **Tracing Integration**: Add integration with monitoring for tracing chat sessions, including spans for public methods with attributes and error handling.
- **Health Checks**: Ensure all providers implement the HealthChecker interface.

## Error Handling
- **Add Error Codes**: Create errors.go with custom error types, codes (e.g., ErrProviderUnavailable, ErrCodeRateLimit), and wrapping for provider-specific failures. Respect context cancellation.

## Integration and Consistency
- **Schema Integration**: Integrate with schema package for consistent message formats. Add adapters if needed to handle provider-specific formats.
- **Dependencies**: Group imports, inject via constructors, and specify versions. Depend on abstractions from llms where possible.

## Testing
- **Update Tests**: Expand tests in chatmodels_test.go to cover new factories, options, metrics, error handling, and health checks. Use table-driven tests and mocks from internal/mock/.

## Documentation
- **Update README.md**: Document provider registration, extension processes, usage examples, configuration options, and how to add new providers.

## General Guidelines
- Follow SemVer for changes; deprecate with notices if needed.
- Use code generation for mocks, validation, and metrics where applicable.
- Ensure all changes promote extensibility (e.g., easy addition of new providers) and configurability without breaking existing code.

Once these changes are planned, implement them step-by-step without altering existing implementations prematurely.
