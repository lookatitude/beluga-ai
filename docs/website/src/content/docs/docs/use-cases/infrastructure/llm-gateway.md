---
title: Multi-Model LLM Gateway
description: "Build a unified LLM gateway with load balancing, failover, rate limiting, and observability across multiple providers in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "LLM gateway, multi-provider routing, load balancing AI, API gateway LLM, failover proxy, Beluga AI, Go, AI infrastructure"
---

When an application depends on a single LLM provider, every rate limit hit, timeout, or outage becomes a user-facing failure. Multi-provider setups solve availability but introduce new problems: each provider has different APIs, error formats, rate limit semantics, and pricing models. Teams end up writing provider-specific client code scattered across the codebase, with no unified view of cost, latency, or error rates across providers.

A gateway layer centralizes these concerns. All application code talks to one interface, and the gateway handles provider selection, rate limit enforcement, failover, and observability. This separation means adding a new provider or changing routing strategy requires zero application code changes.

Beluga AI's LLM package provides this gateway pattern through its registry and `ChatModel` interface — all providers implement the same interface, so the gateway composes them with standard routing, resilience, and monitoring.

## Solution Architecture

The gateway sits between application code and LLM providers. A pluggable router selects the best provider for each request based on configurable strategy (round-robin for load distribution, cost-based for budget optimization, health-based for reliability). Per-provider rate limiting prevents quota exhaustion before requests reach the provider. OpenTelemetry integration using GenAI semantic conventions (`gen_ai.*` attributes) provides unified observability across all providers.

```mermaid
graph TB
    A[Client Apps] --> B[LLM Gateway]
    B --> C[Router<br>Round Robin / Cost / Health]
    C --> D[Rate Limiter]
    D --> E[OpenAI]
    D --> F[Anthropic]
    D --> G[Bedrock]
    D --> H[Ollama]
    I[Metrics/Tracing] --> B
    J[Circuit Breaker] --> C
```

## Gateway Service

The gateway uses Beluga AI's registry pattern (`llm.New()`) to create providers by name, keeping initialization uniform. Each provider is stored by name in a map, enabling dynamic addition and removal at runtime. The blank imports (`_ "github.com/lookatitude/beluga-ai/llm/providers/..."`) register providers via `init()` so they're available through the registry without explicit wiring.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    _ "github.com/lookatitude/beluga-ai/llm/providers/bedrock"
)

type LLMGateway struct {
    providers   map[string]llm.ChatModel
    router      ProviderRouter
    rateLimiter *RateLimiter
    mu          sync.RWMutex
}

func NewLLMGateway(ctx context.Context) (*LLMGateway, error) {
    gateway := &LLMGateway{
        providers:   make(map[string]llm.ChatModel),
        router:      NewRoundRobinRouter(),
        rateLimiter: NewRateLimiter(),
    }

    // Initialize OpenAI
    openaiModel, err := llm.New("openai", llm.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    })
    if err != nil {
        log.Printf("Failed to create OpenAI provider: %v", err)
    } else {
        gateway.providers["openai"] = openaiModel
    }

    // Initialize Anthropic
    anthropicModel, err := llm.New("anthropic", llm.ProviderConfig{
        APIKey: os.Getenv("ANTHROPIC_API_KEY"),
        Model:  "claude-3-5-sonnet-20241022",
    })
    if err != nil {
        log.Printf("Failed to create Anthropic provider: %v", err)
    } else {
        gateway.providers["anthropic"] = anthropicModel
    }

    // Initialize Bedrock
    bedrockModel, err := llm.New("bedrock", llm.ProviderConfig{
        Region: "us-east-1",
        Model:  "anthropic.claude-v2",
    })
    if err != nil {
        log.Printf("Failed to create Bedrock provider: %v", err)
    } else {
        gateway.providers["bedrock"] = bedrockModel
    }

    if len(gateway.providers) == 0 {
        return nil, fmt.Errorf("no providers available")
    }

    return gateway, nil
}

func (g *LLMGateway) Generate(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
    // Select provider
    providerName, err := g.router.SelectProvider(ctx, g.getAvailableProviders())
    if err != nil {
        return nil, err
    }

    // Check rate limit
    if !g.rateLimiter.Allow(providerName) {
        return nil, fmt.Errorf("rate limit exceeded for provider %s", providerName)
    }

    // Get provider
    g.mu.RLock()
    provider, ok := g.providers[providerName]
    g.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("provider not found: %s", providerName)
    }

    // Generate response
    start := time.Now()
    resp, err := provider.Generate(ctx, msgs)
    duration := time.Since(start)

    log.Printf("Provider %s took %v", providerName, duration)

    if err != nil {
        // Retry with fallback provider
        return g.retryWithFallback(ctx, msgs, providerName)
    }

    return resp, nil
}

func (g *LLMGateway) getAvailableProviders() []string {
    g.mu.RLock()
    defer g.mu.RUnlock()

    names := make([]string, 0, len(g.providers))
    for name := range g.providers {
        names = append(names, name)
    }
    return names
}

func (g *LLMGateway) retryWithFallback(ctx context.Context, msgs []schema.Message, failedProvider string) (*schema.AIMessage, error) {
    alternatives := g.getAlternativeProviders(failedProvider)

    for _, providerName := range alternatives {
        g.mu.RLock()
        provider, ok := g.providers[providerName]
        g.mu.RUnlock()

        if !ok || !g.rateLimiter.Allow(providerName) {
            continue
        }

        log.Printf("Falling back to provider %s", providerName)
        resp, err := provider.Generate(ctx, msgs)
        if err == nil {
            return resp, nil
        }
    }

    return nil, fmt.Errorf("all providers failed")
}

func (g *LLMGateway) getAlternativeProviders(exclude string) []string {
    all := g.getAvailableProviders()
    alternatives := []string{}
    for _, name := range all {
        if name != exclude {
            alternatives = append(alternatives, name)
        }
    }
    return alternatives
}
```

## Provider Routing Strategies

The router is defined as an interface (`ProviderRouter`) so strategies are swappable without changing the gateway. This follows the same interface-first pattern used throughout Beluga AI — define the contract, then provide implementations. Three common strategies cover most production needs:

```go
type ProviderRouter interface {
    SelectProvider(ctx context.Context, providers []string) (string, error)
}

// Round-robin routing
type RoundRobinRouter struct {
    current int
    mu      sync.Mutex
}

func NewRoundRobinRouter() *RoundRobinRouter {
    return &RoundRobinRouter{}
}

func (r *RoundRobinRouter) SelectProvider(ctx context.Context, providers []string) (string, error) {
    if len(providers) == 0 {
        return "", fmt.Errorf("no providers available")
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    provider := providers[r.current]
    r.current = (r.current + 1) % len(providers)
    return provider, nil
}

// Cost-based routing
type CostBasedRouter struct {
    costs map[string]float64
}

func NewCostBasedRouter() *CostBasedRouter {
    return &CostBasedRouter{
        costs: map[string]float64{
            "openai":    0.03,  // $0.03 per 1K tokens
            "anthropic": 0.015, // $0.015 per 1K tokens
            "bedrock":   0.008, // $0.008 per 1K tokens
        },
    }
}

func (r *CostBasedRouter) SelectProvider(ctx context.Context, providers []string) (string, error) {
    if len(providers) == 0 {
        return "", fmt.Errorf("no providers available")
    }

    cheapest := providers[0]
    minCost := r.costs[cheapest]

    for _, p := range providers[1:] {
        if cost, ok := r.costs[p]; ok && cost < minCost {
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

func NewHealthBasedRouter(checker *HealthChecker) *HealthBasedRouter {
    return &HealthBasedRouter{healthChecker: checker}
}

func (r *HealthBasedRouter) SelectProvider(ctx context.Context, providers []string) (string, error) {
    healthy := []string{}
    for _, p := range providers {
        if r.healthChecker.IsHealthy(p) {
            healthy = append(healthy, p)
        }
    }

    if len(healthy) == 0 {
        return "", fmt.Errorf("no healthy providers")
    }

    return healthy[0], nil
}
```

## Rate Limiting

LLM providers enforce rate limits (requests per minute, tokens per minute) and return HTTP 429 errors when exceeded. Hitting these limits wastes a round-trip and delays the response. Client-side rate limiting using token bucket algorithms prevents requests from reaching the provider when the quota is near exhaustion, avoiding the latency penalty of server-side rejection.

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        limiters: map[string]*rate.Limiter{
            "openai":    rate.NewLimiter(rate.Every(time.Minute), 60),   // 60 req/min
            "anthropic": rate.NewLimiter(rate.Every(time.Minute), 50),   // 50 req/min
            "bedrock":   rate.NewLimiter(rate.Every(time.Minute), 100),  // 100 req/min
        },
    }
}

func (r *RateLimiter) Allow(provider string) bool {
    r.mu.RLock()
    limiter, ok := r.limiters[provider]
    r.mu.RUnlock()

    if !ok {
        return true // No limit configured
    }

    return limiter.Allow()
}
```

## Observability with OpenTelemetry

Gateway-level observability answers questions that per-provider metrics cannot: which provider is cheapest for a given workload, how does failover affect latency, and which routing strategy produces the best cost-quality tradeoff. The gateway records request counts, latency histograms, error rates, and token usage per provider using OTel GenAI semantic conventions, making all metrics comparable across providers in a single dashboard.

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type ObservableGateway struct {
    *LLMGateway
    meter  metric.Meter
    tracer trace.Tracer
}

func NewObservableGateway(ctx context.Context) (*ObservableGateway, error) {
    gateway, err := NewLLMGateway(ctx)
    if err != nil {
        return nil, err
    }

    return &ObservableGateway{
        LLMGateway: gateway,
        meter:      otel.Meter("llm-gateway"),
        tracer:     otel.Tracer("llm-gateway"),
    }, nil
}

func (g *ObservableGateway) Generate(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
    ctx, span := g.tracer.Start(ctx, "gateway.generate")
    defer span.End()

    providerName, err := g.router.SelectProvider(ctx, g.getAvailableProviders())
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.String("gen_ai.system", providerName),
        attribute.Int("gen_ai.message_count", len(msgs)),
    )

    start := time.Now()
    resp, err := g.LLMGateway.Generate(ctx, msgs)
    duration := time.Since(start)

    // Record metrics
    counter, _ := g.meter.Int64Counter("llm.requests.total")
    counter.Add(ctx, 1, metric.WithAttributes(
        attribute.String("provider", providerName),
        attribute.String("status", status(err)),
    ))

    histogram, _ := g.meter.Float64Histogram("llm.request.duration")
    histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(
        attribute.String("provider", providerName),
    ))

    if err != nil {
        span.RecordError(err)
        errorCounter, _ := g.meter.Int64Counter("llm.errors.total")
        errorCounter.Add(ctx, 1, metric.WithAttributes(
            attribute.String("provider", providerName),
        ))
        return nil, err
    }

    // Record token usage
    if resp.Usage != nil {
        tokenCounter, _ := g.meter.Int64Counter("llm.tokens.total")
        tokenCounter.Add(ctx, int64(resp.Usage.TotalTokens), metric.WithAttributes(
            attribute.String("provider", providerName),
        ))
    }

    span.SetAttributes(
        attribute.Int("gen_ai.usage.prompt_tokens", resp.Usage.PromptTokens),
        attribute.Int("gen_ai.usage.completion_tokens", resp.Usage.CompletionTokens),
    )

    return resp, nil
}

func status(err error) string {
    if err == nil {
        return "success"
    }
    return "error"
}
```

## Production Considerations

### Circuit Breaker

Implement circuit breaker pattern for failing providers:

```go
import "github.com/lookatitude/beluga-ai/resilience"

func (g *LLMGateway) GenerateWithCircuitBreaker(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
    cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
        FailureThreshold: 5,
        SuccessThreshold: 2,
        Timeout:          60 * time.Second,
    })

    return resilience.ExecuteWithCircuitBreaker(ctx, cb, func(ctx context.Context) (*schema.AIMessage, error) {
        return g.Generate(ctx, msgs)
    })
}
```

### Configuration

```yaml
gateway:
  routing_strategy: "health_based"  # round_robin, cost_based, health_based
  enable_failover: true
  max_retries: 3
  timeout: 60s

providers:
  openai:
    enabled: true
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o"
    rate_limit:
      requests_per_minute: 60
      tokens_per_minute: 90000

  anthropic:
    enabled: true
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
    rate_limit:
      requests_per_minute: 50

  bedrock:
    enabled: true
    region: "us-east-1"
    model: "anthropic.claude-v2"
    rate_limit:
      requests_per_minute: 100

observability:
  otel:
    endpoint: "localhost:4317"
  metrics:
    enabled: true
  tracing:
    enabled: true
    sample_rate: 1.0
```

### Scaling

- **Horizontal scaling**: Deploy multiple gateway instances behind a load balancer. Each instance is stateless.
- **Provider management**: Use health checks to automatically remove failing providers from rotation.
- **Caching**: Implement prompt caching for frequently asked questions to reduce API calls.
- **Cost tracking**: Monitor token usage per provider to optimize routing decisions.

## Related Resources

- [LLM Package Guide](/docs/guides/llm-integration/) for LLM integration patterns
- [Cost-Optimized Router](/docs/use-cases/cost-optimized-router/) for cost optimization strategies
- [Observability Guide](/docs/guides/observability/) for OpenTelemetry setup
