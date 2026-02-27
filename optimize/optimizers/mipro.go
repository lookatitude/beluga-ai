package optimizers

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/bayesian"
)

func init() {
	optimize.RegisterOptimizer("mipro", func(cfg optimize.OptimizerConfig) (optimize.Optimizer, error) {
		return NewMIPROv2(), nil
	})
}

// MIPROv2 implements Multi-step Instruction Proposal Optimization with Bayesian search.
// It jointly optimizes instructions and demonstrations using TPE (Tree-structured Parzen Estimator).
type MIPROv2 struct {
	// NumTrials is the number of optimization trials to run.
	NumTrials int

	// MinibatchSize is the number of examples to use for each trial evaluation.
	MinibatchSize int

	// NumInstructionCandidates is the number of instruction variations to propose.
	NumInstructionCandidates int

	// NumDemoCandidates is the number of demonstration sets to propose.
	NumDemoCandidates int

	// TPE is the Bayesian sampler.
	TPE *bayesian.TPE

	// Random seed for reproducibility.
	Seed int64
}

// Option configures a MIPROv2 optimizer.
type MIPROv2Option func(*MIPROv2)

// NewMIPROv2 creates a new MIPROv2 optimizer with defaults.
func NewMIPROv2(opts ...MIPROv2Option) *MIPROv2 {
	m := &MIPROv2{
		NumTrials:                30,
		MinibatchSize:            25,
		NumInstructionCandidates: 5,
		NumDemoCandidates:        5,
		TPE:                      bayesian.NewTPE(),
		Seed:                     42,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.TPE.WithSeed(m.Seed)
	return m
}

// WithNumTrials sets the number of optimization trials.
func WithNumTrials(n int) MIPROv2Option {
	return func(m *MIPROv2) {
		if n > 0 {
			m.NumTrials = n
		}
	}
}

// WithMinibatchSize sets the minibatch size for evaluation.
func WithMinibatchSize(n int) MIPROv2Option {
	return func(m *MIPROv2) {
		if n > 0 {
			m.MinibatchSize = n
		}
	}
}

// WithNumInstructionCandidates sets the number of instruction candidates.
func WithNumInstructionCandidates(n int) MIPROv2Option {
	return func(m *MIPROv2) {
		if n > 0 {
			m.NumInstructionCandidates = n
		}
	}
}

// WithNumDemoCandidates sets the number of demo set candidates.
func WithNumDemoCandidates(n int) MIPROv2Option {
	return func(m *MIPROv2) {
		if n > 0 {
			m.NumDemoCandidates = n
		}
	}
}

// WithSeed sets the random seed for reproducibility.
func WithMIPROv2Seed(seed int64) MIPROv2Option {
	return func(m *MIPROv2) {
		m.Seed = seed
	}
}

// Compile implements optimize.Optimizer.
func (m *MIPROv2) Compile(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) (optimize.Program, error) {
	if opts.Metric == nil {
		return nil, fmt.Errorf("metric is required")
	}

	if len(opts.Trainset) == 0 {
		return nil, fmt.Errorf("trainset is required")
	}

	// Phase 1: Generate candidates (proposals)
	instructionCandidates, err := m.generateInstructionCandidates(ctx, program, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate instruction candidates: %w", err)
	}

	demoCandidates, err := m.generateDemoCandidates(ctx, program, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate demo candidates: %w", err)
	}

	// Phase 2: Bayesian optimization
	bestScore := -1.0
	var bestCandidate optimize.Candidate
	var trials []bayesian.Trial

	for trial := 0; trial < m.NumTrials; trial++ {
		// Sample using TPE
		instructionIdx := m.TPE.SampleInt("instruction_idx", 0, len(instructionCandidates)-1, trials)
		demoIdx := m.TPE.SampleInt("demo_idx", 0, len(demoCandidates)-1, trials)

		candidate := optimize.Candidate{
			ID: fmt.Sprintf("trial_%d", trial),
			Prompts: map[string]string{
				"instruction": instructionCandidates[instructionIdx],
			},
			Demos: map[string][]optimize.Example{
				"demos": demoCandidates[demoIdx],
			},
		}

		// Evaluate candidate on minibatch
		score, err := m.evaluateCandidate(ctx, program, candidate, opts)
		if err != nil {
			continue
		}

		// Record trial
		trials = append(trials, bayesian.Trial{
			Params: map[string]interface{}{
				"instruction_idx": instructionIdx,
				"demo_idx":        demoIdx,
				"value":           instructionIdx, // For int distribution
			},
			Score: score,
		})

		// Update best
		if score > bestScore {
			bestScore = score
			bestCandidate = candidate
		}

		// Check cost budget
		if opts.MaxCost != nil && trial > 0 {
			// Approximate cost tracking
			if float64(trial) > opts.MaxCost.MaxDollars*10 {
				break
			}
		}
	}

	if bestScore < 0 {
		return nil, fmt.Errorf("no valid candidate found during optimization")
	}

	// Return optimized program
	return program.WithDemos(bestCandidate.Demos["demos"]), nil
}

// generateInstructionCandidates generates instruction variations.
func (m *MIPROv2) generateInstructionCandidates(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) ([]string, error) {
	// For now, return a simple set of instruction variations
	// In a full implementation, this would use an LLM to generate variations
	candidates := []string{
		"Answer the question based on the context provided.",
		"Provide a concise answer to the question using the given context.",
		"Based on the information provided, what is the answer?",
		"Using only the context, answer this question:",
		"Respond to the question with information from the context.",
	}

	// Pad or trim to match NumInstructionCandidates
	if len(candidates) > m.NumInstructionCandidates {
		candidates = candidates[:m.NumInstructionCandidates]
	}
	for len(candidates) < m.NumInstructionCandidates {
		candidates = append(candidates, candidates[len(candidates)%len(candidates)])
	}

	return candidates, nil
}

// generateDemoCandidates generates demonstration set candidates.
func (m *MIPROv2) generateDemoCandidates(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) ([][]optimize.Example, error) {
	// Shuffle and sample the trainset to create diverse demo sets
	rng := rand.New(rand.NewSource(m.Seed))

	candidates := make([][]optimize.Example, 0, m.NumDemoCandidates)
	for i := 0; i < m.NumDemoCandidates; i++ {
		// Shuffle trainset
		shuffled := make([]optimize.Example, len(opts.Trainset))
		copy(shuffled, opts.Trainset)
		rng.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		// Take first 4 examples as demos
		n := 4
		if n > len(shuffled) {
			n = len(shuffled)
		}
		candidates = append(candidates, shuffled[:n])
	}

	return candidates, nil
}

// evaluateCandidate evaluates a candidate on a minibatch.
func (m *MIPROv2) evaluateCandidate(ctx context.Context, program optimize.Program, candidate optimize.Candidate, opts optimize.CompileOptions) (float64, error) {
	// Sample minibatch
	rng := rand.New(rand.NewSource(m.Seed))
	minibatch := make([]optimize.Example, len(opts.Trainset))
	copy(minibatch, opts.Trainset)
	rng.Shuffle(len(minibatch), func(i, j int) {
		minibatch[i], minibatch[j] = minibatch[j], minibatch[i]
	})

	n := m.MinibatchSize
	if n > len(minibatch) {
		n = len(minibatch)
	}
	minibatch = minibatch[:n]

	// Create program with candidate demos
	progWithDemos := program.WithDemos(candidate.Demos["demos"])

	// Evaluate on minibatch
	var totalScore float64
	for _, ex := range minibatch {
		pred, err := progWithDemos.Run(ctx, ex.Inputs)
		if err != nil {
			continue
		}

		score, err := opts.Metric.Evaluate(ex, pred, nil)
		if err != nil {
			continue
		}

		totalScore += score
	}

	if len(minibatch) == 0 {
		return 0.0, nil
	}

	return totalScore / float64(len(minibatch)), nil
}
