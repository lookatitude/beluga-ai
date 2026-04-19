package codeact

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// --- ExtractCodeBlocks tests ---

func TestExtractCodeBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []CodeBlock
	}{
		{
			name:  "no code blocks",
			input: "Just plain text with no code.",
			want:  nil,
		},
		{
			name:  "single python block",
			input: "Here is some code:\n```python\nprint('hello')\n```\nDone.",
			want:  []CodeBlock{{Language: "python", Code: "print('hello')"}},
		},
		{
			name:  "multiple blocks",
			input: "```python\nx = 1\n```\nThen:\n```javascript\nconsole.log('hi')\n```",
			want: []CodeBlock{
				{Language: "python", Code: "x = 1"},
				{Language: "javascript", Code: "console.log('hi')"},
			},
		},
		{
			name:  "block without language tag",
			input: "```\necho hello\n```",
			want:  []CodeBlock{{Language: "", Code: "echo hello"}},
		},
		{
			name:  "multiline code",
			input: "```python\nimport math\nresult = math.sqrt(16)\nprint(result)\n```",
			want:  []CodeBlock{{Language: "python", Code: "import math\nresult = math.sqrt(16)\nprint(result)"}},
		},
		{
			name:  "empty code block",
			input: "```python\n\n```",
			want:  []CodeBlock{{Language: "python", Code: ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCodeBlocksEqual(t, ExtractCodeBlocks(tt.input), tt.want)
		})
	}
}

func assertCodeBlocksEqual(t *testing.T, got, want []CodeBlock) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if len(got) != len(want) {
		t.Fatalf("got %d blocks, want %d", len(got), len(want))
	}
	for i, block := range got {
		if block.Language != want[i].Language {
			t.Errorf("block[%d].Language = %q, want %q", i, block.Language, want[i].Language)
		}
		if block.Code != want[i].Code {
			t.Errorf("block[%d].Code = %q, want %q", i, block.Code, want[i].Code)
		}
	}
}

// --- NoopExecutor tests ---

func TestNoopExecutor(t *testing.T) {
	exec := NewNoopExecutor()
	ctx := context.Background()

	tests := []struct {
		name   string
		action CodeAction
		want   CodeResult
	}{
		{
			name:   "returns code as output",
			action: CodeAction{Language: "python", Code: "print('hello')"},
			want:   CodeResult{Output: "print('hello')", ExitCode: 0},
		},
		{
			name:   "empty code",
			action: CodeAction{Language: "go", Code: ""},
			want:   CodeResult{Output: "", ExitCode: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := exec.Execute(ctx, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Output != tt.want.Output {
				t.Errorf("Output = %q, want %q", got.Output, tt.want.Output)
			}
			if got.ExitCode != tt.want.ExitCode {
				t.Errorf("ExitCode = %d, want %d", got.ExitCode, tt.want.ExitCode)
			}
		})
	}
}

// --- ProcessExecutor tests ---

func TestProcessExecutor_UnsupportedLanguage(t *testing.T) {
	exec := NewProcessExecutor()
	ctx := context.Background()

	_, err := exec.Execute(ctx, CodeAction{Language: "cobol", Code: "DISPLAY 'HELLO'"})
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
	if !containsString(err.Error(), "unsupported language") {
		t.Errorf("error = %q, want to contain 'unsupported language'", err.Error())
	}
}

func TestProcessExecutor_EmptyCode(t *testing.T) {
	exec := NewProcessExecutor()
	ctx := context.Background()

	_, err := exec.Execute(ctx, CodeAction{Language: "python", Code: ""})
	if err == nil {
		t.Fatal("expected error for empty code")
	}
	if !containsString(err.Error(), "empty code") {
		t.Errorf("error = %q, want to contain 'empty code'", err.Error())
	}
}

func TestProcessExecutor_WithInterpreter(t *testing.T) {
	exec := NewProcessExecutor(
		WithInterpreter("bash", "bash"),
		WithDefaultTimeout(10*time.Second),
	)
	if exec.interpreters["bash"] != "bash" {
		t.Error("expected bash interpreter to be registered")
	}
	if exec.defaultTimeout != 10*time.Second {
		t.Errorf("defaultTimeout = %v, want 10s", exec.defaultTimeout)
	}
}

// --- CodeResult tests ---

func TestCodeResult_Success(t *testing.T) {
	tests := []struct {
		name   string
		result CodeResult
		want   bool
	}{
		{
			name:   "success",
			result: CodeResult{Output: "ok", ExitCode: 0},
			want:   true,
		},
		{
			name:   "non-zero exit",
			result: CodeResult{Output: "", ExitCode: 1},
			want:   false,
		},
		{
			// Stderr output alone must not imply failure (POSIX exit-code semantics).
			name:   "stderr with zero exit is success",
			result: CodeResult{Output: "ok", Error: "warning", ExitCode: 0},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Success(); got != tt.want {
				t.Errorf("Success() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- ExecutionState tests ---

func TestExecutionState_Variables(t *testing.T) {
	s := NewExecutionState()

	// Get missing variable
	_, ok := s.GetVariable("x")
	if ok {
		t.Error("expected missing variable")
	}

	// Set and get
	s.SetVariable("x", "42")
	v, ok := s.GetVariable("x")
	if !ok || v != "42" {
		t.Errorf("got (%q, %v), want (\"42\", true)", v, ok)
	}

	// Variables copy
	vars := s.Variables()
	if len(vars) != 1 || vars["x"] != "42" {
		t.Errorf("Variables() = %v, want map[x:42]", vars)
	}

	// Verify copy isolation
	vars["y"] = "99"
	_, ok = s.GetVariable("y")
	if ok {
		t.Error("Variables() should return a copy, not a reference")
	}
}

func TestExecutionState_Outputs(t *testing.T) {
	s := NewExecutionState()

	if s.StepCount() != 0 {
		t.Errorf("StepCount() = %d, want 0", s.StepCount())
	}
	if s.LastOutput() != nil {
		t.Error("LastOutput() should be nil for empty state")
	}

	s.AddOutput(StepOutput{Language: "python", Code: "print(1)", Output: "1", ExitCode: 0})
	s.AddOutput(StepOutput{Language: "python", Code: "print(2)", Output: "2", ExitCode: 0})

	if s.StepCount() != 2 {
		t.Errorf("StepCount() = %d, want 2", s.StepCount())
	}

	last := s.LastOutput()
	if last == nil || last.Output != "2" {
		t.Errorf("LastOutput().Output = %v, want \"2\"", last)
	}

	// Copy isolation
	outputs := s.Outputs()
	if len(outputs) != 2 {
		t.Fatalf("Outputs() len = %d, want 2", len(outputs))
	}
}

func TestGetOrCreateState(t *testing.T) {
	meta := make(map[string]any)

	// Creates new state
	s1 := GetOrCreateState(meta)
	if s1 == nil {
		t.Fatal("expected non-nil state")
	}

	// Returns existing state
	s1.SetVariable("x", "1")
	s2 := GetOrCreateState(meta)
	v, _ := s2.GetVariable("x")
	if v != "1" {
		t.Error("expected to get same state instance")
	}

	// Handles non-ExecutionState in metadata
	meta[stateKey] = "invalid"
	s3 := GetOrCreateState(meta)
	if s3 == nil {
		t.Fatal("expected new state when metadata has wrong type")
	}
	if _, ok := s3.GetVariable("x"); ok {
		t.Error("expected fresh state")
	}
}

// --- CodeActPlanner tests ---

func TestCodeActPlanner_ParsesCodeBlocks(t *testing.T) {
	planner := NewCodeActPlanner(nil, WithPlannerLanguage("python"))

	resp := &schema.AIMessage{
		Parts: []schema.ContentPart{
			schema.TextPart{Text: "Here is the solution:\n```python\nprint(42)\n```"},
		},
	}

	actions := planner.parseResponse(resp)
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].Type != ActionCode {
		t.Errorf("action type = %q, want %q", actions[0].Type, ActionCode)
	}
	if actions[0].Metadata["code"] != "print(42)" {
		t.Errorf("code = %q, want \"print(42)\"", actions[0].Metadata["code"])
	}
	if actions[0].Metadata["language"] != "python" {
		t.Errorf("language = %q, want \"python\"", actions[0].Metadata["language"])
	}
}

func TestCodeActPlanner_NoCodeBlockFinishes(t *testing.T) {
	planner := NewCodeActPlanner(nil)

	resp := &schema.AIMessage{
		Parts: []schema.ContentPart{
			schema.TextPart{Text: "The answer is 42."},
		},
	}

	actions := planner.parseResponse(resp)
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].Type != agent.ActionFinish {
		t.Errorf("action type = %q, want %q", actions[0].Type, agent.ActionFinish)
	}
	if actions[0].Message != "The answer is 42." {
		t.Errorf("message = %q, want \"The answer is 42.\"", actions[0].Message)
	}
}

func TestCodeActPlanner_DefaultsLanguageFromPlanner(t *testing.T) {
	planner := NewCodeActPlanner(nil, WithPlannerLanguage("go"))

	resp := &schema.AIMessage{
		Parts: []schema.ContentPart{
			schema.TextPart{Text: "```\nfmt.Println(42)\n```"},
		},
	}

	actions := planner.parseResponse(resp)
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].Metadata["language"] != "go" {
		t.Errorf("language = %q, want \"go\"", actions[0].Metadata["language"])
	}
}

func TestCodeActPlanner_MultipleCodeBlocks(t *testing.T) {
	planner := NewCodeActPlanner(nil, WithPlannerLanguage("python"))

	resp := &schema.AIMessage{
		Parts: []schema.ContentPart{
			schema.TextPart{Text: "Step 1:\n```python\nx = 1\n```\nStep 2:\n```python\nprint(x)\n```"},
		},
	}

	actions := planner.parseResponse(resp)
	if len(actions) != 2 {
		t.Fatalf("got %d actions, want 2", len(actions))
	}
	if actions[0].Metadata["code"] != "x = 1" {
		t.Errorf("actions[0].code = %q, want \"x = 1\"", actions[0].Metadata["code"])
	}
	if actions[1].Metadata["code"] != "print(x)" {
		t.Errorf("actions[1].code = %q, want \"print(x)\"", actions[1].Metadata["code"])
	}
}

// --- CodeActHooks tests ---

func TestComposeCodeActHooks(t *testing.T) {
	var callOrder []string

	h1 := CodeActHooks{
		BeforeExec: func(_ context.Context, _ CodeAction) error {
			callOrder = append(callOrder, "h1-before")
			return nil
		},
		AfterExec: func(_ context.Context, _ CodeAction, _ CodeResult) error {
			callOrder = append(callOrder, "h1-after")
			return nil
		},
	}
	h2 := CodeActHooks{
		BeforeExec: func(_ context.Context, _ CodeAction) error {
			callOrder = append(callOrder, "h2-before")
			return nil
		},
		AfterExec: func(_ context.Context, _ CodeAction, _ CodeResult) error {
			callOrder = append(callOrder, "h2-after")
			return nil
		},
	}

	composed := ComposeCodeActHooks(h1, h2)
	ctx := context.Background()
	action := CodeAction{Language: "python", Code: "pass"}

	if err := composed.BeforeExec(ctx, action); err != nil {
		t.Fatalf("BeforeExec error: %v", err)
	}
	if err := composed.AfterExec(ctx, action, CodeResult{}); err != nil {
		t.Fatalf("AfterExec error: %v", err)
	}

	expected := []string{"h1-before", "h2-before", "h1-after", "h2-after"}
	if len(callOrder) != len(expected) {
		t.Fatalf("call order = %v, want %v", callOrder, expected)
	}
	for i, v := range callOrder {
		if v != expected[i] {
			t.Errorf("callOrder[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestComposeCodeActHooks_ErrorShortCircuits(t *testing.T) {
	errStop := errors.New("stop")

	h1 := CodeActHooks{
		BeforeExec: func(_ context.Context, _ CodeAction) error {
			return errStop
		},
	}
	h2 := CodeActHooks{
		BeforeExec: func(_ context.Context, _ CodeAction) error {
			t.Fatal("h2.BeforeExec should not be called")
			return nil
		},
	}

	composed := ComposeCodeActHooks(h1, h2)
	err := composed.BeforeExec(context.Background(), CodeAction{})
	if !errors.Is(err, errStop) {
		t.Errorf("error = %v, want %v", err, errStop)
	}
}

// --- CodeActAgent tests ---

func TestCodeActAgent_ExecuteCode(t *testing.T) {
	var hookCalls []string

	a := NewCodeActAgent("test-agent",
		WithLanguage("python"),
		WithExecutor(NewNoopExecutor()),
		WithExecTimeout(10*time.Second),
		WithCodeActHooks(CodeActHooks{
			BeforeExec: func(_ context.Context, action CodeAction) error {
				hookCalls = append(hookCalls, fmt.Sprintf("before:%s", action.Language))
				return nil
			},
			AfterExec: func(_ context.Context, _ CodeAction, result CodeResult) error {
				hookCalls = append(hookCalls, fmt.Sprintf("after:%s", result.Output))
				return nil
			},
		}),
	)

	ctx := context.Background()
	result, err := a.ExecuteCode(ctx, CodeAction{Code: "print('hello')"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "print('hello')" {
		t.Errorf("Output = %q, want \"print('hello')\"", result.Output)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	if len(hookCalls) != 2 {
		t.Fatalf("hookCalls = %v, want 2 entries", hookCalls)
	}
	if hookCalls[0] != "before:python" {
		t.Errorf("hookCalls[0] = %q, want \"before:python\"", hookCalls[0])
	}
	if hookCalls[1] != "after:print('hello')" {
		t.Errorf("hookCalls[1] = %q, want \"after:print('hello')\"", hookCalls[1])
	}
}

func TestCodeActAgent_ExecuteCode_DefaultsLanguage(t *testing.T) {
	a := NewCodeActAgent("test",
		WithLanguage("go"),
		WithExecutor(NewNoopExecutor()),
	)

	ctx := context.Background()
	// Action with no language should default to agent's language
	result, err := a.ExecuteCode(ctx, CodeAction{Code: "fmt.Println(1)"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "fmt.Println(1)" {
		t.Errorf("Output = %q, want code echoed back", result.Output)
	}
}

func TestCodeActAgent_ExecuteCode_BeforeHookCancels(t *testing.T) {
	errDenied := errors.New("denied")

	a := NewCodeActAgent("test",
		WithExecutor(NewNoopExecutor()),
		WithCodeActHooks(CodeActHooks{
			BeforeExec: func(_ context.Context, _ CodeAction) error {
				return errDenied
			},
		}),
	)

	ctx := context.Background()
	_, err := a.ExecuteCode(ctx, CodeAction{Language: "python", Code: "pass"})
	if !errors.Is(err, errDenied) {
		t.Errorf("error = %v, want %v", err, errDenied)
	}
}

// stubPlanner returns a canned sequence of action lists across iterations.
type stubPlanner struct {
	iterations [][]agent.Action
	calls      int
}

func (p *stubPlanner) Plan(_ context.Context, _ agent.PlannerState) ([]agent.Action, error) {
	if p.calls >= len(p.iterations) {
		return []agent.Action{{Type: agent.ActionFinish, Message: "done"}}, nil
	}
	actions := p.iterations[p.calls]
	p.calls++
	return actions, nil
}

func (p *stubPlanner) Replan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	return p.Plan(ctx, state)
}

func TestCodeActAgent_Stream_ExecutesCodeAction(t *testing.T) {
	planner := &stubPlanner{
		iterations: [][]agent.Action{
			{{
				Type: ActionCode,
				Metadata: map[string]any{
					"language": "python",
					"code":     "print('hi')",
				},
			}},
			{{Type: agent.ActionFinish, Message: "final-answer"}},
		},
	}

	a := NewCodeActAgent("codeact-test",
		WithPlanner(planner),
		WithExecutor(NewNoopExecutor()),
		WithMaxIterations(5),
	)

	obs := collectStreamObservations(t, a)

	if !obs.sawExec {
		t.Error("expected EventCodeExec event")
	}
	if !obs.sawResult {
		t.Error("expected EventCodeResult event")
	}
	if !obs.sawDone {
		t.Error("expected EventDone event")
	}
	if obs.finalText != "final-answer" {
		t.Errorf("finalText = %q, want final-answer", obs.finalText)
	}
	if planner.calls < 2 {
		t.Errorf("planner.calls = %d, want >= 2 (code observation fed back)", planner.calls)
	}
}

type streamObs struct {
	sawExec, sawResult, sawDone bool
	finalText                   string
}

func collectStreamObservations(t *testing.T, a *CodeActAgent) streamObs {
	t.Helper()
	var obs streamObs
	for event, err := range a.Stream(context.Background(), "hello") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		switch event.Type {
		case EventCodeExec:
			obs.sawExec = true
			if event.Text != "print('hi')" {
				t.Errorf("EventCodeExec Text = %q, want print('hi')", event.Text)
			}
		case EventCodeResult:
			obs.sawResult = true
			if event.ToolResult == nil {
				t.Error("EventCodeResult ToolResult is nil")
			}
		case agent.EventDone:
			obs.sawDone = true
			obs.finalText = event.Text
		}
	}
	return obs
}

func TestCodeActAgent_ID(t *testing.T) {
	a := NewCodeActAgent("my-agent")
	if a.ID() != "my-agent" {
		t.Errorf("ID() = %q, want \"my-agent\"", a.ID())
	}
}

// --- Planner registry test ---

func TestCodeActPlannerRegistered(t *testing.T) {
	planners := agent.ListPlanners()
	found := false
	for _, name := range planners {
		if name == "codeact" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("codeact planner not registered, available: %v", planners)
	}
}

// --- System prompt tests ---

func TestCodeActPlanner_SystemPrompt(t *testing.T) {
	p := NewCodeActPlanner(nil, WithPlannerLanguage("javascript"), WithAllowedImports([]string{"fs", "path"}))
	prompt := p.systemPrompt()

	if !containsString(prompt, "javascript") {
		t.Error("system prompt should mention the language")
	}
	if !containsString(prompt, "fs, path") {
		t.Error("system prompt should mention allowed imports")
	}
}

func TestCodeActPlanner_SystemPrompt_NoImports(t *testing.T) {
	p := NewCodeActPlanner(nil)
	prompt := p.systemPrompt()
	if containsString(prompt, "Only use these imports") {
		t.Error("system prompt should not mention imports when none are set")
	}
}

// --- codeResultToToolResult test ---

func TestCodeResultToToolResult(t *testing.T) {
	tests := []struct {
		name   string
		result CodeResult
		want   string
	}{
		{
			name:   "success",
			result: CodeResult{Output: "42", ExitCode: 0},
			want:   "42",
		},
		{
			name:   "failure",
			result: CodeResult{Output: "", Error: "NameError", ExitCode: 1},
			want:   "Exit code: 1\nStdout: \nStderr: NameError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := codeResultToToolResult(tt.result)
			if len(tr.Content) != 1 {
				t.Fatalf("content len = %d, want 1", len(tr.Content))
			}
			tp, ok := tr.Content[0].(schema.TextPart)
			if !ok {
				t.Fatal("content[0] is not TextPart")
			}
			if tp.Text != tt.want {
				t.Errorf("text = %q, want %q", tp.Text, tt.want)
			}
		})
	}
}

// --- optsFromExtra tests ---

func TestOptsFromExtra_Nil(t *testing.T) {
	opts := optsFromExtra(nil)
	if len(opts) != 0 {
		t.Errorf("expected no opts from nil extra, got %d", len(opts))
	}
}

func TestOptsFromExtra_WithValues(t *testing.T) {
	extra := map[string]any{
		"language":        "go",
		"allowed_imports": []string{"fmt", "os"},
	}
	opts := optsFromExtra(extra)
	if len(opts) != 2 {
		t.Errorf("expected 2 opts, got %d", len(opts))
	}

	// Apply opts and verify
	p := &CodeActPlanner{language: "python"}
	for _, opt := range opts {
		opt(p)
	}
	if p.language != "go" {
		t.Errorf("language = %q, want \"go\"", p.language)
	}
	if len(p.allowedImports) != 2 {
		t.Errorf("allowedImports len = %d, want 2", len(p.allowedImports))
	}
}

// helper
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
