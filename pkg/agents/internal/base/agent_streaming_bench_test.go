// Package base provides benchmarks for streaming agent operations.
package base

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// BenchmarkStreamExecute_Basic benchmarks basic StreamExecute operation.
func BenchmarkStreamExecute_Basic(b *testing.B) {
	llm := &mockStreamingChatModel{
		responses:      []string{"Hello", "world", "!"},
		streamingDelay: 1 * time.Millisecond,
	}
	agent, err := NewBaseAgent("bench-agent", llm, nil, WithStreaming(true))
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	// BaseAgent implements StreamingAgent when streaming is enabled
	var streamingAgent iface.StreamingAgent = agent
	ctx := context.Background()
	inputs := map[string]any{"input": "test input"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := streamingAgent.StreamExecute(ctx, inputs)
		if err != nil {
			b.Fatalf("StreamExecute failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
	}
}

// BenchmarkStreamExecute_WithTools benchmarks StreamExecute with tool calls.
func BenchmarkStreamExecute_WithTools(b *testing.B) {
	llm := &mockStreamingChatModel{
		responses:      []string{"Hello", "world", "!"},
		streamingDelay: 1 * time.Millisecond,
		toolCallChunks: []schema.ToolCallChunk{
			{Name: "test_tool", Arguments: `{"arg1":"value1"}`},
		},
	}
	agent, err := NewBaseAgent("bench-agent", llm, nil, WithStreaming(true))
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	// BaseAgent implements StreamingAgent when streaming is enabled
	var streamingAgent iface.StreamingAgent = agent
	ctx := context.Background()
	inputs := map[string]any{"input": "test input"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := streamingAgent.StreamExecute(ctx, inputs)
		if err != nil {
			b.Fatalf("StreamExecute failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
	}
}

// BenchmarkStreamExecute_LargeResponse benchmarks StreamExecute with large responses.
func BenchmarkStreamExecute_LargeResponse(b *testing.B) {
	// Create a large response with many chunks
	responses := make([]string, 100)
	for i := range responses {
		responses[i] = "chunk"
	}

	llm := &mockStreamingChatModel{
		responses:      responses,
		streamingDelay: 100 * time.Microsecond,
	}
	agent, err := NewBaseAgent("bench-agent", llm, nil, WithStreaming(true))
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	// BaseAgent implements StreamingAgent when streaming is enabled
	var streamingAgent iface.StreamingAgent = agent
	ctx := context.Background()
	inputs := map[string]any{"input": "test input"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := streamingAgent.StreamExecute(ctx, inputs)
		if err != nil {
			b.Fatalf("StreamExecute failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
	}
}

// BenchmarkStreamPlan benchmarks StreamPlan operation.
func BenchmarkStreamPlan(b *testing.B) {
	llm := &mockStreamingChatModel{
		responses:      []string{"Plan", "step", "1"},
		streamingDelay: 1 * time.Millisecond,
	}
	agent, err := NewBaseAgent("bench-agent", llm, nil, WithStreaming(true))
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	// BaseAgent implements StreamingAgent when streaming is enabled
	var streamingAgent iface.StreamingAgent = agent
	ctx := context.Background()
	inputs := map[string]any{"input": "test input"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := streamingAgent.StreamPlan(ctx, nil, inputs)
		if err != nil {
			b.Fatalf("StreamPlan failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
	}
}

// BenchmarkStreamExecute_Concurrent benchmarks concurrent StreamExecute operations.
func BenchmarkStreamExecute_Concurrent(b *testing.B) {
	llm := &mockStreamingChatModel{
		responses:      []string{"Hello", "world"},
		streamingDelay: 1 * time.Millisecond,
	}
	agent, err := NewBaseAgent("bench-agent", llm, nil, WithStreaming(true))
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	// BaseAgent implements StreamingAgent when streaming is enabled
	var streamingAgent iface.StreamingAgent = agent
	ctx := context.Background()
	inputs := map[string]any{"input": "test input"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch, err := streamingAgent.StreamExecute(ctx, inputs)
			if err != nil {
				b.Fatalf("StreamExecute failed: %v", err)
			}
			// Consume all chunks
			for range ch {
			}
		}
	})
}

// WithStreaming is a helper function for enabling streaming in agent options.
func WithStreaming(enabled bool) iface.Option {
	return func(o *iface.Options) {
		o.StreamingConfig.EnableStreaming = enabled
		if enabled && o.StreamingConfig.ChunkBufferSize == 0 {
			o.StreamingConfig.ChunkBufferSize = 20
		}
		if enabled && o.StreamingConfig.MaxStreamDuration == 0 {
			o.StreamingConfig.MaxStreamDuration = 30 * time.Minute
		}
	}
}
