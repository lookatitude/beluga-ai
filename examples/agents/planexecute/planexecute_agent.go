// Package main demonstrates the PlanExecute agent pattern in Beluga AI.
// PlanExecute agents separate planning from execution, creating a detailed plan
// before executing each step. This approach provides more control and makes
// debugging easier compared to ReAct agents.
//
// Key patterns demonstrated:
//   - PlanExecute agent creation and configuration
//   - Separate LLMs for planning vs execution (optional)
//   - Plan generation and step-by-step execution
//   - OTEL instrumentation for observability
//   - Error handling and plan recovery
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/agents/providers/planexecute"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/gofunc"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// We define a tracer for observability - this helps debug agent behavior in production
var tracer = otel.Tracer("beluga.agents.planexecute.example")

// PlanExecuteExample demonstrates PlanExecute agent usage
type PlanExecuteExample struct {
	agent *planexecute.PlanExecuteAgent
	name  string
}

// PlanExecuteResult holds the result of running the agent
type PlanExecuteResult struct {
	Plan          *planexecute.ExecutionPlan
	StepResults   map[string]any
	FinalOutput   string
	TotalDuration time.Duration
	StepsExecuted int
}

// NewPlanExecuteExample creates a new example with the given LLM and tools
func NewPlanExecuteExample(
	name string,
	llmClient llmsiface.ChatModel,
	availableTools []tools.Tool,
	opts ...PlanExecuteOption,
) (*PlanExecuteExample, error) {
	// Create the PlanExecute agent
	agent, err := planexecute.NewPlanExecuteAgent(
		name,
		llmClient,
		availableTools,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create PlanExecute agent: %w", err)
	}

	// Apply configuration options
	config := &planExecuteConfig{
		maxPlanSteps:  10,
		maxIterations: 20,
	}
	for _, opt := range opts {
		opt(config)
	}

	// Configure the agent
	agent = agent.
		WithMaxPlanSteps(config.maxPlanSteps).
		WithMaxIterations(config.maxIterations)

	// Optionally use separate LLMs for planning and execution
	if config.plannerLLM != nil {
		agent = agent.WithPlannerLLM(config.plannerLLM)
	}
	if config.executorLLM != nil {
		agent = agent.WithExecutorLLM(config.executorLLM)
	}

	return &PlanExecuteExample{
		agent: agent,
		name:  name,
	}, nil
}

// planExecuteConfig holds configuration for the example
type planExecuteConfig struct {
	maxPlanSteps  int
	maxIterations int
	plannerLLM    llmsiface.ChatModel
	executorLLM   llmsiface.ChatModel
}

// PlanExecuteOption configures the example
type PlanExecuteOption func(*planExecuteConfig)

// WithMaxPlanSteps sets the maximum number of steps in a plan
func WithMaxPlanSteps(steps int) PlanExecuteOption {
	return func(c *planExecuteConfig) {
		c.maxPlanSteps = steps
	}
}

// WithMaxIterations sets the maximum execution iterations
func WithMaxIterations(iterations int) PlanExecuteOption {
	return func(c *planExecuteConfig) {
		c.maxIterations = iterations
	}
}

// WithPlannerLLM sets a separate LLM for planning (e.g., GPT-4 for better reasoning)
func WithPlannerLLM(llm llmsiface.ChatModel) PlanExecuteOption {
	return func(c *planExecuteConfig) {
		c.plannerLLM = llm
	}
}

// WithExecutorLLM sets a separate LLM for execution (e.g., GPT-3.5 for speed)
func WithExecutorLLM(llm llmsiface.ChatModel) PlanExecuteOption {
	return func(c *planExecuteConfig) {
		c.executorLLM = llm
	}
}

// Run executes the PlanExecute agent with the given task
func (e *PlanExecuteExample) Run(ctx context.Context, task string) (*PlanExecuteResult, error) {
	ctx, span := tracer.Start(ctx, "planexecute.run",
		trace.WithAttributes(
			attribute.String("agent_name", e.name),
			attribute.String("task", task),
		))
	defer span.End()

	start := time.Now()
	result := &PlanExecuteResult{}

	// Phase 1: Generate the plan
	plan, err := e.generatePlan(ctx, task)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	result.Plan = plan
	span.AddEvent("plan_generated", trace.WithAttributes(
		attribute.Int("steps", len(plan.Steps)),
	))

	// Phase 2: Execute the plan
	stepResults, err := e.executePlan(ctx, plan)
	if err != nil {
		span.RecordError(err)
		// Don't fail completely - return partial results
		log.Printf("Execution had errors: %v", err)
	}

	result.StepResults = stepResults
	result.StepsExecuted = len(stepResults)

	// Phase 3: Synthesize final output
	finalOutput, err := e.synthesizeOutput(ctx, plan, stepResults)
	if err != nil {
		span.RecordError(err)
		log.Printf("Output synthesis failed: %v", err)
	}

	result.FinalOutput = finalOutput
	result.TotalDuration = time.Since(start)

	span.SetAttributes(
		attribute.Int("steps_executed", result.StepsExecuted),
		attribute.Float64("duration_ms", float64(result.TotalDuration.Milliseconds())),
	)
	span.SetStatus(codes.Ok, "")

	return result, nil
}

// generatePlan uses the agent to create an execution plan
func (e *PlanExecuteExample) generatePlan(ctx context.Context, task string) (*planexecute.ExecutionPlan, error) {
	ctx, span := tracer.Start(ctx, "planexecute.generate_plan")
	defer span.End()

	// Call the agent's Plan method
	action, _, err := e.agent.Plan(ctx, nil, map[string]any{"input": task})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("agent planning failed: %w", err)
	}

	// Extract the plan from the action
	toolInput, ok := action.ToolInput.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tool input is not a map")
	}
	planJSON, ok := toolInput["plan"].(string)
	if !ok {
		return nil, fmt.Errorf("no plan found in agent action")
	}

	var plan planexecute.ExecutionPlan
	if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	span.SetAttributes(
		attribute.Int("plan_steps", len(plan.Steps)),
		attribute.String("plan_goal", plan.Goal),
	)

	return &plan, nil
}

// executePlan executes each step of the plan
func (e *PlanExecuteExample) executePlan(ctx context.Context, plan *planexecute.ExecutionPlan) (map[string]any, error) {
	ctx, span := tracer.Start(ctx, "planexecute.execute_plan")
	defer span.End()

	results, err := e.agent.ExecutePlan(ctx, plan)
	if err != nil {
		span.RecordError(err)
		return results, fmt.Errorf("plan execution failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("results_count", len(results)),
	)

	return results, nil
}

// synthesizeOutput creates a final summary from the plan execution results
func (e *PlanExecuteExample) synthesizeOutput(ctx context.Context, plan *planexecute.ExecutionPlan, results map[string]any) (string, error) {
	ctx, span := tracer.Start(ctx, "planexecute.synthesize")
	defer span.End()

	// Build a summary of what was accomplished
	var summary string
	summary += fmt.Sprintf("Goal: %s\n", plan.Goal)
	summary += fmt.Sprintf("Steps completed: %d/%d\n", len(results), len(plan.Steps))
	summary += "\nResults:\n"

	for stepKey, result := range results {
		summary += fmt.Sprintf("  %s: %v\n", stepKey, result)
	}

	return summary, nil
}

// DisplayPlan prints a formatted view of the execution plan
func (e *PlanExecuteExample) DisplayPlan(plan *planexecute.ExecutionPlan) {
	fmt.Printf("\n=== Execution Plan ===\n")
	fmt.Printf("Goal: %s\n", plan.Goal)
	fmt.Printf("Total Steps: %d\n\n", plan.TotalSteps)

	for _, step := range plan.Steps {
		fmt.Printf("Step %d: %s\n", step.StepNumber, step.Action)
		if step.Tool != "" {
			fmt.Printf("  Tool: %s\n", step.Tool)
			fmt.Printf("  Input: %s\n", step.Input)
		}
		if step.Reasoning != "" {
			fmt.Printf("  Reasoning: %s\n", step.Reasoning)
		}
		fmt.Println()
	}
}

// createResearchTools creates sample tools for a research agent
func createResearchTools() []tools.Tool {
	// Web search tool
	webSearch, _ := gofunc.NewGoFunctionTool(
		"web_search",
		"Search the web for information. Returns relevant search results.",
		`{"type": "object", "properties": {"query": {"type": "string", "description": "The search query"}}, "required": ["query"]}`,
		func(ctx context.Context, args map[string]any) (any, error) {
			query, _ := args["query"].(string)
			// In production, call a real search API
			return fmt.Sprintf(`{
				"query": "%s",
				"results": [
					{"title": "Result 1", "snippet": "Relevant information about %s"},
					{"title": "Result 2", "snippet": "More details on %s"}
				]
			}`, query, query, query), nil
		},
	)

	// Calculator tool
	calculator, _ := gofunc.NewGoFunctionTool(
		"calculator",
		"Perform arithmetic calculations. Input should be a mathematical expression.",
		`{"type": "object", "properties": {"expression": {"type": "string", "description": "The mathematical expression"}}, "required": ["expression"]}`,
		func(ctx context.Context, args map[string]any) (any, error) {
			expression, _ := args["expression"].(string)
			// In production, use a proper math evaluator
			return fmt.Sprintf(`{"expression": "%s", "result": "calculated"}`, expression), nil
		},
	)

	// Note-taking tool
	notepad, _ := gofunc.NewGoFunctionTool(
		"take_notes",
		"Save notes for later reference. Use this to record important findings.",
		`{"type": "object", "properties": {"note": {"type": "string", "description": "The note to save"}}, "required": ["note"]}`,
		func(ctx context.Context, args map[string]any) (any, error) {
			note, _ := args["note"].(string)
			return fmt.Sprintf(`{"status": "saved", "note": "%s"}`, note), nil
		},
	)

	// Summary tool
	summarize, _ := gofunc.NewGoFunctionTool(
		"summarize",
		"Summarize text or findings into a concise format.",
		`{"type": "object", "properties": {"text": {"type": "string", "description": "The text to summarize"}}, "required": ["text"]}`,
		func(ctx context.Context, args map[string]any) (any, error) {
			text, _ := args["text"].(string)
			// In production, might call an LLM
			return fmt.Sprintf(`{"summary": "Summary of: %s"}`, text), nil
		},
	)

	return []tools.Tool{webSearch, calculator, notepad, summarize}
}

func main() {
	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create the LLM client
	llmClient, err := llms.NewOpenAIChat(
		llms.WithAPIKey(apiKey),
		llms.WithModelName("gpt-4"),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	// Create research tools
	researchTools := createResearchTools()

	// Create the PlanExecute example
	example, err := NewPlanExecuteExample(
		"research-planner",
		llmClient,
		researchTools,
		WithMaxPlanSteps(5),
		WithMaxIterations(10),
	)
	if err != nil {
		log.Fatalf("Failed to create example: %v", err)
	}

	fmt.Println("=== PlanExecute Agent Example ===")
	fmt.Println()

	// Run a research task
	task := "Research the benefits of renewable energy and calculate the potential savings for a household switching to solar panels"

	fmt.Printf("Task: %s\n", task)
	fmt.Println()

	result, err := example.Run(ctx, task)
	if err != nil {
		log.Fatalf("Task failed: %v", err)
	}

	// Display the plan
	if result.Plan != nil {
		example.DisplayPlan(result.Plan)
	}

	// Display results
	fmt.Println("=== Execution Results ===")
	fmt.Printf("Steps executed: %d\n", result.StepsExecuted)
	fmt.Printf("Duration: %v\n", result.TotalDuration)
	fmt.Println()
	fmt.Println("Final Output:")
	fmt.Println(result.FinalOutput)
}
