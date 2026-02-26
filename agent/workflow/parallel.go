package workflow

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// ParallelAgent runs child agents concurrently and collects all results.
type ParallelAgent struct {
	id       string
	persona  agent.Persona
	children []agent.Agent
}

// NewParallelAgent creates a new ParallelAgent.
func NewParallelAgent(id string, children []agent.Agent) *ParallelAgent {
	return &ParallelAgent{
		id:       id,
		children: children,
		persona: agent.Persona{
			Role: "parallel orchestrator",
			Goal: "execute child agents concurrently",
		},
	}
}

// ID returns the agent's unique identifier.
func (a *ParallelAgent) ID() string { return a.id }

// Persona returns the agent's persona.
func (a *ParallelAgent) Persona() agent.Persona { return a.persona }

// Tools returns nil (workflow agents don't use tools directly).
func (a *ParallelAgent) Tools() []tool.Tool { return nil }

// Children returns the child agents.
func (a *ParallelAgent) Children() []agent.Agent { return a.children }

// Invoke runs all children concurrently and returns concatenated results.
func (a *ParallelAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	results := make([]string, len(a.children))
	errs := make([]error, len(a.children))

	var wg sync.WaitGroup
	wg.Add(len(a.children))

	for i, child := range a.children {
		go func(i int, child agent.Agent) {
			defer wg.Done()
			results[i], errs[i] = child.Invoke(ctx, input, opts...)
		}(i, child)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errs {
		if err != nil {
			return "", fmt.Errorf("parallel agent %q: child %q failed: %w", a.id, a.children[i].ID(), err)
		}
	}

	return strings.Join(results, "\n"), nil
}

// eventErr bundles an event and error for channel transport.
type eventErr struct {
	event agent.Event
	err   error
}

// Stream runs all children concurrently, yielding events from all.
func (a *ParallelAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		ch := make(chan eventErr, len(a.children)*8)

		var wg sync.WaitGroup
		wg.Add(len(a.children))

		for _, child := range a.children {
			go func(child agent.Agent) {
				defer wg.Done()
				streamChildToChan(ctx, child, input, ch, opts)
			}(child)
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		for item := range ch {
			if !yield(item.event, item.err) {
				return
			}
			if item.err != nil {
				return
			}
		}

		yield(agent.Event{
			Type:    agent.EventDone,
			AgentID: a.id,
		}, nil)
	}
}

// streamChildToChan streams a child agent's events into a channel.
func streamChildToChan(ctx context.Context, child agent.Agent, input string, ch chan<- eventErr, opts []agent.Option) {
	for event, err := range child.Stream(ctx, input, opts...) {
		ch <- eventErr{event: event, err: err}
		if err != nil {
			return
		}
	}
}
