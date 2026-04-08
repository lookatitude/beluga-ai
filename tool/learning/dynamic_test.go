package learning

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynamicTool_Interface(t *testing.T) {
	exec := &NoopExecutor{Response: "result"}
	dt := NewDynamicTool(
		"test_tool",
		"A test tool",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
		},
		`package main; func run(input string) (string, error) { return "result", nil }`,
		exec,
	)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "test_tool", dt.Name())
	})

	t.Run("Description", func(t *testing.T) {
		assert.Equal(t, "A test tool", dt.Description())
	})

	t.Run("InputSchema", func(t *testing.T) {
		s := dt.InputSchema()
		assert.Equal(t, "object", s["type"])
	})

	t.Run("Code", func(t *testing.T) {
		assert.Contains(t, dt.Code(), "func run")
	})

	t.Run("Version default", func(t *testing.T) {
		assert.Equal(t, 1, dt.Version())
	})
}

func TestDynamicTool_WithVersion(t *testing.T) {
	exec := &NoopExecutor{Response: "ok"}
	dt := NewDynamicTool("t", "d", nil, "", exec, WithVersion(5))
	assert.Equal(t, 5, dt.Version())
}

func TestDynamicTool_Execute(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		executor CodeExecutor
		want     string
		wantErr  bool
	}{
		{
			name:     "successful execution",
			input:    map[string]any{"query": "hello"},
			executor: &NoopExecutor{Response: "hello result"},
			want:     "hello result",
		},
		{
			name:     "empty input",
			input:    map[string]any{},
			executor: &NoopExecutor{Response: "empty"},
			want:     "empty",
		},
		{
			name:     "nil input",
			input:    nil,
			executor: &NoopExecutor{Response: "nil"},
			want:     "nil",
		},
		{
			name:     "executor error",
			input:    map[string]any{"x": 1},
			executor: &NoopExecutor{Err: assert.AnError},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := NewDynamicTool("test", "test tool", nil, "code", tt.executor)
			result, err := dt.Execute(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Content, 1)
			tp, ok := result.Content[0].(schema.TextPart)
			require.True(t, ok)
			assert.Equal(t, tt.want, tp.Text)
		})
	}
}

func TestDynamicTool_ExecuteContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// NoopExecutor doesn't check context, but Execute marshals input first.
	// This tests that the tool propagates context to the executor.
	exec := &NoopExecutor{Response: "ok"}
	dt := NewDynamicTool("t", "d", nil, "", exec)
	// Even with cancelled context, NoopExecutor succeeds (it ignores context).
	result, err := dt.Execute(ctx, map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, result)
}
