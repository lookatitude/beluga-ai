// Package react provides ReAct (Reasoning + Acting) agent implementations.
// ReAct agents combine reasoning and acting in an iterative process.
//
//nolint:printf // Disable printf linter due to panic bug in golangci-lint v2.6.2 when analyzing range loops
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
		toolName := tool.Name()
		toolDesc := tool.Description()
		// Avoid fmt.Sprintf in range loop to prevent printf linter panic
		toolsDesc.WriteString("- ")
		toolsDesc.WriteString(toolName)
		toolsDesc.WriteString(": ")
		toolsDesc.WriteString(toolDesc)
		toolsDesc.WriteString("\n")
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
	config := a.BaseAgent.GetConfig()
	agentName := config.Name
	if err != nil {
		if a.GetMetrics() != nil {
			a.GetMetrics().RecordPlanningCall(ctx, agentName, time.Since(start), false)
		}
		return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("plan failed for agent %s: LLM generation failed: %w", agentName, err)
	}

	// Parse LLM response
	responseContent := llmResponse.GetContent()
	action, finish, err := a.parseResponse(responseContent)
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

// anyToString converts any value to string without using fmt.Sprintf.
// This prevents printf linter panic in golangci-lint v2.6.2 when analyzing range loops.
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case map[string]any:
		// Convert map to JSON string representation
		if jsonBytes, err := json.Marshal(val); err == nil {
			return string(jsonBytes)
		}
		return "{}"
	case []any:
		// Convert slice to JSON string representation
		if jsonBytes, err := json.Marshal(val); err == nil {
			return string(jsonBytes)
		}
		return "[]"
	default:
		// For complex types, use JSON marshaling
		if jsonBytes, err := json.Marshal(val); err == nil {
			return string(jsonBytes)
		}
		// Last resort: empty string (better than fmt.Sprintf which causes panic)
		return ""
	}
}

// constructScratchpad builds the agent scratchpad from intermediate steps.
func (a *ReActAgent) constructScratchpad(intermediateSteps []iface.IntermediateStep) string {
	var scratchpad strings.Builder

	for i, step := range intermediateSteps {
		stepNum := i + 1
		actionTool := step.Action.Tool
		observation := step.Observation
		actionLog := step.Action.Log

		// Avoid fmt.Sprintf in range loop to prevent printf linter panic
		scratchpad.WriteString("Step ")
		scratchpad.WriteString(strconv.Itoa(stepNum))
		scratchpad.WriteString(":\n")
		scratchpad.WriteString("Action: ")
		scratchpad.WriteString(actionTool)
		scratchpad.WriteString("\n")
		if step.Action.ToolInput != nil {
			toolInput := step.Action.ToolInput
			scratchpad.WriteString("Action Input: ")
			// Use helper function to avoid fmt.Sprintf in range loop
			toolInputStr := anyToString(toolInput)
			scratchpad.WriteString(toolInputStr)
			scratchpad.WriteString("\n")
		}
		scratchpad.WriteString("Observation: ")
		scratchpad.WriteString(observation)
		scratchpad.WriteString("\n")
		scratchpad.WriteString("Log: ")
		scratchpad.WriteString(actionLog)
		scratchpad.WriteString("\n\n")
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
		// Use helper function to avoid fmt.Sprintf in range loop
		valueStr := anyToString(value)
		formatted = strings.ReplaceAll(formatted, placeholder, valueStr)
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
