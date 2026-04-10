package debate

import (
	"context"
	"fmt"
	"strings"
)

// Compile-time check.
var _ DebateProtocol = (*JudgedProtocol)(nil)

func init() {
	RegisterProtocol("judged", func(cfg map[string]any) (DebateProtocol, error) {
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

// NextRound returns prompts: participants get standard prompts and the
// judge gets an evaluation prompt derived from the state snapshot provided
// by the orchestrator.
//
// IMPORTANT: NextRound is invoked once per round with a single state
// snapshot, so the judge prompt only contains history from *previous*
// rounds — not the current round's participant contributions, which have
// not been generated yet at prompt-build time. In round 1 the judge's
// prompt therefore has no contributions to evaluate, and in later rounds
// the judge always evaluates the previous round's arguments. A proper
// two-pass implementation (run participants first, then rebuild the judge
// prompt with fresh responses) requires orchestrator-level changes.
// Callers who need strict current-round judging should either use a
// different DebateProtocol or wrap the orchestrator to split rounds.
func (p *JudgedProtocol) NextRound(_ context.Context, state DebateState) (map[string]string, error) {
	if len(state.AgentIDs) < 2 {
		return nil, fmt.Errorf("debate/judged: requires at least 2 agents, got %d", len(state.AgentIDs))
	}

	judgeID := p.resolveJudge(state.AgentIDs)
	history := buildHistory(state)
	prompts := make(map[string]string, len(state.AgentIDs))

	for _, id := range state.AgentIDs {
		if id == judgeID {
			prompts[id] = p.buildJudgePrompt(state, history)
		} else {
			prompts[id] = p.buildParticipantPrompt(state, history)
		}
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

// buildJudgePrompt creates the prompt for the judge agent.
func (p *JudgedProtocol) buildJudgePrompt(state DebateState, history string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Topic: %s\n\n", state.Topic))
	sb.WriteString("You are the JUDGE in this debate.\n\n")
	if history != "" {
		sb.WriteString("Previous discussion:\n")
		sb.WriteString(history)
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("Round %d: Evaluate the arguments presented. Identify strengths, weaknesses, and which positions are most compelling.", state.CurrentRound+1))
	return sb.String()
}
