# Orchestration Package Redesign TODO

The orchestration package is currently incomplete, with many "path_only" files, and requires a full implementation aligned with the Beluga AI Framework's goals and core principles (ISP, DIP, SRP, composition over inheritance). The package must be consistent, easy to use, configurable, and extensible.

## Key Changes and Implementations Needed:

1. **Full Implementation of Core Components**:
   - Implement Chain, Graph, and Workflow interfaces with corresponding factories (e.g., NewChain, NewGraph, NewWorkflow) using functional options for flexibility.

2. **Configuration Management**:
   - Add config.go containing configuration structs for orchestration types, with tags for mapstructure, yaml, env, and validate. Include validation at creation time.

3. **Observability Integration**:
   - Integrate with monitoring for tracing workflows, adding OTEL spans in public methods.
   - Add metrics in metrics.go for executions, such as workflow_duration_seconds histogram with appropriate labels.

4. **Schema Usage**:
   - Use schema definitions for input/output formats to ensure consistency and validation.

5. **Additional Components**:
   - Implement messagebus and scheduler as per existing outlines, ensuring they follow the framework's patterns.

6. **Error Handling**:
   - Add custom error codes (e.g., ErrWorkflowDeadlock) in a dedicated errors.go file, with proper wrapping and context cancellation respect.

7. **Composition and Integration**:
   - Ensure composition with other packages like agents and llms (e.g., allowing agents as steps in workflows).

8. **Testing**:
   - Add comprehensive tests in orchestration_test.go, including table-driven tests and mocks for end-to-end orchestration scenarios.

9. **Documentation**:
   - Document patterns (chains vs graphs vs workflows) in README.md, including usage examples and setup instructions.

All changes must adhere to the framework's standards: small interfaces, dependency injection, observability by default, and SemVer compatibility.
