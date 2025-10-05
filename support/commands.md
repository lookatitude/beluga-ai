# Sequence of / Commands to Execute
Paste these commands one-by-one into your AI agent (e.g., Claude) after project init. They form a logical sequence: framework principles first, then per-package analysis/planning/correction. I've included detailed prompts based on the attached files (e.g., standards from package_design_patterns.md, current status from README.md). Replace <attach-files> with actual file attachments if the AI supports it.

## Implementation Note
Each feature implementation will automatically include the standardized post-implementation workflow (commit, push, PR creation, merge to develop) as defined in the specify system templates.

## Framework Establishment

Establish Framework-Wide Principles (Based on Core Principles, Package Structure, etc., from package_design_patterns.md):

```
/constitution Establish principles for the Beluga AI Framework based on the attached package_design_patterns.md and README.md. Include: Interface Segregation (small focused interfaces), Dependency Inversion (abstractions and injection), Single Responsibility (one responsibility per package/struct), Composition over Inheritance (functional options). Enforce standard package structure (iface/, internal/, providers/, config.go, metrics.go, errors.go, factory.go/registry.go, test_utils.go, advanced_test.go). Mandate OTEL for observability (metrics/tracing/logging in metrics.go). Require global registry for multi-provider packages. Enforce testing (100% coverage, mocks, concurrency, benchmarks, integration). Standardize error handling (Op/Err/Code). Ensure no loss of functionality, flexibility, or ease of use in modernizations. All packages must comply with the 100% implementation status table.
```

# 1- For Package: core (Analyze current from README.md: Foundational utilities, DI container; Ensure standards adherence):
```
/specify For the 'core' package in Beluga AI Framework: Analyze current implementation (foundational utilities, dependency injection container, core model definitions) from attached README.md. Identify gaps in standards from package_design_patterns.md (e.g., OTEL metrics, registry if multi-provider, testing suites). Specify desired state: Fully adhere to all patterns (structure, interfaces, config, errors, metrics, testing). Preserve all functionalities (utils, DI, models), flexibility (extensible), and ease of use (simple APIs). Include user stories for core usage in AI workflows.
```

```/plan For the 'core' package: Create implementation plan to correct gaps and adhere to standards. Use Go best practices, OTEL integration, functional options. Detail file structure updates, code changes. Ensure backward compatibility, no functionality loss.
```

```
/tasks For the 'core' package: Generate actionable tasks from the plan, sequenced with dependencies (e.g., update structure first, then add tests).
```
```
/implement For the 'core' package: Execute all tasks to update the package.
```

# 2 - For Package: schema (Current: Centralized data structures with validation; Ensure standards):
```
/specify For the 'schema' package: Analyze current (centralized data structures, validation, type safety, message/document testing utilities) from README.md. Identify standards gaps (e.g., metrics.go, advanced_test.go). Specify adherence: Add required 
files/structure, OTEL, testing. Preserve schema functionalities, flexibility for extensions.
```

```
/plan For the 'schema' package: Plan corrections with Go interfaces, validation tags. Include integration test additions.
```

```
/tasks For the 'schema' package: Break into tasks (e.g., add metrics, mocks).
```

```
/implement For the 'schema' package: Execute tasks.
```

# 3 - For Package: config (Current: Advanced config management with validation; Ensure standards):
```
/specify For the 'config' package: Analyze current (validation, env vars, defaults, provider testing) from README.md. Gaps in patterns (e.g., registry if applicable, full testing). Specify full compliance, preserve config flexibility.
```


```
/plan For the 'config' package: Plan with struct tags, viper integration updates.
```
```
/tasks For the 'config' package: Tasks for structure, OTEL, tests.
```
```
/implement For the 'config' package: Execute.
```

# 4 - For Package: llms (Current: Unified interfaces with providers, testing, OTEL; Gold standard per doc):
```
/specify For the 'llms' package: Analyze current (LLM interface, providers like OpenAI/Anthropic, factory, comprehensive testing) from README.md. Minimal gaps as it's compliant; specify any minor corrections. Preserve multi-provider flexibility.
```

```
/plan For the 'llms' package: Plan minor updates if needed, focus on benchmarks.
```
```
/tasks For the 'llms' package: Tasks for verification/enhancements.
```

```
/implement For the 'llms' package: Execute.
```

# 5 - For Package: chatmodels (Current: ChatModel interface with providers, runnables):
```
/specify For the 'chatmodels' package: Analyze current (interface, OpenAI provider, mocks, integration). Gaps in standards. Specify adherence, preserve runnable impl.
```

```
/plan For the 'chatmodels' package: Plan with global registry, OTEL.
```

```
/tasks For the 'chatmodels' package: Tasks.
```

```
/implement For the 'chatmodels' package: Execute.
```

# 6 - For Package: embeddings (Current: Embedder interface with providers, registry):
```
/specify For the 'embeddings' package: Analyze current (OpenAI/Ollama, global registry, performance testing). Ensure full patterns.
```

```
/plan For the 'embeddings' package: Plan corrections.
```

```
/tasks For the 'embeddings' package: Tasks.
```

```
/implement For the 'embeddings' package: Execute.
```

# 7 - For Package: vectorstores (Current: VectorStore interface with providers, factory):
```
/specify For the 'vectorstores' package: Analyze current (InMemory/PgVector/Pinecone, similarity testing). Gaps? Specify standards.
```

```
/plan For the 'vectorstores' package: Plan.
```

```
/tasks For the 'vectorstores' package: Tasks.
```

```
/implement For the 'vectorstores' package: Execute.
```

# 8 - For Package: memory (Current: Memory interface with types, registry):
```
/specify For the 'memory' package: Analyze current (Buffer/Summary/VectorStoreMemory, integration). Specify adherence.
```

```
text/plan For the 'memory' package: Plan.
```

```
text/tasks For the 'memory' package: Tasks.
```
```
text/implement For the 'memory' package: Execute.
```

# 9 - For Package: retrievers (Current: Retriever interface with runnables, vector integration):
```
text/specify For the 'retrievers' package: Analyze current (relevance testing, benchmarks). Gaps in structure/tests.
```

```
text/plan For the 'retrievers' package: Plan.
```

```
text/tasks For the 'retrievers' package: Tasks.
```

```
text/implement For the 'retrievers' package: Execute.
```

# 10 - For Package: agents (Current: Agent framework with tools, registry):
```
text/specify For the 'agents' package: Analyze current (ReAct agents, tool integration, execution testing). Ensure patterns.
```

```
text/plan For the 'agents' package: Plan.
```

```
text/tasks For the 'agents' package: Tasks.
```

```
text/implement For the 'agents' package: Execute.
```

# 11 - For Package: prompts (Current: Prompt templates, OTEL, testing):
```
text/specify For the 'prompts' package: Analyze current (dynamic loading, rendering). Specify corrections.
```

```
text/plan For the 'prompts' package: Plan.
```

```
text/tasks For the 'prompts' package: Tasks.
```

```
text/implement For the 'prompts' package: Execute.
```

# 12 - For Package: orchestration (Current: Workflow engine with support, metrics):
text/specify For the 'orchestration' package: Analyze current (Chain/Graph/Workflow, concurrent testing). Gaps?

text/plan For the 'orchestration' package: Plan.

text/tasks For the 'orchestration' package: Tasks.

text/implement For the 'orchestration' package: Execute.

# 13 - For Package: server (Current: REST/MCP servers, load testing):
text/specify For the 'server' package: Analyze current (streaming, health monitoring). Specify adherence.

text/plan For the 'server' package: Plan.

text/tasks For the 'server' package: Tasks.

text/implement For the 'server' package: Execute.

# 14 - For Package: monitoring (Current: Observability suite with OTEL):
text/specify For the 'monitoring' package: Analyze current (logging/metrics/tracing, cross-package). Likely compliant; specify verifications.

text/plan For the 'monitoring' package: Plan minor updates.

text/tasks For the 'monitoring' package: Tasks.

text/implement For the 'monitoring' package: Execute.

# Post-Sequence Steps

## Final Integration and Validation
1. **Review all feature branches**: Verify all 14 packages have been implemented and merged to `develop`
2. **Run comprehensive tests**: 
   ```bash
   go test ./pkg/... -v
   go test ./tests/integration/... -v
   go test ./tests/contract/... -v
   ```
3. **Validate constitutional compliance**: Ensure all packages follow the established standards
4. **Performance benchmarks**: Run package-specific benchmarks to verify performance targets
5. **Final merge to main**: Once all packages are complete and tested, merge `develop` to `main`
6. **Documentation update**: Update main README.md with the completed constitutional compliance status

## Troubleshooting
- **For implementation issues**: Use free-form AI prompts or `/clarify` for refinements
- **For merge conflicts**: Resolve conflicts maintaining constitutional compliance
- **For test failures**: Address failures before proceeding with PR merges
- **For performance issues**: Review benchmarks and optimize as needed

## Branch Management
- **Feature branches**: Named as `XXX-for-the-<package>` (e.g., `001-for-the-core`)
- **Development branch**: `develop` (integration branch for all features)
- **Main branch**: `main` (production-ready code after full validation)
- **PR workflow**: Feature → `develop` → `main`