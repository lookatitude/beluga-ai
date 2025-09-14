// Package orchestrator defines interfaces and components for managing complex flows
// involving multiple steps, agents, or tools. This includes concepts like chains and graphs.
package orchestrator

import (
	"context"
	"errors" // Added missing import
	"fmt"    // Added missing import
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory"
)

// Chain represents a sequence of components (Runnables) executed one after another.
// The output of one step is typically the input to the next.
type Chain interface {
	core.Runnable // Chains themselves are Runnable

	// GetInputKeys returns the expected input keys for the chain.
	GetInputKeys() []string
	// GetOutputKeys returns the keys produced by the chain.
	GetOutputKeys() []string
	// GetMemory returns the memory associated with the chain, if any.
	GetMemory() memory.BaseMemory
}

// Graph represents a more complex orchestration where components can be executed
// based on dependencies or conditions, forming a directed acyclic graph (DAG).
type Graph interface {
	core.Runnable // Graphs are also Runnable

	// AddNode adds a Runnable component as a node in the graph.
	AddNode(name string, runnable core.Runnable) error
	// AddEdge defines a dependency between two nodes.
	// The output of the source node might be used as input for the target node.
	AddEdge(sourceNode string, targetNode string) error
	// SetEntryPoint defines the starting node(s) of the graph.
	SetEntryPoint(nodeNames []string) error
	// SetFinishPoint defines the final node(s) whose output is the graph's output.
	SetFinishPoint(nodeNames []string) error
}

// Workflow represents a long-running, potentially distributed orchestration.
// This interface can be implemented using systems like Temporal.
type Workflow interface {
	// Execute starts the workflow execution.
	// Input can be specific to the workflow definition.
	// Returns a unique ID for the workflow instance and an error.
	Execute(ctx context.Context, input any) (workflowID string, runID string, err error)

	// GetResult retrieves the final result of a completed workflow instance.
	// Blocks until the workflow completes or the context times out.
	GetResult(ctx context.Context, workflowID string, runID string) (any, error)

	// Signal sends a signal to a running workflow instance.
	Signal(ctx context.Context, workflowID string, runID string, signalName string, data any) error

	// Query queries the state of a running workflow instance.
	Query(ctx context.Context, workflowID string, runID string, queryType string, args ...any) (any, error)

	// Cancel requests cancellation of a running workflow instance.
	Cancel(ctx context.Context, workflowID string, runID string) error

	// Terminate forcefully stops a running workflow instance.
	Terminate(ctx context.Context, workflowID string, runID string, reason string, details ...any) error
}

// Activity represents a unit of work within a workflow, often corresponding to a Beluga Runnable.
// This interface helps bridge Beluga components with workflow systems.
type Activity interface {
	// Execute performs the activity's logic.
	// Input/output types depend on the specific activity.
	Execute(ctx context.Context, input any) (any, error)
}

// --- Basic Chain Implementation ---

// SimpleChain provides a basic implementation of the Chain interface.
type SimpleChain struct {
	Steps []core.Runnable
	Mem   memory.BaseMemory
	// TODO: Define input/output keys more explicitly
}

// NewSimpleChain creates a new SimpleChain.
func NewSimpleChain(steps []core.Runnable, mem memory.BaseMemory) *SimpleChain {
	return &SimpleChain{
		Steps: steps,
		Mem:   mem,
	}
}

func (c *SimpleChain) GetInputKeys() []string {
	// TODO: Determine input keys from the first step or configuration
	return []string{"input"} // Placeholder
}

func (c *SimpleChain) GetOutputKeys() []string {
	// TODO: Determine output keys from the last step or configuration
	return []string{"output"} // Placeholder
}

func (c *SimpleChain) GetMemory() memory.BaseMemory {
	return c.Mem
}

func (c *SimpleChain) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	var err error

	// Prepare initial memory variables if memory is present
	memoryVariables := make(map[string]any)
	if c.Mem != nil {
		// Convert initial input to map if needed for memory loading
		inputMap, ok := input.(map[string]any)
		if !ok {
			inputStr, okStr := input.(string)
			if !okStr {
				return nil, fmt.Errorf("chain input must be map[string]any or string when using memory, got %T", input)
			}
			inputMap = map[string]any{c.GetInputKeys()[0]: inputStr} // Assume single input key
		}
		memoryVariables, err = c.Mem.LoadMemoryVariables(ctx, inputMap)
		if err != nil {
			return nil, fmt.Errorf("failed to load memory variables: %w", err)
		}
	}

	// Combine initial input and memory variables
	combinedInput := make(map[string]any)
	// Add memory variables first
	for k, v := range memoryVariables {
		combinedInput[k] = v
	}
	// Add/overwrite with direct input
	switch v := input.(type) {
	case map[string]any:
		for k, val := range v {
			combinedInput[k] = val
		}
	case string:
		// Assume single input key
		if len(c.GetInputKeys()) == 1 {
			combinedInput[c.GetInputKeys()[0]] = v
		} else {
			return nil, fmt.Errorf("string input provided but chain expects multiple input keys: %v", c.GetInputKeys())
		}
	default:
		return nil, fmt.Errorf("unsupported chain input type: %T", input)
	}

	currentStepOutput := any(combinedInput)

	// Execute steps sequentially
	for i, step := range c.Steps {
		currentStepOutput, err = step.Invoke(ctx, currentStepOutput, options...)
		if err != nil {
			return nil, fmt.Errorf("error in chain step %d (%T): %w", i, step, err)
		}
		// TODO: Handle non-map outputs between steps? For now, assume map or final output
	}

	finalOutput := currentStepOutput

	// Save context if memory is present
	if c.Mem != nil {
		// Ensure final output is a map for saving context
		outputMap, ok := finalOutput.(map[string]any)
		if !ok {
			// If final output isn't a map, wrap it using the chain's output key
			if len(c.GetOutputKeys()) == 1 {
				outputMap = map[string]any{c.GetOutputKeys()[0]: finalOutput}
			} else {
				return nil, fmt.Errorf("chain final output type %T cannot be saved to memory expecting multiple output keys: %v", finalOutput, c.GetOutputKeys())
			}
		}
		// No need to check if combinedInput is a map[string]any - it already is
		inputMap := combinedInput

		err = c.Mem.SaveContext(ctx, inputMap, outputMap)
		if err != nil {
			return nil, fmt.Errorf("failed to save context to memory: %w", err)
		}
	}

	return finalOutput, nil
}

func (c *SimpleChain) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	var firstErr error
	for i, input := range inputs {
		output, err := c.Invoke(ctx, input, options...)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("error processing batch item %d: %w", i, err)
		}
		results[i] = output
	}
	return results, firstErr
}

func (c *SimpleChain) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	// Basic stream implementation: only streams the *last* step if it supports streaming.
	// TODO: Implement more sophisticated chain streaming (streaming intermediate steps?)
	if len(c.Steps) == 0 {
		return nil, errors.New("cannot stream an empty chain")
	}

	// Execute all steps except the last one
	var intermediateOutput any = input
	var err error
	for i := 0; i < len(c.Steps)-1; i++ {
		step := c.Steps[i]
		intermediateOutput, err = step.Invoke(ctx, intermediateOutput, options...)
		if err != nil {
			return nil, fmt.Errorf("error in chain stream pre-computation step %d (%T): %w", i, step, err)
		}
	}

	// Stream the last step
	lastStep := c.Steps[len(c.Steps)-1]
	return lastStep.Stream(ctx, intermediateOutput, options...)
}

var _ Chain = (*SimpleChain)(nil)
var _ core.Runnable = (*SimpleChain)(nil)

// TODO:
// - Implement Graph orchestration (e.g., using a DAG library or manually)
// - Implement TemporalWorkflow adapter in orchestrator/temporal.go
// - Implement Activity wrapper for Beluga Runnables
