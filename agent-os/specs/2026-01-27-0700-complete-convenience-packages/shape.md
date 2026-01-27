# Shape: Complete WIP Convenience Packages

## Problem

The Beluga AI framework has three convenience packages that are currently stubs:
- `pkg/convenience/agent/` - Builder pattern exists but no Build() implementation
- `pkg/convenience/rag/` - Builder pattern exists but no Build() implementation
- `pkg/convenience/voiceagent/` - Builder pattern exists but no Build() implementation

Users cannot use these packages in production because they don't actually create working instances.

## Appetite

2-3 days of focused implementation work.

## Solution

### Approach: Composition over Inheritance

Each convenience package will:
1. Accept provider instances directly OR resolve them from the global registry
2. Compose the underlying framework components
3. Wrap them in a simplified interface

### Key Design Decisions

**1. Builder Pattern with Build() Method**

The Build() method returns a concrete interface (not a struct) to allow:
- Easy mocking in tests
- Future implementation changes without breaking consumers

```go
agent, err := agent.NewBuilder().
    WithLLM(llm).
    WithBufferMemory(50).
    Build(ctx)
```

**2. Provider Resolution Strategy**

Support both:
- Direct injection: `WithLLM(existingLLM)`
- Registry resolution: `WithLLMProvider("openai", apiKey)`

This gives users flexibility while maintaining simplicity.

**3. Memory Integration**

The agent package integrates memory transparently:
- Load history before each Run()
- Save context after each Run()
- Optionally expose raw memory for advanced use cases

### Boundaries

**In Scope:**
- Build() methods that create working instances
- Error handling with Op/Err/Code pattern
- OTEL metrics integration
- Unit tests and integration tests

**Out of Scope:**
- New provider implementations
- Changes to existing pkg/agents/, pkg/memory/, etc.
- Documentation website updates (just README.md)

### Rabbit Holes to Avoid

1. **Over-abstracting memory**: Use the existing memory interfaces directly
2. **Complex configuration**: Keep the builder methods simple, advanced config goes in underlying packages
3. **Provider auto-detection**: Require explicit provider specification

## Technical Context

### Existing Patterns to Follow

From `pkg/agents/`:
- `AgentError` with Op/Err/Code pattern
- `Metrics` with OTEL counters and histograms
- Builder pattern with functional options

From `pkg/vectorstores/`:
- In-memory vector store for simple RAG
- Factory pattern with registry

From `pkg/voicesession/`:
- Session management with state machine
- Callback-based event handling

### No-Gos

- Don't modify existing package interfaces
- Don't add dependencies on external services
- Don't break existing tests
