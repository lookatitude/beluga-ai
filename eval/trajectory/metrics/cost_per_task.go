package metrics

import (
	"context"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/eval/trajectory"
)

func init() {
	trajectory.Register("cost_per_task", func(cfg map[string]any) (trajectory.TrajectoryMetric, error) {
		var opts []CostPerTaskOption
		if v, ok := cfg["max_budget"]; ok {
			switch b := v.(type) {
			case float64:
				opts = append(opts, WithMaxBudget(b))
			case int:
				opts = append(opts, WithMaxBudget(float64(b)))
			}
		}
		return NewCostPerTask(opts...), nil
	})
}

// Compile-time interface check.
var _ trajectory.TrajectoryMetric = (*CostPerTask)(nil)

// CostPerTaskOption configures a CostPerTask metric.
type CostPerTaskOption func(*CostPerTask)

// WithMaxBudget sets the maximum budget in dollars. The score is computed as
// 1.0 - (actual_cost / max_budget), clamped to [0, 1].
func WithMaxBudget(budget float64) CostPerTaskOption {
	return func(c *CostPerTask) {
		if budget > 0 {
			c.maxBudget = budget
		}
	}
}

// CostPerTask evaluates the cost efficiency of an agent's trajectory.
// It reads total cost from trajectory metadata (key "total_cost") or
// computes it from per-step token metadata. The score is
// 1.0 - (actual_cost / max_budget), clamped to [0, 1].
type CostPerTask struct {
	maxBudget float64
}

// NewCostPerTask creates a new CostPerTask metric with the given options.
func NewCostPerTask(opts ...CostPerTaskOption) *CostPerTask {
	c := &CostPerTask{
		maxBudget: 1.0, // $1.00 default budget
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns "cost_per_task".
func (c *CostPerTask) Name() string { return "cost_per_task" }

// ScoreTrajectory computes the cost score. It first checks for a "total_cost"
// key in trajectory metadata, then falls back to summing per-step
// "step_cost" metadata values.
func (c *CostPerTask) ScoreTrajectory(_ context.Context, t trajectory.Trajectory) (*trajectory.TrajectoryScore, error) {
	totalCost, err := c.extractCost(t)
	if err != nil {
		return nil, err
	}

	score := 1.0 - (totalCost / c.maxBudget)
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return &trajectory.TrajectoryScore{
		Overall: score,
		Details: map[string]any{
			"total_cost": totalCost,
			"max_budget": c.maxBudget,
		},
	}, nil
}

// extractCost reads the total cost from trajectory metadata or sums per-step costs.
func (c *CostPerTask) extractCost(t trajectory.Trajectory) (float64, error) {
	// Try trajectory-level metadata first.
	if t.Metadata != nil {
		if v, ok := t.Metadata["total_cost"]; ok {
			cost, err := toFloat(v)
			if err != nil {
				return 0, core.Errorf(core.ErrInvalidInput, "cost_per_task: invalid total_cost: %w", err)
			}
			return cost, nil
		}
	}

	// Fall back to summing per-step costs.
	var total float64
	for _, step := range t.Steps {
		if step.Metadata != nil {
			if v, ok := step.Metadata["step_cost"]; ok {
				cost, err := toFloat(v)
				if err != nil {
					return 0, core.Errorf(core.ErrInvalidInput, "cost_per_task: invalid step_cost at step %d: %w", step.Index, err)
				}
				total += cost
			}
		}
	}

	return total, nil
}

// toFloat converts a numeric value to float64.
func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, core.Errorf(core.ErrInvalidInput, "expected numeric value, got %T", v)
	}
}
