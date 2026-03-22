package optimizer_test

// integration_test.go exercises the full optimizer pipeline end-to-end,
// verifying that all four strategies (BootstrapFewShot, MIPROv2, GEPA, SIMBA)
// work correctly when wired together through the optimizer.Compiler API and
// the optimizer ↔ optimize bridge layer.
//
// All tests are race-free: they use no shared mutable state outside of the
// explicitly sync-guarded registries (which are themselves protected by RWMutex).
//
// The "integration" build tag is NOT required here: the tests use only mock
// agents and mock metrics so they run without any external dependencies.

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	optpkg "github.com/lookatitude/beluga-ai/optimizer"
	"github.com/lookatitude/beluga-ai/tool"

	// Register all four optimize.Optimizer implementations so the bridge can look them up.
	_ "github.com/lookatitude/beluga-ai/optimize/optimizers"
)

// ── mock agent ────────────────────────────────────────────────────────────────

// echoAgent returns the first letter of the input as a deterministic "answer".
// This lets exact-match metrics score as 1.0 when the expected output is the
// same first letter — trivially satisfying the metric without real LLM calls.
//
// When the optimizer wraps input with few-shot demonstrations (e.g.
// "Here are some examples:...\nNow answer the following:\nactual-input"),
// the agent extracts the actual question before responding.
type echoAgent struct {
	id    string
	mu    sync.Mutex
	calls int
}

func (a *echoAgent) ID() string              { return a.id }
func (a *echoAgent) Persona() agent.Persona  { return agent.Persona{Role: "test"} }
func (a *echoAgent) Tools() []tool.Tool      { return nil }
func (a *echoAgent) Children() []agent.Agent { return nil }

func (a *echoAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	a.mu.Lock()
	a.calls++
	a.mu.Unlock()
	// Strip demo prefix injected by the optimizer's demoAgent wrapper.
	if idx := strings.LastIndex(input, "Now answer the following:\n"); idx >= 0 {
		input = input[idx+len("Now answer the following:\n"):]
	}
	if len(input) == 0 {
		return "x", nil
	}
	return string(input[0]), nil
}

func (a *echoAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, err := a.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{Type: agent.EventError, AgentID: a.id}, err)
			return
		}
		if !yield(agent.Event{Type: agent.EventText, Text: result, AgentID: a.id}, nil) {
			return
		}
		yield(agent.Event{Type: agent.EventDone, AgentID: a.id}, nil)
	}
}

// ── test helpers ──────────────────────────────────────────────────────────────

// makeExamples returns n examples where the expected answer is the first letter of the input.
func makeExamples(n int) []optpkg.Example {
	ex := make([]optpkg.Example, n)
	for i := range ex {
		q := fmt.Sprintf("%c-question-%d", rune('a'+i%26), i)
		ex[i] = optpkg.Example{
			Inputs:  map[string]any{"input": q, "question": q},
			Outputs: map[string]any{"answer": string(q[0])},
		}
	}
	return ex
}

// containsMetric scores 1.0 when the prediction's Text contains the expected answer.
var containsMetric = optpkg.MetricFunc(func(_ context.Context, ex optpkg.Example, pred optpkg.Prediction) (float64, error) {
	expected, _ := ex.Outputs["answer"].(string)
	if strings.Contains(strings.ToLower(pred.Text), strings.ToLower(expected)) {
		return 1.0, nil
	}
	return 0.0, nil
})

// ── compile + evaluate helper ─────────────────────────────────────────────────

func compileAndCheck(t *testing.T, strategy optpkg.OptimizationStrategy, trainset, testset []optpkg.Example) {
	t.Helper()

	agt := &echoAgent{id: fmt.Sprintf("agent-%s", strategy)}
	compiler := optpkg.CompilerForStrategy(strategy)

	optimized, err := compiler.Compile(context.Background(), agt,
		optpkg.WithMetric(containsMetric),
		optpkg.WithTrainsetExamples(trainset),
		optpkg.WithMaxIterations(3),
		optpkg.WithSeed(42),
	)
	if err != nil {
		t.Fatalf("strategy %q Compile: %v", strategy, err)
	}
	if optimized == nil {
		t.Fatalf("strategy %q: compiled agent is nil", strategy)
	}

	// Evaluate on test set.
	ctx := context.Background()
	var correct int
	for _, ex := range testset {
		input, _ := ex.Inputs["input"].(string)
		result, err := optimized.Invoke(ctx, input)
		if err != nil {
			continue
		}
		expected, _ := ex.Outputs["answer"].(string)
		if strings.Contains(strings.ToLower(result), strings.ToLower(expected)) {
			correct++
		}
	}
	if len(testset) > 0 && correct == 0 {
		t.Errorf("strategy %q: 0/%d test examples passed — optimized agent may be broken",
			strategy, len(testset))
	}
}

// ── tests per strategy ────────────────────────────────────────────────────────

func TestIntegration_BootstrapFewShot(t *testing.T) {
	trainset := makeExamples(10)
	testset := makeExamples(3)
	compileAndCheck(t, optpkg.StrategyBootstrapFewShot, trainset, testset)
}

func TestIntegration_MIPROv2(t *testing.T) {
	trainset := makeExamples(8)
	testset := makeExamples(3)
	compileAndCheck(t, optpkg.StrategyMIPROv2, trainset, testset)
}

func TestIntegration_GEPA(t *testing.T) {
	trainset := makeExamples(8)
	testset := makeExamples(3)
	compileAndCheck(t, optpkg.StrategyGEPA, trainset, testset)
}

func TestIntegration_SIMBA(t *testing.T) {
	trainset := makeExamples(8)
	testset := makeExamples(3)
	compileAndCheck(t, optpkg.StrategySIMBA, trainset, testset)
}

// ── CompileWithResult ─────────────────────────────────────────────────────────

func TestIntegration_CompileWithResult_AllStrategies(t *testing.T) {
	strategies := []optpkg.OptimizationStrategy{
		optpkg.StrategyBootstrapFewShot,
		optpkg.StrategyMIPROv2,
		optpkg.StrategyGEPA,
		optpkg.StrategySIMBA,
	}
	trainset := makeExamples(6)

	for _, strategy := range strategies {
		strategy := strategy // capture
		t.Run(string(strategy), func(t *testing.T) {
			agt := &echoAgent{id: "test"}
			compiler := optpkg.CompilerForStrategy(strategy)

			result, err := compiler.CompileWithResult(context.Background(), agt,
				optpkg.WithMetric(containsMetric),
				optpkg.WithTrainsetExamples(trainset),
				optpkg.WithMaxIterations(2),
				optpkg.WithSeed(1),
			)
			if err != nil {
				t.Fatalf("CompileWithResult: %v", err)
			}
			if result == nil {
				t.Fatal("result is nil")
			}
			if result.Agent == nil {
				t.Error("result.Agent is nil")
			}
		})
	}
}

// ── Callback integration ──────────────────────────────────────────────────────

func TestIntegration_Callbacks_Fired(t *testing.T) {
	var progressCount, completeCount int64

	cb := optpkg.CallbackFunc{
		OnProgressFunc: func(_ context.Context, _ optpkg.Progress) {
			atomic.AddInt64(&progressCount, 1)
		},
		OnCompleteFunc: func(_ context.Context, _ optpkg.Result) {
			atomic.AddInt64(&completeCount, 1)
		},
	}

	agt := &echoAgent{id: "callback-test"}
	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

	_, err := compiler.Compile(context.Background(), agt,
		optpkg.WithMetric(containsMetric),
		optpkg.WithTrainsetExamples(makeExamples(4)),
		optpkg.WithMaxIterations(2),
		optpkg.WithCallback(cb),
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if progressCount == 0 {
		t.Error("OnProgress was never called")
	}
	if completeCount != 1 {
		t.Errorf("OnComplete called %d times, want 1", completeCount)
	}
}

// ── Error cases ───────────────────────────────────────────────────────────────

func TestIntegration_MissingMetric(t *testing.T) {
	agt := &echoAgent{id: "no-metric"}
	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

	_, err := compiler.Compile(context.Background(), agt,
		optpkg.WithTrainsetExamples(makeExamples(4)),
		// No WithMetric — should fail
	)
	if err == nil {
		t.Fatal("expected error when metric is not set")
	}
}

func TestIntegration_MissingTrainset(t *testing.T) {
	agt := &echoAgent{id: "no-trainset"}
	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

	_, err := compiler.Compile(context.Background(), agt,
		optpkg.WithMetric(containsMetric),
		// No WithTrainsetExamples — should fail
	)
	if err == nil {
		t.Fatal("expected error when trainset is not set")
	}
}

// ── Budget enforcement ────────────────────────────────────────────────────────

func TestIntegration_BudgetMaxIterations(t *testing.T) {
	agt := &echoAgent{id: "budget-test"}
	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

	result, err := compiler.CompileWithResult(context.Background(), agt,
		optpkg.WithMetric(containsMetric),
		optpkg.WithTrainsetExamples(makeExamples(20)),
		optpkg.WithMaxIterations(3), // hard cap at 3
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(result.Trials) > 3 {
		t.Errorf("expected <= 3 trials, got %d", len(result.Trials))
	}
}

// ── Race safety ───────────────────────────────────────────────────────────────

// TestIntegration_ConcurrentCompile verifies that running multiple compilations
// concurrently (as might happen in a test suite or server) does not race.
func TestIntegration_ConcurrentCompile(t *testing.T) {
	trainset := makeExamples(5)
	strategies := []optpkg.OptimizationStrategy{
		optpkg.StrategyBootstrapFewShot,
		optpkg.StrategyMIPROv2,
	}

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			strategy := strategies[i%len(strategies)]
			agt := &echoAgent{id: fmt.Sprintf("concurrent-%d", i)}
			compiler := optpkg.CompilerForStrategy(strategy)
			_, err := compiler.Compile(context.Background(), agt,
				optpkg.WithMetric(containsMetric),
				optpkg.WithTrainsetExamples(trainset),
				optpkg.WithMaxIterations(2),
				optpkg.WithSeed(int64(i)),
			)
			if err != nil {
				t.Errorf("goroutine %d: %v", i, err)
			}
		}()
	}
	wg.Wait()
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestIntegration_RegistryRoundTrip(t *testing.T) {
	// Ensure that strategies registered via bridge.go's init() are reachable.
	expected := []string{
		"bootstrap_few_shot",
		"mipro_v2",
		"gepa",
		"simba",
	}

	compilers := optpkg.ListCompilers()
	compilerSet := make(map[string]struct{}, len(compilers))
	for _, c := range compilers {
		compilerSet[c] = struct{}{}
	}

	for _, name := range expected {
		if _, ok := compilerSet[name]; !ok {
			t.Errorf("compiler %q not found in registry (available: %v)", name, compilers)
		}
	}

	optimizers := optpkg.ListOptimizers()
	optimizerSet := make(map[string]struct{}, len(optimizers))
	for _, o := range optimizers {
		optimizerSet[o] = struct{}{}
	}

	for _, name := range expected {
		if _, ok := optimizerSet[name]; !ok {
			t.Errorf("optimizer %q not found in registry (available: %v)", name, optimizers)
		}
	}
}

// TestIntegration_NewCompilerByName tests creating compilers by string name.
func TestIntegration_NewCompilerByName(t *testing.T) {
	names := []string{"bootstrap_few_shot", "mipro_v2", "gepa", "simba"}
	trainset := makeExamples(4)

	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			c, err := optpkg.NewCompiler(name, optpkg.CompilerConfig{})
			if err != nil {
				t.Fatalf("NewCompiler(%q): %v", name, err)
			}

			agt := &echoAgent{id: "named-" + name}
			_, err = c.Compile(context.Background(), agt,
				optpkg.WithMetric(containsMetric),
				optpkg.WithTrainsetExamples(trainset),
				optpkg.WithMaxIterations(2),
			)
			if err != nil {
				t.Fatalf("Compile via named compiler %q: %v", name, err)
			}
		})
	}
}

// ── Determinism ───────────────────────────────────────────────────────────────

func TestIntegration_Deterministic_BootstrapFewShot(t *testing.T) {
	trainset := makeExamples(8)

	compile := func() optpkg.Result {
		agt := &echoAgent{id: "det-test"}
		compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)
		result, err := compiler.CompileWithResult(context.Background(), agt,
			optpkg.WithMetric(containsMetric),
			optpkg.WithTrainsetExamples(trainset),
			optpkg.WithMaxIterations(5),
			optpkg.WithSeed(99),
		)
		if err != nil {
			t.Fatalf("determinism compile: %v", err)
		}
		return *result
	}

	r1 := compile()
	r2 := compile()

	if r1.Agent == nil || r2.Agent == nil {
		t.Fatal("expected non-nil agents")
	}
	// Both runs should produce the same number of trials.
	if len(r1.Trials) != len(r2.Trials) {
		t.Errorf("trial count differs: %d vs %d", len(r1.Trials), len(r2.Trials))
	}
}

// ── ValSet ────────────────────────────────────────────────────────────────────

func TestIntegration_Valset_Passed(t *testing.T) {
	trainset := makeExamples(8)
	valset := makeExamples(3)

	agt := &echoAgent{id: "valset-test"}
	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

	_, err := compiler.Compile(context.Background(), agt,
		optpkg.WithMetric(containsMetric),
		optpkg.WithTrainsetExamples(trainset),
		optpkg.WithValsetExamples(valset),
		optpkg.WithMaxIterations(3),
	)
	if err != nil {
		t.Fatalf("Compile with valset: %v", err)
	}
}

// ── Optimized agent is usable ─────────────────────────────────────────────────

func TestIntegration_OptimizedAgent_Invoke(t *testing.T) {
	trainset := makeExamples(6)

	agt := &echoAgent{id: "usable-test"}
	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

	optimized, err := compiler.Compile(context.Background(), agt,
		optpkg.WithMetric(containsMetric),
		optpkg.WithTrainsetExamples(trainset),
		optpkg.WithSeed(42),
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	// The optimized agent must respond to Invoke without error.
	result, err := optimized.Invoke(context.Background(), "a-question-0")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result from optimized agent")
	}
}

func TestIntegration_OptimizedAgent_Stream(t *testing.T) {
	trainset := makeExamples(4)

	agt := &echoAgent{id: "stream-test"}
	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

	optimized, err := compiler.Compile(context.Background(), agt,
		optpkg.WithMetric(containsMetric),
		optpkg.WithTrainsetExamples(trainset),
		optpkg.WithSeed(42),
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	var eventCount int
	for _, err := range optimized.Stream(context.Background(), "a-question-0") {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		eventCount++
	}
	if eventCount == 0 {
		t.Error("expected at least one event from stream")
	}
}
