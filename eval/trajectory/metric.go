package trajectory

import (
	"context"
	"time"
)

// TrajectoryMetric is the interface that all trajectory evaluation metrics
// implement. A TrajectoryMetric scores a complete agent trajectory and returns
// detailed per-step and overall scores.
type TrajectoryMetric interface {
	// Name returns the unique name of this metric (e.g., "tool_selection").
	Name() string

	// ScoreTrajectory evaluates a single trajectory and returns a detailed score.
	// The overall score should be in [0.0, 1.0] where higher is better.
	ScoreTrajectory(ctx context.Context, t Trajectory) (*TrajectoryScore, error)
}

// TrajectoryScore holds the evaluation result for a single trajectory.
type TrajectoryScore struct {
	// Overall is the aggregate score in [0.0, 1.0].
	Overall float64 `json:"overall"`
	// StepScores provides per-step scoring details.
	StepScores []StepScore `json:"step_scores,omitempty"`
	// Details holds metric-specific data (e.g., precision, recall, F1).
	Details map[string]any `json:"details,omitempty"`
}

// StepScore holds the score for an individual step.
type StepScore struct {
	// StepIndex identifies which step this score applies to.
	StepIndex int `json:"step_index"`
	// Score is the score for this step in [0.0, 1.0].
	Score float64 `json:"score"`
	// Reason explains why this score was assigned.
	Reason string `json:"reason,omitempty"`
}

// TrajectoryResult holds evaluation scores for a single trajectory across
// all configured metrics.
type TrajectoryResult struct {
	// TrajectoryID identifies the trajectory that was evaluated.
	TrajectoryID string `json:"trajectory_id"`
	// Scores maps metric names to their trajectory scores.
	Scores map[string]*TrajectoryScore `json:"scores"`
}

// Report is the aggregate result of running trajectory evaluation.
type Report struct {
	// Trajectories contains per-trajectory results.
	Trajectories []TrajectoryResult `json:"trajectories"`
	// Aggregate contains the average overall score for each metric.
	Aggregate map[string]float64 `json:"aggregate"`
	// Timestamp is when the evaluation was run.
	Timestamp time.Time `json:"timestamp"`
}
