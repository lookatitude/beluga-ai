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
	Timestamp   time.Time
	Err         error
	ToolResult  *ToolExecutionResult
	FinalAnswer *schema.FinalAnswer
	Step        schema.Step
	Content     string
}

// ToolExecutionResult represents the result of tool execution during streaming.
type ToolExecutionResult struct {
	Err      error
	Input    map[string]any
	Output   map[string]any
	ToolName string
	Duration time.Duration
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
