# Beluga AI Framework - Documentation

Welcome to the Beluga AI Framework documentation! This directory contains comprehensive guides, references, and examples to help you build production-ready AI applications.

## 📚 Documentation Index

### Getting Started

- **[Installation Guide](./INSTALLATION.md)** - Comprehensive installation instructions
  - System requirements
  - Platform-specific installation (Linux, macOS, Windows)
  - Docker installation
  - Development environment setup
  - Troubleshooting

- **[Quick Start Guide](./QUICKSTART.md)** - Get up and running in minutes with step-by-step instructions
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

- **[Architecture Visualizations](./architecture/)** - Detailed architecture diagrams
  - [Component Diagrams](./architecture/component-diagrams.md) - Package structure and interface hierarchy
  - [Data Flows](./architecture/data-flows.md) - Data flow through the system
  - [Sequence Diagrams](./architecture/sequences.md) - Component interaction sequences

- **[Package Design Patterns](./package_design_patterns.md)** - Design patterns and conventions for all packages
  - Core principles (ISP, DIP, SRP)
  - Package structure standards
  - Interface design guidelines
  - Configuration management
  - Observability patterns
  - Testing patterns

- **[Pattern Implementation Guides](./patterns/)** - Practical pattern examples
  - [Pattern Examples](./patterns/pattern-examples.md) - Real-world pattern implementations
  - [Cross-Package Patterns](./patterns/cross-package-patterns.md) - How patterns work together
  - [Pattern Decision Guide](./patterns/pattern-decision-guide.md) - When to use which pattern

- **[Concepts Guide](./concepts/)** - Core concepts and architectural patterns
  - [Core Concepts](./concepts/core.md) - Runnable interface, messages, context
  - [LLM Concepts](./concepts/llms.md) - Provider abstraction, streaming, tool calling
  - [Agent Concepts](./concepts/agents.md) - Agent lifecycle, planning, execution
  - [Memory Concepts](./concepts/memory.md) - Memory types, conversation history
  - [RAG Concepts](./concepts/rag.md) - Retrieval-augmented generation
  - [Orchestration Concepts](./concepts/orchestration.md) - Chains, graphs, workflows

### Framework Comparison

- **[Framework Comparison](./FRAMEWORK_COMPARISON.md)** - Detailed comparison with LangChain and CrewAI
  - Feature parity analysis
  - Flexibility and ease of use
  - Pros and cons
  - Use case recommendations
  - Competitive positioning

### Use Cases

The [use-cases](./use-cases/) directory contains detailed examples of real-world applications:

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

### Provider Documentation

- **[Provider Documentation](./providers/)** - Detailed provider guides
  - [LLM Providers](./providers/llms/) - OpenAI, Anthropic, Bedrock, Ollama
  - [Vector Store Providers](./providers/vectorstores/) - InMemory, PgVector, Pinecone
  - [Embedding Providers](./providers/embeddings/) - OpenAI, Ollama
  - Provider comparisons and selection guides

### Best Practices & Guides

- **[Best Practices Guide](./BEST_PRACTICES.md)** - Production best practices
  - Configuration management
  - Error handling patterns
  - Performance optimization
  - Security considerations
  - Testing strategies
  - Deployment patterns

- **[Troubleshooting Guide](./TROUBLESHOOTING.md)** - Common issues and solutions
  - Common errors and fixes
  - Performance issues
  - Configuration problems
  - Provider-specific issues
  - Debugging tips
  - FAQ

- **[Migration Guide](./MIGRATION.md)** - Version upgrades and framework migrations
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

### Documentation Roadmap

- **[Documentation Roadmap](./DOCUMENTATION_ROADMAP.md)** - Planned documentation improvements
  - Missing documentation identified
  - Planned additions for parity with LangChain/CrewAI
  - Priority items

## 🗂️ Documentation Structure

```
docs/
├── README.md                    # This file - documentation index
├── INSTALLATION.md             # Installation guide
├── QUICKSTART.md               # Quick start guide
├── architecture.md             # Architecture documentation
├── package_design_patterns.md  # Design patterns guide
├── FRAMEWORK_COMPARISON.md     # Comparison with other frameworks
├── BEST_PRACTICES.md           # Best practices guide
├── TROUBLESHOOTING.md          # Troubleshooting guide
├── MIGRATION.md                # Migration guide
├── DOCUMENTATION_ROADMAP.md    # Documentation roadmap
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
1. Start with the **[Installation Guide](./INSTALLATION.md)**
2. Follow the **[Getting Started Tutorial](./getting-started/)** series
3. Review the **[Architecture Documentation](./architecture.md)** to understand the framework
4. Explore **[Use Cases](./use-cases/)** for inspiration

### For Developers
1. Read **[Package Design Patterns](./package_design_patterns.md)** for coding standards
2. Study **[Concepts Guide](./concepts/)** for core concepts
3. Review **[Best Practices](./BEST_PRACTICES.md)** for production patterns
4. Check **[Framework Comparison](./FRAMEWORK_COMPARISON.md)** for competitive context

### For Architects
1. Review **[Architecture Documentation](./architecture.md)** for system design
2. Study **[Package Design Patterns](./package_design_patterns.md)** for design principles
3. Explore **[Use Cases](./use-cases/)** for production patterns

## 📖 Additional Resources

### Main Project Documentation
- **[Main README](../README.md)** - Project overview and installation
- **[Contributing Guide](../CONTRIBUTING.md)** - How to contribute
- **[CHANGELOG](../CHANGELOG.md)** - Release notes and changes

### API Documentation
- **[API Documentation](../website/docs/api/)** - Detailed API reference
  - Package-specific documentation
  - Provider implementations
  - Configuration options

### Examples
- **[Examples Directory](../examples/)** - Comprehensive runnable examples
  - [Agent Examples](../examples/agents/) - Basic, tools, ReAct, memory integration
  - [RAG Examples](../examples/rag/) - Simple, with memory, advanced patterns
  - [Orchestration Examples](../examples/orchestration/) - Chains, workflows, multi-agent
  - [Multi-Agent Examples](../examples/multi-agent/) - Collaboration, specialized roles
  - [Integration Examples](../examples/integration/) - Full-stack, observability
  - See [Examples README](../examples/README.md) for complete guide

## 🔍 Finding What You Need

### By Topic

**Installation & Setup**
- [Installation Guide](./INSTALLATION.md) - Comprehensive installation
- [Quick Start Guide](./QUICKSTART.md) - Quick setup and first steps
- [Getting Started Tutorial](./getting-started/) - Step-by-step tutorials

**Architecture & Design**
- [Architecture Documentation](./architecture.md) - System architecture
- [Package Design Patterns](./package_design_patterns.md) - Design principles
- [Concepts Guide](./concepts/) - Core concepts and patterns

**Usage & Examples**
- [Getting Started Tutorial](./getting-started/) - Step-by-step tutorials
- [Quick Start Guide](./QUICKSTART.md) - Basic usage
- [Use Cases](./use-cases/) - Real-world examples
- [Cookbook](./cookbook/) - Quick recipes and snippets
- [Examples Directory](../examples/) - Code examples
- [Provider Documentation](./providers/) - Provider-specific guides

**Comparison & Context**
- [Framework Comparison](./FRAMEWORK_COMPARISON.md) - vs LangChain/CrewAI

**Development**
- [Package Design Patterns](./package_design_patterns.md) - Development guidelines
- [Best Practices](./BEST_PRACTICES.md) - Production best practices
- [Troubleshooting Guide](./TROUBLESHOOTING.md) - Common issues and solutions
- [Migration Guide](./MIGRATION.md) - Version upgrades and migrations
- [Contributing Guide](../CONTRIBUTING.md) - Contribution process

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

See [Documentation Roadmap](./DOCUMENTATION_ROADMAP.md) for details on documentation status.

## 🤝 Contributing to Documentation

We welcome contributions to improve our documentation! Please see the [Contributing Guide](../CONTRIBUTING.md) for guidelines.

### Documentation Standards
- Use clear, concise language
- Include code examples where helpful
- Link to related documentation
- Keep examples up-to-date with the codebase
- Follow the existing documentation style

## 📞 Getting Help

- **Documentation Issues**: Open an issue on GitHub
- **Questions**: Check existing documentation first, then open a discussion
- **Feature Requests**: See the [Documentation Roadmap](./DOCUMENTATION_ROADMAP.md)

---

**Last Updated**: Documentation is actively maintained. Check individual files for last update dates.

