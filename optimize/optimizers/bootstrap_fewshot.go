package optimizers

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/optimize"
)

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

	// Temperature for teacher generation (higher = more diverse).
	Temperature float64

	// MetricThreshold is the minimum score to accept a demonstration.
	MetricThreshold float64
}

// Config holds BootstrapFewShot configuration.
type BootstrapFewShotConfig struct {
	Teacher           optimize.Program
	MaxBootstrapped   int
	MaxLabeled        int
	MaxRounds         int
	Temperature       float64
	MetricThreshold   float64
}

// NewBootstrapFewShot creates a new BootstrapFewShot optimizer.
func NewBootstrapFewShot(config BootstrapFewShotConfig) *BootstrapFewShot {
	if config.MaxBootstrapped == 0 {
		config.MaxBootstrapped = 4
	}
	if config.MaxLabeled == 0 {
		config.MaxLabeled = 16
	}
	if config.MaxRounds == 0 {
		config.MaxRounds = 1
	}
	if config.Temperature == 0 {
		config.Temperature = 1.0
	}
	if config.MetricThreshold == 0 {
		config.MetricThreshold = 1.0 // Default: require perfect match
	}

	return &BootstrapFewShot{
		Teacher:           config.Teacher,
		MaxBootstrapped:   config.MaxBootstrapped,
		MaxLabeled:        config.MaxLabeled,
		MaxRounds:         config.MaxRounds,
		Temperature:       config.Temperature,
		MetricThreshold:   config.MetricThreshold,
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
