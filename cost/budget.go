package cost

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// BudgetAction defines what happens when a Budget threshold is reached.
type BudgetAction string

const (
	// BudgetActionThrottle denies the request for now but marks the decision
	// as Throttled, telling the caller to slow down and retry once the
	// rolling window has room, rather than treating it as a hard failure.
	// The checker itself does not sleep.
	BudgetActionThrottle BudgetAction = "throttle"

	// BudgetActionReject denies the request entirely. An empty Action is
	// treated as BudgetActionReject.
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
	// Empty means BudgetActionReject.
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

	// Throttled is set when a limit was exceeded and the budget's Action is
	// BudgetActionThrottle: the request is not allowed now, and the caller
	// should back off and retry later.
	Throttled bool
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
// tracker for the rolling-window totals. Both windows (hourly tokens and
// daily cost) are always evaluated and the most restrictive outcome wins:
// an exceeded limit beats an alert, and among equals the higher usage ratio
// is reported. Usage is scoped to estimated.TenantID exactly, so an empty
// TenantID checks the default tenant only. The estimated token count is
// estimated.Tokens(). It returns an error only if the tracker query fails.
func (c *InMemoryBudgetChecker) Check(ctx context.Context, budget Budget, estimated Usage) (BudgetDecision, error) {
	now := time.Now()
	var windows []windowResult

	// --- token-per-hour window ---
	if budget.MaxTokensPerHour > 0 {
		summary, err := c.query(ctx, estimated.TenantID, now.Add(-time.Hour), "failed to query hourly token usage")
		if err != nil {
			return BudgetDecision{}, err
		}
		projected := summary.TotalInputTokens + summary.TotalOutputTokens + int64(estimated.Tokens())
		windows = append(windows, windowResult{
			ratio:    float64(projected) / float64(budget.MaxTokensPerHour),
			exceeded: projected > budget.MaxTokensPerHour,
			over:     "hourly token budget exceeded",
			near:     "approaching hourly token budget limit",
		})
	}

	// --- cost-per-day window ---
	if budget.MaxCostPerDay > 0 {
		summary, err := c.query(ctx, estimated.TenantID, now.Add(-24*time.Hour), "failed to query daily cost usage")
		if err != nil {
			return BudgetDecision{}, err
		}
		projectedCost := summary.TotalCost + estimated.Cost
		windows = append(windows, windowResult{
			ratio:    projectedCost / budget.MaxCostPerDay,
			exceeded: projectedCost > budget.MaxCostPerDay,
			over:     "daily cost budget exceeded",
			near:     "approaching daily cost budget limit",
		})
	}

	return decide(budget, windows), nil
}

// windowResult is the outcome of evaluating one budget window.
type windowResult struct {
	ratio    float64
	exceeded bool
	over     string
	near     string
}

// query fetches the summary for one tenant (matched exactly) since t.
func (c *InMemoryBudgetChecker) query(ctx context.Context, tenantID string, since time.Time, msg string) (*Summary, error) {
	summary, err := c.tracker.Query(ctx, Filter{TenantID: tenantID, ExactTenant: true, Since: since})
	if err != nil {
		return nil, core.NewError("cost.budgetchecker.check", core.CodeOf(err, core.ErrInvalidInput), msg, err)
	}
	return summary, nil
}

// decide combines the window results into the most restrictive decision.
func decide(budget Budget, windows []windowResult) BudgetDecision {
	var worstExceeded, worstNear *windowResult
	maxRatio := 0.0
	for i := range windows {
		w := &windows[i]
		if w.ratio > maxRatio {
			maxRatio = w.ratio
		}
		switch {
		case w.exceeded:
			if worstExceeded == nil || w.ratio > worstExceeded.ratio {
				worstExceeded = w
			}
		case budget.AlertThreshold > 0 && w.ratio >= budget.AlertThreshold:
			if worstNear == nil || w.ratio > worstNear.ratio {
				worstNear = w
			}
		}
	}

	switch {
	case worstExceeded != nil:
		return BudgetDecision{
			Allowed:    budget.Action == BudgetActionAlert,
			Reason:     worstExceeded.over,
			UsageRatio: worstExceeded.ratio,
			Throttled:  budget.Action == BudgetActionThrottle,
		}
	case worstNear != nil:
		return BudgetDecision{Allowed: true, Reason: worstNear.near, UsageRatio: worstNear.ratio}
	default:
		return BudgetDecision{Allowed: true, UsageRatio: maxRatio}
	}
}
