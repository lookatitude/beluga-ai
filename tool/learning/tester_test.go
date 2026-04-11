package learning

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolTester_Test(t *testing.T) {
	t.Run("all tests pass", func(t *testing.T) {
		dt := NewDynamicTool("greet", "greet tool", nil, "code",
			&NoopExecutor{Response: "hello"})

		tester := NewToolTester()
		results := tester.Test(context.Background(), dt, []TestCase{
			{Name: "basic", Input: map[string]any{"name": "world"}, WantOutput: "hello"},
			{Name: "no check", Input: map[string]any{}},
		})

		assert.Len(t, results, 2)
		assert.True(t, results[0].Passed)
		assert.True(t, results[1].Passed)
	})

	t.Run("output mismatch", func(t *testing.T) {
		dt := NewDynamicTool("t", "d", nil, "code",
			&NoopExecutor{Response: "actual"})

		tester := NewToolTester()
		results := tester.Test(context.Background(), dt, []TestCase{
			{Name: "mismatch", Input: map[string]any{}, WantOutput: "expected"},
		})

		require.Len(t, results, 1)
		assert.False(t, results[0].Passed)
		assert.Contains(t, results[0].Error.Error(), "output mismatch")
	})

	t.Run("expected error", func(t *testing.T) {
		dt := NewDynamicTool("t", "d", nil, "code",
			&NoopExecutor{Err: fmt.Errorf("boom")})

		tester := NewToolTester()
		results := tester.Test(context.Background(), dt, []TestCase{
			{Name: "error expected", Input: map[string]any{}, WantError: true},
		})

		require.Len(t, results, 1)
		assert.True(t, results[0].Passed)
	})

	t.Run("expected error but none", func(t *testing.T) {
		dt := NewDynamicTool("t", "d", nil, "code",
			&NoopExecutor{Response: "ok"})

		tester := NewToolTester()
		results := tester.Test(context.Background(), dt, []TestCase{
			{Name: "no error", Input: map[string]any{}, WantError: true},
		})

		require.Len(t, results, 1)
		assert.False(t, results[0].Passed)
		assert.Contains(t, results[0].Error.Error(), "expected error")
	})

	t.Run("unexpected error", func(t *testing.T) {
		dt := NewDynamicTool("t", "d", nil, "code",
			&NoopExecutor{Err: fmt.Errorf("surprise")})

		tester := NewToolTester()
		results := tester.Test(context.Background(), dt, []TestCase{
			{Name: "surprise", Input: map[string]any{}},
		})

		require.Len(t, results, 1)
		assert.False(t, results[0].Passed)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		dt := NewDynamicTool("t", "d", nil, "code",
			&NoopExecutor{Response: "ok"})

		tester := NewToolTester()
		results := tester.Test(ctx, dt, []TestCase{
			{Name: "first", Input: map[string]any{}},
			{Name: "second", Input: map[string]any{}},
		})

		// Should stop after detecting cancellation.
		require.GreaterOrEqual(t, len(results), 1)
		lastResult := results[len(results)-1]
		assert.False(t, lastResult.Passed)
	})
}

func TestToolTester_Validate(t *testing.T) {
	t.Run("all pass", func(t *testing.T) {
		dt := NewDynamicTool("t", "d", nil, "code",
			&NoopExecutor{Response: "ok"})

		tester := NewToolTester()
		err := tester.Validate(context.Background(), dt, []TestCase{
			{Name: "pass", Input: map[string]any{}, WantOutput: "ok"},
		})
		require.NoError(t, err)
	})

	t.Run("failure returns error", func(t *testing.T) {
		dt := NewDynamicTool("bad_tool", "d", nil, "code",
			&NoopExecutor{Response: "wrong"})

		tester := NewToolTester()
		err := tester.Validate(context.Background(), dt, []TestCase{
			{Name: "fail", Input: map[string]any{}, WantOutput: "right"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad_tool")
		assert.Contains(t, err.Error(), "validation failed")
	})
}

func TestToolTester_Hooks(t *testing.T) {
	var testedTools []string
	var testedResults []bool

	hooks := Hooks{
		OnToolTested: func(name string, passed bool) {
			testedTools = append(testedTools, name)
			testedResults = append(testedResults, passed)
		},
	}

	dt := NewDynamicTool("hooked_tool", "d", nil, "code",
		&NoopExecutor{Response: "ok"})

	tester := NewToolTester(WithTesterHooks(hooks))
	tester.Test(context.Background(), dt, []TestCase{
		{Name: "test1", Input: map[string]any{}, WantOutput: "ok"},
	})

	assert.Equal(t, []string{"hooked_tool"}, testedTools)
	assert.Equal(t, []bool{true}, testedResults)
}
