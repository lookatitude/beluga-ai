---
title: Best Practices
sidebar_position: 2
---

# Beluga AI Framework - Best Practices Guide

This guide provides best practices for building production-ready AI applications with Beluga AI Framework.

## Table of Contents

1. [Configuration Management](#configuration-management)
2. [Error Handling](#error-handling)
3. [Performance Optimization](#performance-optimization)
4. [Security Considerations](#security-considerations)
5. [Testing Strategies](#testing-strategies)
6. [Observability Setup](#observability-setup)
7. [Production Deployment](#production-deployment)
8. [Code Organization](#code-organization)
9. [Component Selection](#component-selection)

## Configuration Management

### YAML Structure

Organize configuration hierarchically:

```yaml
app:
  name: "my-app"
  environment: "production"

llm_providers:
  - name: "primary"
    provider: "openai"
    model_name: "gpt-4"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 5

embeddings:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-ada-002"
```

### Environment Variables

Use environment variables for secrets:

```go
apiKey := os.Getenv("OPENAI_API_KEY")
if apiKey == "" {
    return fmt.Errorf("OPENAI_API_KEY not set")
}
```

### Configuration Validation

Always validate configuration:

```go
if err := config.Validate(); err != nil {
    return fmt.Errorf("invalid configuration: %w", err)
}
```

### Secrets Management

**DO:**
- Use secret management services (AWS Secrets Manager, HashiCorp Vault)
- Store secrets in environment variables
- Rotate secrets regularly

**DON'T:**
- Commit secrets to version control
- Hardcode API keys
- Log secrets

## Error Handling

### Custom Error Types

Use structured error types:

```go
if err != nil {
    return NewLLMError("Generate", ErrCodeRateLimit, err)
}
```

### Error Wrapping

Always wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to generate response: %w", err)
}
```

### Retry Strategies

Implement retry logic for retryable errors:

```go
maxRetries := 3
for i := 0; i < maxRetries; i++ {
    result, err := provider.Generate(ctx, messages)
    if err == nil {
        return result, nil
    }
    
    if !llms.IsRetryableError(err) {
        return nil, err
    }
    
    time.Sleep(time.Duration(i+1) * time.Second)
}
```

### Circuit Breakers

Use circuit breakers for external services:

```go
circuitBreaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    MaxRequests: 5,
    Interval:    60 * time.Second,
    Timeout:    30 * time.Second,
})

result, err := circuitBreaker.Execute(func() (interface{}, error) {
    return provider.Generate(ctx, messages)
})
```

## Performance Optimization

### Batch Processing

Use batch processing for multiple requests:

```go
inputs := []any{input1, input2, input3}
results, err := provider.Batch(ctx, inputs)
```

### Connection Pooling

Reuse connections:

```go
// Create provider once, reuse
provider, _ := factory.CreateProvider("openai", config)

// Reuse for multiple requests
for _, input := range inputs {
    result, _ := provider.Generate(ctx, input)
}
```

### Caching

Cache identical requests:

```go
cacheKey := generateCacheKey(messages)
if cached, found := cache.Get(cacheKey); found {
    return cached.(Message), nil
}

response, err := provider.Generate(ctx, messages)
cache.Set(cacheKey, response, 1*time.Hour)
```

### Resource Management

Always clean up resources:

```go
defer store.Close()
defer agent.Finalize()
defer tracer.Shutdown(ctx)
```

## Security Considerations

### API Key Management

**DO:**
- Store keys in secure vaults
- Rotate keys regularly
- Use least privilege principles
- Monitor key usage

**DON'T:**
- Log API keys
- Share keys between environments
- Use keys with excessive permissions

### Input Validation

Validate all inputs:

```go
func validateInput(input string) error {
    if len(input) > 10000 {
        return fmt.Errorf("input too long")
    }
    if containsMaliciousContent(input) {
        return fmt.Errorf("malicious content detected")
    }
    return nil
}
```

### Rate Limiting

Implement rate limiting:

```go
limiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 requests/second

if !limiter.Allow() {
    return nil, fmt.Errorf("rate limit exceeded")
}
```

### Data Privacy

**DO:**
- Encrypt sensitive data
- Use local models for sensitive data
- Implement data retention policies
- Comply with regulations (GDPR, etc.)

## Testing Strategies

### Unit Testing

Test individual components:

```go
func TestLLMProvider(t *testing.T) {
    provider := NewMockProvider()
    result, err := provider.Generate(ctx, messages)
    require.NoError(t, err)
    assert.NotEmpty(t, result)
}
```

### Integration Testing

Test component interactions:

```go
func TestRAGPipeline(t *testing.T) {
    embedder := setupEmbedder(t)
    store := setupVectorStore(t, embedder)
    llm := setupLLM(t)
    
    // Test complete pipeline
    result, err := executeRAGQuery(t, embedder, store, llm, "query")
    require.NoError(t, err)
}
```

### Mock Usage

Use mocks for external services:

```go
mockLLM := &MockLLM{
    responses: []string{"response1", "response2"},
}
```

### Test Data Management

Use test fixtures:

```go
testDocuments := []schema.Document{
    schema.NewDocument("test content 1", nil),
    schema.NewDocument("test content 2", nil),
}
```

## Observability Setup

### OpenTelemetry Configuration

```go
tp, err := setupTracing(ctx)
if err != nil {
    log.Fatal(err)
}
defer tp.Shutdown(ctx)
```

### Structured Logging

```go
logger.Info(ctx, "Request processed", map[string]interface{}{
    "user_id": userID,
    "duration_ms": duration,
    "status": "success",
})
```

### Metrics Collection

```go
metrics.Counter(ctx, "requests_total", "Total requests", 1,
    map[string]string{"status": "success"})
```

### Health Checks

Implement health check endpoints:

```go
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    status := checkSystemHealth()
    w.WriteHeader(statusCode(status))
    json.NewEncoder(w).Encode(status)
}
```

## Production Deployment

### Health Checks

```go
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
```

### Graceful Shutdown

```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

server.Shutdown(ctx)
```

### Resource Limits

Set appropriate limits:

```yaml
resources:
  limits:
    memory: "2Gi"
    cpu: "1000m"
  requests:
    memory: "1Gi"
    cpu: "500m"
```

### Scaling Strategies

- **Horizontal scaling**: Multiple instances
- **Vertical scaling**: Larger instances
- **Auto-scaling**: Based on metrics

## Code Organization

### Package Structure

Follow standard package layout:

```
pkg/
├── iface/          # Interfaces
├── internal/       # Private implementations
├── providers/      # Provider implementations
├── config.go       # Configuration
└── metrics.go      # Observability
```

### Interface Design

Use small, focused interfaces:

```go
type Embedder interface {
    EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, text string) ([]float32, error)
}
```

### Dependency Injection

Inject dependencies via constructors:

```go
func NewAgent(llm ChatModel, memory Memory, tools []Tool) *Agent {
    return &Agent{
        llm:    llm,
        memory: memory,
        tools:  tools,
    }
}
```

### Factory Patterns

Use factories for provider creation:

```go
factory := llms.NewFactory()
provider, err := factory.CreateProvider("openai", config)
```

## Component Selection

### When to Use Which Component

#### LLM Selection

- **GPT-4**: Complex reasoning, high quality
- **GPT-3.5**: Cost-effective, fast
- **Claude**: Long context, safety
- **Ollama**: Privacy, local execution

#### Memory Selection

- **Buffer**: Short conversations, full history
- **Window**: Long conversations, recent context
- **Summary**: Very long conversations
- **Vector Store**: Semantic search needed

#### Vector Store Selection

- **InMemory**: Development, small datasets
- **PgVector**: Production, ACID compliance
- **Pinecone**: Cloud-native, managed

### Decision Trees

**Choosing an LLM Provider:**
1. Need privacy? → Ollama
2. Need best quality? → GPT-4 or Claude
3. Cost-sensitive? → GPT-3.5
4. AWS environment? → Bedrock

**Choosing Memory Type:**
1. Short conversations? → Buffer
2. Long conversations? → Window
3. Very long? → Summary
4. Need search? → Vector Store

## Additional Resources

- [Package Design Patterns](../guides/package-design-patterns)
- [Architecture Documentation](../guides/architecture)
- [Troubleshooting Guide](../guides/troubleshooting)
- [Getting Started Tutorial](../../getting-started/)

## API Reference

For detailed API documentation on advanced patterns and extensibility:

- **[Orchestration Package](../../api/packages/orchestration)** - Workflow orchestration, chains, and graphs
- **[Monitoring Package](../../api/packages/monitoring)** - Observability, metrics, and tracing
- **[Config Package](../../api/packages/config)** - Configuration management and providers
- **[Agents Package](../../api/packages/agents)** - Agent interfaces and extensibility
- **[Tools Package](../../api/packages/tools)** - Tool interfaces for custom tools
- **[LLMs Package](../../api/packages/llms)** - LLM interfaces for custom providers
- **[VectorStores Package](../../api/packages/vectorstores)** - Vector store interfaces for custom stores
- **[Embeddings Package](../../api/packages/embeddings)** - Embedding interfaces for custom providers

See the [complete API Reference](../../api/index) for all available packages and extensibility interfaces.

---

**Last Updated:** Best practices guide is actively maintained. Check back for updates.

