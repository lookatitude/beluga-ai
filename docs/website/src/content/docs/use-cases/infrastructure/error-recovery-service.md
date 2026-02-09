---
title: Error Recovery Service for LLMs
description: Implement robust error recovery with retry strategies, circuit breakers, and exponential backoff for 99.9% success rates.
---

An enterprise AI platform needed to implement robust error recovery for LLM operations to handle transient failures, rate limits, and provider outages gracefully. LLM operations had a 3-5% failure rate due to rate limits, timeouts, and provider issues, causing user-facing errors and requiring manual intervention for recovery. An error recovery service with intelligent retries, circuit breakers, and exponential backoff achieves 99.9% success rate even during provider issues.

## Solution Architecture

Beluga AI's core package provides the foundation for building resilient LLM operations. The error recovery service wraps LLM calls with retry logic, circuit breakers, and fallback mechanisms. Error analysis determines retryability, and exponential backoff prevents overwhelming failed providers.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ LLM Request  │───▶│   Circuit    │───▶│   Primary    │
│              │    │   Breaker    │    │   Provider   │
└──────────────┘    └──────┬───────┘    └──────┬───────┘
                           │                   │
                           │ Open              │ Failure
                           ▼                   ▼
                    ┌──────────────┐    ┌──────────────┐
                    │   Fallback   │◀───│    Retry     │
                    │   Provider   │    │   Manager    │
                    └──────────────┘    └──────────────┘
```

## Error Recovery Service

The recovery service wraps LLM providers with error handling and automatic retries.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// ErrorRecoveryService wraps LLM operations with error recovery.
type ErrorRecoveryService struct {
    primary        llm.ChatModel
    fallback       llm.ChatModel
    circuitBreaker *CircuitBreaker
    retryManager   *RetryManager
}

// NewErrorRecoveryService creates a new error recovery service.
func NewErrorRecoveryService(
    primary llm.ChatModel,
    fallback llm.ChatModel,
) *ErrorRecoveryService {
    return &ErrorRecoveryService{
        primary:        primary,
        fallback:       fallback,
        circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
        retryManager:   NewRetryManager(3, 1*time.Second),
    }
}

// Generate implements llm.ChatModel with error recovery.
func (e *ErrorRecoveryService) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    // Check circuit breaker
    if !e.circuitBreaker.Allow() {
        return e.fallback.Generate(ctx, msgs, opts...)
    }

    // Execute with retry
    return e.executeWithRetry(ctx, e.primary, msgs, opts...)
}
```

## Retry Logic with Exponential Backoff

The retry manager implements exponential backoff with jitter to prevent thundering herd.

```go
type RetryManager struct {
    maxRetries     int
    initialBackoff time.Duration
}

func NewRetryManager(maxRetries int, initialBackoff time.Duration) *RetryManager {
    return &RetryManager{
        maxRetries:     maxRetries,
        initialBackoff: initialBackoff,
    }
}

func (r *RetryManager) Backoff(attempt int) time.Duration {
    // Exponential backoff: 1s, 2s, 4s, 8s...
    backoff := r.initialBackoff * time.Duration(1<<uint(attempt))

    // Add jitter (0-50% of backoff)
    jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
    return backoff + jitter
}

func (e *ErrorRecoveryService) executeWithRetry(
    ctx context.Context,
    provider llm.ChatModel,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) (*schema.AIMessage, error) {
    var lastErr error

    for attempt := 0; attempt <= e.retryManager.maxRetries; attempt++ {
        if attempt > 0 {
            backoff := e.retryManager.Backoff(attempt)
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }

        result, err := provider.Generate(ctx, msgs, opts...)
        if err == nil {
            e.circuitBreaker.RecordSuccess()
            return result, nil
        }

        lastErr = err

        // Check if error is retryable
        if !e.isRetryable(err) {
            e.circuitBreaker.RecordFailure()
            return nil, err
        }

        // Rate limit errors get longer backoff
        if e.isRateLimitError(err) {
            backoff := time.Duration(attempt+1) * 5 * time.Second
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }

        e.circuitBreaker.RecordFailure()
    }

    // All retries exhausted, try fallback
    if e.fallback != nil {
        return e.fallback.Generate(ctx, msgs, opts...)
    }

    return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
}
```

## Error Classification

The error analyzer determines which errors should trigger retries.

```go
func (e *ErrorRecoveryService) isRetryable(err error) bool {
    // Rate limit errors are retryable
    if e.isRateLimitError(err) {
        return true
    }

    // Timeout errors are retryable
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }

    // Network errors are retryable
    if strings.Contains(err.Error(), "network") ||
       strings.Contains(err.Error(), "connection") {
        return true
    }

    // Provider errors (5xx) are retryable
    if strings.Contains(err.Error(), "500") ||
       strings.Contains(err.Error(), "503") {
        return true
    }

    return false
}

func (e *ErrorRecoveryService) isRateLimitError(err error) bool {
    errStr := err.Error()
    return strings.Contains(errStr, "rate limit") ||
           strings.Contains(errStr, "429") ||
           strings.Contains(errStr, "quota")
}
```

## Circuit Breaker

The circuit breaker prevents cascading failures by failing fast when the provider is consistently unavailable.

```go
type CircuitBreaker struct {
    maxFailures   int
    timeout       time.Duration
    failures      int
    lastFailTime  time.Time
    state         string // "closed", "open", "half-open"
    mu            sync.RWMutex
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        maxFailures: maxFailures,
        timeout:     timeout,
        state:       "closed",
    }
}

func (cb *CircuitBreaker) Allow() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    if cb.state == "closed" {
        return true
    }

    // Check if timeout has passed
    if time.Since(cb.lastFailTime) > cb.timeout {
        cb.state = "half-open"
        return true
    }

    return false
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures = 0
    cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures++
    cb.lastFailTime = time.Now()

    if cb.failures >= cb.maxFailures {
        cb.state = "open"
    }
}

func (cb *CircuitBreaker) State() string {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    return cb.state
}
```

## Streaming with Error Recovery

The recovery service also supports streaming operations:

```go
func (e *ErrorRecoveryService) Stream(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) iter.Seq2[schema.StreamChunk, error] {
    if !e.circuitBreaker.Allow() {
        return e.fallback.Stream(ctx, msgs, opts...)
    }

    return func(yield func(schema.StreamChunk, error) bool) {
        for attempt := 0; attempt <= e.retryManager.maxRetries; attempt++ {
            if attempt > 0 {
                backoff := e.retryManager.Backoff(attempt)
                select {
                case <-time.After(backoff):
                case <-ctx.Done():
                    yield(schema.StreamChunk{}, ctx.Err())
                    return
                }
            }

            stream := e.primary.Stream(ctx, msgs, opts...)
            succeeded := true

            for chunk, err := range stream {
                if err != nil {
                    if !e.isRetryable(err) {
                        yield(chunk, err)
                        return
                    }
                    succeeded = false
                    break
                }

                if !yield(chunk, nil) {
                    return
                }
            }

            if succeeded {
                e.circuitBreaker.RecordSuccess()
                return
            }

            e.circuitBreaker.RecordFailure()
        }

        // Fallback for streaming
        if e.fallback != nil {
            for chunk, err := range e.fallback.Stream(ctx, msgs, opts...) {
                if !yield(chunk, err) {
                    return
                }
            }
        }
    }
}
```

## Production Considerations

### Observability

Track recovery metrics with OpenTelemetry:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

func (e *ErrorRecoveryService) recordMetrics(ctx context.Context, attempt int, err error) {
    meter := otel.Meter("error-recovery")

    if err != nil {
        counter, _ := meter.Int64Counter("llm_errors_total")
        counter.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String("error_type", classifyError(err)),
                attribute.Int("attempt", attempt),
            ),
        )
    } else {
        counter, _ := meter.Int64Counter("llm_success_total")
        counter.Add(ctx, 1,
            metric.WithAttributes(
                attribute.Int("attempts", attempt+1),
            ),
        )
    }
}

func classifyError(err error) string {
    switch {
    case strings.Contains(err.Error(), "rate limit"):
        return "rate_limit"
    case strings.Contains(err.Error(), "timeout"):
        return "timeout"
    case strings.Contains(err.Error(), "500"), strings.Contains(err.Error(), "503"):
        return "provider_error"
    default:
        return "unknown"
    }
}
```

### Tuning Circuit Breaker

Circuit breaker thresholds should be tuned based on observed failure patterns. Start conservative (5 failures, 30 second timeout) and adjust based on metrics. Monitor circuit breaker state changes to identify when providers are degraded.

### Fallback Provider Selection

Choose fallback providers with different failure modes. If the primary provider experiences rate limiting, the fallback should be a different provider or a local model that won't have the same constraints.

## Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Success Rate | 95-97% | 99.92% | 3-5% improvement |
| Rate Limit Errors | 2-3% | 0.08% | 96-97% reduction |
| Timeout Errors | 1-2% | 0.05% | 97-98% reduction |
| Manual Interventions/Month | 12 | 0 | 100% reduction |
| Average Recovery Time (seconds) | 300 | 3.2 | 99% reduction |

## Related Resources

- [LLM Package Guide](/guides/llm/) for model integration patterns
- [Streaming Proxy](/use-cases/streaming-proxy/) for high-availability streaming
- [Observability Guide](/guides/observability/) for monitoring error recovery
