package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatModel implements llm.ChatModel for testing.
type mockChatModel struct {
	responses []string
	callCount int
}

var _ llm.ChatModel = (*mockChatModel)(nil)

func (m *mockChatModel) Generate(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
	if m.callCount >= len(m.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return schema.NewAIMessage(resp), nil
}

func (m *mockChatModel) Stream(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {}
}

func (m *mockChatModel) BindTools(_ []schema.ToolDefinition) llm.ChatModel {
	return m
}

func (m *mockChatModel) ModelID() string { return "mock" }

func validGenResponse() string {
	gen := generatedTool{
		Code: `package main

import "fmt"

func run(input string) (string, error) {
	return fmt.Sprintf("processed: %s", input), nil
}
`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
		},
	}
	b, _ := json.Marshal(gen)
	return string(b)
}

func invalidCodeGenResponse() string {
	gen := generatedTool{
		Code: `package main

import "os/exec"

func run(input string) (string, error) {
	return "", nil
}
`,
		InputSchema: map[string]any{"type": "object"},
	}
	b, _ := json.Marshal(gen)
	return string(b)
}

func TestToolGenerator_Generate(t *testing.T) {
	t.Run("successful generation", func(t *testing.T) {
		mock := &mockChatModel{responses: []string{validGenResponse()}}
		gen := NewToolGenerator(mock)

		dt, err := gen.Generate(context.Background(), GenerateRequest{
			Name:        "search",
			Description: "Search the web",
			InputFields: map[string]string{"query": "search query"},
		}, &NoopExecutor{Response: "ok"})

		require.NoError(t, err)
		assert.Equal(t, "search", dt.Name())
		assert.Equal(t, "Search the web", dt.Description())
		assert.NotNil(t, dt.InputSchema())
		assert.NotEmpty(t, dt.Code())
	})

	t.Run("empty name", func(t *testing.T) {
		mock := &mockChatModel{responses: []string{validGenResponse()}}
		gen := NewToolGenerator(mock)

		_, err := gen.Generate(context.Background(), GenerateRequest{
			Description: "Search the web",
		}, &NoopExecutor{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("empty description", func(t *testing.T) {
		mock := &mockChatModel{responses: []string{validGenResponse()}}
		gen := NewToolGenerator(mock)

		_, err := gen.Generate(context.Background(), GenerateRequest{
			Name: "search",
		}, &NoopExecutor{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "description is required")
	})

	t.Run("retries on validation failure then succeeds", func(t *testing.T) {
		mock := &mockChatModel{
			responses: []string{
				invalidCodeGenResponse(),
				validGenResponse(),
			},
		}
		gen := NewToolGenerator(mock, WithMaxRetries(3))

		dt, err := gen.Generate(context.Background(), GenerateRequest{
			Name:        "search",
			Description: "Search",
		}, &NoopExecutor{Response: "ok"})

		require.NoError(t, err)
		assert.Equal(t, "search", dt.Name())
		assert.Equal(t, 2, mock.callCount)
	})

	t.Run("exhausts retries", func(t *testing.T) {
		mock := &mockChatModel{
			responses: []string{
				invalidCodeGenResponse(),
				invalidCodeGenResponse(),
				invalidCodeGenResponse(),
				invalidCodeGenResponse(),
			},
		}
		gen := NewToolGenerator(mock, WithMaxRetries(2))

		_, err := gen.Generate(context.Background(), GenerateRequest{
			Name:        "bad",
			Description: "Bad tool",
		}, &NoopExecutor{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed after")
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mock := &mockChatModel{responses: []string{validGenResponse()}}
		gen := NewToolGenerator(mock)

		_, err := gen.Generate(ctx, GenerateRequest{
			Name:        "search",
			Description: "Search",
		}, &NoopExecutor{})
		require.Error(t, err)
	})

	t.Run("LLM error", func(t *testing.T) {
		mock := &mockChatModel{responses: nil} // Will return error
		gen := NewToolGenerator(mock)

		_, err := gen.Generate(context.Background(), GenerateRequest{
			Name:        "search",
			Description: "Search",
		}, &NoopExecutor{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "llm error")
	})

	t.Run("with examples", func(t *testing.T) {
		mock := &mockChatModel{responses: []string{validGenResponse()}}
		gen := NewToolGenerator(mock)

		dt, err := gen.Generate(context.Background(), GenerateRequest{
			Name:        "calc",
			Description: "Calculate",
			Examples: []Example{
				{Input: `{"a": 1, "b": 2}`, Output: "3"},
			},
		}, &NoopExecutor{})

		require.NoError(t, err)
		assert.Equal(t, "calc", dt.Name())
	})

	t.Run("with allowed imports", func(t *testing.T) {
		mock := &mockChatModel{responses: []string{validGenResponse()}}
		gen := NewToolGenerator(mock, WithAllowedImports([]string{"fmt", "strings"}))

		dt, err := gen.Generate(context.Background(), GenerateRequest{
			Name:        "search",
			Description: "Search",
		}, &NoopExecutor{})
		require.NoError(t, err)
		assert.Equal(t, "search", dt.Name())
	})
}

func TestParseGeneratedTool(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid JSON",
			input: `{"code": "package main\nfunc run() {}", "input_schema": {"type": "object"}}`,
		},
		{
			name:  "markdown fenced JSON",
			input: "```json\n{\"code\": \"package main\\nfunc run() {}\", \"input_schema\": {\"type\": \"object\"}}\n```",
		},
		{
			name:    "empty code",
			input:   `{"code": "", "input_schema": {"type": "object"}}`,
			wantErr: "generated code is empty",
		},
		{
			name:    "invalid JSON",
			input:   `not json at all`,
			wantErr: "invalid JSON",
		},
		{
			name:  "nil input schema defaults",
			input: `{"code": "package main\nfunc run() {}"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := parseGeneratedTool(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, gen.Code)
			assert.NotNil(t, gen.InputSchema)
		})
	}
}
