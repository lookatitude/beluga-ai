---
title: "Rate Limiting per Project"
package: "server"
category: "security"
complexity: "intermediate"
---

# Rate Limiting per Project

## Problem

You need to implement rate limiting per project/tenant to prevent abuse and ensure fair resource allocation across multiple projects using your API, while allowing different rate limits for different project tiers.

## Solution

Implement a per-project rate limiter that tracks request counts per project ID, enforces limits using token bucket or sliding window algorithms, and supports different limits per project tier. This works because you can identify projects from request context and maintain separate rate limit state for each project.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "sync"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.server.rate_limiting")

// RateLimiter limits requests per project
type RateLimiter struct {
    limits    map[string]*ProjectLimit
    mu        sync.RWMutex
    defaultLimit *ProjectLimit
}

// ProjectLimit defines rate limits for a project
type ProjectLimit struct {
    RequestsPerMinute int
    RequestsPerHour   int
    RequestsPerDay    int
}

// DefaultProjectLimit provides default limits
var DefaultProjectLimit = &ProjectLimit{
    RequestsPerMinute: 60,
    RequestsPerHour:   1000,
    RequestsPerDay:    10000,
}

// RequestCounter tracks request counts for a project
type RequestCounter struct {
    MinuteCount []time.Time
    HourCount   []time.Time
    DayCount    []time.Time
    mu          sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(defaultLimit *ProjectLimit) *RateLimiter {
    if defaultLimit == nil {
        defaultLimit = DefaultProjectLimit
    }

    return &RateLimiter{
        limits:      make(map[string]*RequestCounter),
        defaultLimit: defaultLimit,
    }
}

// SetProjectLimit sets custom limits for a project
func (rl *RateLimiter) SetProjectLimit(projectID string, limit *ProjectLimit) {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    // Initialize counter if needed
    if _, exists := rl.limits[projectID]; !exists {
        rl.limits[projectID] = &RequestCounter{}
    }
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ctx context.Context, projectID string) (bool, error) {
    ctx, span := tracer.Start(ctx, "rate_limiter.allow")
    defer span.End()
    
    span.SetAttributes(attribute.String("project_id", projectID))
    
    // Get or create counter
    rl.mu.Lock()
    counter, exists := rl.limits[projectID]
    if !exists {
        counter = &RequestCounter{}
        rl.limits[projectID] = counter
    }
    rl.mu.Unlock()
    
    // Get project limit (could be from config/tier)
    limit := rl.getProjectLimit(projectID)
    
    counter.mu.Lock()
    defer counter.mu.Unlock()
    
    now := time.Now()
    
    // Clean old entries
    counter.cleanup(now)
    
    // Check limits
    if len(counter.MinuteCount) >= limit.RequestsPerMinute {
        span.SetAttributes(attribute.String("limit_exceeded", "minute"))
        span.SetStatus(trace.StatusError, "rate limit exceeded")
        return false, fmt.Errorf("rate limit exceeded: %d requests/minute", limit.RequestsPerMinute)
    }
    
    if len(counter.HourCount) >= limit.RequestsPerHour {
        span.SetAttributes(attribute.String("limit_exceeded", "hour"))
        span.SetStatus(trace.StatusError, "rate limit exceeded")
        return false, fmt.Errorf("rate limit exceeded: %d requests/hour", limit.RequestsPerHour)
    }
    
    if len(counter.DayCount) >= limit.RequestsPerDay {
        span.SetAttributes(attribute.String("limit_exceeded", "day"))
        span.SetStatus(trace.StatusError, "rate limit exceeded")
        return false, fmt.Errorf("rate limit exceeded: %d requests/day", limit.RequestsPerDay)
    }
    
    // Record request
    counter.MinuteCount = append(counter.MinuteCount, now)
    counter.HourCount = append(counter.HourCount, now)
    counter.DayCount = append(counter.DayCount, now)
    
    span.SetAttributes(
        attribute.Int("remaining_minute", limit.RequestsPerMinute-len(counter.MinuteCount)),
        attribute.Int("remaining_hour", limit.RequestsPerHour-len(counter.HourCount)),
    )
    span.SetStatus(trace.StatusOK, "request allowed")
    
    return true, nil
}

// getProjectLimit gets the limit for a project
func (rl *RateLimiter) getProjectLimit(projectID string) *ProjectLimit {
    // In practice, this would check project tier/configuration
    return rl.defaultLimit
}

// cleanup removes old entries from counters
func (rc *RequestCounter) cleanup(now time.Time) {
    minuteAgo := now.Add(-1 * time.Minute)
    hourAgo := now.Add(-1 * time.Hour)
    dayAgo := now.Add(-24 * time.Hour)

    // Clean minute count
    valid := []time.Time{}
    for _, t := range rc.MinuteCount {
        if t.After(minuteAgo) {
            valid = append(valid, t)
        }
    }
    rc.MinuteCount = valid
    
    // Clean hour count
    valid = []time.Time{}
    for _, t := range rc.HourCount {
        if t.After(hourAgo) {
            valid = append(valid, t)
        }
    }
    rc.HourCount = valid
    
    // Clean day count
    valid = []time.Time{}
    for _, t := range rc.DayCount {
        if t.After(dayAgo) {
            valid = append(valid, t)
        }
    }
    rc.DayCount = valid
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract project ID from request (header, auth token, etc.)
            projectID := r.Header.Get("X-Project-ID")
            if projectID == "" {
                http.Error(w, "project ID required", http.StatusBadRequest)
                return
            }
            
            // Check rate limit
            allowed, err := limiter.Allow(r.Context(), projectID)
            if err != nil {
                http.Error(w, err.Error(), http.StatusTooManyRequests)
                return
            }
            
            if !allowed {
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            
            // Add rate limit headers
            w.Header().Set("X-RateLimit-Remaining", "60") // Would calculate actual remaining
            next.ServeHTTP(w, r)
        })
    }
}

func main() {
    ctx := context.Background()

    // Create rate limiter
    limiter := NewRateLimiter(nil)
    
    // Check rate limit
    allowed, err := limiter.Allow(ctx, "project-123")
    if err != nil {
        log.Fatalf("Rate limit error: %v", err)
    }
    if !allowed \{
        log.Fatal("Rate limit exceeded")
    }
    fmt.Println("Request allowed")
}
```

## Explanation

Let's break down what's happening:

1. **Multi-window tracking** - Notice how we track requests in multiple time windows (minute, hour, day). This allows fine-grained control while preventing both short bursts and sustained abuse.

2. **Per-project isolation** - Each project has its own counter, ensuring projects don't affect each other's rate limits. This is important for multi-tenant systems.

3. **Automatic cleanup** - We clean old entries from counters to prevent memory growth. Only recent requests are tracked.

```go
**Key insight:** Use multiple time windows for rate limiting. This prevents both burst attacks (minute limit) and sustained abuse (day limit) while allowing legitimate traffic patterns.

## Testing

```
Here's how to test this solution:
```go
func TestRateLimiter_EnforcesLimits(t *testing.T) {
    limit := &ProjectLimit{
        RequestsPerMinute: 2,
        RequestsPerHour:   10,
        RequestsPerDay:    100,
    }
    limiter := NewRateLimiter(limit)
    
    // First two requests should pass
    allowed, _ := limiter.Allow(context.Background(), "test")
    require.True(t, allowed)
    
    allowed, _ = limiter.Allow(context.Background(), "test")
    require.True(t, allowed)
    
    // Third should fail
    allowed, _ = limiter.Allow(context.Background(), "test")
    require.False(t, allowed)
}

## Variations

### Token Bucket Algorithm

Use token bucket for smoother rate limiting:
type TokenBucket struct {
    tokens    float64
    capacity  float64
    refillRate float64
    lastRefill time.Time
}
```

### Distributed Rate Limiting

Use Redis for distributed rate limiting:
```go
type DistributedRateLimiter struct {
    redis *redis.Client
}
```

## Related Recipes

- **[Server Correlating Request-IDs across Services](./server-correlating-request-ids.md)** - Track requests across services
- **[Core Global Retry Wrappers](./core-global-retry-wrappers.md)** - Handle rate limit errors
- **[Server Package Guide](../package_design_patterns.md)** - For a deeper understanding of server
