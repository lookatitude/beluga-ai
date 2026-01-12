# LLM Error Handling

## Problem

You're calling an LLM API and need to handle rate limits, timeouts, and API errors gracefully without crashing your application or losing user requests.

## Solution

Implement a layered error handling strategy with retry logic, exponential backoff, and proper error classification. This approach ensures transient errors are retried while permanent errors fail fast with clear messages.

## Code Example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "math/rand"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// RetryConfig configures retry behavior for LLM calls
type RetryConfig struct {
    MaxRetries     int           // Maximum number of retry attempts
    InitialBackoff time.Duration // Starting backoff duration
    MaxBackoff     time.Duration // Maximum backoff duration
    BackoffFactor  float64       // Multiplier for each retry (e.g., 2.0 for doubling)
    Jitter         float64       // Random jitter factor (0.0 to 1.0)
}

// DefaultRetryConfig provides sensible defaults for most use cases
var DefaultRetryConfig = RetryConfig{
    MaxRetries:     3,
    InitialBackoff: 1 * time.Second,
    MaxBackoff:     30 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         0.1,
}

// LLMClient wraps an LLM with retry and error handling
type LLMClient struct {
    model  iface.ChatModel
    config RetryConfig
}

// NewLLMClient creates a new client with retry support
func NewLLMClient(model iface.ChatModel, config RetryConfig) *LLMClient {
    return &LLMClient{model: model, config: config}
}

// GenerateWithRetry calls the LLM with automatic retry on transient errors
func (c *LLMClient) GenerateWithRetry(ctx context.Context, messages []schema.Message) (schema.Message, error) {
    var lastErr error
    backoff := c.config.InitialBackoff

    for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
        // Check context before attempting
        if ctx.Err() != nil {
            return nil, fmt.Errorf("context cancelled before attempt %d: %w", attempt, ctx.Err())
        }

        // Make the LLM call
        response, err := c.model.Generate(ctx, messages)
        if err == nil {
            return response, nil
        }

        lastErr = err

        // Classify the error
        if !isRetryable(err) {
            // Permanent error - fail immediately
            return nil, fmt.Errorf("permanent error (not retrying): %w", err)
        }

        // Don't sleep after the last attempt
        if attempt < c.config.MaxRetries {
            // Calculate backoff with jitter
            jitteredBackoff := c.calculateBackoff(backoff)
            
            log.Printf("LLM call failed (attempt %d/%d): %v. Retrying in %v",
                attempt+1, c.config.MaxRetries+1, err, jitteredBackoff)

            select {
            case <-ctx.Done():
                return nil, fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
            case <-time.After(jitteredBackoff):
            }

            // Increase backoff for next attempt
            backoff = time.Duration(float64(backoff) * c.config.BackoffFactor)
            if backoff > c.config.MaxBackoff {
                backoff = c.config.MaxBackoff
            }
        }
    }

    return nil, fmt.Errorf("max retries (%d) exceeded: %w", c.config.MaxRetries, lastErr)
}

// calculateBackoff adds jitter to prevent thundering herd
func (c *LLMClient) calculateBackoff(base time.Duration) time.Duration {
    if c.config.Jitter == 0 {
        return base
    }
    jitter := float64(base) * c.config.Jitter * (rand.Float64()*2 - 1)
    return time.Duration(float64(base) + jitter)
}

// isRetryable determines if an error should be retried
func isRetryable(err error) bool {
    // Check for Beluga AI error types
    var llmErr *llms.LLMError
    if errors.As(err, &llmErr) {
        switch llmErr.Code {
        case llms.ErrCodeRateLimit:
            return true // Rate limits are always retryable
        case llms.ErrCodeTimeout:
            return true // Timeouts can be retried
        case llms.ErrCodeServerError:
            return true // Server errors (5xx) are usually transient
        case llms.ErrCodeInvalidRequest:
            return false // Client errors (4xx) won't succeed on retry
        case llms.ErrCodeAuthentication:
            return false // Auth errors need manual intervention
        default:
            return false
        }
    }

    // Check for context errors
    if errors.Is(err, context.DeadlineExceeded) {
        return false // Don't retry if our own deadline passed
    }
    if errors.Is(err, context.Canceled) {
        return false // Don't retry if explicitly cancelled
    }

    // Check error message for common patterns
    errMsg := err.Error()
    retryablePatterns := []string{
        "rate limit",
        "429",
        "too many requests",
        "temporarily unavailable",
        "service unavailable",
        "503",
        "connection reset",
        "connection refused",
        "timeout",
    }
    
    for _, pattern := range retryablePatterns {
        if contains(errMsg, pattern) {
            return true
        }
    }

    return false
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || 
        len(s) > len(substr) && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
    // Simple case-insensitive contains check
    for i := 0; i <= len(s)-len(substr); i++ {
        if equalIgnoreCase(s[i:i+len(substr)], substr) {
            return true
        }
    }
    return false
}

func equalIgnoreCase(a, b string) bool {
    if len(a) != len(b) {
        return false
    }
    for i := 0; i < len(a); i++ {
        ca, cb := a[i], b[i]
        if ca >= 'A' && ca <= 'Z' {
            ca += 'a' - 'A'
        }
        if cb >= 'A' && cb <= 'Z' {
            cb += 'a' - 'A'
        }
        if ca != cb {
            return false
        }
    }
    return true
}

func main() {
    ctx := context.Background()

    // Create base LLM client
    model, err := llms.NewOpenAIChat(
        llms.WithAPIKey("your-api-key"),
    )
    if err != nil {
        log.Fatalf("Failed to create LLM: %v", err)
    }

    // Wrap with retry logic
    client := NewLLMClient(model, DefaultRetryConfig)

    // Make a call with automatic retry
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

Let's break down what's happening:

1. **RetryConfig structure** - Notice how we separate configuration from logic. This makes the retry behavior customizable per use case. Production systems might want longer backoffs; tests might want none.

2. **Error classification with isRetryable** - We check for specific error codes from Beluga AI's LLM package. Rate limits (429) and server errors (5xx) are retried, while authentication errors are not. Without this, you'd waste time retrying errors that will never succeed.

3. **Exponential backoff with jitter** - The backoff doubles each time (1s → 2s → 4s) but never exceeds MaxBackoff. The jitter prevents multiple clients from retrying simultaneously, which would cause another rate limit.

4. **Context awareness** - We check `ctx.Err()` before each attempt and during backoff. This ensures we respect timeouts and cancellations from the calling code.

**Key insight:** Always classify errors before retrying. Retrying authentication errors just wastes API calls and delays the real fix.

## Testing

Here's how to test the retry logic:

```go
func TestGenerateWithRetry_RetriesOnRateLimit(t *testing.T) {
    attempts := 0
    mockModel := &MockChatModel{
        generateFunc: func(ctx context.Context, msgs []schema.Message) (schema.Message, error) {
            attempts++
            if attempts < 3 {
                return nil, llms.NewLLMError("Generate", llms.ErrCodeRateLimit, errors.New("rate limited"))
            }
            return schema.NewAIMessage("Success!"), nil
        },
    }

    client := NewLLMClient(mockModel, RetryConfig{
        MaxRetries:     3,
        InitialBackoff: 1 * time.Millisecond, // Fast for tests
        MaxBackoff:     10 * time.Millisecond,
        BackoffFactor:  2.0,
    })

    response, err := client.GenerateWithRetry(context.Background(), nil)
    
    if err != nil {
        t.Errorf("Expected success after retries, got: %v", err)
    }
    if attempts != 3 {
        t.Errorf("Expected 3 attempts, got %d", attempts)
    }
    if response.GetContent() != "Success!" {
        t.Errorf("Unexpected response: %s", response.GetContent())
    }
}

func TestGenerateWithRetry_FailsOnAuthError(t *testing.T) {
    mockModel := &MockChatModel{
        generateFunc: func(ctx context.Context, msgs []schema.Message) (schema.Message, error) {
            return nil, llms.NewLLMError("Generate", llms.ErrCodeAuthentication, errors.New("invalid API key"))
        },
    }

    client := NewLLMClient(mockModel, DefaultRetryConfig)
    
    _, err := client.GenerateWithRetry(context.Background(), nil)
    
    if err == nil {
        t.Error("Expected error for auth failure")
    }
    if !strings.Contains(err.Error(), "permanent error") {
        t.Errorf("Expected permanent error message, got: %v", err)
    }
}
```

## Variations

### Circuit Breaker Pattern

If you're experiencing sustained failures, add a circuit breaker:

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

For non-critical features, provide a fallback:

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

- **[Custom Agent Extension](./custom-agent.md)** - Use error handling in custom agents
- **[Streaming LLM Guide](../guides/llm-streaming-tool-calls.md)** - Error handling in streaming scenarios
- **[Configuration Guide](../guides/config-providers.md)** - Configure retry behavior via config files
- **[Observability Guide](../guides/observability-tracing.md)** - Track retry metrics with OTEL
