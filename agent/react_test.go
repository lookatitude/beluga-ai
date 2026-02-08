package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestNewReActPlanner(t *testing.T) {
	model := &testLLM{}
	p := NewReActPlanner(model)
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
	if p.llm == nil {
		t.Error("llm should not be nil")
	}
}

func TestReActPlanner_Plan_TextResponse(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("The answer is 42"), nil
		},
	}

	p := NewReActPlanner(model)
	state := PlannerState{
		Input:    "What is the meaning of life?",
		Messages: []schema.Message{schema.NewHumanMessage("What is the meaning of life?")},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != ActionFinish {
		t.Errorf("action type = %q, want %q", actions[0].Type, ActionFinish)
	}
	if actions[0].Message != "The answer is 42" {
		t.Errorf("message = %q, want %q", actions[0].Message, "The answer is 42")
	}
}

func TestReActPlanner_Plan_ToolCallResponse(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return &schema.AIMessage{
				ToolCalls: []schema.ToolCall{
					{
						ID:        "call-1",
						Name:      "search",
						Arguments: `{"q":"test"}`,
					},
					{
						ID:        "call-2",
						Name:      "calculate",
						Arguments: `{"expr":"1+1"}`,
					},
				},
			}, nil
		},
	}

	p := NewReActPlanner(model)
	state := PlannerState{
		Input:    "search and calculate",
		Messages: []schema.Message{schema.NewHumanMessage("search and calculate")},
		Tools:    []tool.Tool{&simpleTool{toolName: "search"}, &simpleTool{toolName: "calculate"}},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Type != ActionTool {
		t.Errorf("actions[0].Type = %q, want %q", actions[0].Type, ActionTool)
	}
	if actions[0].ToolCall.Name != "search" {
		t.Errorf("actions[0].ToolCall.Name = %q, want %q", actions[0].ToolCall.Name, "search")
	}
	if actions[1].ToolCall.Name != "calculate" {
		t.Errorf("actions[1].ToolCall.Name = %q, want %q", actions[1].ToolCall.Name, "calculate")
	}
}

func TestReActPlanner_Plan_LLMError(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("model unavailable")
		},
	}

	p := NewReActPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.Plan(context.Background(), state)
	if err == nil {
		t.Fatal("expected error from LLM")
	}
}

func TestReActPlanner_Replan_UsesObservations(t *testing.T) {
	var receivedMsgs []schema.Message
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			receivedMsgs = msgs
			return schema.NewAIMessage("final answer"), nil
		},
	}

	p := NewReActPlanner(model)
	state := PlannerState{
		Input: "test",
		Messages: []schema.Message{
			schema.NewHumanMessage("test"),
		},
		Observations: []Observation{
			{
				Action: Action{
					Type: ActionTool,
					ToolCall: &schema.ToolCall{
						ID:        "call-1",
						Name:      "search",
						Arguments: `{"q":"test"}`,
					},
				},
				Result: tool.TextResult("search results"),
			},
		},
	}

	actions, err := p.Replan(context.Background(), state)
	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}
	if len(actions) != 1 || actions[0].Type != ActionFinish {
		t.Errorf("expected finish action, got %v", actions)
	}

	// Verify observations were converted to messages.
	if len(receivedMsgs) < 3 {
		t.Fatalf("expected at least 3 messages (human + AI tool call + tool result), got %d", len(receivedMsgs))
	}

	// Check there's a tool message in there.
	hasToolMsg := false
	for _, msg := range receivedMsgs {
		if msg.GetRole() == schema.RoleTool {
			hasToolMsg = true
		}
	}
	if !hasToolMsg {
		t.Error("expected tool message from observations")
	}
}

func TestReActPlanner_Plan_BindsTools(t *testing.T) {
	var boundToolCount int
	model := &bindTrackingLLM{
		testLLM: testLLM{
			generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
				return schema.NewAIMessage("ok"), nil
			},
		},
		onBind: func(tools []schema.ToolDefinition) {
			boundToolCount = len(tools)
		},
	}

	p := NewReActPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		Tools:    []tool.Tool{&simpleTool{toolName: "a"}, &simpleTool{toolName: "b"}},
	}

	_, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if boundToolCount != 2 {
		t.Errorf("bound tools = %d, want 2", boundToolCount)
	}
}

func TestReActPlanner_Plan_NoTools(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("no tools needed"), nil
		},
	}

	p := NewReActPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		// No tools.
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) != 1 || actions[0].Type != ActionFinish {
		t.Errorf("expected 1 finish action, got %v", actions)
	}
}

func TestParseAIResponse_TextOnly(t *testing.T) {
	resp := schema.NewAIMessage("hello world")
	actions := parseAIResponse(resp)

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != ActionFinish {
		t.Errorf("type = %q, want %q", actions[0].Type, ActionFinish)
	}
	if actions[0].Message != "hello world" {
		t.Errorf("message = %q, want %q", actions[0].Message, "hello world")
	}
}

func TestParseAIResponse_ToolCalls(t *testing.T) {
	resp := &schema.AIMessage{
		ToolCalls: []schema.ToolCall{
			{ID: "1", Name: "search", Arguments: `{"q":"test"}`},
		},
	}
	actions := parseAIResponse(resp)

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != ActionTool {
		t.Errorf("type = %q, want %q", actions[0].Type, ActionTool)
	}
	if actions[0].ToolCall.Name != "search" {
		t.Errorf("tool name = %q, want %q", actions[0].ToolCall.Name, "search")
	}
}

func TestParseAIResponse_MultipleToolCalls(t *testing.T) {
	resp := &schema.AIMessage{
		ToolCalls: []schema.ToolCall{
			{ID: "1", Name: "a"},
			{ID: "2", Name: "b"},
			{ID: "3", Name: "c"},
		},
	}
	actions := parseAIResponse(resp)

	if len(actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(actions))
	}
	for i, a := range actions {
		if a.Type != ActionTool {
			t.Errorf("actions[%d].Type = %q, want %q", i, a.Type, ActionTool)
		}
	}
}

func TestBuildMessagesFromState_NoObservations(t *testing.T) {
	state := PlannerState{
		Messages: []schema.Message{
			schema.NewSystemMessage("You are helpful"),
			schema.NewHumanMessage("hello"),
		},
	}

	msgs := buildMessagesFromState(state)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestBuildMessagesFromState_WithObservations(t *testing.T) {
	state := PlannerState{
		Messages: []schema.Message{
			schema.NewHumanMessage("hello"),
		},
		Observations: []Observation{
			{
				Action: Action{
					Type: ActionTool,
					ToolCall: &schema.ToolCall{
						ID:        "call-1",
						Name:      "search",
						Arguments: `{}`,
					},
				},
				Result: tool.TextResult("found it"),
			},
		},
	}

	msgs := buildMessagesFromState(state)
	// 1 human + 1 AI (tool call) + 1 tool result = 3
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[1].GetRole() != schema.RoleAI {
		t.Errorf("msgs[1] role = %q, want %q", msgs[1].GetRole(), schema.RoleAI)
	}
	if msgs[2].GetRole() != schema.RoleTool {
		t.Errorf("msgs[2] role = %q, want %q", msgs[2].GetRole(), schema.RoleTool)
	}
}

func TestBuildMessagesFromState_ObservationWithError(t *testing.T) {
	state := PlannerState{
		Messages: []schema.Message{schema.NewHumanMessage("hello")},
		Observations: []Observation{
			{
				Action: Action{
					Type: ActionTool,
					ToolCall: &schema.ToolCall{
						ID:   "call-1",
						Name: "fail",
					},
				},
				Error: errors.New("tool failed"),
			},
		},
	}

	msgs := buildMessagesFromState(state)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	// Last message should be a tool message with error text.
	toolMsg := msgs[2]
	if toolMsg.GetRole() != schema.RoleTool {
		t.Errorf("last message role = %q, want %q", toolMsg.GetRole(), schema.RoleTool)
	}
}

func TestToolDefinitions(t *testing.T) {
	tools := []tool.Tool{
		&simpleTool{toolName: "a"},
		&simpleTool{toolName: "b"},
	}

	defs := toolDefinitions(tools)
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}
	if defs[0].Name != "a" {
		t.Errorf("defs[0].Name = %q, want %q", defs[0].Name, "a")
	}
	if defs[1].Name != "b" {
		t.Errorf("defs[1].Name = %q, want %q", defs[1].Name, "b")
	}
}

func TestResultText(t *testing.T) {
	tests := []struct {
		name   string
		result *tool.Result
		want   string
	}{
		{
			name:   "text result",
			result: tool.TextResult("hello"),
			want:   "hello",
		},
		{
			name:   "empty content",
			result: &tool.Result{},
			want:   "",
		},
		{
			name:   "error result",
			result: tool.ErrorResult(errors.New("oops")),
			want:   "oops",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resultText(tt.result)
			if got != tt.want {
				t.Errorf("resultText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReActPlanner_RegisteredInRegistry(t *testing.T) {
	// The react planner registers itself in init().
	names := ListPlanners()
	found := false
	for _, name := range names {
		if name == "react" {
			found = true
			break
		}
	}
	if !found {
		t.Error("react planner not found in registry")
	}
}

func TestReActPlanner_CreateFromRegistry(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("react", PlannerConfig{LLM: model})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

func TestReActPlanner_CreateFromRegistry_NoLLM(t *testing.T) {
	_, err := NewPlanner("react", PlannerConfig{})
	if err == nil {
		t.Fatal("expected error when no LLM provided")
	}
}

// bindTrackingLLM tracks BindTools calls.
type bindTrackingLLM struct {
	testLLM
	onBind func(tools []schema.ToolDefinition)
}

func (m *bindTrackingLLM) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	if m.onBind != nil {
		m.onBind(tools)
	}
	return &testLLM{generateFn: m.generateFn}
}
