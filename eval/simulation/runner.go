package simulation

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/eval"
)

// AgentFunc is a function that takes a user message and returns the agent's
// response. This decouples the simulation runner from any specific agent
// implementation.
type AgentFunc func(ctx context.Context, userMessage string) (string, error)

// EpisodeResult holds the outcome of a single simulation episode.
type EpisodeResult struct {
	// Turns contains each user-agent exchange as an EvalSample.
	Turns []eval.EvalSample

	// GoalComplete indicates whether the simulated user achieved its goal.
	GoalComplete bool

	// GoalFailed indicates whether the simulated user gave up.
	GoalFailed bool

	// TurnCount is the number of turns in the episode.
	TurnCount int

	// Duration is the wall-clock time of the episode.
	Duration time.Duration
}

// SimReport holds results from running multiple episodes.
type SimReport struct {
	// Episodes contains the results of each episode.
	Episodes []EpisodeResult

	// SuccessRate is the fraction of episodes where the goal was completed.
	SuccessRate float64

	// AverageTurns is the mean number of turns across all episodes.
	AverageTurns float64

	// Duration is the total wall-clock time.
	Duration time.Duration
}

// runnerOptions holds configuration for SimRunner.
type runnerOptions struct {
	env      SimEnvironment
	user     *SimulatedUser
	agent    AgentFunc
	maxTurns int
	timeout  time.Duration
	metrics  []eval.Metric
	onTurn   func(turnIndex int, userMsg, agentResp string)
}

// RunnerOption configures a SimRunner.
type RunnerOption func(*runnerOptions)

// WithEnvironment sets the simulation environment.
func WithEnvironment(env SimEnvironment) RunnerOption {
	return func(o *runnerOptions) {
		o.env = env
	}
}

// WithSimUser sets the simulated user.
func WithSimUser(u *SimulatedUser) RunnerOption {
	return func(o *runnerOptions) {
		o.user = u
	}
}

// WithAgent sets the agent function.
func WithAgent(fn AgentFunc) RunnerOption {
	return func(o *runnerOptions) {
		o.agent = fn
	}
}

// WithMaxTurns sets the maximum number of turns per episode. Defaults to 10.
func WithMaxTurns(n int) RunnerOption {
	return func(o *runnerOptions) {
		if n > 0 {
			o.maxTurns = n
		}
	}
}

// WithRunTimeout sets the timeout for the entire simulation run.
func WithRunTimeout(d time.Duration) RunnerOption {
	return func(o *runnerOptions) {
		o.timeout = d
	}
}

// WithMetrics sets the evaluation metrics applied to each turn's EvalSample.
func WithMetrics(metrics ...eval.Metric) RunnerOption {
	return func(o *runnerOptions) {
		o.metrics = metrics
	}
}

// WithOnTurn sets a callback invoked after each turn.
func WithOnTurn(fn func(turnIndex int, userMsg, agentResp string)) RunnerOption {
	return func(o *runnerOptions) {
		o.onTurn = fn
	}
}

// SimRunner runs multi-turn simulation episodes where a simulated user
// interacts with an agent in a controlled environment.
type SimRunner struct {
	opts runnerOptions
}

// NewSimRunner creates a new SimRunner with the given options.
func NewSimRunner(opts ...RunnerOption) (*SimRunner, error) {
	o := runnerOptions{maxTurns: 10}
	for _, opt := range opts {
		opt(&o)
	}
	if o.agent == nil {
		return nil, core.NewError("simulation.runner.new", core.ErrInvalidInput, "agent function is required", nil)
	}
	if o.user == nil {
		return nil, core.NewError("simulation.runner.new", core.ErrInvalidInput, "simulated user is required", nil)
	}
	return &SimRunner{opts: o}, nil
}

// RunEpisode executes a single simulation episode and returns the result.
func (r *SimRunner) RunEpisode(ctx context.Context) (*EpisodeResult, error) {
	if r.opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.opts.timeout)
		defer cancel()
	}

	start := time.Now()

	// Reset environment if present.
	if r.opts.env != nil {
		if _, err := r.opts.env.Reset(ctx); err != nil {
			return nil, fmt.Errorf("simulation: env reset: %w", err)
		}
	}

	r.opts.user.Reset()

	result := &EpisodeResult{}

	// Start with an initial agent greeting.
	agentResp, err := r.opts.agent(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("simulation: initial agent response: %w", err)
	}

	for turn := 0; turn < r.opts.maxTurns; turn++ {
		if ctx.Err() != nil {
			break
		}

		// Simulated user responds to agent.
		userResp, err := r.opts.user.Respond(ctx, agentResp)
		if err != nil {
			return nil, fmt.Errorf("simulation: user respond at turn %d: %w", turn, err)
		}

		sample := eval.EvalSample{
			Input:    userResp.Message,
			Output:   agentResp,
			Metadata: map[string]any{"turn": turn},
		}
		result.Turns = append(result.Turns, sample)
		result.TurnCount = turn + 1

		if r.opts.onTurn != nil {
			r.opts.onTurn(turn, userResp.Message, agentResp)
		}

		if userResp.GoalComplete {
			result.GoalComplete = true
			break
		}
		if userResp.GoalFailed {
			result.GoalFailed = true
			break
		}

		// Step environment if present.
		if r.opts.env != nil {
			if _, err := r.opts.env.Step(ctx, userResp.Message); err != nil {
				return nil, fmt.Errorf("simulation: env step at turn %d: %w", turn, err)
			}
		}

		// Agent processes user message.
		agentResp, err = r.opts.agent(ctx, userResp.Message)
		if err != nil {
			return nil, fmt.Errorf("simulation: agent at turn %d: %w", turn, err)
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// Run executes multiple episodes and returns an aggregated report.
func (r *SimRunner) Run(ctx context.Context, episodes int) (*SimReport, error) {
	if episodes <= 0 {
		return nil, core.NewError("simulation.runner.run", core.ErrInvalidInput, "episodes must be positive", nil)
	}

	start := time.Now()
	report := &SimReport{
		Episodes: make([]EpisodeResult, 0, episodes),
	}

	var totalTurns int
	var successes int

	for i := 0; i < episodes; i++ {
		if ctx.Err() != nil {
			break
		}

		result, err := r.RunEpisode(ctx)
		if err != nil {
			return nil, fmt.Errorf("simulation: episode %d: %w", i, err)
		}

		report.Episodes = append(report.Episodes, *result)
		totalTurns += result.TurnCount
		if result.GoalComplete {
			successes++
		}
	}

	n := len(report.Episodes)
	if n > 0 {
		report.SuccessRate = float64(successes) / float64(n)
		report.AverageTurns = float64(totalTurns) / float64(n)
	}
	report.Duration = time.Since(start)

	return report, nil
}
