// Package planexecute provides Plan-and-Execute agent implementations.
// Plan-and-Execute agents first create a detailed plan, then execute it step by step.
// This approach is useful for complex tasks that require multi-step reasoning.
package planexecute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/internal/base"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// PlanExecuteAgent implements the Plan-and-Execute strategy.
// It first creates a plan, then executes it step by step.
type PlanExecuteAgent struct {
	*base.BaseAgent
	llm           llmsiface.ChatModel
	tools         []tools.Tool
	toolMap       map[string]tools.Tool
	plannerLLM    llmsiface.ChatModel // Optional separate LLM for planning
	executorLLM   llmsiface.ChatModel // Optional separate LLM for execution
	maxPlanSteps  int
	maxIterations int
}

// PlanStep represents a single step in the execution plan.
type PlanStep struct {
	StepNumber int    `json:"step_number"`
	Action     string `json:"action"`
	Tool       string `json:"tool,omitempty"`
	Input      string `json:"input,omitempty"`
	Reasoning  string `json:"reasoning,omitempty"`
}

// ExecutionPlan represents a complete execution plan.
type ExecutionPlan struct {
	Goal        string     `json:"goal"`
	Steps       []PlanStep `json:"steps"`
	TotalSteps  int        `json:"total_steps"`
}

// NewPlanExecuteAgent creates a new Plan-and-Execute agent.
//
// Parameters:
//   - name: Unique identifier for the agent
//   - chatLLM: Chat model for planning and execution
//   - agentTools: Tools available to the agent
//   - opts: Optional configuration
//
// Returns:
//   - New Plan-and-Execute agent instance
//   - Error if initialization fails
func NewPlanExecuteAgent(name string, chatLLM llmsiface.ChatModel, agentTools []tools.Tool, opts ...iface.Option) (*PlanExecuteAgent, error) {
	// Validate required parameters
	if chatLLM == nil {
		return nil, fmt.Errorf("chatLLM cannot be nil")
	}

	// Create base agent first
	baseAgent, err := base.NewBaseAgent(name, chatLLM, agentTools, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create base agent: %w", err)
	}

	// Build tool map for efficient lookup
	toolMap := make(map[string]tools.Tool)
	for _, tool := range agentTools {
		toolName := tool.Name()
		if _, exists := toolMap[toolName]; exists {
			return nil, fmt.Errorf("initialize agent %s: duplicate tool name: %s", name, toolName)
		}
		toolMap[toolName] = tool
	}

	agent := &PlanExecuteAgent{
		BaseAgent:     baseAgent,
		llm:           chatLLM,
		tools:         agentTools,
		toolMap:       toolMap,
		plannerLLM:    chatLLM, // Use same LLM by default
		executorLLM:   chatLLM, // Use same LLM by default
		maxPlanSteps:  10,      // Default max steps
		maxIterations: 20,      // Default max iterations
	}

	return agent, nil
}

// WithPlannerLLM sets a separate LLM for planning.
func (a *PlanExecuteAgent) WithPlannerLLM(llm llmsiface.ChatModel) *PlanExecuteAgent {
	a.plannerLLM = llm
	return a
}

// WithExecutorLLM sets a separate LLM for execution.
func (a *PlanExecuteAgent) WithExecutorLLM(llm llmsiface.ChatModel) *PlanExecuteAgent {
	a.executorLLM = llm
	return a
}

// WithMaxPlanSteps sets the maximum number of steps in a plan.
func (a *PlanExecuteAgent) WithMaxPlanSteps(maxSteps int) *PlanExecuteAgent {
	a.maxPlanSteps = maxSteps
	return a
}

// WithMaxIterations sets the maximum number of execution iterations.
func (a *PlanExecuteAgent) WithMaxIterations(maxIterations int) *PlanExecuteAgent {
	a.maxIterations = maxIterations
	return a
}

// Plan implements the planning phase of the Plan-and-Execute strategy.
// It generates a complete execution plan based on the input.
func (a *PlanExecuteAgent) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	start := time.Now()

	// Extract the main input (typically "input" key)
	inputText := ""
	if inputVal, ok := inputs["input"].(string); ok {
		inputText = inputVal
	} else {
		// Try to find any string input
		for _, v := range inputs {
			if str, ok := v.(string); ok && str != "" {
				inputText = str
				break
			}
		}
	}

	if inputText == "" {
		return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("no input found in inputs")
	}

	// Generate plan using planner LLM
	plan, err := a.generatePlan(ctx, inputText)
	if err != nil {
		config := a.BaseAgent.GetConfig()
		agentName := config.Name
		if a.GetMetrics() != nil {
			a.GetMetrics().RecordPlanningCall(ctx, agentName, time.Since(start), false)
		}
		return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("plan generation failed: %w", err)
	}

	// Store plan in context for execution phase
	// For now, we'll return an action that indicates planning is complete
	// The actual execution will happen in a separate phase

	config := a.BaseAgent.GetConfig()
	agentName := config.Name
	if a.GetMetrics() != nil {
		a.GetMetrics().RecordPlanningCall(ctx, agentName, time.Since(start), true)
	}

	// Return a special action indicating plan creation
	planJSON, _ := json.Marshal(plan)
	return iface.AgentAction{
		Tool:      "execute_plan",
		ToolInput: map[string]any{"plan": string(planJSON)},
		Log:       fmt.Sprintf("Generated plan with %d steps", len(plan.Steps)),
	}, iface.AgentFinish{}, nil
}

// generatePlan creates an execution plan using the planner LLM.
func (a *PlanExecuteAgent) generatePlan(ctx context.Context, goal string) (*ExecutionPlan, error) {
	// Build tools description
	var toolsDesc strings.Builder
	for _, tool := range a.tools {
		toolName := tool.Name()
		toolDesc := tool.Description()
		toolsDesc.WriteString("- ")
		toolsDesc.WriteString(toolName)
		toolsDesc.WriteString(": ")
		toolsDesc.WriteString(toolDesc)
		toolsDesc.WriteString("\n")
	}

	// Create planning prompt
	prompt := fmt.Sprintf(`You are a planning agent. Your task is to create a detailed execution plan to achieve the following goal:

Goal: %s

Available tools:
%s

Create a step-by-step plan. Each step should:
1. Have a clear action description
2. Specify which tool to use (if applicable)
3. Include the input for the tool
4. Explain the reasoning

Return the plan as a JSON object with this structure:
{
  "goal": "the goal",
  "steps": [
    {
      "step_number": 1,
      "action": "description of the action",
      "tool": "tool_name",
      "input": "input for the tool",
      "reasoning": "why this step is needed"
    }
  ],
  "total_steps": <number>
}

Limit the plan to a maximum of %d steps.`, goal, toolsDesc.String(), a.maxPlanSteps)

	// Create messages for LLM
	messages := []schema.Message{
		schema.NewSystemMessage("You are a planning agent that creates detailed execution plans."),
		schema.NewHumanMessage(prompt),
	}

	// Call planner LLM
	response, err := a.plannerLLM.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse response
	responseText := response.GetContent()
	plan, err := a.parsePlan(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	// Validate plan
	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("plan has no steps")
	}

	if len(plan.Steps) > a.maxPlanSteps {
		plan.Steps = plan.Steps[:a.maxPlanSteps]
	}

	plan.TotalSteps = len(plan.Steps)
	plan.Goal = goal

	return plan, nil
}

// parsePlan parses the LLM response to extract the execution plan.
func (a *PlanExecuteAgent) parsePlan(responseText string) (*ExecutionPlan, error) {
	// Try to extract JSON from response
	// The LLM might wrap the JSON in markdown code blocks or add extra text
	jsonStart := strings.Index(responseText, "{")
	jsonEnd := strings.LastIndex(responseText, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonText := responseText[jsonStart : jsonEnd+1]

	var plan ExecutionPlan
	if err := json.Unmarshal([]byte(jsonText), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}

	return &plan, nil
}

// ExecutePlan executes a complete plan step by step.
func (a *PlanExecuteAgent) ExecutePlan(ctx context.Context, plan *ExecutionPlan) (map[string]any, error) {
	results := make(map[string]any)
	intermediateSteps := make([]iface.IntermediateStep, 0)

	for i, step := range plan.Steps {
		// Check iteration limit
		if i >= a.maxIterations {
			return results, fmt.Errorf("max iterations (%d) reached", a.maxIterations)
		}

		// Execute step
		observation, err := a.executeStep(ctx, step, intermediateSteps)
		if err != nil {
			return results, fmt.Errorf("step %d execution failed: %w", step.StepNumber, err)
		}

		// Record intermediate step
		action := iface.AgentAction{
			Tool:      step.Tool,
			ToolInput: step.Input,
			Log:       step.Reasoning,
		}

		intermediateSteps = append(intermediateSteps, iface.IntermediateStep{
			Action:      action,
			Observation: observation,
		})

		// Store result
		results[fmt.Sprintf("step_%d", step.StepNumber)] = observation
	}

	return results, nil
}

// executeStep executes a single plan step.
func (a *PlanExecuteAgent) executeStep(ctx context.Context, step PlanStep, previousSteps []iface.IntermediateStep) (string, error) {
	// If step has a tool, execute it
	if step.Tool != "" {
		tool, exists := a.toolMap[step.Tool]
		if !exists {
			return "", fmt.Errorf("tool '%s' not found", step.Tool)
		}

		// Parse tool input
		var toolInput any = step.Input
		if strings.HasPrefix(step.Input, "{") {
			// Try to parse as JSON
			var inputMap map[string]any
			if err := json.Unmarshal([]byte(step.Input), &inputMap); err == nil {
				toolInput = inputMap
			}
		}

		// Execute tool
		result, err := tool.Invoke(ctx, toolInput)
		if err != nil {
			return "", fmt.Errorf("tool execution failed: %w", err)
		}

		// Convert result to string
		resultStr := fmt.Sprintf("%v", result)
		return resultStr, nil
	}

	// If no tool, use executor LLM to perform the action
	// This allows for actions that don't require tools
	prompt := fmt.Sprintf(`Execute the following action:

Action: %s
Reasoning: %s

Previous steps:
%s

Provide the result of executing this action.`, step.Action, step.Reasoning, a.formatPreviousSteps(previousSteps))

	messages := []schema.Message{
		schema.NewSystemMessage("You are an execution agent that performs actions."),
		schema.NewHumanMessage(prompt),
	}

	response, err := a.executorLLM.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("executor LLM failed: %w", err)
	}

	return response.GetContent(), nil
}

// formatPreviousSteps formats previous steps for context.
func (a *PlanExecuteAgent) formatPreviousSteps(steps []iface.IntermediateStep) string {
	var builder strings.Builder
	for i, step := range steps {
		builder.WriteString(fmt.Sprintf("Step %d: %s\nResult: %s\n\n", i+1, step.Action.Log, step.Observation))
	}
	return builder.String()
}

// InputVariables returns the expected input keys for the agent.
func (a *PlanExecuteAgent) InputVariables() []string {
	return []string{"input"}
}

// OutputVariables returns the keys for the agent's final output.
func (a *PlanExecuteAgent) OutputVariables() []string {
	return []string{"output", "plan", "steps"}
}

// GetTools returns the tools available to the agent.
func (a *PlanExecuteAgent) GetTools() []tools.Tool {
	return a.tools
}

// GetLLM returns the LLM instance used by the agent.
func (a *PlanExecuteAgent) GetLLM() llmsiface.LLM {
	return a.llm
}
