# Provider Documentation

This directory contains detailed documentation for all Beluga AI Framework providers.

## Table of Contents

### LLM Providers
- [OpenAI](./llms/openai.md) - GPT models
- [Anthropic](./llms/anthropic.md) - Claude models
- [AWS Bedrock](./llms/bedrock.md) - AWS foundation models
- [Ollama](./llms/ollama.md) - Local models
- [Provider Comparison](./llms/comparison.md) - Compare all LLM providers

### Vector Store Providers
- [InMemory](./vectorstores/inmemory.md) - In-memory vector store
- [PgVector](./vectorstores/pgvector.md) - PostgreSQL with pgvector
- [Pinecone](./vectorstores/pinecone.md) - Managed vector database
- [Provider Comparison](./vectorstores/comparison.md) - Compare all vector stores

### Embedding Providers
- [OpenAI Embeddings](./embeddings/openai.md) - OpenAI embedding models
- [Ollama Embeddings](./embeddings/ollama.md) - Local embedding models
- [Provider Selection Guide](./embeddings/selection.md) - Choose the right provider

## Quick Reference

### LLM Provider Selection

| Provider | Best For | Cost | Privacy |
|----------|----------|------|---------|
| OpenAI | General purpose, high quality | Medium | Cloud |
| Anthropic | Safety, long context | Medium | Cloud |
| Bedrock | AWS environments | Low | Cloud |
| Ollama | Privacy, local execution | Free | Local |

### Vector Store Selection

| Provider | Best For | Persistence | Scalability |
|----------|----------|-------------|--------------|
| InMemory | Development, testing | No | Limited |
| PgVector | Production, ACID | Yes | High |
| Pinecone | Cloud-native, managed | Yes | Very High |

### Embedding Provider Selection

| Provider | Best For | Quality | Cost |
|----------|----------|---------|------|
| OpenAI | Production, high quality | High | Medium |
| Ollama | Privacy, local | Medium | Free |

## Getting Started

1. Choose providers based on your needs
2. Review provider-specific documentation
3. Configure providers in your application
4. Test with provider examples

## Related Documentation

- [Installation Guide](../INSTALLATION.md) - Setup instructions
- [Getting Started Tutorial](../getting-started/) - Step-by-step guides
- [Best Practices](../BEST_PRACTICES.md) - Production patterns
- [Troubleshooting](../TROUBLESHOOTING.md) - Common issues

---

**Start Here:** Review [LLM Provider Comparison](./llms/comparison.md) or [Vector Store Comparison](./vectorstores/comparison.md)

