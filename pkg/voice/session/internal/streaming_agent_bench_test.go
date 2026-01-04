// Package internal provides benchmarks for streaming agent operations in voice sessions.
package internal

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// mockStreamingAgentForVoiceBenchmark implements StreamingAgent for voice session benchmarking.
type mockStreamingAgentForVoiceBenchmark struct{}

func (m *mockStreamingAgentForVoiceBenchmark) StreamExecute(ctx context.Context, inputs map[string]any) (<-chan agentsiface.AgentStreamChunk, error) {
	ch := make(chan agentsiface.AgentStreamChunk, 10)
	go func() {
		defer close(ch)
		chunks := []string{"Hello", "world", "!"}
		for _, content := range chunks {
			ch <- agentsiface.AgentStreamChunk{Content: content}
			time.Sleep(1 * time.Millisecond)
		}
	}()
	return ch, nil
}

func (m *mockStreamingAgentForVoiceBenchmark) StreamPlan(ctx context.Context, intermediateSteps []agentsiface.IntermediateStep, inputs map[string]any) (<-chan agentsiface.AgentStreamChunk, error) {
	return m.StreamExecute(ctx, inputs)
}

func (m *mockStreamingAgentForVoiceBenchmark) GetConfig() schema.AgentConfig {
	return schema.AgentConfig{Name: "bench-agent"}
}

func (m *mockStreamingAgentForVoiceBenchmark) GetTools() []tools.Tool {
	return nil
}

func (m *mockStreamingAgentForVoiceBenchmark) GetMetrics() agentsiface.MetricsRecorder {
	return nil
}

func (m *mockStreamingAgentForVoiceBenchmark) GetLLM() llmsiface.LLM {
	return nil
}

func (m *mockStreamingAgentForVoiceBenchmark) InputVariables() []string {
	return []string{"input"}
}

func (m *mockStreamingAgentForVoiceBenchmark) OutputVariables() []string {
	return []string{"output"}
}

func (m *mockStreamingAgentForVoiceBenchmark) Execute(ctx context.Context, inputs map[string]any, options ...agentsiface.Option) (any, error) {
	return "result", nil
}

func (m *mockStreamingAgentForVoiceBenchmark) Plan(ctx context.Context, intermediateSteps []agentsiface.IntermediateStep, inputs map[string]any) (agentsiface.AgentAction, agentsiface.AgentFinish, error) {
	return agentsiface.AgentAction{}, agentsiface.AgentFinish{}, nil
}

// mockTTSProviderForBenchmark implements TTSProvider for benchmarking.
type mockTTSProviderForBenchmark struct{}

func (m *mockTTSProviderForBenchmark) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3, 4, 5}, nil
}

func (m *mockTTSProviderForBenchmark) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return strings.NewReader(text), nil
}

// BenchmarkStartStreaming_Basic benchmarks basic StartStreaming operation.
func BenchmarkStartStreaming_Basic(b *testing.B) {
	agent := &mockStreamingAgentForVoiceBenchmark{}
	agentInstance := NewAgentInstance(agent, schema.AgentConfig{Name: "bench-agent"})
	ttsProvider := &mockTTSProviderForBenchmark{}
	config := DefaultStreamingAgentConfig()

	streamingAgent := NewStreamingAgent(agentInstance, ttsProvider, config)
	ctx := context.Background()
	transcript := "test input"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := streamingAgent.StartStreaming(ctx, transcript)
		if err != nil {
			b.Fatalf("StartStreaming failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
		streamingAgent.StopStreaming()
	}
}

// BenchmarkStartStreaming_LargeTranscript benchmarks StartStreaming with large transcript.
func BenchmarkStartStreaming_LargeTranscript(b *testing.B) {
	agent := &mockStreamingAgentForVoiceBenchmark{}
	agentInstance := NewAgentInstance(agent, schema.AgentConfig{Name: "bench-agent"})
	ttsProvider := &mockTTSProviderForBenchmark{}
	config := DefaultStreamingAgentConfig()

	streamingAgent := NewStreamingAgent(agentInstance, ttsProvider, config)
	ctx := context.Background()
	// Create a large transcript
	transcript := ""
	for i := 0; i < 1000; i++ {
		transcript += "word "
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := streamingAgent.StartStreaming(ctx, transcript)
		if err != nil {
			b.Fatalf("StartStreaming failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
		streamingAgent.StopStreaming()
	}
}

// BenchmarkStartStreaming_Concurrent benchmarks concurrent StartStreaming operations.
func BenchmarkStartStreaming_Concurrent(b *testing.B) {
	agent := &mockStreamingAgentForVoiceBenchmark{}
	agentInstance := NewAgentInstance(agent, schema.AgentConfig{Name: "bench-agent"})
	ttsProvider := &mockTTSProviderForBenchmark{}
	config := DefaultStreamingAgentConfig()

	streamingAgent := NewStreamingAgent(agentInstance, ttsProvider, config)
	ctx := context.Background()
	transcript := "test input"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch, err := streamingAgent.StartStreaming(ctx, transcript)
			if err != nil {
				b.Fatalf("StartStreaming failed: %v", err)
			}
			// Consume all chunks
			for range ch {
			}
			streamingAgent.StopStreaming()
		}
	})
}

// BenchmarkChunkProcessing benchmarks chunk processing performance.
func BenchmarkChunkProcessing(b *testing.B) {
	agent := &mockStreamingAgentForVoiceBenchmark{}
	agentInstance := NewAgentInstance(agent, schema.AgentConfig{Name: "bench-agent"})
	ttsProvider := &mockTTSProviderForBenchmark{}
	config := DefaultStreamingAgentConfig()

	streamingAgent := NewStreamingAgent(agentInstance, ttsProvider, config)
	ctx := context.Background()
	transcript := "test input"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := streamingAgent.StartStreaming(ctx, transcript)
		if err != nil {
			b.Fatalf("StartStreaming failed: %v", err)
		}
		// Process chunks
		chunkCount := 0
		for range ch {
			chunkCount++
		}
		streamingAgent.StopStreaming()
	}
}
