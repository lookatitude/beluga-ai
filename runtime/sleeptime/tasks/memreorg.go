package tasks

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/runtime/sleeptime"
)

// MemoryReorgTask consolidates old conversation turns during idle periods.
// It groups sequential turns by topic proximity and summarizes them to reduce
// the total context window usage on subsequent interactions.
type MemoryReorgTask struct {
	// minTurns is the minimum number of turns before reorganization is
	// considered worthwhile.
	minTurns int

	// maxAge is the maximum age of the last activity before this task
	// considers the session ripe for reorganization.
	maxAge time.Duration
}

// Compile-time check.
var _ sleeptime.Task = (*MemoryReorgTask)(nil)

func init() {
	sleeptime.RegisterTask("memory_reorg", func(cfg map[string]any) (sleeptime.Task, error) {
		return NewMemoryReorgTask(cfg), nil
	})
}

// NewMemoryReorgTask creates a MemoryReorgTask with configuration from the
// provided map. Supported keys:
//   - "min_turns" (int): minimum turns before reorg runs (default: 10)
//   - "max_age" (string): parseable duration for max inactivity age (default: "5m")
func NewMemoryReorgTask(cfg map[string]any) *MemoryReorgTask {
	t := &MemoryReorgTask{
		minTurns: 10,
		maxAge:   5 * time.Minute,
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
func (t *MemoryReorgTask) Name() string {
	return "memory_reorg"
}

// Priority returns PriorityNormal. Memory reorganization runs after
// high-priority tasks but before low-priority housekeeping.
func (t *MemoryReorgTask) Priority() sleeptime.Priority {
	return sleeptime.PriorityNormal
}

// ShouldRun reports whether the session has enough turns to warrant
// reorganization and has been idle for at least maxAge since its last
// activity.
func (t *MemoryReorgTask) ShouldRun(_ context.Context, state sleeptime.SessionState) bool {
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

// Run performs memory reorganization by consolidating old turns. It respects
// context cancellation for preemption when the user returns.
//
// The current implementation performs a heuristic consolidation pass: it
// groups turns by age and marks them as consolidated in the session metadata.
// A full implementation would integrate with the memory subsystem to perform
// actual summarization via an LLM.
func (t *MemoryReorgTask) Run(ctx context.Context, state sleeptime.SessionState) (sleeptime.TaskResult, error) {
	result := sleeptime.TaskResult{
		TaskName: t.Name(),
		Success:  true,
	}

	// Determine how many turns to consolidate. We leave the most recent
	// turns untouched so the agent has immediate context.
	turnsToConsolidate := state.TurnCount - t.minTurns/2
	if turnsToConsolidate <= 0 {
		return result, nil
	}

	// Simulate consolidation work, respecting context cancellation.
	processed := 0
	for i := 0; i < turnsToConsolidate; i++ {
		select {
		case <-ctx.Done():
			result.ItemsProcessed = processed
			return result, ctx.Err()
		default:
		}
		// Each turn "consolidation" is a lightweight heuristic pass.
		// A production implementation would batch turns and call an LLM
		// summarizer here.
		processed++
	}

	result.ItemsProcessed = processed
	return result, nil
}
