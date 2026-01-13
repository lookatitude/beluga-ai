# Beluga AI Examples

This directory contains comprehensive, runnable examples demonstrating how to use the Beluga AI Framework.

## Quick Start

1. **Clone the repository**
2. **Set up environment variables** (if using real providers):
   ```bash
   export OPENAI_API_KEY=your-api-key
   ```
3. **Navigate to an example directory**
4. **Run the example**:
   ```bash
   go run main.go
   ```

## Example Categories

### Core Packages

Examples demonstrating core framework packages:

- **[config](config/basic/)** - Configuration management and loading
- **[core](core/basic/)** - Core utilities (errors, context, runnable)
- **[schema](schema/basic/)** - Message and document schemas
- **[embeddings](embeddings/basic/)** - Text embedding generation
- **[memory](memory/basic/)** - Conversation memory management
- **[chatmodels](chatmodels/basic/)** - Chat-based language models
- **[monitoring](monitoring/basic/)** - Observability and monitoring
- **[prompts](prompts/basic/)** - Prompt template management
- **[retrievers](retrievers/basic/)** - Document retrieval for RAG
- **[server](server/basic/)** - REST and MCP server creation

### Provider Examples

Examples demonstrating specific provider implementations:

#### LLM Providers
- **[OpenAI](llms/providers/openai/)** - Using OpenAI GPT models
- **[Anthropic](llms/providers/anthropic/)** - Using Anthropic Claude models
- **[Ollama](llms/providers/ollama/)** - Using local Ollama models

#### Embedding Providers
- **[OpenAI Embeddings](embeddings/providers/openai/)** - Using OpenAI embedding models

#### Vector Store Providers
- **[In-Memory](vectorstores/providers/inmemory/)** - Using in-memory vector storage

### Voice Packages

Examples demonstrating voice processing components:

- **[STT](voice/stt/)** - Speech-to-Text transcription
- **[TTS](voice/tts/)** - Text-to-Speech synthesis
- **[VAD](voice/vad/)** - Voice Activity Detection
- **[Turn Detection](voice/turndetection/)** - Detecting when speakers finish
- **[Transport](voice/transport/)** - Audio data transmission
- **[Noise Cancellation](voice/noise/)** - Removing background noise
- **[Backend](voice/backend/)** - Voice infrastructure (WebRTC, LiveKit)
- **[S2S](voice/s2s/basic_conversation/)** - Speech-to-Speech conversations
- **[Session](voice/simple/)** - Complete voice session management

### Agents

Examples demonstrating agent creation and usage:

- **[basic](agents/basic/)** - Basic agent creation and execution
- **[with_tools](agents/with_tools/)** - Agent with tool integration
- **[react](agents/react/)** - ReAct (Reasoning + Acting) agent implementation
- **[with_memory](agents/with_memory/)** - Agent with conversation memory
- **[vector_memory](agents/vector_memory/)** - Agent with vector store memory

### RAG Pipelines

Examples showing Retrieval-Augmented Generation:

- **[simple](rag/simple/)** - Complete RAG pipeline from scratch
- **[with_memory](rag/with_memory/)** - RAG with conversation memory
- **[advanced](rag/advanced/)** - Advanced RAG patterns and strategies

### Orchestration

Examples for workflow and chain orchestration:

- **[chain](orchestration/chain/)** - Simple chain creation and execution
- **[workflow](orchestration/workflow/)** - Workflow creation with conditional branching
- **[multi_agent](orchestration/multi_agent/)** - Multi-agent coordination

### Multi-Agent Systems

Examples demonstrating multi-agent collaboration:

- **[collaboration](multi-agent/collaboration/)** - Multiple agents working together
- **[specialized](multi-agent/specialized/)** - Specialized agent roles and delegation

### Integration

Complete integration examples:

- **[full_stack](integration/full_stack/)** - Complete application combining all components
- **[observability](integration/observability/)** - OTEL integration and monitoring

## Learning Path

### Beginner

1. Start with **[agents/basic](agents/basic/)** to understand basic agent usage
2. Try **[rag/simple](rag/simple/)** to learn RAG fundamentals
3. Explore **[agents/with_tools](agents/with_tools/)** for tool integration

### Intermediate

1. Learn **[agents/react](agents/react/)** for advanced agent patterns
2. Try **[rag/with_memory](rag/with_memory/)** for memory integration
3. Explore **[orchestration/chain](orchestration/chain/)** for workflow orchestration

### Advanced

1. Study **[multi-agent/collaboration](multi-agent/collaboration/)** for multi-agent systems
2. Try **[integration/full_stack](integration/full_stack/)** for complete applications
3. Explore **[integration/observability](integration/observability/)** for production patterns

## Example Selection Guide

### By Use Case

**Building a Chatbot:**
- Start with `agents/basic`
- Add `agents/with_memory` for conversation history
- Use `rag/simple` for knowledge base integration

**Creating a RAG System:**
- Start with `rag/simple`
- Add `rag/with_memory` for multi-turn conversations
- Explore `rag/advanced` for optimization

**Building Multi-Agent Systems:**
- Start with `orchestration/multi_agent`
- Study `multi-agent/collaboration` for coordination
- Try `multi-agent/specialized` for role specialization

**Production Deployment:**
- Review `integration/full_stack` for complete setup
- Study `integration/observability` for monitoring
- Check `orchestration/workflow` for complex workflows

## Prerequisites

### Required

- Go 1.21 or later
- Beluga AI Framework installed

### Optional (for real providers)

- OpenAI API key (for OpenAI providers)
- Anthropic API key (for Anthropic providers)
- Database setup (for PgVector examples)

## Running Examples

### With Mock Providers

Examples work out of the box with mock providers:

```bash
cd examples/agents/basic
go run main.go
```

### With Real Providers

Set environment variables and run:

```bash
export OPENAI_API_KEY=your-key
cd examples/agents/basic
go run main.go
```

## Understanding the Code

Each example includes:

- **Step-by-step comments** explaining each operation
- **Error handling** demonstrating best practices
- **Configuration** showing how to set up components
- **Output** showing expected results

## Extending Examples

Examples are designed to be extended:

1. **Modify configuration** to use different providers
2. **Add custom tools** for agent examples
3. **Extend workflows** for orchestration examples
4. **Add custom agents** for multi-agent examples

## Related Documentation

- [Getting Started Guide](../docs/getting-started/) - Step-by-step tutorials
- [Concepts Guide](../docs/concepts/) - Core concepts and patterns
- [API Reference](../docs/) - Complete API documentation
- [Best Practices](../docs/best-practices.md) - Production patterns

## Contributing

When adding new examples:

1. Follow the existing example structure
2. Include comprehensive comments
3. Add error handling
4. Update this README
5. Test with both mock and real providers

## Support

For questions or issues:

- Check the [Troubleshooting Guide](../docs/troubleshooting.md)
- Review the [Documentation](../docs/)
- Open an issue on GitHub
