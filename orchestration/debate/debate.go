package debate

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/v2/internal/hookutil"
)

// DebateState captures the full state of a debate at any point during execution.
type DebateState struct {
	// Topic is the original input/question being debated.
	Topic string
	// Rounds contains all completed rounds of the debate.
	Rounds []Round
	// CurrentRound is the zero-based index of the current round.
	CurrentRound int
	// MaxRounds is the configured maximum number of rounds.
	MaxRounds int
	// AgentIDs lists all participating agent identifiers.
	AgentIDs []string
}

// LastRound returns the most recently completed round, or an empty Round
// if no rounds have been completed.
func (s DebateState) LastRound() Round {
	if len(s.Rounds) == 0 {
		return Round{}
	}
	return s.Rounds[len(s.Rounds)-1]
}

// Round represents a single round of debate.
type Round struct {
	// Number is the one-based round number.
	Number int
	// Contributions holds each agent's contribution for this round.
	Contributions []Contribution
}

// Contribution is a single agent's response in one round.
type Contribution struct {
	// AgentID identifies which agent produced this contribution.
	AgentID string
	// Content is the text response from the agent.
	Content string
	// Role is the assigned role for the agent in this round (e.g., "pro", "con", "judge").
	Role string
}

// ConvergenceResult reports whether the debate has converged and why.
type ConvergenceResult struct {
	// Converged is true if the debate has reached a stopping condition.
	Converged bool
	// Reason describes why convergence was detected (or not).
	Reason string
	// Score is an optional numeric measure of convergence (0.0 to 1.0).
	Score float64
}

// DebateResult is the final output of a debate.
type DebateResult struct {
	// Topic is the original input/question.
	Topic string
	// Rounds contains all rounds of the debate.
	Rounds []Round
	// Convergence describes the final convergence state.
	Convergence ConvergenceResult
	// FinalAnswer is the synthesized final answer, if available.
	FinalAnswer string
	// Metrics contains execution statistics.
	Metrics DebateMetrics
}

// DebateMetrics captures execution statistics for a debate.
type DebateMetrics struct {
	// TotalRounds is the number of rounds completed.
	TotalRounds int
	// TotalContributions is the total number of agent contributions.
	TotalContributions int
	// Duration is the wall-clock time of the debate.
	Duration time.Duration
}

// DebateEventType identifies the kind of debate event.
type DebateEventType string

const (
	// EventRoundStart indicates a new round is beginning.
	EventRoundStart DebateEventType = "round_start"
	// EventContribution indicates an agent has contributed.
	EventContribution DebateEventType = "contribution"
	// EventRoundEnd indicates a round has completed.
	EventRoundEnd DebateEventType = "round_end"
	// EventConvergence indicates a convergence check result.
	EventConvergence DebateEventType = "convergence"
	// EventComplete indicates the debate is finished.
	EventComplete DebateEventType = "complete"
)

// DebateEvent is an event emitted during debate execution.
type DebateEvent struct {
	// Type identifies the kind of event.
	Type DebateEventType
	// Round is the round number associated with this event.
	Round int
	// AgentID identifies the agent for contribution events.
	AgentID string
	// Content holds text content (contribution text or final answer).
	Content string
	// Convergence holds the convergence result for convergence events.
	Convergence *ConvergenceResult
	// Result holds the final debate result for complete events.
	Result *DebateResult
}

// Critique is feedback on a generated response from an evaluator.
type Critique struct {
	// Approved is true if the evaluator accepts the response.
	Approved bool
	// Score is a numeric quality score (0.0 to 1.0).
	Score float64
	// Feedback is textual feedback explaining the evaluation.
	Feedback string
}

// EvaluatorFunc evaluates a generated response and returns a critique.
// The input is the original prompt and response is the generated text.
type EvaluatorFunc func(ctx context.Context, input, response string) (Critique, error)

// Hooks provides optional callback functions invoked during debate execution.
// All fields are optional; nil hooks are skipped.
type Hooks struct {
	// BeforeRound is called before each round begins.
	BeforeRound func(ctx context.Context, state DebateState) error
	// AfterRound is called after each round completes.
	AfterRound func(ctx context.Context, state DebateState) error
	// OnConvergence is called when a convergence check completes.
	OnConvergence func(ctx context.Context, result ConvergenceResult) error
	// OnError is called when an error occurs. The returned error replaces
	// the original; returning nil suppresses the error.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeRound: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, DebateState) error {
			return hk.BeforeRound
		}),
		AfterRound: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, DebateState) error {
			return hk.AfterRound
		}),
		OnConvergence: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, ConvergenceResult) error {
			return hk.OnConvergence
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}
