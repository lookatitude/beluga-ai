---
title: "Token Counting without Latency"
description: "Recipe for counting LLM tokens in Go using caching and async estimation to track costs and enforce limits without adding request latency."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, token counting, Go LLM cost tracking, rate limiting, async token estimation, token cache, performance recipe"
---

## Problem

You need to count tokens in LLM requests for cost tracking and rate limiting, but token counting libraries can be slow and add significant latency to your requests, especially for large inputs or high-throughput scenarios.

## Solution

Implement asynchronous token counting with caching and estimation for non-critical paths. This works because you can count tokens in parallel with the LLM request, cache counts for repeated inputs, and use fast estimation algorithms when exact counts aren't needed.

## Why This Matters

Token counting is a necessary evil in LLM applications. You need it for cost tracking, rate limiting, context window management, and billing, but accurate token counting requires running the model's tokenizer, which can add 5-50ms of latency per request depending on input size. At 1000 requests per second, that overhead becomes significant.

The solution is to recognize that not all token counts need to be exact at the same time. Rate limiting needs a fast estimate immediately (is this request roughly within budget?), while billing needs exact counts eventually (how many tokens did we actually use?). By separating these concerns, you can use a character-ratio estimator (approximately 4 characters per token for English text) for immediate decisions and compute exact counts asynchronously in the background.

The caching layer addresses a pattern common in production: system prompts, few-shot examples, and template prefixes repeat across requests. Caching their token counts eliminates redundant computation for the most common inputs. The `sync.RWMutex` protects the cache for concurrent access, using a read lock for cache hits (the hot path) and a write lock only when inserting new entries. The OTel spans distinguish between cached, estimated, and exact counts, which is valuable for monitoring cache hit rates and tuning the estimation ratio for your specific model and language distribution.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.llms.token_counting")

// TokenCounter provides fast token counting
type TokenCounter struct {
    cache        map[string]int
    mu           sync.RWMutex
    estimator    TokenEstimator
    exactCounter TokenCounterFunc
}

// TokenCounterFunc counts tokens exactly (may be slow)
type TokenCounterFunc func(text string) (int, error)

// TokenEstimator estimates tokens quickly (approximate)
type TokenEstimator func(text string) int

// NewTokenCounter creates a new token counter
func NewTokenCounter(exactCounter TokenCounterFunc, estimator TokenEstimator) *TokenCounter {
    return &TokenCounter{
        cache:        make(map[string]int),
        estimator:    estimator,
        exactCounter: exactCounter,
    }
}

// CountTokens counts tokens with caching and async fallback
func (tc *TokenCounter) CountTokens(ctx context.Context, text string, requireExact bool) (int, error) {
    ctx, span := tracer.Start(ctx, "token_counter.count")
    defer span.End()

    // Check cache first
    tc.mu.RLock()
    if count, exists := tc.cache[text]; exists {
        tc.mu.RUnlock()
        span.SetAttributes(
            attribute.Bool("token_count.cached", true),
            attribute.Int("token_count.value", count),
        )
        return count, nil
    }
    tc.mu.RUnlock()

    // If exact count not required, use fast estimation
    if !requireExact {
        estimated := tc.estimator(text)
        span.SetAttributes(
            attribute.Bool("token_count.estimated", true),
            attribute.Int("token_count.value", estimated),
        )

        // Cache estimation asynchronously
        go tc.cacheExactCount(text)

        return estimated, nil
    }

    // Count exactly (may be slow)
    count, err := tc.exactCounter(text)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return 0, err
    }

    // Cache result
    tc.mu.Lock()
    tc.cache[text] = count
    tc.mu.Unlock()

    span.SetAttributes(
        attribute.Bool("token_count.exact", true),
        attribute.Int("token_count.value", count),
    )

    return count, nil
}

// cacheExactCount caches exact count asynchronously
func (tc *TokenCounter) cacheExactCount(text string) {
    count, err := tc.exactCounter(text)
    if err != nil {
        return
    }

    tc.mu.Lock()
    tc.cache[text] = count
    tc.mu.Unlock()
}

// CountTokensAsync counts tokens asynchronously
func (tc *TokenCounter) CountTokensAsync(ctx context.Context, text string) <-chan TokenCountResult {
    resultCh := make(chan TokenCountResult, 1)

    go func() {
        defer close(resultCh)

        // Use estimation for immediate response
        estimated := tc.estimator(text)
        resultCh <- TokenCountResult{
            Count:     estimated,
            Estimated: true,
        }

        // Count exactly in background
        exact, err := tc.exactCounter(text)
        if err == nil {
            tc.mu.Lock()
            tc.cache[text] = exact
            tc.mu.Unlock()

            resultCh <- TokenCountResult{
                Count:     exact,
                Estimated: false,
            }
        }
    }()

    return resultCh
}

// TokenCountResult represents token count result
type TokenCountResult struct {
    Count     int
    Estimated bool
}

// FastEstimator estimates tokens using character count
func FastEstimator(model string) TokenEstimator {
    ratios := map[string]float64{
        "gpt-3.5-turbo": 0.25,
        "gpt-4":         0.25,
        "claude-3":      0.25,
    }

    ratio := ratios[model]
    if ratio == 0 {
        ratio = 0.25 // Default
    }

    return func(text string) int {
        return int(float64(len(text)) * ratio)
    }
}

// CountTokensForMessages counts tokens for message arrays
func (tc *TokenCounter) CountTokensForMessages(ctx context.Context, messages []schema.Message, requireExact bool) (int, error) {
    ctx, span := tracer.Start(ctx, "token_counter.count_messages")
    defer span.End()

    total := 0

    for i, msg := range messages {
        content := msg.GetContent()
        count, err := tc.CountTokens(ctx, content, requireExact)
        if err != nil {
            span.RecordError(err)
            return 0, fmt.Errorf("failed to count tokens for message %d: %w", i, err)
        }
        total += count
    }

    span.SetAttributes(
        attribute.Int("token_count.total", total),
        attribute.Int("message_count", len(messages)),
    )

    return total, nil
}

func main() {
    ctx := context.Background()

    exactCounter := func(text string) (int, error) {
        return len(text) / 4, nil // Simplified
    }

    estimator := FastEstimator("gpt-3.5-turbo")
    counter := NewTokenCounter(exactCounter, estimator)

    text := "This is a test message for token counting"
    count, err := counter.CountTokens(ctx, text, false)
    if err != nil {
        log.Fatalf("Failed to count tokens: %v", err)
    }
    fmt.Printf("Token count: %d\n", count)
}
```

## Explanation

1. **Cache-first lookup** -- Token counts are cached by text content using a `sync.RWMutex`-protected map. The read lock is acquired first for cache lookups (the common case), avoiding write lock contention. Repeated inputs like system prompts, few-shot examples, and template prefixes are counted once and served from cache thereafter, eliminating the most common source of redundant computation.

2. **Estimation with async backfill** -- When exact counts aren't required (e.g., for rate limiting or pre-flight checks), the `FastEstimator` returns immediately using a character-to-token ratio. Meanwhile, a background goroutine computes the exact count and caches it for future use. This means the first request uses an estimate, but all subsequent requests for the same text get an exact cached count without any computation.

3. **Async counting channel** -- The `CountTokensAsync` method returns a channel that delivers two results: an immediate estimate followed by the exact count. Consumers can use the estimate for fast decisions (like rate limiting) and the exact count for accurate operations (like billing). The channel is buffered with capacity 1 to prevent the goroutine from blocking if the consumer only reads one result.

4. **Model-specific estimation** -- The `FastEstimator` function accepts a model name and returns a tuned ratio. Different models tokenize differently (GPT-4 and Claude use roughly 4 characters per token for English, but this varies by language), so the ratio can be customized per model for better estimation accuracy.

## Testing

```go
func TestTokenCounter_Caching(t *testing.T) {
    callCount := 0
    exactCounter := func(text string) (int, error) {
        callCount++
        return len(text) / 4, nil
    }

    estimator := FastEstimator("gpt-3.5-turbo")
    counter := NewTokenCounter(exactCounter, estimator)

    text := "test message"

    // First call - should call exact counter
    count1, _ := counter.CountTokens(context.Background(), text, true)

    // Second call - should use cache
    count2, _ := counter.CountTokens(context.Background(), text, true)

    require.Equal(t, count1, count2)
    require.Equal(t, 1, callCount) // Should only call once
}
```

## Variations

### Model-Specific Counting

Use different counting strategies per model:

```go
type ModelTokenCounter struct {
    counters map[string]*TokenCounter
}
```

### Batch Counting

Count tokens for multiple texts in parallel:

```go
func (tc *TokenCounter) CountTokensBatch(ctx context.Context, texts []string) ([]int, error) {
    // Count in parallel
}
```

## Related Recipes

- **[Streaming Tool Calls](./streaming-tool-calls)** — Handle streaming with tool calls
- **[S2S Voice Metrics](./s2s-voice-metrics)** — Track token usage metrics
