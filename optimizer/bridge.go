package optimizer

// bridge.go wires the high-level optimizer.Compiler to the low-level
// optimize/ DSPy optimizers (bootstrapfewshot, mipro, gepa, simba).
//
// The four bridges are registered into the optimizer registry so callers can
// use them by strategy name via NewCompiler / CompilerForStrategy without
// needing to know about the underlying optimize package.
//
// Import side-effects:
//
//	import _ "github.com/lookatitude/beluga-ai/optimize/optimizers"
//
// registers the four optimize.Optimizer implementations; the bridge then
// looks them up by name.

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/tool"
)

func init() {
	// Register bridges for the four optimizers as both Compiler and Optimizer entries.
	for _, mapping := range []struct {
		strategy   OptimizationStrategy
		optimizeID string
	}{
		{StrategyBootstrapFewShot, "bootstrapfewshot"},
		{StrategyMIPROv2, "mipro"},
		{StrategyGEPA, "gepa"},
		{StrategySIMBA, "simba"},
	} {
		m := mapping // capture loop variable
		RegisterOptimizer(string(m.strategy), func(cfg OptimizerConfig) (Optimizer, error) {
			return &bridgeOptimizer{
				strategy:   m.strategy,
				optimizeID: m.optimizeID,
			}, nil
		})
		RegisterCompiler(string(m.strategy), func(cfg CompilerConfig) (Compiler, error) {
			return &baseCompiler{
				BaseCompiler: NewBaseCompiler(m.strategy),
				optimizer: &bridgeOptimizer{
					strategy:   m.strategy,
					optimizeID: m.optimizeID,
				},
			}, nil
		})
	}
}

// bridgeOptimizer adapts optimizer.Optimizer to optimize.Optimizer by:
//  1. Wrapping agent.Agent as optimize.Program via optimize.AgentProgram.
//  2. Looking up the real optimize.Optimizer from the optimize registry.
//  3. Running Compile and mapping results back to optimizer.Result.
//
// This keeps optimizer/ free from direct imports of optimize/optimizers and
// avoids circular dependencies.
type bridgeOptimizer struct {
	strategy   OptimizationStrategy
	optimizeID string
}

// Optimize implements optimizer.Optimizer.
func (b *bridgeOptimizer) Optimize(ctx context.Context, agt agent.Agent, opts OptimizeOptions) (*Result, error) {
	// 1. Resolve the underlying optimize.Optimizer from the registry.
	//    The optimize/optimizers package must be blank-imported by the caller
	//    to populate the registry (e.g., _ "github.com/lookatitude/beluga-ai/optimize/optimizers").
	underlying, err := optimize.NewOptimizer(b.optimizeID, optimize.OptimizerConfig{})
	if err != nil {
		return nil, fmt.Errorf("bridge(%s): optimizer %q not found — did you blank-import optimize/optimizers? %w",
			b.strategy, b.optimizeID, err)
	}

	// 2. Wrap the agent as an optimize.Program.
	program := newBridgeProgram(agt)

	// 3. Map optimizer.Dataset → []optimize.Example.
	trainset := datasetToExamples(opts.Trainset)
	valset := datasetToExamples(opts.Valset)

	// 4. Map optimizer.Metric → optimize.Metric.
	metric := bridgeMetric(opts.Metric)
	if metric == nil {
		return nil, fmt.Errorf("bridge(%s): metric is required", b.strategy)
	}

	// 5. Map optimizer.Budget → *optimize.CostBudget.
	var maxCost *optimize.CostBudget
	if opts.Budget.MaxIterations > 0 || opts.Budget.MaxCost > 0 {
		maxCost = &optimize.CostBudget{
			MaxIterations: opts.Budget.MaxIterations,
			MaxDollars:    opts.Budget.MaxCost,
		}
	}

	// 6. Bridge callbacks.
	var callbacks []optimize.Callback
	for _, cb := range opts.Callbacks {
		callbacks = append(callbacks, &bridgeCallback{underlying: cb})
	}

	// 7. Run compilation.
	start := time.Now()
	compiled, err := underlying.Compile(ctx, program, optimize.CompileOptions{
		Trainset:   trainset,
		Valset:     valset,
		Metric:     metric,
		MaxCost:    maxCost,
		Callbacks:  callbacks,
		NumWorkers: opts.NumWorkers,
		Seed:       opts.Seed,
	})
	if err != nil {
		return nil, fmt.Errorf("bridge(%s): compile failed: %w", b.strategy, err)
	}

	// 8. Wrap the compiled program back as an agent.
	optimizedAgent := &compiledAgent{
		base:    agt,
		program: compiled,
	}

	return &Result{
		Agent:             optimizedAgent,
		Score:             0.0, // score is surfaced via callbacks; not returned by optimize.Compile
		Trials:            nil,
		TotalDuration:     time.Since(start),
		ConvergenceStatus: ConvergenceNotReached,
	}, nil
}

// ─── bridgeProgram ─────────────────────────────────────────────────────────

// bridgeProgram wraps an agent.Agent as optimize.Program.
type bridgeProgram struct {
	agt   agent.Agent
	demos []optimize.Example
}

func newBridgeProgram(agt agent.Agent) *bridgeProgram {
	return &bridgeProgram{agt: agt}
}

// Run implements optimize.Program.
func (p *bridgeProgram) Run(ctx context.Context, inputs map[string]interface{}) (optimize.Prediction, error) {
	inputStr, _ := inputs["input"].(string)
	if inputStr == "" {
		for k, v := range inputs {
			if s, ok := v.(string); ok {
				inputStr = fmt.Sprintf("%s: %s", k, s)
			}
		}
	}

	result, err := p.agt.Invoke(ctx, inputStr)
	if err != nil {
		return optimize.Prediction{}, err
	}
	return optimize.Prediction{
		Outputs: map[string]interface{}{"output": result},
		Raw:     result,
	}, nil
}

// WithDemos implements optimize.Program.
func (p *bridgeProgram) WithDemos(demos []optimize.Example) optimize.Program {
	return &bridgeProgram{agt: p.agt, demos: demos}
}

// GetSignature implements optimize.Program.
func (p *bridgeProgram) GetSignature() optimize.Signature { return nil }

// ─── compiledAgent ──────────────────────────────────────────────────────────

// compiledAgent wraps an optimized optimize.Program as an agent.Agent.
type compiledAgent struct {
	base    agent.Agent
	program optimize.Program
}

func (ca *compiledAgent) ID() string              { return ca.base.ID() + "-compiled" }
func (ca *compiledAgent) Persona() agent.Persona  { return ca.base.Persona() }
func (ca *compiledAgent) Tools() []tool.Tool      { return ca.base.Tools() }
func (ca *compiledAgent) Children() []agent.Agent { return ca.base.Children() }

func (ca *compiledAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	pred, err := ca.program.Run(ctx, map[string]interface{}{"input": input})
	if err != nil {
		return "", err
	}
	if out, ok := pred.Outputs["output"].(string); ok {
		return out, nil
	}
	return pred.Raw, nil
}

func (ca *compiledAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, err := ca.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{Type: agent.EventError, AgentID: ca.ID()}, err)
			return
		}
		if !yield(agent.Event{Type: agent.EventText, Text: result, AgentID: ca.ID()}, nil) {
			return
		}
		yield(agent.Event{Type: agent.EventDone, AgentID: ca.ID()}, nil)
	}
}

// ─── bridgeMetric ───────────────────────────────────────────────────────────

// bridgeMetric wraps optimizer.Metric as optimize.Metric.
type bridgeMetricWrapper struct {
	m   Metric
	ctx context.Context
}

func bridgeMetric(m Metric) optimize.Metric {
	if m == nil {
		return nil
	}
	return &bridgeMetricAdapter{m: m}
}

type bridgeMetricAdapter struct {
	m Metric
}

func (a *bridgeMetricAdapter) Evaluate(ex optimize.Example, pred optimize.Prediction, _ *optimize.Trace) (float64, error) {
	opEx := Example{
		Inputs:   ex.Inputs,
		Outputs:  ex.Outputs,
		Metadata: ex.Metadata,
	}
	opPred := Prediction{
		Text:     pred.Raw,
		Outputs:  pred.Outputs,
		Metadata: nil,
	}
	return a.m.Score(context.Background(), opEx, opPred)
}

// ─── bridgeCallback ─────────────────────────────────────────────────────────

// bridgeCallback adapts optimizer.Callback to optimize.Callback.
type bridgeCallback struct {
	underlying Callback
}

func (bc *bridgeCallback) OnTrialComplete(t optimize.Trial) {
	bc.underlying.OnTrialComplete(context.Background(), Trial{
		ID:    t.ID,
		Score: t.Score,
		Cost:  t.Cost,
		Error: t.Error,
	})
}

func (bc *bridgeCallback) OnOptimizationComplete(r optimize.OptimizationResult) {
	bc.underlying.OnComplete(context.Background(), Result{
		Score:             r.BestScore,
		TotalCost:         r.TotalCost,
		ConvergenceStatus: ConvergenceNotReached,
	})
}

// ─── helpers ────────────────────────────────────────────────────────────────

// datasetToExamples converts optimizer.Dataset to []optimize.Example.
func datasetToExamples(d Dataset) []optimize.Example {
	examples := make([]optimize.Example, len(d.Examples))
	for i, ex := range d.Examples {
		examples[i] = optimize.Example{
			Inputs:   ex.Inputs,
			Outputs:  ex.Outputs,
			Metadata: ex.Metadata,
		}
	}
	return examples
}
