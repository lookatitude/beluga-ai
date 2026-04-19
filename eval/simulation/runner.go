package simulation

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/eval"
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
			return nil, core.Errorf(core.ErrProviderDown, "simulation: env reset: %w", err)
		}
	}

	r.opts.user.Reset()

	result := &EpisodeResult{}

	// Start with an initial agent greeting.
	agentResp, err := r.opts.agent(ctx, "")
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "simulation: initial agent response: %w", err)
	}

	for turn := 0; turn < r.opts.maxTurns; turn++ {
		if ctx.Err() != nil {
			break
		}

		// Simulated user responds to agent.
		userResp, err := r.opts.user.Respond(ctx, agentResp)
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "simulation: user respond at turn %d: %w", turn, err)
		}

		// If the user has reached a terminal state, record the final
		// (user_message -> previous_agent_message) pair for completeness and
		// stop. No further agent turn is required.
		if userResp.GoalComplete || userResp.GoalFailed {
			sample := eval.EvalSample{
				Input:    userResp.Message,
				Output:   "",
				Metadata: map[string]any{"turn": turn, "terminal": true},
			}
			r.scoreSample(ctx, &sample)
			result.Turns = append(result.Turns, sample)
			result.TurnCount = turn + 1
			if r.opts.onTurn != nil {
				r.opts.onTurn(turn, userResp.Message, "")
			}
			if userResp.GoalComplete {
				result.GoalComplete = true
			} else {
				result.GoalFailed = true
			}
			break
		}

		// Step environment if present.
		if r.opts.env != nil {
			if _, err := r.opts.env.Step(ctx, userResp.Message); err != nil {
				return nil, core.Errorf(core.ErrProviderDown, "simulation: env step at turn %d: %w", turn, err)
			}
		}

		// Agent processes user message — this is the output that should be
		// paired with the user message in the EvalSample so metrics see the
		// actual (input, output) pair rather than the agent's prior turn.
		nextAgentResp, err := r.opts.agent(ctx, userResp.Message)
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "simulation: agent at turn %d: %w", turn, err)
		}

		sample := eval.EvalSample{
			Input:    userResp.Message,
			Output:   nextAgentResp,
			Metadata: map[string]any{"turn": turn},
		}
		r.scoreSample(ctx, &sample)
		result.Turns = append(result.Turns, sample)
		result.TurnCount = turn + 1

		if r.opts.onTurn != nil {
			r.opts.onTurn(turn, userResp.Message, nextAgentResp)
		}

		agentResp = nextAgentResp
	}

	result.Duration = time.Since(start)
	return result, nil
}

// scoreSample applies the configured metrics to the given sample, storing
// each score in sample.Metadata keyed by metric name. Metric errors are
// recorded under "<name>_error" so callers can detect scoring failures
// without aborting the episode.
func (r *SimRunner) scoreSample(ctx context.Context, sample *eval.EvalSample) {
	if len(r.opts.metrics) == 0 {
		return
	}
	if sample.Metadata == nil {
		sample.Metadata = make(map[string]any)
	}
	for _, m := range r.opts.metrics {
		score, err := m.Score(ctx, *sample)
		if err != nil {
			sample.Metadata[m.Name()+"_error"] = err.Error()
			continue
		}
		sample.Metadata[m.Name()] = score
	}
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
			return nil, core.Errorf(core.ErrProviderDown, "simulation: episode %d: %w", i, err)
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
