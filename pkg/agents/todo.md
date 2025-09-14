# Agents Package Redesign TODO

This TODO list outlines the necessary changes to redesign the agents package in the Beluga AI Framework. The redesign aims to fully integrate with monitoring, configuration, server exposure, and orchestration while adhering to core principles (ISP, DIP, SRP, composition over inheritance). The package must be consistent, easy to use and configure, and extensible. Changes are prioritized for minimal disruption, using factories, options patterns, and embedding for compatibility.

## 1. Interface and Composition Enhancements
- Ensure all agent interfaces embed `core.Runnable` to enable composability with chains and graphs in the orchestration package.
- Review and refine interfaces in `iface/` to be small and focused (ISP); use embedding for backward compatibility and extensibility.

## 2. Factory Pattern Implementation
- Add factories for agent creation, e.g., `NewReActAgent(caller LLMCaller, opts ...Option) (*Agent, error)`, fully utilizing the functional options pattern for flexibility.
- Ensure factories inject dependencies via constructors (DIP) and validate configurations at creation.

## 3. Configuration Integration
- Integrate with the config package for dynamic loading of agent configs from YAML, env, or other sources.
- Define `Config` struct in `config.go` with appropriate tags (`mapstructure`, `yaml`, `env`, `validate`) and defaults.
- Add validation using the validator library, triggered in factories.

## 4. Observability and Metrics
- Create `metrics.go` with OTEL counters and histograms, e.g.:
  - `agent_executions_total` (counter for agent runs, labeled by agent type).
  - `tool_calls_total` (counter for tool invocations, labeled by tool and success).
  - `agent_errors_total` (counter for errors, labeled by error code).
  - Histograms for execution durations.
- Integrate monitoring for tracing agent lifecycles, e.g., spans for Plan, Execute, and tool calls in public methods.
- Add structured logging with context and trace IDs.

## 5. Error Handling Improvements
- Enhance `errors.go` with additional custom error codes, e.g., `ErrAgentTimeout`, `ErrToolFailure`, using structs with Op, Err, Code.
- Implement error wrapping for better tracing and respect context cancellation (e.g., check `ctx.Done()`).

## 6. Server Exposure
- Expose agents via the server package, allowing agents to be served as MCP servers or REST APIs with streaming support.
- Implement necessary handlers or providers in `providers/` if multiple backends are needed, ensuring integration with server internals.

## 7. Agent-to-Agent (A2A) Protocol
- Define A2A protocol using the schema package, e.g., create `AgentMessage` schema for standardized message formats.
- Implement protocol handling in agents for inter-agent communication, ensuring extensibility.

## 8. Health Checks
- Have agents implement the `HealthChecker` interface for monitoring health status.
- Include health check logic in factories or public methods where applicable.

## 9. Testing
- Add unit tests for new factories, configurations, metrics, errors, and integrations in `agents_test.go`.
- Use table-driven tests and mocks from `internal/mock/` for dependencies like LLM callers or tools.
- Include benchmarks for performance-critical paths (e.g., agent execution).

## 10. Documentation
- Update or create `README.md` with examples of:
  - Agent composition (e.g., embedding in orchestration workflows).
  - Configuration loading and factory usage.
  - Server exposure (e.g., setting up an agent as a REST API).
  - A2A protocol usage.
- Ensure package-level comments explain purpose, and function docs include params, returns, errors, and examples.

## Additional Considerations
- Maintain standard package structure: `iface/`, `internal/`, `providers/` (if multi-backend), `config.go`, `metrics.go`, etc.
- Follow SemVer for changes; deprecate old APIs with notices and migration guides if needed.
- Verify no breaking changes; use composition for extensibility.
- Automate where possible: Generate mocks, validation, and metrics using code gen tools.

Once all items are addressed, review against the framework's guidelines checklist for compliance.
