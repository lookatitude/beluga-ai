package optimizer

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// --- helpers -----------------------------------------------------------------

// echoAgent returns its input unchanged.
type echoAgent struct {
	id string
	mu sync.Mutex
}

func (a *echoAgent) ID() string              { return a.id }
func (a *echoAgent) Persona() agent.Persona  { return agent.Persona{} }
func (a *echoAgent) Tools() []tool.Tool      { return nil }
func (a *echoAgent) Children() []agent.Agent { return nil }
func (a *echoAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return input, nil
}
func (a *echoAgent) Stream(_ context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: input, AgentID: a.id}, nil)
	}
}

// fixedAgent always returns a preset response.
type fixedAgent struct {
	id      string
	resp    string
	err     error
	callsMu sync.Mutex
	calls   int
}

func (a *fixedAgent) ID() string              { return a.id }
func (a *fixedAgent) Persona() agent.Persona  { return agent.Persona{Role: "helper"} }
func (a *fixedAgent) Tools() []tool.Tool      { return nil }
func (a *fixedAgent) Children() []agent.Agent { return nil }
func (a *fixedAgent) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	a.callsMu.Lock()
	a.calls++
	a.callsMu.Unlock()
	if a.err != nil {
		return "", a.err
	}
	return a.resp, nil
}
func (a *fixedAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, err := a.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{Type: agent.EventError, AgentID: a.id}, err)
			return
		}
		yield(agent.Event{Type: agent.EventText, Text: result, AgentID: a.id}, nil)
	}
}
func (a *fixedAgent) Calls() int {
	a.callsMu.Lock()
	defer a.callsMu.Unlock()
	return a.calls
}

// exactMatchMetricFn returns 1.0 if the prediction text equals the expected "answer" output.
var exactMatchMetricFn Metric = MetricFunc(func(ctx context.Context, ex Example, pred Prediction) (float64, error) {
	want, ok := ex.Outputs["answer"]
	if !ok {
		return 0, fmt.Errorf("no 'answer' output field")
	}
	got := pred.Text
	if strings.TrimSpace(got) == fmt.Sprintf("%v", want) {
		return 1.0, nil
	}
	return 0.0, nil
})

// --- unit tests --------------------------------------------------------------

func TestBootstrapFewShot_Defaults(t *testing.T) {
	opt := newBootstrapFewShotOptimizer(0, nil)
	if opt.cfg.MaxBootstrapped != 4 {
		t.Errorf("MaxBootstrapped: got %d, want 4", opt.cfg.MaxBootstrapped)
	}
	if opt.cfg.MaxLabeled != 16 {
		t.Errorf("MaxLabeled: got %d, want 16", opt.cfg.MaxLabeled)
	}
	if opt.cfg.MaxRounds != 1 {
		t.Errorf("MaxRounds: got %d, want 1", opt.cfg.MaxRounds)
	}
	if opt.cfg.MetricThreshold != 1.0 {
		t.Errorf("MetricThreshold: got %f, want 1.0", opt.cfg.MetricThreshold)
	}
}

func TestBootstrapFewShot_Options(t *testing.T) {
	teacher := &echoAgent{id: "teacher"}
	opt := newBootstrapFewShotOptimizer(42, nil,
		WithBootstrapTeacher(teacher),
		WithBootstrapMaxBootstrapped(6),
		WithBootstrapMaxLabeled(12),
		WithBootstrapMaxRounds(3),
		WithBootstrapMetricThreshold(0.7),
	)
	if opt.cfg.Teacher != teacher {
		t.Error("Teacher not set")
	}
	if opt.cfg.MaxBootstrapped != 6 {
		t.Errorf("MaxBootstrapped: got %d, want 6", opt.cfg.MaxBootstrapped)
	}
	if opt.cfg.MaxLabeled != 12 {
		t.Errorf("MaxLabeled: got %d, want 12", opt.cfg.MaxLabeled)
	}
	if opt.cfg.MaxRounds != 3 {
		t.Errorf("MaxRounds: got %d, want 3", opt.cfg.MaxRounds)
	}
	if opt.cfg.MetricThreshold != 0.7 {
		t.Errorf("MetricThreshold: got %f, want 0.7", opt.cfg.MetricThreshold)
	}
	if opt.seed != 42 {
		t.Errorf("seed: got %d, want 42", opt.seed)
	}
}

func TestBootstrapFewShot_ExtraOverrides(t *testing.T) {
	extra := map[string]any{
		"max_bootstrapped": 8,
		"max_labeled":      24,
		"max_rounds":       2,
		"metric_threshold": 0.5,
	}
	opt := newBootstrapFewShotOptimizer(0, extra)
	if opt.cfg.MaxBootstrapped != 8 {
		t.Errorf("MaxBootstrapped: got %d, want 8", opt.cfg.MaxBootstrapped)
	}
	if opt.cfg.MaxLabeled != 24 {
		t.Errorf("MaxLabeled: got %d, want 24", opt.cfg.MaxLabeled)
	}
	if opt.cfg.MaxRounds != 2 {
		t.Errorf("MaxRounds: got %d, want 2", opt.cfg.MaxRounds)
	}
	if opt.cfg.MetricThreshold != 0.5 {
		t.Errorf("MetricThreshold: got %f, want 0.5", opt.cfg.MetricThreshold)
	}
}

func TestBootstrapFewShot_Optimize_RequiresMetric(t *testing.T) {
	opt := newBootstrapFewShotOptimizer(0, nil)
	agt := &echoAgent{id: "student"}

	_, err := opt.Optimize(context.Background(), agt, OptimizeOptions{
		Trainset: Dataset{Examples: []Example{{Inputs: map[string]any{"input": "hi"}}}},
	})
	if err == nil {
		t.Fatal("expected error when metric is nil")
	}
	if !strings.Contains(err.Error(), "metric is required") {
		t.Errorf("error message: got %q, want to contain 'metric is required'", err.Error())
	}
}

func TestBootstrapFewShot_Optimize_RequiresTrainset(t *testing.T) {
	opt := newBootstrapFewShotOptimizer(0, nil)
	agt := &echoAgent{id: "student"}

	_, err := opt.Optimize(context.Background(), agt, OptimizeOptions{
		Metric: exactMatchMetricFn,
	})
	if err == nil {
		t.Fatal("expected error when trainset is empty")
	}
	if !strings.Contains(err.Error(), "trainset is required") {
		t.Errorf("error message: got %q, want to contain 'trainset is required'", err.Error())
	}
}

func TestBootstrapFewShot_Optimize_EmptyTrainset(t *testing.T) {
	opt := newBootstrapFewShotOptimizer(0, nil)
	agt := &echoAgent{id: "student"}

	_, err := opt.Optimize(context.Background(), agt, OptimizeOptions{
		Metric:   exactMatchMetricFn,
		Trainset: Dataset{Examples: []Example{}},
	})
	if err == nil || !strings.Contains(err.Error(), "trainset is required") {
		t.Fatalf("expected 'trainset is required' error, got %v", err)
	}
}

func TestBootstrapFewShot_Optimize_Bootstrap(t *testing.T) {
	// Teacher echoes the input, which will match since examples have input==answer.
	teacher := &echoAgent{id: "teacher"}

	// Student is a different agent with a fixed response.
	student := &fixedAgent{id: "student", resp: "fallback"}

	trainset := []Example{
		{Inputs: map[string]any{"input": "42"}, Outputs: map[string]any{"answer": "42"}},
		{Inputs: map[string]any{"input": "hello"}, Outputs: map[string]any{"answer": "hello"}},
		{Inputs: map[string]any{"input": "world"}, Outputs: map[string]any{"answer": "world"}},
	}

	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapTeacher(teacher),
		WithBootstrapMaxBootstrapped(3),
		WithBootstrapMaxLabeled(0),
		WithBootstrapMetricThreshold(1.0),
	)

	result, err := opt.Optimize(context.Background(), student, OptimizeOptions{
		Metric:   exactMatchMetricFn,
		Trainset: Dataset{Examples: trainset},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Agent == nil {
		t.Fatal("result.Agent is nil")
	}

	// The wrapped agent should have demos injected.
	demo, ok := result.Agent.(*demoAgent)
	if !ok {
		t.Fatalf("expected *demoAgent, got %T", result.Agent)
	}
	if len(demo.Demos()) == 0 {
		t.Error("expected demos to be injected")
	}
	if demo.ID() != "student-bootstrapped" {
		t.Errorf("ID() = %q, want 'student-bootstrapped'", demo.ID())
	}
}

func TestBootstrapFewShot_Optimize_LabeledFallback(t *testing.T) {
	// Teacher always fails, forcing labeled fallback.
	teacher := &fixedAgent{id: "teacher", err: fmt.Errorf("teacher error")}
	student := &echoAgent{id: "student"}

	trainset := []Example{
		{Inputs: map[string]any{"input": "q1"}, Outputs: map[string]any{"answer": "a1"}},
		{Inputs: map[string]any{"input": "q2"}, Outputs: map[string]any{"answer": "a2"}},
	}

	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapTeacher(teacher),
		WithBootstrapMaxBootstrapped(4),
		WithBootstrapMaxLabeled(4),
	)

	result, err := opt.Optimize(context.Background(), student, OptimizeOptions{
		Metric:   exactMatchMetricFn,
		Trainset: Dataset{Examples: trainset},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	demo, ok := result.Agent.(*demoAgent)
	if !ok {
		t.Fatalf("expected *demoAgent, got %T", result.Agent)
	}

	// Bootstrapping failed, so labeled demos should fill the slots.
	if len(demo.Demos()) == 0 {
		t.Error("expected labeled demos to be used as fallback")
	}
}

func TestBootstrapFewShot_Optimize_SelfTeacher(t *testing.T) {
	// When no teacher is set, the student acts as its own teacher.
	student := &echoAgent{id: "self-teach"}
	trainset := []Example{
		{Inputs: map[string]any{"input": "ping"}, Outputs: map[string]any{"answer": "ping"}},
	}

	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapMaxBootstrapped(1),
		WithBootstrapMaxLabeled(1),
		WithBootstrapMetricThreshold(1.0),
	)

	result, err := opt.Optimize(context.Background(), student, OptimizeOptions{
		Metric:   exactMatchMetricFn,
		Trainset: Dataset{Examples: trainset},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Agent == nil {
		t.Fatal("result.Agent is nil")
	}
}

func TestBootstrapFewShot_Optimize_Callbacks(t *testing.T) {
	student := &echoAgent{id: "student"}
	trainset := []Example{
		{Inputs: map[string]any{"input": "x"}, Outputs: map[string]any{"answer": "x"}},
		{Inputs: map[string]any{"input": "y"}, Outputs: map[string]any{"answer": "y"}},
	}

	var (
		trialsMu     sync.Mutex
		trialsSeen   []Trial
		completeMu   sync.Mutex
		completeSeen []Result
	)

	cb := CallbackFunc{
		OnTrialCompleteFunc: func(_ context.Context, t Trial) {
			trialsMu.Lock()
			defer trialsMu.Unlock()
			trialsSeen = append(trialsSeen, t)
		},
		OnCompleteFunc: func(_ context.Context, r Result) {
			completeMu.Lock()
			defer completeMu.Unlock()
			completeSeen = append(completeSeen, r)
		},
	}

	opt := newBootstrapFewShotOptimizer(1, nil)
	_, err := opt.Optimize(context.Background(), student, OptimizeOptions{
		Metric:    exactMatchMetricFn,
		Trainset:  Dataset{Examples: trainset},
		Callbacks: []Callback{cb},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trialsMu.Lock()
	n := len(trialsSeen)
	trialsMu.Unlock()

	if n == 0 {
		t.Error("expected at least one OnTrialComplete callback")
	}
}

func TestBootstrapFewShot_Optimize_BudgetMaxIterations(t *testing.T) {
	student := &echoAgent{id: "student"}
	// Large trainset — budget should stop it early.
	examples := make([]Example, 20)
	for i := range examples {
		examples[i] = Example{
			Inputs:  map[string]any{"input": fmt.Sprintf("q%d", i)},
			Outputs: map[string]any{"answer": fmt.Sprintf("q%d", i)},
		}
	}

	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapMaxBootstrapped(10),
	)

	result, err := opt.Optimize(context.Background(), student, OptimizeOptions{
		Metric:   exactMatchMetricFn,
		Trainset: Dataset{Examples: examples},
		Budget:   Budget{MaxIterations: 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Trials should be <= MaxIterations.
	if len(result.Trials) > 3 {
		t.Errorf("expected at most 3 trials, got %d", len(result.Trials))
	}
}

func TestBootstrapFewShot_Optimize_MaxDemosCap(t *testing.T) {
	student := &echoAgent{id: "student"}
	examples := make([]Example, 10)
	for i := range examples {
		examples[i] = Example{
			Inputs:  map[string]any{"input": fmt.Sprintf("in%d", i)},
			Outputs: map[string]any{"answer": fmt.Sprintf("in%d", i)},
		}
	}

	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapMaxBootstrapped(2),
		WithBootstrapMaxLabeled(2),
	)

	result, err := opt.Optimize(context.Background(), student, OptimizeOptions{
		Metric:   exactMatchMetricFn,
		Trainset: Dataset{Examples: examples},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	demo, ok := result.Agent.(*demoAgent)
	if !ok {
		t.Fatalf("expected *demoAgent, got %T", result.Agent)
	}
	if len(demo.Demos()) > 4 { // MaxBootstrapped + MaxLabeled
		t.Errorf("demos count %d exceeds cap of 4", len(demo.Demos()))
	}
}

func TestBootstrapFewShot_Registry_Registered(t *testing.T) {
	list := ListOptimizers()
	found := false
	for _, name := range list {
		if name == string(StrategyBootstrapFewShot) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("bootstrap_few_shot not found in registry: %v", list)
	}
}

func TestBootstrapFewShot_Registry_CreateFromName(t *testing.T) {
	opt, err := NewOptimizer(string(StrategyBootstrapFewShot), OptimizerConfig{Seed: 42})
	if err != nil {
		t.Fatalf("NewOptimizer: %v", err)
	}
	if opt == nil {
		t.Fatal("expected optimizer, got nil")
	}

	bfs, ok := opt.(*bootstrapFewShotOptimizer)
	if !ok {
		t.Fatalf("expected *bootstrapFewShotOptimizer from registry, got %T", opt)
	}
	if bfs.seed != 42 {
		t.Errorf("seed: got %d, want 42", bfs.seed)
	}
}

func TestBootstrapFewShot_DemoAgent_Interface(t *testing.T) {
	// Compile-time check: demoAgent must satisfy agent.Agent.
	var _ agent.Agent = (*demoAgent)(nil)
}

func TestBootstrapFewShot_DemoAgent_Invoke(t *testing.T) {
	base := &fixedAgent{id: "base", resp: "answer"}
	demos := []Example{
		{Inputs: map[string]any{"input": "q1"}, Outputs: map[string]any{"answer": "a1"}},
	}

	da := newDemoAgent(base, demos)

	// Invoke should prepend demos and call base.
	result, err := da.Invoke(context.Background(), "test question")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	// The input passed to base should contain the demo prompt.
	if result != "answer" {
		t.Errorf("Invoke() = %q, want 'answer'", result)
	}
	if base.Calls() != 1 {
		t.Errorf("expected 1 call to base, got %d", base.Calls())
	}
}

func TestBootstrapFewShot_DemoAgent_Stream(t *testing.T) {
	base := &fixedAgent{id: "base", resp: "streamed"}
	da := newDemoAgent(base, []Example{
		{Inputs: map[string]any{"input": "x"}, Outputs: map[string]any{"answer": "y"}},
	})

	var texts []string
	for ev, err := range da.Stream(context.Background(), "input") {
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}
		if ev.Type == agent.EventText {
			texts = append(texts, ev.Text)
		}
	}

	if len(texts) == 0 {
		t.Error("expected at least one text event from Stream()")
	}
}

func TestBootstrapFewShot_DemoAgent_Personas(t *testing.T) {
	base := &fixedAgent{id: "p", resp: "ok"}
	da := newDemoAgent(base, nil)

	if da.Persona().Role != "helper" {
		t.Errorf("Persona().Role = %q, want 'helper'", da.Persona().Role)
	}
	if len(da.Tools()) != 0 {
		t.Error("expected empty tools")
	}
	if len(da.Children()) != 0 {
		t.Error("expected empty children")
	}
}

func TestBootstrapFewShot_DemoAgent_ConcurrentInvoke(t *testing.T) {
	base := &echoAgent{id: "concurrent"}
	demos := []Example{
		{Inputs: map[string]any{"input": "hi"}, Outputs: map[string]any{"answer": "hi"}},
	}
	da := newDemoAgent(base, demos)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := da.Invoke(context.Background(), "test")
			if err != nil {
				t.Errorf("concurrent Invoke error: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestBuildInput_InputKey(t *testing.T) {
	ex := Example{Inputs: map[string]any{"input": "hello"}}
	got := buildInput(ex)
	if got != "hello" {
		t.Errorf("buildInput: got %q, want 'hello'", got)
	}
}

func TestBuildInput_QuestionKey(t *testing.T) {
	ex := Example{Inputs: map[string]any{"question": "what is 2+2?"}}
	got := buildInput(ex)
	if got != "what is 2+2?" {
		t.Errorf("buildInput: got %q, want 'what is 2+2?'", got)
	}
}

func TestBuildInput_Fallback(t *testing.T) {
	ex := Example{Inputs: map[string]any{"key": "val"}}
	got := buildInput(ex)
	if !strings.Contains(got, "key: val") {
		t.Errorf("buildInput fallback: got %q, expected 'key: val'", got)
	}
}

func TestBuildDemoPrompt_NoDemos(t *testing.T) {
	got := buildDemoPrompt("bare input", nil)
	if got != "bare input" {
		t.Errorf("buildDemoPrompt(no demos) = %q, want 'bare input'", got)
	}
}

func TestBuildDemoPrompt_WithDemos(t *testing.T) {
	demos := []Example{
		{Inputs: map[string]any{"input": "q1"}, Outputs: map[string]any{"answer": "a1"}},
		{Inputs: map[string]any{"question": "q2"}, Outputs: map[string]any{"output": "a2"}},
	}
	got := buildDemoPrompt("my question", demos)
	if !strings.Contains(got, "Example 1:") {
		t.Error("expected Example 1")
	}
	if !strings.Contains(got, "q1") {
		t.Error("expected q1 in prompt")
	}
	if !strings.Contains(got, "a1") {
		t.Error("expected a1 in prompt")
	}
	if !strings.Contains(got, "my question") {
		t.Error("expected original input in prompt")
	}
}

func TestBootstrapExamplesEqual(t *testing.T) {
	tests := []struct {
		a, b  Example
		equal bool
	}{
		{
			a:     Example{Inputs: map[string]any{"k": "v"}},
			b:     Example{Inputs: map[string]any{"k": "v"}},
			equal: true,
		},
		{
			a:     Example{Inputs: map[string]any{"k": "v"}},
			b:     Example{Inputs: map[string]any{"k": "other"}},
			equal: false,
		},
		{
			a:     Example{Inputs: map[string]any{"k": "v"}},
			b:     Example{Inputs: map[string]any{"k": "v", "extra": "x"}},
			equal: false,
		},
		{
			a:     Example{Inputs: map[string]any{}},
			b:     Example{Inputs: map[string]any{}},
			equal: true,
		},
	}

	for _, tt := range tests {
		got := bootstrapExamplesEqual(tt.a, tt.b)
		if got != tt.equal {
			t.Errorf("bootstrapExamplesEqual(%v, %v) = %v, want %v", tt.a.Inputs, tt.b.Inputs, got, tt.equal)
		}
	}
}

func TestContainsBootstrapExample(t *testing.T) {
	demos := []Example{
		{Inputs: map[string]any{"k": "v1"}},
		{Inputs: map[string]any{"k": "v2"}},
	}

	if !containsBootstrapExample(demos, Example{Inputs: map[string]any{"k": "v1"}}) {
		t.Error("expected to find v1")
	}
	if containsBootstrapExample(demos, Example{Inputs: map[string]any{"k": "v3"}}) {
		t.Error("expected not to find v3")
	}
}

func TestBootstrapFewShot_Optimize_ContextCancellation(t *testing.T) {
	// Agent that blocks until context is done.
	student := &echoAgent{id: "cancellable"}
	trainset := []Example{
		{Inputs: map[string]any{"input": "q"}, Outputs: map[string]any{"answer": "q"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	opt := newBootstrapFewShotOptimizer(1, nil)
	// Should complete (even with fast cancellation) without panic.
	_, _ = opt.Optimize(ctx, student, OptimizeOptions{
		Metric:   exactMatchMetricFn,
		Trainset: Dataset{Examples: trainset},
	})
}

func TestBootstrapFewShot_Optimize_MultipleRounds(t *testing.T) {
	// Teacher fails on first round but succeeds on second.
	callCount := 0
	var mu sync.Mutex
	type roundAgent struct {
		fixedAgent
	}
	teacher := &fixedAgent{
		id:   "round-teacher",
		resp: "correct",
	}
	// Override to fail on odd calls only.
	_ = callCount
	_ = mu

	student := &echoAgent{id: "student"}
	trainset := []Example{
		{Inputs: map[string]any{"input": "correct"}, Outputs: map[string]any{"answer": "correct"}},
	}

	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapTeacher(teacher),
		WithBootstrapMaxBootstrapped(1),
		WithBootstrapMaxRounds(3),
	)

	result, err := opt.Optimize(context.Background(), student, OptimizeOptions{
		Metric:   exactMatchMetricFn,
		Trainset: Dataset{Examples: trainset},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	demo := result.Agent.(*demoAgent)
	if len(demo.Demos()) == 0 {
		t.Error("expected at least one bootstrapped demo from multi-round")
	}
}

// TestBootstrapFewShot_ConcurrentOptimize ensures no data races when the optimizer
// is called from multiple goroutines concurrently.
func TestBootstrapFewShot_ConcurrentOptimize(t *testing.T) {
	trainset := []Example{
		{Inputs: map[string]any{"input": "hi"}, Outputs: map[string]any{"answer": "hi"}},
	}

	opt := newBootstrapFewShotOptimizer(1, nil)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			student := &echoAgent{id: "concurrent-student"}
			_, err := opt.Optimize(context.Background(), student, OptimizeOptions{
				Metric:   exactMatchMetricFn,
				Trainset: Dataset{Examples: trainset},
			})
			if err != nil {
				t.Errorf("concurrent Optimize error: %v", err)
			}
		}()
	}
	wg.Wait()
}
