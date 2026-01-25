# Beluga AI Framework - Documentation

Welcome to the Beluga AI Framework documentation! This directory contains comprehensive guides, references, and examples to help you build production-ready AI applications.

## 📚 Documentation Index

### Getting Started

- **[Installation Guide](./installation.md)** - Comprehensive installation instructions
  - System requirements
  - Platform-specific installation (Linux, macOS, Windows)
  - Docker installation
  - Development environment setup
  - Troubleshooting

- **[Quick Start Guide](./quickstart.md)** - Get up and running in minutes with step-by-step instructions
  - Installation
  - First LLM call
  - Configuration setup
  - Creating your first agent
  - Troubleshooting

- **[Getting Started Tutorial](./getting-started/)** - Multi-part tutorial series
  1. [Your First LLM Call](./getting-started/01-first-llm-call.md)
  2. [Building a Simple RAG Application](./getting-started/02-simple-rag.md)
  3. [Creating Your First Agent](./getting-started/03-first-agent.md)
  4. [Working with Tools](./getting-started/04-working-with-tools.md)
  5. [Memory Management](./getting-started/05-memory-management.md)
  6. [Orchestration Basics](./getting-started/06-orchestration-basics.md)
  7. [Production Deployment](./getting-started/07-production-deployment.md)

### Core Documentation

- **[Architecture Documentation](./architecture.md)** - Comprehensive overview of the framework's architecture
  - Module structure and responsibilities
  - Design patterns and principles
  - How to extend the framework
  - Future considerations
  - Visual architecture diagrams
  - System architecture overview
  - Data flow diagrams
  - Component interaction diagrams
  - Sequence diagrams

- **[Architecture Visualizations](./concepts/architecture/)** - Detailed architecture diagrams
  - [Component Diagrams](./concepts/architecture/component-diagrams.md) - Package structure and interface hierarchy
  - [Data Flows](./concepts/architecture/data-flows.md) - Data flow through the system
  - [Sequence Diagrams](./concepts/architecture/sequences.md) - Component interaction sequences

- **[Package Design Patterns](./package_design_patterns.md)** - Design patterns and conventions for all packages
  - Core principles (ISP, DIP, SRP)
  - Package structure standards
  - Interface design guidelines
  - Configuration management
  - Observability patterns
  - Testing patterns

- **[Pattern Implementation Guides](./concepts/patterns/pattern-examples.md)** - Practical pattern examples
  - [Pattern Examples](./concepts/patterns/pattern-examples.md) - Real-world pattern implementations
  - [Cross-Package Patterns](./concepts/patterns/cross-package-patterns.md) - How patterns work together
  - [Pattern Decision Guide](./concepts/patterns/pattern-decision-guide.md) - When to use which pattern

- **[Concepts Guide](./concepts/)** - Core concepts and architectural patterns
  - [Core Concepts](./concepts/core.md) - Runnable interface, messages, context
  - [LLM Concepts](./concepts/llms.md) - Provider abstraction, streaming, tool calling
  - [Agent Concepts](./concepts/agents.md) - Agent lifecycle, planning, execution
  - [Memory Concepts](./concepts/memory.md) - Memory types, conversation history
  - [RAG Concepts](./concepts/rag.md) - Retrieval-augmented generation
  - [Orchestration Concepts](./concepts/orchestration.md) - Chains, graphs, workflows

### Framework Comparison

- **[Framework Comparison](./framework-comparison.md)** - Detailed comparison with LangChain and CrewAI
  - Feature parity analysis
  - Flexibility and ease of use
  - Pros and cons
  - Use case recommendations
  - Competitive positioning

### Advanced Guides

- **[Guides](./guides/llm-providers.md)** - In-depth guides for advanced features
  - [Streaming LLM with Tool Calls](./guides/llm-streaming-tool-calls.md) - Real-time streaming with function calling
  - [Agent Types (PlanExecute/ReAct)](./guides/agent-types.md) - Choosing and implementing agent patterns
  - [Multimodal RAG](./guides/rag-multimodal.md) - RAG with images and video
  - [LLM Provider Integration](./guides/llm-providers.md) - Adding custom LLM providers
  - [Voice Provider Integration](./guides/voice-providers.md) - STT, TTS, and S2S integration
  - [Extensibility Guide](./guides/extensibility.md) - Extending the framework
  - [Observability & Tracing](./guides/observability-tracing.md) - Distributed tracing setup

### Use Cases

The [use-cases](./use-cases/) directory contains detailed examples of real-world applications:

**Core Use Cases:**
1. **[Enterprise RAG Knowledge Base](./use-cases/01-enterprise-rag-knowledge-base.md)** - Building a knowledge base with RAG
2. **[Multi-Agent Customer Support](./use-cases/02-multi-agent-customer-support.md)** - Customer support system with multiple agents
3. **[Intelligent Document Processing](./use-cases/03-intelligent-document-processing.md)** - Automated document analysis
4. **[Real-Time Data Analysis Agent](./use-cases/04-real-time-data-analysis-agent.md)** - Live data analysis with agents
5. **[Conversational AI Assistant](./use-cases/05-conversational-ai-assistant.md)** - Building a conversational assistant
6. **[Automated Code Review System](./use-cases/06-automated-code-review-system.md)** - AI-powered code review
7. **[Distributed Workflow Orchestration](./use-cases/07-distributed-workflow-orchestration.md)** - Complex workflow management
8. **[Semantic Search Recommendation](./use-cases/08-semantic-search-recommendation.md)** - Recommendation systems with semantic search
9. **[Multi-Model LLM Gateway](./use-cases/09-multi-model-llm-gateway.md)** - Unified LLM gateway
10. **[Production Agent Platform](./use-cases/10-production-agent-platform.md)** - Enterprise agent platform

**Advanced Scenarios:**
11. **[Batch Processing](./use-cases/11-batch-processing.md)** - Processing multiple queries with concurrency control
12. **[Monitoring Dashboards](./use-cases/monitoring-dashboards.md)** - Prometheus and Grafana setup
13. **[Voice Sessions](./use-cases/voice-sessions.md)** - Real-time voice agent implementation
14. **[RAG Strategies](./use-cases/rag-strategies.md)** - Choosing the right retrieval approach

### Provider Documentation

- **[Provider Documentation](./providers/)** - Detailed provider guides
  - [LLM Providers](https://github.com/lookatitude/beluga-ai/tree/main/pkg/llms) - OpenAI, Anthropic, Bedrock, Ollama
  - [Vector Store Providers](https://github.com/lookatitude/beluga-ai/tree/main/pkg/vectorstores) - InMemory, PgVector, Pinecone
  - [Embedding Providers](https://github.com/lookatitude/beluga-ai/tree/main/pkg/embeddings) - OpenAI, Ollama
  - Provider comparisons and selection guides

### Best Practices & Guides

- **[Best Practices Guide](./best-practices.md)** - Production best practices
  - Configuration management
  - Error handling patterns
  - Performance optimization
  - Security considerations
  - Testing strategies
  - Deployment patterns

- **[Troubleshooting Guide](./troubleshooting.md)** - Common issues and solutions
  - Common errors and fixes
  - Performance issues
  - Configuration problems
  - Provider-specific issues
  - Debugging tips
  - FAQ

- **[Migration Guide](./migration.md)** - Version upgrades and framework migrations
  - Version upgrade guides
  - Migration from LangChain/CrewAI
  - Deprecation notices

- **[Cookbook](./cookbook/)** - Quick recipes and code snippets
  - [RAG Recipes](./cookbook/rag-recipes.md)
  - [Agent Recipes](./cookbook/agent-recipes.md)
  - [Tool Recipes](./cookbook/tool-recipes.md)
  - [Memory Recipes](./cookbook/memory-recipes.md)
  - [Integration Recipes](./cookbook/integration-recipes.md)
  - [Quick Solutions](./cookbook/quick-solutions.md)
  - [LLM Error Handling](./cookbook/llm-error-handling.md) - Retry logic and error recovery
  - [Custom Agent Extensions](./cookbook/custom-agent.md) - Extending agents with custom behavior
  - [Voice Backends](./cookbook/voice-backends.md) - Switching between voice providers

### Documentation Roadmap

- **[Documentation Roadmap](./documentation-roadmap.md)** - Planned documentation improvements
  - Missing documentation identified
  - Planned additions for parity with LangChain/CrewAI
  - Priority items

## 🗂️ Documentation Structure

```
docs/
├── README.md                    # This file - documentation index
├── installation.md             # Installation guide
├── quickstart.md               # Quick start guide
├── architecture.md             # Architecture documentation
├── package_design_patterns.md  # Design patterns guide
├── framework-comparison.md     # Comparison with other frameworks
├── best-practices.md           # Best practices guide
├── troubleshooting.md          # Troubleshooting guide
├── migration.md                # Migration guide
├── documentation-roadmap.md    # Documentation roadmap
├── getting-started/            # Multi-part tutorial series
│   ├── README.md
│   └── 01-*.md through 07-*.md
├── concepts/                   # Core concepts guide
│   ├── README.md
│   └── core.md, llms.md, agents.md, memory.md, rag.md, orchestration.md
├── providers/                  # Provider documentation
│   ├── README.md
│   ├── llms/
│   ├── vectorstores/
│   └── embeddings/
├── cookbook/                   # Recipe collection
│   ├── README.md
│   └── *.md recipes
└── use-cases/                  # Real-world use case examples
    ├── README.md
    └── 01-*.md through 10-*.md
```

## 🚀 Quick Navigation

### For New Users
1. Start with the **[Installation Guide](./installation.md)**
2. Follow the **[Getting Started Tutorial](./getting-started/)** series
3. Review the **[Architecture Documentation](./architecture.md)** to understand the framework
4. Explore **[Use Cases](./use-cases/)** for inspiration

### For Developers
1. Read **[Package Design Patterns](./package_design_patterns.md)** for coding standards
2. Study **[Concepts Guide](./concepts/)** for core concepts
3. Review **[Best Practices](./best-practices.md)** for production patterns
4. Check **[Framework Comparison](./framework-comparison.md)** for competitive context

### For Architects
1. Review **[Architecture Documentation](./architecture.md)** for system design
2. Study **[Package Design Patterns](./package_design_patterns.md)** for design principles
3. Explore **[Use Cases](./use-cases/)** for production patterns

## 📖 Additional Resources

### Main Project Documentation
- **[Main README](https://github.com/lookatitude/beluga-ai/blob/main/README.md)** - Project overview and installation
- **[Contributing Guide](https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING.md)** - How to contribute
- **[CHANGELOG](https://github.com/lookatitude/beluga-ai/blob/main/CHANGELOG.md)** - Release notes and changes

### API Documentation
- **[API Documentation](./api-reference.md)** - Detailed API reference
  - Package-specific documentation
  - Provider implementations
  - Configuration options

### Examples
- **[Examples Directory](https://github.com/lookatitude/beluga-ai/tree/main/examples/)** - Comprehensive runnable examples
  - [Agent Examples](https://github.com/lookatitude/beluga-ai/tree/main/examples/agents/) - Basic, tools, ReAct, memory integration
  - [RAG Examples](https://github.com/lookatitude/beluga-ai/tree/main/examples/rag/) - Simple, with memory, advanced patterns
  - [Orchestration Examples](https://github.com/lookatitude/beluga-ai/tree/main/examples/orchestration/) - Chains, workflows, multi-agent
  - [Multi-Agent Examples](https://github.com/lookatitude/beluga-ai/tree/main/examples/multi-agent/) - Collaboration, specialized roles
  - [Integration Examples](https://github.com/lookatitude/beluga-ai/tree/main/examples/integration/) - Full-stack, observability
  - See [Examples README](https://github.com/lookatitude/beluga-ai/tree/main/examples/README.md) for complete guide

## 🔍 Finding What You Need

### By Topic

**Installation & Setup**
- [Installation Guide](./installation.md) - Comprehensive installation
- [Quick Start Guide](./quickstart.md) - Quick setup and first steps
- [Getting Started Tutorial](./getting-started/) - Step-by-step tutorials

**Architecture & Design**
- [Architecture Documentation](./architecture.md) - System architecture
- [Package Design Patterns](./package_design_patterns.md) - Design principles
- [Concepts Guide](./concepts/) - Core concepts and patterns

**Usage & Examples**
- [Getting Started Tutorial](./getting-started/) - Step-by-step tutorials
- [Quick Start Guide](./quickstart.md) - Basic usage
- [Use Cases](./use-cases/) - Real-world examples
- [Cookbook](./cookbook/) - Quick recipes and snippets
- [Examples Directory](https://github.com/lookatitude/beluga-ai/tree/main/examples/) - Code examples
- [Provider Documentation](./providers/) - Provider-specific guides

**Comparison & Context**
- [Framework Comparison](./framework-comparison.md) - vs LangChain/CrewAI

**Development**
- [Package Design Patterns](./package_design_patterns.md) - Development guidelines
- [Best Practices](./best-practices.md) - Production best practices
- [Troubleshooting Guide](./troubleshooting.md) - Common issues and solutions
- [Migration Guide](./migration.md) - Version upgrades and migrations
- [Contributing Guide](https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING.md) - Contribution process

## 📝 Documentation Status

### ✅ Complete
- Installation Guide
- Quick Start Guide
- Getting Started Tutorial (7 parts)
- Architecture Documentation
- Package Design Patterns
- Concepts Guide (6 concepts)
- Framework Comparison
- Provider Documentation (LLMs, VectorStores, Embeddings)
- Best Practices Guide
- Troubleshooting Guide
- Migration Guide
- Cookbook (6 recipe collections)
- Use Cases (10 examples)

See [Documentation Roadmap](./documentation-roadmap.md) for details on documentation status.

## 🤝 Contributing to Documentation

We welcome contributions to improve our documentation! Please see the [Contributing Guide](https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING.md) for guidelines.

### Documentation Standards
- Use clear, concise language
- Include code examples where helpful
- Link to related documentation
- Keep examples up-to-date with the codebase
- Follow the existing documentation style

## 📞 Getting Help

- **Documentation Issues**: Open an issue on GitHub
- **Questions**: Check existing documentation first, then open a discussion
- **Feature Requests**: See the [Documentation Roadmap](./documentation-roadmap.md)

---

**Last Updated**: Documentation is actively maintained. Check individual files for last update dates.

