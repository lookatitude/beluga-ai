package agent

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// describeMessages renders a message list as e.g. "human ai:c1 tool:c1".
func describeMessages(msgs []schema.Message) string {
	parts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		switch v := m.(type) {
		case *schema.HumanMessage:
			parts = append(parts, "human")
		case *schema.SystemMessage:
			parts = append(parts, "system")
		case *schema.AIMessage:
			if len(v.ToolCalls) > 0 {
				parts = append(parts, "ai:"+v.ToolCalls[0].ID)
			} else {
				parts = append(parts, "ai")
			}
		case *schema.ToolMessage:
			parts = append(parts, "tool:"+v.ToolCallID)
		default:
			parts = append(parts, fmt.Sprintf("%T", m))
		}
	}
	return strings.Join(parts, " ")
}

// TestReAct_ToolHistoryNotDuplicated is a regression test for FWC-01: the
// Executor folded observations into state.Messages while every planner also
// re-appended state.Observations, so each tool call was sent twice and the
// duplication grew every iteration.
func TestReAct_ToolHistoryNotDuplicated(t *testing.T) {
	var calls []string
	model := &testLLM{generateFn: func(_ context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
		calls = append(calls, describeMessages(msgs))
		switch len(calls) {
		case 1:
			return &schema.AIMessage{ToolCalls: []schema.ToolCall{{ID: "c1", Name: "echo", Arguments: `{}`}}}, nil
		case 2:
			return &schema.AIMessage{ToolCalls: []schema.ToolCall{{ID: "c2", Name: "echo", Arguments: `{}`}}}, nil
		default:
			return schema.NewAIMessage("done"), nil
		}
	}}
	echo := &funcTool{name: "echo", fn: func(context.Context, map[string]any) (*tool.Result, error) {
		return tool.TextResult("ok"), nil
	}}

	a := New("a", WithLLM(model), WithTools([]tool.Tool{echo}))
	if _, err := a.Invoke(context.Background(), "hi"); err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	want := []string{
		"human",
		"human ai:c1 tool:c1",
		"human ai:c1 tool:c1 ai:c2 tool:c2",
	}
	if len(calls) != len(want) {
		t.Fatalf("LLM calls = %d (%q), want %d", len(calls), calls, len(want))
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Errorf("LLM call %d messages = %q, want %q", i+1, calls[i], want[i])
		}
	}
}

// TestExecutor_MessagesNotMutatedByLoop checks the planner-visible
// state.Messages stays the starting conversation across iterations.
func TestExecutor_MessagesNotMutatedByLoop(t *testing.T) {
	var lens []int
	p := &funcPlanner{planFn: func(_ context.Context, state PlannerState) ([]Action, error) {
		lens = append(lens, len(state.Messages))
		if len(state.Observations) >= 3 {
			return []Action{{Type: ActionFinish, Message: "done"}}, nil
		}
		id := fmt.Sprintf("c%d", len(state.Observations)+1)
		return []Action{{Type: ActionTool, ToolCall: &schema.ToolCall{ID: id, Name: "echo", Arguments: `{}`}}}, nil
	}}
	echo := &funcTool{name: "echo", fn: func(context.Context, map[string]any) (*tool.Result, error) {
		return tool.TextResult("ok"), nil
	}}
	e := NewExecutor(WithExecutorPlanner(p))
	for _, err := range e.Run(context.Background(), "hi", "a", []tool.Tool{echo}, []schema.Message{schema.NewHumanMessage("hi")}, Hooks{}) {
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	}
	want := "[1 1 1 1]"
	if got := fmt.Sprint(lens); got != want {
		t.Errorf("len(state.Messages) per iteration = %s, want %s", got, want)
	}
}

// TestBaseAgent_StreamEarlyBreakDoesNotPanic is a regression test for FWC-02:
// the Executor ignored yield's return value, so breaking out of Stream made
// Go panic ("range function continued iteration") and the tool still ran.
func TestBaseAgent_StreamEarlyBreakDoesNotPanic(t *testing.T) {
	var toolRuns atomic.Int32
	model := &testLLM{generateFn: func(_ context.Context, _ []schema.Message) (*schema.AIMessage, error) {
		return &schema.AIMessage{ToolCalls: []schema.ToolCall{{ID: "c1", Name: "echo", Arguments: `{}`}}}, nil
	}}
	echo := &funcTool{name: "echo", fn: func(context.Context, map[string]any) (*tool.Result, error) {
		toolRuns.Add(1)
		return tool.TextResult("ok"), nil
	}}
	a := New("a", WithLLM(model), WithTools([]tool.Tool{echo}))

	breakOn := []EventType{EventToolCall, EventToolResult}
	for _, stopAt := range breakOn {
		t.Run(string(stopAt), func(t *testing.T) {
			toolRuns.Store(0)
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on early break: %v", r)
				}
			}()
			for ev, err := range a.Stream(context.Background(), "hi") {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if ev.Type == stopAt {
					break
				}
			}
			wantRuns := int32(1)
			if stopAt == EventToolCall {
				wantRuns = 0
			}
			if got := toolRuns.Load(); got != wantRuns {
				t.Errorf("tool runs after break on %s = %d, want %d", stopAt, got, wantRuns)
			}
		})
	}
}

// TestExecutor_EarlyBreakOnText checks breaking on a text/handoff event
// stops the loop without yielding EventDone.
func TestExecutor_EarlyBreakOnText(t *testing.T) {
	for _, act := range []Action{
		{Type: ActionFinish, Message: "bye"},
		{Type: ActionHandoff, Message: "go", Metadata: map[string]any{"target": "b"}},
	} {
		p := &funcPlanner{planFn: func(context.Context, PlannerState) ([]Action, error) {
			return []Action{act}, nil
		}}
		e := NewExecutor(WithExecutorPlanner(p))
		n := 0
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			for range e.Run(context.Background(), "hi", "a", nil, nil, Hooks{}) {
				n++
				break
			}
		}()
		if n != 1 {
			t.Errorf("events = %d, want 1", n)
		}
	}
}

// TestExecutor_CancelledContextIsNotRetryable is a regression test for
// FWC-03: caller cancellation was reported as the retryable ErrTimeout.
func TestExecutor_CancelledContextIsNotRetryable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &funcPlanner{planFn: func(context.Context, PlannerState) ([]Action, error) {
		return []Action{{Type: ActionFinish, Message: "x"}}, nil
	}}
	e := NewExecutor(WithExecutorPlanner(p))
	var gotErr error
	for _, err := range e.Run(ctx, "hi", "a", nil, nil, Hooks{}) {
		if err != nil {
			gotErr = err
		}
	}
	if gotErr == nil {
		t.Fatal("expected error")
	}
	if isRetryableErr(gotErr) {
		t.Errorf("cancellation error is retryable: %v", gotErr)
	}
}

func isRetryableErr(err error) bool { return core.IsRetryable(err) }
