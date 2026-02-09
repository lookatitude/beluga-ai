---
title: "Rate Limiting per Project"
description: "Implement per-project rate limiting with multi-window tracking and tier-based limits."
---

## Problem

You need to implement rate limiting per project or tenant to prevent abuse and ensure fair resource allocation across multiple projects using your API, while allowing different rate limits for different project tiers.

## Solution

Implement a per-project rate limiter that tracks request counts per project ID, enforces limits using a sliding window algorithm, and supports different limits per project tier. You can identify projects from request context and maintain separate rate limit state for each.

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

// ProjectLimit defines rate limits for a project.
type ProjectLimit struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
}

var DefaultProjectLimit = &ProjectLimit{
	RequestsPerMinute: 60,
	RequestsPerHour:   1000,
	RequestsPerDay:    10000,
}

// RequestCounter tracks request counts for a project.
type RequestCounter struct {
	MinuteCount []time.Time
	HourCount   []time.Time
	DayCount    []time.Time
	mu          sync.Mutex
}

// RateLimiter limits requests per project.
type RateLimiter struct {
	counters     map[string]*RequestCounter
	mu           sync.RWMutex
	defaultLimit *ProjectLimit
}

func NewRateLimiter(defaultLimit *ProjectLimit) *RateLimiter {
	if defaultLimit == nil {
		defaultLimit = DefaultProjectLimit
	}
	return &RateLimiter{
		counters:     make(map[string]*RequestCounter),
		defaultLimit: defaultLimit,
	}
}

// Allow checks if a request should be allowed for the given project.
func (rl *RateLimiter) Allow(ctx context.Context, projectID string) (bool, error) {
	ctx, span := tracer.Start(ctx, "rate_limiter.allow")
	defer span.End()

	span.SetAttributes(attribute.String("project_id", projectID))

	rl.mu.Lock()
	counter, exists := rl.counters[projectID]
	if !exists {
		counter = &RequestCounter{}
		rl.counters[projectID] = counter
	}
	rl.mu.Unlock()

	limit := rl.defaultLimit

	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()
	counter.cleanup(now)

	if len(counter.MinuteCount) >= limit.RequestsPerMinute {
		span.SetAttributes(attribute.String("limit_exceeded", "minute"))
		return false, fmt.Errorf("rate limit exceeded: %d requests/minute", limit.RequestsPerMinute)
	}
	if len(counter.HourCount) >= limit.RequestsPerHour {
		span.SetAttributes(attribute.String("limit_exceeded", "hour"))
		return false, fmt.Errorf("rate limit exceeded: %d requests/hour", limit.RequestsPerHour)
	}
	if len(counter.DayCount) >= limit.RequestsPerDay {
		span.SetAttributes(attribute.String("limit_exceeded", "day"))
		return false, fmt.Errorf("rate limit exceeded: %d requests/day", limit.RequestsPerDay)
	}

	counter.MinuteCount = append(counter.MinuteCount, now)
	counter.HourCount = append(counter.HourCount, now)
	counter.DayCount = append(counter.DayCount, now)

	span.SetAttributes(
		attribute.Int("remaining_minute", limit.RequestsPerMinute-len(counter.MinuteCount)),
	)
	span.SetStatus(trace.StatusOK, "request allowed")
	return true, nil
}

func (rc *RequestCounter) cleanup(now time.Time) {
	minuteAgo := now.Add(-1 * time.Minute)
	hourAgo := now.Add(-1 * time.Hour)
	dayAgo := now.Add(-24 * time.Hour)

	rc.MinuteCount = filterAfter(rc.MinuteCount, minuteAgo)
	rc.HourCount = filterAfter(rc.HourCount, hourAgo)
	rc.DayCount = filterAfter(rc.DayCount, dayAgo)
}

func filterAfter(times []time.Time, threshold time.Time) []time.Time {
	valid := make([]time.Time, 0, len(times))
	for _, t := range times {
		if t.After(threshold) {
			valid = append(valid, t)
		}
	}
	return valid
}

// RateLimitMiddleware creates HTTP middleware for rate limiting.
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			projectID := r.Header.Get("X-Project-ID")
			if projectID == "" {
				http.Error(w, "project ID required", http.StatusBadRequest)
				return
			}

			allowed, err := limiter.Allow(r.Context(), projectID)
			if err != nil || !allowed {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	limiter := NewRateLimiter(nil)

	allowed, err := limiter.Allow(context.Background(), "project-123")
	if err != nil {
		log.Fatalf("Rate limit error: %v", err)
	}
	if !allowed {
		log.Fatal("Rate limit exceeded")
	}
	fmt.Println("Request allowed")
}
```

## Explanation

1. **Multi-window tracking** -- Requests are tracked in multiple time windows (minute, hour, day). This provides fine-grained control and prevents both short bursts and sustained abuse.

2. **Per-project isolation** -- Each project has its own counter, ensuring projects do not affect each other's rate limits. This is essential for multi-tenant systems.

3. **Automatic cleanup** -- Old entries are removed from counters on each check to prevent unbounded memory growth.

**Key insight:** Use multiple time windows for rate limiting. This prevents both burst attacks (minute limit) and sustained abuse (day limit) while allowing legitimate traffic patterns.

## Variations

### Distributed Rate Limiting

Use Redis for distributed rate limiting across multiple server instances:

```go
type DistributedRateLimiter struct {
	redis *redis.Client
}

func (d *DistributedRateLimiter) Allow(ctx context.Context, projectID string) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s:%d", projectID, time.Now().Unix()/60)
	count, err := d.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	d.redis.Expire(ctx, key, 2*time.Minute)
	return count <= 60, nil
}
```

## Related Recipes

- **[Request ID Correlation](./request-id-correlation)** -- Track requests across services
- **[Global Retry Wrappers](./global-retry)** -- Handle rate limit errors with retries
