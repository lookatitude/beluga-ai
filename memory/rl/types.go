package rl

import (
	"context"
	"time"
)

// Step records a single policy decision within an episode. Each step captures
// the observation features, the action taken, the policy's confidence, and
// the wall-clock time of the decision.
type Step struct {
	// Features is the observation the policy received.
	Features PolicyFeatures

	// Action is the action selected by the policy.
	Action MemoryAction

	// Confidence is the policy's reported confidence in [0, 1].
	Confidence float64

	// Timestamp records when the decision was made.
	Timestamp time.Time
}

// Episode groups a sequence of Steps that belong to a single task or
// conversation, along with an outcome used for reward computation.
type Episode struct {
	// ID uniquely identifies the episode.
	ID string

	// Steps is the ordered sequence of decisions in this episode.
	Steps []Step

	// Outcome holds task-level success information used by RewardFunc.
	// For example, a boolean success flag or a numeric score.
	Outcome any

	// StartTime is when the episode began.
	StartTime time.Time

	// EndTime is when the episode ended.
	EndTime time.Time
}

// Hooks provides optional callbacks for observing RL memory decisions and
// episode lifecycle events. All fields are optional; nil hooks are skipped.
type Hooks struct {
	// OnDecision is called after the policy makes a decision. It receives
	// the features, chosen action, and confidence. Returning an error
	// prevents the action from being executed.
	OnDecision func(ctx context.Context, features PolicyFeatures, action MemoryAction, confidence float64) error

	// OnEpisodeEnd is called when an episode is finalized via the
	// TrajectoryCollector. It receives the completed episode.
	OnEpisodeEnd func(ctx context.Context, episode Episode)
}
