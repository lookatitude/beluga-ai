package debate

import (
	"context"
	"fmt"
	"strings"
)

// Compile-time check.
var _ DebateProtocol = (*AdversarialProtocol)(nil)

func init() {
	RegisterProtocol("adversarial", func(_ map[string]any) (DebateProtocol, error) {
		return NewAdversarialProtocol(), nil
	})
}

// AdversarialProtocol assigns pro/con roles to agents. The first half of
// agents argue in favor, the second half argue against. If there is an odd
// number of agents, the extra agent argues in favor.
type AdversarialProtocol struct{}

// NewAdversarialProtocol creates a new AdversarialProtocol.
func NewAdversarialProtocol() *AdversarialProtocol {
	return &AdversarialProtocol{}
}

// NextRound returns prompts with pro/con role assignments for all agents.
func (p *AdversarialProtocol) NextRound(_ context.Context, state DebateState) (map[string]string, error) {
	if len(state.AgentIDs) < 2 {
		return nil, fmt.Errorf("debate/adversarial: requires at least 2 agents, got %d", len(state.AgentIDs))
	}

	history := buildHistory(state)
	prompts := make(map[string]string, len(state.AgentIDs))
	midpoint := (len(state.AgentIDs) + 1) / 2

	for i, id := range state.AgentIDs {
		role := "pro"
		roleDesc := "argue IN FAVOR of"
		if i >= midpoint {
			role = "con"
			roleDesc = "argue AGAINST"
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Topic: %s\n\n", state.Topic))
		sb.WriteString(fmt.Sprintf("Your role: %s — %s the topic.\n\n", strings.ToUpper(role), roleDesc))
		if history != "" {
			sb.WriteString("Previous discussion:\n")
			sb.WriteString(history)
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("Round %d: Present your %s argument.", state.CurrentRound+1, role))
		prompts[id] = sb.String()
	}

	return prompts, nil
}
