package optimizers

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/optimize"
)

func init() {
	optimize.RegisterOptimizer("bootstrapfewshot", func(cfg optimize.OptimizerConfig) (optimize.Optimizer, error) {
		return NewBootstrapFewShot(), nil
	})
}

// BootstrapFewShot bootstraps few-shot examples from training data.
// It uses a teacher program to generate demonstrations and filters by metric threshold.
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
}

// Option configures a BootstrapFewShot optimizer.
type Option func(*BootstrapFewShot)

// NewBootstrapFewShot creates a new BootstrapFewShot optimizer with defaults.
func NewBootstrapFewShot(opts ...Option) *BootstrapFewShot {
	bs := &BootstrapFewShot{
		MaxBootstrapped: 4,
		MaxLabeled:      16,
		MaxRounds:       1,
		MetricThreshold: 1.0,
	}
	for _, opt := range opts {
		opt(bs)
	}
	return bs
}

// WithTeacher sets the teacher program for bootstrapping.
func WithTeacher(teacher optimize.Program) Option {
	return func(bs *BootstrapFewShot) {
		bs.Teacher = teacher
	}
}

// WithMaxBootstrapped sets the maximum number of bootstrapped examples.
func WithMaxBootstrapped(n int) Option {
	return func(bs *BootstrapFewShot) {
		if n > 0 {
			bs.MaxBootstrapped = n
		}
	}
}

// WithMaxLabeled sets the maximum number of labeled examples.
func WithMaxLabeled(n int) Option {
	return func(bs *BootstrapFewShot) {
		if n > 0 {
			bs.MaxLabeled = n
		}
	}
}

// WithMaxRounds sets the number of bootstrap attempts per example.
func WithMaxRounds(n int) Option {
	return func(bs *BootstrapFewShot) {
		if n > 0 {
			bs.MaxRounds = n
		}
	}
}

// WithMetricThreshold sets the minimum score to accept a demonstration.
func WithMetricThreshold(threshold float64) Option {
	return func(bs *BootstrapFewShot) {
		if threshold >= 0 {
			bs.MetricThreshold = threshold
		}
	}
}

// Compile implements optimize.Optimizer.
func (bs *BootstrapFewShot) Compile(ctx context.Context, student optimize.Program, opts optimize.CompileOptions) (optimize.Program, error) {
	if opts.Metric == nil {
		return nil, fmt.Errorf("metric is required")
	}

	teacher := bs.Teacher
	if teacher == nil {
		teacher = student
	}

	// Bootstrap demonstrations
	demos := []optimize.Example{}
	for _, ex := range opts.Trainset {
		if len(demos) >= bs.MaxBootstrapped+bs.MaxLabeled {
			break
		}

		// Try to bootstrap from this example
		for round := 0; round < bs.MaxRounds; round++ {
			// Run teacher on input
			pred, err := teacher.Run(ctx, ex.Inputs)
			if err != nil {
				continue
			}

			// Check if prediction passes metric
			score, err := opts.Metric.Evaluate(ex, pred, nil)
			if err != nil {
				continue
			}

			if score >= bs.MetricThreshold {
				// Create demonstration from this successful trace
				demo := optimize.Example{
					Inputs:  ex.Inputs,
					Outputs: pred.Outputs,
				}
				demos = append(demos, demo)
				break // Move to next training example
			}
		}
	}

	// If we didn't get enough bootstrapped examples, add labeled examples
	if len(demos) < bs.MaxBootstrapped+bs.MaxLabeled {
		for _, ex := range opts.Trainset {
			if len(demos) >= bs.MaxBootstrapped+bs.MaxLabeled {
				break
			}
			// Check if already in demos
			if !containsExample(demos, ex) {
				demos = append(demos, ex)
			}
		}
	}

	// Limit to MaxBootstrapped + MaxLabeled
	if len(demos) > bs.MaxBootstrapped+bs.MaxLabeled {
		demos = demos[:bs.MaxBootstrapped+bs.MaxLabeled]
	}

	// Return student with demonstrations
	return student.WithDemos(demos), nil
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
