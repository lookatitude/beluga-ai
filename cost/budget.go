package cost

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// BudgetAction defines what happens when a Budget threshold is reached.
type BudgetAction string

const (
	// BudgetActionThrottle slows down subsequent requests.
	BudgetActionThrottle BudgetAction = "throttle"

	// BudgetActionReject denies the request entirely.
	BudgetActionReject BudgetAction = "reject"

	// BudgetActionAlert sends a notification but still allows the request.
	BudgetActionAlert BudgetAction = "alert"
)

// Budget defines spending limits and the action to take when they are exceeded.
type Budget struct {
	// MaxTokensPerHour is the maximum total tokens allowed in a rolling one-hour
	// window. Zero means no token limit.
	MaxTokensPerHour int64

	// MaxCostPerDay is the maximum USD cost allowed in a rolling 24-hour window.
	// Zero means no cost limit.
	MaxCostPerDay float64

	// AlertThreshold is the fraction of the budget (0.0–1.0) at which to start
	// alerting. For example, 0.8 means alert at 80% usage.
	AlertThreshold float64

	// Action is what the BudgetChecker should do when a limit is exceeded.
	Action BudgetAction
}

// BudgetDecision is the result of a BudgetChecker.Check call.
type BudgetDecision struct {
	// Allowed reports whether the estimated usage fits within the budget.
	Allowed bool

	// Reason is a human-readable explanation when Allowed is false or when an
	// alert threshold has been crossed.
	Reason string

	// UsageRatio is the current consumption expressed as a fraction of the
	// most-binding limit (0.0 = empty, 1.0 = exactly at the limit).
	UsageRatio float64
}

// BudgetChecker evaluates an estimated Usage against a Budget and decides
// whether to proceed.
type BudgetChecker interface {
	// Check compares the estimated Usage plus the already-consumed usage
	// (determined by querying the underlying Tracker) against the Budget.
	// It returns a BudgetDecision and any error encountered during the check.
	Check(ctx context.Context, budget Budget, estimated Usage) (BudgetDecision, error)
}

// InMemoryBudgetChecker is a BudgetChecker backed by any Tracker.  It queries
// the tracker for the rolling-window totals before deciding whether to allow
// the estimated usage.
type InMemoryBudgetChecker struct {
	tracker Tracker
}

// Compile-time check.
var _ BudgetChecker = (*InMemoryBudgetChecker)(nil)

// NewInMemoryBudgetChecker creates an InMemoryBudgetChecker that queries
// tracker for current usage when evaluating a Budget.
func NewInMemoryBudgetChecker(tracker Tracker) *InMemoryBudgetChecker {
	return &InMemoryBudgetChecker{tracker: tracker}
}

// Check evaluates the estimated usage against the budget by querying the
// tracker for the rolling-window totals. It returns an error only if the
// tracker query fails.
func (c *InMemoryBudgetChecker) Check(ctx context.Context, budget Budget, estimated Usage) (BudgetDecision, error) {
	now := time.Now()

	// --- token-per-hour window ---
	if budget.MaxTokensPerHour > 0 {
		hourFilter := Filter{
			TenantID: estimated.TenantID,
			Since:    now.Add(-time.Hour),
		}
		summary, err := c.tracker.Query(ctx, hourFilter)
		if err != nil {
			return BudgetDecision{}, core.NewError(
				"cost.budgetchecker.check",
				core.ErrInvalidInput,
				"failed to query hourly token usage",
				err,
			)
		}

		currentTotal := summary.TotalInputTokens + summary.TotalOutputTokens
		projected := currentTotal + int64(estimated.TotalTokens)
		ratio := float64(projected) / float64(budget.MaxTokensPerHour)

		if projected > budget.MaxTokensPerHour {
			return BudgetDecision{
				Allowed:    budget.Action == BudgetActionAlert,
				Reason:     "hourly token budget exceeded",
				UsageRatio: ratio,
			}, nil
		}

		if budget.AlertThreshold > 0 && ratio >= budget.AlertThreshold {
			return BudgetDecision{
				Allowed:    true,
				Reason:     "approaching hourly token budget limit",
				UsageRatio: ratio,
			}, nil
		}
	}

	// --- cost-per-day window ---
	if budget.MaxCostPerDay > 0 {
		dayFilter := Filter{
			TenantID: estimated.TenantID,
			Since:    now.Add(-24 * time.Hour),
		}
		summary, err := c.tracker.Query(ctx, dayFilter)
		if err != nil {
			return BudgetDecision{}, core.NewError(
				"cost.budgetchecker.check",
				core.ErrInvalidInput,
				"failed to query daily cost usage",
				err,
			)
		}

		projectedCost := summary.TotalCost + estimated.Cost
		ratio := projectedCost / budget.MaxCostPerDay

		if projectedCost > budget.MaxCostPerDay {
			return BudgetDecision{
				Allowed:    budget.Action == BudgetActionAlert,
				Reason:     "daily cost budget exceeded",
				UsageRatio: ratio,
			}, nil
		}

		if budget.AlertThreshold > 0 && ratio >= budget.AlertThreshold {
			return BudgetDecision{
				Allowed:    true,
				Reason:     "approaching daily cost budget limit",
				UsageRatio: ratio,
			}, nil
		}
	}

	return BudgetDecision{Allowed: true, UsageRatio: 0}, nil
}
