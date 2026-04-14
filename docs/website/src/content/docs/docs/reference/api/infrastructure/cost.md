---
title: "Cost API — Token Tracking and Budget Enforcement"
description: "Cost package API reference for Beluga AI. LLM usage tracking, budget enforcement with rolling windows, filter-based queries, and registry pattern for pluggable tracker backends."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "cost API, usage tracking, budget, token limit, BudgetChecker, Tracker, InMemoryTracker, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/cost"
```

Package cost tracks LLM token usage and enforces spending budgets. It records
Usage entries after each LLM call and answers aggregate queries so that budget
checks can compare projected consumption against rolling-window totals before
a call is made.

## Core Types

### Usage

Usage captures the token consumption and monetary cost of a single LLM call.

```go
type Usage struct {
    InputTokens  int
    OutputTokens int
    CachedTokens int
    TotalTokens  int     // set explicitly; not derived automatically
    Cost         float64 // USD
    Model        string  // e.g. "gpt-4o"
    Provider     string  // e.g. "openai"
    TenantID     string
    Timestamp    time.Time
}
```

### Filter

Filter constrains a Query call. Zero-value fields are ignored and match any
record.

```go
type Filter struct {
    TenantID string
    Model    string
    Provider string
    Since    time.Time
    Until    time.Time
}
```

### Summary

Summary is the aggregated result of a Query call.

```go
type Summary struct {
    TotalInputTokens  int64
    TotalOutputTokens int64
    TotalCost         float64
    EntryCount        int64
}
```

## Tracker Interface

Tracker records usage entries and answers aggregate queries. Implementations
must be safe for concurrent use.

```go
type Tracker interface {
    Record(ctx context.Context, usage Usage) error
    Query(ctx context.Context, filter Filter) (*Summary, error)
}
```

### InMemoryTracker

The built-in InMemoryTracker stores entries in a protected slice. Entries are
evicted in FIFO order when the capacity limit is reached.

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/cost"
)

tracker := cost.NewInMemoryTracker(
    cost.WithMaxEntries(50000),
)

err := tracker.Record(ctx, cost.Usage{
    Model:        "gpt-4o",
    Provider:     "openai",
    TenantID:     "tenant-1",
    InputTokens:  512,
    OutputTokens: 128,
    TotalTokens:  640,
    Cost:         0.0032,
    Timestamp:    time.Now(),
})
if err != nil {
    fmt.Println("record error:", err)
}

summary, err := tracker.Query(ctx, cost.Filter{
    TenantID: "tenant-1",
    Since:    time.Now().Add(-24 * time.Hour),
})
if err != nil {
    fmt.Println("query error:", err)
} else {
    fmt.Printf("24h cost: $%.4f over %d calls\n", summary.TotalCost, summary.EntryCount)
}
```

`WithMaxEntries` defaults to 100,000 when not specified.

## Budget and BudgetChecker

### Budget

Budget defines spending limits and the action to take when they are exceeded.

```go
type Budget struct {
    MaxTokensPerHour int64        // rolling 1-hour token limit; 0 means no limit
    MaxCostPerDay    float64      // rolling 24-hour USD limit; 0 means no limit
    AlertThreshold   float64      // fraction [0,1] at which to return an alert decision
    Action           BudgetAction // "throttle", "reject", or "alert"
}
```

BudgetAction values:

| Value | Meaning |
|-------|---------|
| `BudgetActionThrottle` | Allow but signal the caller to slow down. |
| `BudgetActionReject` | Deny the request entirely (Allowed: false). |
| `BudgetActionAlert` | Allow but surface a warning reason. |

### BudgetDecision

```go
type BudgetDecision struct {
    Allowed    bool
    Reason     string  // non-empty when denied or when alert threshold is crossed
    UsageRatio float64 // current usage as a fraction of the binding limit
}
```

### BudgetChecker Interface

```go
type BudgetChecker interface {
    Check(ctx context.Context, budget Budget, estimated Usage) (BudgetDecision, error)
}
```

### InMemoryBudgetChecker

InMemoryBudgetChecker queries the Tracker for rolling-window totals before
deciding whether to allow the estimated usage.

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/cost"
)

tracker := cost.NewInMemoryTracker()
checker := cost.NewInMemoryBudgetChecker(tracker)

budget := cost.Budget{
    MaxTokensPerHour: 100000,
    MaxCostPerDay:    10.0,
    AlertThreshold:   0.8,
    Action:           cost.BudgetActionReject,
}

estimated := cost.Usage{
    TenantID:    "tenant-1",
    TotalTokens: 2000,
    Cost:        0.01,
}

decision, err := checker.Check(context.Background(), budget, estimated)
if err != nil {
    fmt.Println("check error:", err)
} else if !decision.Allowed {
    fmt.Println("budget exceeded:", decision.Reason)
} else if decision.Reason != "" {
    fmt.Printf("warning: %s (ratio=%.2f)\n", decision.Reason, decision.UsageRatio)
}
```

Check evaluates the hourly token window first, then the daily cost window. It
returns an error only if the underlying Tracker.Query fails.

## Registry

The registry pattern allows pluggable Tracker backends. The "inmemory" name is
registered automatically in the package's init function.

```go
// Register a custom backend (call from your package's init).
func init() {
    cost.Register("postgres", func(cfg cost.Config) (cost.Tracker, error) {
        return newPostgresTracker(cfg)
    })
}

// Create a tracker by name.
tracker, err := cost.New("postgres", cost.Config{
    MaxEntries: 0,
    Options: map[string]any{
        "dsn": "postgres://...",
    },
})
if err != nil {
    // handle registration miss
}

// Discover registered names.
names := cost.List() // e.g. ["inmemory", "postgres"]
```

### Config

```go
type Config struct {
    MaxEntries int            // 0 means unlimited for in-memory; ignored by some backends
    Options    map[string]any // provider-specific key-value configuration
}
```

## Related Packages

- `audit` — Structured audit logging that complements cost tracking.
- `o11y` — OpenTelemetry integration; emit cost metrics via OTel instruments.
- `core` — Error types returned by Tracker and BudgetChecker on context cancellation.
- `docs/concepts.md` — Multi-tenancy model and how TenantID scopes records.
