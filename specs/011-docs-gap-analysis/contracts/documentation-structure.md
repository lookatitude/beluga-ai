# Documentation Structure Contract

**Feature**: Documentation Gap Analysis and Resource Creation  
**Date**: 2025-01-27  
**Phase**: 1 - Design

## Overview

This contract defines the structure and organization requirements for all documentation resources created as part of this feature.

## Directory Structure

### Root Documentation (`docs/`)

```
docs/
├── guides/              # Step-by-step tutorials
│   ├── llm-providers.md
│   ├── agent-types.md
│   ├── multimodal-embeddings.md
│   ├── voice-providers.md
│   ├── orchestration-graphs.md
│   ├── tools-mcp.md
│   ├── rag-multimodal.md
│   ├── config-providers.md
│   ├── observability-tracing.md
│   ├── concurrency.md
│   ├── extensibility.md
│   └── memory-entity.md
├── cookbook/            # Quick-reference recipes
│   ├── llm-error-handling.md
│   ├── custom-agent.md
│   ├── memory-window.md
│   ├── rag-prep.md
│   ├── multimodal-streaming.md
│   ├── voice-backends.md
│   ├── orchestration-concurrency.md
│   ├── custom-tools.md
│   ├── text-splitters.md
│   └── benchmarking.md
└── use-cases/           # Real-world scenarios
    ├── batch-processing.md
    ├── event-driven-agents.md
    ├── memory-backends.md
    ├── custom-vectorstore.md
    ├── multimodal-providers.md
    ├── voice-sessions.md
    ├── distributed-orchestration.md
    ├── tool-monitoring.md
    ├── rag-strategies.md
    ├── config-overrides.md
    └── performance-optimization.md
```

### Code Examples (`examples/`)

```
examples/
├── llms/
│   └── streaming/          # Streaming LLM examples
│       ├── README.md
│       ├── streaming_tool_call.go
│       └── streaming_tool_call_test.go
├── agents/
│   └── planexecute/         # PlanExecute agent examples
│       ├── README.md
│       ├── planexecute_agent.go
│       └── planexecute_agent_test.go
├── memory/
│   ├── summary/            # Summary memory examples
│   └── vector_store/       # Vector store memory examples
├── vectorstores/
│   └── advanced_retrieval/ # Advanced retrieval examples
├── rag/
│   ├── multimodal/         # Multimodal RAG examples
│   └── evaluation/         # RAG evaluation examples
├── orchestration/
│   └── resilient/          # Resilient orchestration examples
├── tools/
│   └── custom_chains/      # Custom tool chain examples
├── config/
│   └── formats/            # Config format examples
├── monitoring/
│   └── metrics/            # Metrics examples
└── deployment/
    └── single_binary/      # Deployment examples
```

## File Naming Conventions

- **Guides**: `{feature}-{topic}.md` (e.g., `llm-providers.md`, `agent-types.md`)
- **Cookbooks**: `{feature}-{task}.md` (e.g., `llm-error-handling.md`, `custom-agent.md`)
- **Use Cases**: `{use-case-name}.md` (e.g., `batch-processing.md`, `event-driven-agents.md`)
- **Examples**: `{example-name}.go` and `{example-name}_test.go`

## Documentation Template Requirements

### Guide Template

```markdown
# {Title}

## Introduction
[Overview of the feature/topic]

## Prerequisites
[Required knowledge, dependencies, setup]

## Concepts
[Key concepts and background]

## Step-by-Step Tutorial
[Detailed steps with code examples]

## Code Examples
[Complete, production-ready examples]

## Testing
[Testing patterns and examples]

## Best Practices
[Recommended patterns and approaches]

## Troubleshooting
[Common issues and solutions]

## Related Resources
- [Link to related guide]
- [Link to related example]
- [Link to related cookbook]
- [Link to related use case]
```

### Cookbook Template

```markdown
# {Title}

## Problem
[Problem statement]

## Solution
[Solution overview]

## Code Example
[Focused code snippet]

## Explanation
[Brief explanation]

## Testing
[How to test the solution]

## Related Recipes
- [Link to related recipe]
```

### Use Case Template

```markdown
# {Title}

## Overview
[Use case overview]

## Business Context
[Business problem or need]

## Requirements
[Functional and non-functional requirements]

## Architecture
[Architecture overview]

## Implementation
[Implementation steps with code]

## Results
[Results and outcomes]

## Lessons Learned
[Key takeaways]

## Related Use Cases
- [Link to related use case]
```

### Example Template

```markdown
# {Example Name}

## Description
[What this example demonstrates]

## Prerequisites
[Required dependencies, API keys, configuration]

## Usage
[How to run the example]

## Expected Output
[What to expect when running]

## Code Structure
[Overview of code organization]

## Testing
[How to run tests]

## Related Examples
- [Link to related example]
```

## Code Example Requirements

### Source Code (`*.go`)

- Must be production-ready with full error handling
- Must include OTEL instrumentation with standardized naming
- Must demonstrate DI via functional options
- Must follow SOLID principles
- Must include clear comments explaining each section

### Test Suite (`*_test.go`)

- Must include complete, passing test suite
- Must use test_utils.go patterns (AdvancedMock, MockOption, ConcurrentTestRunner)
- Must include unit tests, integration tests, and error handling tests
- Must include table-driven tests for multiple scenarios
- Should include benchmarks for performance-critical examples

### README (`README.md`)

- Must include description, prerequisites, usage instructions
- Must include expected output
- Must include testing instructions
- Must include related examples

## Cross-Referencing Requirements

- All guides must include "Related Resources" section
- All cookbooks must include "Related Recipes" section
- All use cases must include "Related Use Cases" section
- All examples must include "Related Examples" section
- Links must use relative paths
- Links must include brief description

## Validation Rules

1. All documentation must follow template structure
2. All code examples must be production-ready
3. All code examples must include complete test suites
4. All documentation must include cross-references
5. All documentation must be verified to work with current framework version
