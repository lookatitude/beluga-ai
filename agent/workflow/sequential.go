// Package workflow provides deterministic workflow agents that orchestrate
// child agents without LLM reasoning. SequentialAgent, ParallelAgent, and
// LoopAgent compose child agents in common patterns.
package workflow

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// SequentialAgent runs child agents in sequence, passing the output of each
// as the input to the next.
type SequentialAgent struct {
	id       string
	persona  agent.Persona
	children []agent.Agent
}

// NewSequentialAgent creates a new SequentialAgent.
func NewSequentialAgent(id string, children []agent.Agent) *SequentialAgent {
	return &SequentialAgent{
		id:       id,
		children: children,
		persona: agent.Persona{
			Role: "sequential orchestrator",
			Goal: "execute child agents in sequence",
		},
	}
}

// ID returns the agent's unique identifier.
func (a *SequentialAgent) ID() string { return a.id }

// Persona returns the agent's persona.
func (a *SequentialAgent) Persona() agent.Persona { return a.persona }

// Tools returns nil (workflow agents don't use tools directly).
func (a *SequentialAgent) Tools() []tool.Tool { return nil }

// Children returns the child agents.
func (a *SequentialAgent) Children() []agent.Agent { return a.children }

// Invoke runs children sequentially, passing output to next input.
func (a *SequentialAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	current := input
	for _, child := range a.children {
		result, err := child.Invoke(ctx, current, opts...)
		if err != nil {
			return "", fmt.Errorf("sequential agent %q: child %q failed: %w", a.id, child.ID(), err)
		}
		current = result
	}
	return current, nil
}

// Stream runs children sequentially, yielding events from each.
func (a *SequentialAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		current := input
		for _, child := range a.children {
			var result strings.Builder
			for event, err := range child.Stream(ctx, current, opts...) {
				if err != nil {
					yield(agent.Event{
						Type:    agent.EventError,
						AgentID: a.id,
					}, fmt.Errorf("sequential agent %q: child %q failed: %w", a.id, child.ID(), err))
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
		}
		yield(agent.Event{
			Type:    agent.EventDone,
			AgentID: a.id,
			Text:    current,
		}, nil)
	}
}
