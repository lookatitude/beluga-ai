---
title: Token Cost Attribution per User
description: "Track and attribute LLM token costs to individual users and tenants with OpenTelemetry. Accurate billing and cost optimization."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "token cost attribution, LLM billing, per-user cost tracking, OpenTelemetry metrics, tenant billing, Beluga AI, Go, SaaS billing"
---

When LLM costs appear as a single line item on the infrastructure bill, teams cannot answer basic questions: which feature consumes the most tokens, which customers generate disproportionate cost, and where optimization efforts would have the most impact. Enterprise customers expect usage-based billing with transparent cost breakdowns, but system-level token counters provide no attribution.

The deeper problem is that LLM costs are non-obvious — a chatbot conversation might use 500 tokens or 50,000 depending on conversation length, tool calls, and context window management. Without per-user attribution, billing disputes are unresolvable and cost optimization is guesswork.

User-level cost attribution using OpenTelemetry metrics solves this by attaching user and tenant context to every token counter increment. Costs are attributed at the point of consumption, not estimated after the fact.

## Solution Architecture

Beluga AI's `o11y/` package integrates with OpenTelemetry for standardized metrics export. The key design choice is implementing cost tracking as LLM middleware (`func(ChatModel) ChatModel`) rather than application-level instrumentation. This middleware pattern means cost tracking is transparent to callers — wrap the model once, and every Generate/Stream call is automatically tracked with user context from `context.Context`. Metrics flow to Prometheus for aggregation and PromQL queries power billing dashboards.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ LLM Request  │───▶│ Token        │───▶│ Cost         │
│ (with user   │    │ Counter      │    │ Calculator   │
│  context)    │    └──────────────┘    └──────┬───────┘
└──────────────┘                                │
                                                ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Billing      │◀───│ Cost         │◀───│ OTEL Metrics │
│ System       │    │ Database     │    │ (per user)   │
└──────────────┘    └──────────────┘    └──────────────┘
```

## Token Cost Tracker Implementation

Implement per-user token tracking with provider-specific pricing:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

// TokenCostTracker tracks token usage and costs per user
type TokenCostTracker struct {
    inputTokensCounter  metric.Int64Counter
    outputTokensCounter metric.Int64Counter
    costCounter         metric.Float64Counter
    providerPricing     map[string]*ProviderPricing
    tracer              trace.Tracer
}

// ProviderPricing defines pricing for a provider model
type ProviderPricing struct {
    InputCostPer1KTokens  float64
    OutputCostPer1KTokens float64
}

// NewTokenCostTracker creates a new token cost tracker
func NewTokenCostTracker(ctx context.Context) (*TokenCostTracker, error) {
    meter := otel.GetMeterProvider().Meter("token-cost-tracker")

    inputTokensCounter, err := meter.Int64Counter(
        "llm_tokens_input_total",
        metric.WithDescription("Total input tokens consumed"),
        metric.WithUnit("1"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create input tokens counter: %w", err)
    }

    outputTokensCounter, err := meter.Int64Counter(
        "llm_tokens_output_total",
        metric.WithDescription("Total output tokens generated"),
        metric.WithUnit("1"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create output tokens counter: %w", err)
    }

    costCounter, err := meter.Float64Counter(
        "llm_cost_total",
        metric.WithDescription("Total LLM cost in USD"),
        metric.WithUnit("USD"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create cost counter: %w", err)
    }

    return &TokenCostTracker{
        inputTokensCounter:  inputTokensCounter,
        outputTokensCounter: outputTokensCounter,
        costCounter:         costCounter,
        providerPricing:     loadProviderPricing(),
        tracer:              otel.Tracer("token-cost-tracker"),
    }, nil
}

func loadProviderPricing() map[string]*ProviderPricing {
    return map[string]*ProviderPricing{
        "openai:gpt-4": {
            InputCostPer1KTokens:  0.03,
            OutputCostPer1KTokens: 0.06,
        },
        "openai:gpt-3.5-turbo": {
            InputCostPer1KTokens:  0.0015,
            OutputCostPer1KTokens: 0.002,
        },
        "anthropic:claude-3-opus": {
            InputCostPer1KTokens:  0.015,
            OutputCostPer1KTokens: 0.075,
        },
    }
}
```

## Cost Tracking with User Context

Track costs with user and tenant attributes for accurate attribution:

```go
// TrackCost tracks token usage and calculates cost for a request
func (t *TokenCostTracker) TrackCost(
    ctx context.Context,
    userID string,
    tenantID string,
    provider string,
    model string,
    inputTokens int,
    outputTokens int,
) error {
    ctx, span := t.tracer.Start(ctx, "token_cost.track")
    defer span.End()

    span.SetAttributes(
        attribute.String("user_id", userID),
        attribute.String("tenant_id", tenantID),
        attribute.String("provider", provider),
        attribute.String("model", model),
        attribute.Int("input_tokens", inputTokens),
        attribute.Int("output_tokens", outputTokens),
    )

    // Get provider pricing
    pricingKey := fmt.Sprintf("%s:%s", provider, model)
    pricing, exists := t.providerPricing[pricingKey]
    if !exists {
        return fmt.Errorf("pricing not found for %s", pricingKey)
    }

    // Calculate costs
    inputCost := float64(inputTokens) / 1000.0 * pricing.InputCostPer1KTokens
    outputCost := float64(outputTokens) / 1000.0 * pricing.OutputCostPer1KTokens
    totalCost := inputCost + outputCost

    // Record metrics with user/tenant attributes
    attrs := metric.WithAttributes(
        attribute.String("user_id", userID),
        attribute.String("tenant_id", tenantID),
        attribute.String("provider", provider),
        attribute.String("model", model),
    )

    t.inputTokensCounter.Add(ctx, int64(inputTokens), attrs)
    t.outputTokensCounter.Add(ctx, int64(outputTokens), attrs)
    t.costCounter.Add(ctx, totalCost, attrs)

    span.SetAttributes(
        attribute.Float64("input_cost_usd", inputCost),
        attribute.Float64("output_cost_usd", outputCost),
        attribute.Float64("total_cost_usd", totalCost),
    )

    return nil
}
```

## Integration with LLM Calls

The cost tracker integrates as Beluga AI middleware — the standard `func(ChatModel) ChatModel` pattern. This is the recommended approach because it's composable (stack it with other middleware like caching and resilience), transparent (callers don't know tracking is happening), and complete (every LLM call is tracked, not just the ones where developers remembered to add instrumentation):

```go
package main

import (
    "context"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// CostTrackingMiddleware wraps an LLM to track costs
func CostTrackingMiddleware(tracker *TokenCostTracker) llm.Middleware {
    return func(next llm.ChatModel) llm.ChatModel {
        return &costTrackingModel{
            inner:   next,
            tracker: tracker,
        }
    }
}

type costTrackingModel struct {
    inner   llm.ChatModel
    tracker *TokenCostTracker
}

func (m *costTrackingModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    // Extract user context
    userID := getUserID(ctx)
    tenantID := getTenantID(ctx)

    // Call inner model
    resp, err := m.inner.Generate(ctx, msgs, opts...)
    if err != nil {
        return nil, err
    }

    // Track cost
    if err := m.tracker.TrackCost(
        ctx,
        userID,
        tenantID,
        "openai",
        m.inner.ModelID(),
        resp.Usage.InputTokens,
        resp.Usage.OutputTokens,
    ); err != nil {
        // Log error but don't fail the request
        logger.Error("failed to track cost", "error", err)
    }

    return resp, nil
}

func (m *costTrackingModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    return m.inner.Stream(ctx, msgs, opts...)
}

func (m *costTrackingModel) BindTools(tools []tool.Tool) llm.ChatModel {
    return &costTrackingModel{
        inner:   m.inner.BindTools(tools),
        tracker: m.tracker,
    }
}

func (m *costTrackingModel) ModelID() string {
    return m.inner.ModelID()
}
```

## Cost Aggregation and Reporting

Query aggregated costs from Prometheus for billing:

```promql
# Daily cost per user
sum(increase(llm_cost_total[24h])) by (user_id)

# Cost by tenant and model
sum(increase(llm_cost_total[24h])) by (tenant_id, model)

# Top 10 users by cost this month
topk(10, sum(increase(llm_cost_total[30d])) by (user_id))

# Cost trend over time
sum(rate(llm_cost_total[1h])) by (user_id)
```

## Production Considerations

### High-Cardinality Mitigation

Limit cardinality for user attributes to prevent metric explosion:

```go
// Map user IDs to tiers instead of tracking individual IDs
func getUserTier(userID string) string {
    user := lookupUser(userID)
    return user.Tier  // "free", "pro", "enterprise"
}

// Track by tier, not individual user
attrs := metric.WithAttributes(
    attribute.String("tier", getUserTier(userID)),
    attribute.String("tenant_id", tenantID),
    attribute.String("model", model),
)
```

For detailed per-user attribution, export raw events to a time-series database and aggregate there.

### Dynamic Pricing Updates

Load pricing from configuration to handle price changes without code deployment:

```go
func (t *TokenCostTracker) UpdatePricing(pricingConfig map[string]*ProviderPricing) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.providerPricing = pricingConfig
}

// Load pricing from config file or API
func loadProviderPricing() map[string]*ProviderPricing {
    var pricing map[string]*ProviderPricing
    if err := config.Load("pricing.yaml", &pricing); err != nil {
        log.Fatal("failed to load pricing", err)
    }
    return pricing
}
```

### Billing Integration

Export cost data to billing systems:

```go
type CostExporter struct {
    billingAPI BillingAPI
}

func (e *CostExporter) ExportDailyCosts(ctx context.Context, date time.Time) error {
    // Query Prometheus for daily costs
    costs, err := e.queryCosts(ctx, date)
    if err != nil {
        return err
    }

    // Send to billing API
    for userID, cost := range costs {
        if err := e.billingAPI.RecordUsage(ctx, userID, date, cost); err != nil {
            return fmt.Errorf("failed to export cost for user %s: %w", userID, err)
        }
    }

    return nil
}
```

## Related Resources

- [Monitoring Dashboards](/use-cases/monitoring-dashboards/) for cost visualization
- [PII Leakage Detection](/use-cases/pii-leakage-detection/) for privacy monitoring
- [Observability Guide](/guides/observability/) for OpenTelemetry setup
