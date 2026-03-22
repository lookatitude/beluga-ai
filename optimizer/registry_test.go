package optimizer

import (
	"context"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
)

func TestOptimizerRegistry_RegisterAndNew(t *testing.T) {
	name := "test_optimizer_" + t.Name()
	RegisterOptimizer(name, func(_ OptimizerConfig) (Optimizer, error) {
		return &simpleOptimizer{strategy: StrategyBootstrapFewShot}, nil
	})

	opt, err := NewOptimizer(name, OptimizerConfig{})
	if err != nil {
		t.Fatalf("NewOptimizer: %v", err)
	}
	if opt == nil {
		t.Fatal("NewOptimizer returned nil")
	}
}

func TestNewOptimizer_NotRegistered(t *testing.T) {
	_, err := NewOptimizer("nonexistent_optimizer", OptimizerConfig{})
	if err == nil {
		t.Fatal("expected error for unregistered optimizer")
	}
}

func TestMustOptimizer_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unregistered optimizer")
		}
	}()
	MustOptimizer("nonexistent_must_optimizer", OptimizerConfig{})
}

func TestListOptimizers_Sorted(t *testing.T) {
	RegisterOptimizer("zzz_test_opt", func(_ OptimizerConfig) (Optimizer, error) {
		return &simpleOptimizer{}, nil
	})
	RegisterOptimizer("aaa_test_opt", func(_ OptimizerConfig) (Optimizer, error) {
		return &simpleOptimizer{}, nil
	})

	list := ListOptimizers()
	for i := 1; i < len(list); i++ {
		if list[i] < list[i-1] {
			t.Errorf("list not sorted: %v", list)
			break
		}
	}
}

func TestListOptimizers_BuiltInRegistered(t *testing.T) {
	list := ListOptimizers()
	found := false
	for _, name := range list {
		if name == string(StrategyBootstrapFewShot) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("built-in bootstrap_few_shot optimizer not found in: %v", list)
	}
}

func TestSimpleOptimizer_Optimize(t *testing.T) {
	opt := &simpleOptimizer{strategy: StrategyBootstrapFewShot}
	agt := &mockAgent{id: "test"}

	result, err := opt.Optimize(context.Background(), agt, OptimizeOptions{
		Metric:   &ExactMatchMetric{},
		Trainset: Dataset{Examples: []Example{{Inputs: map[string]any{"q": "hello"}}}},
	})
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

func TestOptimizerRegistry_ConcurrentAccess(t *testing.T) {
	// Verify race-free concurrent reads and writes.
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				RegisterOptimizer("concurrent_"+string(rune('A'+i)), func(_ OptimizerConfig) (Optimizer, error) {
					return &simpleOptimizer{}, nil
				})
			} else {
				_ = ListOptimizers()
			}
		}(i)
	}
	wg.Wait()
}

func TestCompilerRegistry_ConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				RegisterCompiler("concurrent_comp_"+string(rune('A'+i)), func(_ CompilerConfig) (Compiler, error) {
					return DefaultCompiler(), nil
				})
			} else {
				_ = ListCompilers()
			}
		}(i)
	}
	wg.Wait()
}

func TestMetricRegistry_ConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				RegisterMetric("concurrent_metric_"+string(rune('A'+i)), func(_ MetricConfig) (Metric, error) {
					return &ExactMatchMetric{}, nil
				})
			} else {
				_ = ListMetrics()
			}
		}(i)
	}
	wg.Wait()
}

// Verify that the Optimizer interface is satisfied by simpleOptimizer.
var _ Optimizer = (*simpleOptimizer)(nil)

// Verify that the Compiler interface is satisfied.
var _ Compiler = (*baseCompiler)(nil)

// Verify that the Metric interface is satisfied.
var _ Metric = (*ExactMatchMetric)(nil)
var _ Metric = (*ContainsMetric)(nil)
var _ Metric = MetricFunc(nil)

// Verify that agent.Agent interface is satisfied by mockAgent.
var _ agent.Agent = (*mockAgent)(nil)
