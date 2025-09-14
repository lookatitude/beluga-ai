
# Server Package Redesign TODO

This file outlines the planned changes for redesigning the server package to make it consistent, easy to use, configurable, and extensible, while adhering to Beluga AI Framework principles (ISP, DIP, SRP, composition over inheritance, etc.). No implementation changes will be made until these are reviewed and approved.

## Core Goals
- Expose all components (e.g., agents, workflows) through the server.
- Ensure consistency with framework patterns: small interfaces, dependency injection, configuration validation, observability.
- Make the package easy to configure and extend (e.g., support for multiple server types like MCP and REST).
- Integrate observability, proper error handling, and testing.

## Planned Changes

1. **Add config.go**
   - Define configuration structs for server types (MCP, REST).
   - Include tags for mapstructure, yaml, env, and validate.
   - Add validation logic at creation time.
   - Use functional options for runtime configuration.

2. **Implement Factories for Servers**
   - Create factory functions (e.g., NewServer) with options pattern.
   - Support different server types via interfaces and composition.
   - Embed interfaces for extensibility and backward compatibility.

3. **Integrate with Monitoring for API Metrics**
   - Add metrics.go to define OTEL metrics (counters, histograms) for API calls.
   - Instrument public methods with traces and metrics.
   - Include structured logging with context and trace IDs.
   - Implement HealthChecker interface if applicable.

4. **Use Schema for Request/Response Formats**
   - Define schemas (e.g., using structs with JSON tags) for all API requests and responses.
   - Ensure consistency across endpoints.

5. **Add Streaming Support for REST**
   - Implement streaming capabilities for REST endpoints where appropriate (e.g., for real-time data).
   - Use appropriate Go libraries or patterns for handling streams.

6. **Enhance Errors for API Failures**
   - Define custom error types with codes (e.g., ErrCodeInvalidRequest).
   - Wrap errors and provide meaningful messages.
   - Respect context cancellation in all operations.

7. **Add Tests for Integrations**
   - Create table-driven tests in server_test.go.
   - Add mocks in internal/mock/ for dependencies.
   - Include benchmarks for performance-critical parts.
   - Test integrations with other components (agents, workflows).

8. **Document Endpoints in README.md**
   - Update or create README.md with setup, usage, and endpoint documentation.
   - Include examples for configuration and API calls.
   - Describe package purpose, interfaces, and extension points.

## Additional Considerations
- Follow standard package structure: iface/, internal/, providers/ if needed.
- Ensure no breaking changes; use deprecation notices if necessary.
- Validate all configs at startup.
- Group imports properly and inject dependencies via constructors.

These changes will be implemented step-by-step after confirmation.
