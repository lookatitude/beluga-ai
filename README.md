# Beluga AI Framework

<div align="center">
  <img src="./assets/beluga-logo.svg" alt="Beluga AI Framework Logo" width="300"/>
  
  <p>
    <strong>A comprehensive Go framework for building production-ready AI and agentic applications</strong>
  </p>
  
  <p>
    <a href="https://github.com/lookatitude/beluga-ai/actions/workflows/ci-cd.yml"><img src="https://github.com/lookatitude/beluga-ai/workflows/CI/CD/badge.svg" alt="CI/CD"></a>
    <a href="https://goreportcard.com/report/github.com/lookatitude/beluga-ai"><img src="https://goreportcard.com/badge/github.com/lookatitude/beluga-ai" alt="Go Report Card"></a>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.24+-00ADD8.svg" alt="Go Version"></a>
    <a href="https://github.com/lookatitude/beluga-ai/releases"><img src="https://img.shields.io/github/v/release/lookatitude/beluga-ai" alt="Release"></a>
    <a href="https://github.com/lookatitude/beluga-ai/graphs/contributors"><img src="https://img.shields.io/github/contributors/lookatitude/beluga-ai" alt="Contributors"></a>
  </p>
  
  <p>
    <a href="./docs/quickstart.md">Quick Start</a> •
    <a href="./docs/README.md">Documentation</a> •
    <a href="./examples/README.md">Examples</a> •
    <a href="#features">Features</a> •
    <a href="#contributing">Contributing</a>
  </p>
</div>

---

## 🚀 Production Ready

**Beluga AI Framework** has completed comprehensive standardization and is now **enterprise-grade** with consistent patterns, extensive testing, and production-ready observability across all 16 packages. The framework is ready for large-scale deployment and development teams.

## 📖 What is Beluga AI?

**Beluga AI Framework** is a comprehensive Go-native framework designed for building sophisticated AI and agentic applications. Inspired by frameworks like [LangChain](https://www.langchain.com/) and [CrewAI](https://www.crewai.com/), Beluga AI provides a robust set of tools and abstractions to streamline the development of applications leveraging Large Language Models (LLMs).

### Why Beluga AI?

- **Go-Native**: Built specifically for Go developers, leveraging Go's strengths in concurrency and performance
- **Production-Ready**: Enterprise-grade patterns, comprehensive testing, and full observability
- **Extensible**: Interface-driven design makes it easy to add new providers and components
- **Modular**: Clean architecture with well-defined interfaces and separation of concerns
- **Well-Documented**: Comprehensive documentation, examples, and guides

## Features {#features}

### Core Features

- **🤖 LLM Integration**: Unified interface for multiple LLM providers (OpenAI, Anthropic, Bedrock, Ollama) with streaming support
- **🧠 Agent Framework**: Build autonomous agents with reasoning, planning, and execution capabilities
- **🛠️ Tool Management**: Extensible tool system supporting Shell, Go Functions, API callers, and custom tools
- **💾 Memory Management**: Multiple memory types (Buffer, Summary, VectorStore) with various backends
- **🔍 Retrieval-Augmented Generation (RAG)**: Complete RAG pipelines with document loaders, text splitters, embeddings, vector stores, and retrieval

### Advanced Capabilities

- **🎯 Orchestration**: Complex workflow management with chains, graphs, and schedulers
- **🎤 Voice Agents**: Complete voice interaction framework with STT, TTS, VAD, turn detection, and session management
- **📊 Observability**: Full OpenTelemetry integration with metrics, tracing, and structured logging
- **⚙️ Configuration**: Advanced configuration management with validation, environment variables, and YAML support
- **🔌 Extensibility**: Easy provider addition through global registries and factory patterns

### Production Features

- **✅ Enterprise-Grade Testing**: Comprehensive test coverage with mocks, integration tests, and benchmarks
- **📈 Monitoring**: Built-in metrics, distributed tracing, and health checks
- **🔒 Security**: Security scanning, vulnerability checks, and best practices
- **📚 Documentation**: Extensive documentation with guides, examples, and API references

## 🚀 Quick Start

### Installation

```bash
go get github.com/lookatitude/beluga-ai
```

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    ctx := context.Background()
    
    // Create an LLM instance
    llm, err := llms.NewLLM("openai", llms.WithAPIKey("your-api-key"))
    if err != nil {
        panic(err)
    }
    
    // Generate a response
    response, err := llm.Generate(ctx, "Hello, world!")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(response)
}
```

**New to Beluga AI?** Check out our [Quick Start Guide](./docs/quickstart.md) to get up and running in minutes!

## 📚 Documentation

### Getting Started

- **[Installation Guide](./docs/installation.md)** - Comprehensive installation instructions
- **[Quick Start Guide](./docs/quickstart.md)** - Get started in minutes
- **[Getting Started Tutorial](./docs/getting-started/)** - Multi-part tutorial series

### Core Documentation

- **[Architecture Documentation](./docs/architecture.md)** - System architecture and design patterns
- **[Package Design Patterns](./docs/package_design_patterns.md)** - Design principles and conventions
- **[Concepts Guide](./docs/concepts/)** - Core concepts and architectural patterns
- **[Best Practices](./docs/best-practices.md)** - Production best practices
- **[Troubleshooting Guide](./docs/troubleshooting.md)** - Common issues and solutions

### Reference

- **[API Documentation](./docs/)** - Complete API reference
- **[Provider Documentation](./docs/providers/)** - Provider-specific guides
- **[Migration Guide](./docs/migration.md)** - Version upgrades and migrations
- **[Framework Comparison](./docs/framework-comparison.md)** - Comparison with LangChain and CrewAI

### Additional Resources

- **[Use Cases](./docs/use-cases/)** - Real-world application examples
- **[Cookbook](./docs/cookbook/)** - Quick recipes and code snippets
- **[Documentation Index](./docs/README.md)** - Complete documentation overview

## 💡 Examples

Comprehensive, runnable examples demonstrating Beluga AI Framework capabilities:

### By Category

- **[Agents](./examples/agents/)** - Agent creation, tools, ReAct, and memory integration
- **[RAG](./examples/rag/)** - Simple and advanced RAG pipelines with document loaders and text splitters
- **[Document Loaders](./examples/documentloaders/)** - Loading documents from files and directories
- **[Text Splitters](./examples/textsplitters/)** - Splitting documents into chunks
- **[Orchestration](./examples/orchestration/)** - Chains, workflows, and multi-agent coordination
- **[Multi-Agent](./examples/multi-agent/)** - Agent collaboration and specialized roles
- **[Integration](./examples/integration/)** - Full-stack applications and observability
- **[Voice](./examples/voice/)** - Voice agents, streaming, and custom configurations

### Learning Path

- **Beginner**: Start with [basic agents](./examples/agents/basic/) and [simple RAG](./examples/rag/simple/)
- **Intermediate**: Explore [ReAct agents](./examples/agents/react/) and [orchestration](./examples/orchestration/chain/)
- **Advanced**: Study [multi-agent systems](./examples/multi-agent/collaboration/) and [full-stack integration](./examples/integration/full_stack/)

See the [Examples README](./examples/README.md) for a complete guide and learning path.

## 🏗️ Architecture

Beluga AI Framework follows a modular, interface-driven architecture:

- **Interface Segregation**: Small, focused interfaces for each component
- **Dependency Inversion**: Depend on abstractions, not implementations
- **Single Responsibility**: Clear separation of concerns
- **Composition over Inheritance**: Flexible component composition

### Package Structure

```
pkg/
├── schema/          # Core data structures
├── core/            # Foundational utilities
├── llms/            # LLM providers and interfaces
├── agents/          # Agent framework
├── memory/          # Memory management
├── vectorstores/    # Vector database providers
├── embeddings/      # Embedding providers
├── documentloaders/ # Document loading from files and directories
├── textsplitters/   # Text splitting for RAG pipelines
├── orchestration/   # Workflow orchestration
├── monitoring/      # Observability (OTEL)
├── config/          # Configuration management
└── voice/           # Voice agent framework
```

For detailed architecture information, see the [Architecture Documentation](./docs/architecture.md).

## 🔧 Installation & Setup

### Prerequisites

- Go 1.24 or later
- (Optional) API keys for LLM providers (OpenAI, Anthropic, etc.)

### Install

```bash
go get github.com/lookatitude/beluga-ai
```

### Development Setup

```bash
# Clone the repository
git clone https://github.com/lookatitude/beluga-ai.git
cd beluga-ai

# Install dependencies
go mod download

# Install development tools
make install-tools

# Run tests
make test
```

For detailed installation instructions, see the [Installation Guide](./docs/installation.md).

## ⚙️ Configuration

Beluga AI uses Viper for advanced configuration management. Configuration can be provided via:

1. **Environment Variables** (prefixed with `BELUGA_`)
2. **YAML/JSON Files**
3. **Programmatic Configuration**

### Example Configuration

```yaml
# config.yaml
llm_providers:
  - name: "openai-gpt4"
    provider: "openai"
    model_name: "gpt-4"
    api_key: "${OPENAI_API_KEY}"

vector_stores:
  - name: "pinecone-store"
    provider: "pinecone"
    api_key: "${PINECONE_API_KEY}"
    index_name: "beluga-index"
```

See the [Configuration Documentation](./docs/package_design_patterns.md#configuration-management) for details.

## 🧪 Testing Infrastructure

Beluga AI includes comprehensive testing infrastructure:

- **Unit Tests**: Table-driven tests with advanced mocks
- **Integration Tests**: Cross-package compatibility testing
- **Performance Benchmarks**: Built-in benchmarking utilities
- **Test Analyzer**: Tool for identifying and fixing test performance issues

```bash
# Run all tests
make test

# Run integration tests
go test ./tests/integration/... -v

# Run benchmarks
go test ./pkg/llms/... -bench=.
```

## 📊 Project Status

**✅ Enterprise-Grade Framework Complete**

Beluga AI Framework has achieved **100% standardization** across all 16 packages:

- ✅ All packages follow identical OTEL metrics, factory patterns, and testing standards
- ✅ Comprehensive testing infrastructure with mocks, integration tests, and benchmarks
- ✅ Production-ready observability with 100% OTEL integration
- ✅ Consistent factory patterns with global registries
- ✅ Enterprise-grade test coverage and documentation

### Implemented Packages

All 16 framework packages are production-ready:

- ✅ **LLMs** - Unified interfaces with multiple providers
- ✅ **Agents** - Complete agent framework with tools and memory
- ✅ **Memory** - Multiple memory types with various backends
- ✅ **VectorStores** - InMemory, PgVector, Pinecone providers
- ✅ **Embeddings** - OpenAI, Ollama providers
- ✅ **DocumentLoaders** - Load documents from files and directories
- ✅ **TextSplitters** - Split documents into chunks for RAG
- ✅ **Orchestration** - Workflow engine with chains and graphs
- ✅ **Voice** - Complete voice interaction framework
- ✅ **Monitoring** - Full OTEL integration
- ✅ **Config** - Advanced configuration management
- ✅ **And more...**

See the [Architecture Documentation](./docs/architecture.md) for complete details.

## Contributing {#contributing}

We welcome contributions! Please see our [Contributing Guide](./CONTRIBUTING.md) for:

- Development setup and workflow
- Code quality standards
- How to add new providers
- Pull request process
- Release process

### Quick Contribution Guide

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Make your changes following our [design patterns](./docs/package_design_patterns.md)
4. Run tests (`make test`)
5. Commit using [Conventional Commits](./CONTRIBUTING.md#conventional-commits)
6. Push and create a Pull Request

## 📄 License

Beluga AI Framework is licensed under the [MIT License](./LICENSE).

## 🆘 Support & Community

- **Documentation**: [docs/README.md](./docs/README.md)
- **Issues**: [GitHub Issues](https://github.com/lookatitude/beluga-ai/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lookatitude/beluga-ai/discussions)
- **Contributors**: [View Contributors](https://github.com/lookatitude/beluga-ai/graphs/contributors)
- **Releases**: [View Releases](https://github.com/lookatitude/beluga-ai/releases)

## 🌟 Star Us

If you find Beluga AI Framework useful, please consider giving us a star on GitHub!

---

---

<div align="center">
  <p>
    <strong>Built with ❤️ by the Beluga AI Team</strong>
  </p>
  <p>
    <a href="./LICENSE">License</a> •
    <a href="./CONTRIBUTING.md">Contributing</a> •
    <a href="./docs/README.md">Documentation</a> •
    <a href="https://github.com/lookatitude/beluga-ai/issues">Issues</a>
  </p>
</div>
