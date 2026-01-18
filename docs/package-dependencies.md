# Package Dependencies Map

**Generated**: $(date -u +%Y-%m-%dT%H:%M:%SZ)
**Purpose**: Map all direct package dependencies for integration testing

## Foundation Packages (No Dependencies)

### pkg/schema
- **Direct Dependencies**: None (foundation package)
- **Integration Test Targets**: None (no dependencies to test)
- **Used By**: All packages

### pkg/core
- **Direct Dependencies**: 
  - `pkg/config` (configuration utilities)
  - `pkg/schema` (core data structures)
  - `pkg/monitoring` (OTEL observability)
- **Integration Test Targets**:
  - `pkg/core` ↔ `pkg/config`
  - `pkg/core` ↔ `pkg/schema`
  - `pkg/core` ↔ `pkg/monitoring`

### pkg/config
- **Direct Dependencies**:
  - `pkg/core` (utilities)
  - `pkg/schema` (data structures)
- **Integration Test Targets**:
  - `pkg/config` ↔ `pkg/core`
  - `pkg/config` ↔ `pkg/schema`

### pkg/monitoring
- **Direct Dependencies**:
  - `pkg/core` (utilities)
  - `pkg/schema` (data structures)
- **Integration Test Targets**:
  - `pkg/monitoring` ↔ `pkg/core`
  - `pkg/monitoring` ↔ `pkg/schema`

## Provider Packages

### pkg/llms
- **Direct Dependencies**:
  - `pkg/schema` (message types)
  - `pkg/config` (configuration)
  - `pkg/monitoring` (OTEL)
  - `pkg/core` (utilities)
- **Integration Test Targets**:
  - `pkg/llms` ↔ `pkg/memory` (conversation history)
  - `pkg/llms` ↔ `pkg/prompts` (prompt management)
  - `pkg/llms` ↔ `pkg/schema` (message types)

### pkg/embeddings
- **Direct Dependencies**:
  - `pkg/schema` (document types)
  - `pkg/config` (configuration)
  - `pkg/monitoring` (OTEL)
- **Integration Test Targets**:
  - `pkg/embeddings` ↔ `pkg/vectorstores` (storage)
  - `pkg/embeddings` ↔ `pkg/schema` (document types)

### pkg/vectorstores
- **Direct Dependencies**:
  - `pkg/schema` (document types)
  - `pkg/config` (configuration)
  - `pkg/monitoring` (OTEL)
  - `pkg/embeddings` (embeddings)
- **Integration Test Targets**:
  - `pkg/vectorstores` ↔ `pkg/embeddings` (embedding generation)
  - `pkg/vectorstores` ↔ `pkg/memory` (memory storage)
  - `pkg/vectorstores` ↔ `pkg/schema` (document types)

### pkg/prompts
- **Direct Dependencies**:
  - `pkg/schema` (message types)
  - `pkg/config` (configuration)
- **Integration Test Targets**:
  - `pkg/prompts` ↔ `pkg/llms` (LLM integration)
  - `pkg/prompts` ↔ `pkg/schema` (message types)

## Higher-Level Packages

### pkg/chatmodels
- **Direct Dependencies**:
  - `pkg/llms` (LLM providers)
  - `pkg/schema` (message types)
  - `pkg/prompts` (prompt management)
- **Integration Test Targets**:
  - `pkg/chatmodels` ↔ `pkg/llms` (LLM integration)
  - `pkg/chatmodels` ↔ `pkg/memory` (conversation history)
  - `pkg/chatmodels` ↔ `pkg/schema` (message types)

### pkg/memory
- **Direct Dependencies**:
  - `pkg/schema` (message types)
  - `pkg/config` (configuration)
  - `pkg/monitoring` (OTEL)
  - `pkg/vectorstores` (vector storage)
  - `pkg/embeddings` (embeddings)
- **Integration Test Targets**:
  - `pkg/memory` ↔ `pkg/vectorstores` (storage)
  - `pkg/memory` ↔ `pkg/embeddings` (embeddings)
  - `pkg/memory` ↔ `pkg/schema` (message types)

### pkg/retrievers
- **Direct Dependencies**:
  - `pkg/vectorstores` (vector storage)
  - `pkg/embeddings` (embeddings)
  - `pkg/schema` (document types)
  - `pkg/core` (utilities)
- **Integration Test Targets**:
  - `pkg/retrievers` ↔ `pkg/vectorstores` (storage)
  - `pkg/retrievers` ↔ `pkg/embeddings` (embeddings)
  - `pkg/retrievers` ↔ `pkg/schema` (document types)

### pkg/agents
- **Direct Dependencies**:
  - `pkg/llms` (LLM providers)
  - `pkg/schema` (message types)
  - `pkg/config` (configuration)
  - `pkg/monitoring` (OTEL)
  - `pkg/core` (utilities)
  - `pkg/memory` (conversation memory)
- **Integration Test Targets**:
  - `pkg/agents` ↔ `pkg/llms` (LLM integration)
  - `pkg/agents` ↔ `pkg/memory` (memory integration)
  - `pkg/agents` ↔ `pkg/orchestration` (workflow integration)

### pkg/orchestration
- **Direct Dependencies**:
  - `pkg/agents` (agent execution)
  - `pkg/schema` (message types)
  - `pkg/config` (configuration)
  - `pkg/monitoring` (OTEL)
  - `pkg/core` (utilities)
- **Integration Test Targets**:
  - `pkg/orchestration` ↔ `pkg/agents` (agent coordination)
  - `pkg/orchestration` ↔ `pkg/schema` (message types)

### pkg/server
- **Direct Dependencies**:
  - `pkg/agents` (agent execution)
  - `pkg/orchestration` (workflow management)
  - `pkg/schema` (message types)
- **Integration Test Targets**:
  - `pkg/server` ↔ `pkg/agents` (agent API)
  - `pkg/server` ↔ `pkg/orchestration` (workflow API)

### pkg/messaging
- **Direct Dependencies**:
  - `pkg/orchestration` (workflow management)
  - `pkg/schema` (message types)
- **Integration Test Targets**:
  - `pkg/messaging` ↔ `pkg/orchestration` (message bus)

### pkg/multimodal
- **Direct Dependencies**:
  - `pkg/llms` (LLM providers)
  - `pkg/agents` (agent framework)
  - `pkg/schema` (multimodal types)
- **Integration Test Targets**:
  - `pkg/multimodal` ↔ `pkg/llms` (LLM integration)
  - `pkg/multimodal` ↔ `pkg/agents` (agent integration)

### pkg/documentloaders
- **Direct Dependencies**:
  - `pkg/textsplitters` (text splitting)
  - `pkg/embeddings` (embeddings)
  - `pkg/schema` (document types)
- **Integration Test Targets**:
  - `pkg/documentloaders` ↔ `pkg/textsplitters` (text splitting)
  - `pkg/documentloaders` ↔ `pkg/embeddings` (embeddings)
  - `pkg/documentloaders` ↔ `pkg/schema` (document types)

### pkg/textsplitters
- **Direct Dependencies**:
  - `pkg/schema` (document types)
- **Integration Test Targets**:
  - `pkg/textsplitters` ↔ `pkg/schema` (document types)

### pkg/voice
- **Direct Dependencies**:
  - `pkg/agents` (agent framework)
  - `pkg/llms` (LLM providers)
  - `pkg/memory` (conversation memory)
  - `pkg/orchestration` (workflow management)
- **Integration Test Targets**:
  - `pkg/voice/backend` ↔ `pkg/agents` (agent integration)
  - `pkg/voice/backend` ↔ `pkg/llms` (LLM integration)
  - `pkg/voice/s2s` ↔ `pkg/llms` (LLM integration)

## Summary

### Integration Test Coverage Requirements

Each package must have integration tests covering:
1. **Direct dependencies** (packages it imports)
2. **Provider integrations** (for multi-provider packages)
3. **Schema usage** (for packages using pkg/schema)

### Test File Locations

Integration tests should be placed in:
- `tests/integration/package_pairs/{package1}_{package2}_test.go`
- `tests/integration/{package}/` (for package-specific integration tests)

### Priority Order

1. **Foundation packages first** (schema, core, config, monitoring)
2. **Provider packages** (llms, embeddings, vectorstores, prompts)
3. **Higher-level packages** (agents, orchestration, server, etc.)
