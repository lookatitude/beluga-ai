# Embedding Provider Selection Guide

Choose the right embedding provider for your needs.

## Quick Decision Guide

1. **Need privacy?** → Ollama
2. **Need best quality?** → OpenAI
3. **Cost-sensitive?** → Ollama (free) or OpenAI (pay-per-use)
4. **Production use?** → OpenAI

## Provider Comparison

| Provider | Quality | Cost | Privacy | Setup |
|----------|---------|------|---------|-------|
| OpenAI | High | Medium | Cloud | Easy |
| Ollama | Medium | Free | Local | Medium |

## Use Cases

### Production RAG Systems
**Recommended:** OpenAI
- High quality embeddings
- Reliable API
- Good documentation

### Privacy-Sensitive Applications
**Recommended:** Ollama
- Local execution
- No data leaves system
- Free

### Development/Testing
**Recommended:** Ollama or Mock
- Fast iteration
- No API costs
- Easy setup

## Model Selection

### OpenAI Models

- `text-embedding-ada-002`: 1536 dims, cost-effective
- `text-embedding-3-small`: 1536 dims, improved quality
- `text-embedding-3-large`: 3072 dims, best quality

### Ollama Models

- `nomic-embed-text`: Good quality, local
- Check available models: `ollama list`

## Migration

All providers use the same interface:

```go
// Switch providers easily
embedder, _ := factory.NewEmbedder("openai") // or "ollama"
```

---

**Next:** [OpenAI Embeddings](./openai.md) or [Ollama Embeddings](./ollama.md)

