// Package chains defines interfaces and implementations for sequences of calls (chains)
// that can be composed together to create complex workflows.
package chains

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

// Chain represents a sequence of calls that can be executed together.
// Chains allow for composing multiple operations into a single, reusable unit.
type Chain interface {
	core.Runnable

	// AddStep adds a new step to the chain.
	AddStep(step ChainStep) Chain

	// GetSteps returns all steps in the chain.
	GetSteps() []ChainStep

	// GetMemory returns the memory module used by the chain.
	GetMemory() Memory

	// GetConfig returns the chain's configuration.
	GetConfig() ChainConfig
}

// ChainStep represents a single step in a chain.
type ChainStep interface {
	// Execute runs this step with the given inputs.
	Execute(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error)

	// GetName returns the name of this step.
	GetName() string

	// GetInputs returns the expected input keys for this step.
	GetInputs() []string

	// GetOutputs returns the output keys produced by this step.
	GetOutputs() []string
}

// ChainConfig holds configuration for chain execution.
type ChainConfig struct {
	Name             string
	Description      string
	MaxExecutionTime int // in seconds
	RetryPolicy      RetryPolicy
	ErrorHandling    ErrorHandlingStrategy
}

// RetryPolicy defines how the chain should handle retries.
type RetryPolicy struct {
	MaxRetries int
	BaseDelay  int // in milliseconds
	MaxDelay   int // in milliseconds
}

// ErrorHandlingStrategy defines how errors should be handled.
type ErrorHandlingStrategy int

const (
	// StopOnError stops the chain execution on the first error.
	StopOnError ErrorHandlingStrategy = iota
	// ContinueOnError continues execution but collects errors.
	ContinueOnError
	// RetryOnError retries failed steps according to the retry policy.
	RetryOnError
)

// Memory represents the memory component for chains.
type Memory interface {
	// LoadVariables loads memory variables for the given inputs.
	LoadVariables(inputs map[string]interface{}) (map[string]interface{}, error)

	// SaveContext saves the context of the chain execution.
	SaveContext(inputs, outputs map[string]interface{}) error

	// Clear clears the memory.
	Clear() error
}
