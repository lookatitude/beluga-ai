package plancache

import (
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
)

// Template is a cached plan that records the sequence of actions taken for a
// given input pattern. Templates track success and deviation counts to enable
// eviction of unreliable plans.
type Template struct {
	// ID is the deterministic identifier derived from the action sequence.
	ID string

	// Input is the original input that produced this plan.
	Input string

	// Keywords are the extracted keywords from the input for matching.
	Keywords []string

	// Actions is the sequence of template actions (tool types and names,
	// without argument values).
	Actions []TemplateAction

	// AgentID is the identifier of the agent that produced this plan.
	AgentID string

	// Version tracks template updates. Incremented on each save.
	Version int

	// SuccessCount is the number of times this template was used successfully.
	SuccessCount int

	// DeviationCount is the number of times a replan was needed after using
	// this template, indicating the cached plan was insufficient.
	DeviationCount int

	// CreatedAt is when the template was first created.
	CreatedAt time.Time

	// UpdatedAt is when the template was last modified.
	UpdatedAt time.Time
}

// TemplateAction describes a single step in a cached plan. It records the
// action type, tool name and the original argument payload. Arguments are
// preserved so executors can directly run cached plans without needing to
// re-derive tool inputs.
type TemplateAction struct {
	// Type is the kind of action (tool, respond, finish, handoff).
	Type agent.ActionType

	// ToolName is the name of the tool for tool actions. Empty for non-tool actions.
	ToolName string

	// Arguments is the JSON-encoded tool arguments captured when the template
	// was extracted. Empty for non-tool actions.
	Arguments string

	// Description is an optional human-readable description of the action.
	Description string
}

// DeviationRatio returns the ratio of deviations to total uses (successes +
// deviations). Returns 0.0 if the template has never been used.
func (t *Template) DeviationRatio() float64 {
	total := t.SuccessCount + t.DeviationCount
	if total == 0 {
		return 0.0
	}
	return float64(t.DeviationCount) / float64(total)
}
