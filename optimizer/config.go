package optimizer

import (
	"time"

	"github.com/lookatitude/beluga-ai/agent"
)

// CompileOption configures the compilation process.
type CompileOption func(*compileConfig)

// compileConfig holds the configuration for compilation.
type compileConfig struct {
	// strategy is the optimization strategy to use.
	strategy OptimizationStrategy
	// metric evaluates agent performance.
	metric Metric
	// trainset contains training examples.
	trainset Dataset
	// valset contains validation examples.
	valset Dataset
	// budget limits optimization resources.
	budget Budget
	// callbacks receive progress updates.
	callbacks []Callback
	// numWorkers controls parallel trial execution.
	numWorkers int
	// seed for reproducibility.
	seed int64
	// timeout for the overall compilation.
	timeout time.Duration
	// extra holds optimizer-specific configuration.
	extra map[string]any
}

// defaultCompileConfig returns the default configuration.
func defaultCompileConfig() compileConfig {
	return compileConfig{
		strategy:   StrategyBootstrapFewShot,
		numWorkers: 10,
		seed:       time.Now().UnixNano(),
		timeout:    30 * time.Minute,
		budget: Budget{
			MaxIterations: 100,
		},
	}
}

// applyCompileOptions applies options to a config and returns it.
func applyCompileOptions(opts ...CompileOption) compileConfig {
	cfg := defaultCompileConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithStrategy sets the optimization strategy.
func WithStrategy(s OptimizationStrategy) CompileOption {
	return func(c *compileConfig) {
		c.strategy = s
	}
}

// WithMetric sets the metric for evaluating agent performance.
func WithMetric(m Metric) CompileOption {
	return func(c *compileConfig) {
		c.metric = m
	}
}

// WithTrainset sets the training dataset.
func WithTrainset(d Dataset) CompileOption {
	return func(c *compileConfig) {
		c.trainset = d
	}
}

// WithTrainsetExamples sets the training dataset from examples.
func WithTrainsetExamples(examples []Example) CompileOption {
	return func(c *compileConfig) {
		c.trainset = Dataset{Examples: examples}
	}
}

// WithValset sets the validation dataset.
func WithValset(d Dataset) CompileOption {
	return func(c *compileConfig) {
		c.valset = d
	}
}

// WithValsetExamples sets the validation dataset from examples.
func WithValsetExamples(examples []Example) CompileOption {
	return func(c *compileConfig) {
		c.valset = Dataset{Examples: examples}
	}
}

// WithBudget sets the optimization budget.
func WithBudget(b Budget) CompileOption {
	return func(c *compileConfig) {
		c.budget = b
	}
}

// WithMaxIterations sets the maximum number of iterations.
func WithMaxIterations(n int) CompileOption {
	return func(c *compileConfig) {
		c.budget.MaxIterations = n
	}
}

// WithMaxCost sets the maximum cost budget in USD.
func WithMaxCost(usd float64) CompileOption {
	return func(c *compileConfig) {
		c.budget.MaxCost = usd
	}
}

// WithMaxDuration sets the maximum duration for optimization.
func WithMaxDuration(d time.Duration) CompileOption {
	return func(c *compileConfig) {
		c.budget.MaxDuration = d
	}
}

// WithMaxCalls sets the maximum number of LLM calls.
func WithMaxCalls(n int) CompileOption {
	return func(c *compileConfig) {
		c.budget.MaxCalls = n
	}
}

// WithCallback adds a callback for progress updates.
func WithCallback(cb Callback) CompileOption {
	return func(c *compileConfig) {
		c.callbacks = append(c.callbacks, cb)
	}
}

// WithCallbacks sets all callbacks (replaces any existing).
func WithCallbacks(cbs ...Callback) CompileOption {
	return func(c *compileConfig) {
		c.callbacks = cbs
	}
}

// WithNumWorkers sets the number of parallel workers.
func WithNumWorkers(n int) CompileOption {
	return func(c *compileConfig) {
		if n > 0 {
			c.numWorkers = n
		}
	}
}

// WithSeed sets the random seed for reproducibility.
func WithSeed(seed int64) CompileOption {
	return func(c *compileConfig) {
		c.seed = seed
	}
}

// WithTimeout sets the overall compilation timeout.
func WithTimeout(t time.Duration) CompileOption {
	return func(c *compileConfig) {
		if t > 0 {
			c.timeout = t
		}
	}
}

// WithExtra sets optimizer-specific extra configuration.
func WithExtra(key string, value any) CompileOption {
	return func(c *compileConfig) {
		if c.extra == nil {
			c.extra = make(map[string]any)
		}
		c.extra[key] = value
	}
}

// OptimizationConfig is the public configuration struct for creating optimizers.
// This is used when creating compilers through the registry.
type OptimizationConfig struct {
	// Strategy is the optimization strategy to use.
	Strategy OptimizationStrategy
	// Metric is the evaluation metric.
	Metric Metric
	// Budget contains resource limits.
	Budget Budget
	// NumWorkers controls parallel execution.
	NumWorkers int
	// Seed for reproducibility.
	Seed int64
	// Timeout for the overall process.
	Timeout time.Duration
	// Extra holds optimizer-specific configuration.
	Extra map[string]any
}

// DefaultOptimizationConfig returns a default configuration.
func DefaultOptimizationConfig() OptimizationConfig {
	return OptimizationConfig{
		Strategy:   StrategyBootstrapFewShot,
		NumWorkers: 10,
		Seed:       time.Now().UnixNano(),
		Timeout:    30 * time.Minute,
		Budget: Budget{
			MaxIterations: 100,
		},
	}
}

// ToCompileOptions converts OptimizationConfig to CompileOptions.
func (cfg OptimizationConfig) ToCompileOptions(trainset, valset Dataset) []CompileOption {
	opts := []CompileOption{
		WithStrategy(cfg.Strategy),
		WithBudget(cfg.Budget),
		WithNumWorkers(cfg.NumWorkers),
		WithSeed(cfg.Seed),
		WithTimeout(cfg.Timeout),
		WithTrainset(trainset),
		WithValset(valset),
	}
	if cfg.Metric != nil {
		opts = append(opts, WithMetric(cfg.Metric))
	}
	for k, v := range cfg.Extra {
		opts = append(opts, WithExtra(k, v))
	}
	return opts
}

// CompilerConfig holds configuration for creating a compiler via the registry.
type CompilerConfig struct {
	// LLM is the language model for optimization.
	LLM agent.Agent
	// Metric is the evaluation metric.
	Metric Metric
	// Extra holds compiler-specific configuration.
	Extra map[string]any
}

// MetricConfig holds configuration for creating a metric via the registry.
type MetricConfig struct {
	// Type is the metric type name.
	Type string
	// Extra holds metric-specific configuration.
	Extra map[string]any
}
