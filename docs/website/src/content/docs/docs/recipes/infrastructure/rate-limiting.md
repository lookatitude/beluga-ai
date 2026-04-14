---
title: "Rate Limiting per Project"
description: "Recipe for implementing per-project rate limiting in Go with multi-window tracking, tier-based limits, and fair multi-tenant resource allocation."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, rate limiting, Go multi-tenant, per-project limits, tier-based pricing, token bucket, resource allocation recipe"
---

## Problem

Multi-tenant AI systems require fair resource allocation across projects. Without rate limiting, a single project can consume excessive API quota, exhaust LLM provider limits, or overwhelm infrastructure, degrading service for other tenants. This becomes critical when you offer tiered pricing where different projects pay for different usage levels.

The challenge is implementing per-project isolation that prevents one project from affecting others while allowing flexible limit configurations. You need to track usage across multiple time windows to prevent both burst abuse and sustained overuse. A project might attempt to circumvent a per-minute limit by spreading requests across minutes, necessitating hour and day limits as well.

Additionally, rate limiting must be fast and accurate. Checking limits cannot add significant latency to requests, and limit enforcement must be consistent to avoid accidentally allowing over-limit requests during concurrent access.

## Solution

Per-project rate limiting isolates each project's usage tracking, ensuring projects cannot interfere with each other. The sliding window algorithm tracks request timestamps rather than just counts, providing accurate rate limiting across arbitrary time ranges. This is more precise than fixed windows, which can allow bursts at window boundaries.

Multi-window tracking enforces limits at different time scales simultaneously. You check minute, hour, and day limits on every request. This prevents both short burst attacks (caught by minute limit) and sustained abuse (caught by day limit). Legitimate traffic patterns naturally spread requests across time, so multiple windows don't typically create false positives.

The automatic cleanup mechanism prevents unbounded memory growth by removing expired timestamps on each request. This is more efficient than background cleanup threads because work is distributed across requests, and cleanup only occurs for active projects.

The middleware pattern integrates rate limiting into your HTTP stack without coupling it to business logic. Extract the project ID from request headers, check limits, and either allow or reject the request before it reaches your handlers. This ensures rate limiting applies uniformly across all endpoints.

The key design decision is using timestamps instead of counters. Timestamps allow precise sliding windows, support multiple time windows from a single data structure, and naturally expire as they age. The tradeoff is slightly higher memory usage, which is acceptable because you only store timestamps for recent requests.

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

1. **Multi-window tracking prevents multiple attack vectors** — Tracking requests across minute, hour, and day windows simultaneously catches different types of abuse. A burst attack that sends 60 requests in one second hits the minute limit. A sustained attack that sends steady traffic all day hits the day limit. Legitimate traffic rarely exceeds multiple windows simultaneously, making this approach effective with low false positives.

2. **Per-project isolation ensures fairness** — Each project has its own counter, stored in a separate map entry. This provides complete isolation between tenants. A noisy project that hits rate limits does not affect other projects at all. This isolation is essential in multi-tenant systems where users expect predictable performance regardless of other tenants' behavior.

3. **Automatic cleanup prevents memory leaks** — The cleanup function runs on every request and removes expired timestamps. This is efficient because cleanup work is proportional to the number of requests, and inactive projects naturally have fewer cleanup operations. The alternative, background cleanup threads, adds complexity and can cause spiky CPU usage when cleaning many inactive projects simultaneously.

4. **Sliding windows provide accuracy** — Storing timestamps instead of simple counters enables true sliding window rate limiting. If a project sends 60 requests between 12:00:30 and 12:01:30, the minute limit prevents additional requests until 12:01:31, when the first request timestamp expires. Fixed windows would allow another 60 requests at 12:01:00, enabling double the intended rate at window boundaries.

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
