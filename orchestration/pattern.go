package orchestration

import (
	"github.com/lookatitude/beluga-ai/v2/core"
)

// OrchestrationPattern is the extension interface for named orchestration
// patterns. Every pattern that participates in the orchestration registry must
// implement this interface.
//
// OrchestrationPattern embeds [core.Runnable] so it can be composed
// transparently with the rest of the framework. The additional Name method
// identifies the pattern for observability and registry lookup.
type OrchestrationPattern interface { //nolint:revive // stutter: renaming would break the exported API
	core.Runnable

	// Name returns the human-readable identifier for this pattern.
	Name() string
}
