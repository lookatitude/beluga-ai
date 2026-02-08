package workflow

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// LoopCondition is a function that determines whether the loop should continue.
// It receives the iteration number and the last result, and returns true to stop.
type LoopCondition func(iteration int, lastResult string) bool

// LoopAgent runs a single child agent repeatedly until a condition is met
// or the maximum number of iterations is reached.
type LoopAgent struct {
	id            string
	persona       agent.Persona
	child         agent.Agent
	maxIterations int
	condition     LoopCondition
}

// LoopOption configures a LoopAgent.
type LoopOption func(*LoopAgent)

// WithLoopMaxIterations sets the maximum number of loop iterations.
func WithLoopMaxIterations(n int) LoopOption {
	return func(a *LoopAgent) {
		if n > 0 {
			a.maxIterations = n
		}
	}
}

// WithLoopCondition sets the stop condition for the loop.
func WithLoopCondition(cond LoopCondition) LoopOption {
	return func(a *LoopAgent) {
		a.condition = cond
	}
}

// NewLoopAgent creates a new LoopAgent that runs the child repeatedly.
func NewLoopAgent(id string, child agent.Agent, opts ...LoopOption) *LoopAgent {
	a := &LoopAgent{
		id:            id,
		child:         child,
		maxIterations: 10,
		persona: agent.Persona{
			Role: "loop orchestrator",
			Goal: "execute child agent in a loop until condition is met",
		},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// ID returns the agent's unique identifier.
func (a *LoopAgent) ID() string { return a.id }

// Persona returns the agent's persona.
func (a *LoopAgent) Persona() agent.Persona { return a.persona }

// Tools returns nil (workflow agents don't use tools directly).
func (a *LoopAgent) Tools() []tool.Tool { return nil }

// Children returns the child agent as a single-element slice.
func (a *LoopAgent) Children() []agent.Agent { return []agent.Agent{a.child} }

// Invoke runs the child in a loop until the condition is met.
func (a *LoopAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	current := input
	for i := 0; i < a.maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			return current, fmt.Errorf("loop agent %q: cancelled: %w", a.id, err)
		}

		result, err := a.child.Invoke(ctx, current, opts...)
		if err != nil {
			return "", fmt.Errorf("loop agent %q: iteration %d failed: %w", a.id, i, err)
		}

		current = result

		if a.condition != nil && a.condition(i, result) {
			break
		}
	}
	return current, nil
}

// Stream runs the child in a loop, yielding events from each iteration.
func (a *LoopAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		current := input
		for i := 0; i < a.maxIterations; i++ {
			if err := ctx.Err(); err != nil {
				yield(agent.Event{
					Type:    agent.EventError,
					AgentID: a.id,
				}, fmt.Errorf("loop agent %q: cancelled: %w", a.id, err))
				return
			}

			var result strings.Builder
			for event, err := range a.child.Stream(ctx, current, opts...) {
				if err != nil {
					yield(agent.Event{
						Type:    agent.EventError,
						AgentID: a.id,
					}, fmt.Errorf("loop agent %q: iteration %d failed: %w", a.id, i, err))
					return
				}
				if !yield(event, nil) {
					return
				}
				if event.Type == agent.EventText {
					result.WriteString(event.Text)
				}
			}

			current = result.String()

			if a.condition != nil && a.condition(i, current) {
				break
			}
		}

		yield(agent.Event{
			Type:    agent.EventDone,
			AgentID: a.id,
			Text:    current,
		}, nil)
	}
}
