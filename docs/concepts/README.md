# Beluga AI Framework - Concepts Guide

This guide explains the core concepts and architectural patterns used in the Beluga AI Framework. Understanding these concepts will help you build more effective AI applications.

## Table of Contents

1. [Core Concepts](./core.md) - Runnable interface, messages, context, dependency injection
2. [LLM Concepts](./llms.md) - Provider abstraction, streaming, tool calling, batch processing
3. [Agent Concepts](./agents.md) - Agent lifecycle, planning, execution, ReAct pattern
4. [Memory Concepts](./memory.md) - Memory types, conversation history, persistence
5. [RAG Concepts](./rag.md) - Retrieval-augmented generation, embeddings, vector stores
6. [Orchestration Concepts](./orchestration.md) - Chains, graphs, workflows, task scheduling

### [Design Patterns](./patterns/README.md)
Implementation patterns, guidelines, and real-world examples for extending the framework.

### [Architecture](./architecture/README.md)
System-level architecture documentation, component diagrams, and data flows.

## How to Use This Guide

### For Beginners

Start with [Core Concepts](./core.md) to understand the foundation, then progress through each concept guide in order.

### For Intermediate Users

Jump to specific concepts you need:
- Building RAG applications? → [RAG Concepts](./rag.md)
- Creating agents? → [Agent Concepts](./agents.md)
- Managing conversations? → [Memory Concepts](./memory.md)

### For Advanced Users

Use this guide as a reference for:
- Architecture decisions
- Design pattern explanations
- Best practices
- Framework internals

## Concept Overview

### Core Concepts

The foundation of Beluga AI:
- **Runnable Interface**: Universal execution pattern
- **Message Types**: Communication between components
- **Context Propagation**: Request context and cancellation
- **Dependency Injection**: Flexible component composition

### LLM Concepts

Working with Large Language Models:
- **Provider Abstraction**: Unified interface across providers
- **Streaming**: Real-time response generation
- **Tool Calling**: Function calling capabilities
- **Batch Processing**: Efficient bulk operations

### Agent Concepts

Autonomous AI agents:
- **Agent Lifecycle**: Initialize, execute, finalize
- **Planning**: Breaking down complex tasks
- **Execution**: Tool orchestration
- **ReAct Pattern**: Reasoning and acting

### Memory Concepts

Conversation and context management:
- **Memory Types**: Buffer, window, summary, vector store
- **Conversation History**: Message storage and retrieval
- **Persistence**: Long-term memory storage
- **Context Management**: Window and token limits

### RAG Concepts

Retrieval-Augmented Generation:
- **Embeddings**: Vector representations of text
- **Vector Stores**: Similarity search
- **Retrievers**: Document retrieval strategies
- **Chunking**: Document processing

### Orchestration Concepts

Workflow and task management:
- **Chains**: Sequential execution
- **Graphs**: DAG-based workflows
- **Workflows**: Distributed orchestration
- **Task Scheduling**: Dependency management

## Related Documentation

- **[Getting Started Tutorial](../getting-started/)** - Step-by-step tutorials
- **[Architecture Documentation](./architecture/README.md)** - System architecture
- **[Design Patterns](./patterns/README.md)** - Implementation patterns
- **[Package Design Patterns](../package_design_patterns.md)** - Design principles
- **[API Reference](../api-reference.md)** - Detailed API docs

## Contributing

Found an issue or want to improve a concept guide? See the [Contributing Guide](https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING.md).

---

**Start Learning:** Begin with [Core Concepts](./core.md) or jump to a specific topic that interests you!

