package cost

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// Compile-time check that InMemoryTracker satisfies the Tracker interface.
var _ Tracker = (*InMemoryTracker)(nil)

// options holds configuration for an InMemoryTracker.
type options struct {
	maxEntries int
}

// Option is a functional option for configuring an InMemoryTracker.
type Option func(*options)

// WithMaxEntries sets the maximum number of usage records the tracker will
// retain. When the limit is reached, the oldest records are evicted first.
// Zero (the default) means unlimited.
func WithMaxEntries(n int) Option {
	return func(o *options) {
		o.maxEntries = n
	}
}

// InMemoryTracker is a thread-safe, in-memory implementation of Tracker.
// It stores Usage records in a slice and supports filtering by tenant, model,
// provider, and time range.
type InMemoryTracker struct {
	mu      sync.RWMutex
	entries []Usage
	opts    options
}

// NewInMemoryTracker creates a new InMemoryTracker with the supplied options.
func NewInMemoryTracker(opts ...Option) *InMemoryTracker {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	return &InMemoryTracker{opts: o}
}

// Record stores a usage entry. It returns an error if the context is already
// cancelled. When MaxEntries is set and the limit has been reached, the oldest
// entry is evicted before inserting the new one.
func (t *InMemoryTracker) Record(ctx context.Context, usage Usage) error {
	if err := ctx.Err(); err != nil {
		return core.NewError("cost.tracker.record", core.ErrTimeout, "context cancelled", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.opts.maxEntries > 0 && len(t.entries) >= t.opts.maxEntries {
		// Evict oldest entry.
		t.entries = t.entries[1:]
	}

	t.entries = append(t.entries, usage)
	return nil
}

// Query returns an aggregated Summary of all stored records that satisfy the
// filter. It returns an error only if the context is already cancelled.
func (t *InMemoryTracker) Query(ctx context.Context, filter Filter) (*Summary, error) {
	if err := ctx.Err(); err != nil {
		return nil, core.NewError("cost.tracker.query", core.ErrTimeout, "context cancelled", err)
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	var summary Summary
	for i := range t.entries {
		e := &t.entries[i]
		if !matchesFilter(e, filter) {
			continue
		}
		summary.TotalInputTokens += int64(e.InputTokens)
		summary.TotalOutputTokens += int64(e.OutputTokens)
		summary.TotalCost += e.Cost
		summary.EntryCount++
	}
	return &summary, nil
}

// matchesFilter reports whether a Usage record satisfies all non-zero fields
// of the filter.
func matchesFilter(u *Usage, f Filter) bool {
	if f.TenantID != "" && u.TenantID != f.TenantID {
		return false
	}
	if f.Model != "" && u.Model != f.Model {
		return false
	}
	if f.Provider != "" && u.Provider != f.Provider {
		return false
	}
	if !f.Since.IsZero() && u.Timestamp.Before(f.Since) {
		return false
	}
	if !f.Until.IsZero() && !u.Timestamp.Before(f.Until) {
		return false
	}
	return true
}
