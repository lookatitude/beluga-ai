# LLM Provider Comparison

Compare all LLM providers to choose the right one for your needs.

## Feature Matrix

| Feature | OpenAI | Anthropic | Bedrock | Ollama |
|---------|--------|-----------|---------|--------|
| Models | GPT-3.5, GPT-4 | Claude 3 | Various | Local models |
| Streaming | ✅ | ✅ | ✅ | ✅ |
| Tool Calling | ✅ | ✅ | Varies | Limited |
| Cost | Medium | Medium | Low | Free |
| Privacy | Cloud | Cloud | Cloud | Local |
| Setup | Easy | Easy | Medium | Medium |

## Use Case Recommendations

### General Purpose
**Recommended:** OpenAI GPT-4 or Anthropic Claude
- Best quality
- Good tool support
- Reliable

### Cost-Sensitive
**Recommended:** OpenAI GPT-3.5 or AWS Bedrock
- Lower costs
- Good performance
- Production-ready

### Privacy-Critical
**Recommended:** Ollama
- Local execution
- No data leaves your system
- Free

### AWS Environments
**Recommended:** AWS Bedrock
- Integrated with AWS
- Cost-effective
- Managed service

## Cost Comparison

Approximate costs per 1M tokens (input/output):

- GPT-4: $30/$60
- GPT-3.5-turbo: $0.50/$1.50
- Claude 3 Sonnet: $3/$15
- Bedrock: Varies by model
- Ollama: Free (compute costs only)

## Performance Comparison

| Provider | Speed | Quality | Context Window |
|----------|-------|---------|----------------|
| GPT-4 | Medium | High | 128k |
| GPT-3.5 | Fast | Good | 16k |
| Claude 3 | Medium | High | 200k |
| Ollama | Varies | Medium | Model-dependent |

## Migration Between Providers

All providers use the same interface:

```go
// Switch providers easily
config := llms.NewConfig(
    llms.WithProvider("openai"), // or "anthropic", "ollama"
    llms.WithModelName("gpt-4"),
)
```

## Decision Guide

1. **Need privacy?** → Ollama
2. **Need best quality?** → GPT-4 or Claude
3. **Cost-sensitive?** → GPT-3.5 or Bedrock
4. **AWS environment?** → Bedrock
5. **Long context?** → Claude

---

**Next:** Read provider-specific guides or [Getting Started Tutorial](../../getting-started/)

