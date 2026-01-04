// Package iface defines streaming interfaces for agents.
// This file extends the Agent interface with streaming capabilities for real-time interactions.
package iface

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// AgentStreamChunk represents a chunk of agent execution output during streaming.
// It contains incremental results as they become available from the LLM or agent processing.
type AgentStreamChunk struct {
	// Content is the text content from LLM (can be partial)
	Content string

	// ToolCalls contains tool calls if any (may be partial)
	ToolCalls []schema.ToolCall

	// Action is the next action if determined during planning
	Action *AgentAction

	// Finish is the final result if execution is complete
	Finish *AgentFinish

	// Err is an error if one occurred (stream ends on error)
	Err error

	// Metadata contains additional metadata (latency, timestamps, etc.)
	Metadata map[string]any
}

// StreamingConfig extends AgentConfig with streaming-specific settings.
// It controls how streaming behavior works for agents.
type StreamingConfig struct {
	// EnableStreaming enables streaming mode for the agent
	EnableStreaming bool

	// ChunkBufferSize is the buffer size for chunks (must be > 0 and <= 100)
	ChunkBufferSize int

	// SentenceBoundary indicates whether to wait for sentence boundaries before processing
	SentenceBoundary bool

	// InterruptOnNewInput allows interruption on new input
	InterruptOnNewInput bool

	// MaxStreamDuration is the maximum duration a stream can run
	MaxStreamDuration time.Duration
}

// StreamingAgent extends Agent with streaming execution capabilities.
// This interface enables real-time streaming of agent responses, which is essential
// for voice interactions where low latency is critical.
//
// Example usage:
//
//	agent := NewBaseAgent(...)
//	streamingAgent := agent.(StreamingAgent) // If agent implements StreamingAgent
//	chunkChan, err := streamingAgent.StreamExecute(ctx, inputs)
//	if err != nil {
//	    return err
//	}
//	for chunk := range chunkChan {
//	    if chunk.Err != nil {
//	        return chunk.Err
//	    }
//	    // Process chunk content, tool calls, etc.
//	    if chunk.Finish != nil {
//	        // Execution complete
//	        break
//	    }
//	}
type StreamingAgent interface {
	// Agent embeds the existing Agent interface
	Agent

	// StreamExecute executes the agent with streaming LLM responses.
	// Returns a channel of AgentStreamChunk that will be closed when execution completes.
	//
	// The channel will receive chunks as they become available from the LLM.
	// The stream must:
	//   - Start immediately (no blocking on first chunk)
	//   - Send chunks as soon as available
	//   - Close when execution completes or an error occurs
	//   - Include tool calls when detected
	//   - Send a final chunk with either Finish or Err set
	//   - Respect context cancellation (close stream on ctx.Done())
	//
	// Input validation:
	//   - ctx must not be nil
	//   - inputs must contain all required input variables (as defined by InputVariables())
	//
	// Performance:
	//   - First chunk should arrive within 200ms
	//   - Subsequent chunks should arrive within 100ms of previous chunk
	StreamExecute(ctx context.Context, inputs map[string]any) (<-chan AgentStreamChunk, error)

	// StreamPlan plans the next action with streaming model responses.
	// Returns a channel of AgentStreamChunk that will be closed when planning completes.
	//
	// This method is similar to StreamExecute but focuses on planning rather than execution.
	// It streams the planning process, allowing for incremental planning decisions.
	//
	// The channel behavior is the same as StreamExecute:
	//   - Streams chunks as planning progresses
	//   - Can include Action chunks for next steps
	//   - Final chunk has Finish or Err set
	//   - Respects context cancellation
	StreamPlan(ctx context.Context, intermediateSteps []IntermediateStep, inputs map[string]any) (<-chan AgentStreamChunk, error)
}
