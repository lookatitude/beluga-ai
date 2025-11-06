// Package react provides ReAct (Reasoning + Acting) agent implementations.
// ReAct agents combine reasoning and acting in an iterative process.
package react

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/internal/base"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ReActAgent implements the ReAct prompting strategy.
// It iteratively reasons about problems and executes actions using tools.
type ReActAgent struct {
	*base.BaseAgent
	llm           llmsiface.ChatModel
	tools         []tools.Tool
	prompt        interface{} // Prompt template (to be defined)
	toolMap       map[string]tools.Tool
	scratchpadKey string
}

// NewReActAgent creates a new ReAct agent.
//
// Parameters:
//   - name: Unique identifier for the agent
//   - chatLLM: Chat model for reasoning and action generation
//   - agentTools: Tools available to the agent
//   - promptTemplate: Template defining the ReAct behavior
//   - opts: Optional configuration
//
// Returns:
//   - New ReAct agent instance
//   - Error if initialization fails
func NewReActAgent(name string, chatLLM llmsiface.ChatModel, agentTools []tools.Tool, promptTemplate interface{}, opts ...iface.Option) (*ReActAgent, error) {
	// Validate required parameters
	if chatLLM == nil {
		return nil, fmt.Errorf("chatLLM cannot be nil")
	}

	// Create base agent first
	baseAgent, err := base.NewBaseAgent(name, chatLLM, agentTools, opts...) // Pass the ChatModel as it implements LLM interface
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

	agent := &ReActAgent{
		BaseAgent:     baseAgent,
		llm:           chatLLM,
		tools:         agentTools,
		prompt:        promptTemplate,
		toolMap:       toolMap,
		scratchpadKey: "agent_scratchpad",
	}

	return agent, nil
}

// Plan implements the planning phase of the ReAct strategy.
// It generates the next action or final answer based on the current state.
func (a *ReActAgent) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	start := time.Now()

	// Construct scratchpad from intermediate steps
	scratchpad := a.constructScratchpad(intermediateSteps)

	// Prepare prompt inputs
	promptInputs := make(map[string]any)
	for k, v := range inputs {
		promptInputs[k] = v
	}
	promptInputs[a.scratchpadKey] = scratchpad

	// Add tools description to inputs
	var toolsDesc strings.Builder
	for _, tool := range a.tools {
		toolsDesc.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}
	promptInputs["tools"] = toolsDesc.String()

	// Format prompt
	promptText := a.formatPrompt(promptInputs)

	// Create messages for LLM
	messages := []schema.Message{
		schema.NewHumanMessage(promptText),
	}

	// Call LLM
	llmResponse, err := a.llm.Generate(ctx, messages)
	agentName := a.BaseAgent.GetConfig().Name
	if err != nil {
		if a.GetMetrics() != nil {
			a.GetMetrics().RecordPlanningCall(ctx, agentName, time.Since(start), false)
		}
		return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("plan failed for agent %s: LLM generation failed: %w", agentName, err)
	}

	// Parse LLM response
	action, finish, err := a.parseResponse(llmResponse.GetContent())
	if err != nil {
		if a.GetMetrics() != nil {
			a.GetMetrics().RecordPlanningCall(ctx, agentName, time.Since(start), false)
		}
		return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("plan failed for agent %s: failed to parse LLM response: %w", agentName, err)
	}

	if a.GetMetrics() != nil {
		a.GetMetrics().RecordPlanningCall(ctx, agentName, time.Since(start), true)
	}

	return action, finish, nil
}

// constructScratchpad builds the agent scratchpad from intermediate steps.
func (a *ReActAgent) constructScratchpad(intermediateSteps []iface.IntermediateStep) string {
	var scratchpad strings.Builder

	for i, step := range intermediateSteps {
		scratchpad.WriteString(fmt.Sprintf("Step %d:\n", i+1))
		scratchpad.WriteString(fmt.Sprintf("Action: %s\n", step.Action.Tool))
		if step.Action.ToolInput != nil {
			scratchpad.WriteString(fmt.Sprintf("Action Input: %v\n", step.Action.ToolInput))
		}
		scratchpad.WriteString(fmt.Sprintf("Observation: %s\n", step.Observation))
		scratchpad.WriteString(fmt.Sprintf("Log: %s\n\n", step.Action.Log))
	}

	return scratchpad.String()
}

// formatPrompt formats the prompt template with inputs.
func (a *ReActAgent) formatPrompt(inputs map[string]any) string {
	template, ok := a.prompt.(string)
	if !ok {
		// Fallback or error handling
		return ""
	}

	formatted := template
	for key, value := range inputs {
		placeholder := "{" + key + "}"
		formatted = strings.ReplaceAll(formatted, placeholder, fmt.Sprintf("%v", value))
	}

	return formatted
}

// parseResponse parses the LLM response to extract actions or final answers.
func (a *ReActAgent) parseResponse(response string) (iface.AgentAction, iface.AgentFinish, error) {
	// Regex for Final Answer
	finalAnswerRegex := regexp.MustCompile(`(?i)Final Answer:\s*(.*)`)
	if matches := finalAnswerRegex.FindStringSubmatch(response); len(matches) > 0 {
		finalAnswer := strings.TrimSpace(matches[1])
		return iface.AgentAction{}, iface.AgentFinish{
			ReturnValues: map[string]any{"output": finalAnswer},
			Log:          response,
		}, nil
	}

	// Regex for Action/Action Input format
	actionRegex := regexp.MustCompile(`(?s)Action:\s*([\w\-]+)\s*Action Input:\s*(.*)`)
	if matches := actionRegex.FindStringSubmatch(response); len(matches) >= 3 {
		toolName := strings.TrimSpace(matches[1])
		toolInputStr := strings.TrimSpace(matches[2])

		// Verify tool exists
		if _, exists := a.toolMap[toolName]; !exists {
			return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("unknown tool: %s", toolName)
		}

		// Parse tool input
		toolInput, err := a.parseToolInput(toolInputStr)
		if err != nil {
			return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("failed to parse tool input: %w", err)
		}

		return iface.AgentAction{
			Tool:      toolName,
			ToolInput: toolInput,
			Log:       response,
		}, iface.AgentFinish{}, nil
	}

	return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("could not parse response as action or final answer: %s", response)
}

// parseToolInput parses the tool input string into the appropriate format.
func (a *ReActAgent) parseToolInput(inputStr string) (any, error) {
	// First try to parse as JSON
	var jsonInput map[string]any
	if err := json.Unmarshal([]byte(inputStr), &jsonInput); err == nil {
		return jsonInput, nil
	}

	// If not JSON, try to handle as simple string
	trimmed := strings.TrimSpace(inputStr)
	if strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
		unquoted, err := strconv.Unquote(trimmed)
		if err == nil {
			return unquoted, nil
		}
	}

	// Return as string if nothing else works
	return inputStr, nil
}

// GetLLM returns the chat model used by this agent.
func (a *ReActAgent) GetLLM() llmsiface.LLM {
	// Return the ChatModel from BaseAgent (ChatModel implements LLM interface)
	return a.BaseAgent.GetLLM()
}

// GetMetrics returns the metrics recorder for the agent.
func (a *ReActAgent) GetMetrics() iface.MetricsRecorder {
	return a.BaseAgent.GetMetrics()
}

// Ensure ReActAgent implements the required interfaces
var (
	_ iface.Agent          = (*ReActAgent)(nil)
	_ iface.CompositeAgent = (*ReActAgent)(nil)
)
