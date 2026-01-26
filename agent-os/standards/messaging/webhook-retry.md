# Webhook Retry with Exponential Backoff

Retry transient webhook failures with increasing delays.

```go
maxRetries := 3
retryDelay := time.Second

for attempt := 0; attempt < maxRetries; attempt++ {
    err = p.handleWebhookEvent(ctx, event)
    if err == nil {
        break
    }
    if attempt < maxRetries-1 {
        time.Sleep(retryDelay * time.Duration(1<<attempt))  // 1s, 2s, 4s
    }
}
```

## Timing
- 3 retries with delays: 1s, 2s, 4s
- Total max time: ~7s before final failure
- Conservative choice balancing reliability with HTTP timeout constraints

## When to Use
- Webhook handlers with transient failures (network, rate limits)
- NOT for permanent errors (validation, auth failures)

## Note
Webhook providers typically timeout at 5-30s. Total retry time must fit within this window.
