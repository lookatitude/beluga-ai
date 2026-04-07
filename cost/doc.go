// Package cost provides token usage tracking and budget enforcement for the
// Beluga AI framework. It defines the Tracker interface for recording and
// querying LLM usage, a BudgetChecker interface for enforcing spending limits,
// and an in-memory Tracker implementation registered under "inmemory".
//
// # Tracker Interface
//
// The Tracker interface provides two operations:
//
//   - Record stores a Usage entry (input/output tokens, cost, model, provider).
//   - Query retrieves aggregated usage matching a Filter, returning a Summary.
//
// # BudgetChecker Interface
//
// The BudgetChecker interface evaluates a Budget against an estimated Usage and
// returns a BudgetDecision indicating whether the operation is allowed.
//
// # Registry
//
// Tracker implementations register via the standard Beluga registry pattern.
// Import a provider package for side-effect registration, then create instances
// via New.
//
// # Usage
//
// Record token usage and query totals:
//
//	import _ "github.com/lookatitude/beluga-ai/cost/providers/inmemory"
//
//	t, err := cost.New("inmemory", cost.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = t.Record(ctx, cost.Usage{
//	    InputTokens:  100,
//	    OutputTokens: 50,
//	    TotalTokens:  150,
//	    Cost:         0.0015,
//	    Model:        "gpt-4o",
//	    Provider:     "openai",
//	    Timestamp:    time.Now(),
//	})
//	summary, err := t.Query(ctx, cost.Filter{Provider: "openai"})
//
// Budget enforcement:
//
//	checker := cost.NewInMemoryBudgetChecker(t)
//	decision, err := checker.Check(ctx, cost.Budget{
//	    MaxTokensPerHour: 100_000,
//	    MaxCostPerDay:    10.0,
//	    AlertThreshold:   0.8,
//	    Action:           cost.BudgetActionReject,
//	}, cost.Usage{InputTokens: 500, OutputTokens: 200, TotalTokens: 700})
//	if !decision.Allowed {
//	    // handle budget exceeded
//	}
package cost
