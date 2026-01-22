# Beluga AI Integrations

Welcome to the Beluga AI Integrations documentation! This directory contains step-by-step guides for integrating external services, tools, and platforms with the Beluga AI Framework.

## Overview

Integration guides help you connect Beluga AI with external services, third-party APIs, cloud platforms, and development tools. Each guide follows a tutorial-style format with runnable code examples and verification steps.

## Integration Categories

### Foundation Packages

#### Schema
- **[JSON Schema Validation](./schema/json-schema-validation.md)** - Validate data structures using JSON Schema
- **[Pydantic/Go Struct Bridge](./schema/pydantic-go-struct-bridge.md)** - Convert between Pydantic models and Go structs

#### Core
- **[Standard Library Context Deep Dive](./core/context-deep-dive.md)** - Advanced context usage patterns
- **[Zap/Logrus Logger Providers](./core/zap-logrus-providers.md)** - Integrate structured logging with Zap or Logrus

#### Config
- **[Viper & Environment Overrides](./config/viper-environment-overrides.md)** - Configuration management with Viper
- **[HashiCorp Vault Connector](./config/hashicorp-vault-connector.md)** - Secure secret management with Vault

#### Monitoring
- **[Datadog Dashboard Templates](./monitoring/datadog-dashboard-templates.md)** - Pre-built dashboards for Beluga AI metrics
- **[LangSmith Debugging Integration](./monitoring/langsmith-debugging-integration.md)** - Debug LLM calls with LangSmith

### Provider Packages

#### LLMs
- **[AWS Bedrock Integration](./llms/aws-bedrock-integration.md)** - Use AWS Bedrock models with Beluga AI
- **[Anthropic Claude Enterprise](./llms/anthropic-claude-enterprise.md)** - Enterprise Claude setup and configuration

#### Embeddings
- **[Ollama Local Embeddings](./embeddings/ollama-local-embeddings.md)** - Run embeddings locally with Ollama
- **[Cohere Multilingual Embedder](./embeddings/cohere-multilingual-embedder.md)** - Multilingual embeddings with Cohere

#### Vector Stores
- **[Qdrant Cloud Cluster](./vectorstores/qdrant-cloud-cluster.md)** - Deploy and use Qdrant cloud clusters
- **[Pinecone Serverless Integration](./vectorstores/pinecone-serverless.md)** - Serverless vector search with Pinecone

#### Prompts
- **[LangChain Hub Prompt Loading](./prompts/langchain-hub-loading.md)** - Load prompts from LangChain Hub
- **[Local Filesystem Template Store](./prompts/local-filesystem-template-store.md)** - Manage prompts in local files

### Higher-Level Packages

#### ChatModels
- **[OpenAI Assistants API Bridge](./chatmodels/openai-assistants-api-bridge.md)** - Integrate OpenAI Assistants with Beluga AI
- **[Custom Mock for UI Testing](./chatmodels/custom-mock-ui-testing.md)** - Create mocks for UI testing

#### Memory
- **[MongoDB Context Persistence](./memory/mongodb-context-persistence.md)** - Persist conversation context in MongoDB
- **[Redis Distributed Locking](./memory/redis-distributed-locking.md)** - Distributed locking for memory operations

#### Retrievers
- **[Elasticsearch Keyword Search](./retrievers/elasticsearch-keyword-search.md)** - Hybrid search with Elasticsearch
- **[Weaviate RAG Connector](./retrievers/weaviate-rag-connector.md)** - RAG pipeline with Weaviate

#### Orchestration
- **[NATS Message Bus](./orchestration/nats-message-bus.md)** - Distributed messaging with NATS
- **[Kubernetes Job Scheduler](./orchestration/kubernetes-job-scheduler.md)** - Schedule workflows on Kubernetes

### Specialized Packages

#### Safety
- **[Third-Party Ethical API Filter](./safety/third-party-ethical-api-filter.md)** - Content safety with external APIs
- **[SafetyResult JSON Reporting](./safety/safety-result-json-reporting.md)** - Export safety results as JSON

#### Server
- **[Kubernetes Helm Deployment](./server/kubernetes-helm-deployment.md)** - Deploy Beluga AI services with Helm
- **[Auth0/JWT Authentication](./server/auth0-jwt-authentication.md)** - Secure APIs with Auth0 and JWT

#### Voice - STT
- **[Deepgram Live Streams](./voice/stt/deepgram-live-streams.md)** - Real-time transcription with Deepgram
- **[Amazon Transcribe Audio Websockets](./voice/stt/amazon-transcribe-websockets.md)** - Streaming transcription with AWS Transcribe

#### Voice - TTS
- **[ElevenLabs Streaming API](./voice/tts/elevenlabs-streaming-api.md)** - High-quality voice synthesis with ElevenLabs
- **[Azure Cognitive Services Speech](./voice/tts/azure-cognitive-services-speech.md)** - TTS with Azure Speech Services

#### Voice - S2S
- **[OpenAI Realtime API](./voice/s2s/openai-realtime-api.md)** - End-to-end voice with OpenAI Realtime
- **[Amazon Nova Bedrock Streaming](./voice/s2s/amazon-nova-bedrock-streaming.md)** - Streaming S2S with Amazon Nova

#### Messaging
- **[Twilio Conversations API](./messaging/twilio-conversations-api.md)** - Multi-channel messaging with Twilio
- **[Slack Webhook Handler](./messaging/slack-webhook-handler.md)** - Integrate with Slack via webhooks

#### Multimodal
- **[Pixtral Mistral Integration](./multimodal/pixtral-mistral-integration.md)** - Vision-language models with Pixtral
- **[Google Vertex AI Vision](./multimodal/google-vertex-ai-vision.md)** - Vision capabilities with Vertex AI

#### Document Loaders
- **[AWS S3 Event-Driven Loader](./docloaders/aws-s3-event-driven-loader.md)** - Auto-load documents from S3 events
- **[Google Drive API Scraper](./docloaders/google-drive-api-scraper.md)** - Load documents from Google Drive

#### Text Splitters
- **[Tiktoken Byte-Pair Encoding](./textsplitters/tiktoken-byte-pair-encoding.md)** - Token-aware splitting with Tiktoken
- **[SpaCy Sentence Tokenizer](./textsplitters/spacy-sentence-tokenizer.md)** - Language-aware splitting with SpaCy

## Quick Start

1. **Choose your integration** - Browse the categories above to find the integration you need
2. **Follow the guide** - Each guide includes step-by-step instructions with runnable code
3. **Verify it works** - Each guide includes verification steps to test your integration
4. **Customize** - Adapt the examples to your specific use case

## Integration Template

Creating a new integration? Use the [Integration Template](./_template.md) as a starting point.

## Related Documentation

- **[Getting Started Tutorials](../getting-started/)** - Learn the basics of Beluga AI
- **[Guides](../guides/)** - Deep dives into framework concepts
- **[Use Cases](../use-cases/)** - Real-world implementation examples
- **[Cookbook](../cookbook/)** - Quick reference recipes
- **[API Documentation](../api/)** - Complete API reference

## Contributing

Found an issue with an integration guide or want to add a new one? See our [Contributing Guide](../../CONTRIBUTING.md) for details.

---

**Need help?** Check the [Troubleshooting Guide](../troubleshooting.md) or open an issue on GitHub.
