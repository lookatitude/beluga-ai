package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestNewExecutor_Defaults(t *testing.T) {
	e := NewExecutor()
	if e.maxIterations != 10 {
		t.Errorf("maxIterations = %d, want 10", e.maxIterations)
	}
	if e.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v", e.timeout, 5*time.Minute)
	}
	if e.planner != nil {
		t.Error("planner should be nil by default")
	}
}

func TestNewExecutor_WithOptions(t *testing.T) {
	p := &mockPlanner{name: "test"}
	e := NewExecutor(
		WithExecutorPlanner(p),
		WithExecutorMaxIterations(3),
		WithExecutorTimeout(30*time.Second),
	)
	if e.maxIterations != 3 {
		t.Errorf("maxIterations = %d, want 3", e.maxIterations)
	}
	if e.timeout != 30*time.Second {
		t.Errorf("timeout = %v, want %v", e.timeout, 30*time.Second)
	}
	if e.planner == nil {
		t.Fatal("planner should not be nil")
	}
}

func TestWithExecutorMaxIterations_IgnoresNonPositive(t *testing.T) {
	e := NewExecutor(WithExecutorMaxIterations(0))
	if e.maxIterations != 10 {
		t.Errorf("maxIterations = %d, want 10 (unchanged)", e.maxIterations)
	}

	e = NewExecutor(WithExecutorMaxIterations(-5))
	if e.maxIterations != 10 {
		t.Errorf("maxIterations = %d, want 10 (unchanged)", e.maxIterations)
	}
}

func TestWithExecutorTimeout_IgnoresNonPositive(t *testing.T) {
	e := NewExecutor(WithExecutorTimeout(0))
	if e.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v (unchanged)", e.timeout, 5*time.Minute)
	}

	e = NewExecutor(WithExecutorTimeout(-1))
	if e.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v (unchanged)", e.timeout, 5*time.Minute)
	}
}

func TestExecutor_Run_FinishAction(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			return []Action{{Type: ActionFinish, Message: "done!"}}, nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var events []Event
	for event, err := range e.Run(context.Background(), "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, event)
	}

	hasDone := false
	for _, event := range events {
		if event.Type == EventDone {
			hasDone = true
			if event.Text != "done!" {
				t.Errorf("done text = %q, want %q", event.Text, "done!")
			}
		}
	}
	if !hasDone {
		t.Error("expected EventDone event")
	}
}

func TestExecutor_Run_RespondAction(t *testing.T) {
	calls := 0
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			calls++
			if calls == 1 {
				return []Action{{Type: ActionRespond, Message: "thinking..."}}, nil
			}
			return []Action{{Type: ActionFinish, Message: "done"}}, nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var texts []string
	for event, err := range e.Run(context.Background(), "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == EventText {
			texts = append(texts, event.Text)
		}
	}

	if len(texts) < 1 {
		t.Fatal("expected at least 1 text event")
	}
	if texts[0] != "thinking..." {
		t.Errorf("first text = %q, want %q", texts[0], "thinking...")
	}
}

func TestExecutor_Run_ToolAction(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			if len(state.Observations) > 0 {
				return []Action{{Type: ActionFinish, Message: "done"}}, nil
			}
			return []Action{{
				Type: ActionTool,
				ToolCall: &schema.ToolCall{
					ID:        "call-1",
					Name:      "echo",
					Arguments: `{"text":"hello"}`,
				},
			}}, nil
		},
	}

	echoTool := &funcTool{
		name: "echo",
		fn: func(ctx context.Context, args map[string]any) (*tool.Result, error) {
			return tool.TextResult("echoed: " + args["text"].(string)), nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var toolCallEvents int
	var toolResultEvents int
	for event, err := range e.Run(context.Background(), "test", "agent-1", []tool.Tool{echoTool}, nil, Hooks{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == EventToolCall {
			toolCallEvents++
		}
		if event.Type == EventToolResult {
			toolResultEvents++
		}
	}

	if toolCallEvents != 1 {
		t.Errorf("tool call events = %d, want 1", toolCallEvents)
	}
	if toolResultEvents != 1 {
		t.Errorf("tool result events = %d, want 1", toolResultEvents)
	}
}

func TestExecutor_Run_ToolNotFound(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			if len(state.Observations) > 0 {
				return []Action{{Type: ActionFinish, Message: "done"}}, nil
			}
			return []Action{{
				Type: ActionTool,
				ToolCall: &schema.ToolCall{
					ID:        "call-1",
					Name:      "nonexistent",
					Arguments: `{}`,
				},
			}}, nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var toolResultEvents int
	for event, err := range e.Run(context.Background(), "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == EventToolResult {
			toolResultEvents++
			if event.ToolResult == nil {
				t.Error("expected non-nil tool result for not-found tool")
			}
		}
	}
	if toolResultEvents != 1 {
		t.Errorf("expected 1 tool result event, got %d", toolResultEvents)
	}
}

func TestExecutor_Run_ToolMissingToolCall(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			if len(state.Observations) > 0 {
				return []Action{{Type: ActionFinish, Message: "done"}}, nil
			}
			return []Action{{
				Type:     ActionTool,
				ToolCall: nil, // missing
			}}, nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	for _, err := range e.Run(context.Background(), "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			// The observation will have an error but execution continues.
			break
		}
	}
	// Just verifying it doesn't panic.
}

func TestExecutor_Run_InvalidToolArguments(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			if len(state.Observations) > 0 {
				return []Action{{Type: ActionFinish, Message: "done"}}, nil
			}
			return []Action{{
				Type: ActionTool,
				ToolCall: &schema.ToolCall{
					ID:        "call-1",
					Name:      "echo",
					Arguments: "not valid json",
				},
			}}, nil
		},
	}

	echoTool := &funcTool{
		name: "echo",
		fn: func(ctx context.Context, args map[string]any) (*tool.Result, error) {
			return tool.TextResult("ok"), nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var gotToolResult bool
	for event, err := range e.Run(context.Background(), "test", "agent-1", []tool.Tool{echoTool}, nil, Hooks{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == EventToolResult && event.ToolResult != nil && event.ToolResult.IsError {
			gotToolResult = true
		}
	}
	if !gotToolResult {
		t.Error("expected error tool result for invalid arguments")
	}
}

func TestExecutor_Run_MaxIterations(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			// Never finish — always respond.
			return []Action{{Type: ActionRespond, Message: "still going"}}, nil
		},
	}

	e := NewExecutor(
		WithExecutorPlanner(p),
		WithExecutorMaxIterations(3),
	)

	var gotErr error
	for _, err := range e.Run(context.Background(), "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Fatal("expected max iterations error")
	}
}

func TestExecutor_Run_PlanError(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			return nil, errors.New("plan failed")
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var gotErr error
	for _, err := range e.Run(context.Background(), "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Fatal("expected plan error")
	}
}

func TestExecutor_Run_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			return []Action{{Type: ActionFinish, Message: "done"}}, nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var gotErr error
	for _, err := range e.Run(ctx, "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Fatal("expected cancelled context error")
	}
}

func TestExecutor_Run_Hooks_OnStartOnEnd(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			return []Action{{Type: ActionFinish, Message: "done"}}, nil
		},
	}

	var startInput string
	var endResult string

	hooks := Hooks{
		OnStart: func(ctx context.Context, input string) error {
			startInput = input
			return nil
		},
		OnEnd: func(ctx context.Context, result string, err error) {
			endResult = result
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))
	for range e.Run(context.Background(), "hello", "agent-1", nil, nil, hooks) {
	}

	if startInput != "hello" {
		t.Errorf("OnStart input = %q, want %q", startInput, "hello")
	}
	if endResult != "done" {
		t.Errorf("OnEnd result = %q, want %q", endResult, "done")
	}
}

func TestExecutor_Run_Hooks_OnStartError(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			return []Action{{Type: ActionFinish, Message: "done"}}, nil
		},
	}

	hooks := Hooks{
		OnStart: func(ctx context.Context, input string) error {
			return errors.New("start blocked")
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var gotErr error
	for _, err := range e.Run(context.Background(), "test", "agent-1", nil, nil, hooks) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil || gotErr.Error() != "start blocked" {
		t.Errorf("expected 'start blocked' error, got: %v", gotErr)
	}
}

func TestExecutor_Run_HandoffAction(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			return []Action{{
				Type:    ActionHandoff,
				Message: "handing off",
				Metadata: map[string]any{
					"target": "other-agent",
				},
			}}, nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var handoffEvent *Event
	for event, err := range e.Run(context.Background(), "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == EventHandoff {
			handoffEvent = &event
		}
	}

	if handoffEvent == nil {
		t.Fatal("expected handoff event")
	}
	if handoffEvent.Metadata["target"] != "other-agent" {
		t.Errorf("handoff target = %v, want %q", handoffEvent.Metadata["target"], "other-agent")
	}
}

func TestExecutor_Run_Hooks_BeforeAfterPlan(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			return []Action{{Type: ActionFinish, Message: "done"}}, nil
		},
	}

	var order []string
	hooks := Hooks{
		BeforePlan: func(ctx context.Context, state PlannerState) error {
			order = append(order, "before_plan")
			return nil
		},
		AfterPlan: func(ctx context.Context, actions []Action) error {
			order = append(order, "after_plan")
			return nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))
	for range e.Run(context.Background(), "test", "agent-1", nil, nil, hooks) {
	}

	if len(order) != 2 || order[0] != "before_plan" || order[1] != "after_plan" {
		t.Errorf("hook order = %v, want [before_plan, after_plan]", order)
	}
}

func TestExecutor_Run_Hooks_OnToolCallAndResult(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			if len(state.Observations) > 0 {
				return []Action{{Type: ActionFinish, Message: "done"}}, nil
			}
			return []Action{{
				Type: ActionTool,
				ToolCall: &schema.ToolCall{
					ID:        "call-1",
					Name:      "test-tool",
					Arguments: `{}`,
				},
			}}, nil
		},
	}

	testTool := &funcTool{
		name: "test-tool",
		fn: func(ctx context.Context, args map[string]any) (*tool.Result, error) {
			return tool.TextResult("result"), nil
		},
	}

	var toolCallName string
	var toolResultReceived bool
	hooks := Hooks{
		OnToolCall: func(ctx context.Context, call ToolCallInfo) error {
			toolCallName = call.Name
			return nil
		},
		OnToolResult: func(ctx context.Context, call ToolCallInfo, result *tool.Result) error {
			toolResultReceived = true
			return nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))
	for range e.Run(context.Background(), "test", "agent-1", []tool.Tool{testTool}, nil, hooks) {
	}

	if toolCallName != "test-tool" {
		t.Errorf("OnToolCall name = %q, want %q", toolCallName, "test-tool")
	}
	if !toolResultReceived {
		t.Error("OnToolResult was not called")
	}
}

func TestExtractResultText(t *testing.T) {
	tests := []struct {
		name   string
		result *tool.Result
		want   string
	}{
		{
			name:   "single text",
			result: tool.TextResult("hello"),
			want:   "hello",
		},
		{
			name: "multiple text parts",
			result: &tool.Result{
				Content: []schema.ContentPart{
					schema.TextPart{Text: "line1"},
					schema.TextPart{Text: "line2"},
				},
			},
			want: "line1\nline2",
		},
		{
			name:   "empty content",
			result: &tool.Result{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResultText(tt.result)
			if got != tt.want {
				t.Errorf("extractResultText() = %q, want %q", got, tt.want)
			}
		})
	}
}

// funcPlanner is a test helper planner with configurable behavior.
type funcPlanner struct {
	planFn func(ctx context.Context, state PlannerState) ([]Action, error)
}

func (p *funcPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.planFn(ctx, state)
}

func (p *funcPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.planFn(ctx, state)
}

// funcTool is a simple configurable tool for tests.
type funcTool struct {
	name string
	fn   func(ctx context.Context, args map[string]any) (*tool.Result, error)
}

func (t *funcTool) Name() string                { return t.name }
func (t *funcTool) Description() string         { return "test tool" }
func (t *funcTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *funcTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	return t.fn(ctx, input)
}

var _ tool.Tool = (*funcTool)(nil)

// TestExecutor_Run_ToolExecuteError tests tool execution error path.
func TestExecutor_Run_ToolExecuteError(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			if len(state.Observations) > 0 {
				return []Action{{Type: ActionFinish, Message: "done"}}, nil
			}
			return []Action{{
				Type: ActionTool,
				ToolCall: &schema.ToolCall{
					ID:        "call-1",
					Name:      "failing-tool",
					Arguments: `{}`,
				},
			}}, nil
		},
	}

	failingTool := &funcTool{
		name: "failing-tool",
		fn: func(ctx context.Context, args map[string]any) (*tool.Result, error) {
			return nil, errors.New("tool execution failed")
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	var gotToolResult bool
	for event, err := range e.Run(context.Background(), "test", "agent-1", []tool.Tool{failingTool}, nil, Hooks{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == EventToolResult && event.ToolResult != nil && event.ToolResult.IsError {
			gotToolResult = true
		}
	}

	if !gotToolResult {
		t.Error("expected error tool result")
	}
}

// TestExecutor_buildMessages_NonToolObservation tests buildMessages skipping non-tool observations.
func TestExecutor_buildMessages_NonToolObservation(t *testing.T) {
	e := NewExecutor()
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("initial")},
		Observations: []Observation{
			{
				Action: Action{Type: ActionRespond, Message: "response"},
			},
		},
	}

	msgs := e.buildMessages(state)
	// Should only have the initial message (non-tool observations are skipped)
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}

// TestExecutor_Run_Timeout_ViaContext tests timeout via context.
func TestExecutor_Run_Timeout_ViaContext(t *testing.T) {
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			// Check context to see if it's cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return []Action{{Type: ActionFinish, Message: "done"}}, nil
			}
		},
	}

	e := NewExecutor(WithExecutorPlanner(p), WithExecutorTimeout(0)) // No timeout set

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	var gotErr error
	for _, err := range e.Run(ctx, "test", "agent-1", nil, nil, Hooks{}) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Fatal("expected context timeout error")
	}
}

// TestExecutor_Run_ReplanAfterFirstIteration tests that Replan is called after first iteration.
func TestExecutor_Run_ReplanAfterFirstIteration(t *testing.T) {
	planCalls := 0
	replanCalls := 0

	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			if state.Iteration == 0 {
				planCalls++
				return []Action{{Type: ActionRespond, Message: "plan"}}, nil
			}
			replanCalls++
			return []Action{{Type: ActionFinish, Message: "replan"}}, nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	for range e.Run(context.Background(), "test", "agent-1", nil, nil, Hooks{}) {
	}

	if planCalls != 1 {
		t.Errorf("plan calls = %d, want 1", planCalls)
	}
	if replanCalls != 1 {
		t.Errorf("replan calls = %d, want 1", replanCalls)
	}
}

// TestExecutor_Run_buildMessages_MultipleObservations tests buildMessages with multiple observations.
func TestExecutor_Run_buildMessages_MultipleObservations(t *testing.T) {
	observationCount := 0
	p := &funcPlanner{
		planFn: func(ctx context.Context, state PlannerState) ([]Action, error) {
			observationCount = len(state.Observations)
			if len(state.Observations) >= 2 {
				return []Action{{Type: ActionFinish, Message: "done"}}, nil
			}
			return []Action{{
				Type: ActionTool,
				ToolCall: &schema.ToolCall{
					ID:        fmt.Sprintf("call-%d", len(state.Observations)+1),
					Name:      "test",
					Arguments: `{}`,
				},
			}}, nil
		},
	}

	testTool := &funcTool{
		name: "test",
		fn: func(ctx context.Context, args map[string]any) (*tool.Result, error) {
			return tool.TextResult("ok"), nil
		},
	}

	e := NewExecutor(WithExecutorPlanner(p))

	for range e.Run(context.Background(), "test", "agent-1", []tool.Tool{testTool}, nil, Hooks{}) {
	}

	if observationCount < 2 {
		t.Errorf("expected at least 2 observations to accumulate, got %d", observationCount)
	}
}
