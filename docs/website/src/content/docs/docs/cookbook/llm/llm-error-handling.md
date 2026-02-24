---
title: "LLM Error Handling"
description: "Recipe for handling LLM rate limits, timeouts, and API errors in Go with retry logic, exponential backoff, and circuit breakers using Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, LLM error handling, Go retry logic, rate limit handling, exponential backoff, API error recovery, resilience patterns"
---

## Problem

You are calling an LLM API and need to handle rate limits, timeouts, and API errors gracefully without crashing your application or losing user requests.

## Solution

Implement a layered error handling strategy with retry logic, exponential backoff, and proper error classification. Transient errors are retried automatically while permanent errors fail fast with clear messages.

## Why This Matters

LLM APIs are external services with well-known failure modes: rate limits (HTTP 429), server errors (HTTP 5xx), and network timeouts. Each failure type requires a different response strategy. Retrying an authentication error (HTTP 401) wastes API calls and delays the real fix. Not retrying a rate limit error drops a request that would succeed after a brief wait.

Beluga AI's `core.IsRetryable()` function provides the first layer of error classification by inspecting the error's `ErrorCode`. This recipe adds a second layer by pattern-matching on error messages for cases where the error isn't wrapped in Beluga's error types (e.g., raw HTTP errors from provider SDKs). Exponential backoff with jitter prevents the thundering herd problem -- when many clients hit a rate limit simultaneously, jitter ensures they don't all retry at the same time and trigger another rate limit.

## Code Example

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// RetryConfig configures retry behavior for LLM calls.
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
	Jitter         float64
}

var DefaultRetryConfig = RetryConfig{
	MaxRetries:     3,
	InitialBackoff: 1 * time.Second,
	MaxBackoff:     30 * time.Second,
	BackoffFactor:  2.0,
	Jitter:         0.1,
}

// LLMClient wraps an LLM with retry and error handling.
type LLMClient struct {
	model  llm.ChatModel
	config RetryConfig
}

func NewLLMClient(model llm.ChatModel, config RetryConfig) *LLMClient {
	return &LLMClient{model: model, config: config}
}

// GenerateWithRetry calls the LLM with automatic retry on transient errors.
func (c *LLMClient) GenerateWithRetry(ctx context.Context, messages []schema.Message) (schema.Message, error) {
	var lastErr error
	backoff := c.config.InitialBackoff

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled before attempt %d: %w", attempt, ctx.Err())
		}

		response, err := c.model.Generate(ctx, messages)
		if err == nil {
			return response, nil
		}

		lastErr = err

		if !isRetryable(err) {
			return nil, fmt.Errorf("permanent error (not retrying): %w", err)
		}

		if attempt < c.config.MaxRetries {
			jitteredBackoff := c.calculateBackoff(backoff)
			log.Printf("LLM call failed (attempt %d/%d): %v. Retrying in %v",
				attempt+1, c.config.MaxRetries+1, err, jitteredBackoff)

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			case <-time.After(jitteredBackoff):
			}

			backoff = time.Duration(float64(backoff) * c.config.BackoffFactor)
			if backoff > c.config.MaxBackoff {
				backoff = c.config.MaxBackoff
			}
		}
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: %w", c.config.MaxRetries, lastErr)
}

func (c *LLMClient) calculateBackoff(base time.Duration) time.Duration {
	if c.config.Jitter == 0 {
		return base
	}
	jitter := float64(base) * c.config.Jitter * (rand.Float64()*2 - 1)
	return time.Duration(float64(base) + jitter)
}

func isRetryable(err error) bool {
	if core.IsRetryable(err) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"rate limit", "429", "too many requests",
		"temporarily unavailable", "service unavailable", "503",
		"connection reset", "connection refused", "timeout",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: "your-api-key",
	})
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	client := NewLLMClient(model, DefaultRetryConfig)

	messages := []schema.Message{
		schema.NewHumanMessage("Hello, how are you?"),
	}

	response, err := client.GenerateWithRetry(ctx, messages)
	if err != nil {
		log.Fatalf("LLM call failed: %v", err)
	}

	fmt.Printf("Response: %s\n", response.GetContent())
}
```

## Explanation

1. **Error classification** -- The `isRetryable` function checks Beluga AI's error types using `core.IsRetryable()` first, then falls back to pattern matching on error messages. Rate limits and server errors are retried; authentication errors and client errors are not. This two-layer approach handles both framework-wrapped errors and raw provider SDK errors.

2. **Exponential backoff with jitter** -- The backoff doubles each time (1s, 2s, 4s...) but never exceeds `MaxBackoff`. Random jitter (controlled by the `Jitter` config parameter) prevents multiple clients from retrying simultaneously, which would cause another rate limit spike. The jitter range is +-10% of the backoff by default.

3. **Context awareness** -- The function checks `ctx.Err()` before each attempt and during backoff waits using `select`. This respects timeouts and cancellations from the calling code, preventing wasted retries when the caller has already given up.

4. **MaxBackoff cap** -- Without a cap, exponential backoff grows unbounded (1s, 2s, 4s, 8s, 16s, 32s, 64s...). The `MaxBackoff` of 30 seconds ensures the wait never becomes unreasonable, even after many retries.

## Variations

### Circuit Breaker Pattern

If experiencing sustained failures, add a circuit breaker to stop retrying entirely and fail fast:

```go
type CircuitBreaker struct {
	failures    int
	lastFailure time.Time
	threshold   int
	resetAfter  time.Duration
}

func (cb *CircuitBreaker) Allow() bool {
	if cb.failures >= cb.threshold {
		if time.Since(cb.lastFailure) < cb.resetAfter {
			return false // Circuit is open
		}
		cb.failures = 0 // Reset after cooldown
	}
	return true
}
```

### Fallback Response

For non-critical features, provide a cached or template fallback when the LLM is unavailable:

```go
func (c *LLMClient) GenerateWithFallback(ctx context.Context, messages []schema.Message, fallback string) string {
	response, err := c.GenerateWithRetry(ctx, messages)
	if err != nil {
		log.Printf("LLM failed, using fallback: %v", err)
		return fallback
	}
	return response.GetContent()
}
```

## Related Recipes

- **[Custom Agent Patterns](/cookbook/agents/custom-agent-patterns)** -- Use error handling in custom agents
- **[Global Retry Wrappers](/cookbook/llm/global-retry)** -- Framework-level retry strategies
- **[Context Timeout Management](/cookbook/infrastructure/context-timeout)** -- Timeout handling for operations
