---
title: Troubleshooting
sidebar_position: 3
---

# Beluga AI Framework - Troubleshooting Guide

This guide helps you diagnose and resolve common issues when using Beluga AI Framework.

## Table of Contents

1. [Common Errors](#common-errors)
2. [Performance Issues](#performance-issues)
3. [Configuration Problems](#configuration-problems)
4. [Provider-Specific Issues](#provider-specific-issues)
5. [Debugging Tips](#debugging-tips)
6. [FAQ](#faq)

## Common Errors

### Installation Errors

#### "Go version mismatch"

**Problem:** `go version` shows version less than 1.24

**Solution:**
```bash
# Check current version
go version

# Update Go (see Installation Guide)
# Verify after update
go version
```

#### "Module download errors"

**Problem:** `go mod download` fails

**Solution:**
```bash
# Set Go proxy
go env -w GOPROXY=https://proxy.golang.org,direct

# Clear module cache
go clean -modcache

# Retry download
go mod download
```

### Configuration Errors

#### "API key not found"

**Problem:** Provider authentication fails

**Solution:**
```bash
# Verify environment variable is set
echo $OPENAI_API_KEY

# Set if missing
export OPENAI_API_KEY="your-key-here"

# Or use config file
api_key: "${OPENAI_API_KEY}"
```

#### "Invalid configuration"

**Problem:** Configuration validation fails

**Solution:**
```go
// Check configuration structure
if err := config.Validate(); err != nil {
    fmt.Printf("Config error: %v\n", err)
}

// Verify required fields
// Check data types
// Validate ranges
```

### Provider Connection Errors

#### "Provider not found"

**Problem:** Provider name not recognized

**Solution:**
```go
// Check available providers
factory := llms.NewFactory()
providers := factory.ListProviders()
fmt.Printf("Available: %v\n", providers)

// Use correct provider name
// Supported: "openai", "anthropic", "bedrock", "ollama"
```

#### "Connection timeout"

**Problem:** Requests timeout

**Solution:**
```go
// Increase timeout
config := llms.NewConfig(
    llms.WithTimeout(60 * time.Second),
)

// Check network connectivity
// Verify API endpoint URL
// Check firewall settings
```

### Authentication Errors

#### "Invalid API key"

**Problem:** Authentication fails

**Solution:**
```bash
# Verify API key is correct
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# Check key permissions
# Verify key hasn't expired
# Regenerate if needed
```

## Performance Issues

### Slow LLM Responses

**Problem:** LLM calls are slow

**Solutions:**
1. **Use faster models:**
   ```go
   llms.WithModelName("gpt-3.5-turbo") // Faster than gpt-4
   ```

2. **Reduce max tokens:**
   ```go
   llms.WithMaxTokensConfig(500) // Limit response length
   ```

3. **Enable streaming:**
   ```go
   streamChan, _ := provider.StreamChat(ctx, messages)
   ```

4. **Use batch processing:**
   ```go
   results, _ := provider.Batch(ctx, inputs)
   ```

### Memory Issues

**Problem:** High memory usage

**Solutions:**
1. **Use window memory:**
   ```go
   mem, _ := memory.NewMemory(
       memory.MemoryTypeWindow,
       memory.WithWindowSize(10),
   )
   ```

2. **Clear memory periodically:**
   ```go
   mem.Clear(ctx)
   ```

3. **Use summary memory for long conversations:**
   ```go
   mem, _ := memory.NewMemory(memory.MemoryTypeSummary)
   ```

### Vector Store Performance

**Problem:** Slow similarity search

**Solutions:**
1. **Optimize index:**
   ```sql
   CREATE INDEX ON documents USING ivfflat (embedding vector_cosine_ops);
   ```

2. **Reduce search K:**
   ```go
   store.SimilaritySearchByQuery(ctx, query, 5, embedder) // Lower K
   ```

3. **Use appropriate vector store:**
   - InMemory: Fast but limited
   - PgVector: Good for production
   - Pinecone: Managed, optimized

### Embedding Bottlenecks

**Problem:** Embedding generation is slow

**Solutions:**
1. **Use batch processing:**
   ```go
   embeddings, _ := embedder.EmbedDocuments(ctx, texts) // Batch
   ```

2. **Use local models:**
   ```go
   // Ollama for local, fast embeddings
   embedder, _ := factory.NewEmbedder("ollama")
   ```

3. **Cache embeddings:**
   ```go
   if cached, found := cache.Get(text); found {
       return cached.([]float32), nil
   }
   ```

## Configuration Problems

### Invalid YAML

**Problem:** YAML parsing fails

**Solution:**
```bash
# Validate YAML syntax
yamllint config.yaml

# Check indentation
# Verify quotes for strings
# Check for special characters
```

### Missing Environment Variables

**Problem:** Environment variables not found

**Solution:**
```bash
# Check if set
env | grep OPENAI_API_KEY

# Set in shell
export OPENAI_API_KEY="your-key"

# Or use .env file
source .env

# Or set in system
# Linux: ~/.bashrc or ~/.zshrc
# macOS: ~/.zshrc
# Windows: System Properties > Environment Variables
```

### Provider Misconfiguration

**Problem:** Provider doesn't work

**Solution:**
```go
// Verify provider configuration
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"), // Verify model exists
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)

// Check model availability
// Verify API endpoint
// Check rate limits
```

### Validation Errors

**Problem:** Configuration validation fails

**Solution:**
```go
// Check validation errors
if err := config.Validate(); err != nil {
    fmt.Printf("Validation errors:\n")
    // Review each error
    // Fix invalid values
    // Check required fields
}
```

## Provider-Specific Issues

### OpenAI

#### Rate Limit Errors

**Problem:** "Rate limit exceeded"

**Solution:**
```go
// Implement retry with backoff
config := llms.NewConfig(
    llms.WithRetryConfig(5, 2*time.Second, 2.0),
)

// Or reduce request rate
limiter := rate.NewLimiter(rate.Every(time.Second), 10)
```

#### Model Not Found

**Problem:** "Model not found"

**Solution:**
```go
// Use correct model name
// Available: "gpt-4", "gpt-3.5-turbo", "gpt-4-turbo"
llms.WithModelName("gpt-4") // Not "gpt4"
```

### Anthropic

#### API Errors

**Problem:** Anthropic API errors

**Solution:**
```go
// Check API key format
// Verify model name: "claude-3-sonnet-20240229"
// Check message format
// Verify tool use format
```

### AWS Bedrock

#### Permission Errors

**Problem:** "Access denied"

**Solution:**
```bash
# Verify IAM permissions
aws bedrock list-foundation-models

# Check model access
# Verify region
# Check credentials
```

#### Region Issues

**Problem:** Model not available in region

**Solution:**
```go
// Use correct region
config := llms.NewConfig(
    llms.WithProvider("bedrock"),
    llms.WithProviderSpecific("region", "us-east-1"),
)
```

### Ollama

#### Connection Issues

**Problem:** Cannot connect to Ollama

**Solution:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama
ollama serve

# Verify model is available
ollama list
```

#### Model Not Found

**Problem:** "Model not found"

**Solution:**
```bash
# Pull the model
ollama pull llama2

# Verify model exists
ollama list
```

### Vector Stores

#### PgVector Connection

**Problem:** Cannot connect to PostgreSQL

**Solution:**
```bash
# Check PostgreSQL is running
sudo systemctl status postgresql

# Verify connection string
psql "postgres://user:pass@localhost/db"

# Check pgvector extension
psql -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

#### Pinecone Connection

**Problem:** Pinecone connection fails

**Solution:**
```go
// Verify API key
// Check environment/region
// Verify index exists
// Check index configuration
```

## Debugging Tips

### Enable Debug Logging

```go
logger := monitoring.NewStructuredLogger(
    "app",
    monitoring.WithLogLevel("debug"),
)
```

### Use Tracing

```go
ctx, span := tracer.StartSpan(ctx, "operation")
defer span.End()

span.SetAttributes(
    attribute.String("key", "value"),
)
```

### Check Metrics

```go
metrics := monitoring.NewMetricsCollector()
// Review metrics for patterns
// Check error rates
// Monitor latency
```

### Common Pitfalls

1. **Forgetting context cancellation:**
   ```go
   // Always check context
   select {
   case <-ctx.Done():
       return ctx.Err()
   default:
   }
   ```

2. **Not handling errors:**
   ```go
   // Always check errors
   result, err := provider.Generate(ctx, messages)
   if err != nil {
       // Handle error
   }
   ```

3. **Memory leaks:**
   ```go
   // Always clean up
   defer store.Close()
   defer agent.Finalize()
   ```

## FAQ

### General Questions

**Q: Which LLM provider should I use?**
A: Depends on your needs:
- Best quality: GPT-4 or Claude
- Cost-effective: GPT-3.5
- Privacy: Ollama
- AWS environment: Bedrock

**Q: How do I handle rate limits?**
A: Implement retry logic with exponential backoff:
```go
llms.WithRetryConfig(5, 2*time.Second, 2.0)
```

**Q: Can I use multiple providers?**
A: Yes, create multiple provider instances and switch between them.

### Memory Questions

**Q: Which memory type should I use?**
A:
- Short conversations: Buffer
- Long conversations: Window
- Very long: Summary
- Need search: Vector Store

**Q: How do I persist memory?**
A: Save memory variables to database or file, reload on startup.

### RAG Questions

**Q: What chunk size should I use?**
A: Typically 500-1000 tokens, adjust based on your documents.

**Q: How many documents should I retrieve?**
A: Start with 3-5, adjust based on context window and quality.

### Performance Questions

**Q: How do I improve response time?**
A:
- Use faster models
- Enable streaming
- Reduce max tokens
- Use batch processing

**Q: How do I reduce costs?**
A:
- Use cheaper models
- Cache responses
- Batch requests
- Optimize prompts

## Getting Help

If you can't resolve an issue:

1. **Check documentation:**
   - [Quick Start Guide](./quickstart)
   - [Getting Started Tutorial](../../getting-started/)
   - [Best Practices](../guides/best-practices)

2. **Search GitHub Issues:**
   - https://github.com/lookatitude/beluga-ai/issues

3. **Create a new issue:**
   - Include Go version
   - Include error messages
   - Include code snippets
   - Include configuration

---

**Last Updated:** Troubleshooting guide is actively maintained. Check back for updates.

