package rl

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Rewarder computes per-step rewards for an episode. The returned slice
// must have the same length as episode.Steps. Implementations map task-level
// outcomes to step-level credit assignments.
type Rewarder interface {
	// Compute returns a reward value for each step in the episode.
	// Positive rewards encourage the action; negative rewards discourage it.
	Compute(ctx context.Context, episode Episode) ([]float64, error)
}

// SimpleReward implements Rewarder with a binary success/failure model.
// If the episode outcome is truthy (bool true or numeric > 0), every step
// receives SuccessReward; otherwise every step receives FailureReward.
type SimpleReward struct {
	// SuccessReward is the per-step reward on episode success. Default: 1.0.
	SuccessReward float64

	// FailureReward is the per-step reward on episode failure. Default: -1.0.
	FailureReward float64
}

// NewSimpleReward creates a SimpleReward with default values.
func NewSimpleReward() *SimpleReward {
	return &SimpleReward{
		SuccessReward: 1.0,
		FailureReward: -1.0,
	}
}

// Compute implements Rewarder. It interprets the episode outcome as a
// boolean or numeric success signal and assigns uniform rewards to all steps.
func (r *SimpleReward) Compute(_ context.Context, episode Episode) ([]float64, error) {
	if len(episode.Steps) == 0 {
		return nil, nil
	}

	success, err := interpretOutcome(episode.Outcome)
	if err != nil {
		return nil, err
	}

	reward := r.FailureReward
	if success {
		reward = r.SuccessReward
	}

	rewards := make([]float64, len(episode.Steps))
	for i := range rewards {
		rewards[i] = reward
	}
	return rewards, nil
}

// interpretOutcome converts an outcome value to a boolean success signal.
func interpretOutcome(outcome any) (bool, error) {
	switch v := outcome.(type) {
	case bool:
		return v, nil
	case int:
		return v > 0, nil
	case int64:
		return v > 0, nil
	case float64:
		return v > 0, nil
	case float32:
		return v > 0, nil
	case nil:
		return false, nil
	default:
		return false, core.NewError(
			"rl.reward",
			core.ErrInvalidInput,
			"unsupported outcome type; expected bool, int, or float",
			nil,
		)
	}
}

// Compile-time interface check.
var _ Rewarder = (*SimpleReward)(nil)
