package teambuilder

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/agent"
)

// Hooks provides optional callbacks for observing and augmenting team
// formation. All fields are optional; nil fields are skipped.
type Hooks struct {
	// OnTeamFormed is called after a team has been successfully built.
	OnTeamFormed func(ctx context.Context, task string, agents []agent.Agent)

	// OnAgentSelected is called for each agent selected during team formation,
	// along with the score assigned by the selector.
	OnAgentSelected func(ctx context.Context, task string, entry PoolEntry, score float64)

	// OnSelectionFailed is called when agent selection fails.
	OnSelectionFailed func(ctx context.Context, task string, err error)
}

// ComposeHooks merges multiple Hooks into one. Each hook function chains
// the callbacks in order: earlier hooks run first.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnTeamFormed:      composeOnTeamFormed(hooks),
		OnAgentSelected:   composeOnAgentSelected(hooks),
		OnSelectionFailed: composeOnSelectionFailed(hooks),
	}
}

// composeOnTeamFormed chains the non-nil OnTeamFormed callbacks in hooks, in
// order. It returns nil when none are set.
func composeOnTeamFormed(hooks []Hooks) func(ctx context.Context, task string, agents []agent.Agent) {
	var fns []func(ctx context.Context, task string, agents []agent.Agent)
	for _, h := range hooks {
		if h.OnTeamFormed != nil {
			fns = append(fns, h.OnTeamFormed)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, task string, agents []agent.Agent) {
		for _, fn := range fns {
			fn(ctx, task, agents)
		}
	}
}

// composeOnAgentSelected chains the non-nil OnAgentSelected callbacks in
// hooks, in order. It returns nil when none are set.
func composeOnAgentSelected(hooks []Hooks) func(ctx context.Context, task string, entry PoolEntry, score float64) {
	var fns []func(ctx context.Context, task string, entry PoolEntry, score float64)
	for _, h := range hooks {
		if h.OnAgentSelected != nil {
			fns = append(fns, h.OnAgentSelected)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, task string, entry PoolEntry, score float64) {
		for _, fn := range fns {
			fn(ctx, task, entry, score)
		}
	}
}

// composeOnSelectionFailed chains the non-nil OnSelectionFailed callbacks in
// hooks, in order. It returns nil when none are set.
func composeOnSelectionFailed(hooks []Hooks) func(ctx context.Context, task string, err error) {
	var fns []func(ctx context.Context, task string, err error)
	for _, h := range hooks {
		if h.OnSelectionFailed != nil {
			fns = append(fns, h.OnSelectionFailed)
		}
	}
	if len(fns) == 0 {
		return nil
	}
	return func(ctx context.Context, task string, err error) {
		for _, fn := range fns {
			fn(ctx, task, err)
		}
	}
}
