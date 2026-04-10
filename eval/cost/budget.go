package cost

import (
	"fmt"
	"sync"
)

// BudgetAlert monitors cumulative evaluation cost and fires a callback when
// the cost exceeds a configured threshold.
type BudgetAlert struct {
	mu        sync.Mutex
	threshold float64
	total     float64
	fired     bool
	onAlert   func(total float64, threshold float64)
}

// BudgetAlertOption configures a BudgetAlert.
type BudgetAlertOption func(*BudgetAlert)

// WithThreshold sets the dollar threshold that triggers the alert.
func WithThreshold(threshold float64) BudgetAlertOption {
	return func(b *BudgetAlert) {
		b.threshold = threshold
	}
}

// WithOnAlert sets the callback invoked when the threshold is exceeded.
// The callback receives the current total and the threshold. The callback is
// invoked without holding the internal BudgetAlert mutex, so it is safe to
// re-enter other BudgetAlert methods (Total, Exceeded, Reset) from within it.
func WithOnAlert(fn func(total float64, threshold float64)) BudgetAlertOption {
	return func(b *BudgetAlert) {
		b.onAlert = fn
	}
}

// NewBudgetAlert creates a new BudgetAlert with the given options.
func NewBudgetAlert(opts ...BudgetAlertOption) (*BudgetAlert, error) {
	b := &BudgetAlert{}
	for _, opt := range opts {
		opt(b)
	}
	if b.threshold <= 0 {
		return nil, fmt.Errorf("budget: threshold must be positive, got %f", b.threshold)
	}
	return b, nil
}

// Add accumulates cost and fires the alert callback if the threshold is
// exceeded. The alert fires at most once. Returns true if the threshold
// has been exceeded (including prior calls).
//
// The onAlert callback is invoked after releasing the internal mutex so that
// the callback may safely call other BudgetAlert methods (Total, Exceeded,
// Reset) without deadlocking, and so that a slow callback does not block
// concurrent Add callers.
func (b *BudgetAlert) Add(amount float64) bool {
	b.mu.Lock()
	b.total += amount
	total := b.total
	threshold := b.threshold
	shouldFire := total >= threshold && !b.fired
	if shouldFire {
		b.fired = true
	}
	callback := b.onAlert
	exceeded := total >= threshold
	b.mu.Unlock()

	if shouldFire && callback != nil {
		callback(total, threshold)
	}

	return exceeded
}

// Total returns the current cumulative cost.
func (b *BudgetAlert) Total() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.total
}

// Exceeded returns whether the threshold has been exceeded.
func (b *BudgetAlert) Exceeded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.total >= b.threshold
}

// Reset clears the accumulated cost and fired state.
func (b *BudgetAlert) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.total = 0
	b.fired = false
}
