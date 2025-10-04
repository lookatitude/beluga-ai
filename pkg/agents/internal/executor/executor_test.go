package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// mockAgent implements the Agent interface for testing
type mockAgent struct {
	name        string
	llm         llmsiface.LLM
	tools       []tools.Tool
	shouldError bool
	planCount   int
}

func (m *mockAgent) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	m.planCount++
	if m.shouldError {
		return iface.AgentAction{}, iface.AgentFinish{}, errors.New("mock agent planning error")
	}

	if m.planCount >= 3 {
		// Return finish after a few calls
		return iface.AgentAction{}, iface.AgentFinish{
			ReturnValues: map[string]any{"output": "completed"},
			Log:          "Task completed successfully",
		}, nil
	}

	// Return action to continue
	return iface.AgentAction{
		Tool:      "test_tool",
		ToolInput: map[string]any{"input": "test"},
		Log:       "Planning next action",
	}, iface.AgentFinish{}, nil
}

func (m *mockAgent) InputVariables() []string {
	return []string{"input"}
}

func (m *mockAgent) OutputVariables() []string {
	return []string{"output"}
}

func (m *mockAgent) GetTools() []tools.Tool {
	return m.tools
}

func (m *mockAgent) GetConfig() schema.AgentConfig {
	return schema.AgentConfig{Name: m.name}
}

func (m *mockAgent) GetLLM() llmsiface.LLM {
	return m.llm
}

func (m *mockAgent) GetMetrics() iface.MetricsRecorder {
	return nil
}

// mockTool implements the Tool interface for testing
type mockTool struct {
	name        string
	result      any
	shouldError bool
	callCount   int
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return "Mock tool for testing"
}

func (m *mockTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: "Mock tool for testing",
	}
}

func (m *mockTool) Execute(ctx context.Context, input any) (any, error) {
	m.callCount++
	if m.shouldError {
		return nil, errors.New("mock tool execution error")
	}
	return m.result, nil
}

func (m *mockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
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

// mockLLM implements the LLM interface for testing
type mockLLM struct{}

func (m *mockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return "mock llm response", nil
}

func (m *mockLLM) GetModelName() string {
	return "mock-llm"
}

func (m *mockLLM) GetProviderName() string {
	return "mock-provider"
}

func TestNewAgentExecutor(t *testing.T) {
	executor := NewAgentExecutor()
	if executor == nil {
		t.Error("NewAgentExecutor() returned nil")
	}

	// Test default values
	if executor.maxIterations != 15 {
		t.Errorf("Expected default maxIterations 15, got %d", executor.maxIterations)
	}
	if executor.returnIntermediateSteps {
		t.Error("Expected default returnIntermediateSteps to be false")
	}
	if !executor.handleParsingErrors {
		t.Error("Expected default handleParsingErrors to be true")
	}
}

func TestNewAgentExecutorWithOptions(t *testing.T) {
	executor := NewAgentExecutor(
		WithMaxIterations(10),
		WithReturnIntermediateSteps(true),
		WithHandleParsingErrors(false),
	)

	if executor.maxIterations != 10 {
		t.Errorf("Expected maxIterations 10, got %d", executor.maxIterations)
	}
	if !executor.returnIntermediateSteps {
		t.Error("Expected returnIntermediateSteps to be true")
	}
	if executor.handleParsingErrors {
		t.Error("Expected handleParsingErrors to be false")
	}
}

func TestExecutorOptionFunctions(t *testing.T) {
	t.Run("WithMaxIterations", func(t *testing.T) {
		executor := NewAgentExecutor(WithMaxIterations(5))
		if executor.maxIterations != 5 {
			t.Errorf("Expected maxIterations 5, got %d", executor.maxIterations)
		}
	})

	t.Run("WithReturnIntermediateSteps", func(t *testing.T) {
		executor := NewAgentExecutor(WithReturnIntermediateSteps(true))
		if !executor.returnIntermediateSteps {
			t.Error("Expected returnIntermediateSteps to be true")
		}
	})

	t.Run("WithHandleParsingErrors", func(t *testing.T) {
		executor := NewAgentExecutor(WithHandleParsingErrors(false))
		if executor.handleParsingErrors {
			t.Error("Expected handleParsingErrors to be false")
		}
	})
}

func TestExecutePlan_Success(t *testing.T) {
	// Create mock agent and tool
	mockTool := &mockTool{name: "test_tool", result: "tool_result"}
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{mockTool},
	}

	executor := NewAgentExecutor(WithMaxIterations(5))

	// Create a simple plan
	plan := []schema.Step{
		{
			Action: schema.AgentAction{
				Tool:      "test_tool",
				ToolInput: map[string]any{"input": "test"},
			},
		},
	}

	ctx := context.Background()
	result, err := executor.ExecutePlan(ctx, agent, plan)

	if err != nil {
		t.Errorf("ExecutePlan failed: %v", err)
	}

	if result.Output != "tool_result" {
		t.Errorf("Expected output 'tool_result', got '%s'", result.Output)
	}

	if mockTool.callCount != 1 {
		t.Errorf("Expected tool to be called 1 time, got %d", mockTool.callCount)
	}
}

func TestExecutePlan_EmptyPlan(t *testing.T) {
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{},
	}

	executor := NewAgentExecutor()
	plan := []schema.Step{}

	ctx := context.Background()
	result, err := executor.ExecutePlan(ctx, agent, plan)

	if err != nil {
		t.Errorf("ExecutePlan with empty plan failed: %v", err)
	}

	if result.Output != "No steps to execute" {
		t.Errorf("Expected output 'No steps to execute', got '%s'", result.Output)
	}
}

func TestExecutePlan_MaxIterationsExceeded(t *testing.T) {
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{},
	}

	executor := NewAgentExecutor(WithMaxIterations(2))

	// Create a plan that will exceed max iterations
	plan := []schema.Step{
		{
			Action: schema.AgentAction{
				Tool: "non_existent_tool",
			},
		},
	}

	ctx := context.Background()
	_, err := executor.ExecutePlan(ctx, agent, plan)

	if err == nil {
		t.Error("Expected error for max iterations exceeded")
	}

	if !errors.Is(err, errors.New("execution failed")) {
		t.Errorf("Expected 'execution failed' error, got: %v", err)
	}
}

func TestExecutePlan_ToolNotFound(t *testing.T) {
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{},
	}

	executor := NewAgentExecutor()

	plan := []schema.Step{
		{
			Action: schema.AgentAction{
				Tool: "non_existent_tool",
			},
		},
	}

	ctx := context.Background()
	_, err := executor.ExecutePlan(ctx, agent, plan)

	if err == nil {
		t.Error("Expected error for tool not found")
	}

	if !errors.Is(err, errors.New("execution failed")) {
		t.Errorf("Expected 'execution failed' error, got: %v", err)
	}
}

func TestExecutePlan_ToolExecutionError(t *testing.T) {
	mockTool := &mockTool{name: "test_tool", shouldError: true}
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{mockTool},
	}

	executor := NewAgentExecutor()

	plan := []schema.Step{
		{
			Action: schema.AgentAction{
				Tool: "test_tool",
			},
		},
	}

	ctx := context.Background()
	_, err := executor.ExecutePlan(ctx, agent, plan)

	if err == nil {
		t.Error("Expected error for tool execution failure")
	}

	if !errors.Is(err, errors.New("execution failed")) {
		t.Errorf("Expected 'execution failed' error, got: %v", err)
	}
}

func TestExecutePlan_WithIntermediateSteps(t *testing.T) {
	mockTool := &mockTool{name: "test_tool", result: "step_result"}
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{mockTool},
	}

	executor := NewAgentExecutor(WithReturnIntermediateSteps(true))

	plan := []schema.Step{
		{
			Action: schema.AgentAction{
				Tool:      "test_tool",
				ToolInput: map[string]any{"input": "test"},
			},
		},
	}

	ctx := context.Background()
	result, err := executor.ExecutePlan(ctx, agent, plan)

	if err != nil {
		t.Errorf("ExecutePlan failed: %v", err)
	}

	if result.IntermediateSteps == nil {
		t.Error("Expected intermediate steps to be returned")
	}

	if len(result.IntermediateSteps) != 1 {
		t.Errorf("Expected 1 intermediate step, got %d", len(result.IntermediateSteps))
	}

	if result.IntermediateSteps[0].Observation.Output != "step_result" {
		t.Errorf("Expected observation 'step_result', got '%s'", result.IntermediateSteps[0].Observation.Output)
	}
}

func TestExecuteStep_ToolExecution(t *testing.T) {
	mockTool := &mockTool{name: "test_tool", result: "tool_output"}
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{mockTool},
	}

	executor := NewAgentExecutor()

	step := schema.Step{
		Action: schema.AgentAction{
			Tool:      "test_tool",
			ToolInput: map[string]any{"input": "test"},
		},
	}

	ctx := context.Background()
	observation, err := executor.executeStep(ctx, agent, step)

	if err != nil {
		t.Errorf("executeStep failed: %v", err)
	}

	if observation != "tool_output" {
		t.Errorf("Expected observation 'tool_output', got '%s'", observation)
	}
}

func TestExecuteStep_NonToolAction(t *testing.T) {
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{},
	}

	executor := NewAgentExecutor()

	step := schema.Step{
		Action: schema.AgentAction{
			Log: "Test log message",
		},
		Observation: schema.AgentObservation{
			Output: "Test observation",
		},
	}

	ctx := context.Background()
	observation, err := executor.executeStep(ctx, agent, step)

	if err != nil {
		t.Errorf("executeStep failed: %v", err)
	}

	if observation != "Test observation" {
		t.Errorf("Expected observation 'Test observation', got '%s'", observation)
	}
}

func TestExecuteTool_Success(t *testing.T) {
	mockTool := &mockTool{name: "test_tool", result: "success_result"}
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{mockTool},
	}

	executor := NewAgentExecutor()

	action := schema.AgentAction{
		Tool:      "test_tool",
		ToolInput: map[string]any{"input": "test"},
	}

	ctx := context.Background()
	result, err := executor.executeTool(ctx, agent, action)

	if err != nil {
		t.Errorf("executeTool failed: %v", err)
	}

	if result != "success_result" {
		t.Errorf("Expected result 'success_result', got '%s'", result)
	}
}

func TestExecuteTool_ToolNotFound(t *testing.T) {
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{},
	}

	executor := NewAgentExecutor()

	action := schema.AgentAction{
		Tool: "non_existent_tool",
	}

	ctx := context.Background()
	_, err := executor.executeTool(ctx, agent, action)

	if err == nil {
		t.Error("Expected error for tool not found")
	}

	expectedErr := "tool 'non_existent_tool' not found"
	if !errors.Is(err, errors.New(expectedErr)) && !errors.Is(err, errors.New("tool execution failed")) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestExecuteTool_ExecutionError(t *testing.T) {
	mockTool := &mockTool{name: "test_tool", shouldError: true}
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{mockTool},
	}

	executor := NewAgentExecutor()

	action := schema.AgentAction{
		Tool: "test_tool",
	}

	ctx := context.Background()
	_, err := executor.executeTool(ctx, agent, action)

	if err == nil {
		t.Error("Expected error for tool execution failure")
	}

	expectedErr := "tool execution failed"
	if !errors.Is(err, errors.New(expectedErr)) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestConvertToSchemaSteps(t *testing.T) {
	intermediateSteps := []iface.IntermediateStep{
		{
			Action: iface.AgentAction{
				Tool:      "tool1",
				ToolInput: map[string]any{"input": "value1"},
				Log:       "log1",
			},
			Observation: "observation1",
		},
		{
			Action: iface.AgentAction{
				Tool:      "tool2",
				ToolInput: map[string]any{"input": "value2"},
				Log:       "log2",
			},
			Observation: "observation2",
		},
	}

	steps := convertToSchemaSteps(intermediateSteps)

	if len(steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(steps))
	}

	if steps[0].Action.Tool != "tool1" {
		t.Errorf("Expected first step tool 'tool1', got '%s'", steps[0].Action.Tool)
	}

	if steps[0].Observation.Output != "observation1" {
		t.Errorf("Expected first step observation 'observation1', got '%s'", steps[0].Observation.Output)
	}

	if steps[1].Action.Tool != "tool2" {
		t.Errorf("Expected second step tool 'tool2', got '%s'", steps[1].Action.Tool)
	}

	if steps[1].Observation.Output != "observation2" {
		t.Errorf("Expected second step observation 'observation2', got '%s'", steps[1].Observation.Output)
	}
}

// Benchmark tests
func BenchmarkNewAgentExecutor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewAgentExecutor()
	}
}

func BenchmarkExecutePlan(b *testing.B) {
	mockTool := &mockTool{name: "test_tool", result: "result"}
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{mockTool},
	}

	executor := NewAgentExecutor()

	plan := []schema.Step{
		{
			Action: schema.AgentAction{
				Tool:      "test_tool",
				ToolInput: map[string]any{"input": "test"},
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecutePlan(ctx, agent, plan)
	}
}

func BenchmarkExecuteTool(b *testing.B) {
	mockTool := &mockTool{name: "test_tool", result: "result"}
	mockLLM := &mockLLM{}
	agent := &mockAgent{
		name:  "test_agent",
		llm:   mockLLM,
		tools: []tools.Tool{mockTool},
	}

	executor := NewAgentExecutor()

	action := schema.AgentAction{
		Tool:      "test_tool",
		ToolInput: map[string]any{"input": "test"},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.executeTool(ctx, agent, action)
	}
}
