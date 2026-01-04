// Package executor provides streaming executor interfaces for agents.
// This file extends the Executor interface with streaming capabilities.
package executor

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ExecutionChunk represents a chunk of execution output during streaming plan execution.
// It contains incremental results as each step of the plan is executed.
type ExecutionChunk struct {
	// Step is the current step being executed
	Step schema.Step

	// Content is the text content from this step's execution
	Content string

	// ToolResult is the tool execution result if this step executed a tool
	ToolResult *ToolExecutionResult

	// FinalAnswer is the final answer if execution is complete
	FinalAnswer *schema.FinalAnswer

	// Err is an error if one occurred (execution ends on error)
	Err error

	// Timestamp is the chunk timestamp for latency measurement
	Timestamp time.Time
}

// ToolExecutionResult represents the result of tool execution during streaming.
type ToolExecutionResult struct {
	// ToolName is the name of the tool executed
	ToolName string

	// Input is the tool input that was used
	Input map[string]any

	// Output is the tool output result
	Output map[string]any

	// Duration is how long the tool execution took
	Duration time.Duration

	// Err is an error if tool execution failed
	Err error
}

// StreamingExecutor extends Executor with streaming execution capabilities.
// This interface enables real-time streaming of plan execution, which is essential
// for voice interactions where users need immediate feedback.
//
// The streaming executor executes plans step-by-step, streaming content from each
// step as it becomes available. This allows for incremental updates during tool
// execution and LLM responses.
//
// Example usage:
//
//	executor := NewStreamingExecutor(...)
//	chunkChan, err := executor.ExecuteStreamingPlan(ctx, streamingAgent, plan)
//	if err != nil {
//	    return err
//	}
//	for chunk := range chunkChan {
//	    if chunk.Err != nil {
//	        return chunk.Err
//	    }
//	    // Process chunk content, tool results, etc.
//	    if chunk.FinalAnswer != nil {
//	        // Execution complete
//	        break
//	    }
//	}
type StreamingExecutor interface {
	// Executor embeds the existing Executor interface
	iface.Executor

	// ExecuteStreamingPlan executes a plan with streaming LLM responses.
	// Returns a channel of ExecutionChunk that will be closed when execution completes.
	//
	// The executor will:
	//   - Execute steps sequentially (one at a time)
	//   - Stream content from each step as it arrives
	//   - Execute tools when steps require tool execution
	//   - Include tool results in chunks
	//   - Send a final chunk with either FinalAnswer or Err set
	//   - Respect context cancellation (close stream on ctx.Done())
	//
	// Input validation:
	//   - ctx must not be nil
	//   - agent must implement StreamingAgent interface
	//   - plan must not be empty
	//   - plan steps must be valid (as defined by schema)
	//
	// Performance:
	//   - First chunk should arrive within 200ms
	//   - Step execution should not block on streaming
	ExecuteStreamingPlan(ctx context.Context, agent iface.StreamingAgent, plan []schema.Step) (<-chan ExecutionChunk, error)
}
