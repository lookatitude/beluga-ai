package orchestration

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
)

// HandoffOrchestrator manages peer-to-peer agent transfers. When an agent
// emits an [agent.EventHandoff] event (triggered by a transfer_to_{name} tool
// call), HandoffOrchestrator routes control to the target agent and continues
// execution. Control passes back to the caller only when no further handoff
// is emitted or maxHops is reached.
//
// HandoffOrchestrator auto-generates transfer_to_{id} tools for every
// registered peer and injects them into the source agent via
// [agent.HandoffsToTools]. It satisfies [OrchestrationPattern].
type HandoffOrchestrator struct {
	agents  map[string]agent.Agent
	order   []string // insertion order for deterministic iteration
	maxHops int
	entryID string
}

// compile-time check.
var _ OrchestrationPattern = (*HandoffOrchestrator)(nil)

// NewHandoffOrchestrator creates a HandoffOrchestrator with the given agents.
// The first agent in the list is used as the entry point.
func NewHandoffOrchestrator(agents ...agent.Agent) *HandoffOrchestrator {
	h := &HandoffOrchestrator{
		agents:  make(map[string]agent.Agent, len(agents)),
		order:   make([]string, 0, len(agents)),
		maxHops: 10,
	}
	for i, a := range agents {
		h.agents[a.ID()] = a
		h.order = append(h.order, a.ID())
		if i == 0 {
			h.entryID = a.ID()
		}
	}
	return h
}

// WithMaxHops sets the maximum number of agent-to-agent transfers allowed in a
// single invocation. Defaults to 10.
func (h *HandoffOrchestrator) WithMaxHops(n int) *HandoffOrchestrator {
	if n > 0 {
		h.maxHops = n
	}
	return h
}

// WithEntry sets the entry agent by ID. Panics if the ID is not registered.
func (h *HandoffOrchestrator) WithEntry(id string) *HandoffOrchestrator {
	if _, ok := h.agents[id]; !ok {
		panic(fmt.Sprintf("orchestration/handoff: unknown entry agent %q", id))
	}
	h.entryID = id
	return h
}

// Name returns the pattern identifier.
func (h *HandoffOrchestrator) Name() string { return "handoff" }

// Invoke executes the entry agent and follows any handoffs until no further
// transfer is emitted or maxHops is reached. Returns the last agent's text
// output.
func (h *HandoffOrchestrator) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if len(h.agents) == 0 {
		return nil, core.Errorf(core.ErrInvalidInput, "orchestration/handoff: no agents registered")
	}

	inputStr := fmt.Sprintf("%v", input)
	currentID := h.entryID
	hops := 0

	for hops <= h.maxHops {
		a, ok := h.agents[currentID]
		if !ok {
			return nil, core.Errorf(core.ErrNotFound, "orchestration/handoff: unknown agent %q", currentID)
		}

		// Stream the agent to detect handoff events.
		nextID, result, err := h.runAgent(ctx, a, inputStr)
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "orchestration/handoff: agent %q: %w", currentID, err)
		}

		if nextID == "" {
			// No handoff — this is the final result.
			return result, nil
		}

		if _, known := h.agents[nextID]; !known {
			return nil, core.Errorf(core.ErrNotFound, "orchestration/handoff: unknown target agent %q", nextID)
		}

		// Carry the result forward as input to the next agent.
		inputStr = result
		currentID = nextID
		hops++
	}

	return nil, core.Errorf(core.ErrInvalidInput, "orchestration/handoff: max hops (%d) exceeded", h.maxHops)
}

// Stream executes the entry agent and follows handoffs, yielding all events
// from every agent in the chain.
func (h *HandoffOrchestrator) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		if len(h.agents) == 0 {
			yield(nil, core.Errorf(core.ErrInvalidInput, "orchestration/handoff: no agents registered"))
			return
		}

		inputStr := fmt.Sprintf("%v", input)
		currentID := h.entryID
		hops := 0

		for hops <= h.maxHops {
			a, ok := h.agents[currentID]
			if !ok {
				yield(nil, core.Errorf(core.ErrNotFound, "orchestration/handoff: unknown agent %q", currentID))
				return
			}

			nextID, result, stop := h.streamAgent(ctx, a, inputStr, yield)
			if stop {
				return
			}

			if nextID == "" {
				// No handoff — done.
				_ = result
				return
			}

			if _, known := h.agents[nextID]; !known {
				yield(nil, core.Errorf(core.ErrNotFound, "orchestration/handoff: unknown target agent %q", nextID))
				return
			}

			inputStr = result
			currentID = nextID
			hops++
		}

		yield(nil, core.Errorf(core.ErrInvalidInput, "orchestration/handoff: max hops (%d) exceeded", h.maxHops))
	}
}

// runAgent invokes an agent, collecting events. Returns the next agent ID if a
// handoff was detected, the accumulated text result, and any error.
func (h *HandoffOrchestrator) runAgent(ctx context.Context, a agent.Agent, input string) (nextID, result string, err error) {
	var textBuf strings.Builder

	for event, evErr := range a.Stream(ctx, input) {
		if evErr != nil {
			return "", "", evErr
		}
		switch event.Type {
		case agent.EventText:
			textBuf.WriteString(event.Text)
		case agent.EventHandoff:
			if event.Metadata != nil {
				if id, ok := event.Metadata["target_id"].(string); ok && id != "" {
					return id, textBuf.String(), nil
				}
			}
		case agent.EventDone:
			return "", textBuf.String(), nil
		}
	}

	return "", textBuf.String(), nil
}

// streamAgent streams an agent's events through yield. Returns the next agent
// ID if a handoff was detected, the accumulated text, and a stop flag
// indicating the consumer broke out of the loop.
func (h *HandoffOrchestrator) streamAgent(
	ctx context.Context,
	a agent.Agent,
	input string,
	yield func(any, error) bool,
) (nextID, result string, stop bool) {
	var textBuf strings.Builder

	for event, evErr := range a.Stream(ctx, input) {
		if evErr != nil {
			if !yield(nil, core.Errorf(core.ErrProviderDown, "orchestration/handoff: agent %q: %w", a.ID(), evErr)) {
				return "", textBuf.String(), true
			}
			return "", textBuf.String(), true
		}

		switch event.Type {
		case agent.EventText:
			textBuf.WriteString(event.Text)
		case agent.EventHandoff:
			// Yield the handoff event before routing.
			if !yield(event, nil) {
				return "", textBuf.String(), true
			}
			if event.Metadata != nil {
				if id, ok := event.Metadata["target_id"].(string); ok && id != "" {
					return id, textBuf.String(), false
				}
			}
			continue
		}

		if !yield(event, nil) {
			return "", textBuf.String(), true
		}
	}

	return "", textBuf.String(), false
}
