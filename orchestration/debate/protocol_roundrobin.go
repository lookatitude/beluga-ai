package debate

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
)

// Compile-time check.
var _ DebateProtocol = (*RoundRobinProtocol)(nil)

func init() {
	RegisterProtocol("roundrobin", func(_ map[string]any) (DebateProtocol, error) {
		return NewRoundRobinProtocol(), nil
	})
}

// RoundRobinProtocol is a debate protocol where all agents speak each round
// with full context from previous rounds.
type RoundRobinProtocol struct{}

// NewRoundRobinProtocol creates a new RoundRobinProtocol.
func NewRoundRobinProtocol() *RoundRobinProtocol {
	return &RoundRobinProtocol{}
}

// NextRound returns prompts for all agents, including the debate history
// from previous rounds.
func (p *RoundRobinProtocol) NextRound(_ context.Context, state DebateState) (map[string]string, error) {
	if len(state.AgentIDs) == 0 {
		return nil, core.Errorf(core.ErrInvalidInput, "debate/roundrobin: no agents configured")
	}

	history := buildHistory(state)
	prompts := make(map[string]string, len(state.AgentIDs))

	for _, id := range state.AgentIDs {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Topic: %s\n\n", state.Topic))
		if history != "" {
			sb.WriteString("Previous discussion:\n")
			sb.WriteString(history)
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("Round %d: Please share your perspective.", state.CurrentRound+1))
		prompts[id] = sb.String()
	}

	return prompts, nil
}

// buildHistory formats previous rounds into a readable string.
func buildHistory(state DebateState) string {
	if len(state.Rounds) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, round := range state.Rounds {
		sb.WriteString(fmt.Sprintf("--- Round %d ---\n", round.Number))
		for _, c := range round.Contributions {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", c.AgentID, c.Content))
		}
	}
	return sb.String()
}
