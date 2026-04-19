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
	var composed Hooks

	var teamFormedFns []func(ctx context.Context, task string, agents []agent.Agent)
	var agentSelectedFns []func(ctx context.Context, task string, entry PoolEntry, score float64)
	var selectionFailedFns []func(ctx context.Context, task string, err error)

	for _, h := range hooks {
		if h.OnTeamFormed != nil {
			teamFormedFns = append(teamFormedFns, h.OnTeamFormed)
		}
		if h.OnAgentSelected != nil {
			agentSelectedFns = append(agentSelectedFns, h.OnAgentSelected)
		}
		if h.OnSelectionFailed != nil {
			selectionFailedFns = append(selectionFailedFns, h.OnSelectionFailed)
		}
	}

	if len(teamFormedFns) > 0 {
		composed.OnTeamFormed = func(ctx context.Context, task string, agents []agent.Agent) {
			for _, fn := range teamFormedFns {
				fn(ctx, task, agents)
			}
		}
	}
	if len(agentSelectedFns) > 0 {
		composed.OnAgentSelected = func(ctx context.Context, task string, entry PoolEntry, score float64) {
			for _, fn := range agentSelectedFns {
				fn(ctx, task, entry, score)
			}
		}
	}
	if len(selectionFailedFns) > 0 {
		composed.OnSelectionFailed = func(ctx context.Context, task string, err error) {
			for _, fn := range selectionFailedFns {
				fn(ctx, task, err)
			}
		}
	}

	return composed
}
