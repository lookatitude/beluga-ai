package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	RegisterPlanner("self-discover", func(cfg PlannerConfig) (Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("self-discover planner requires an LLM")
		}
		var opts []SelfDiscoverOption
		if modules, ok := cfg.Extra["modules"].([]ReasoningModule); ok {
			opts = append(opts, WithReasoningModules(modules))
		}
		return NewSelfDiscoverPlanner(cfg.LLM, opts...), nil
	})
}

// ReasoningModule represents a self-discovered reasoning module that the LLM
// can select and compose to solve a task. Each module encapsulates a particular
// reasoning strategy.
type ReasoningModule struct {
	// Name is the identifier for this reasoning module.
	Name string
	// Description explains what this module does.
	Description string
	// Template is the prompt template for applying this module.
	Template string
}

// DefaultReasoningModules is the standard set of reasoning modules available
// for the Self-Discover strategy.
var DefaultReasoningModules = []ReasoningModule{
	{
		Name:        "critical_thinking",
		Description: "Evaluate assumptions and consider alternative viewpoints",
		Template:    "Think critically about the problem. What assumptions are being made? What alternative perspectives should be considered?",
	},
	{
		Name:        "decomposition",
		Description: "Break complex problems into smaller, manageable sub-problems",
		Template:    "Decompose this problem into smaller sub-problems. What are the key components that need to be addressed individually?",
	},
	{
		Name:        "analogical_reasoning",
		Description: "Draw parallels to similar known problems and apply solutions",
		Template:    "Think of analogous problems you know how to solve. How can solutions from similar domains be applied here?",
	},
	{
		Name:        "causal_reasoning",
		Description: "Identify cause-and-effect relationships",
		Template:    "Identify the cause-and-effect relationships in this problem. What causes what? What are the downstream effects?",
	},
	{
		Name:        "constraint_analysis",
		Description: "Identify and work within problem constraints",
		Template:    "What are the constraints of this problem? How do these constraints shape the solution space?",
	},
	{
		Name:        "abstraction",
		Description: "Abstract away details to find core problem structure",
		Template:    "Abstract the problem to its core structure. What is the essential pattern or structure underlying this problem?",
	},
	{
		Name:        "step_by_step",
		Description: "Solve the problem through sequential logical steps",
		Template:    "Solve this step by step. What is the first step? What logically follows from each step?",
	},
	{
		Name:        "hypothesis_testing",
		Description: "Generate and test hypotheses systematically",
		Template:    "Generate hypotheses about the solution. How can each hypothesis be tested or validated?",
	},
}

// SelfDiscoverPlanner implements the Self-Discover reasoning strategy.
// It operates in three phases: SELECT relevant reasoning modules for the task,
// ADAPT the selected modules to the specific problem, and IMPLEMENT the adapted
// modules as a structured reasoning plan. This approach typically uses only 2-3
// LLM calls, making it very efficient compared to multi-turn strategies.
//
// Reference: "Self-Discover: Large Language Models Self-Compose Reasoning Structures"
// (Zhou et al., 2024)
type SelfDiscoverPlanner struct {
	llm     llm.ChatModel
	modules []ReasoningModule
}

// SelfDiscoverOption configures a SelfDiscoverPlanner.
type SelfDiscoverOption func(*SelfDiscoverPlanner)

// WithReasoningModules overrides the default set of reasoning modules.
func WithReasoningModules(modules []ReasoningModule) SelfDiscoverOption {
	return func(p *SelfDiscoverPlanner) {
		p.modules = modules
	}
}

// NewSelfDiscoverPlanner creates a new Self-Discover planner with the given LLM.
func NewSelfDiscoverPlanner(model llm.ChatModel, opts ...SelfDiscoverOption) *SelfDiscoverPlanner {
	p := &SelfDiscoverPlanner{
		llm:     model,
		modules: DefaultReasoningModules,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Plan implements the Self-Discover three-phase strategy: SELECT → ADAPT → IMPLEMENT.
// It discovers task-specific reasoning structure, then uses it to solve the problem.
func (p *SelfDiscoverPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	// Phase 1: SELECT — choose relevant reasoning modules for this task
	selected, err := p.selectModules(ctx, state.Input)
	if err != nil {
		return nil, fmt.Errorf("self-discover select: %w", err)
	}

	// Phase 2: ADAPT — adapt the selected modules to the specific task
	adapted, err := p.adaptModules(ctx, state.Input, selected)
	if err != nil {
		return nil, fmt.Errorf("self-discover adapt: %w", err)
	}

	// Phase 3: IMPLEMENT — use the adapted reasoning structure to solve the task
	return p.implement(ctx, state, adapted)
}

// Replan re-runs the implementation phase with observations from previous actions,
// reusing any previously discovered reasoning structure stored in metadata.
func (p *SelfDiscoverPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	// If we have a cached reasoning structure, reuse it
	if structure, ok := state.Metadata["self_discover_structure"].(string); ok {
		return p.implement(ctx, state, structure)
	}
	// Otherwise do a full plan
	return p.Plan(ctx, state)
}

// selectModules asks the LLM to select the most relevant reasoning modules for the task.
func (p *SelfDiscoverPlanner) selectModules(ctx context.Context, task string) ([]ReasoningModule, error) {
	var moduleDescriptions strings.Builder
	for i, m := range p.modules {
		fmt.Fprintf(&moduleDescriptions, "%d. %s: %s\n", i+1, m.Name, m.Description)
	}

	prompt := fmt.Sprintf(
		"Given the following task, select the reasoning modules that are most useful for solving it.\n\n"+
			"Task: %s\n\n"+
			"Available reasoning modules:\n%s\n"+
			"Select the most relevant modules by listing their names, one per line. "+
			"Only include modules that are directly useful for this task.",
		task, moduleDescriptions.String(),
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("You are an expert at selecting reasoning strategies. Output only module names, one per line."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return nil, err
	}

	// Parse selected module names
	text := resp.Text()
	lines := strings.Split(text, "\n")

	moduleMap := make(map[string]ReasoningModule, len(p.modules))
	for _, m := range p.modules {
		moduleMap[m.Name] = m
	}

	var selected []ReasoningModule
	for _, line := range lines {
		name := strings.TrimSpace(strings.ToLower(line))
		// Strip numbering prefixes like "1. " or "- "
		name = strings.TrimLeft(name, "0123456789.- ")
		if m, ok := moduleMap[name]; ok {
			selected = append(selected, m)
		}
	}

	// If no modules were parsed, use all modules as fallback
	if len(selected) == 0 {
		selected = p.modules
	}

	return selected, nil
}

// adaptModules asks the LLM to adapt the selected reasoning modules to the specific task.
func (p *SelfDiscoverPlanner) adaptModules(ctx context.Context, task string, modules []ReasoningModule) (string, error) {
	var moduleTemplates strings.Builder
	for _, m := range modules {
		fmt.Fprintf(&moduleTemplates, "Module: %s\n%s\n\n", m.Name, m.Template)
	}

	prompt := fmt.Sprintf(
		"Adapt the following reasoning modules into a structured reasoning plan for the given task.\n\n"+
			"Task: %s\n\n"+
			"Selected reasoning modules:\n%s\n"+
			"Create an integrated reasoning structure by adapting these modules specifically "+
			"for this task. Describe the step-by-step reasoning process to follow.",
		task, moduleTemplates.String(),
	)

	resp, err := p.llm.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("You are an expert at composing reasoning strategies into structured plans."),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}

// implement applies the discovered reasoning structure to solve the task using the LLM.
func (p *SelfDiscoverPlanner) implement(ctx context.Context, state PlannerState, structure string) ([]Action, error) {
	messages := buildMessagesFromState(state)

	// Inject the reasoning structure as system context
	structureMsg := schema.NewSystemMessage(
		"Use the following reasoning structure to solve the task:\n\n" + structure,
	)
	msgs := make([]schema.Message, 0, len(messages)+1)
	msgs = append(msgs, structureMsg)
	msgs = append(msgs, messages...)

	// Bind tools
	model := p.llm
	if len(state.Tools) > 0 {
		model = model.BindTools(toolDefinitions(state.Tools))
	}

	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("self-discover implement: %w", err)
	}

	actions := parseAIResponse(resp)

	// Store the reasoning structure in the first action's metadata for reuse
	if len(actions) > 0 {
		if actions[0].Metadata == nil {
			actions[0].Metadata = make(map[string]any)
		}
		actions[0].Metadata["self_discover_structure"] = structure
	}

	return actions, nil
}
