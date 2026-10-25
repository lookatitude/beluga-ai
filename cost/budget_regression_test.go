package cost

import (
	"context"
	"testing"
	"time"
)

func record(t *testing.T, tr *InMemoryTracker, u Usage) {
	t.Helper()
	if u.Timestamp.IsZero() {
		u.Timestamp = time.Now()
	}
	if err := tr.Record(context.Background(), u); err != nil {
		t.Fatal(err)
	}
}

// TestBudgetCheck_AlertDoesNotSkipCostLimit is a regression test for
// FWC-15(a): the hourly-token alert branch returned early, so a request over
// the daily cost budget was allowed.
func TestBudgetCheck_AlertDoesNotSkipCostLimit(t *testing.T) {
	tr := NewInMemoryTracker()
	record(t, tr, Usage{InputTokens: 80, Cost: 10})
	c := NewInMemoryBudgetChecker(tr)
	d, err := c.Check(context.Background(), Budget{
		MaxTokensPerHour: 100, MaxCostPerDay: 5, AlertThreshold: 0.5, Action: BudgetActionReject,
	}, Usage{TotalTokens: 1, Cost: 1})
	if err != nil {
		t.Fatal(err)
	}
	if d.Allowed || d.Reason != "daily cost budget exceeded" {
		t.Errorf("decision = %+v, want rejected for cost", d)
	}
}

// TestBudgetCheck_DefaultTenantIsolated is a regression test for FWC-15(b):
// the default tenant's budget counted every tenant's spend.
func TestBudgetCheck_DefaultTenantIsolated(t *testing.T) {
	tr := NewInMemoryTracker()
	record(t, tr, Usage{TenantID: "other", Cost: 100})
	c := NewInMemoryBudgetChecker(tr)
	d, err := c.Check(context.Background(), Budget{MaxCostPerDay: 10}, Usage{Cost: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allowed {
		t.Errorf("default tenant charged for another tenant's spend: %+v", d)
	}
	// Plain Filter{} still aggregates every tenant.
	s, _ := tr.Query(context.Background(), Filter{})
	if s.TotalCost != 100 {
		t.Errorf("Filter{} total = %v, want 100", s.TotalCost)
	}
}

// TestBudgetCheck_ThrottleAndEmptyAction is a regression test for
// FWC-15(c): Throttle was handled exactly like Reject.
func TestBudgetCheck_ThrottleAndEmptyAction(t *testing.T) {
	tr := NewInMemoryTracker()
	record(t, tr, Usage{InputTokens: 100})
	c := NewInMemoryBudgetChecker(tr)
	ctx := context.Background()
	d, _ := c.Check(ctx, Budget{MaxTokensPerHour: 50, Action: BudgetActionThrottle}, Usage{TotalTokens: 1})
	if d.Allowed || !d.Throttled {
		t.Errorf("throttle decision = %+v, want not allowed + Throttled", d)
	}
	d, _ = c.Check(ctx, Budget{MaxTokensPerHour: 50}, Usage{TotalTokens: 1})
	if d.Allowed || d.Throttled {
		t.Errorf("empty action decision = %+v, want plain reject", d)
	}
	d, _ = c.Check(ctx, Budget{MaxTokensPerHour: 50, Action: BudgetActionAlert}, Usage{TotalTokens: 1})
	if !d.Allowed {
		t.Errorf("alert decision = %+v, want allowed", d)
	}
}

// TestBudgetCheck_UsesInputOutputWhenTotalUnset is a regression test for
// FWC-15(d): only estimated.TotalTokens counted toward the projection.
func TestBudgetCheck_UsesInputOutputWhenTotalUnset(t *testing.T) {
	c := NewInMemoryBudgetChecker(NewInMemoryTracker())
	d, _ := c.Check(context.Background(), Budget{MaxTokensPerHour: 100}, Usage{InputTokens: 80, OutputTokens: 40})
	if d.Allowed {
		t.Errorf("decision = %+v, want rejected (120 > 100 tokens)", d)
	}
	d, _ = c.Check(context.Background(), Budget{MaxTokensPerHour: 100, AlertThreshold: 0.5}, Usage{InputTokens: 30, OutputTokens: 30})
	if !d.Allowed || d.Reason != "approaching hourly token budget limit" {
		t.Errorf("decision = %+v, want alert", d)
	}
	if got := (Usage{TotalTokens: 7, InputTokens: 1}).Tokens(); got != 7 {
		t.Errorf("Tokens() = %d", got)
	}
}
