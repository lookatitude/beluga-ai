// Package runtime provides multi-agent orchestration primitives including
// teams, orchestration patterns, and session management.
package runtime

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/agent"
)

// OrchestrationPattern defines how a Team coordinates its member agents.
// Implementations determine the execution strategy: sequential pipeline,
// parallel scatter-gather, supervisor delegation, and so on.
type OrchestrationPattern interface {
	// Execute runs the orchestration strategy over the given agents with the
	// provided input. It returns an iterator of agent events and errors,
	// following the iter.Seq2 streaming convention.
	Execute(ctx context.Context, agents []agent.Agent, input string) iter.Seq2[agent.Event, error]
}
