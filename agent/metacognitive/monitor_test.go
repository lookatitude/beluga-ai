package metacognitive

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestMonitor_NewMonitor(t *testing.T) {
	m := NewMonitor()
	require.NotNil(t, m)
	s := m.Signals()
	assert.Empty(t, s.TaskInput)
	assert.False(t, s.Success)
}

func TestMonitor_HooksOnStart(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()

	err := hooks.OnStart(context.Background(), "test input")
	require.NoError(t, err)

	s := m.Signals()
	assert.Equal(t, "test input", s.TaskInput)
	assert.True(t, s.Success, "success should be optimistic after OnStart")
}

func TestMonitor_HooksAfterAct(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()
	ctx := context.Background()

	_ = hooks.OnStart(ctx, "input")

	// Successful observation.
	err := hooks.AfterAct(ctx, agent.Action{}, agent.Observation{})
	require.NoError(t, err)

	s := m.Signals()
	assert.Equal(t, 1, s.IterationCount)
	assert.Empty(t, s.Errors)

	// Observation with error.
	err = hooks.AfterAct(ctx, agent.Action{}, agent.Observation{
		Error: errors.New("tool failed"),
	})
	require.NoError(t, err)

	s = m.Signals()
	assert.Equal(t, 2, s.IterationCount)
	assert.Contains(t, s.Errors, "tool failed")
}

func TestMonitor_HooksOnEnd(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()
	ctx := context.Background()

	_ = hooks.OnStart(ctx, "input")
	time.Sleep(5 * time.Millisecond) // Ensure measurable latency.
	hooks.OnEnd(ctx, "result text", nil)

	s := m.Signals()
	assert.Equal(t, "result text", s.Outcome)
	assert.True(t, s.Success)
	assert.Greater(t, s.TotalLatency, time.Duration(0))
}

func TestMonitor_HooksOnEndWithError(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()
	ctx := context.Background()

	_ = hooks.OnStart(ctx, "input")
	hooks.OnEnd(ctx, "", errors.New("execution failed"))

	s := m.Signals()
	assert.False(t, s.Success)
	assert.Contains(t, s.Errors, "execution failed")
}

func TestMonitor_HooksOnError(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()
	ctx := context.Background()

	_ = hooks.OnStart(ctx, "input")
	origErr := errors.New("something broke")
	retErr := hooks.OnError(ctx, origErr)

	assert.Equal(t, origErr, retErr, "OnError must pass through the error")
	s := m.Signals()
	assert.False(t, s.Success)
	assert.Contains(t, s.Errors, "something broke")
}

func TestMonitor_HooksOnToolCall(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()
	ctx := context.Background()

	_ = hooks.OnStart(ctx, "input")
	err := hooks.OnToolCall(ctx, agent.ToolCallInfo{Name: "search", CallID: "c1"})
	require.NoError(t, err)
	err = hooks.OnToolCall(ctx, agent.ToolCallInfo{Name: "calculate", CallID: "c2"})
	require.NoError(t, err)

	s := m.Signals()
	assert.Equal(t, []string{"search", "calculate"}, s.ToolCalls)
}

func TestMonitor_HooksOnToolResult(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()
	ctx := context.Background()

	_ = hooks.OnStart(ctx, "input")

	// Successful tool result.
	err := hooks.OnToolResult(ctx, agent.ToolCallInfo{Name: "search"}, tool.TextResult("ok"))
	require.NoError(t, err)
	s := m.Signals()
	assert.Empty(t, s.Errors)

	// Failed tool result.
	err = hooks.OnToolResult(ctx, agent.ToolCallInfo{Name: "fetch"}, &tool.Result{IsError: true})
	require.NoError(t, err)
	s = m.Signals()
	assert.Contains(t, s.Errors, "tool fetch failed")
}

func TestMonitor_Reset(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()
	ctx := context.Background()

	_ = hooks.OnStart(ctx, "input")
	_ = hooks.OnToolCall(ctx, agent.ToolCallInfo{Name: "search"})
	hooks.OnEnd(ctx, "result", nil)

	s := m.Signals()
	assert.NotEmpty(t, s.TaskInput)

	m.Reset()
	s = m.Signals()
	assert.Empty(t, s.TaskInput)
	assert.Empty(t, s.ToolCalls)
	assert.Empty(t, s.Outcome)
}

func TestMonitor_SignalsReturnsCopy(t *testing.T) {
	m := NewMonitor()
	hooks := m.Hooks()
	ctx := context.Background()

	_ = hooks.OnStart(ctx, "input")
	_ = hooks.OnToolCall(ctx, agent.ToolCallInfo{Name: "search"})

	s1 := m.Signals()
	s1.ToolCalls = append(s1.ToolCalls, "mutated")

	s2 := m.Signals()
	assert.Len(t, s2.ToolCalls, 1, "mutation of returned signals must not affect monitor")
}
