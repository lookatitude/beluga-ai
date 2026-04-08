package debate

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
)

// Compile-time check.
var _ core.Runnable = (*DebateOrchestrator)(nil)

// debateOptions holds configuration for a DebateOrchestrator.
type debateOptions struct {
	maxRounds  int
	protocol   DebateProtocol
	detector   ConvergenceDetector
	hooks      Hooks
	synthesize bool
}

// defaultDebateOptions returns sensible defaults.
func defaultDebateOptions() debateOptions {
	return debateOptions{
		maxRounds:  5,
		protocol:   NewRoundRobinProtocol(),
		detector:   &MaxRoundsDetector{},
		synthesize: true,
	}
}

// Option configures a DebateOrchestrator or GeneratorEvaluator.
type Option func(*debateOptions)

// WithMaxRounds sets the maximum number of debate rounds.
func WithMaxRounds(n int) Option {
	return func(o *debateOptions) {
		if n > 0 {
			o.maxRounds = n
		}
	}
}

// WithProtocol sets the debate protocol.
func WithProtocol(p DebateProtocol) Option {
	return func(o *debateOptions) {
		if p != nil {
			o.protocol = p
		}
	}
}

// WithConvergenceDetector sets the convergence detector.
func WithConvergenceDetector(d ConvergenceDetector) Option {
	return func(o *debateOptions) {
		if d != nil {
			o.detector = d
		}
	}
}

// WithHooks sets the lifecycle hooks.
func WithHooks(h Hooks) Option {
	return func(o *debateOptions) {
		o.hooks = h
	}
}

// WithSynthesize controls whether a final answer is synthesized from
// the last round's contributions. Default is true.
func WithSynthesize(v bool) Option {
	return func(o *debateOptions) {
		o.synthesize = v
	}
}

// DebateOrchestrator coordinates multi-agent debate. It implements
// core.Runnable and supports both Invoke and Stream.
type DebateOrchestrator struct {
	agents map[string]agent.Agent
	ids    []string
	opts   debateOptions
}

// NewDebateOrchestrator creates a new DebateOrchestrator with the given agents
// and options. At least 2 agents are required for debate.
func NewDebateOrchestrator(agents []agent.Agent, opts ...Option) *DebateOrchestrator {
	o := defaultDebateOptions()
	for _, opt := range opts {
		opt(&o)
	}

	agentMap := make(map[string]agent.Agent, len(agents))
	ids := make([]string, 0, len(agents))
	for _, a := range agents {
		agentMap[a.ID()] = a
		ids = append(ids, a.ID())
	}

	return &DebateOrchestrator{
		agents: agentMap,
		ids:    ids,
		opts:   o,
	}
}

// Invoke runs the debate to completion and returns the DebateResult.
// The input must be a string (the debate topic).
func (d *DebateOrchestrator) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	topic, err := d.extractTopic(input)
	if err != nil {
		return nil, err
	}

	if len(d.agents) < 2 {
		return nil, core.NewError("debate.invoke", core.ErrInvalidInput, "at least 2 agents required for debate", nil)
	}

	state := d.initState(topic)
	start := time.Now()

	for round := 0; round < d.opts.maxRounds; round++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		state.CurrentRound = round

		if d.opts.hooks.BeforeRound != nil {
			if err := d.opts.hooks.BeforeRound(ctx, state); err != nil {
				return nil, d.handleError(ctx, fmt.Errorf("debate: before round %d: %w", round, err))
			}
		}

		roundData, err := d.executeRound(ctx, state)
		if err != nil {
			return nil, d.handleError(ctx, err)
		}

		state.Rounds = append(state.Rounds, roundData)

		if d.opts.hooks.AfterRound != nil {
			if err := d.opts.hooks.AfterRound(ctx, state); err != nil {
				return nil, d.handleError(ctx, fmt.Errorf("debate: after round %d: %w", round, err))
			}
		}

		conv, err := d.opts.detector.Check(ctx, state)
		if err != nil {
			return nil, d.handleError(ctx, fmt.Errorf("debate: convergence check: %w", err))
		}

		if d.opts.hooks.OnConvergence != nil {
			if err := d.opts.hooks.OnConvergence(ctx, conv); err != nil {
				return nil, d.handleError(ctx, fmt.Errorf("debate: on convergence: %w", err))
			}
		}

		if conv.Converged {
			return d.buildResult(state, conv, start), nil
		}
	}

	// Max rounds exhausted.
	conv := ConvergenceResult{
		Converged: true,
		Reason:    fmt.Sprintf("reached maximum rounds (%d)", d.opts.maxRounds),
	}
	return d.buildResult(state, conv, start), nil
}

// Stream runs the debate and yields DebateEvent values for each step.
func (d *DebateOrchestrator) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		topic, err := d.extractTopic(input)
		if err != nil {
			yield(nil, err)
			return
		}

		if len(d.agents) < 2 {
			yield(nil, core.NewError("debate.stream", core.ErrInvalidInput, "at least 2 agents required for debate", nil))
			return
		}

		state := d.initState(topic)
		start := time.Now()

		for round := 0; round < d.opts.maxRounds; round++ {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			state.CurrentRound = round

			if !yield(DebateEvent{Type: EventRoundStart, Round: round + 1}, nil) {
				return
			}

			if d.opts.hooks.BeforeRound != nil {
				if err := d.opts.hooks.BeforeRound(ctx, state); err != nil {
					yield(nil, d.handleError(ctx, fmt.Errorf("debate: before round %d: %w", round, err)))
					return
				}
			}

			roundData, err := d.executeRound(ctx, state)
			if err != nil {
				yield(nil, d.handleError(ctx, err))
				return
			}

			// Yield individual contributions.
			for _, c := range roundData.Contributions {
				if !yield(DebateEvent{
					Type:    EventContribution,
					Round:   round + 1,
					AgentID: c.AgentID,
					Content: c.Content,
				}, nil) {
					return
				}
			}

			state.Rounds = append(state.Rounds, roundData)

			if !yield(DebateEvent{Type: EventRoundEnd, Round: round + 1}, nil) {
				return
			}

			if d.opts.hooks.AfterRound != nil {
				if err := d.opts.hooks.AfterRound(ctx, state); err != nil {
					yield(nil, d.handleError(ctx, fmt.Errorf("debate: after round %d: %w", round, err)))
					return
				}
			}

			conv, err := d.opts.detector.Check(ctx, state)
			if err != nil {
				yield(nil, d.handleError(ctx, fmt.Errorf("debate: convergence check: %w", err)))
				return
			}

			if !yield(DebateEvent{Type: EventConvergence, Convergence: &conv}, nil) {
				return
			}

			if d.opts.hooks.OnConvergence != nil {
				if err := d.opts.hooks.OnConvergence(ctx, conv); err != nil {
					yield(nil, d.handleError(ctx, fmt.Errorf("debate: on convergence: %w", err)))
					return
				}
			}

			if conv.Converged {
				result := d.buildResult(state, conv, start)
				yield(DebateEvent{Type: EventComplete, Result: result}, nil)
				return
			}
		}

		conv := ConvergenceResult{
			Converged: true,
			Reason:    fmt.Sprintf("reached maximum rounds (%d)", d.opts.maxRounds),
		}
		result := d.buildResult(state, conv, start)
		yield(DebateEvent{Type: EventComplete, Result: result}, nil)
	}
}

// initState creates the initial DebateState.
func (d *DebateOrchestrator) initState(topic string) DebateState {
	return DebateState{
		Topic:     topic,
		MaxRounds: d.opts.maxRounds,
		AgentIDs:  d.ids,
	}
}

// executeRound runs one round of the debate using the configured protocol.
func (d *DebateOrchestrator) executeRound(ctx context.Context, state DebateState) (Round, error) {
	prompts, err := d.opts.protocol.NextRound(ctx, state)
	if err != nil {
		return Round{}, fmt.Errorf("debate: protocol.NextRound: %w", err)
	}

	round := Round{Number: state.CurrentRound + 1}

	for _, id := range d.ids {
		select {
		case <-ctx.Done():
			return Round{}, ctx.Err()
		default:
		}

		prompt, ok := prompts[id]
		if !ok {
			continue
		}

		a, ok := d.agents[id]
		if !ok {
			continue
		}

		result, err := a.Invoke(ctx, prompt)
		if err != nil {
			return Round{}, fmt.Errorf("debate: agent %q round %d: %w", id, state.CurrentRound+1, err)
		}

		round.Contributions = append(round.Contributions, Contribution{
			AgentID: id,
			Content: result,
		})
	}

	return round, nil
}

// buildResult constructs the final DebateResult.
func (d *DebateOrchestrator) buildResult(state DebateState, conv ConvergenceResult, start time.Time) *DebateResult {
	totalContributions := 0
	for _, r := range state.Rounds {
		totalContributions += len(r.Contributions)
	}

	result := &DebateResult{
		Topic:       state.Topic,
		Rounds:      state.Rounds,
		Convergence: conv,
		Metrics: DebateMetrics{
			TotalRounds:        len(state.Rounds),
			TotalContributions: totalContributions,
			Duration:           time.Since(start),
		},
	}

	if d.opts.synthesize && len(state.Rounds) > 0 {
		result.FinalAnswer = d.synthesizeFinalAnswer(state)
	}

	return result
}

// synthesizeFinalAnswer creates a summary from the last round's contributions.
func (d *DebateOrchestrator) synthesizeFinalAnswer(state DebateState) string {
	lastRound := state.Rounds[len(state.Rounds)-1]
	var sb strings.Builder
	for i, c := range lastRound.Contributions {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s", c.AgentID, c.Content))
	}
	return sb.String()
}

// extractTopic validates and extracts the topic string from input.
func (d *DebateOrchestrator) extractTopic(input any) (string, error) {
	topic, ok := input.(string)
	if !ok {
		return "", core.NewError("debate", core.ErrInvalidInput, "input must be a string topic", nil)
	}
	if strings.TrimSpace(topic) == "" {
		return "", core.NewError("debate", core.ErrInvalidInput, "topic must not be empty", nil)
	}
	return topic, nil
}

// handleError invokes the OnError hook if configured, then returns the error.
func (d *DebateOrchestrator) handleError(ctx context.Context, err error) error {
	if d.opts.hooks.OnError != nil {
		return d.opts.hooks.OnError(ctx, err)
	}
	return err
}
