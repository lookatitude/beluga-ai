Sequence of / Commands to Execute
Paste these commands one-by-one into your AI agent (e.g., Claude) after project init. They form a logical sequence: framework principles first, then per-package analysis/planning/correction. I've included detailed prompts based on the attached files (e.g., standards from package_design_patterns.md, current status from README.md). Replace <attach-files> with actual file attachments if the AI supports it.

Establish Framework-Wide Principles (Based on Core Principles, Package Structure, etc., from package_design_patterns.md):
text/constitution Establish principles for the Beluga AI Framework based on the attached package_design_patterns.md and README.md. Include: Interface Segregation (small focused interfaces), Dependency Inversion (abstractions and injection), Single Responsibility (one responsibility per package/struct), Composition over Inheritance (functional options). Enforce standard package structure (iface/, internal/, providers/, config.go, metrics.go, errors.go, factory.go/registry.go, test_utils.go, advanced_test.go). Mandate OTEL for observability (metrics/tracing/logging in metrics.go). Require global registry for multi-provider packages. Enforce testing (100% coverage, mocks, concurrency, benchmarks, integration). Standardize error handling (Op/Err/Code). Ensure no loss of functionality, flexibility, or ease of use in modernizations. All packages must comply with the 100% implementation status table.

# For Package: core (Analyze current from README.md: Foundational utilities, DI container; Ensure standards adherence):
text/specify For the 'core' package in Beluga AI Framework: Analyze current implementation (foundational utilities, dependency injection container, core model definitions) from attached README.md. Identify gaps in standards from package_design_patterns.md (e.g., OTEL metrics, registry if multi-provider, testing suites). Specify desired state: Fully adhere to all patterns (structure, interfaces, config, errors, metrics, testing). Preserve all functionalities (utils, DI, models), flexibility (extensible), and ease of use (simple APIs). Include user stories for core usage in AI workflows.

text/plan For the 'core' package: Create implementation plan to correct gaps and adhere to standards. Use Go best practices, OTEL integration, functional options. Detail file structure updates, code changes. Ensure backward compatibility, no functionality loss.

text/tasks For the 'core' package: Generate actionable tasks from the plan, sequenced with dependencies (e.g., update structure first, then add tests).

text/implement For the 'core' package: Execute all tasks to update the package.

# For Package: schema (Current: Centralized data structures with validation; Ensure standards):
text/specify For the 'schema' package: Analyze current (centralized data structures, validation, type safety, message/document testing utilities) from README.md. Identify standards gaps (e.g., metrics.go, advanced_test.go). Specify adherence: Add required 
files/structure, OTEL, testing. Preserve schema functionalities, flexibility for extensions.

text/plan For the 'schema' package: Plan corrections with Go interfaces, validation tags. Include integration test additions.

text/tasks For the 'schema' package: Break into tasks (e.g., add metrics, mocks).

text/implement For the 'schema' package: Execute tasks.

# For Package: config (Current: Advanced config management with validation; Ensure standards):
text/specify For the 'config' package: Analyze current (validation, env vars, defaults, provider testing) from README.md. Gaps in patterns (e.g., registry if applicable, full testing). Specify full compliance, preserve config flexibility.

text/plan For the 'config' package: Plan with struct tags, viper integration updates.

text/tasks For the 'config' package: Tasks for structure, OTEL, tests.

text/implement For the 'config' package: Execute.

# For Package: llms (Current: Unified interfaces with providers, testing, OTEL; Gold standard per doc):
text/specify For the 'llms' package: Analyze current (LLM interface, providers like OpenAI/Anthropic, factory, comprehensive testing) from README.md. Minimal gaps as it's compliant; specify any minor corrections. Preserve multi-provider flexibility.

text/plan For the 'llms' package: Plan minor updates if needed, focus on benchmarks.

text/tasks For the 'llms' package: Tasks for verification/enhancements.

text/implement For the 'llms' package: Execute.

# For Package: chatmodels (Current: ChatModel interface with providers, runnables):
text/specify For the 'chatmodels' package: Analyze current (interface, OpenAI provider, mocks, integration). Gaps in standards. Specify adherence, preserve runnable impl.

text/plan For the 'chatmodels' package: Plan with global registry, OTEL.

text/tasks For the 'chatmodels' package: Tasks.

text/implement For the 'chatmodels' package: Execute.

# For Package: embeddings (Current: Embedder interface with providers, registry):
text/specify For the 'embeddings' package: Analyze current (OpenAI/Ollama, global registry, performance testing). Ensure full patterns.

text/plan For the 'embeddings' package: Plan corrections.

text/tasks For the 'embeddings' package: Tasks.

text/implement For the 'embeddings' package: Execute.

# For Package: vectorstores (Current: VectorStore interface with providers, factory):
text/specify For the 'vectorstores' package: Analyze current (InMemory/PgVector/Pinecone, similarity testing). Gaps? Specify standards.

text/plan For the 'vectorstores' package: Plan.

text/tasks For the 'vectorstores' package: Tasks.

text/implement For the 'vectorstores' package: Execute.

# For Package: memory (Current: Memory interface with types, registry):
text/specify For the 'memory' package: Analyze current (Buffer/Summary/VectorStoreMemory, integration). Specify adherence.

text/plan For the 'memory' package: Plan.

text/tasks For the 'memory' package: Tasks.

text/implement For the 'memory' package: Execute.

# For Package: retrievers (Current: Retriever interface with runnables, vector integration):
text/specify For the 'retrievers' package: Analyze current (relevance testing, benchmarks). Gaps in structure/tests.

text/plan For the 'retrievers' package: Plan.

text/tasks For the 'retrievers' package: Tasks.

text/implement For the 'retrievers' package: Execute.

# For Package: agents (Current: Agent framework with tools, registry):
text/specify For the 'agents' package: Analyze current (ReAct agents, tool integration, execution testing). Ensure patterns.

text/plan For the 'agents' package: Plan.

text/tasks For the 'agents' package: Tasks.

text/implement For the 'agents' package: Execute.

# For Package: prompts (Current: Prompt templates, OTEL, testing):
text/specify For the 'prompts' package: Analyze current (dynamic loading, rendering). Specify corrections.

text/plan For the 'prompts' package: Plan.

text/tasks For the 'prompts' package: Tasks.

text/implement For the 'prompts' package: Execute.

# For Package: orchestration (Current: Workflow engine with support, metrics):
text/specify For the 'orchestration' package: Analyze current (Chain/Graph/Workflow, concurrent testing). Gaps?

text/plan For the 'orchestration' package: Plan.

text/tasks For the 'orchestration' package: Tasks.

text/implement For the 'orchestration' package: Execute.

# For Package: server (Current: REST/MCP servers, load testing):
text/specify For the 'server' package: Analyze current (streaming, health monitoring). Specify adherence.

text/plan For the 'server' package: Plan.

text/tasks For the 'server' package: Tasks.

text/implement For the 'server' package: Execute.

# For Package: monitoring (Current: Observability suite with OTEL):
text/specify For the 'monitoring' package: Analyze current (logging/metrics/tracing, cross-package). Likely compliant; specify verifications.

text/plan For the 'monitoring' package: Plan minor updates.

text/tasks For the 'monitoring' package: Tasks.

text/implement For the 'monitoring' package: Execute.


# Post-Sequence Steps
Review generated branches/files in the repo.
Run tests: go test ./pkg/... -v and integration tests as per README.md.
Merge features: Use Git to merge branches into main.
If issues, use free-form AI prompts or /clarify for refinements.