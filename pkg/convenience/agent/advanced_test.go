package agent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// TestAgent_Run tests the Run method with various scenarios.
func TestAgent_Run(t *testing.T) {
	tests := []struct {
		name           string
		setupAgent     func() (Agent, error)
		input          string
		expectedOutput string
		expectedCode   string
		wantErr        bool
	}{
		{
			name: "successful run with LLM",
			setupAgent: func() (Agent, error) {
				mockLLM := NewMockLLM()
				mockLLM.SetResponse("Hello, human!")
				return NewBuilder().WithLLM(mockLLM).Build(context.Background())
			},
			input:          "Hello",
			expectedOutput: "Hello, human!",
			wantErr:        false,
		},
		{
			name: "successful run with ChatModel",
			setupAgent: func() (Agent, error) {
				mockChatModel := NewMockChatModel()
				mockChatModel.SetGenerateResponse("AI response")
				return NewBuilder().WithChatModel(mockChatModel).Build(context.Background())
			},
			input:          "Test input",
			expectedOutput: "AI response",
			wantErr:        false,
		},
		{
			name: "LLM error",
			setupAgent: func() (Agent, error) {
				mockLLM := NewMockLLM()
				mockLLM.SetNextError(ErrMockLLM)
				return NewBuilder().WithLLM(mockLLM).Build(context.Background())
			},
			input:        "Hello",
			expectedCode: ErrCodeExecution,
			wantErr:      true,
		},
		{
			name: "ChatModel error",
			setupAgent: func() (Agent, error) {
				mockChatModel := NewMockChatModel()
				mockChatModel.SetGenerateError(ErrMockLLM)
				return NewBuilder().WithChatModel(mockChatModel).Build(context.Background())
			},
			input:        "Hello",
			expectedCode: ErrCodeExecution,
			wantErr:      true,
		},
		{
			name: "with system prompt",
			setupAgent: func() (Agent, error) {
				mockChatModel := NewMockChatModel()
				return NewBuilder().
					WithChatModel(mockChatModel).
					WithSystemPrompt("You are a helpful assistant").
					Build(context.Background())
			},
			input:          "Hello",
			expectedOutput: "Mock AI response",
			wantErr:        false,
		},
		{
			name: "with memory load error",
			setupAgent: func() (Agent, error) {
				mockLLM := NewMockLLM()
				mockMemory := NewMockMemory()
				mockMemory.SetLoadError(ErrMockMemory)
				return NewBuilder().
					WithLLM(mockLLM).
					WithMemory(mockMemory).
					Build(context.Background())
			},
			input:        "Hello",
			expectedCode: ErrCodeExecution,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := tt.setupAgent()
			if err != nil {
				t.Fatalf("failed to setup agent: %v", err)
			}

			output, err := agent.Run(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.expectedCode != "" {
					code := GetErrorCode(err)
					if code != tt.expectedCode {
						t.Errorf("expected error code %s, got %s", tt.expectedCode, code)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, output)
			}
		})
	}
}

// TestAgent_RunWithInputs tests the RunWithInputs method.
func TestAgent_RunWithInputs(t *testing.T) {
	tests := []struct {
		name           string
		setupAgent     func() (Agent, error)
		inputs         map[string]any
		expectedOutput string
		wantErr        bool
	}{
		{
			name: "with input key",
			setupAgent: func() (Agent, error) {
				mockChatModel := NewMockChatModel()
				mockChatModel.SetGenerateResponse("Response to input")
				return NewBuilder().WithChatModel(mockChatModel).Build(context.Background())
			},
			inputs:         map[string]any{"input": "Hello"},
			expectedOutput: "Response to input",
			wantErr:        false,
		},
		{
			name: "with query key",
			setupAgent: func() (Agent, error) {
				mockChatModel := NewMockChatModel()
				mockChatModel.SetGenerateResponse("Response to query")
				return NewBuilder().WithChatModel(mockChatModel).Build(context.Background())
			},
			inputs:         map[string]any{"query": "What is AI?"},
			expectedOutput: "Response to query",
			wantErr:        false,
		},
		{
			name: "with arbitrary key",
			setupAgent: func() (Agent, error) {
				mockChatModel := NewMockChatModel()
				mockChatModel.SetGenerateResponse("Generic response")
				return NewBuilder().WithChatModel(mockChatModel).Build(context.Background())
			},
			inputs:         map[string]any{"custom": "Custom input"},
			expectedOutput: "Generic response",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := tt.setupAgent()
			if err != nil {
				t.Fatalf("failed to setup agent: %v", err)
			}

			outputs, err := agent.RunWithInputs(context.Background(), tt.inputs)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output, ok := outputs["output"].(string)
			if !ok {
				t.Fatal("expected 'output' key in outputs")
			}
			if output != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, output)
			}
		})
	}
}

// TestAgent_Stream tests the Stream method.
func TestAgent_Stream(t *testing.T) {
	t.Run("successful streaming", func(t *testing.T) {
		mockChatModel := NewMockChatModel()
		mockChatModel.StreamChatFunc = func(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
			ch := make(chan llmsiface.AIMessageChunk)
			go func() {
				defer close(ch)
				ch <- llmsiface.AIMessageChunk{Content: "Hello "}
				ch <- llmsiface.AIMessageChunk{Content: "World"}
			}()
			return ch, nil
		}

		agent, err := NewBuilder().WithChatModel(mockChatModel).Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build agent: %v", err)
		}

		ch, err := agent.Stream(context.Background(), "Hello")
		if err != nil {
			t.Fatalf("failed to start stream: %v", err)
		}

		var content string
		var contentSb233 strings.Builder
		for chunk := range ch {
			if chunk.Error != nil {
				t.Fatalf("unexpected error in stream: %v", chunk.Error)
			}
			contentSb233.WriteString(chunk.Content)
		}
		content += contentSb233.String()

		if content != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", content)
		}
	})

	t.Run("stream error", func(t *testing.T) {
		mockChatModel := NewMockChatModel()
		mockChatModel.StreamChatFunc = func(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
			return nil, errors.New("stream error")
		}

		agent, err := NewBuilder().WithChatModel(mockChatModel).Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build agent: %v", err)
		}

		_, err = agent.Stream(context.Background(), "Hello")
		if err == nil {
			t.Error("expected error but got none")
		}
	})

	t.Run("fallback to non-streaming LLM", func(t *testing.T) {
		mockLLM := NewMockLLM()
		mockLLM.SetResponse("Non-streaming response")

		agent, err := NewBuilder().WithLLM(mockLLM).Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build agent: %v", err)
		}

		ch, err := agent.Stream(context.Background(), "Hello")
		if err != nil {
			t.Fatalf("failed to start stream: %v", err)
		}

		var content string
		for chunk := range ch {
			if chunk.Error != nil {
				t.Fatalf("unexpected error: %v", chunk.Error)
			}
			content = chunk.Content
		}

		if content != "Non-streaming response" {
			t.Errorf("expected 'Non-streaming response', got %q", content)
		}
	})
}

// TestAgent_Memory tests memory integration.
func TestAgent_Memory(t *testing.T) {
	t.Run("memory save after run", func(t *testing.T) {
		mockChatModel := NewMockChatModel()
		mockChatModel.SetGenerateResponse("AI response")
		mockMemory := NewMockMemory()

		agent, err := NewBuilder().
			WithChatModel(mockChatModel).
			WithMemory(mockMemory).
			Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build agent: %v", err)
		}

		_, err = agent.Run(context.Background(), "Hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		inputs, outputs := mockMemory.GetSavedContexts()
		if len(inputs) != 1 {
			t.Errorf("expected 1 saved input, got %d", len(inputs))
		}
		if len(outputs) != 1 {
			t.Errorf("expected 1 saved output, got %d", len(outputs))
		}

		if inputs[0]["input"] != "Hello" {
			t.Errorf("expected input 'Hello', got %v", inputs[0]["input"])
		}
		if outputs[0]["output"] != "AI response" {
			t.Errorf("expected output 'AI response', got %v", outputs[0]["output"])
		}
	})

	t.Run("memory with history", func(t *testing.T) {
		mockChatModel := NewMockChatModel()
		mockMemory := NewMockMemory()
		history := []schema.Message{
			schema.NewHumanMessage("Previous question"),
			schema.NewAIMessage("Previous answer"),
		}
		// Set up LoadMemoryFunc to return history
		mockMemory.LoadMemoryFunc = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			return map[string]any{"history": history}, nil
		}

		agent, err := NewBuilder().
			WithChatModel(mockChatModel).
			WithMemory(mockMemory).
			Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build agent: %v", err)
		}

		_, err = agent.Run(context.Background(), "Follow-up")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify chat model received history
		calls := mockChatModel.GetGenerateCalls()
		if len(calls) != 1 {
			t.Fatalf("expected 1 generate call, got %d", len(calls))
		}

		// Messages should include history plus new input
		messages := calls[0]
		if len(messages) < 3 {
			t.Errorf("expected at least 3 messages (history + new input), got %d", len(messages))
		}
	})
}

// TestAgent_Shutdown tests the Shutdown method.
func TestAgent_Shutdown(t *testing.T) {
	t.Run("shutdown clears memory", func(t *testing.T) {
		mockLLM := NewMockLLM()
		mockMemory := NewMockMemory()

		agent, err := NewBuilder().
			WithLLM(mockLLM).
			WithMemory(mockMemory).
			Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build agent: %v", err)
		}

		err = agent.Shutdown()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("shutdown without memory", func(t *testing.T) {
		mockLLM := NewMockLLM()

		agent, err := NewBuilder().
			WithLLM(mockLLM).
			Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build agent: %v", err)
		}

		err = agent.Shutdown()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestAgent_Timeout tests timeout functionality.
func TestAgent_Timeout(t *testing.T) {
	t.Run("execution within timeout", func(t *testing.T) {
		mockLLM := NewMockLLM()

		agent, err := NewBuilder().
			WithLLM(mockLLM).
			WithTimeout(1 * time.Second).
			Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build agent: %v", err)
		}

		_, err = agent.Run(context.Background(), "Hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestAgent_Concurrent tests concurrent access to the agent.
func TestAgent_Concurrent(t *testing.T) {
	mockChatModel := NewMockChatModel()
	mockChatModel.SetGenerateResponse("Concurrent response")

	agent, err := NewBuilder().
		WithChatModel(mockChatModel).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := agent.Run(context.Background(), "Concurrent request")
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent run failed: %v", err)
	}
}

// TestAgent_Tools tests tool configuration.
func TestAgent_Tools(t *testing.T) {
	tool1 := NewMockTool("calculator", "Performs calculations")
	tool2 := NewMockTool("weather", "Gets weather data")

	mockLLM := NewMockLLM()
	agent, err := NewBuilder().
		WithLLM(mockLLM).
		WithTool(tool1).
		WithTools([]core.Tool{tool2}).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}

	tools := agent.GetTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}
