// Package executor provides streaming executor implementation.
// This file implements the StreamingExecutor interface for streaming plan execution.
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// StreamingAgentExecutor implements the StreamingExecutor interface.
// It executes plans with streaming, sending chunks as each step is executed.
type StreamingAgentExecutor struct {
	*AgentExecutor
	maxIterations int
}

// NewStreamingAgentExecutor creates a new streaming executor.
func NewStreamingAgentExecutor(opts ...ExecutorOption) *StreamingAgentExecutor {
	baseExecutor := NewAgentExecutor(opts...)
	return &StreamingAgentExecutor{
		AgentExecutor: baseExecutor,
		maxIterations: 15,
	}
}

// ExecuteStreamingPlan implements the StreamingExecutor interface.
// It executes a plan with streaming, sending chunks for each step.
func (e *StreamingAgentExecutor) ExecuteStreamingPlan(ctx context.Context, agent iface.StreamingAgent, plan []schema.Step) (<-chan ExecutionChunk, error) {
	// Validate inputs
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if agent == nil {
		return nil, fmt.Errorf("agent cannot be nil")
	}
	if len(plan) == 0 {
		return nil, fmt.Errorf("plan cannot be empty")
	}

	// Create output channel with buffer
	outputChan := make(chan ExecutionChunk, 10)

	// Start execution in a goroutine
	go func() {
		defer close(outputChan)
		startTime := time.Now()

		// Send initial chunk indicating execution started
		outputChan <- ExecutionChunk{
			Content:   fmt.Sprintf("Starting execution of plan with %d steps", len(plan)),
			Timestamp: time.Now(),
		}

		var intermediateSteps []iface.IntermediateStep

		// Execute each step sequentially
		for i, step := range plan {
			// Check for cancellation
			select {
			case <-ctx.Done():
				outputChan <- ExecutionChunk{
					Err:       fmt.Errorf("execution cancelled at step %d: %w", i+1, ctx.Err()),
					Timestamp: time.Now(),
				}
				return
			default:
			}

			// Check max iterations
			if e.maxIterations > 0 && i >= e.maxIterations {
				outputChan <- ExecutionChunk{
					Err:       fmt.Errorf("execution failed: maximum iterations (%d) exceeded", e.maxIterations),
					Timestamp: time.Now(),
				}
				return
			}

			// Send chunk for step start
			stepStartChunk := ExecutionChunk{
				Step:      step,
				Content:   fmt.Sprintf("Executing step %d/%d: %s", i+1, len(plan), step.Action.Tool),
				Timestamp: time.Now(),
			}
			select {
			case outputChan <- stepStartChunk:
			case <-ctx.Done():
				outputChan <- ExecutionChunk{
					Err:       fmt.Errorf("execution cancelled: %w", ctx.Err()),
					Timestamp: time.Now(),
				}
				return
			}

			// Execute the step
			observation, err := e.executeStep(ctx, agent, step)
			stepDuration := time.Since(stepStartChunk.Timestamp)

			if err != nil {
				// Send error chunk
				outputChan <- ExecutionChunk{
					Step:      step,
					Err:       fmt.Errorf("step %d execution failed: %w", i+1, err),
					Timestamp: time.Now(),
				}
				return
			}

			// Record intermediate step
			intermediateStep := iface.IntermediateStep{
				Action:      iface.AgentAction(step.Action),
				Observation: observation,
			}
			intermediateSteps = append(intermediateSteps, intermediateStep)

			// Create tool result if step executed a tool
			var toolResult *ToolExecutionResult
			if step.Action.Tool != "" {
				// Find tool for metadata
				var toolName string
				for _, tool := range agent.GetTools() {
					if tool.Name() == step.Action.Tool {
						toolName = tool.Name()
						break
					}
				}

				// Record tool execution metrics
				if agent.GetMetrics() != nil {
					agent.GetMetrics().RecordToolCall(ctx, toolName, stepDuration, err == nil)
				}

				// Convert tool input/output to map
				inputMap := make(map[string]any)
				if step.Action.ToolInput != nil {
					inputMap["input"] = step.Action.ToolInput
				}

				outputMap := make(map[string]any)
				if observation != "" {
					outputMap["output"] = observation
				}

				toolResult = &ToolExecutionResult{
					ToolName: toolName,
					Input:    inputMap,
					Output:   outputMap,
					Duration: stepDuration,
				}
			}

			// Send step completion chunk
			stepChunk := ExecutionChunk{
				Step:       step,
				Content:    observation,
				ToolResult: toolResult,
				Timestamp:  time.Now(),
			}

			select {
			case outputChan <- stepChunk:
			case <-ctx.Done():
				outputChan <- ExecutionChunk{
					Err:       fmt.Errorf("execution cancelled: %w", ctx.Err()),
					Timestamp: time.Now(),
				}
				return
			}
		}

		// All steps completed - send final answer chunk
		finalOutput := ""
		if len(intermediateSteps) > 0 {
			finalOutput = intermediateSteps[len(intermediateSteps)-1].Observation
		}

		finalAnswer := &schema.FinalAnswer{
			Output: finalOutput,
		}

		if e.returnIntermediateSteps {
			finalAnswer.IntermediateSteps = convertToSchemaSteps(intermediateSteps)
		}

		totalDuration := time.Since(startTime)

		// Send final chunk
		outputChan <- ExecutionChunk{
			FinalAnswer: finalAnswer,
			Timestamp:   time.Now(),
		}

		// Record metrics if available
		if agent.GetMetrics() != nil {
			agent.GetMetrics().RecordExecutorRun(ctx, "streaming_agent_executor", totalDuration, len(plan), true)
		}
	}()

	return outputChan, nil
}

// Ensure StreamingAgentExecutor implements StreamingExecutor interface.
var _ StreamingExecutor = (*StreamingAgentExecutor)(nil)
