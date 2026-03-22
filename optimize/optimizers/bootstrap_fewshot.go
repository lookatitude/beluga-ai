package optimizers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/lookatitude/beluga-ai/optimize"
)

func init() {
	optimize.RegisterOptimizer("bootstrapfewshot", func(cfg optimize.OptimizerConfig) (optimize.Optimizer, error) {
		var opts []BootstrapOption
		if cfg.LLM != nil {
			opts = append(opts, WithBootstrapLLMClient(cfg.LLM))
		}
		return NewBootstrapFewShot(opts...), nil
	})
}

// BootstrapFewShot implements the DSPy BootstrapFewShot algorithm.
//
// The optimizer generates few-shot demonstrations by running a teacher program
// on training examples and filtering outputs that exceed a metric threshold.
// Successfully bootstrapped examples are combined with labeled (ground-truth)
// examples to produce an optimized program.
//
// The algorithm proceeds as follows:
//  1. Shuffle trainset using a deterministic RNG for reproducibility.
//  2. For each training example, run the teacher (or student if no teacher)
//     up to MaxRounds times. Accept the first prediction that meets MetricThreshold.
//  3. Once MaxBootstrapped demos are collected, stop bootstrapping.
//  4. Fill remaining capacity with labeled examples from the trainset.
//  5. If a validation set is provided, evaluate the final program on it.
//  6. Report results via callbacks and return the optimized program.
//
// All RNG usage is seed-based for deterministic, race-safe execution.
type BootstrapFewShot struct {
	// Teacher is the program used to generate demonstrations.
	// If nil, the student program itself is used as teacher.
	Teacher optimize.Program

	// MaxBootstrapped is the maximum number of bootstrapped examples.
	MaxBootstrapped int

	// MaxLabeled is the maximum number of labeled (ground truth) examples.
	MaxLabeled int

	// MaxRounds is the number of bootstrap attempts per example.
	MaxRounds int

	// MetricThreshold is the minimum score to accept a demonstration.
	MetricThreshold float64

	// Seed for reproducibility.
	Seed int64

	// llm is an optional LLM client passed from the registry config.
	llm optimize.LLMClient
}

// BootstrapOption configures a BootstrapFewShot optimizer.
type BootstrapOption func(*BootstrapFewShot)

// NewBootstrapFewShot creates a new BootstrapFewShot optimizer with defaults.
func NewBootstrapFewShot(opts ...BootstrapOption) *BootstrapFewShot {
	bs := &BootstrapFewShot{
		MaxBootstrapped: 4,
		MaxLabeled:      16,
		MaxRounds:       1,
		MetricThreshold: 1.0,
		Seed:            42,
	}
	for _, opt := range opts {
		opt(bs)
	}
	return bs
}

// WithTeacher sets the teacher program for bootstrapping.
func WithTeacher(teacher optimize.Program) BootstrapOption {
	return func(bs *BootstrapFewShot) {
		bs.Teacher = teacher
	}
}

// WithMaxBootstrapped sets the maximum number of bootstrapped examples.
func WithMaxBootstrapped(n int) BootstrapOption {
	return func(bs *BootstrapFewShot) {
		if n > 0 {
			bs.MaxBootstrapped = n
		}
	}
}

// WithMaxLabeled sets the maximum number of labeled examples.
func WithMaxLabeled(n int) BootstrapOption {
	return func(bs *BootstrapFewShot) {
		if n > 0 {
			bs.MaxLabeled = n
		}
	}
}

// WithMaxRounds sets the number of bootstrap attempts per example.
func WithMaxRounds(n int) BootstrapOption {
	return func(bs *BootstrapFewShot) {
		if n > 0 {
			bs.MaxRounds = n
		}
	}
}

// WithMetricThreshold sets the minimum score to accept a demonstration.
func WithMetricThreshold(threshold float64) BootstrapOption {
	return func(bs *BootstrapFewShot) {
		if threshold >= 0 {
			bs.MetricThreshold = threshold
		}
	}
}

// WithBootstrapSeed sets the random seed for reproducibility.
func WithBootstrapSeed(seed int64) BootstrapOption {
	return func(bs *BootstrapFewShot) {
		bs.Seed = seed
	}
}

// WithBootstrapLLMClient wires an LLM client from the registry config.
func WithBootstrapLLMClient(llm optimize.LLMClient) BootstrapOption {
	return func(bs *BootstrapFewShot) {
		bs.llm = llm
	}
}

// -------------------------------------------------------------------------
// Compile — main entry point
// -------------------------------------------------------------------------

// Compile implements optimize.Optimizer.
//
// It bootstraps few-shot demonstrations from the trainset by running the teacher
// on each example, evaluating with the metric, and collecting passing outputs.
func (bs *BootstrapFewShot) Compile(ctx context.Context, student optimize.Program, opts optimize.CompileOptions) (optimize.Program, error) {
	if opts.Metric == nil {
		return nil, fmt.Errorf("bootstrapfewshot: metric is required")
	}

	teacher := bs.Teacher
	if teacher == nil {
		teacher = student
	}

	startTime := time.Now()

	// Shuffle trainset deterministically.
	rng := rand.New(rand.NewSource(bs.Seed))
	shuffled := make([]optimize.Example, len(opts.Trainset))
	copy(shuffled, opts.Trainset)
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Phase 1: Bootstrap demonstrations from teacher.
	bootstrapped := make([]optimize.Example, 0, bs.MaxBootstrapped)
	var trials []optimize.Trial
	var totalTokens int64
	trialID := 0

	for _, ex := range shuffled {
		if len(bootstrapped) >= bs.MaxBootstrapped {
			break
		}

		// Check context cancellation.
		select {
		case <-ctx.Done():
			if len(bootstrapped) > 0 {
				break
			}
			return nil, ctx.Err()
		default:
		}

		// Check cost budget.
		if opts.MaxCost != nil && opts.MaxCost.Exceeded(0, totalTokens, trialID) {
			break
		}

		accepted := false
		for round := 0; round < bs.MaxRounds; round++ {
			pred, err := teacher.Run(ctx, ex.Inputs)
			if err != nil {
				trials = append(trials, optimize.Trial{
					ID:    trialID,
					Score: 0,
					Error: err,
				})
				trialID++
				continue
			}

			totalTokens += int64(pred.Usage.TotalTokens)

			score, err := opts.Metric.Evaluate(ex, pred, nil)
			if err != nil {
				trials = append(trials, optimize.Trial{
					ID:    trialID,
					Score: 0,
					Error: err,
				})
				trialID++
				continue
			}

			trial := optimize.Trial{
				ID:    trialID,
				Score: score,
			}
			trials = append(trials, trial)

			// Notify callbacks.
			for _, cb := range opts.Callbacks {
				cb.OnTrialComplete(trial)
			}

			trialID++

			if score >= bs.MetricThreshold {
				demo := optimize.Example{
					Inputs:  ex.Inputs,
					Outputs: pred.Outputs,
				}
				bootstrapped = append(bootstrapped, demo)
				accepted = true
				break
			}
		}

		_ = accepted // suppress unused warning
	}

	// Phase 2: Fill with labeled examples.
	labeled := make([]optimize.Example, 0, bs.MaxLabeled)
	for _, ex := range shuffled {
		if len(labeled) >= bs.MaxLabeled {
			break
		}
		if !containsExample(bootstrapped, ex) {
			labeled = append(labeled, ex)
		}
	}

	// Combine: bootstrapped first, then labeled.
	demos := make([]optimize.Example, 0, len(bootstrapped)+len(labeled))
	demos = append(demos, bootstrapped...)
	demos = append(demos, labeled...)

	compiled := student.WithDemos(demos)

	// Phase 3: Validation scoring.
	bestScore := bs.computeScore(ctx, compiled, bootstrapped, opts)
	if len(opts.Valset) > 0 {
		bestScore = bs.evaluateOnSet(ctx, compiled, opts.Valset, opts.Metric)
	}

	// Report final result via callbacks.
	result := optimize.OptimizationResult{
		BestCandidate: optimize.Candidate{
			ID: "bootstrap_best",
			Demos: map[string][]optimize.Example{
				"bootstrapped": bootstrapped,
				"labeled":      labeled,
			},
			Metadata: map[string]interface{}{
				"num_bootstrapped": len(bootstrapped),
				"num_labeled":      len(labeled),
			},
		},
		BestScore:   bestScore,
		TotalTokens: totalTokens,
		NumTrials:   len(trials),
		Duration:    time.Since(startTime).Milliseconds(),
		AllTrials:   trials,
	}
	for _, cb := range opts.Callbacks {
		cb.OnOptimizationComplete(result)
	}

	return compiled, nil
}

// computeScore calculates the average metric score for bootstrapped demos.
func (bs *BootstrapFewShot) computeScore(ctx context.Context, program optimize.Program, bootstrapped []optimize.Example, opts optimize.CompileOptions) float64 {
	if len(bootstrapped) == 0 {
		return 0
	}
	var total float64
	var count int
	for _, ex := range bootstrapped {
		pred, err := program.Run(ctx, ex.Inputs)
		if err != nil {
			continue
		}
		score, err := opts.Metric.Evaluate(ex, pred, nil)
		if err != nil {
			continue
		}
		total += score
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// evaluateOnSet scores the program against the supplied example set.
func (bs *BootstrapFewShot) evaluateOnSet(ctx context.Context, program optimize.Program, examples []optimize.Example, metric optimize.Metric) float64 {
	var total float64
	var count int
	for _, ex := range examples {
		select {
		case <-ctx.Done():
			if count > 0 {
				return total / float64(count)
			}
			return 0
		default:
		}

		pred, err := program.Run(ctx, ex.Inputs)
		if err != nil {
			continue
		}
		score, err := metric.Evaluate(ex, pred, nil)
		if err != nil {
			continue
		}
		total += score
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// containsExample checks if an example is already in the list.
func containsExample(demos []optimize.Example, ex optimize.Example) bool {
	for _, d := range demos {
		if examplesEqual(d, ex) {
			return true
		}
	}
	return false
}

// examplesEqual checks if two examples have the same inputs.
func examplesEqual(a, b optimize.Example) bool {
	if len(a.Inputs) != len(b.Inputs) {
		return false
	}
	for k, v := range a.Inputs {
		if b.Inputs[k] != v {
			return false
		}
	}
	return true
}
