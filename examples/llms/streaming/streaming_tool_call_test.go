package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/tools"
)

// TestStreamingToolCallExample_Run tests the main Run method with various scenarios.
func TestStreamingToolCallExample_Run(t *testing.T) {
	tests := []struct {
		name            string
		prompt          string
		setupMock       func() iface.ChatModel
		wantToolsCalled int
		wantChunks      int
		wantErr         bool
	}{
		{
			name:   "simple response without tool calls",
			prompt: "Hello there!",
			setupMock: func() iface.ChatModel {
				return llms.NewAdvancedMockChatModel(
					llms.WithResponses("Hello! How can I help you today?"),
				)
			},
			wantToolsCalled: 0,
			wantChunks:      7, // Based on word count in response
			wantErr:         false,
		},
		{
			name:   "handles empty response gracefully",
			prompt: "Test empty",
			setupMock: func() iface.ChatModel {
				return llms.NewAdvancedMockChatModel(
					llms.WithResponses(""),
				)
			},
			wantToolsCalled: 0,
			wantChunks:      0,
			wantErr:         false,
		},
	}

	// Create a simple test tool
	testTool := tools.NewSimpleTool(
		"test_tool",
		"A test tool",
		func(ctx context.Context, args map[string]any) (string, error) {
			return `{"result": "test"}`, nil
		},
		tools.WithParameter("input", "string", "Test input", false),
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock()
			example := NewStreamingToolCallExample(mockClient, []tools.Tool{testTool})

			result, err := example.Run(context.Background(), tt.prompt)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(result.ToolsCalled) != tt.wantToolsCalled {
					t.Errorf("ToolsCalled = %d, want %d", len(result.ToolsCalled), tt.wantToolsCalled)
				}
				if result.TotalDuration <= 0 {
					t.Error("TotalDuration should be positive")
				}
			}
		})
	}
}

// TestStreamingToolCallExample_ContextCancellation verifies proper context handling.
func TestStreamingToolCallExample_ContextCancellation(t *testing.T) {
	mockClient := llms.NewAdvancedMockChatModel(
		llms.WithResponses("This is a response that takes a while"),
		llms.WithStreamingDelay(100*time.Millisecond),
		llms.WithSimulateNetworkDelay(true),
	)

	example := NewStreamingToolCallExample(mockClient, nil)

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := example.Run(ctx, "Test prompt")

	// We expect an error due to context cancellation
	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}
}

// TestStreamingToolCallExample_ErrorHandling tests various error scenarios.
func TestStreamingToolCallExample_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() iface.ChatModel
		wantErr   bool
		errMsg    string
	}{
		{
			name: "stream start failure",
			setupMock: func() iface.ChatModel {
				mock := llms.NewAdvancedMockChatModel()
				mock.SetShouldError(true)
				mock.SetErrorToReturn(errors.New("connection refused"))
				return mock
			},
			wantErr: true,
			errMsg:  "failed to start stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock()
			example := NewStreamingToolCallExample(mockClient, nil)

			_, err := example.Run(context.Background(), "test")

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExecuteToolCalls tests the tool execution logic.
func TestExecuteToolCalls(t *testing.T) {
	// Create test tools
	successTool := tools.NewSimpleTool(
		"success_tool",
		"A tool that succeeds",
		func(ctx context.Context, args map[string]any) (string, error) {
			return `{"status": "ok"}`, nil
		},
	)

	failTool := tools.NewSimpleTool(
		"fail_tool",
		"A tool that fails",
		func(ctx context.Context, args map[string]any) (string, error) {
			return "", errors.New("tool failure")
		},
	)

	tests := []struct {
		name       string
		toolCalls  []schema.ToolCallChunk
		tools      []tools.Tool
		wantCount  int
		wantErr    bool
		wantNames  []string
	}{
		{
			name: "successful tool call",
			toolCalls: []schema.ToolCallChunk{
				{ID: "1", Name: "success_tool", Arguments: `{}`},
			},
			tools:     []tools.Tool{successTool},
			wantCount: 1,
			wantErr:   false,
			wantNames: []string{"success_tool"},
		},
		{
			name: "unknown tool",
			toolCalls: []schema.ToolCallChunk{
				{ID: "1", Name: "unknown_tool", Arguments: `{}`},
			},
			tools:     []tools.Tool{successTool},
			wantCount: 1,
			wantErr:   true,
			wantNames: []string{"unknown_tool"},
		},
		{
			name: "tool execution failure",
			toolCalls: []schema.ToolCallChunk{
				{ID: "1", Name: "fail_tool", Arguments: `{}`},
			},
			tools:     []tools.Tool{failTool},
			wantCount: 1,
			wantErr:   true,
			wantNames: []string{"fail_tool"},
		},
		{
			name: "invalid JSON arguments",
			toolCalls: []schema.ToolCallChunk{
				{ID: "1", Name: "success_tool", Arguments: `{invalid json}`},
			},
			tools:     []tools.Tool{successTool},
			wantCount: 1,
			wantErr:   true,
			wantNames: []string{"success_tool"},
		},
		{
			name: "multiple tools mixed success",
			toolCalls: []schema.ToolCallChunk{
				{ID: "1", Name: "success_tool", Arguments: `{}`},
				{ID: "2", Name: "fail_tool", Arguments: `{}`},
			},
			tools:     []tools.Tool{successTool, failTool},
			wantCount: 2,
			wantErr:   true, // At least one failed
			wantNames: []string{"success_tool", "fail_tool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := llms.NewAdvancedMockChatModel()
			example := NewStreamingToolCallExample(mockClient, tt.tools)

			results, names, err := example.executeToolCalls(context.Background(), tt.toolCalls)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeToolCalls() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(results) != tt.wantCount {
				t.Errorf("results count = %d, want %d", len(results), tt.wantCount)
			}

			if len(names) != len(tt.wantNames) {
				t.Errorf("names count = %d, want %d", len(names), len(tt.wantNames))
			}

			for i, name := range names {
				if i < len(tt.wantNames) && name != tt.wantNames[i] {
					t.Errorf("name[%d] = %s, want %s", i, name, tt.wantNames[i])
				}
			}
		})
	}
}

// TestProcessStream tests the stream processing logic.
func TestProcessStream(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func() iface.ChatModel
		wantResponse string
		wantChunks   int
		wantErr      bool
	}{
		{
			name: "processes multi-word response",
			setupMock: func() iface.ChatModel {
				return llms.NewAdvancedMockChatModel(
					llms.WithResponses("Hello world from Beluga"),
				)
			},
			wantResponse: "Hello world from Beluga ",
			wantChunks:   4,
			wantErr:      false,
		},
		{
			name: "handles single word response",
			setupMock: func() iface.ChatModel {
				return llms.NewAdvancedMockChatModel(
					llms.WithResponses("OK"),
				)
			},
			wantResponse: "OK ",
			wantChunks:   1,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock()
			example := NewStreamingToolCallExample(mockClient, nil)

			// Bind tools (empty list) to get a model we can stream from
			model := mockClient.BindTools(nil)

			text, _, chunks, err := example.processStream(
				context.Background(),
				model,
				[]schema.Message{schema.NewHumanMessage("test")},
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("processStream() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && text != tt.wantResponse {
				t.Errorf("response = %q, want %q", text, tt.wantResponse)
			}

			if !tt.wantErr && chunks != tt.wantChunks {
				t.Errorf("chunks = %d, want %d", chunks, tt.wantChunks)
			}
		})
	}
}

// TestCreateWeatherTool verifies the weather tool works correctly.
func TestCreateWeatherTool(t *testing.T) {
	tool := createWeatherTool()

	if tool.Name() != "get_weather" {
		t.Errorf("Name() = %s, want get_weather", tool.Name())
	}

	// Test successful execution
	result, err := tool.Execute(context.Background(), map[string]any{
		"location": "San Francisco, CA",
	})
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Execute() returned empty result")
	}

	// Test missing location
	_, err = tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("Execute() expected error for missing location")
	}
}

// TestCreateCalculatorTool verifies the calculator tool works correctly.
func TestCreateCalculatorTool(t *testing.T) {
	tool := createCalculatorTool()

	if tool.Name() != "calculator" {
		t.Errorf("Name() = %s, want calculator", tool.Name())
	}

	// Test successful execution
	result, err := tool.Execute(context.Background(), map[string]any{
		"expression": "2 + 2",
	})
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Execute() returned empty result")
	}

	// Test missing expression
	_, err = tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("Execute() expected error for missing expression")
	}
}

// BenchmarkStreamProcessing benchmarks the stream processing performance.
func BenchmarkStreamProcessing(b *testing.B) {
	mockClient := llms.NewAdvancedMockChatModel(
		llms.WithResponses("This is a longer response with multiple words to simulate real streaming behavior"),
	)

	example := NewStreamingToolCallExample(mockClient, nil)
	model := mockClient.BindTools(nil)
	messages := []schema.Message{schema.NewHumanMessage("test")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = example.processStream(context.Background(), model, messages)
	}
}

// BenchmarkToolExecution benchmarks tool execution performance.
func BenchmarkToolExecution(b *testing.B) {
	tool := createWeatherTool()
	mockClient := llms.NewAdvancedMockChatModel()
	example := NewStreamingToolCallExample(mockClient, []tools.Tool{tool})

	toolCalls := []schema.ToolCallChunk{
		{ID: "1", Name: "get_weather", Arguments: `{"location": "NYC"}`},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = example.executeToolCalls(context.Background(), toolCalls)
	}
}
