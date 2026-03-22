package optimizer

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
)

// OptimizerType identifies the type of optimizer algorithm.
type OptimizerType string

const (
	// OptimizerBootstrapFewShot bootstraps few-shot examples from training data.
	OptimizerBootstrapFewShot OptimizerType = "bootstrap_few_shot"
	// OptimizerMIPROv2 uses Bayesian optimization with TPE sampler.
	OptimizerMIPROv2 OptimizerType = "mipro_v2"
	// OptimizerGEPA uses genetic-Pareto prompt evolution.
	OptimizerGEPA OptimizerType = "gepa"
	// OptimizerSIMBA uses stochastic introspective mini-batch ascent.
	OptimizerSIMBA OptimizerType = "simba"
)

// OptimizationStrategy determines the high-level optimization approach.
type OptimizationStrategy string

const (
	// StrategyBootstrapFewShot optimizes by selecting best demonstrations.
	StrategyBootstrapFewShot OptimizationStrategy = "bootstrap_few_shot"
	// StrategyMIPROv2 optimizes prompts using Bayesian optimization.
	StrategyMIPROv2 OptimizationStrategy = "mipro_v2"
	// StrategyGEPA evolves prompts using genetic algorithms.
	StrategyGEPA OptimizationStrategy = "gepa"
	// StrategySIMBA uses stochastic gradient-like ascent.
	StrategySIMBA OptimizationStrategy = "simba"
)

// CompilePhase represents a phase in the compilation process.
type CompilePhase string

const (
	// PhaseInitializing indicates the optimizer is being set up.
	PhaseInitializing CompilePhase = "initializing"
	// PhaseTraining indicates the optimizer is training on examples.
	PhaseTraining CompilePhase = "training"
	// PhaseValidating indicates the optimizer is validating on held-out data.
	PhaseValidating CompilePhase = "validating"
	// PhaseFinalizing indicates the optimizer is producing final output.
	PhaseFinalizing CompilePhase = "finalizing"
	// PhaseComplete indicates optimization is finished.
	PhaseComplete CompilePhase = "complete"
	// PhaseError indicates an error occurred.
	PhaseError CompilePhase = "error"
)

// ConvergenceStatus indicates whether optimization has converged.
type ConvergenceStatus int

const (
	// ConvergenceNotReached indicates optimization is still progressing.
	ConvergenceNotReached ConvergenceStatus = iota
	// ConvergenceReached indicates optimization has converged.
	ConvergenceReached
	// ConvergenceMaxIterations indicates max iterations was reached.
	ConvergenceMaxIterations
	// ConvergenceMaxCost indicates budget was exhausted.
	ConvergenceMaxCost
)

// String returns the string representation of ConvergenceStatus.
func (c ConvergenceStatus) String() string {
	switch c {
	case ConvergenceNotReached:
		return "not_reached"
	case ConvergenceReached:
		return "reached"
	case ConvergenceMaxIterations:
		return "max_iterations"
	case ConvergenceMaxCost:
		return "max_cost"
	default:
		return "unknown"
	}
}

// Example represents a single training/validation example.
type Example struct {
	// Inputs contains the input fields for the example.
	Inputs map[string]any
	// Outputs contains the expected output fields.
	Outputs map[string]any
	// Metadata contains optional example-level metadata.
	Metadata map[string]any
}

// Prediction represents an agent's prediction/output.
type Prediction struct {
	// Text is the raw text output from the agent.
	Text string
	// Outputs contains structured output fields.
	Outputs map[string]any
	// Metadata contains prediction-level metadata.
	Metadata map[string]any
}

// Dataset represents a collection of examples for training or validation.
type Dataset struct {
	// Examples are the individual data points.
	Examples []Example
	// Metadata contains dataset-level metadata.
	Metadata map[string]any
}

// Trial represents a single optimization trial (configuration + result).
type Trial struct {
	// ID is the unique identifier for this trial.
	ID int
	// Score is the achieved metric score.
	Score float64
	// Cost is the accumulated cost for this trial.
	Cost float64
	// Duration is how long the trial took.
	Duration time.Duration
	// Error is set if the trial failed.
	Error error
	// Config contains the configuration tested.
	Config TrialConfig
}

// TrialConfig represents a candidate configuration being tested.
type TrialConfig struct {
	// Prompts contains prompt variations by module/agent.
	Prompts map[string]string
	// Demos contains selected demonstrations by module/agent.
	Demos map[string][]Example
	// Parameters contains optimizer-specific parameters.
	Parameters map[string]any
}

// Result contains the final optimization result.
type Result struct {
	// Agent is the optimized agent.
	Agent agent.Agent
	// Score is the best achieved score.
	Score float64
	// Trials contains all executed trials.
	Trials []Trial
	// TotalCost is the total cost spent.
	TotalCost float64
	// TotalDuration is the total optimization time.
	TotalDuration time.Duration
	// BestTrial is the ID of the best trial.
	BestTrial int
	// ConvergenceStatus indicates why optimization stopped.
	ConvergenceStatus ConvergenceStatus
}

// Progress represents the current state of optimization.
type Progress struct {
	// Phase is the current compilation phase.
	Phase CompilePhase
	// CurrentTrial is the trial being executed (0-indexed).
	CurrentTrial int
	// TotalTrials is the expected number of trials.
	TotalTrials int
	// CurrentScore is the best score so far.
	CurrentScore float64
	// CurrentCost is the accumulated cost.
	CurrentCost float64
	// StartTime is when optimization began.
	StartTime time.Time
	// Elapsed is how long optimization has been running.
	Elapsed time.Duration
}

// IsComplete returns true if optimization is finished.
func (p Progress) IsComplete() bool {
	return p.Phase == PhaseComplete || p.Phase == PhaseError
}

// PercentComplete returns the completion percentage (0.0 to 1.0).
func (p Progress) PercentComplete() float64 {
	if p.TotalTrials == 0 {
		return 0.0
	}
	return float64(p.CurrentTrial) / float64(p.TotalTrials)
}

// Callback receives progress updates during optimization.
type Callback interface {
	// OnProgress is called when progress updates.
	OnProgress(ctx context.Context, progress Progress)
	// OnTrialComplete is called when a trial finishes.
	OnTrialComplete(ctx context.Context, trial Trial)
	// OnComplete is called when optimization finishes.
	OnComplete(ctx context.Context, result Result)
}

// CallbackFunc is a function-based callback adapter.
type CallbackFunc struct {
	OnProgressFunc      func(ctx context.Context, progress Progress)
	OnTrialCompleteFunc func(ctx context.Context, trial Trial)
	OnCompleteFunc      func(ctx context.Context, result Result)
}

// OnProgress implements Callback.
func (c CallbackFunc) OnProgress(ctx context.Context, progress Progress) {
	if c.OnProgressFunc != nil {
		c.OnProgressFunc(ctx, progress)
	}
}

// OnTrialComplete implements Callback.
func (c CallbackFunc) OnTrialComplete(ctx context.Context, trial Trial) {
	if c.OnTrialCompleteFunc != nil {
		c.OnTrialCompleteFunc(ctx, trial)
	}
}

// OnComplete implements Callback.
func (c CallbackFunc) OnComplete(ctx context.Context, result Result) {
	if c.OnCompleteFunc != nil {
		c.OnCompleteFunc(ctx, result)
	}
}

// Budget defines resource limits for optimization.
type Budget struct {
	// MaxIterations limits the number of optimization iterations.
	MaxIterations int
	// MaxCost limits total spending (in USD).
	MaxCost float64
	// MaxDuration limits total runtime.
	MaxDuration time.Duration
	// MaxCalls limits total LLM calls.
	MaxCalls int
}

// IsExceeded returns true if any budget limit is exceeded.
func (b Budget) IsExceeded(cost float64, calls int, elapsed time.Duration) bool {
	if b.MaxCost > 0 && cost >= b.MaxCost {
		return true
	}
	if b.MaxCalls > 0 && calls >= b.MaxCalls {
		return true
	}
	if b.MaxDuration > 0 && elapsed >= b.MaxDuration {
		return true
	}
	return false
}

// Remaining returns the remaining budget ratios (0.0 to 1.0).
func (b Budget) Remaining(cost float64, calls int, elapsed time.Duration) map[string]float64 {
	remaining := make(map[string]float64)
	if b.MaxCost > 0 {
		remaining["cost"] = 1.0 - (cost / b.MaxCost)
		if remaining["cost"] < 0 {
			remaining["cost"] = 0
		}
	}
	if b.MaxCalls > 0 {
		remaining["calls"] = 1.0 - (float64(calls) / float64(b.MaxCalls))
		if remaining["calls"] < 0 {
			remaining["calls"] = 0
		}
	}
	if b.MaxDuration > 0 {
		remaining["duration"] = 1.0 - (float64(elapsed) / float64(b.MaxDuration))
		if remaining["duration"] < 0 {
			remaining["duration"] = 0
		}
	}
	return remaining
}
