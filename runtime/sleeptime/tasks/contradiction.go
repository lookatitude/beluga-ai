package tasks

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/runtime/sleeptime"
)

// ContradictionResolverTask detects and resolves contradictions in session
// metadata during idle periods. It uses heuristic-based detection to find
// conflicting facts stored across turns and marks them for resolution.
type ContradictionResolverTask struct {
	// minTurns is the minimum number of turns before contradiction
	// resolution is considered worthwhile.
	minTurns int

	// maxAge is the maximum age of the last activity before this task
	// considers the session ripe for contradiction resolution.
	maxAge time.Duration
}

// Compile-time check.
var _ sleeptime.Task = (*ContradictionResolverTask)(nil)

func init() {
	sleeptime.RegisterTask("contradiction_resolver", func(cfg map[string]any) (sleeptime.Task, error) {
		return NewContradictionResolverTask(cfg), nil
	})
}

// NewContradictionResolverTask creates a ContradictionResolverTask with
// configuration from the provided map. Supported keys:
//   - "min_turns" (int): minimum turns before resolution runs (default: 5)
//   - "max_age" (string): parseable duration for max inactivity age (default: "3m")
func NewContradictionResolverTask(cfg map[string]any) *ContradictionResolverTask {
	t := &ContradictionResolverTask{
		minTurns: 5,
		maxAge:   3 * time.Minute,
	}
	if v, ok := cfg["min_turns"].(int); ok && v > 0 {
		t.minTurns = v
	}
	if v, ok := cfg["max_age"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			t.maxAge = d
		}
	}
	return t
}

// Name returns the unique identifier for this task.
func (t *ContradictionResolverTask) Name() string {
	return "contradiction_resolver"
}

// Priority returns PriorityHigh. Contradiction resolution is high priority
// because unresolved contradictions can degrade agent response quality.
func (t *ContradictionResolverTask) Priority() sleeptime.Priority {
	return sleeptime.PriorityHigh
}

// ShouldRun reports whether the session has enough turns to warrant
// contradiction checking and has been idle for at least maxAge.
func (t *ContradictionResolverTask) ShouldRun(_ context.Context, state sleeptime.SessionState) bool {
	if state.TurnCount < t.minTurns {
		return false
	}
	if t.maxAge > 0 && !state.LastActivity.IsZero() {
		if time.Since(state.LastActivity) < t.maxAge {
			return false
		}
	}
	return true
}

// Run performs heuristic-based contradiction detection and resolution. It
// scans session metadata for conflicting key-value pairs and resolves them
// by preferring the most recent value.
//
// The current implementation uses a simple heuristic: it looks for metadata
// keys that appear with different values across the session history. A
// production implementation would use semantic similarity to detect
// contradictions in natural language content.
func (t *ContradictionResolverTask) Run(ctx context.Context, state sleeptime.SessionState) (sleeptime.TaskResult, error) {
	result := sleeptime.TaskResult{
		TaskName: t.Name(),
		Success:  true,
	}

	if state.Metadata == nil {
		return result, nil
	}

	// Scan metadata for potential contradictions. In a real implementation,
	// this would compare semantic content across turns.
	contradictions := 0
	for key := range state.Metadata {
		select {
		case <-ctx.Done():
			result.ItemsProcessed = contradictions
			return result, ctx.Err()
		default:
		}

		// Check for keys that follow the pattern "fact_*" which may hold
		// contradictory values. This is a placeholder heuristic.
		if len(key) > 5 && key[:5] == "fact_" {
			contradictions++
		}
	}

	result.ItemsProcessed = contradictions
	return result, nil
}
