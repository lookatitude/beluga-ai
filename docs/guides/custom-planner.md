# Guide: Build a Custom Planner

**Time:** ~45 minutes
**You will build:** a simple "budget-aware" planner that falls back from expensive strategies to cheap ones as the cost budget depletes.
**Prerequisites:** [First Agent guide](./first-agent.md), [Reasoning Strategies](../architecture/06-reasoning-strategies.md).

## What you'll learn

- Implementing the `agent.Planner` interface.
- Using `PlannerState.Metadata` for planner-specific state.
- Registering a planner via `agent.RegisterPlanner()`.
- Producing `ActionTool`, `ActionRespond`, `ActionFinish`, and `ActionHandoff`.

## The idea

Real agents have budget constraints. This planner:

- Starts with an expensive strategy (Reflexion).
- Monitors remaining budget via the cost plugin.
- Falls back to Self-Discover, then ReAct, as the budget drains.

## Step 1 — scaffold

```bash
mkdir -p agent/planners/budget
touch agent/planners/budget/budget.go agent/planners/budget/budget_test.go
```

## Step 2 — implement

```go
// agent/planners/budget/budget.go
//
// Package budget is a meta-planner that delegates to progressively cheaper
// sub-planners as the cost budget depletes.
package budget

import (
    "context"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/cost"
)

var _ agent.Planner = (*Planner)(nil)

// Planner delegates to one of three sub-planners based on remaining budget.
type Planner struct {
    expensive agent.Planner // Reflexion
    medium    agent.Planner // Self-Discover
    cheap     agent.Planner // ReAct

    highThreshold  float64 // fraction of budget above which we use expensive
    lowThreshold   float64 // fraction below which we fall back to cheap
}

type Config struct {
    HighThreshold float64 // e.g. 0.5 — use expensive above 50% remaining
    LowThreshold  float64 // e.g. 0.1 — use cheap below 10% remaining
}

func New(cfg Config) (*Planner, error) {
    expensive, err := agent.NewPlanner("reflexion", nil)
    if err != nil {
        return nil, err
    }
    medium, err := agent.NewPlanner("self-discover", nil)
    if err != nil {
        return nil, err
    }
    cheap, err := agent.NewPlanner("react", nil)
    if err != nil {
        return nil, err
    }
    return &Planner{
        expensive:     expensive,
        medium:        medium,
        cheap:         cheap,
        highThreshold: cfg.HighThreshold,
        lowThreshold:  cfg.LowThreshold,
    }, nil
}

func (p *Planner) pickSub(ctx context.Context) agent.Planner {
    tenant := core.GetTenant(ctx)
    remaining := cost.FractionRemaining(ctx, tenant) // 0.0 … 1.0
    switch {
    case remaining >= p.highThreshold:
        return p.expensive
    case remaining >= p.lowThreshold:
        return p.medium
    default:
        return p.cheap
    }
}

// Plan delegates to the currently-selected sub-planner.
func (p *Planner) Plan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
    sub := p.pickSub(ctx)
    actions, err := sub.Plan(ctx, state)
    if err != nil {
        return nil, err
    }
    // stamp metadata so replays can see which sub-planner ran
    if state.Metadata == nil {
        state.Metadata = make(map[string]any)
    }
    state.Metadata["budget.sub_planner"] = planerName(sub)
    return actions, nil
}

func (p *Planner) Replan(ctx context.Context, state agent.PlannerState, obs agent.Observation) ([]agent.Action, error) {
    sub := p.pickSub(ctx)
    return sub.Replan(ctx, state, obs)
}

func planerName(p agent.Planner) string {
    type named interface{ Name() string }
    if n, ok := p.(named); ok {
        return n.Name()
    }
    return "unknown"
}

func init() {
    agent.RegisterPlanner("budget", func(cfg agent.PlannerConfig) (agent.Planner, error) {
        high, _ := cfg.GetFloat("high_threshold", 0.5)
        low, _ := cfg.GetFloat("low_threshold", 0.1)
        return New(Config{HighThreshold: high, LowThreshold: low})
    })
}
```

## Step 3 — use it

```go
import _ "example.com/myrepo/agent/planners/budget"

a := agent.NewLLMAgent(
    agent.WithPersona(agent.Persona{Role: "research assistant"}),
    agent.WithLLM(model),
    agent.WithPlanner(must(agent.NewPlanner("budget", agent.PlannerConfig{
        "high_threshold": 0.6,
        "low_threshold":  0.2,
    }))),
    agent.WithTools(tools...),
)
```

## Step 4 — write a test

```go
// agent/planners/budget/budget_test.go
package budget

import (
    "context"
    "testing"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/core"
)

type fakePlanner struct {
    name   string
    called *bool
}

func (f *fakePlanner) Name() string { return f.name }
func (f *fakePlanner) Plan(ctx context.Context, _ agent.PlannerState) ([]agent.Action, error) {
    *f.called = true
    return []agent.Action{&agent.ActionRespond{Text: f.name}}, nil
}
func (f *fakePlanner) Replan(ctx context.Context, _ agent.PlannerState, _ agent.Observation) ([]agent.Action, error) {
    return nil, nil
}

func TestPicks(t *testing.T) {
    tests := []struct {
        name        string
        remaining   float64
        wantPlanner string
    }{
        {"high budget → expensive", 0.9, "expensive"},
        {"medium budget → medium", 0.3, "medium"},
        {"low budget → cheap", 0.05, "cheap"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var expCalled, medCalled, chCalled bool
            p := &Planner{
                expensive: &fakePlanner{name: "expensive", called: &expCalled},
                medium:    &fakePlanner{name: "medium", called: &medCalled},
                cheap:     &fakePlanner{name: "cheap", called: &chCalled},
                highThreshold: 0.5,
                lowThreshold:  0.1,
            }
            ctx := cost.WithFractionRemaining(context.Background(), "test", tt.remaining)
            _, err := p.Plan(core.WithTenant(ctx, "test"), agent.PlannerState{})
            if err != nil {
                t.Fatal(err)
            }
            called := map[string]bool{"expensive": expCalled, "medium": medCalled, "cheap": chCalled}
            if !called[tt.wantPlanner] {
                t.Errorf("wanted %s planner, got called=%v", tt.wantPlanner, called)
            }
        })
    }
}
```

## Checklist

- [ ] Implements `Planner` interface (Plan + Replan).
- [ ] Compile-time check `var _ agent.Planner = (*Planner)(nil)`.
- [ ] Registers via `init()` + `agent.RegisterPlanner`.
- [ ] Uses `PlannerState.Metadata` for planner-specific state, never shared mutables.
- [ ] `context.Context` first parameter.
- [ ] Test with table-driven cases.

## Common mistakes

- **Keeping per-request state in the Planner struct.** State belongs in `PlannerState.Metadata` so parallel requests don't collide.
- **Ignoring `core.GetTenant(ctx)` in budget decisions.** You'll make cross-tenant budget decisions.
- **Not propagating context to sub-planners.** `sub.Plan(ctx, state)` — never `context.Background()`.
- **Calling `cost.FractionRemaining` without a tenant.** Returns something nonsensical; always set the tenant on the context first.

## Related

- [06 — Reasoning Strategies](../architecture/06-reasoning-strategies.md) — the built-in planners.
- [05 — Agent Anatomy](../architecture/05-agent-anatomy.md#agent-types) — where the planner sits.
- [Registry + Factory pattern](../patterns/registry-factory.md).
