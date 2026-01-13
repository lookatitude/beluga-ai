# Quick Solutions

Quick fixes for common problems.

## Problem: API Key Not Found

**Solution:**
```bash
export OPENAI_API_KEY="your-key-here"
```

## Problem: Slow Responses

**Solution:**
```go
// Use faster model
llms.WithModelName("gpt-3.5-turbo")

// Enable streaming
streamChan, _ := provider.StreamChat(ctx, messages)
```

## Problem: Memory Issues

**Solution:**
```go
// Use window memory
mem, _ := memory.NewMemory(
    memory.MemoryTypeWindow,
    memory.WithWindowSize(10),
)
```

## Problem: Rate Limits

**Solution:**
```go
// Add retry configuration
llms.WithRetryConfig(5, 2*time.Second, 2.0)
```

---

**More Help:** [Troubleshooting Guide](../troubleshooting.md)

