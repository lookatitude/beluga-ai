# Agent Examples

This directory contains examples demonstrating agent creation, configuration, and usage in the Beluga AI Framework.

## Examples

### [basic](basic/)

Basic agent creation and execution without tools or memory.

**What you'll learn:**
- Creating a simple agent
- Basic agent execution
- Error handling

**Run:**
```bash
cd basic
go run main.go
```

### [with_tools](with_tools/)

Agent with tool integration for performing actions.

**What you'll learn:**
- Creating tools
- Registering tools with agents
- Tool execution flow

**Run:**
```bash
cd with_tools
go run main.go
```

### [react](react/)

ReAct (Reasoning + Acting) agent implementation.

**What you'll learn:**
- ReAct pattern implementation
- Reasoning loop
- Tool usage in ReAct

**Run:**
```bash
cd react
go run main.go
```

### [with_memory](with_memory/)

Agent with buffer memory for conversation history.

**What you'll learn:**
- Memory integration
- Conversation history management
- Multi-turn conversations

**Run:**
```bash
cd with_memory
go run main.go
```

### [vector_memory](vector_memory/)

Agent with vector store memory for semantic retrieval.

**What you'll learn:**
- Vector store memory setup
- Semantic memory retrieval
- Context augmentation

**Run:**
```bash
cd vector_memory
go run main.go
```

## Prerequisites

- Go 1.21 or later
- Beluga AI Framework
- (Optional) OpenAI API key for real LLM providers

## Learning Path

1. Start with `basic` to understand fundamentals
2. Try `with_tools` to learn tool integration
3. Explore `react` for advanced reasoning
4. Add `with_memory` for conversation context
5. Use `vector_memory` for semantic memory

## Related Documentation

- [Agent Concepts](../../docs/concepts/agents.md)
- [Agent Recipes](../../docs/cookbook/agent-recipes.md)
- [Package Documentation](../../pkg/agents/README.md)
