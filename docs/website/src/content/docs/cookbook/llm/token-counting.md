---
title: "Token Counting without Performance Hit"
description: "Count tokens in LLM requests using caching and async estimation to avoid adding latency to your requests."
---

## Problem

You need to count tokens in LLM requests for cost tracking and rate limiting, but token counting libraries can be slow and add significant latency to your requests, especially for large inputs or high-throughput scenarios.

## Solution

Implement asynchronous token counting with caching and estimation for non-critical paths. This works because you can count tokens in parallel with the LLM request, cache counts for repeated inputs, and use fast estimation algorithms when exact counts aren't needed.

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

1. **Caching** — Token counts are cached by text content. Repeated inputs (like common prompts) are counted once and cached, eliminating redundant counting. This is important because many applications reuse the same prompts or system messages.

2. **Estimation fallback** — When exact counts aren't required (e.g., for rate limiting), fast estimation is used. The exact count is computed asynchronously and cached for future use. This gives immediate results without blocking.

3. **Async counting** — For non-blocking scenarios, async counting returns an estimated count immediately and the exact count later. This allows your application to proceed while accurate counts are computed in the background.

Use estimation for rate limiting and caching for exact counts. Most use cases don't need exact counts immediately — they can be computed asynchronously and used for billing or analytics later.

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
