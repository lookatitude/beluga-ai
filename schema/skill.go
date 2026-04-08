package schema

import "time"

// Skill represents a procedural memory entry — a reusable, goal-directed
// process that an agent has learned. Skills are the unit of storage in the
// procedural memory tier, analogous to "how-to" knowledge.
//
// Skills are NOT executable code; they are structured descriptions of
// procedures that an agent can reference when encountering similar tasks.
type Skill struct {
	// ID is the unique identifier for this skill.
	ID string `json:"id"`

	// Name is a short, human-readable name for the skill.
	Name string `json:"name"`

	// Description explains what the skill accomplishes.
	Description string `json:"description"`

	// Steps lists the ordered actions that comprise this procedure.
	Steps []string `json:"steps"`

	// Triggers describes the conditions or queries that should activate
	// this skill during retrieval.
	Triggers []string `json:"triggers"`

	// Tags are optional labels for categorization and filtering.
	Tags []string `json:"tags,omitempty"`

	// Version is a monotonically increasing integer incremented each time
	// the skill is updated.
	Version int `json:"version"`

	// Confidence is a score in [0, 1] indicating how reliable this skill is,
	// based on past usage outcomes.
	Confidence float64 `json:"confidence"`

	// Dependencies lists the IDs of other skills that this skill depends on.
	Dependencies []string `json:"dependencies,omitempty"`

	// AgentID identifies the agent that owns or created this skill.
	AgentID string `json:"agent_id"`

	// UsageCount tracks how many times this skill has been retrieved and applied.
	UsageCount int `json:"usage_count"`

	// CreatedAt is the timestamp when the skill was first created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp of the most recent update to this skill.
	UpdatedAt time.Time `json:"updated_at"`
}
