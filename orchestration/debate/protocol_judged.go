package debate

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Compile-time check.
var (
	_ Protocol  = (*JudgedProtocol)(nil)
	_ TwoPassProtocol = (*JudgedProtocol)(nil)
)

func init() {
	RegisterProtocol("judged", func(cfg map[string]any) (Protocol, error) {
		judgeID, _ := cfg["judge_id"].(string)
		return NewJudgedProtocol(judgeID), nil
	})
}

// JudgedProtocol assigns one agent as the judge who evaluates the
// contributions of all other participants. If JudgeID is empty, the
// first agent in the list is used as the judge.
type JudgedProtocol struct {
	// JudgeID is the agent ID of the judge. If empty, the first agent is used.
	JudgeID string
}

// NewJudgedProtocol creates a new JudgedProtocol with the specified judge.
func NewJudgedProtocol(judgeID string) *JudgedProtocol {
	return &JudgedProtocol{JudgeID: judgeID}
}

// NextRound returns prompts for the participants only. The judge is
// excluded from the first pass because it must evaluate the current
// round's contributions, which do not yet exist. The judge receives its
// prompt from FollowUp after the first pass has completed.
func (p *JudgedProtocol) NextRound(_ context.Context, state DebateState) (map[string]string, error) {
	if len(state.AgentIDs) < 2 {
		return nil, core.Errorf(core.ErrInvalidInput, "debate/judged: requires at least 2 agents, got %d", len(state.AgentIDs))
	}

	judgeID := p.resolveJudge(state.AgentIDs)
	history := buildHistory(state)
	prompts := make(map[string]string, len(state.AgentIDs))

	for _, id := range state.AgentIDs {
		if id == judgeID {
			continue
		}
		prompts[id] = p.buildParticipantPrompt(state, history)
	}

	return prompts, nil
}

// FollowUp runs after first-pass participant contributions are collected.
// It constructs the judge prompt with the current round's arguments so
// the judge can evaluate them directly rather than relying on prior-round
// history.
func (p *JudgedProtocol) FollowUp(_ context.Context, state DebateState, currentRound Round) (map[string]string, error) {
	if len(state.AgentIDs) < 2 {
		return nil, core.Errorf(core.ErrInvalidInput, "debate/judged: requires at least 2 agents, got %d", len(state.AgentIDs))
	}

	judgeID := p.resolveJudge(state.AgentIDs)
	history := buildHistory(state)
	prompts := map[string]string{
		judgeID: p.buildJudgePromptWithCurrent(state, history, currentRound),
	}
	return prompts, nil
}

// resolveJudge returns the judge ID, defaulting to the first agent.
func (p *JudgedProtocol) resolveJudge(agentIDs []string) string {
	if p.JudgeID != "" {
		return p.JudgeID
	}
	return agentIDs[0]
}

// buildParticipantPrompt creates the prompt for debate participants.
func (p *JudgedProtocol) buildParticipantPrompt(state DebateState, history string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Topic: %s\n\n", state.Topic))
	if history != "" {
		sb.WriteString("Previous discussion:\n")
		sb.WriteString(history)
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("Round %d: Share your perspective. A judge will evaluate contributions.", state.CurrentRound+1))
	return sb.String()
}

// buildJudgePromptWithCurrent creates the prompt for the judge agent,
// including the current round's participant contributions so the judge
// can evaluate them directly.
func (p *JudgedProtocol) buildJudgePromptWithCurrent(state DebateState, history string, currentRound Round) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Topic: %s\n\n", state.Topic))
	sb.WriteString("You are the JUDGE in this debate.\n\n")
	if history != "" {
		sb.WriteString("Previous discussion:\n")
		sb.WriteString(history)
		sb.WriteString("\n")
	}
	if len(currentRound.Contributions) > 0 {
		sb.WriteString(fmt.Sprintf("Round %d contributions to evaluate:\n", state.CurrentRound+1))
		for _, c := range currentRound.Contributions {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", c.AgentID, c.Content))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("Round %d: Evaluate the arguments above. Identify strengths, weaknesses, and which positions are most compelling.", state.CurrentRound+1))
	return sb.String()
}
