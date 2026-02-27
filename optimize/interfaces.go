// Package optimize provides DSPy-style automated prompt/agent optimization for Beluga AI.
//
// This package implements four key optimizers:
//   - BootstrapFewShot: Bootstraps few-shot examples from training data
//   - MIPROv2: Bayesian optimization with TPE sampler
//   - GEPA: Genetic-Pareto prompt evolution
//   - SIMBA: Stochastic introspective mini-batch ascent
//
// Each optimizer transforms an uncompiled program into an optimized version
// by systematically exploring the prompt space using training data and metrics.
package optimize

import (
	"context"
	"encoding/json"
)

// Optimizer transforms a program into an optimized version.
// Implementations include BootstrapFewShot, MIPROv2, GEPA, and SIMBA.
type Optimizer interface {
	// Compile optimizes the program using the provided training data and metric.
	// Returns a compiled (optimized) program ready for inference.
	Compile(ctx context.Context, program Program, opts CompileOptions) (Program, error)
}

// Metric evaluates prediction quality. Higher scores are better.
// Binary metrics (0/1) work best for optimization.
type Metric interface {
	// Evaluate returns a score for the prediction on the given example.
	// The trace contains the full execution history for inspection.
	Evaluate(example Example, pred Prediction, trace *Trace) (float64, error)
}

// MetricFunc is a function type that implements Metric.
type MetricFunc func(example Example, pred Prediction, trace *Trace) float64

// Evaluate implements the Metric interface for MetricFunc.
func (f MetricFunc) Evaluate(example Example, pred Prediction, trace *Trace) (float64, error) {
	return f(example, pred, trace), nil
}

// Program represents an optimizable program (agent, chain, etc.).
type Program interface {
	// Run executes the program with the given inputs.
	Run(ctx context.Context, inputs map[string]interface{}) (Prediction, error)

	// WithDemos returns a new program with the specified demonstrations.
	WithDemos(demos []Example) Program

	// GetSignature returns the program's input/output signature.
	GetSignature() Signature
}

// Signature defines the input/output contract for a program.
type Signature interface {
	// Render converts inputs to a prompt string.
	Render(inputs map[string]interface{}) (string, error)

	// Parse extracts outputs from the LLM response.
	Parse(response string) (map[string]interface{}, error)

	// GetInputFields returns the input field definitions.
	GetInputFields() []Field

	// GetOutputFields returns the output field definitions.
	GetOutputFields() []Field
}

// Field represents a single input or output field.
type Field struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// Example represents a training example with inputs and expected outputs.
type Example struct {
	Inputs  map[string]interface{}
	Outputs map[string]interface{}
	// Metadata can store example-level metadata (source, difficulty, etc.)
	Metadata map[string]interface{}
}

// Prediction represents the output of running a program.
type Prediction struct {
	Outputs map[string]interface{}
	// Raw contains the raw LLM response.
	Raw string
	// Usage contains token usage information.
	Usage TokenUsage
}

// TokenUsage tracks LLM token consumption.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Trace contains the full execution history of a program run.
type Trace struct {
	// Steps contains each intermediate execution step.
	Steps []TraceStep
	// Inputs contains the original inputs.
	Inputs map[string]interface{}
	// Final output.
	Output Prediction
}

// TraceStep represents a single step in the execution trace.
type TraceStep struct {
	ModuleID string
	Inputs   map[string]interface{}
	Output   Prediction
	Duration int64 // milliseconds
}

// CompileOptions configures the optimization process.
type CompileOptions struct {
	// Trainset contains examples for optimization.
	Trainset []Example

	// Metric evaluates prediction quality.
	Metric Metric

	// Valset is optional validation set for early stopping.
	Valset []Example

	// MaxCost limits the optimization budget.
	MaxCost *CostBudget

	// Callbacks receive optimization progress updates.
	Callbacks []Callback

	// NumWorkers controls parallel trial execution (default: 10).
	NumWorkers int

	// Seed for reproducibility.
	Seed int64
}

// CostBudget limits optimization spending.
type CostBudget struct {
	MaxDollars    float64
	MaxTokens     int64
	MaxIterations int
}

// Exceeded returns true if the budget is exceeded given current usage.
func (b *CostBudget) Exceeded(cost float64, tokens int64, iterations int) bool {
	if b.MaxDollars > 0 && cost >= b.MaxDollars {
		return true
	}
	if b.MaxTokens > 0 && tokens >= b.MaxTokens {
		return true
	}
	if b.MaxIterations > 0 && iterations >= b.MaxIterations {
		return true
	}
	return false
}

// Callback receives optimization progress updates.
type Callback interface {
	// OnTrialComplete is called after each trial completes.
	OnTrialComplete(trial Trial)
	// OnOptimizationComplete is called when optimization finishes.
	OnOptimizationComplete(result OptimizationResult)
}

// Trial represents a single optimization trial.
type Trial struct {
	ID        int
	Candidate Candidate
	Score     float64
	Cost      float64
	Error     error
	Duration  int64 // milliseconds
}

// Candidate represents a potential program configuration.
type Candidate struct {
	ID       string
	Prompts  map[string]string   // module -> prompt
	Demos    map[string][]Example // module -> demonstrations
	Metadata map[string]interface{}
}

// OptimizationResult contains the final optimization results.
type OptimizationResult struct {
	BestCandidate Candidate
	BestScore     float64
	TotalCost     float64
	TotalTokens   int64
	NumTrials     int
	Duration      int64 // milliseconds
	AllTrials     []Trial
}

// LLMClient abstracts LLM providers for optimizers.
type LLMClient interface {
	// Complete generates a completion for the given prompt.
	Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error)

	// CompleteJSON generates a JSON completion matching the schema.
	CompleteJSON(ctx context.Context, prompt string, schema json.RawMessage, opts CompletionOptions) (json.RawMessage, error)

	// GetUsage returns accumulated token usage.
	GetUsage() TokenUsage
}

// CompletionOptions configures LLM completion requests.
type CompletionOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
	TopP        float64
}

// Module represents a reusable component in a program.
type Module interface {
	// Run executes the module.
	Run(ctx context.Context, inputs map[string]interface{}) (Prediction, error)

	// GetSignature returns the module's signature.
	GetSignature() Signature

	// WithPrompt returns a new module with the specified prompt.
	WithPrompt(prompt string) Module
}

// ProgramGraph represents a multi-module program structure.
type ProgramGraph interface {
	// GetModules returns all modules in the program.
	GetModules() []Module

	// GetEdges returns connections between modules.
	GetEdges() []Edge

	// GetModule returns a module by ID.
	GetModule(id string) (Module, bool)
}

// Edge represents a connection between modules.
type Edge struct {
	From string
	To   string
	// Field mapping from output to input.
	Mapping map[string]string
}

// ConvergenceChecker determines if optimization has converged.
type ConvergenceChecker struct {
	WindowSize int
	Threshold  float64
	history    []float64
}

// Update adds a new score and checks for convergence.
func (c *ConvergenceChecker) Update(score float64) bool {
	c.history = append(c.history, score)
	if len(c.history) < c.WindowSize {
		return false
	}
	// Keep only recent scores
	c.history = c.history[len(c.history)-c.WindowSize:]
	// Calculate variance
	mean := 0.0
	for _, s := range c.history {
		mean += s
	}
	mean /= float64(len(c.history))
	variance := 0.0
	for _, s := range c.history {
		diff := s - mean
		variance += diff * diff
	}
	variance /= float64(len(c.history))
	// Converged if variance is below threshold
	return variance < c.Threshold
}

// Cache provides result caching for optimization.
type Cache interface {
	// Get retrieves a cached result.
	Get(key string) (string, bool)
	// Set stores a result in the cache.
	Set(key string, value string, ttl int) // ttl in seconds
}

// CostTracker tracks optimization costs.
type CostTracker struct {
	TotalCost        float64
	TotalTokens      int64
	PromptTokens     int64
	CompletionTokens int64
}

// TrackRequest updates cost tracking with a request.
func (ct *CostTracker) TrackRequest(usage TokenUsage, costPer1K float64) {
	ct.PromptTokens += int64(usage.PromptTokens)
	ct.CompletionTokens += int64(usage.CompletionTokens)
	ct.TotalTokens += int64(usage.TotalTokens)
	ct.TotalCost += float64(usage.PromptTokens) * costPer1K / 1000
	ct.TotalCost += float64(usage.CompletionTokens) * costPer1K / 1000
}
