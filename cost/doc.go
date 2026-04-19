// Package cost provides token usage tracking and budget enforcement for the
// Beluga AI framework. It defines the [Tracker] interface for recording and
// querying LLM usage, a [BudgetChecker] interface for enforcing spending limits,
// and an in-memory [Tracker] implementation registered under "inmemory".
//
// # Tracker Interface
//
// [Tracker] provides two operations:
//
//   - Record stores a [Usage] entry (input/output tokens, cost, model, provider).
//   - Query returns an aggregated [Summary] matching a [Filter].
//
// # BudgetChecker Interface
//
// [BudgetChecker] evaluates a [Budget] against estimated [Usage] and returns a
// [BudgetDecision] indicating whether the operation is allowed.
//
// Budget enforcement checks two rolling windows:
//
//   - MaxTokensPerHour: total tokens in the past hour.
//   - MaxCostPerDay: total USD cost in the past 24 hours.
//
// When a threshold is reached, the configured [BudgetAction] determines the
// response: [BudgetActionReject] denies the request, [BudgetActionThrottle]
// signals the caller to slow down, and [BudgetActionAlert] allows the request
// but sets [BudgetDecision.Reason].
//
// # Registry
//
// Tracker implementations register via the standard Beluga registry pattern.
// The "inmemory" backend is registered automatically. Additional backends
// register in their own init() functions.
//
// # Usage
//
// Record token usage and query totals:
//
//	import "github.com/lookatitude/beluga-ai/v2/cost"
//
//	tracker, err := cost.New("inmemory", cost.Config{MaxEntries: 100_000})
//	if err != nil {
//	    return err
//	}
//	err = tracker.Record(ctx, cost.Usage{
//	    InputTokens:  512,
//	    OutputTokens: 128,
//	    TotalTokens:  640,
//	    Cost:         0.0048,
//	    Model:        "gpt-4o",
//	    Provider:     "openai",
//	    TenantID:     "tenant-a",
//	    Timestamp:    time.Now(),
//	})
//	if err != nil {
//	    return err
//	}
//	summary, err := tracker.Query(ctx, cost.Filter{
//	    Provider: "openai",
//	    Since:    time.Now().Add(-24 * time.Hour),
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("calls: %d  cost: $%.4f\n", summary.EntryCount, summary.TotalCost)
//
// Budget enforcement:
//
//	checker := cost.NewInMemoryBudgetChecker(tracker)
//	decision, err := checker.Check(ctx, cost.Budget{
//	    MaxTokensPerHour: 100_000,
//	    MaxCostPerDay:    10.0,
//	    AlertThreshold:   0.8,
//	    Action:           cost.BudgetActionReject,
//	}, cost.Usage{TotalTokens: 700, Cost: 0.005, TenantID: "tenant-a"})
//	if err != nil {
//	    return err
//	}
//	if !decision.Allowed {
//	    return fmt.Errorf("budget exceeded: %s", decision.Reason)
//	}
//
// # Extension
//
// Implement [Tracker] and register in init():
//
//	func init() {
//	    cost.Register("postgres", func(cfg cost.Config) (cost.Tracker, error) {
//	        return newPostgresTracker(cfg)
//	    })
//	}
//
// Users import your package for side effects:
//
//	import _ "myorg/cost/postgres"
//
// # Related packages
//
//   - [github.com/lookatitude/beluga-ai/v2/runtime/plugins] — CostTracking plugin
//   - [github.com/lookatitude/beluga-ai/v2/audit] — audit logging
//   - [github.com/lookatitude/beluga-ai/v2/o11y] — OpenTelemetry metrics
package cost
