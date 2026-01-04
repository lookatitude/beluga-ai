// Package executor provides benchmarks for streaming executor operations.
package executor

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// mockStreamingAgentForBenchmark implements StreamingAgent for benchmarking.
type mockStreamingAgentForBenchmark struct {
	tools []tools.Tool
}

func (m *mockStreamingAgentForBenchmark) StreamExecute(ctx context.Context, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	ch := make(chan iface.AgentStreamChunk, 10)
	go func() {
		defer close(ch)
		ch <- iface.AgentStreamChunk{Content: "response"}
	}()
	return ch, nil
}

func (m *mockStreamingAgentForBenchmark) StreamPlan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	return m.StreamExecute(ctx, inputs)
}

func (m *mockStreamingAgentForBenchmark) GetConfig() schema.AgentConfig {
	return schema.AgentConfig{Name: "bench-agent"}
}

func (m *mockStreamingAgentForBenchmark) GetTools() []tools.Tool {
	return m.tools
}

func (m *mockStreamingAgentForBenchmark) GetMetrics() iface.MetricsRecorder {
	return nil
}

func (m *mockStreamingAgentForBenchmark) GetLLM() llmsiface.LLM {
	return nil
}

func (m *mockStreamingAgentForBenchmark) InputVariables() []string {
	return []string{"input"}
}

func (m *mockStreamingAgentForBenchmark) OutputVariables() []string {
	return []string{"output"}
}

func (m *mockStreamingAgentForBenchmark) Execute(ctx context.Context, inputs map[string]any, options ...iface.Option) (any, error) {
	return "result", nil
}

func (m *mockStreamingAgentForBenchmark) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	return iface.AgentAction{}, iface.AgentFinish{}, nil
}

// mockToolForBenchmark implements Tool for benchmarking.
type mockToolForBenchmark struct {
	name string
}

func (m *mockToolForBenchmark) Name() string {
	return m.name
}

func (m *mockToolForBenchmark) Description() string {
	return "mock tool"
}

func (m *mockToolForBenchmark) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: "mock tool",
	}
}

func (m *mockToolForBenchmark) Execute(ctx context.Context, input any) (any, error) {
	time.Sleep(1 * time.Millisecond) // Simulate tool execution
	return "tool result", nil
}

func (m *mockToolForBenchmark) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := m.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

// BenchmarkExecuteStreamingPlan_Basic benchmarks basic ExecuteStreamingPlan operation.
func BenchmarkExecuteStreamingPlan_Basic(b *testing.B) {
	agent := &mockStreamingAgentForBenchmark{}
	executor := NewStreamingAgentExecutor()

	plan := []schema.Step{
		{Action: schema.AgentAction{Tool: "test_tool", ToolInput: map[string]any{"input": "value"}}},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := executor.ExecuteStreamingPlan(ctx, agent, plan)
		if err != nil {
			b.Fatalf("ExecuteStreamingPlan failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
	}
}

// BenchmarkExecuteStreamingPlan_MultipleSteps benchmarks ExecuteStreamingPlan with multiple steps.
func BenchmarkExecuteStreamingPlan_MultipleSteps(b *testing.B) {
	agent := &mockStreamingAgentForBenchmark{
		tools: []tools.Tool{&mockToolForBenchmark{name: "test_tool"}},
	}
	executor := NewStreamingAgentExecutor()

	plan := make([]schema.Step, 10)
	for i := range plan {
		plan[i] = schema.Step{
			Action: schema.AgentAction{
				Tool:      "test_tool",
				ToolInput: map[string]any{"step": i},
			},
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := executor.ExecuteStreamingPlan(ctx, agent, plan)
		if err != nil {
			b.Fatalf("ExecuteStreamingPlan failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
	}
}

// BenchmarkExecuteStreamingPlan_Concurrent benchmarks concurrent ExecuteStreamingPlan operations.
func BenchmarkExecuteStreamingPlan_Concurrent(b *testing.B) {
	agent := &mockStreamingAgentForBenchmark{
		tools: []tools.Tool{&mockToolForBenchmark{name: "test_tool"}},
	}
	executor := NewStreamingAgentExecutor()

	plan := []schema.Step{
		{Action: schema.AgentAction{Tool: "test_tool", ToolInput: map[string]any{"input": "value"}}},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch, err := executor.ExecuteStreamingPlan(ctx, agent, plan)
			if err != nil {
				b.Fatalf("ExecuteStreamingPlan failed: %v", err)
			}
			// Consume all chunks
			for range ch {
			}
		}
	})
}

// BenchmarkExecuteStreamingPlan_LargePlan benchmarks ExecuteStreamingPlan with a large plan.
func BenchmarkExecuteStreamingPlan_LargePlan(b *testing.B) {
	agent := &mockStreamingAgentForBenchmark{
		tools: []tools.Tool{&mockToolForBenchmark{name: "test_tool"}},
	}
	executor := NewStreamingAgentExecutor()

	plan := make([]schema.Step, 100)
	for i := range plan {
		plan[i] = schema.Step{
			Action: schema.AgentAction{
				Tool:      "test_tool",
				ToolInput: map[string]any{"step": i},
			},
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, err := executor.ExecuteStreamingPlan(ctx, agent, plan)
		if err != nil {
			b.Fatalf("ExecuteStreamingPlan failed: %v", err)
		}
		// Consume all chunks
		for range ch {
		}
	}
}
