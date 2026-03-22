package optimizer

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
)

// Compiler orchestrates the optimization of agents using various strategies.
// It implements DSPy-style compilation, transforming an unoptimized agent into
// an optimized version through systematic exploration of the prompt/agent space.
type Compiler interface {
	// Compile optimizes the given agent using the provided options.
	// Returns an optimized agent that should achieve better performance
	// on the target metric.
	Compile(ctx context.Context, agt agent.Agent, opts ...CompileOption) (agent.Agent, error)

	// CompileWithResult optimizes the agent and returns detailed results.
	// This provides access to the full optimization history.
	CompileWithResult(ctx context.Context, agt agent.Agent, opts ...CompileOption) (*Result, error)
}

// BaseCompiler provides a foundation for compiler implementations.
// It handles common concerns like configuration management and callbacks.
type BaseCompiler struct {
	strategy   OptimizationStrategy
	numWorkers int
	seed       int64
}

// NewBaseCompiler creates a new base compiler with the given strategy.
func NewBaseCompiler(strategy OptimizationStrategy) *BaseCompiler {
	cfg := defaultCompileConfig()
	return &BaseCompiler{
		strategy:   strategy,
		numWorkers: cfg.numWorkers,
		seed:       cfg.seed,
	}
}

// Strategy returns the compiler's optimization strategy.
func (bc *BaseCompiler) Strategy() OptimizationStrategy {
	return bc.strategy
}

// baseCompiler is the default compiler implementation.
type baseCompiler struct {
	*BaseCompiler
	optimizer Optimizer
}

// Compile implements Compiler.
func (c *baseCompiler) Compile(ctx context.Context, agt agent.Agent, opts ...CompileOption) (agent.Agent, error) {
	result, err := c.CompileWithResult(ctx, agt, opts...)
	if err != nil {
		return nil, err
	}
	return result.Agent, nil
}

// CompileWithResult implements Compiler.
func (c *baseCompiler) CompileWithResult(ctx context.Context, agt agent.Agent, opts ...CompileOption) (*Result, error) {
	cfg := applyCompileOptions(opts...)

	if cfg.metric == nil {
		return nil, fmt.Errorf("metric is required for compilation")
	}

	if len(cfg.trainset.Examples) == 0 {
		return nil, fmt.Errorf("trainset is required for compilation")
	}

	// Initialize progress
	progress := Progress{
		Phase:        PhaseInitializing,
		CurrentTrial: 0,
		TotalTrials:  cfg.budget.MaxIterations,
		StartTime:    now(),
	}

	// Notify callbacks of initialization
	for _, cb := range cfg.callbacks {
		cb.OnProgress(ctx, progress)
	}

	// Create the optimizer instance.
	optimizer := c.optimizer
	if optimizer == nil {
		// Resolve from the optimizer registry. Prefer the compiler's strategy
		// (set via CompilerForStrategy) over the config default, but allow
		// an explicit WithStrategy option to override both.
		strategy := c.strategy
		if cfg.strategy != "" {
			strategy = cfg.strategy
		}
		var err error
		optimizer, err = NewOptimizer(string(strategy), OptimizerConfig{
			Seed:  cfg.seed,
			Extra: cfg.extra,
		})
		if err != nil {
			return nil, fmt.Errorf("create optimizer: %w", err)
		}
	}

	// Execute optimization
	progress.Phase = PhaseTraining
	result, err := optimizer.Optimize(ctx, agt, OptimizeOptions{
		Metric:     cfg.metric,
		Trainset:   cfg.trainset,
		Valset:     cfg.valset,
		Budget:     cfg.budget,
		Callbacks:  cfg.callbacks,
		NumWorkers: cfg.numWorkers,
		Seed:       cfg.seed,
	})

	if err != nil {
		progress.Phase = PhaseError
		for _, cb := range cfg.callbacks {
			cb.OnProgress(ctx, progress)
		}
		return nil, fmt.Errorf("optimization failed: %w", err)
	}

	progress.Phase = PhaseComplete
	progress.CurrentScore = result.Score
	for _, cb := range cfg.callbacks {
		cb.OnProgress(ctx, progress)
		cb.OnComplete(ctx, *result)
	}

	return result, nil
}

// Optimizer is the internal interface for optimization algorithms.
type Optimizer interface {
	// Optimize runs the optimization process.
	Optimize(ctx context.Context, agt agent.Agent, opts OptimizeOptions) (*Result, error)
}

// OptimizeOptions contains options for the optimization process.
type OptimizeOptions struct {
	Metric     Metric
	Trainset   Dataset
	Valset     Dataset
	Budget     Budget
	Callbacks  []Callback
	NumWorkers int
	Seed       int64
}

// simpleOptimizer is a basic optimizer implementation.
type simpleOptimizer struct {
	strategy OptimizationStrategy
}

// Optimize implements Optimizer.
func (o *simpleOptimizer) Optimize(ctx context.Context, agt agent.Agent, opts OptimizeOptions) (*Result, error) {
	// This is a minimal implementation that returns the original agent
	// with a default score. Real implementations would explore configurations.

	result := &Result{
		Agent:             agt,
		Score:             0.0,
		Trials:            []Trial{},
		TotalCost:         0.0,
		TotalDuration:     0,
		ConvergenceStatus: ConvergenceNotReached,
	}

	return result, nil
}

// CompilerFactory creates a Compiler from configuration.
type CompilerFactory func(cfg CompilerConfig) (Compiler, error)

var (
	compilerMu       sync.RWMutex
	compilerRegistry = make(map[string]CompilerFactory)
)

// RegisterCompiler registers a compiler factory under the given name.
// This is typically called from init() in compiler implementation files.
func RegisterCompiler(name string, factory CompilerFactory) {
	compilerMu.Lock()
	defer compilerMu.Unlock()
	compilerRegistry[name] = factory
}

// NewCompiler creates a new compiler by name from the registry.
func NewCompiler(name string, cfg CompilerConfig) (Compiler, error) {
	compilerMu.RLock()
	factory, ok := compilerRegistry[name]
	compilerMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("compiler %q not registered (available: %v)", name, ListCompilers())
	}
	return factory(cfg)
}

// MustCompiler creates a compiler or panics if registration fails.
func MustCompiler(name string, cfg CompilerConfig) Compiler {
	c, err := NewCompiler(name, cfg)
	if err != nil {
		panic(err)
	}
	return c
}

// ListCompilers returns the sorted names of all registered compilers.
func ListCompilers() []string {
	compilerMu.RLock()
	defer compilerMu.RUnlock()

	names := make([]string, 0, len(compilerRegistry))
	for name := range compilerRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultCompiler returns a default compiler instance using BootstrapFewShot.
// The compiler resolves its optimizer from the registry at compile time,
// supporting both direct implementations and bridge-based optimizers.
func DefaultCompiler() Compiler {
	return CompilerForStrategy(StrategyBootstrapFewShot)
}

// CompilerForStrategy returns a compiler configured for the given strategy.
// The optimizer is resolved lazily from the registry at compile time, so
// callers must ensure the desired optimizer is registered (e.g. by importing
// optimize/optimizers for bridge-based access to all four algorithms).
func CompilerForStrategy(strategy OptimizationStrategy) Compiler {
	return &baseCompiler{
		BaseCompiler: NewBaseCompiler(strategy),
	}
}

// now is a variable to allow mocking in tests.
var now = func() time.Time {
	return time.Now()
}
