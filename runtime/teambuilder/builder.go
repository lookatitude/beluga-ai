package teambuilder

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/runtime"
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

const opBuild = "teambuilder.build"

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
		return nil, core.NewError(opBuild, core.ErrInvalidInput,
			"agent pool must not be nil", nil)
	}
	if task == "" {
		return nil, core.NewError(opBuild, core.ErrInvalidInput,
			"task must not be empty", nil)
	}

	candidates := tb.pool.List()
	if len(candidates) == 0 {
		return nil, core.NewError(opBuild, core.ErrNotFound,
			"agent pool is empty", nil)
	}

	selected, scores, err := tb.runSelection(ctx, task, candidates)
	if err != nil {
		return nil, err
	}

	// Apply maxAgents limit.
	selected, scores = tb.applyAgentLimit(selected, scores)

	tb.fireAgentSelectedHooks(ctx, task, selected, scores)

	agents := entriesToAgents(selected)
	team := runtime.NewTeam(tb.buildTeamOptions(agents)...)

	if tb.hooks.OnTeamFormed != nil {
		tb.hooks.OnTeamFormed(ctx, task, agents)
	}

	return team, nil
}

// runSelection executes the configured selector and returns the selected
// entries with their scores (nil scores when a plain Selector is used).
func (tb *TeamBuilder) runSelection(ctx context.Context, task string, candidates []PoolEntry) ([]PoolEntry, []float64, error) {
	if scorer, ok := tb.selector.(ScoredSelector); ok {
		return tb.runScoredSelection(ctx, task, candidates, scorer)
	}
	return tb.runPlainSelection(ctx, task, candidates)
}

// runScoredSelection uses a ScoredSelector to select and rank agents.
func (tb *TeamBuilder) runScoredSelection(ctx context.Context, task string, candidates []PoolEntry, scorer ScoredSelector) ([]PoolEntry, []float64, error) {
	scoredEntries, err := scorer.SelectScored(ctx, task, candidates)
	if err != nil {
		if tb.hooks.OnSelectionFailed != nil {
			tb.hooks.OnSelectionFailed(ctx, task, err)
		}
		return nil, nil, core.NewError(opBuild, core.ErrToolFailed,
			"agent selection failed", err)
	}

	selected := make([]PoolEntry, len(scoredEntries))
	scores := make([]float64, len(scoredEntries))
	for i, e := range scoredEntries {
		selected[i] = e.Entry
		scores[i] = e.Score
	}

	if err := tb.checkNonEmpty(ctx, task, selected); err != nil {
		return nil, nil, err
	}
	return selected, scores, nil
}

// runPlainSelection uses a plain Selector (without scores) to select agents.
func (tb *TeamBuilder) runPlainSelection(ctx context.Context, task string, candidates []PoolEntry) ([]PoolEntry, []float64, error) {
	selected, err := tb.selector.Select(ctx, task, candidates)
	if err != nil {
		if tb.hooks.OnSelectionFailed != nil {
			tb.hooks.OnSelectionFailed(ctx, task, err)
		}
		return nil, nil, core.NewError(opBuild, core.ErrToolFailed,
			"agent selection failed", err)
	}

	if err := tb.checkNonEmpty(ctx, task, selected); err != nil {
		return nil, nil, err
	}
	return selected, nil, nil
}

// checkNonEmpty returns an error (and fires the hook) when selected is empty.
func (tb *TeamBuilder) checkNonEmpty(ctx context.Context, task string, selected []PoolEntry) error {
	if len(selected) > 0 {
		return nil
	}
	err := core.NewError(opBuild, core.ErrNotFound,
		fmt.Sprintf("no suitable agents found for task: %s", truncate(task, 100)), nil)
	if tb.hooks.OnSelectionFailed != nil {
		tb.hooks.OnSelectionFailed(ctx, task, err)
	}
	return err
}

// applyAgentLimit trims selected and scores to tb.maxAgents when a limit is set.
func (tb *TeamBuilder) applyAgentLimit(selected []PoolEntry, scores []float64) ([]PoolEntry, []float64) {
	if tb.maxAgents <= 0 || len(selected) <= tb.maxAgents {
		return selected, scores
	}
	selected = selected[:tb.maxAgents]
	if scores != nil {
		scores = scores[:tb.maxAgents]
	}
	return selected, scores
}

// fireAgentSelectedHooks fires OnAgentSelected for each selected entry.
// When scores is nil (plain Selector), a positional approximation is used.
func (tb *TeamBuilder) fireAgentSelectedHooks(ctx context.Context, task string, selected []PoolEntry, scores []float64) {
	if tb.hooks.OnAgentSelected == nil {
		return
	}
	for i, entry := range selected {
		var score float64
		if scores != nil {
			score = scores[i]
		} else {
			score = 1.0 - float64(i)*0.1
			if score < 0.1 {
				score = 0.1
			}
		}
		tb.hooks.OnAgentSelected(ctx, task, entry, score)
	}
}

// buildTeamOptions assembles the runtime.TeamOption slice for the new team.
func (tb *TeamBuilder) buildTeamOptions(agents []agent.Agent) []runtime.TeamOption {
	opts := []runtime.TeamOption{
		runtime.WithTeamID(tb.teamID),
		runtime.WithAgents(agents...),
	}
	if tb.pattern != nil {
		opts = append(opts, runtime.WithPattern(tb.pattern(agents)))
	}
	return opts
}

// entriesToAgents extracts the Agent from each PoolEntry.
func entriesToAgents(entries []PoolEntry) []agent.Agent {
	agents := make([]agent.Agent, len(entries))
	for i, e := range entries {
		agents[i] = e.Agent
	}
	return agents
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
