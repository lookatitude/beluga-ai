# Memory Package Redesign TODO

The memory package needs to be updated to align with Beluga AI Framework design patterns: consistent, easy to use, configurable, extensible, with proper observability and error handling. Do not change implementations until these tasks are planned and approved.

## Tasks

- [ ] Add config.go for all memory types (e.g., BufferConfig embeds base Config). Ensure configs use structs with tags (mapstructure, yaml, env, validate) and functional options for runtime config. Validate configs at creation.

- [ ] Add metrics.go for memory operations (e.g., load_duration_seconds histogram, save_counter). Use OTEL metrics with appropriate labels.

- [ ] Integrate observability: Add OTEL traces and spans for public methods like saves/loads, including error handling. Use structured logging with context/trace IDs.

- [ ] Add custom error codes (e.g., ErrMemoryOverflow, ErrCodeInvalidConfig) in a errors.go file if needed, following the framework's error handling patterns: custom types with Op, Err, Code; wrap errors; respect context cancellation.

- [ ] Make histories composable (e.g., allow chaining multiple histories for advanced use cases). Use composition over inheritance.

- [ ] Update providers (e.g., base_history) with factory functions (e.g., NewBaseHistory with options).

- [ ] Add comprehensive tests: Table-driven tests for configs, metrics, and operations. Include mocks in internal/mock/ for dependencies. Add benchmarks for performance-critical parts.

- [ ] Document memory types, configurations, and extensions in README.md. Include package comments, function docs with examples.

- [ ] Ensure interface design follows ISP: small, focused interfaces (e.g., Memory interface). Use embedding for compatibility and extensions.

- [ ] Review for SRP and DIP: Ensure single responsibility, depend on abstractions via constructor injection.

- [ ] If multiple backends are needed in the future, prepare for providers/ directory structure.

Once all tasks are completed, verify against the framework's review checklist: config validation, health checks (if applicable), docs, no breaking changes.
