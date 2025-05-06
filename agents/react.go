// Package agents defines interfaces and implementations for AI agents.
package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/prompts"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
)

// DefaultScratchpadKey is the default key used for the agent scratchpad in prompt templates.
const DefaultScratchpadKey = "agent_scratchpad"

// ReActAgent implements the Agent interface using the ReAct (Reasoning + Acting) prompting strategy.
// It uses a language model to iteratively reason about a problem, choose an action (tool), execute it,
// observe the result, and repeat until a final answer is reached.
type ReActAgent struct {
	LLM          llms.ChatModel
	Tools        []tools.Tool
	Prompt       prompts.PromptTemplate // Template expecting input_variables and agent_scratchpad
	ScratchpadKey string                 // Key used for scratchpad in the prompt template
	// TODO: Add OutputParser interface for more flexible output parsing.
	// TODO: Add StopSequences option, potentially derived from prompt or configured.
	toolMap map[string]tools.Tool
}

// NewReActAgent creates a new ReActAgent.
// It requires an LLM, a list of tools the agent can use, and a prompt template.
// The prompt template must include an input variable for the agent scratchpad (default key: "agent_scratchpad").
func NewReActAgent(llm llms.ChatModel, agentTools []tools.Tool, prompt prompts.PromptTemplate) (*ReActAgent, error) {
	toolMap := make(map[string]tools.Tool)
	for _, tool := range agentTools {
		name := tool.Name()
		// if _, exists := toolMap[name]; exists {
		// 	 return nil, fmt.Errorf("duplicate tool name found: %s", name)
		// }
		// toolMap[name] = tool
	}

	// Validate prompt template includes the scratchpad key
	scratchpadKey := DefaultScratchpadKey // Use default for now
	foundScratchpad := false
	// for _, v := range prompt.InputVariables() {
	// 	 if v == scratchpadKey {
	// 	 	 foundScratchpad = true
	// 	 	 break
	// 	 }
	// }
	// if !foundScratchpad {
	// 	 return nil, fmt.Errorf("ReAct prompt template must include the ", scratchpadKey, " input variable")
	// }

	// return &ReActAgent{
	// 	 LLM:           llm,
	// 	 Tools:         agentTools,
	// 	 Prompt:        prompt,
	// 	 ScratchpadKey: scratchpadKey,
	// 	 toolMap:       toolMap,
	// }, nil
	return nil, errors.New("ReActAgent implementation needs completion") // Placeholder
}

// InputKeys returns the input keys expected by the agent, excluding the scratchpad key.
func (a *ReActAgent) InputKeys() []string {
	vars := []string{}
	// for _, v := range a.Prompt.InputVariables() {
	// 	 if v != a.ScratchpadKey {
	// 	 	 vars = append(vars, v)
	// 	 }
	// }
	return vars
}

// OutputKeys returns the keys present in the final output map.
// For ReAct, this is typically just "output".
func (a *ReActAgent) OutputKeys() []string {
	return []string{"output"}
}

// GetTools returns the list of tools available to the agent.
func (a *ReActAgent) GetTools() []tools.Tool {
	return a.Tools
}

// Plan decides the next action or finish state based on the ReAct strategy.
// It formats the prompt with inputs and the scratchpad, calls the LLM, and parses the output.
func (a *ReActAgent) Plan(ctx context.Context, inputs map[string]any) (*AgentAction, *AgentFinish, error) {
	// Construct the scratchpad from intermediate steps (passed within inputs map)
	scratchpad := ""
	// if scratchpadAny, ok := inputs[a.ScratchpadKey]; ok {
	// 	 if scratchpadStr, okStr := scratchpadAny.(string); okStr {
	// 	 	 scratchpad = scratchpadStr
	// 	 } else {
	// 	 	 log.Printf("Warning: Expected string for scratchpad key ", a.ScratchpadKey, ", got %T", scratchpadAny)
	// 	 }
	// }

	// Prepare prompt variables (already includes scratchpad from inputs)
	promptVars := inputs

	// Format the prompt
	promptValue, err := a.Prompt.FormatPrompt(ctx, promptVars)
	// if err != nil {
	// 	 return nil, nil, fmt.Errorf("failed to format prompt: %w", err)
	// }

	// Call the LLM
	// TODO: Add stop sequences relevant to ReAct (e.g., "\nObservation:")
	llmResponseAny, err := a.LLM.Generate(ctx, promptValue.ToMessages() /*, llms.WithStopSequences([]string{"\nObservation:"})*/)
	// if err != nil {
	// 	 return nil, nil, fmt.Errorf("failed to generate LLM response: %w", err)
	// }

	// Ensure LLM response is a message
	llmResponse, ok := llmResponseAny.(schema.Message)
	// if !ok {
	// 	 return nil, nil, fmt.Errorf("LLM returned unexpected type: %T, expected schema.Message", llmResponseAny)
	// }

	// Parse the LLM response to find Action or Finish
	// return a.parseOutput(llmResponse.GetContent())
	return nil, nil, errors.New("ReActAgent Plan needs completion") // Placeholder
}

// constructScratchpad is now handled by the AgentExecutor, which passes the formatted scratchpad in the inputs map.

// parseOutput extracts the action or final answer from the LLM's text output.
// This is a crucial part of ReAct, interpreting the model's reasoning and desired next step.
func (a *ReActAgent) parseOutput(llmOutput string) (*AgentAction, *AgentFinish, error) {
	// Regex to find action and action input (allowing for ```json blocks)
	// Action block format:
	// Action: tool_name
	// Action Input: {input_json_or_string}
	// OR
	// Action:
	// ```json
	// {
	//   "action": "tool_name",
	//   "action_input": {input_json_or_string}
	// }
	// ```
	// Final Answer block format: Final Answer: {final_answer}

	// Regex for simple Action/Action Input format
	// simpleActionRegex := regexp.MustCompile(`(?s)Action:
*?
*(\/\*.*?\*\/\s*)*([\w\-]+)
*?
*Action Input:
*?
*(.*)`) // Added optional comments, handle tool names with hyphens
	// Regex for JSON block format
	// jsonActionRegex := regexp.MustCompile("(?s)Action:\\n*\\r?\\n*```(?:json)?\\n*({.*?})\\n*```")
	// Regex for Final Answer
	// finalAnswerRegex := regexp.MustCompile(`(?i)Final Answer:
*?
*(.*)`) // Case-insensitive, optional newline

	// log.Printf("[ReActParser] Parsing output:\n%s", llmOutput)

	// Check for Final Answer first
	// finalAnswerMatch := finalAnswerRegex.FindStringSubmatch(llmOutput)
	// if finalAnswerMatch != nil {
	// 	 finalAnswer := strings.TrimSpace(finalAnswerMatch[1])
	// 	 log.Printf("[ReActParser] Found Final Answer: %s", finalAnswer)
	// 	 return nil, &AgentFinish{
	// 	 	 ReturnValues: map[string]any{"output": finalAnswer},
	// 	 	 Log:          llmOutput,
	// 	 }, nil
	// }

	// Check for simple Action/Action Input format
	// simpleMatch := simpleActionRegex.FindStringSubmatch(llmOutput)
	// if simpleMatch != nil {
	// 	 toolName := strings.TrimSpace(simpleMatch[2])
	// 	 toolInputStr := strings.TrimSpace(simpleMatch[3])
	// 	 log.Printf("[ReActParser] Found Simple Action: Tool=%s, Input=%s", toolName, toolInputStr)
	// 	 toolInput, err := parseToolInput(toolInputStr)
	// 	 if err != nil {
	// 	 	 // Return parsing error to be handled by executor
	// 	 	 return nil, nil, fmt.Errorf("failed to parse simple action input ", toolInputStr, ": %w", err)
	// 	 }
	// 	 return &AgentAction{
	// 	 	 Tool:      toolName,
	// 	 	 ToolInput: toolInput,
	// 	 	 Log:       llmOutput, // Include the full thought process
	// 	 }, nil, nil
	// }

	// Check for JSON Action block format
	// jsonMatch := jsonActionRegex.FindStringSubmatch(llmOutput)
	// if jsonMatch != nil {
	// 	 jsonStr := strings.TrimSpace(jsonMatch[1])
	// 	 log.Printf("[ReActParser] Found JSON Action Block: %s", jsonStr)
	// 	 var jsonAction struct {
	// 	 	 Action      string `json:"action"`
	// 	 	 ActionInput any    `json:"action_input"` // Use any to handle string or object
	// 	 }
	// 	 err := json.Unmarshal([]byte(jsonStr), &jsonAction)
	// 	 if err != nil {
	// 	 	 return nil, nil, fmt.Errorf("failed to parse JSON action block ", jsonStr, ": %w", err)
	// 	 }
	// 	 // Further parse ActionInput if it's a string that might be JSON
	// 	 toolInput := jsonAction.ActionInput
	// 	 if inputStr, ok := toolInput.(string); ok {
	// 	 	 parsedInput, parseErr := parseToolInput(inputStr)
	// 	 	 if parseErr == nil {
	// 	 	 	 toolInput = parsedInput // Use parsed version if successful
	// 	 	 }
	// 	 	 // If parseErr is not nil, keep the original string input
	// 	 }

	// 	 return &AgentAction{
	// 	 	 Tool:      jsonAction.Action,
	// 	 	 ToolInput: toolInput,
	// 	 	 Log:       llmOutput,
	// 	 }, nil, nil
	// }

	// If no structured action or final answer is found, return parsing error.
	// log.Printf("[ReActParser] No action or final answer found in output.")
	// return nil, nil, fmt.Errorf("could not parse LLM output into a valid action or final answer: %s", llmOutput)
	return nil, nil, errors.New("ReActAgent parseOutput needs completion") // Placeholder
}

// parseToolInput attempts to parse the tool input string.
// It first tries to unmarshal as a JSON object (map[string]any).
// If that fails, it returns the original string, potentially unquoting it.
func parseToolInput(inputStr string) (any, error) {
	var jsonInput map[string]any
	// err := json.Unmarshal([]byte(inputStr), &jsonInput)
	// if err == nil {
	// 	 return jsonInput, nil // Successfully parsed as JSON map
	// }

	// If not a valid JSON map, treat as a simple string input.
	// Remove potential surrounding quotes if it's just a string.
	// if strings.HasPrefix(inputStr, `"`) && strings.HasSuffix(inputStr, `"`) {
	// 	 unquotedStr, err := strconv.Unquote(inputStr)
	// 	 if err == nil {
	// 	 	 return unquotedStr, nil
	// 	 }
	// 	 // If unquoting fails, return the original string minus the quotes
	// 	 return inputStr[1 : len(inputStr)-1], nil
	// }

	// Return the original string if it wasn't JSON and wasn't quoted
	// return inputStr, nil
	return nil, errors.New("parseToolInput needs completion") // Placeholder
}

// Ensure ReActAgent implements Agent interface
var _ Agent = (*ReActAgent)(nil)

