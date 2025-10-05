package chain

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	scheduler "github.com/lookatitude/beluga-ai/pkg/orchestration/internal/scheduler"
)

// SimpleChain provides a basic implementation of the Chain interface
type SimpleChain struct {
	config iface.ChainConfig
	memory memory.Memory
	tracer trace.Tracer
}

// NewSimpleChain creates a new SimpleChain
func NewSimpleChain(config iface.ChainConfig, memory memory.Memory, tracer trace.Tracer) *SimpleChain {
	return &SimpleChain{
		config: config,
		memory: memory,
		tracer: tracer,
	}
}

func (c *SimpleChain) GetInputKeys() []string {
	if len(c.config.InputKeys) > 0 {
		return c.config.InputKeys
	}
	return []string{"input"} // Default
}

func (c *SimpleChain) GetOutputKeys() []string {
	if len(c.config.OutputKeys) > 0 {
		return c.config.OutputKeys
	}
	return []string{"output"} // Default
}

func (c *SimpleChain) GetMemory() memory.Memory {
	return c.memory
}

func (c *SimpleChain) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	var span trace.Span
	if c.tracer != nil {
		ctx, span = c.tracer.Start(ctx, "chain.invoke",
			trace.WithAttributes(
				attribute.String("chain.name", c.config.Name),
				attribute.Int("chain.steps", len(c.config.Steps)),
			))
		defer span.End()
	}

	startTime := time.Now()
	var err error
	defer func() {
		duration := time.Since(startTime).Seconds()
		if c.tracer != nil && span != nil {
			if err != nil {
				span.RecordError(err)
			}
			span.SetAttributes(attribute.Float64("chain.duration", duration))
		}
	}()

	// Apply timeout if configured
	if c.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
		defer cancel()
	}

	// Prepare initial memory variables if memory is present
	memoryVariables := make(map[string]any)
	if c.memory != nil {
		inputMap, ok := input.(map[string]any)
		if !ok {
			inputStr, okStr := input.(string)
			if !okStr {
				err = iface.ErrInvalidConfig("chain.invoke", fmt.Errorf("chain input must be map[string]any or string when using memory, got %T", input))
				return nil, err
			}
			inputMap = map[string]any{c.GetInputKeys()[0]: inputStr}
		}
		memoryVariables, err = c.memory.LoadMemoryVariables(ctx, inputMap)
		if err != nil {
			err = iface.ErrExecutionFailed("chain.invoke", err)
			return nil, err
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
		if len(c.GetInputKeys()) == 1 {
			combinedInput[c.GetInputKeys()[0]] = v
		} else {
			err = iface.ErrInvalidConfig("chain.invoke", fmt.Errorf("string input provided but chain expects multiple input keys: %v", c.GetInputKeys()))
			return nil, err
		}
	default:
		err = iface.ErrInvalidConfig("chain.invoke", fmt.Errorf("unsupported chain input type: %T", input))
		return nil, err
	}

	currentStepOutput := any(combinedInput)

	// Execute steps sequentially
	for i, step := range c.config.Steps {
		stepCtx := ctx
		var stepSpan trace.Span
		if c.tracer != nil {
			stepCtx, stepSpan = c.tracer.Start(ctx, fmt.Sprintf("chain.step.%d", i),
				trace.WithAttributes(
					attribute.String("step.type", fmt.Sprintf("%T", step)),
					attribute.Int("step.index", i),
				))
			defer stepSpan.End()
		}
		stepStart := time.Now()

		// Execute step with retry logic
		var stepOutput any
		var attemptCount int64

		retryConfig := scheduler.RetryConfig{
			MaxAttempts:   c.config.Retries,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
			JitterFactor:  0.1,
		}
		retryExecutor := scheduler.NewRetryExecutor(retryConfig)

		err = retryExecutor.ExecuteWithRetry(stepCtx, func() error {
			atomic.AddInt64(&attemptCount, 1)
			var invokeErr error
			stepOutput, invokeErr = step.Invoke(stepCtx, currentStepOutput, options...)
			return invokeErr
		})

		// If we got an error and it's not retryable, don't retry
		if err != nil && !iface.IsRetryable(err) && attemptCount == 1 {
			// This is a non-retryable error, don't wrap it as a retry failure
			err = iface.ErrExecutionFailed("chain.invoke", fmt.Errorf("error in chain step %d (%T): %w", i, step, err))
			return nil, err
		}

		// Update current step output if successful
		if err == nil {
			currentStepOutput = stepOutput
		}

		if err != nil {
			stepDuration := time.Since(stepStart)
			if c.tracer != nil && stepSpan != nil {
				stepSpan.SetAttributes(
					attribute.Float64("step.duration", stepDuration.Seconds()),
					attribute.Int64("step.attempts", attemptCount),
					attribute.Bool("step.success", false),
				)
				stepSpan.RecordError(err)
			}
			err = iface.ErrExecutionFailed("chain.invoke", fmt.Errorf("error in chain step %d (%T) after %d attempts: %w", i, step, attemptCount, err))
			return nil, err
		}

		stepDuration := time.Since(stepStart)
		if c.tracer != nil && stepSpan != nil {
			stepSpan.SetAttributes(
				attribute.Float64("step.duration", stepDuration.Seconds()),
				attribute.Int64("step.attempts", attemptCount),
				attribute.Bool("step.success", true),
			)
		}
	}

	finalOutput := currentStepOutput

	// Save context if memory is present
	if c.memory != nil {
		outputMap, ok := finalOutput.(map[string]any)
		if !ok {
			if len(c.GetOutputKeys()) == 1 {
				outputMap = map[string]any{c.GetOutputKeys()[0]: finalOutput}
			} else {
				err = iface.ErrInvalidConfig("chain.invoke", fmt.Errorf("chain final output type %T cannot be saved to memory expecting multiple output keys: %v", finalOutput, c.GetOutputKeys()))
				return nil, err
			}
		}
		inputMap := combinedInput

		err = c.memory.SaveContext(ctx, inputMap, outputMap)
		if err != nil {
			err = iface.ErrExecutionFailed("chain.invoke", fmt.Errorf("failed to save context to memory: %w", err))
			return nil, err
		}
	}

	return finalOutput, nil
}

func (c *SimpleChain) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	var lastErr error

	for i, input := range inputs {
		output, err := c.Invoke(ctx, input, options...)
		if err != nil {
			lastErr = iface.ErrExecutionFailed("chain.batch", fmt.Errorf("error processing batch item %d: %w", i, err))
		}
		results[i] = output
	}

	return results, lastErr
}

func (c *SimpleChain) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	// Basic stream implementation: only streams the last step if it supports streaming
	if len(c.config.Steps) == 0 {
		return nil, iface.ErrInvalidConfig("chain.stream", fmt.Errorf("cannot stream an empty chain"))
	}

	// Execute all steps except the last one
	var intermediateOutput any = input
	var err error
	for i := 0; i < len(c.config.Steps)-1; i++ {
		step := c.config.Steps[i]
		intermediateOutput, err = step.Invoke(ctx, intermediateOutput, options...)
		if err != nil {
			return nil, iface.ErrExecutionFailed("chain.stream", fmt.Errorf("error in chain stream pre-computation step %d (%T): %w", i, step, err))
		}
	}

	// Stream the last step
	lastStep := c.config.Steps[len(c.config.Steps)-1]
	return lastStep.Stream(ctx, intermediateOutput, options...)
}

// Ensure SimpleChain implements the Chain interface
var _ iface.Chain = (*SimpleChain)(nil)
var _ core.Runnable = (*SimpleChain)(nil)
