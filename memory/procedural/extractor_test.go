package procedural

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLM implements llm.ChatModel for testing the extractor.
type mockLLM struct {
	response string
	err      error
}

var _ llm.ChatModel = (*mockLLM)(nil)

func (m *mockLLM) Generate(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return schema.NewAIMessage(m.response), nil
}

func (m *mockLLM) Stream(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {}
}

func (m *mockLLM) BindTools(_ []schema.ToolDefinition) llm.ChatModel {
	return m
}

func (m *mockLLM) ModelID() string { return "mock-llm" }

func TestLLMExtractor_Extract(t *testing.T) {
	ctx := context.Background()

	t.Run("extracts skill from trace", func(t *testing.T) {
		model := &mockLLM{
			response: `{"name":"deploy-service","description":"Deploy a microservice","steps":["build image","push to registry","apply manifests"],"triggers":["deploy","release"],"tags":["devops"],"confidence":0.85}`,
		}
		ext, err := NewLLMExtractor(model)
		require.NoError(t, err)

		skill, err := ext.Extract(ctx, "deploy the user service", "deployed successfully", nil)
		require.NoError(t, err)
		require.NotNil(t, skill)
		assert.Equal(t, "deploy-service", skill.Name)
		assert.Equal(t, "Deploy a microservice", skill.Description)
		assert.Len(t, skill.Steps, 3)
		assert.Equal(t, []string{"deploy", "release"}, skill.Triggers)
		assert.Equal(t, []string{"devops"}, skill.Tags)
		assert.InDelta(t, 0.85, skill.Confidence, 0.001)
	})

	t.Run("returns nil for empty response", func(t *testing.T) {
		model := &mockLLM{response: ""}
		ext, err := NewLLMExtractor(model)
		require.NoError(t, err)

		skill, err := ext.Extract(ctx, "input", "output", nil)
		require.NoError(t, err)
		assert.Nil(t, skill)
	})

	t.Run("returns nil for empty skill name", func(t *testing.T) {
		model := &mockLLM{response: `{"name":"","description":"something"}`}
		ext, err := NewLLMExtractor(model)
		require.NoError(t, err)

		skill, err := ext.Extract(ctx, "input", "output", nil)
		require.NoError(t, err)
		assert.Nil(t, skill)
	})

	t.Run("returns error on LLM failure", func(t *testing.T) {
		model := &mockLLM{err: errors.New("api error")}
		ext, err := NewLLMExtractor(model)
		require.NoError(t, err)

		skill, err := ext.Extract(ctx, "input", "output", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "generate")
		assert.Nil(t, skill)
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		model := &mockLLM{response: "not valid json"}
		ext, err := NewLLMExtractor(model)
		require.NoError(t, err)

		skill, err := ext.Extract(ctx, "input", "output", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse skill JSON")
		assert.Nil(t, skill)
	})

	t.Run("passes metadata in prompt", func(t *testing.T) {
		model := &mockLLM{
			response: `{"name":"test","description":"test","steps":["step1"],"triggers":["trigger"],"confidence":0.9}`,
		}
		ext, err := NewLLMExtractor(model)
		require.NoError(t, err)

		meta := map[string]any{"task_type": "deployment", "success": true}
		skill, err := ext.Extract(ctx, "input", "output", meta)
		require.NoError(t, err)
		require.NotNil(t, skill)
		assert.Equal(t, "test", skill.Name)
	})

	t.Run("returns empty object as nil", func(t *testing.T) {
		model := &mockLLM{response: `{}`}
		ext, err := NewLLMExtractor(model)
		require.NoError(t, err)

		skill, err := ext.Extract(ctx, "input", "output", nil)
		require.NoError(t, err)
		assert.Nil(t, skill) // name is empty
	})
}

func TestNewLLMExtractor_NilModel(t *testing.T) {
	ext, err := NewLLMExtractor(nil)
	require.Error(t, err)
	assert.Nil(t, ext)
	assert.Contains(t, err.Error(), "must not be nil")
}

func TestBuildExtractionPrompt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		output   string
		metadata map[string]any
		contains []string
	}{
		{
			name:     "basic",
			input:    "deploy service",
			output:   "done",
			contains: []string{"Input:", "deploy service", "Output:", "done"},
		},
		{
			name:     "with metadata",
			input:    "input",
			output:   "output",
			metadata: map[string]any{"key": "val"},
			contains: []string{"Metadata:", "key", "val"},
		},
		{
			name:     "nil metadata",
			input:    "input",
			output:   "output",
			metadata: nil,
			contains: []string{"Input:", "Output:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildExtractionPrompt(tt.input, tt.output, tt.metadata)
			for _, s := range tt.contains {
				assert.Contains(t, prompt, s)
			}
		})
	}
}
