package optimizer

import (
	"testing"
	"time"
)

func TestDefaultCompileConfig(t *testing.T) {
	cfg := defaultCompileConfig()

	if cfg.strategy != StrategyBootstrapFewShot {
		t.Errorf("default strategy: got %q, want %q", cfg.strategy, StrategyBootstrapFewShot)
	}
	if cfg.numWorkers != 10 {
		t.Errorf("default numWorkers: got %d, want 10", cfg.numWorkers)
	}
	if cfg.timeout != 30*time.Minute {
		t.Errorf("default timeout: got %v, want 30m", cfg.timeout)
	}
	if cfg.budget.MaxIterations != 100 {
		t.Errorf("default MaxIterations: got %d, want 100", cfg.budget.MaxIterations)
	}
	if cfg.seed == 0 {
		t.Error("default seed should be non-zero")
	}
}

func TestWithStrategy(t *testing.T) {
	cfg := applyCompileOptions(WithStrategy(StrategyGEPA))
	if cfg.strategy != StrategyGEPA {
		t.Errorf("got %q, want %q", cfg.strategy, StrategyGEPA)
	}
}

func TestWithMetric(t *testing.T) {
	m := &ExactMatchMetric{}
	cfg := applyCompileOptions(WithMetric(m))
	if cfg.metric == nil {
		t.Fatal("metric should not be nil")
	}
}

func TestWithTrainset(t *testing.T) {
	ds := Dataset{Examples: []Example{{Inputs: map[string]any{"q": "hello"}}}}
	cfg := applyCompileOptions(WithTrainset(ds))
	if len(cfg.trainset.Examples) != 1 {
		t.Errorf("got %d examples, want 1", len(cfg.trainset.Examples))
	}
}

func TestWithTrainsetExamples(t *testing.T) {
	examples := []Example{{Inputs: map[string]any{"a": "1"}}, {Inputs: map[string]any{"b": "2"}}}
	cfg := applyCompileOptions(WithTrainsetExamples(examples))
	if len(cfg.trainset.Examples) != 2 {
		t.Errorf("got %d examples, want 2", len(cfg.trainset.Examples))
	}
}

func TestWithValset(t *testing.T) {
	ds := Dataset{Examples: []Example{{Inputs: map[string]any{"q": "test"}}}}
	cfg := applyCompileOptions(WithValset(ds))
	if len(cfg.valset.Examples) != 1 {
		t.Errorf("got %d examples, want 1", len(cfg.valset.Examples))
	}
}

func TestWithValsetExamples(t *testing.T) {
	examples := []Example{{Inputs: map[string]any{"a": "1"}}}
	cfg := applyCompileOptions(WithValsetExamples(examples))
	if len(cfg.valset.Examples) != 1 {
		t.Errorf("got %d examples, want 1", len(cfg.valset.Examples))
	}
}

func TestWithBudget(t *testing.T) {
	b := Budget{MaxCost: 50.0, MaxCalls: 500}
	cfg := applyCompileOptions(WithBudget(b))
	if cfg.budget.MaxCost != 50.0 {
		t.Errorf("MaxCost: got %v, want 50.0", cfg.budget.MaxCost)
	}
	if cfg.budget.MaxCalls != 500 {
		t.Errorf("MaxCalls: got %d, want 500", cfg.budget.MaxCalls)
	}
}

func TestWithMaxIterations(t *testing.T) {
	cfg := applyCompileOptions(WithMaxIterations(200))
	if cfg.budget.MaxIterations != 200 {
		t.Errorf("got %d, want 200", cfg.budget.MaxIterations)
	}
}

func TestWithMaxCost(t *testing.T) {
	cfg := applyCompileOptions(WithMaxCost(25.0))
	if cfg.budget.MaxCost != 25.0 {
		t.Errorf("got %v, want 25.0", cfg.budget.MaxCost)
	}
}

func TestWithMaxDuration(t *testing.T) {
	cfg := applyCompileOptions(WithMaxDuration(10 * time.Minute))
	if cfg.budget.MaxDuration != 10*time.Minute {
		t.Errorf("got %v, want 10m", cfg.budget.MaxDuration)
	}
}

func TestWithMaxCalls(t *testing.T) {
	cfg := applyCompileOptions(WithMaxCalls(1000))
	if cfg.budget.MaxCalls != 1000 {
		t.Errorf("got %d, want 1000", cfg.budget.MaxCalls)
	}
}

func TestWithNumWorkers(t *testing.T) {
	cfg := applyCompileOptions(WithNumWorkers(5))
	if cfg.numWorkers != 5 {
		t.Errorf("got %d, want 5", cfg.numWorkers)
	}
}

func TestWithNumWorkers_IgnoresZero(t *testing.T) {
	cfg := applyCompileOptions(WithNumWorkers(0))
	if cfg.numWorkers != 10 {
		t.Errorf("got %d, want 10 (default)", cfg.numWorkers)
	}
}

func TestWithSeed(t *testing.T) {
	cfg := applyCompileOptions(WithSeed(42))
	if cfg.seed != 42 {
		t.Errorf("got %d, want 42", cfg.seed)
	}
}

func TestWithTimeout(t *testing.T) {
	cfg := applyCompileOptions(WithTimeout(5 * time.Minute))
	if cfg.timeout != 5*time.Minute {
		t.Errorf("got %v, want 5m", cfg.timeout)
	}
}

func TestWithTimeout_IgnoresZero(t *testing.T) {
	cfg := applyCompileOptions(WithTimeout(0))
	if cfg.timeout != 30*time.Minute {
		t.Errorf("got %v, want 30m (default)", cfg.timeout)
	}
}

func TestWithCallback(t *testing.T) {
	cb := CallbackFunc{}
	cfg := applyCompileOptions(WithCallback(cb))
	if len(cfg.callbacks) != 1 {
		t.Errorf("got %d callbacks, want 1", len(cfg.callbacks))
	}
}

func TestWithCallbacks(t *testing.T) {
	cb1 := CallbackFunc{}
	cb2 := CallbackFunc{}
	cfg := applyCompileOptions(WithCallbacks(cb1, cb2))
	if len(cfg.callbacks) != 2 {
		t.Errorf("got %d callbacks, want 2", len(cfg.callbacks))
	}
}

func TestWithExtra(t *testing.T) {
	cfg := applyCompileOptions(WithExtra("key1", "val1"), WithExtra("key2", 42))
	if cfg.extra["key1"] != "val1" {
		t.Errorf("key1: got %v, want val1", cfg.extra["key1"])
	}
	if cfg.extra["key2"] != 42 {
		t.Errorf("key2: got %v, want 42", cfg.extra["key2"])
	}
}

func TestMultipleOptions(t *testing.T) {
	cfg := applyCompileOptions(
		WithStrategy(StrategySIMBA),
		WithMaxIterations(50),
		WithMaxCost(10.0),
		WithSeed(123),
		WithNumWorkers(4),
	)

	if cfg.strategy != StrategySIMBA {
		t.Errorf("strategy: got %q, want %q", cfg.strategy, StrategySIMBA)
	}
	if cfg.budget.MaxIterations != 50 {
		t.Errorf("MaxIterations: got %d, want 50", cfg.budget.MaxIterations)
	}
	if cfg.budget.MaxCost != 10.0 {
		t.Errorf("MaxCost: got %v, want 10.0", cfg.budget.MaxCost)
	}
	if cfg.seed != 123 {
		t.Errorf("seed: got %d, want 123", cfg.seed)
	}
	if cfg.numWorkers != 4 {
		t.Errorf("numWorkers: got %d, want 4", cfg.numWorkers)
	}
}

func TestDefaultOptimizationConfig(t *testing.T) {
	cfg := DefaultOptimizationConfig()
	if cfg.Strategy != StrategyBootstrapFewShot {
		t.Errorf("default strategy: got %q, want %q", cfg.Strategy, StrategyBootstrapFewShot)
	}
	if cfg.NumWorkers != 10 {
		t.Errorf("default NumWorkers: got %d, want 10", cfg.NumWorkers)
	}
	if cfg.Budget.MaxIterations != 100 {
		t.Errorf("default MaxIterations: got %d, want 100", cfg.Budget.MaxIterations)
	}
}

func TestOptimizationConfig_ToCompileOptions(t *testing.T) {
	m := &ExactMatchMetric{}
	cfg := OptimizationConfig{
		Strategy:   StrategyMIPROv2,
		Metric:     m,
		NumWorkers: 8,
		Seed:       42,
		Timeout:    10 * time.Minute,
		Budget:     Budget{MaxIterations: 50},
		Extra:      map[string]any{"k": "v"},
	}

	train := Dataset{Examples: []Example{{Inputs: map[string]any{"q": "1"}}}}
	val := Dataset{Examples: []Example{{Inputs: map[string]any{"q": "2"}}}}

	opts := cfg.ToCompileOptions(train, val)
	result := applyCompileOptions(opts...)

	if result.strategy != StrategyMIPROv2 {
		t.Errorf("strategy: got %q, want %q", result.strategy, StrategyMIPROv2)
	}
	if result.numWorkers != 8 {
		t.Errorf("numWorkers: got %d, want 8", result.numWorkers)
	}
	if result.seed != 42 {
		t.Errorf("seed: got %d, want 42", result.seed)
	}
	if len(result.trainset.Examples) != 1 {
		t.Errorf("trainset: got %d examples, want 1", len(result.trainset.Examples))
	}
	if len(result.valset.Examples) != 1 {
		t.Errorf("valset: got %d examples, want 1", len(result.valset.Examples))
	}
	if result.metric == nil {
		t.Error("metric should not be nil")
	}
	if result.extra["k"] != "v" {
		t.Errorf("extra[k]: got %v, want v", result.extra["k"])
	}
}
