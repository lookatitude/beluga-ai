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

---

## Core Use Cases

These foundational use cases provide complete, end-to-end implementations covering major AI application patterns.

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

### 11. [Batch Processing Pipeline](./11-batch-processing.md)
**Domain:** Data Processing at Scale
**Components:** Orchestration, LLMs, DocumentLoaders, Monitoring, Config
**Focus:** High-throughput batch processing for large-scale AI workloads with job scheduling and progress tracking.

---

## Package Deep-Dives

These use cases focus on specific packages, providing in-depth examples of their capabilities and integration patterns.

### Agents

| Use Case | Description |
|----------|-------------|
| [DevSecOps Auditor](./agents-devsecops-auditor.md) | Automated security auditing agent for CI/CD pipelines |
| [Autonomous Support Agent](./agents-autonomous-support.md) | Self-directed customer support with escalation handling |

### ChatModels

| Use Case | Description |
|----------|-------------|
| [Model A/B Testing](./chatmodels-model-ab-testing.md) | Compare chat model performance with controlled experiments |
| [Cost-Optimized Router](./chatmodels-cost-optimized-router.md) | Intelligent routing to minimize costs while maintaining quality |

### Config

| Use Case | Description |
|----------|-------------|
| [Multi-Tenant API Keys](./config-multi-tenant-api-keys.md) | Secure API key management for multi-tenant deployments |
| [Dynamic Feature Flagging](./config-dynamic-feature-flagging.md) | Runtime configuration changes with feature flags |

### Core

| Use Case | Description |
|----------|-------------|
| [Error Recovery Service](./core-error-recovery-service.md) | Resilient error handling with automatic recovery |
| [High-Availability Streaming Proxy](./core-high-availability-streaming-proxy.md) | Fault-tolerant streaming with load balancing |

### DocumentLoaders

| Use Case | Description |
|----------|-------------|
| [Cloud Sync Pipeline](./docloaders-cloud-sync.md) | Real-time document synchronization from cloud storage |
| [Legacy Archive Migration](./docloaders-legacy-archive.md) | Migrate and process legacy document archives |

### Embeddings

| Use Case | Description |
|----------|-------------|
| [Semantic Image Search](./embeddings-semantic-image-search.md) | Multi-modal embeddings for image-text search |
| [Cross-Lingual Retrieval](./embeddings-cross-lingual-retrieval.md) | Multi-language document retrieval with unified embeddings |

### LLMs

| Use Case | Description |
|----------|-------------|
| [Model Benchmarking Dashboard](./llms-model-benchmarking-dashboard.md) | Compare LLM performance across providers |
| [Automated Code Generation](./llms-automated-code-generation.md) | Generate production code with validation |

### Memory

| Use Case | Description |
|----------|-------------|
| [IDE Extension Memory](./memory-ide-extension.md) | Context persistence for AI-powered IDE extensions |
| [Patient History System](./memory-patient-history.md) | HIPAA-compliant medical conversation memory |

### Messaging

| Use Case | Description |
|----------|-------------|
| [Multi-Channel Hub](./messaging-multi-channel-hub.md) | Unified messaging across SMS, WhatsApp, and more |
| [SMS Reminders System](./messaging-sms-reminders.md) | Automated appointment reminders via SMS |

### Monitoring

| Use Case | Description |
|----------|-------------|
| [PII Leakage Detection](./monitoring-pii-leakage-detection.md) | Real-time detection of sensitive data exposure |
| [Observability Dashboards](./monitoring-dashboards.md) | Pre-built dashboards for AI system monitoring |
| [Token Cost Attribution](./monitoring-token-cost-attribution.md) | Track and attribute LLM costs by user/feature |

### Multimodal

| Use Case | Description |
|----------|-------------|
| [Security Camera Analysis](./multimodal-security-camera.md) | Real-time video analysis for security applications |
| [Audio-Visual Search](./multimodal-audio-visual-search.md) | Combined audio and visual content search |

### Orchestration

| Use Case | Description |
|----------|-------------|
| [Invoice Processor](./orchestration-invoice-processor.md) | Automated invoice extraction and validation |
| [Multi-Stage ETL Pipeline](./orchestration-multi-stage-etl.md) | Complex data transformation workflows |

### Prompts

| Use Case | Description |
|----------|-------------|
| [Few-Shot SQL Generation](./prompts-few-shot-sql.md) | Natural language to SQL with example-based learning |
| [Dynamic Tool Injection](./prompts-dynamic-tool-injection.md) | Runtime prompt modification for tool availability |

### Retrievers

| Use Case | Description |
|----------|-------------|
| [Multi-Document Summarizer](./retrievers-multi-doc-summarizer.md) | Synthesize information across multiple documents |
| [Regulatory Search](./retrievers-regulatory-search.md) | Compliance-focused document retrieval |

### Safety

| Use Case | Description |
|----------|-------------|
| [Children's Story Generator](./safety-children-stories.md) | Age-appropriate content generation with guardrails |
| [Financial Compliance](./safety-financial-compliance.md) | Regulated industry content safety |

### Schema

| Use Case | Description |
|----------|-------------|
| [Medical Record Standardization](./schema-medical-record-standardization.md) | Convert medical data to standard formats |
| [Legal Entity Extraction](./schema-legal-entity-extraction.md) | Extract structured data from legal documents |

### Server

| Use Case | Description |
|----------|-------------|
| [Search Everything API](./server-search-everything.md) | Unified search API across multiple data sources |
| [Customer Support Gateway](./server-customer-support-gateway.md) | REST API for customer support integration |

### TextSplitters

| Use Case | Description |
|----------|-------------|
| [Optimizing RAG for Large Repos](./textsplitters-optimizing-rag-large-repos.md) | Code-aware splitting for large codebases |
| [Scientific Paper Processing](./textsplitters-scientific-paper-processing.md) | Section-aware splitting for academic papers |

### VectorStores

| Use Case | Description |
|----------|-------------|
| [Recommendation Engine](./vectorstores-recommendation-engine.md) | Content recommendations using vector similarity |
| [Enterprise Knowledge Q&A](./vectorstores-enterprise-knowledge-qa.md) | Scalable knowledge base with vector search |

---

## Voice Framework Use Cases

The voice framework provides comprehensive support for building voice-enabled AI applications.

### Backend

| Use Case | Description |
|----------|-------------|
| [Interactive Voice Response (IVR)](./voice-backend-ivr.md) | Automated phone system with intelligent routing |
| [Outbound Calling System](./voice-backend-outbound-calling.md) | Proactive voice outreach with personalization |

### Speech-to-Speech (S2S)

| Use Case | Description |
|----------|-------------|
| [Bilingual Tutor](./voice-s2s-bilingual-tutor.md) | Real-time language learning with voice interaction |
| [Hotel Concierge](./voice-s2s-hotel-concierge.md) | 24/7 voice-based guest services |

### Sessions

| Use Case | Description |
|----------|-------------|
| [Voice Sessions Overview](./voice-sessions.md) | Session management patterns and best practices |
| [Multi-Turn Forms](./voice-session-multi-turn-forms.md) | Complex form filling through voice interaction |

### Speech-to-Text (STT)

| Use Case | Description |
|----------|-------------|
| [Meeting Minutes](./voice-stt-meeting-minutes.md) | Automated transcription with speaker diarization |
| [Industrial Control](./voice-stt-industrial-control.md) | Voice commands for industrial environments |

### Text-to-Speech (TTS)

| Use Case | Description |
|----------|-------------|
| [E-Learning Voiceovers](./voice-tts-elearning-voiceovers.md) | Generate educational content narration |
| [Interactive Audiobooks](./voice-tts-interactive-audiobooks.md) | Dynamic storytelling with voice synthesis |

### Turn Detection

| Use Case | Description |
|----------|-------------|
| [Barge-In Detection](./voice-turn-barge-in-detection.md) | Allow users to interrupt AI responses |
| [Low-Latency Prediction](./voice-turn-low-latency-prediction.md) | Predictive turn-taking for natural conversations |

### Voice Activity Detection (VAD)

| Use Case | Description |
|----------|-------------|
| [Multi-Speaker Segmentation](./voice-vad-multi-speaker-segmentation.md) | Identify and separate multiple speakers |
| [Noise-Resistant VAD](./voice-vad-noise-resistant.md) | Robust voice detection in noisy environments |

---

## Additional Resources

| Resource | Description |
|----------|-------------|
| [RAG Strategies](./rag-strategies.md) | Advanced retrieval-augmented generation patterns |

---

## Common Patterns Across Use Cases

All use cases demonstrate:

### 1. Configuration Management
- YAML configuration files
- Environment variable support
- Provider-specific settings
- Validation and defaults

### 2. Observability
- OpenTelemetry metrics
- Distributed tracing
- Structured logging
- Health checks

### 3. Error Handling
- Custom error types
- Error wrapping and context
- Retry strategies
- Circuit breakers

### 4. Testing
- Unit test examples
- Integration test scenarios
- Mock implementations
- Performance benchmarks

### 5. Deployment
- Production considerations
- Scaling strategies
- Performance optimization
- Security best practices

---

## Getting Started

1. **Choose a Use Case**: Review the categories above and select one that matches your needs
2. **Read the Documentation**: Each use case includes comprehensive implementation guides
3. **Review Configuration**: Study the YAML configuration examples
4. **Examine Code Examples**: Go code snippets demonstrate key patterns
5. **Set Up Observability**: Follow the OpenTelemetry setup instructions
6. **Deploy and Monitor**: Use the deployment guides and monitoring dashboards

> [!TIP]
> Start with a [Core Use Case](#core-use-cases) to understand the full framework integration patterns, then explore [Package Deep-Dives](#package-deep-dives) for specific component expertise.

---

## Framework Components Reference

Each use case leverages different combinations of Beluga AI components:

| Package | Path | Description |
|---------|------|-------------|
| Core | `pkg/core` | Runnable interface, dependency injection, utilities |
| Schema | `pkg/schema` | Message, Document, ToolCall types |
| Config | `pkg/config` | Configuration management with Viper |
| LLMs | `pkg/llms` | Unified LLM interface with multiple providers |
| ChatModels | `pkg/chatmodels` | ChatModel interface and implementations |
| Embeddings | `pkg/embeddings` | Embedder interface with providers |
| VectorStores | `pkg/vectorstores` | Vector storage and similarity search |
| Retrievers | `pkg/retrievers` | Document retrieval strategies |
| Memory | `pkg/memory` | Conversation memory management |
| Prompts | `pkg/prompts` | Prompt templates and rendering |
| Agents | `pkg/agents` | Agent framework with ReAct support |
| Tools | `pkg/agents/tools` | Tool registry and implementations |
| Orchestration | `pkg/orchestration` | Chains, graphs, and workflows |
| Monitoring | `pkg/monitoring` | OpenTelemetry observability |
| Server | `pkg/server` | REST and MCP server implementations |
| Voice | `pkg/voice` | Complete voice framework (STT, TTS, VAD) |

---

## Contributing

When creating new use cases:

1. Follow the standard use case structure (see existing use cases for examples)
2. Include comprehensive architecture diagrams
3. Provide complete configuration examples
4. Document observability setup
5. Include troubleshooting guides
6. Add code examples and test scenarios

---

## Additional Documentation

- [Framework Comparison](../framework-comparison.md) - Comparison with LangChain and CrewAI
- [Package Design Patterns](../package_design_patterns.md) - Framework design principles
- [Main README](https://github.com/lookatitude/beluga-ai/blob/main/README.md) - Framework overview and installation
