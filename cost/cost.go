// Package cost provides abstractions for tracking and budgeting LLM token
// usage and associated costs during agent execution.
package cost

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// Usage records token consumption for a single agent turn.
type Usage struct {
	// InputTokens is the number of prompt/input tokens consumed.
	InputTokens int
	// OutputTokens is the number of generated output tokens consumed.
	OutputTokens int
	// TotalTokens is InputTokens + OutputTokens.
	TotalTokens int
	// CachedTokens is the number of tokens served from cache.
	CachedTokens int
	// ModelID identifies the model that produced this usage.
	ModelID string
}

// Budget defines the maximum allowed resource consumption for a session.
type Budget struct {
	// MaxTotalTokens is the maximum number of total tokens allowed.
	// Zero means unlimited.
	MaxTotalTokens int
	// MaxInputTokens is the maximum number of input tokens allowed.
	// Zero means unlimited.
	MaxInputTokens int
	// MaxOutputTokens is the maximum number of output tokens allowed.
	// Zero means unlimited.
	MaxOutputTokens int
}

// Tracker is the interface for recording and querying accumulated token usage.
// Implementations must be safe for concurrent use.
type Tracker interface {
	// Record adds the given usage to the accumulated totals.
	Record(ctx context.Context, usage Usage) error
	// Total returns the accumulated usage since the tracker was created or reset.
	Total(ctx context.Context) (Usage, error)
}

// InMemoryTracker is a thread-safe in-memory implementation of Tracker.
type InMemoryTracker struct {
	mu     sync.Mutex
	total  Usage
	budget Budget
}

// NewInMemoryTracker returns a new InMemoryTracker with the given budget.
// A zero-value Budget means unlimited on all dimensions.
func NewInMemoryTracker(b Budget) *InMemoryTracker {
	return &InMemoryTracker{budget: b}
}

// Record adds usage to the accumulated total and checks it against the budget.
// Returns a core.Error with code ErrBudgetExhausted if any budget limit is exceeded.
func (t *InMemoryTracker) Record(_ context.Context, u Usage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check prospective totals against budget before committing.
	if t.budget.MaxTotalTokens > 0 && t.total.TotalTokens+u.TotalTokens > t.budget.MaxTotalTokens {
		return core.NewError("cost.record", core.ErrBudgetExhausted,
			fmt.Sprintf("total token budget exhausted: limit=%d used=%d additional=%d",
				t.budget.MaxTotalTokens, t.total.TotalTokens, u.TotalTokens), nil)
	}
	if t.budget.MaxInputTokens > 0 && t.total.InputTokens+u.InputTokens > t.budget.MaxInputTokens {
		return core.NewError("cost.record", core.ErrBudgetExhausted,
			fmt.Sprintf("input token budget exhausted: limit=%d used=%d additional=%d",
				t.budget.MaxInputTokens, t.total.InputTokens, u.InputTokens), nil)
	}
	if t.budget.MaxOutputTokens > 0 && t.total.OutputTokens+u.OutputTokens > t.budget.MaxOutputTokens {
		return core.NewError("cost.record", core.ErrBudgetExhausted,
			fmt.Sprintf("output token budget exhausted: limit=%d used=%d additional=%d",
				t.budget.MaxOutputTokens, t.total.OutputTokens, u.OutputTokens), nil)
	}

	t.total.InputTokens += u.InputTokens
	t.total.OutputTokens += u.OutputTokens
	t.total.TotalTokens += u.TotalTokens
	t.total.CachedTokens += u.CachedTokens
	return nil
}

// Total returns the accumulated usage.
func (t *InMemoryTracker) Total(_ context.Context) (Usage, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.total, nil
}
