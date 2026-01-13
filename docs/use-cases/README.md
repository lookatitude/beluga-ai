# Beluga AI Framework - End-to-End Use Cases

This directory contains comprehensive, production-ready use case documentation demonstrating Beluga AI Framework's capabilities across different domains and application types.

## Overview

Each use case is a complete, end-to-end example that showcases:
- **Multiple Framework Components**: Demonstrates integration between different Beluga AI packages
- **Production-Ready Patterns**: Enterprise-grade architecture and best practices
- **Observability**: Full OpenTelemetry integration for monitoring and debugging
- **Configuration Management**: Comprehensive YAML and environment variable examples
- **Error Handling**: Robust error management strategies
- **Testing Strategies**: Unit, integration, and performance testing approaches

## Use Cases

### 1. [Enterprise RAG Knowledge Base System](./01-enterprise-rag-knowledge-base.md)
**Domain:** Enterprise Knowledge Management  
**Components:** VectorStores, Embeddings, Retrievers, LLMs, Memory, Prompts, Orchestration, Monitoring, Config, Server  
**Focus:** Complete RAG pipeline for enterprise knowledge management with semantic search, document retrieval, and intelligent Q&A capabilities.

### 2. [Multi-Agent Customer Support System](./02-multi-agent-customer-support.md)
**Domain:** Customer Service Automation  
**Components:** Agents (ReAct), Tools (API, Shell, Calculator), Memory, LLMs, Orchestration, Monitoring, Server  
**Focus:** Intelligent customer support system with multiple specialized agents working together to resolve customer issues.

### 3. [Intelligent Document Processing Pipeline](./03-intelligent-document-processing.md)
**Domain:** Document Management and Analysis  
**Components:** VectorStores, Embeddings, Retrievers, LLMs, Prompts, Orchestration, Memory, Monitoring, Config  
**Focus:** Automated document ingestion, processing, embedding, and retrieval system with comprehensive observability.

### 4. [Real-Time Data Analysis Agent](./04-real-time-data-analysis-agent.md)
**Domain:** Business Intelligence and Analytics  
**Components:** Agents, Tools (API, GoFunc, Calculator), LLMs, Memory, Orchestration, Monitoring, Server  
**Focus:** Autonomous agent that fetches, analyzes, and reports on real-time data from multiple sources.

### 5. [Conversational AI Assistant with Long-Term Memory](./05-conversational-ai-assistant.md)
**Domain:** Personal Assistants and Chatbots  
**Components:** ChatModels, Memory (Buffer, Summary, VectorStore), LLMs, Prompts, VectorStores, Embeddings, Monitoring, Server  
**Focus:** Advanced conversational AI with persistent memory, context management, and semantic retrieval capabilities.

### 6. [Automated Code Review and Analysis System](./06-automated-code-review-system.md)
**Domain:** Software Development and DevOps  
**Components:** Agents, Tools (Shell, API, GoFunc), LLMs, Memory, Orchestration, Monitoring, Server  
**Focus:** AI-powered code analysis system that reviews code, identifies issues, and provides recommendations.

### 7. [Distributed Workflow Orchestration System](./07-distributed-workflow-orchestration.md)
**Domain:** Enterprise Workflow Automation  
**Components:** Orchestration (Workflows, Graphs, Chains), LLMs, Memory, Monitoring, Config, Server  
**Focus:** Complex distributed workflows with Temporal integration, dependency management, and comprehensive observability.

### 8. [Semantic Search and Recommendation Engine](./08-semantic-search-recommendation.md)
**Domain:** Search and Recommendation Systems  
**Components:** VectorStores, Embeddings, Retrievers, LLMs, Prompts, Monitoring, Config  
**Focus:** High-performance semantic search system with recommendation capabilities using vector similarity.

### 9. [Multi-Model LLM Gateway with Observability](./09-multi-model-llm-gateway.md)
**Domain:** LLM Infrastructure and API Gateway  
**Components:** LLMs (multiple providers), ChatModels, Monitoring (OTEL), Config, Server, Orchestration  
**Focus:** Unified LLM gateway supporting multiple providers with load balancing, observability, and failover.

### 10. [Production-Grade AI Agent Platform](./10-production-agent-platform.md)
**Domain:** General-Purpose AI Agent Platform  
**Components:** ALL Framework Components  
**Focus:** Complete, production-ready AI agent platform showcasing all Beluga AI capabilities in a unified system.

## Common Patterns Across Use Cases

All use cases demonstrate:

1. **Configuration Management**
   - YAML configuration files
   - Environment variable support
   - Provider-specific settings
   - Validation and defaults

2. **Observability**
   - OpenTelemetry metrics
   - Distributed tracing
   - Structured logging
   - Health checks

3. **Error Handling**
   - Custom error types
   - Error wrapping and context
   - Retry strategies
   - Circuit breakers

4. **Testing**
   - Unit test examples
   - Integration test scenarios
   - Mock implementations
   - Performance benchmarks

5. **Deployment**
   - Production considerations
   - Scaling strategies
   - Performance optimization
   - Security best practices

## Getting Started

1. **Choose a Use Case**: Review the use cases above and select one that matches your needs
2. **Read the Documentation**: Each use case includes comprehensive implementation guides
3. **Review Configuration**: Study the YAML configuration examples
4. **Examine Code Examples**: Go code snippets demonstrate key patterns
5. **Set Up Observability**: Follow the OpenTelemetry setup instructions
6. **Deploy and Monitor**: Use the deployment guides and monitoring dashboards

## Framework Components Reference

Each use case leverages different combinations of Beluga AI components:

- **Core**: `pkg/core` - Runnable interface, dependency injection, utilities
- **Schema**: `pkg/schema` - Message, Document, ToolCall types
- **Config**: `pkg/config` - Configuration management with Viper
- **LLMs**: `pkg/llms` - Unified LLM interface with multiple providers
- **ChatModels**: `pkg/chatmodels` - ChatModel interface and implementations
- **Embeddings**: `pkg/embeddings` - Embedder interface with providers
- **VectorStores**: `pkg/vectorstores` - Vector storage and similarity search
- **Retrievers**: `pkg/retrievers` - Document retrieval strategies
- **Memory**: `pkg/memory` - Conversation memory management
- **Prompts**: `pkg/prompts` - Prompt templates and rendering
- **Agents**: `pkg/agents` - Agent framework with ReAct support
- **Tools**: `pkg/agents/tools` - Tool registry and implementations
- **Orchestration**: `pkg/orchestration` - Chains, graphs, and workflows
- **Monitoring**: `pkg/monitoring` - OpenTelemetry observability
- **Server**: `pkg/server` - REST and MCP server implementations

## Contributing

When creating new use cases:

1. Follow the standard structure outlined in each existing use case
2. Include comprehensive architecture diagrams
3. Provide complete configuration examples
4. Document observability setup
5. Include troubleshooting guides
6. Add code examples and test scenarios

## Additional Resources

- [Framework Comparison](../framework-comparison.md) - Comparison with LangChain and CrewAI
- [Package Design Patterns](../package_design_patterns.md) - Framework design principles
- [Main README](../../README.md) - Framework overview and installation

