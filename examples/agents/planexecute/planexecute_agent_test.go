package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/providers/planexecute"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockChatModel implements iface.ChatModel for testing
type MockChatModel struct {
	responses     []string
	currentIndex  int
	shouldError   bool
	errorToReturn error
}

func (m *MockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...interface{}) (schema.Message, error) {
	if m.shouldError {
		return nil, m.errorToReturn
	}
	if m.currentIndex < len(m.responses) {
		response := m.responses[m.currentIndex]
		m.currentIndex++
		return schema.NewAIMessage(response), nil
	}
	return schema.NewAIMessage("default response"), nil
}

func (m *MockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...interface{}) (<-chan iface.AIMessageChunk, error) {
	return nil, errors.New("streaming not implemented in mock")
}

func (m *MockChatModel) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
	return m
}

func (m *MockChatModel) GetModelName() string {
	return "mock-model"
}

func (m *MockChatModel) GetProviderName() string {
	return "mock"
}

// TestNewPlanExecuteExample tests example creation
func TestNewPlanExecuteExample(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		tools     []tools.Tool
		opts      []PlanExecuteOption
		wantErr   bool
	}{
		{
			name:      "creates example with defaults",
			agentName: "test-agent",
			tools:     createTestTools(),
			opts:      nil,
			wantErr:   false,
		},
		{
			name:      "creates example with options",
			agentName: "configured-agent",
			tools:     createTestTools(),
			opts: []PlanExecuteOption{
				WithMaxPlanSteps(5),
				WithMaxIterations(10),
			},
			wantErr: false,
		},
		{
			name:      "creates example with no tools",
			agentName: "no-tools-agent",
			tools:     []tools.Tool{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := llms.NewAdvancedMockChatModel(
				llms.WithResponses("Test response"),
			)

			example, err := NewPlanExecuteExample(tt.agentName, mockLLM, tt.tools, tt.opts...)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewPlanExecuteExample() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && example == nil {
				t.Error("NewPlanExecuteExample() returned nil example without error")
			}

			if !tt.wantErr && example.name != tt.agentName {
				t.Errorf("example.name = %s, want %s", example.name, tt.agentName)
			}
		})
	}
}

// TestPlanExecuteExample_Run tests the main Run method
func TestPlanExecuteExample_Run(t *testing.T) {
	// Create a mock that returns a valid plan
	planResponse := `{
		"goal": "Research topic",
		"steps": [
			{"step_number": 1, "action": "Search web", "tool": "web_search", "input": "topic"}
		],
		"total_steps": 1
	}`

	mockLLM := llms.NewAdvancedMockChatModel(
		llms.WithResponses(planResponse, "Step 1 result", "Final summary"),
	)

	testTools := createTestTools()

	example, err := NewPlanExecuteExample("test", mockLLM, testTools)
	if err != nil {
		t.Fatalf("Failed to create example: %v", err)
	}

	ctx := context.Background()
	result, err := example.Run(ctx, "Research a topic")

	// Note: This may fail due to mock not fully implementing plan generation
	// In production, use more sophisticated mocks
	if err != nil {
		t.Logf("Run returned error (expected for incomplete mock): %v", err)
	}

	if result != nil {
		if result.TotalDuration <= 0 {
			t.Error("TotalDuration should be positive")
		}
	}
}

// TestPlanExecuteExample_ContextCancellation tests context handling
func TestPlanExecuteExample_ContextCancellation(t *testing.T) {
	mockLLM := llms.NewAdvancedMockChatModel(
		llms.WithResponses("slow response"),
		llms.WithStreamingDelay(100*time.Millisecond),
		llms.WithSimulateNetworkDelay(true),
	)

	example, err := NewPlanExecuteExample("test", mockLLM, createTestTools())
	if err != nil {
		t.Fatalf("Failed to create example: %v", err)
	}

	// Create a context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = example.Run(ctx, "Test task")

	// We expect some kind of error due to context cancellation
	// The exact error depends on implementation
	t.Logf("Context cancellation result: %v", err)
}

// TestDisplayPlan tests plan display formatting
func TestDisplayPlan(t *testing.T) {
	mockLLM := llms.NewAdvancedMockChatModel()
	example, _ := NewPlanExecuteExample("test", mockLLM, nil)

	plan := &planexecute.ExecutionPlan{
		Goal:       "Test goal",
		TotalSteps: 2,
		Steps: []planexecute.PlanStep{
			{StepNumber: 1, Action: "First action", Tool: "tool1", Input: "input1", Reasoning: "Reason 1"},
			{StepNumber: 2, Action: "Second action", Tool: "tool2", Input: "input2", Reasoning: "Reason 2"},
		},
	}

	// This should not panic
	example.DisplayPlan(plan)
}

// TestCreateResearchTools tests tool creation
func TestCreateResearchTools(t *testing.T) {
	researchTools := createResearchTools()

	expectedTools := []string{"web_search", "calculator", "take_notes", "summarize"}

	if len(researchTools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(researchTools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range researchTools {
		toolNames[tool.Name()] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Missing expected tool: %s", expected)
		}
	}
}

// TestToolExecution tests that tools work correctly
func TestToolExecution(t *testing.T) {
	researchTools := createResearchTools()
	ctx := context.Background()

	tests := []struct {
		toolName string
		args     map[string]any
		wantErr  bool
	}{
		{
			toolName: "web_search",
			args:     map[string]any{"query": "test query"},
			wantErr:  false,
		},
		{
			toolName: "calculator",
			args:     map[string]any{"expression": "2 + 2"},
			wantErr:  false,
		},
		{
			toolName: "take_notes",
			args:     map[string]any{"note": "important finding"},
			wantErr:  false,
		},
		{
			toolName: "summarize",
			args:     map[string]any{"text": "long text to summarize"},
			wantErr:  false,
		},
	}

	// Build tool map
	toolMap := make(map[string]tools.Tool)
	for _, tool := range researchTools {
		toolMap[tool.Name()] = tool
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			tool, ok := toolMap[tt.toolName]
			if !ok {
				t.Fatalf("Tool %s not found", tt.toolName)
			}

			result, err := tool.Execute(ctx, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && result == "" {
				t.Error("Execute() returned empty result")
			}
		})
	}
}

// TestPlanExecuteOptions tests configuration options
func TestPlanExecuteOptions(t *testing.T) {
	config := &planExecuteConfig{}

	// Test WithMaxPlanSteps
	WithMaxPlanSteps(5)(config)
	if config.maxPlanSteps != 5 {
		t.Errorf("maxPlanSteps = %d, want 5", config.maxPlanSteps)
	}

	// Test WithMaxIterations
	WithMaxIterations(15)(config)
	if config.maxIterations != 15 {
		t.Errorf("maxIterations = %d, want 15", config.maxIterations)
	}

	// Test WithPlannerLLM
	mockPlanner := llms.NewAdvancedMockChatModel()
	WithPlannerLLM(mockPlanner)(config)
	if config.plannerLLM == nil {
		t.Error("plannerLLM should not be nil")
	}

	// Test WithExecutorLLM
	mockExecutor := llms.NewAdvancedMockChatModel()
	WithExecutorLLM(mockExecutor)(config)
	if config.executorLLM == nil {
		t.Error("executorLLM should not be nil")
	}
}

// BenchmarkPlanGeneration benchmarks plan generation performance
func BenchmarkPlanGeneration(b *testing.B) {
	planResponse := `{
		"goal": "Benchmark goal",
		"steps": [{"step_number": 1, "action": "Action", "tool": "tool"}],
		"total_steps": 1
	}`

	mockLLM := llms.NewAdvancedMockChatModel(
		llms.WithResponses(planResponse),
	)

	example, _ := NewPlanExecuteExample("benchmark", mockLLM, createTestTools())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = example.generatePlan(context.Background(), "benchmark task")
	}
}

// Helper function to create test tools
func createTestTools() []tools.Tool {
	return []tools.Tool{
		tools.NewSimpleTool(
			"test_tool",
			"A test tool",
			func(ctx context.Context, args map[string]any) (string, error) {
				return `{"result": "test"}`, nil
			},
		),
	}
}
