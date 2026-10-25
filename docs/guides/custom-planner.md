# Guide: Build a Custom Planner

**Time:** ~45 minutes
**You will build:** a simple "budget-aware" planner that falls back from expensive strategies to cheap ones as the cost budget depletes.
**Prerequisites:** [First Agent guide](./first-agent.md), [Reasoning Strategies](../architecture/06-reasoning-strategies.md).

## What you'll learn

- Implementing the `agent.Planner` interface.
- Using `Action.Metadata` and `PlannerState.Metadata` for planner-specific state.
- Registering a planner via `agent.RegisterPlanner()`.
- Returning `agent.Action` values (`Type` is one of `ActionTool`, `ActionRespond`, `ActionFinish`, `ActionHandoff`).

## The idea

Real agents have budget constraints. This planner:

- Starts with an expensive strategy (Reflexion).
- Asks a budget source how much of the tenant's budget is left.
- Falls back to Self-Discover, then ReAct, as the budget drains.

## Step 1 — scaffold

```bash
mkdir -p planners/budget
touch planners/budget/budget.go planners/budget/budget_test.go
```

## Step 2 — implement

```go
// planners/budget/budget.go
//
// Package budget is a meta-planner that delegates to progressively cheaper
// sub-planners as the tenant's cost budget depletes.
package budget

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/cost"
)

var _ agent.Planner = (*Planner)(nil)

// BudgetSource reports the fraction (0.0 to 1.0) of a tenant's budget that is
// still available.
type BudgetSource interface {
	FractionRemaining(ctx context.Context, tenant core.TenantID) (float64, error)
}

// CostBudget is a BudgetSource backed by a cost.BudgetChecker.
type CostBudget struct {
	Checker cost.BudgetChecker
	Budget  cost.Budget
}

// FractionRemaining checks a zero-usage estimate against the budget and
// converts the reported usage ratio into the remaining fraction.
func (c CostBudget) FractionRemaining(ctx context.Context, tenant core.TenantID) (float64, error) {
	d, err := c.Checker.Check(ctx, c.Budget, cost.Usage{TenantID: string(tenant)})
	if err != nil {
		return 0, err
	}
	return max(0, 1-d.UsageRatio), nil
}

// Planner delegates to one of three sub-planners based on remaining budget.
type Planner struct {
	expensive agent.Planner // Reflexion
	medium    agent.Planner // Self-Discover
	cheap     agent.Planner // ReAct
	budget    BudgetSource

	highThreshold float64 // fraction of budget above which we use expensive
	lowThreshold  float64 // fraction below which we fall back to cheap
}

// Config configures the budget planner.
type Config struct {
	Budget        BudgetSource
	HighThreshold float64 // e.g. 0.5: use expensive above 50% remaining
	LowThreshold  float64 // e.g. 0.1: use cheap below 10% remaining
}

// New builds the three sub-planners from the registry. pc carries the LLM
// they reason with.
func New(pc agent.PlannerConfig, cfg Config) (*Planner, error) {
	if cfg.Budget == nil {
		return nil, core.Errorf(core.ErrInvalidInput, "budget: Config.Budget is required")
	}
	expensive, err := agent.NewPlanner("reflexion", pc)
	if err != nil {
		return nil, err
	}
	medium, err := agent.NewPlanner("self-discover", pc)
	if err != nil {
		return nil, err
	}
	cheap, err := agent.NewPlanner("react", pc)
	if err != nil {
		return nil, err
	}
	return &Planner{
		expensive:     expensive,
		medium:        medium,
		cheap:         cheap,
		budget:        cfg.Budget,
		highThreshold: cfg.HighThreshold,
		lowThreshold:  cfg.LowThreshold,
	}, nil
}

func (p *Planner) pickSub(ctx context.Context) (agent.Planner, string, error) {
	remaining, err := p.budget.FractionRemaining(ctx, core.GetTenant(ctx))
	if err != nil {
		return nil, "", err
	}
	switch {
	case remaining >= p.highThreshold:
		return p.expensive, "reflexion", nil
	case remaining >= p.lowThreshold:
		return p.medium, "self-discover", nil
	default:
		return p.cheap, "react", nil
	}
}

// Plan delegates to the currently selected sub-planner.
func (p *Planner) Plan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	sub, name, err := p.pickSub(ctx)
	if err != nil {
		return nil, err
	}
	actions, err := sub.Plan(ctx, state)
	return stamp(actions, name), err
}

// Replan re-checks the budget, so a run can step down mid-conversation.
func (p *Planner) Replan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	sub, name, err := p.pickSub(ctx)
	if err != nil {
		return nil, err
	}
	actions, err := sub.Replan(ctx, state)
	return stamp(actions, name), err
}

// stamp records which sub-planner produced each action, so traces and
// replays can see it.
func stamp(actions []agent.Action, name string) []agent.Action {
	for i := range actions {
		if actions[i].Metadata == nil {
			actions[i].Metadata = make(map[string]any)
		}
		actions[i].Metadata["budget.sub_planner"] = name
	}
	return actions
}

func init() {
	agent.RegisterPlanner("budget", func(pc agent.PlannerConfig) (agent.Planner, error) {
		cfg := Config{HighThreshold: 0.5, LowThreshold: 0.1}
		if v, ok := pc.Extra["budget"].(BudgetSource); ok {
			cfg.Budget = v
		}
		if v, ok := pc.Extra["high_threshold"].(float64); ok {
			cfg.HighThreshold = v
		}
		if v, ok := pc.Extra["low_threshold"].(float64); ok {
			cfg.LowThreshold = v
		}
		return New(pc, cfg)
	})
}
```

`agent.PlannerConfig` has two fields: `LLM` (the model the planner reasons with) and `Extra` (planner-specific settings). The built-in planners return an error when `LLM` is nil.

## Step 3 — use it

```go
import _ "example.com/myrepo/planners/budget"

tracker := cost.NewInMemoryTracker()
planner, err := agent.NewPlanner("budget", agent.PlannerConfig{
	LLM: model,
	Extra: map[string]any{
		"budget": budget.CostBudget{
			Checker: cost.NewInMemoryBudgetChecker(tracker),
			Budget:  cost.Budget{MaxCostPerDay: 5.00},
		},
		"high_threshold": 0.6,
		"low_threshold":  0.2,
	},
})
if err != nil {
	log.Fatal(err)
}

a := agent.New("research-assistant",
	agent.WithPersona(agent.Persona{Role: "research assistant"}),
	agent.WithLLM(model),
	agent.WithPlanner(planner),
	agent.WithTools(tools),
)
```

Record each LLM call's usage in `tracker` (`tracker.Record(ctx, cost.Usage{...})`) so the budget source sees it.

## Step 4 — write a test

```go
// planners/budget/budget_test.go
package budget

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
)

type fakePlanner struct {
	name   string
	called *bool
}

func (f *fakePlanner) Plan(_ context.Context, _ agent.PlannerState) ([]agent.Action, error) {
	*f.called = true
	return []agent.Action{{Type: agent.ActionRespond, Message: f.name}}, nil
}

func (f *fakePlanner) Replan(_ context.Context, _ agent.PlannerState) ([]agent.Action, error) {
	return nil, nil
}

type fixedBudget float64

func (b fixedBudget) FractionRemaining(context.Context, core.TenantID) (float64, error) {
	return float64(b), nil
}

func TestPicks(t *testing.T) {
	tests := []struct {
		name        string
		remaining   float64
		wantPlanner string
	}{
		{"high budget uses expensive", 0.9, "reflexion"},
		{"medium budget uses medium", 0.3, "self-discover"},
		{"low budget uses cheap", 0.05, "react"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var expCalled, medCalled, chCalled bool
			p := &Planner{
				expensive:     &fakePlanner{name: "expensive", called: &expCalled},
				medium:        &fakePlanner{name: "medium", called: &medCalled},
				cheap:         &fakePlanner{name: "cheap", called: &chCalled},
				budget:        fixedBudget(tt.remaining),
				highThreshold: 0.5,
				lowThreshold:  0.1,
			}
			ctx := core.WithTenant(context.Background(), "test")
			actions, err := p.Plan(ctx, agent.PlannerState{})
			if err != nil {
				t.Fatal(err)
			}
			if got := actions[0].Metadata["budget.sub_planner"]; got != tt.wantPlanner {
				t.Errorf("sub_planner = %v, want %s", got, tt.wantPlanner)
			}
		})
	}
}
```

## Checklist

- [ ] Implements the `Planner` interface: `Plan(ctx, state)` and `Replan(ctx, state)`.
- [ ] Compile-time check `var _ agent.Planner = (*Planner)(nil)`.
- [ ] Registers via `init()` + `agent.RegisterPlanner`.
- [ ] Keeps per-request state in `PlannerState.Metadata` or `Action.Metadata`, never in the planner struct.
- [ ] `context.Context` first parameter.
- [ ] Test with table-driven cases.

## Common mistakes

- **Keeping per-request state in the Planner struct.** State belongs in `PlannerState.Metadata` so parallel requests don't collide.
- **Ignoring `core.GetTenant(ctx)` in budget decisions.** You'll make cross-tenant budget decisions.
- **Not propagating context to sub-planners.** `sub.Plan(ctx, state)` — never `context.Background()`.
- **Checking a budget without a tenant.** `core.GetTenant` returns an empty ID, and an empty `cost.Filter.TenantID` matches every tenant. Always set the tenant on the context first.

## Related

- [06 — Reasoning Strategies](../architecture/06-reasoning-strategies.md) — the built-in planners.
- [05 — Agent Anatomy](../architecture/05-agent-anatomy.md#agent-types) — where the planner sits.
- [Registry + Factory pattern](../patterns/registry-factory.md).
