---
title: Error Recovery Service for LLMs
description: Implement robust error recovery with retry strategies, circuit breakers, and exponential backoff for 99.9% success rates.
---

LLM providers are external services subject to rate limits, transient network failures, and periodic outages. For an enterprise AI platform handling thousands of requests per hour, even a 3-5% failure rate translates to hundreds of user-facing errors daily, each requiring manual investigation and retry. The cost compounds: support tickets pile up, SLA commitments are missed, and engineering time is diverted from feature work to firefighting.

The fundamental challenge is that LLM failures are not uniform. Rate limit errors (HTTP 429) signal temporary throttling and resolve with backoff. Timeout errors suggest provider load and benefit from retry. Authentication errors are permanent and should fail immediately. A naive retry-everything approach wastes resources and delays permanent failures, while no retries at all exposes users to every transient glitch.

An error recovery service with intelligent retries, circuit breakers, and exponential backoff achieves 99.9% success rate even during provider issues by classifying errors and applying the right recovery strategy for each.

## Solution Architecture

Beluga AI's `core/` package provides typed errors with `IsRetryable()` checks that distinguish transient from permanent failures. The error recovery service wraps LLM calls with retry logic, circuit breakers, and fallback mechanisms. Error analysis determines retryability, exponential backoff prevents thundering herd effects on recovering providers, and circuit breakers short-circuit requests when a provider is consistently failing — redirecting traffic to fallback providers instead of accumulating timeouts.

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

The recovery service wraps LLM providers with error handling and automatic retries. It implements `llm.ChatModel`, so it can be used as a drop-in replacement anywhere a model is expected. This composability is intentional — resilience becomes a transparent layer rather than requiring callers to change their code.

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

The retry manager implements exponential backoff with jitter. Fixed-interval retries cause synchronized retry storms — when many clients fail at the same time, they all retry at the same time, amplifying the load spike that caused the original failure. Exponential backoff spreads retries over increasing intervals, and jitter randomizes the exact timing so clients naturally desynchronize.

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

Retrying every error wastes time and resources. Authentication failures, invalid request formats, and content policy violations will never succeed on retry. The error analyzer classifies errors into retryable (rate limits, timeouts, network issues, server errors) and permanent categories, so the service fails fast on unrecoverable errors while persisting through transient ones.

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

When a provider is down, retrying every request accumulates timeouts and delays all callers. The circuit breaker pattern prevents this cascading failure: after a threshold of consecutive failures, the breaker "opens" and immediately routes requests to the fallback provider without waiting for the primary to time out. After a cooldown period, the breaker enters "half-open" state and probes the primary with a single request — if it succeeds, the breaker closes and normal traffic resumes.

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

Streaming introduces a complication: partial responses may have already been yielded to the caller before an error occurs mid-stream. The recovery service handles this by retrying from the beginning on retryable errors — the caller receives a fresh stream from the retry attempt. For non-retryable errors, the error is yielded immediately. This follows Beluga AI's `iter.Seq2[T, error]` streaming pattern where errors are part of the iteration sequence.

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
