package agent

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/tool"
)

// AgentProgram adapts an Agent to the optimize.Program interface, enabling
// DSPy-style automated optimization of agent prompts and demonstrations.
type AgentProgram struct {
	agent Agent
	demos []optimize.Example
}

// NewAgentProgram wraps an Agent as an optimize.Program.
func NewAgentProgram(agent Agent) *AgentProgram {
	return &AgentProgram{agent: agent}
}

// Run executes the agent with the given inputs and returns a Prediction.
// The "input" key is used as the agent's text input. If missing, all input
// fields are joined as a formatted string.
func (ap *AgentProgram) Run(ctx context.Context, inputs map[string]interface{}) (optimize.Prediction, error) {
	input, ok := inputs["input"].(string)
	if !ok {
		// Build input from all fields
		var parts []string
		for k, v := range inputs {
			parts = append(parts, fmt.Sprintf("%s: %v", k, v))
		}
		input = strings.Join(parts, "\n")
	}

	// Build runtime options with demos baked into persona
	var opts []Option
	if len(ap.demos) > 0 {
		opts = append(opts, withDemoPersona(ap.agent.Persona(), ap.demos))
	}

	result, err := ap.agent.Invoke(ctx, input, opts...)
	if err != nil {
		return optimize.Prediction{}, err
	}

	return optimize.Prediction{
		Outputs: map[string]interface{}{
			"output": result,
		},
		Raw: result,
	}, nil
}

// WithDemos returns a new AgentProgram with the specified demonstrations.
func (ap *AgentProgram) WithDemos(demos []optimize.Example) optimize.Program {
	return &AgentProgram{
		agent: ap.agent,
		demos: demos,
	}
}

// GetSignature derives an optimize.Signature from the agent's persona and tools.
func (ap *AgentProgram) GetSignature() optimize.Signature {
	return &agentSignature{
		persona: ap.agent.Persona(),
		tools:   ap.agent.Tools(),
		demos:   ap.demos,
	}
}

// withDemoPersona creates an Option that augments the persona with demonstrations.
func withDemoPersona(base Persona, demos []optimize.Example) Option {
	return func(c *agentConfig) {
		c.persona = personaWithDemos(base, demos)
	}
}

// personaWithDemos returns a new Persona with demonstrations appended to the backstory.
func personaWithDemos(base Persona, demos []optimize.Example) Persona {
	if len(demos) == 0 {
		return base
	}

	var sb strings.Builder
	if base.Backstory != "" {
		sb.WriteString(base.Backstory)
		sb.WriteString("\n\n")
	}
	sb.WriteString("Here are some examples of how to handle inputs:\n")
	for i, demo := range demos {
		sb.WriteString(fmt.Sprintf("\n--- Example %d ---\n", i+1))
		if input, ok := demo.Inputs["input"]; ok {
			sb.WriteString(fmt.Sprintf("Input: %v\n", input))
		}
		if output, ok := demo.Outputs["output"]; ok {
			sb.WriteString(fmt.Sprintf("Output: %v\n", output))
		}
	}

	return Persona{
		Role:      base.Role,
		Goal:      base.Goal,
		Backstory: sb.String(),
		Traits:    base.Traits,
	}
}

// agentSignature implements optimize.Signature for an agent.
type agentSignature struct {
	persona Persona
	tools   []tool.Tool
	demos   []optimize.Example
}

// Render converts inputs to a prompt string using the agent's persona context.
func (s *agentSignature) Render(inputs map[string]interface{}) (string, error) {
	var sb strings.Builder

	// Include persona as context
	if !s.persona.IsEmpty() {
		if s.persona.Role != "" {
			sb.WriteString(fmt.Sprintf("You are a %s.\n", s.persona.Role))
		}
		if s.persona.Goal != "" {
			sb.WriteString(fmt.Sprintf("Your goal is to %s.\n", s.persona.Goal))
		}
		if s.persona.Backstory != "" {
			sb.WriteString(s.persona.Backstory)
			sb.WriteString("\n")
		}
	}

	// Include demos
	if len(s.demos) > 0 {
		sb.WriteString("\nExamples:\n")
		for i, demo := range s.demos {
			sb.WriteString(fmt.Sprintf("Example %d:\n", i+1))
			if input, ok := demo.Inputs["input"]; ok {
				sb.WriteString(fmt.Sprintf("  Input: %v\n", input))
			}
			if output, ok := demo.Outputs["output"]; ok {
				sb.WriteString(fmt.Sprintf("  Output: %v\n", output))
			}
		}
	}

	// Include the actual input
	sb.WriteString("\n")
	if input, ok := inputs["input"]; ok {
		sb.WriteString(fmt.Sprintf("%v", input))
	}

	return sb.String(), nil
}

// Parse extracts outputs from the LLM response.
func (s *agentSignature) Parse(response string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"output": response,
	}, nil
}

// GetInputFields returns the input field definitions.
func (s *agentSignature) GetInputFields() []optimize.Field {
	return []optimize.Field{
		{
			Name:        "input",
			Type:        "string",
			Description: "The input text for the agent",
			Required:    true,
		},
	}
}

// GetOutputFields returns the output field definitions.
func (s *agentSignature) GetOutputFields() []optimize.Field {
	return []optimize.Field{
		{
			Name:        "output",
			Type:        "string",
			Description: "The agent's text response",
			Required:    true,
		},
	}
}

// optimizeConfig holds optimization-specific configuration.
type optimizeConfig struct {
	optimizerName string
	optimizerCfg  optimize.OptimizerConfig
	trainset      []optimize.Example
	valset        []optimize.Example
	metric        optimize.Metric
	budget        *optimize.CostBudget
	callbacks     []optimize.Callback
	numWorkers    int
	seed          int64
}

// WithOptimizer attaches an optimizer by registered name.
func WithOptimizer(name string, cfg optimize.OptimizerConfig) Option {
	return func(c *agentConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata["_optimize_name"] = name
		c.metadata["_optimize_cfg"] = cfg
	}
}

// WithTrainset sets training data for optimization.
func WithTrainset(examples []optimize.Example) Option {
	return func(c *agentConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata["_optimize_trainset"] = examples
	}
}

// WithValset sets validation data for optimization.
func WithValset(examples []optimize.Example) Option {
	return func(c *agentConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata["_optimize_valset"] = examples
	}
}

// WithMetric sets the evaluation metric for optimization.
func WithMetric(metric optimize.Metric) Option {
	return func(c *agentConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata["_optimize_metric"] = metric
	}
}

// WithOptimizationBudget sets cost limits for optimization.
func WithOptimizationBudget(budget optimize.CostBudget) Option {
	return func(c *agentConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata["_optimize_budget"] = budget
	}
}

// WithOptimizationCallbacks sets callbacks for optimization progress.
func WithOptimizationCallbacks(callbacks ...optimize.Callback) Option {
	return func(c *agentConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata["_optimize_callbacks"] = callbacks
	}
}

// WithOptimizationWorkers sets the number of parallel workers for optimization.
func WithOptimizationWorkers(n int) Option {
	return func(c *agentConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata["_optimize_workers"] = n
	}
}

// WithOptimizationSeed sets the random seed for reproducible optimization.
func WithOptimizationSeed(seed int64) Option {
	return func(c *agentConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata["_optimize_seed"] = seed
	}
}

// extractOptimizeConfig reads optimization configuration from agent metadata.
func extractOptimizeConfig(meta map[string]any) (*optimizeConfig, error) {
	if meta == nil {
		return nil, fmt.Errorf("no optimization configuration found: use WithOptimizer, WithTrainset, and WithMetric")
	}

	cfg := &optimizeConfig{}

	name, ok := meta["_optimize_name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("no optimizer specified: use WithOptimizer")
	}
	cfg.optimizerName = name

	if oc, ok := meta["_optimize_cfg"].(optimize.OptimizerConfig); ok {
		cfg.optimizerCfg = oc
	}

	trainset, ok := meta["_optimize_trainset"].([]optimize.Example)
	if !ok || len(trainset) == 0 {
		return nil, fmt.Errorf("no training data specified: use WithTrainset")
	}
	cfg.trainset = trainset

	metric, ok := meta["_optimize_metric"].(optimize.Metric)
	if !ok || metric == nil {
		return nil, fmt.Errorf("no metric specified: use WithMetric")
	}
	cfg.metric = metric

	if valset, ok := meta["_optimize_valset"].([]optimize.Example); ok {
		cfg.valset = valset
	}
	if budget, ok := meta["_optimize_budget"].(optimize.CostBudget); ok {
		cfg.budget = &budget
	}
	if callbacks, ok := meta["_optimize_callbacks"].([]optimize.Callback); ok {
		cfg.callbacks = callbacks
	}
	if workers, ok := meta["_optimize_workers"].(int); ok {
		cfg.numWorkers = workers
	}
	if seed, ok := meta["_optimize_seed"].(int64); ok {
		cfg.seed = seed
	}

	return cfg, nil
}

// Optimize runs the configured optimizer on the agent and returns an OptimizedAgent.
// The agent must be configured with WithOptimizer, WithTrainset, and WithMetric.
func (a *BaseAgent) Optimize(ctx context.Context) (*OptimizedAgent, error) {
	optCfg, err := extractOptimizeConfig(a.config.metadata)
	if err != nil {
		return nil, fmt.Errorf("agent %q: %w", a.id, err)
	}

	// Create the optimizer from the registry
	optimizer, err := optimize.NewOptimizer(optCfg.optimizerName, optCfg.optimizerCfg)
	if err != nil {
		return nil, fmt.Errorf("agent %q: create optimizer: %w", a.id, err)
	}

	// Wrap the agent as a Program
	program := NewAgentProgram(a)

	// Build compile options
	compileOpts := optimize.CompileOptions{
		Trainset:   optCfg.trainset,
		Metric:     optCfg.metric,
		Valset:     optCfg.valset,
		MaxCost:    optCfg.budget,
		Callbacks:  optCfg.callbacks,
		NumWorkers: optCfg.numWorkers,
		Seed:       optCfg.seed,
	}

	// Run optimization
	optimized, err := optimizer.Compile(ctx, program, compileOpts)
	if err != nil {
		return nil, fmt.Errorf("agent %q: optimization failed: %w", a.id, err)
	}

	return &OptimizedAgent{
		base:     a,
		program:  optimized,
		original: program,
	}, nil
}

// OptimizedAgent wraps an agent with optimized prompts and demonstrations.
// It implements the Agent interface and is a drop-in replacement for the original.
type OptimizedAgent struct {
	base     *BaseAgent
	program  optimize.Program
	original *AgentProgram
}

// ID returns the agent's identifier with an "-optimized" suffix.
func (oa *OptimizedAgent) ID() string { return oa.base.ID() + "-optimized" }

// Persona returns the original agent's persona.
func (oa *OptimizedAgent) Persona() Persona { return oa.base.Persona() }

// Tools returns the original agent's tools.
func (oa *OptimizedAgent) Tools() []tool.Tool { return oa.base.Tools() }

// Children returns the original agent's children.
func (oa *OptimizedAgent) Children() []Agent { return oa.base.Children() }

// Invoke executes the optimized agent synchronously.
func (oa *OptimizedAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	pred, err := oa.program.Run(ctx, map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return "", err
	}

	if output, ok := pred.Outputs["output"].(string); ok {
		return output, nil
	}
	return pred.Raw, nil
}

// Stream executes the optimized agent and returns an iterator of events.
// The optimized program is executed and results are yielded as events.
func (oa *OptimizedAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		result, err := oa.Invoke(ctx, input, opts...)
		if err != nil {
			yield(Event{Type: EventError, AgentID: oa.ID()}, err)
			return
		}
		if !yield(Event{Type: EventText, Text: result, AgentID: oa.ID()}, nil) {
			return
		}
		yield(Event{Type: EventDone, AgentID: oa.ID()}, nil)
	}
}

// Program returns the underlying optimized program.
func (oa *OptimizedAgent) Program() optimize.Program { return oa.program }

// BaseAgent returns the original unoptimized agent.
func (oa *OptimizedAgent) BaseAgent() *BaseAgent { return oa.base }
