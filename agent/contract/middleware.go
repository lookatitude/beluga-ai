package contract

import (
	"context"
	"iter"
	"strings"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// ValidationMiddleware returns an agent.Middleware that validates inputs and
// outputs against the agent's contract. Agents that do not implement
// ContractProvider pass through without validation.
func ValidationMiddleware() agent.Middleware {
	return func(a agent.Agent) agent.Agent {
		c := ContractOf(a)
		if c == nil {
			return a
		}
		return &validatingAgent{
			Agent:    a,
			contract: c,
		}
	}
}

// validatingAgent wraps an agent to enforce contract validation on Invoke
// and Stream.
type validatingAgent struct {
	agent.Agent
	contract *schema.Contract
}

// compile-time check that validatingAgent still satisfies agent.Agent.
var _ agent.Agent = (*validatingAgent)(nil)

// ID delegates to the wrapped agent.
func (v *validatingAgent) ID() string { return v.Agent.ID() }

// Persona delegates to the wrapped agent.
func (v *validatingAgent) Persona() agent.Persona { return v.Agent.Persona() }

// Tools delegates to the wrapped agent.
func (v *validatingAgent) Tools() []tool.Tool { return v.Agent.Tools() }

// Children delegates to the wrapped agent.
func (v *validatingAgent) Children() []agent.Agent { return v.Agent.Children() }

// Contract delegates to the wrapped agent's contract.
func (v *validatingAgent) Contract() *schema.Contract { return v.contract }

// Invoke validates the input, delegates to the wrapped agent, then validates
// the output.
func (v *validatingAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	if err := ValidateInput(ctx, v.contract, input); err != nil {
		return "", err
	}

	result, err := v.Agent.Invoke(ctx, input, opts...)
	if err != nil {
		return result, err
	}

	if err := ValidateOutput(ctx, v.contract, result); err != nil {
		return "", err
	}

	return result, nil
}

// Stream validates the input, then streams the wrapped agent's output.
// Output validation is performed on the concatenated text result after
// streaming completes.
func (v *validatingAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		if err := ValidateInput(ctx, v.contract, input); err != nil {
			yield(agent.Event{Type: agent.EventError, AgentID: v.Agent.ID()}, err)
			return
		}

		var buf strings.Builder
		for event, err := range v.Agent.Stream(ctx, input, opts...) {
			if !yield(event, err) {
				return
			}
			if err != nil {
				return
			}
			if event.Type == agent.EventText {
				buf.WriteString(event.Text)
			}
		}

		if err := ValidateOutput(ctx, v.contract, buf.String()); err != nil {
			yield(agent.Event{Type: agent.EventError, AgentID: v.Agent.ID()}, err)
		}
	}
}
