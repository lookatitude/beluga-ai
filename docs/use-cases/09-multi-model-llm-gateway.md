# Use Case 9: Multi-Model LLM Gateway with Observability

## Overview & Objectives

### Business Problem

Applications need to use multiple LLM providers for different use cases, load balancing, failover, and cost optimization. Managing multiple provider APIs, handling rate limits, monitoring usage, and ensuring reliability is complex and error-prone.

### Solution Approach

This use case implements a unified LLM gateway that:
- Provides a single interface for multiple LLM providers
- Implements load balancing and failover
- Monitors usage and performance comprehensively
- Handles rate limiting and retries
- Provides detailed observability

### Key Benefits

- **Provider Abstraction**: Single API for multiple providers
- **High Availability**: Automatic failover between providers
- **Cost Optimization**: Route to cost-effective providers
- **Comprehensive Observability**: Full metrics and tracing
- **Scalable**: Handles high request volumes

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Client Applications                          │
│              (Services, Agents, APIs)                          │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              │ Unified API
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              LLM Gateway (pkg/llms, pkg/server)                  │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Request    │→ │   Provider   │→ │   Response  │         │
│  │  Router      │  │  Selection   │  │  Handler    │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└────────────────────────────┬────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   LLM        │    │   Monitoring  │    │  Rate Limit  │
│  Providers   │    │  (pkg/        │    │  Manager     │
│  (pkg/llms)  │    │  monitoring)  │    │              │
│              │    │               │    │              │
│  - OpenAI    │    │  - Metrics    │    │  - Per       │
│  - Anthropic │    │  - Tracing    │    │    Provider  │
│  - Bedrock   │    │  - Logging    │    │  - Global    │
│  - Ollama    │    │               │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
```

## Component Usage

### Beluga AI Packages Used

1. **pkg/llms**
   - Unified LLM interface
   - Multiple provider implementations
   - Provider factory pattern

2. **pkg/chatmodels**
   - ChatModel interface
   - Provider implementations

3. **pkg/monitoring**
   - OpenTelemetry metrics
   - Distributed tracing
   - Structured logging

4. **pkg/server**
   - REST API server
   - Request routing

5. **pkg/config**
   - Provider configuration
   - Rate limit settings

6. **pkg/orchestration**
   - Request routing logic

## Implementation Guide

### Step 1: Create Gateway Service

```go
type LLMGateway struct {
    providers map[string]llms.ChatModel
    router    *ProviderRouter
    monitor   *monitoring.MetricsCollector
    rateLimiter *RateLimiter
}

func NewLLMGateway(ctx context.Context, cfg *config.Config) (*LLMGateway, error) {
    gateway := &LLMGateway{
        providers: make(map[string]llms.ChatModel),
        router:    NewProviderRouter(),
        monitor:   monitoring.NewMetricsCollector(),
        rateLimiter: NewRateLimiter(),
    }
    
    // Initialize providers
    providers := []string{"openai", "anthropic", "bedrock"}
    for _, name := range providers {
        provider, err := createProvider(ctx, name, cfg)
        if err != nil {
            log.Printf("Failed to create provider %s: %v", name, err)
            continue
        }
        gateway.providers[name] = provider
    }
    
    return gateway, nil
}
```

### Step 2: Provider Router

```go
type ProviderRouter struct {
    strategy RoutingStrategy
}

type RoutingStrategy interface {
    SelectProvider(ctx context.Context, request *LLMRequest, providers []string) (string, error)
}

// Round-robin routing
type RoundRobinRouter struct {
    current int
    mu      sync.Mutex
}

func (r *RoundRobinRouter) SelectProvider(ctx context.Context, request *LLMRequest, providers []string) (string, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if len(providers) == 0 {
        return "", fmt.Errorf("no providers available")
    }
    
    provider := providers[r.current]
    r.current = (r.current + 1) % len(providers)
    return provider, nil
}

// Cost-based routing
type CostBasedRouter struct{}

func (r *CostBasedRouter) SelectProvider(ctx context.Context, request *LLMRequest, providers []string) (string, error) {
    costs := map[string]float64{
        "openai":    0.03,
        "anthropic": 0.015,
        "bedrock":   0.008,
    }
    
    cheapest := providers[0]
    minCost := costs[cheapest]
    
    for _, p := range providers {
        if cost, ok := costs[p]; ok && cost < minCost {
            cheapest = p
            minCost = cost
        }
    }
    
    return cheapest, nil
}

// Health-based routing with failover
type HealthBasedRouter struct {
    healthChecker *HealthChecker
}

func (r *HealthBasedRouter) SelectProvider(ctx context.Context, request *LLMRequest, providers []string) (string, error) {
    // Check provider health
    healthy := []string{}
    for _, p := range providers {
        if r.healthChecker.IsHealthy(p) {
            healthy = append(healthy, p)
        }
    }
    
    if len(healthy) == 0 {
        return "", fmt.Errorf("no healthy providers")
    }
    
    // Select from healthy providers
    return healthy[0], nil
}
```

### Step 3: Request Handling

```go
func (g *LLMGateway) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
    // Start tracing
    tracer := otel.Tracer("llm-gateway")
    ctx, span := tracer.Start(ctx, "gateway.generate")
    defer span.End()
    
    // Check rate limits
    if !g.rateLimiter.Allow(request.Provider) {
        span.RecordError(fmt.Errorf("rate limit exceeded"))
        return nil, fmt.Errorf("rate limit exceeded")
    }
    
    // Select provider
    providerName, err := g.router.SelectProvider(ctx, request, g.getAvailableProviders())
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    span.SetAttributes(
        attribute.String("provider", providerName),
        attribute.String("model", request.Model),
    )
    
    // Get provider
    provider, ok := g.providers[providerName]
    if !ok {
        return nil, fmt.Errorf("provider not found: %s", providerName)
    }
    
    // Record metrics
    start := time.Now()
    g.monitor.Counter(ctx, "llm_requests_total", 1, map[string]string{
        "provider": providerName,
        "model":   request.Model,
    })
    
    // Generate response
    response, err := provider.Generate(ctx, request.Messages)
    
    duration := time.Since(start)
    g.monitor.Histogram(ctx, "llm_request_duration_seconds", duration.Seconds(), map[string]string{
        "provider": providerName,
        "model":   request.Model,
        "status":  getStatus(err),
    })
    
    if err != nil {
        span.RecordError(err)
        g.monitor.Counter(ctx, "llm_errors_total", 1, map[string]string{
            "provider": providerName,
            "error":    err.Error(),
        })
        
        // Retry with different provider
        return g.retryWithFallback(ctx, request, providerName)
    }
    
    // Record success
    g.monitor.Counter(ctx, "llm_success_total", 1, map[string]string{
        "provider": providerName,
    })
    
    return &LLMResponse{
        Content: response.GetContent(),
        Provider: providerName,
        Tokens: response.GetTokenUsage(),
    }, nil
}

func (g *LLMGateway) retryWithFallback(ctx context.Context, request *LLMRequest, failedProvider string) (*LLMResponse, error) {
    // Get alternative providers
    alternatives := g.getAlternativeProviders(failedProvider)
    
    for _, providerName := range alternatives {
        provider, ok := g.providers[providerName]
        if !ok {
            continue
        }
        
        response, err := provider.Generate(ctx, request.Messages)
        if err == nil {
            log.Printf("Fallback to provider %s succeeded", providerName)
            return &LLMResponse{
                Content:  response.GetContent(),
                Provider: providerName,
                Tokens:   response.GetTokenUsage(),
            }, nil
        }
    }
    
    return nil, fmt.Errorf("all providers failed")
}
```

## Workflow & Data Flow

### End-to-End Process Flow

1. **Request Reception**
   ```
   Client Request → Gateway → Validate Request
   ```

2. **Provider Selection**
   ```
   Request → Router → Select Provider
   ```

3. **Rate Limiting**
   ```
   Provider → Rate Limiter → Check Limits
   ```

4. **LLM Call**
   ```
   Provider → LLM API → Generate Response
   ```

5. **Response Handling**
   ```
   Response → Record Metrics → Return to Client
   ```

## Observability Setup

### Metrics to Monitor

- `llm_requests_total`: Total requests by provider
- `llm_request_duration_seconds`: Request latency
- `llm_errors_total`: Errors by provider and type
- `llm_tokens_used_total`: Token usage by provider
- `llm_rate_limit_hits_total`: Rate limit hits
- `provider_health_status`: Provider health status

## Configuration Examples

### Complete YAML Configuration

```yaml
# config.yaml
app:
  name: "llm-gateway"
  version: "1.0.0"

gateway:
  routing_strategy: "health_based"  # round_robin, cost_based, health_based
  enable_failover: true
  max_retries: 3
  timeout: 60s

providers:
  openai:
    enabled: true
    api_key: "${OPENAI_API_KEY}"
    models:
      - "gpt-4"
      - "gpt-3.5-turbo"
    rate_limit:
      requests_per_minute: 60
      tokens_per_minute: 90000
  
  anthropic:
    enabled: true
    api_key: "${ANTHROPIC_API_KEY}"
    models:
      - "claude-3-opus"
      - "claude-3-sonnet"
    rate_limit:
      requests_per_minute: 50
  
  bedrock:
    enabled: true
    region: "us-east-1"
    models:
      - "anthropic.claude-v2"
    rate_limit:
      requests_per_minute: 100

monitoring:
  otel:
    endpoint: "localhost:4317"
  metrics:
    enabled: true
    prefix: "llm_gateway"
  tracing:
    enabled: true
    sample_rate: 1.0

server:
  host: "0.0.0.0"
  port: 8080
```

## Deployment Considerations

### Production Requirements

- **High Availability**: Multiple gateway instances
- **Load Balancing**: Distribute requests across instances
- **Monitoring**: Comprehensive observability stack
- **Rate Limiting**: Per-provider and global limits

## Testing Strategy

### Unit Tests

```go
func TestProviderRouter(t *testing.T) {
    router := &RoundRobinRouter{}
    
    providers := []string{"openai", "anthropic"}
    provider1, _ := router.SelectProvider(context.Background(), nil, providers)
    provider2, _ := router.SelectProvider(context.Background(), nil, providers)
    
    assert.NotEqual(t, provider1, provider2)
}
```

## Troubleshooting Guide

### Common Issues

1. **Provider Failures**
   - Implement health checks
   - Enable automatic failover
   - Monitor provider status

2. **Rate Limit Issues**
   - Adjust rate limit settings
   - Implement request queuing
   - Use multiple provider accounts

3. **High Latency**
   - Optimize provider selection
   - Implement caching
   - Use faster models

## Conclusion

This Multi-Model LLM Gateway demonstrates Beluga AI's capabilities in building production-ready LLM infrastructure. The architecture showcases:

- **Provider Abstraction**: Unified interface for multiple providers
- **High Availability**: Automatic failover and retry logic
- **Comprehensive Observability**: Full metrics and tracing
- **Scalable Design**: Handles high request volumes

The system can be extended with:
- Advanced routing strategies
- Cost optimization algorithms
- Request caching
- A/B testing capabilities
- Multi-region support

