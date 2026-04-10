package teambuilder

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/runtime"
)

// TeamBuilder composes runtime.Team instances dynamically by selecting agents
// from an AgentPool based on a task description and configured constraints.
type TeamBuilder struct {
	pool      *AgentPool
	selector  Selector
	maxAgents int
	teamID    string
	pattern   func([]agent.Agent) runtime.OrchestrationPattern
	hooks     Hooks
}

// BuilderOption configures a TeamBuilder.
type BuilderOption func(*TeamBuilder)

// WithSelector sets the agent selection strategy.
func WithSelector(s Selector) BuilderOption {
	return func(tb *TeamBuilder) {
		if s != nil {
			tb.selector = s
		}
	}
}

// WithMaxAgents sets the maximum number of agents to include in a team.
// A value of 0 means no limit.
func WithMaxAgents(n int) BuilderOption {
	return func(tb *TeamBuilder) {
		if n >= 0 {
			tb.maxAgents = n
		}
	}
}

// WithTeamID sets a custom ID prefix for built teams.
func WithTeamID(id string) BuilderOption {
	return func(tb *TeamBuilder) {
		tb.teamID = id
	}
}

// WithPatternFactory sets a function that creates an OrchestrationPattern
// from the selected agents. If not set, PipelinePattern is used by default
// (via runtime.NewTeam's default behavior).
func WithPatternFactory(f func([]agent.Agent) runtime.OrchestrationPattern) BuilderOption {
	return func(tb *TeamBuilder) {
		tb.pattern = f
	}
}

// WithHooks sets the lifecycle hooks for team formation.
func WithHooks(h Hooks) BuilderOption {
	return func(tb *TeamBuilder) {
		tb.hooks = h
	}
}

// NewTeamBuilder creates a TeamBuilder backed by the given pool. If no
// selector is configured, KeywordSelector is used as the default.
func NewTeamBuilder(pool *AgentPool, opts ...BuilderOption) *TeamBuilder {
	tb := &TeamBuilder{
		pool:      pool,
		selector:  NewKeywordSelector(),
		maxAgents: 0, // no limit
		teamID:    "dynamic-team",
	}
	for _, opt := range opts {
		opt(tb)
	}
	return tb
}

// Build dynamically forms a runtime.Team by selecting agents from the pool
// that are suitable for the given task. Returns an error if the pool is nil,
// no agents are selected, or the selector fails.
func (tb *TeamBuilder) Build(ctx context.Context, task string) (*runtime.Team, error) {
	if tb.pool == nil {
		return nil, core.NewError("teambuilder.build", core.ErrInvalidInput,
			"agent pool must not be nil", nil)
	}

	if task == "" {
		return nil, core.NewError("teambuilder.build", core.ErrInvalidInput,
			"task must not be empty", nil)
	}

	candidates := tb.pool.List()
	if len(candidates) == 0 {
		return nil, core.NewError("teambuilder.build", core.ErrNotFound,
			"agent pool is empty", nil)
	}

	selected, err := tb.selector.Select(ctx, task, candidates)
	if err != nil {
		if tb.hooks.OnSelectionFailed != nil {
			tb.hooks.OnSelectionFailed(ctx, task, err)
		}
		return nil, core.NewError("teambuilder.build", core.ErrToolFailed,
			"agent selection failed", err)
	}

	if len(selected) == 0 {
		err := core.NewError("teambuilder.build", core.ErrNotFound,
			fmt.Sprintf("no suitable agents found for task: %s", truncate(task, 100)), nil)
		if tb.hooks.OnSelectionFailed != nil {
			tb.hooks.OnSelectionFailed(ctx, task, err)
		}
		return nil, err
	}

	// Apply maxAgents limit.
	if tb.maxAgents > 0 && len(selected) > tb.maxAgents {
		selected = selected[:tb.maxAgents]
	}

	// Fire OnAgentSelected hooks.
	//
	// NOTE: the score passed here is a positional approximation derived from
	// rank — not the concrete relevance score the selector computed. The
	// Selector interface currently only returns []PoolEntry, so the real
	// score is not available at this layer. Hook consumers should treat this
	// value as ordinal signal, not absolute relevance. A future API revision
	// could extend Selector to return scored entries.
	if tb.hooks.OnAgentSelected != nil {
		for i, entry := range selected {
			score := 1.0 - float64(i)*0.1
			if score < 0.1 {
				score = 0.1
			}
			tb.hooks.OnAgentSelected(ctx, task, entry, score)
		}
	}

	// Extract agents from selected entries.
	agents := make([]agent.Agent, len(selected))
	for i, e := range selected {
		agents[i] = e.Agent
	}

	// Build team options.
	teamOpts := []runtime.TeamOption{
		runtime.WithTeamID(tb.teamID),
		runtime.WithAgents(agents...),
	}

	if tb.pattern != nil {
		teamOpts = append(teamOpts, runtime.WithPattern(tb.pattern(agents)))
	}

	team := runtime.NewTeam(teamOpts...)

	// Fire OnTeamFormed hook.
	if tb.hooks.OnTeamFormed != nil {
		tb.hooks.OnTeamFormed(ctx, task, agents)
	}

	return team, nil
}

// truncate shortens a string to maxLen runes (not bytes), appending "..." if
// truncated. Rune-based truncation preserves valid UTF-8 for multi-byte
// characters (e.g. CJK or emoji) that would otherwise be cut mid-sequence.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
