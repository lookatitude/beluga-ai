package codeact

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// stubChatModel is a minimal llm.ChatModel used to exercise the CodeActPlanner
// end-to-end without pulling a real provider.
type stubChatModel struct {
	resp *schema.AIMessage
	err  error
}

func (m *stubChatModel) Generate(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func (m *stubChatModel) Stream(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(_ func(schema.StreamChunk, error) bool) {}
}

func (m *stubChatModel) BindTools(_ []schema.ToolDefinition) llm.ChatModel { return m }
func (m *stubChatModel) ModelID() string                                   { return "stub" }

var _ llm.ChatModel = (*stubChatModel)(nil)

// --- Agent accessor coverage ---

func TestCodeActAgent_Accessors(t *testing.T) {
	a := NewCodeActAgent("acc",
		WithLanguage("python"),
		WithExecutor(NewNoopExecutor()),
	)

	if a.ID() != "acc" {
		t.Errorf("ID = %q, want acc", a.ID())
	}
	// Persona/Tools/Children should be safe passthroughs on a freshly built agent.
	_ = a.Persona()
	if tools := a.Tools(); tools == nil && len(tools) != 0 {
		t.Errorf("Tools() returned nil-with-length, unexpected")
	}
	_ = a.Children()
}

// WithAgentLLM / WithAllowedCodeImports / WithAgentOption coverage.
func TestCodeActAgent_OptionsCoverage(t *testing.T) {
	llmStub := &stubChatModel{
		resp: &schema.AIMessage{
			Parts: []schema.ContentPart{schema.TextPart{Text: "final answer"}},
		},
	}

	a := NewCodeActAgent("opts",
		WithAgentLLM(llmStub),
		WithAllowedCodeImports([]string{"math"}),
		WithAgentOption(agent.WithMaxIterations(2)),
		WithExecutor(NewNoopExecutor()),
	)
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
	if a.llm != llmStub {
		t.Error("WithAgentLLM did not install the stub model")
	}
}

// --- Invoke coverage ---

func TestCodeActAgent_Invoke(t *testing.T) {
	planner := &stubPlanner{
		iterations: [][]agent.Action{
			{{Type: agent.ActionFinish, Message: "hello"}},
		},
	}
	a := NewCodeActAgent("inv",
		WithPlanner(planner),
		WithExecutor(NewNoopExecutor()),
		WithMaxIterations(3),
	)

	out, err := a.Invoke(context.Background(), "ping")
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if out == "" {
		t.Error("Invoke returned empty string")
	}
}

func TestCodeActAgent_Invoke_PlannerError(t *testing.T) {
	errBoom := errors.New("boom")
	planner := &errPlanner{err: errBoom}

	a := NewCodeActAgent("inv-err",
		WithPlanner(planner),
		WithExecutor(NewNoopExecutor()),
	)

	_, err := a.Invoke(context.Background(), "ping")
	if err == nil {
		t.Fatal("expected error from planner")
	}
}

type errPlanner struct {
	err error
}

func (p *errPlanner) Plan(_ context.Context, _ agent.PlannerState) ([]agent.Action, error) {
	return nil, p.err
}
func (p *errPlanner) Replan(_ context.Context, _ agent.PlannerState) ([]agent.Action, error) {
	return nil, p.err
}

// --- Stream: no planner fallback path ---

func TestCodeActAgent_Stream_NoPlannerFallback(t *testing.T) {
	// No planner configured, no llm -> falls back to base agent's Stream.
	a := NewCodeActAgent("nop",
		WithExecutor(NewNoopExecutor()),
	)
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so the base agent's stream terminates cleanly.
	cancel()
	count := 0
	for _, err := range a.Stream(ctx, "hi") {
		count++
		_ = err
		if count > 10 {
			break
		}
	}
}

// --- Stream: context-cancelled inside loop ---

func TestCodeActAgent_Stream_ContextCanceled(t *testing.T) {
	planner := &stubPlanner{
		iterations: [][]agent.Action{
			{{Type: agent.ActionFinish, Message: "done"}},
		},
	}
	a := NewCodeActAgent("cancel",
		WithPlanner(planner),
		WithExecutor(NewNoopExecutor()),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sawError := false
	for event, err := range a.Stream(ctx, "q") {
		if err != nil {
			sawError = true
			break
		}
		_ = event
	}
	if !sawError {
		t.Error("expected context cancellation to surface as error event")
	}
}

// --- Stream: unsupported action ---

func TestCodeActAgent_Stream_UnsupportedAction(t *testing.T) {
	planner := &stubPlanner{
		iterations: [][]agent.Action{
			{{Type: agent.ActionType("custom"), Message: "x"}},
		},
	}
	a := NewCodeActAgent("bad",
		WithPlanner(planner),
		WithExecutor(NewNoopExecutor()),
	)
	sawError := false
	for _, err := range a.Stream(context.Background(), "q") {
		if err != nil {
			sawError = true
			break
		}
	}
	if !sawError {
		t.Error("expected error for unsupported action type")
	}
}

// --- Stream: max iterations exhausted ---

func TestCodeActAgent_Stream_MaxIterationsExceeded(t *testing.T) {
	// Planner that never finishes: always issues a code action so the loop
	// can iterate.
	planner := &loopPlanner{}
	a := NewCodeActAgent("loop",
		WithPlanner(planner),
		WithExecutor(NewNoopExecutor()),
		WithMaxIterations(2),
	)
	sawError := false
	for _, err := range a.Stream(context.Background(), "q") {
		if err != nil {
			sawError = true
		}
	}
	if !sawError {
		t.Error("expected max-iterations error")
	}
}

type loopPlanner struct{}

func (loopPlanner) Plan(_ context.Context, _ agent.PlannerState) ([]agent.Action, error) {
	return []agent.Action{{
		Type:     ActionCode,
		Metadata: map[string]any{"code": "x", "language": "python"},
	}}, nil
}
func (l loopPlanner) Replan(ctx context.Context, s agent.PlannerState) ([]agent.Action, error) {
	return l.Plan(ctx, s)
}

// --- Stream: Respond action (no Finish) ---

func TestCodeActAgent_Stream_Respond(t *testing.T) {
	planner := &stubPlanner{
		iterations: [][]agent.Action{
			{{Type: agent.ActionRespond, Message: "hello-respond"}},
			{{Type: agent.ActionFinish, Message: "bye"}},
		},
	}
	a := NewCodeActAgent("resp",
		WithPlanner(planner),
		WithExecutor(NewNoopExecutor()),
		WithMaxIterations(3),
	)
	var sawRespondText bool
	for event, err := range a.Stream(context.Background(), "q") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == agent.EventText && event.Text == "hello-respond" {
			sawRespondText = true
		}
	}
	if !sawRespondText {
		t.Error("expected to see the Respond action text as an EventText")
	}
}

// --- ExecuteCode: AfterExec hook error ---

func TestCodeActAgent_ExecuteCode_AfterHookError(t *testing.T) {
	errAfter := errors.New("after-denied")
	a := NewCodeActAgent("after",
		WithExecutor(NewNoopExecutor()),
		WithCodeActHooks(CodeActHooks{
			AfterExec: func(_ context.Context, _ CodeAction, _ CodeResult) error {
				return errAfter
			},
		}),
	)
	_, err := a.ExecuteCode(context.Background(), CodeAction{Language: "python", Code: "pass"})
	if !errors.Is(err, errAfter) {
		t.Errorf("error = %v, want %v", err, errAfter)
	}
}

// --- Planner Generate / buildMessages / observationToMessage / extractResultText ---

func TestCodeActPlanner_Plan_And_Replan(t *testing.T) {
	model := &stubChatModel{
		resp: &schema.AIMessage{
			Parts: []schema.ContentPart{schema.TextPart{Text: "Here:\n```python\nprint(1)\n```"}},
		},
	}
	p := NewCodeActPlanner(model, WithPlannerLanguage("python"))

	// Plan (initial).
	actions, err := p.Plan(context.Background(), agent.PlannerState{
		Input:    "hello",
		Messages: []schema.Message{schema.NewHumanMessage("hello")},
	})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) != 1 || actions[0].Type != ActionCode {
		t.Fatalf("actions = %+v, want single ActionCode", actions)
	}

	// Replan with an observation fed back in to exercise observationToMessage.
	obs := agent.Observation{
		Action: agent.Action{
			Type: ActionCode,
			Metadata: map[string]any{
				"code":     "print(1)",
				"language": "python",
			},
		},
		Result: &tool.Result{Content: []schema.ContentPart{schema.TextPart{Text: "1"}}},
	}
	state := agent.PlannerState{
		Input:        "hello",
		Messages:     []schema.Message{schema.NewSystemMessage("sys"), schema.NewHumanMessage("hello")},
		Observations: []agent.Observation{obs},
	}
	actions2, err := p.Replan(context.Background(), state)
	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}
	if len(actions2) == 0 {
		t.Error("Replan returned no actions")
	}
}

func TestCodeActPlanner_Plan_LLMError(t *testing.T) {
	model := &stubChatModel{err: errors.New("llm-down")}
	p := NewCodeActPlanner(model)
	_, err := p.Plan(context.Background(), agent.PlannerState{})
	if err == nil {
		t.Error("expected error from failing LLM")
	}
}

func TestObservationToMessage_WithExecError(t *testing.T) {
	obs := agent.Observation{
		Action: agent.Action{
			Type: ActionCode,
			Metadata: map[string]any{
				"code":     "boom",
				"language": "python",
			},
		},
		Error: errors.New("runtime error"),
	}
	msg := observationToMessage(obs)
	if msg == nil {
		t.Fatal("observationToMessage returned nil")
	}
}

func TestExtractResultText(t *testing.T) {
	if got := extractResultText(nil); got != "" {
		t.Errorf("nil result should yield empty string, got %q", got)
	}
	if got := extractResultText(&tool.Result{}); got != "" {
		t.Errorf("empty result should yield empty string, got %q", got)
	}
	r := &tool.Result{Content: []schema.ContentPart{schema.TextPart{Text: "hello"}}}
	if got := extractResultText(r); got != "hello" {
		t.Errorf("extractResultText = %q, want hello", got)
	}
}

// --- executor.go: Execute error paths via NewProcessExecutor ---

func TestProcessExecutor_ExecuteRuns(t *testing.T) {
	// Use a shell built-in that exists virtually everywhere: /bin/echo.
	exec := NewProcessExecutor(WithInterpreter("echoer", "/bin/echo"))
	res, err := exec.Execute(context.Background(), CodeAction{Language: "echoer", Code: "ignored-stdin"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// /bin/echo ignores stdin and prints its own args; we only care that it ran.
	if res.Duration <= 0 {
		t.Error("expected non-zero duration")
	}
}
