package optimizer

import (
	"context"
	"iter"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id string
}

func (m *mockAgent) ID() string                  { return m.id }
func (m *mockAgent) Persona() agent.Persona       { return agent.Persona{} }
func (m *mockAgent) Tools() []tool.Tool            { return nil }
func (m *mockAgent) Children() []agent.Agent       { return nil }
func (m *mockAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	return input, nil
}
func (m *mockAgent) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {}
}

func TestDefaultCompiler(t *testing.T) {
	c := DefaultCompiler()
	if c == nil {
		t.Fatal("DefaultCompiler returned nil")
	}
}

func TestCompilerForStrategy(t *testing.T) {
	strategies := []OptimizationStrategy{
		StrategyBootstrapFewShot,
		StrategyMIPROv2,
		StrategyGEPA,
		StrategySIMBA,
	}

	for _, s := range strategies {
		t.Run(string(s), func(t *testing.T) {
			c := CompilerForStrategy(s)
			if c == nil {
				t.Fatalf("CompilerForStrategy(%q) returned nil", s)
			}
		})
	}
}

func TestBaseCompiler_Strategy(t *testing.T) {
	bc := NewBaseCompiler(StrategyGEPA)
	if bc.Strategy() != StrategyGEPA {
		t.Errorf("strategy: got %q, want %q", bc.Strategy(), StrategyGEPA)
	}
}

func TestCompile_RequiresMetric(t *testing.T) {
	c := DefaultCompiler()
	agt := &mockAgent{id: "test"}

	_, err := c.Compile(context.Background(), agt,
		WithTrainsetExamples([]Example{{Inputs: map[string]any{"q": "hello"}}}),
	)
	if err == nil {
		t.Fatal("expected error when metric is missing")
	}
}

func TestCompile_RequiresTrainset(t *testing.T) {
	c := DefaultCompiler()
	agt := &mockAgent{id: "test"}

	_, err := c.Compile(context.Background(), agt,
		WithMetric(&ExactMatchMetric{}),
	)
	if err == nil {
		t.Fatal("expected error when trainset is empty")
	}
}

func TestCompile_Success(t *testing.T) {
	// Override now for deterministic timing.
	origNow := now
	now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	defer func() { now = origNow }()

	c := DefaultCompiler()
	agt := &mockAgent{id: "test-agent"}

	result, err := c.Compile(context.Background(), agt,
		WithMetric(&ExactMatchMetric{}),
		WithTrainsetExamples([]Example{
			{Inputs: map[string]any{"q": "hello"}, Outputs: map[string]any{"answer": "world"}},
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestCompileWithResult_Success(t *testing.T) {
	origNow := now
	now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	defer func() { now = origNow }()

	c := DefaultCompiler()
	agt := &mockAgent{id: "test-agent"}

	result, err := c.CompileWithResult(context.Background(), agt,
		WithMetric(&ExactMatchMetric{}),
		WithTrainsetExamples([]Example{
			{Inputs: map[string]any{"q": "hello"}, Outputs: map[string]any{"answer": "world"}},
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Agent == nil {
		t.Error("result.Agent should not be nil")
	}
}

func TestCompile_WithCallbacks(t *testing.T) {
	origNow := now
	now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	defer func() { now = origNow }()

	var progressPhases []CompilePhase
	var completeCalled bool

	cb := CallbackFunc{
		OnProgressFunc: func(_ context.Context, p Progress) {
			progressPhases = append(progressPhases, p.Phase)
		},
		OnCompleteFunc: func(_ context.Context, _ Result) {
			completeCalled = true
		},
	}

	c := DefaultCompiler()
	agt := &mockAgent{id: "test"}

	_, err := c.Compile(context.Background(), agt,
		WithMetric(&ExactMatchMetric{}),
		WithTrainsetExamples([]Example{
			{Inputs: map[string]any{"q": "hello"}, Outputs: map[string]any{"answer": "world"}},
		}),
		WithCallback(cb),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(progressPhases) < 2 {
		t.Errorf("expected at least 2 progress updates, got %d", len(progressPhases))
	}
	if progressPhases[0] != PhaseInitializing {
		t.Errorf("first phase: got %q, want %q", progressPhases[0], PhaseInitializing)
	}
	if !completeCalled {
		t.Error("OnComplete should have been called")
	}
}

func TestCompilerRegistry(t *testing.T) {
	name := "test_compiler_" + t.Name()
	RegisterCompiler(name, func(_ CompilerConfig) (Compiler, error) {
		return DefaultCompiler(), nil
	})

	compilers := ListCompilers()
	found := false
	for _, c := range compilers {
		if c == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("registered compiler %q not found in list: %v", name, compilers)
	}

	c, err := NewCompiler(name, CompilerConfig{})
	if err != nil {
		t.Fatalf("NewCompiler: %v", err)
	}
	if c == nil {
		t.Fatal("NewCompiler returned nil")
	}
}

func TestNewCompiler_NotRegistered(t *testing.T) {
	_, err := NewCompiler("nonexistent_compiler", CompilerConfig{})
	if err == nil {
		t.Fatal("expected error for unregistered compiler")
	}
}

func TestMustCompiler_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unregistered compiler")
		}
	}()
	MustCompiler("nonexistent_must_compiler", CompilerConfig{})
}

func TestListCompilers_Sorted(t *testing.T) {
	RegisterCompiler("zzz_test_sort", func(_ CompilerConfig) (Compiler, error) {
		return DefaultCompiler(), nil
	})
	RegisterCompiler("aaa_test_sort", func(_ CompilerConfig) (Compiler, error) {
		return DefaultCompiler(), nil
	})

	list := ListCompilers()
	for i := 1; i < len(list); i++ {
		if list[i] < list[i-1] {
			t.Errorf("list not sorted: %v", list)
			break
		}
	}
}
