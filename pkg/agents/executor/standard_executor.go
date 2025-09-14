// Package agents defines interfaces and implementations for autonomous agents
// that can reason, plan, and execute actions using tools.
package executor

import (
	"context"
	"errors"
	
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/base"
)

// AgentExecutorInput defines the expected input structure for the AgentExecutor.
type AgentExecutorInput map[string]any

// AgentExecutorOutput defines the expected output structure for the AgentExecutor.
type AgentExecutorOutput map[string]any

// AgentExecutorIntermediateStep represents a single step taken by the agent executor,
// containing the action taken and the resulting observation.
type AgentExecutorIntermediateStep struct {
	Action      base.AgentAction `json:"action"`
	Observation string      `json:"observation"`
}

// AgentExecutorOption is a function type for configuring a StandardAgentExecutor.
type AgentExecutorOption func(*StandardAgentExecutor)

// WithMaxIterations sets the maximum number of steps the agent can take.
func WithMaxIterations(max int) AgentExecutorOption {
	return func(e *StandardAgentExecutor) {
		// if max > 0 {
		// 	 e.MaxIterations = max
		// }
	}
}

// WithReturnIntermediateSteps configures the executor to return the sequence of actions and observations.
func WithReturnIntermediateSteps(ret bool) AgentExecutorOption {
	return func(e *StandardAgentExecutor) {
		// e.ReturnIntermediateSteps = ret
	}
}

// WithMemory sets the memory module for the agent executor.
func WithMemory(mem memory.Memory) AgentExecutorOption {
	return func(e *StandardAgentExecutor) {
		// e.Memory = mem
	}
}

// WithHandleParsingErrors configures how agent planning/parsing errors are handled.
// If true, errors are returned as observations; otherwise, execution stops.
func WithHandleParsingErrors(handle bool) AgentExecutorOption {
	return func(e *StandardAgentExecutor) {
		// e.HandleParsingErrors = handle
	}
}

// StandardAgentExecutor implements the AgentExecutor interface.
// It provides the standard loop for running an agent: Plan -> Action -> Observation -> Plan ...
type StandardAgentExecutor struct {
	Agent                 base.Agent        // The agent instance containing the planning logic.
	Tools                 []tools.Tool // The list of tools available to the agent during execution.
	Memory                memory.Memory // Optional: Memory module to manage conversation history.
	MaxIterations         int          // Maximum number of iterations (Plan -> Action -> Observation cycles) allowed.
	ReturnIntermediateSteps bool         // Whether to return the sequence of actions and observations.
	HandleParsingErrors   bool         // How to handle errors during agent action parsing (return as observation or fail).
	// TODO: Add CallbackManager/Tracer for logging/monitoring steps.
	toolMap map[string]tools.Tool // Internal map for efficient tool lookup by name.
}

// NewAgentExecutor creates a new StandardAgentExecutor.
// It takes the agent, a list of tools the agent can use, and functional options for configuration.
func NewAgentExecutor(agent base.Agent, agentTools []tools.Tool, options ...AgentExecutorOption) (*StandardAgentExecutor, error) {
	// When implementing, process the tools to check for duplicates
	toolMap := make(map[string]tools.Tool)
	for _, t := range agentTools {
		name := t.Name()
		// Check for duplicate tool names
		if _, exists := toolMap[name]; exists {
			// return nil, fmt.Errorf("duplicate tool name found: %s", name)
		}
		toolMap[name] = t
	}
	// }

	// Initialize executor with defaults
	// e := &StandardAgentExecutor{
	// 	 Agent:         agent,
	// 	 Tools:         agentTools,
	// 	 toolMap:       toolMap,
	// 	 MaxIterations: 15, // Default max iterations
	// 	 // Default HandleParsingErrors to false (fail fast)
	// 	 HandleParsingErrors: false,
	// }

	// // Apply functional options
	// for _, opt := range options {
	// 	 opt(e)
	// }

	// // Validate memory variables if memory is set
	// if e.Memory != nil {
	// 	 expectedKeys := e.Memory.MemoryVariables()
	// 	 agentInputKeys := agent.InputKeys() // Assuming agent provides its expected input keys
	// 	 keySet := make(map[string]struct{}, len(agentInputKeys))
	// 	 for _, k := range agentInputKeys {
	// 	 	 keySet[k] = struct{}{} 
	// 	 }
	// 	 for _, memKey := range expectedKeys {
	// 	 	 if _, ok := keySet[memKey]; !ok {
	// 	 	 	 return nil, fmt.Errorf("memory key \"%s\" is not an expected input key for the agent", memKey)
	// 	 	 }
	// 	 }
	// }

	// return e, nil
	return nil, errors.New("AgentExecutor implementation needs completion") // Placeholder
}

// prepareInputs merges user inputs with memory variables.
func (e *StandardAgentExecutor) prepareInputs(inputs AgentExecutorInput) (map[string]any, error) {
	// if e.Memory == nil {
	// 	 return inputs, nil
	// }

	// memoryVariables := e.Memory.MemoryVariables()
	// memoryData, err := e.Memory.LoadMemoryVariables(inputs) // Pass inputs for context if needed
	// if err != nil {
	// 	 return nil, fmt.Errorf("failed to load memory variables: %w", err)
	// }

	// // Check for overlap between user inputs and memory variables
	// mergedInputs := make(map[string]any)
	// for k, v := range inputs {
	// 	 mergedInputs[k] = v
	// }
	// for _, key := range memoryVariables {
	// 	 if _, exists := inputs[key]; exists {
	// 	 	 log.Printf("Warning: Input key \"%s\" overlaps with memory variable. Input value will be used.", key)
	// 	 }
	// 	 // Add memory data, potentially overwriting if warning above is ignored
	// 	 mergedInputs[key] = memoryData[key]
	// }
	// return mergedInputs, nil
	return inputs, nil // Placeholder
}

// saveContext saves the input and output of the current step to memory.
func (e *StandardAgentExecutor) saveContext(inputs map[string]any, outputs map[string]any) error {
	// if e.Memory == nil {
	// 	 return nil
	// }
	// return e.Memory.SaveContext(inputs, outputs)
	return nil // Placeholder
}

// handlePlanError formats an error encountered during planning/parsing as an observation.
func (e *StandardAgentExecutor) handlePlanError(err error) string {
	// You might want to customize this message based on the error type
	// return fmt.Sprintf("Error during planning/parsing: %v. Please try again.", err)
	return "" // Placeholder
}

// Invoke implements the core.Runnable interface for the AgentExecutor.
// It runs the agent execution loop with the given input.
func (e *StandardAgentExecutor) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// userInput, ok := input.(AgentExecutorInput)
	// if !ok {
	// 	 // Try converting from map[string]string
	// 	 if mapStrStr, okStr := input.(map[string]string); okStr {
	// 	 	 userInput = make(AgentExecutorInput, len(mapStrStr))
	// 	 	 for k, v := range mapStrStr {
	// 	 	 	 userInput[k] = v
	// 	 	 }
	// 	 } else {
	// 	 	 return nil, fmt.Errorf("input to AgentExecutor must be map[string]any or map[string]string, got %T", input)
	// 	 }
	// }

	// // Prepare inputs by merging with memory
	// currentInputs, err := e.prepareInputs(userInput)
	// if err != nil {
	// 	 return nil, err
	// }

	// intermediateSteps := []AgentExecutorIntermediateStep{}
	// iterations := 0

	// // Loop until the agent finishes, an error occurs, or max iterations are reached.
	// for e.MaxIterations <= 0 || iterations < e.MaxIterations {
	// 	 iterations++
	// 	 log.Printf("[AgentExecutor] Starting Step %d", iterations)

	// 	 // 1. Agent Plans:
	// 	 // Pass intermediate steps as part of the input map for the agent
	// 	 planInput := make(map[string]any)
	// 	 for k, v := range currentInputs {
	// 	 	 planInput[k] = v
	// 	 }
	// 	 // Add intermediate steps under a specific key, e.g., "agent_scratchpad"
	// 	 // The format depends on what the specific Agent implementation expects.
	// 	 // Example: Formatting as a string
	// 	 scratchpad := ""
	// 	 for _, step := range intermediateSteps {
	// 	 	 scratchpad += fmt.Sprintf("Action: %s\nInput: %v\nObservation: %s\n", step.Action.Tool, step.Action.ToolInput, step.Observation)
	// 	 }
	// 	 planInput["agent_scratchpad"] = scratchpad // Or use agent.ScratchpadKey()

	// 	 action, finish, err := e.Agent.Plan(ctx, planInput) // Agent now takes map input

	// 	 // Handle Planning/Parsing Errors
	// 	 if err != nil {
	// 	 	 if e.HandleParsingErrors {
	// 	 	 	 log.Printf("[AgentExecutor] Step %d: Agent planning/parsing error: %v. Handling as observation.", iterations, err)
	// 	 	 	 observation := e.handlePlanError(err)
	// 	 	 	 // Create a dummy action to associate the error observation with
	// 	 	 	 // This might need refinement depending on how agents handle parse errors.
	// 	 	 	 dummyAction := AgentAction{Log: fmt.Sprintf("Planning Error: %v", err)}
	// 	 	 	 intermediateSteps = append(intermediateSteps, AgentExecutorIntermediateStep{dummyAction, observation})
	// 	 	 	 // TODO: Log error via Callbacks/Tracing
	// 	 	 	 continue // Continue to the next planning iteration
	// 	 	 } else {
	// 	 	 	 // TODO: Log error via Callbacks/Tracing
	// 	 	 	 return nil, fmt.Errorf("agent planning failed on step %d: %w", iterations, err)
	// 	 	 }
	// 	 }

	// 	 // 2. Check for Finish State:
	// 	 if finish != nil {
	// 	 	 log.Printf("[AgentExecutor] Step %d: Agent Finish: %v", iterations, finish.ReturnValues)
	// 	 	 // TODO: Log finish via Callbacks/Tracing

	// 	 	 // Save final context to memory
	// 	 	 finalOutputs := finish.ReturnValues
	// 	 	 if err := e.saveContext(currentInputs, finalOutputs); err != nil {
	// 	 	 	 log.Printf("Warning: Failed to save final context to memory: %v", err)
	// 	 	 }

	// 	 	 // Prepare final result, potentially including intermediate steps
	// 	 	 result := finalOutputs
	// 	 	 if e.ReturnIntermediateSteps {
	// 	 	 	 result["intermediate_steps"] = intermediateSteps
	// 	 	 }
	// 	 	 return result, nil // Agent has finished.
	// 	 }

	// 	 // 3. Check for Action State:
	// 	 if action == nil || action.Tool == "" {
	// 	 	 // This should ideally be caught by HandleParsingErrors if it stems from parsing
	// 	 	 return nil, fmt.Errorf("agent plan on step %d returned neither valid action nor finish state", iterations)
	// 	 }

	// 	 // TODO: Log Action via Callbacks/Tracing
	// 	 log.Printf("[AgentExecutor] Step %d: Action: Tool=%s, Input=%v", iterations, action.Tool, action.ToolInput)
	// 	 if action.Log != "" {
	// 	 	 log.Printf("[AgentExecutor] Step %d: Thought: %s", iterations, action.Log)
	// 	 }

	// 	 // 4. Execute Tool:
	// 	 var observation string
	// 	 toolToUse, exists := e.toolMap[action.Tool]
	// 	 if !exists {
	// 	 	 observation = fmt.Sprintf("Error: Tool \"%s\" not found. Available tools: %s", action.Tool, e.getToolNames())
	// 	 } else {
	// 	 	 // Execute the tool using Invoke.
	// 	 	 // Input preparation: Assume ToolInput is the correct type for the tool.
	// 	 	 // More robust handling might involve checking tool.InputSchema() if available.
	// 	 	 var toolInputForInvoke any = action.ToolInput

	// 	 	 // Example: If tool expects map[string]any but gets string, try JSON decode
	// 	 	 // if _, expectsMap := toolToUse.InputSchema(); expectsMap { // Hypothetical schema check
	// 	 	 // 	 if inputStr, isStr := action.ToolInput.(string); isStr {
	// 	 	 // 	 	 var decodedInput map[string]any
	// 	 	 // 	 	 if err := json.Unmarshal([]byte(inputStr), &decodedInput); err == nil {
	// 	 	 // 	 	 	 toolInputForInvoke = decodedInput
	// 	 	 // 	 	 } else {
	// 	 	 // 	 	 	 log.Printf("Warning: Tool %s expects map input, got string that failed JSON decode: %v. Passing raw string.", action.Tool, err)
	// 	 	 // 	 	 }
	// 	 	 // 	 }
	// 	 	 // }

	// 	 	 toolOutput, toolErr := toolToUse.Invoke(ctx, toolInputForInvoke)
	// 	 	 if toolErr != nil {
	// 	 	 	 // Handle Tool Execution Errors (similar to parsing errors, maybe different config flag?)
	// 	 	 	 log.Printf("[AgentExecutor] Step %d: Tool execution error for %s: %v", iterations, action.Tool, toolErr)
	// 	 	 	 // TODO: Log error via Callbacks/Tracing
	// 	 	 	 // Format error as observation by default
	// 	 	 	 observation = fmt.Sprintf("Error executing tool %s: %v", action.Tool, toolErr)
	// 	 	 	 // Optionally, allow stopping execution based on config
	// 	 	 	 // if !e.HandleToolExecutionErrors { return nil, toolErr }
	// 	 	 } else {
	// 	 	 	 // Convert tool output to string observation
	// 	 	 	 obsStr, ok := toolOutput.(string)
	// 	 	 	 if !ok {
	// 	 	 	 	 jsonBytes, jsonErr := json.Marshal(toolOutput)
	// 	 	 	 	 if jsonErr != nil {
	// 	 	 	 	 	 observation = fmt.Sprintf("Error: Tool %s returned unserializable type %T: %v", action.Tool, toolOutput, jsonErr)
	// 	 	 	 	 } else {
	// 	 	 	 	 	 observation = string(jsonBytes)
	// 	 	 	 	 }
	// 	 	 	 } else {
	// 	 	 	 	 observation = obsStr
	// 	 	 	 }
	// 	 	 }
	// 	 }

	// 	 // TODO: Log Observation via Callbacks/Tracing
	// 	 log.Printf("[AgentExecutor] Step %d: Observation: %s", iterations, observation)

	// 	 // 5. Record Step and Loop:
	// 	 intermediateSteps = append(intermediateSteps, AgentExecutorIntermediateStep{*action, observation})
	// }

	// // If loop completes without finishing, max iterations were reached.
	// // TODO: Log warning/error via Callbacks/Tracing
	// return nil, fmt.Errorf("agent stopped after reaching max iterations (%d)", e.MaxIterations)
	return nil, errors.New("AgentExecutor Invoke needs completion") // Placeholder
}

// Batch implements the core.Runnable interface for AgentExecutor.
// It runs the agent loop for multiple inputs.
func (e *StandardAgentExecutor) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	// Basic batch implementation by calling Invoke sequentially.
	// TODO: Consider parallel execution using goroutines and a semaphore based on a MaxConcurrency option.
	results := make([]any, len(inputs))
	var firstErr error
	for i, input := range inputs {
		// Note: Memory handling in sequential batch might lead to unexpected results
		// if the same memory instance is shared without clearing between runs.
		// Parallel execution would require careful memory scoping or cloning.
		result, err := e.Invoke(ctx, input, options...)
		if err != nil {
			results[i] = err
			if firstErr == nil {
				firstErr = err
			}
		} else {
			results[i] = result
		}
		// 	 log.Printf("Error in AgentExecutor batch item %d: %v", i, err)
		// 	 if firstErr == nil {
		// 	 	 firstErr = fmt.Errorf("error processing batch item %d: %w", i, err)
		// 	 }
		// 	 results[i] = err // Store the error itself
		// } else {
		// 	 results[i] = output
		// }
	}
	return results, firstErr
}

// StreamOutput defines the structure for events streamed by the AgentExecutor.
type StreamOutput struct {
	Step        *AgentExecutorIntermediateStep `json:"step,omitempty"`
	FinalOutput AgentExecutorOutput            `json:"final_output,omitempty"`
	Err         error                          `json:"-"` // Error encountered
}

// Stream implements the core.Runnable interface for AgentExecutor.
// It runs the agent execution loop and streams intermediate steps and the final result.
func (e *StandardAgentExecutor) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	// resultChan := make(chan any)

	// go func() {
	// 	 defer close(resultChan)

	// 	 userInput, ok := input.(AgentExecutorInput)
	// 	 if !ok {
	// 	 	 // Handle map[string]string conversion like in Invoke
	// 	 	 if mapStrStr, okStr := input.(map[string]string); okStr {
	// 	 	 	 userInput = make(AgentExecutorInput, len(mapStrStr))
	// 	 	 	 for k, v := range mapStrStr {
	// 	 	 	 	 userInput[k] = v
	// 	 	 	 }
	// 	 	 } else {
	// 	 	 	 resultChan <- StreamOutput{Err: fmt.Errorf("input to AgentExecutor must be map[string]any or map[string]string, got %T", input)}
	// 	 	 	 return
	// 	 	 }
	// 	 }

	// 	 currentInputs, err := e.prepareInputs(userInput)
	// 	 if err != nil {
	// 	 	 resultChan <- StreamOutput{Err: err}
	// 	 	 return
	// 	 }

	// 	 intermediateSteps := []AgentExecutorIntermediateStep{}
	// 	 iterations := 0

	// 	 for e.MaxIterations <= 0 || iterations < e.MaxIterations {
	// 	 	 iterations++
	// 	 	 log.Printf("[AgentExecutor-Stream] Starting Step %d", iterations)

	// 	 	 // Prepare plan input (including scratchpad)
	// 	 	 planInput := make(map[string]any)
	// 	 	 for k, v := range currentInputs {
	// 	 	 	 planInput[k] = v
	// 	 	 }
	// 	 	 scratchpad := ""
	// 	 	 for _, step := range intermediateSteps {
	// 	 	 	 scratchpad += fmt.Sprintf("Action: %s\nInput: %v\nObservation: %s\n", step.Action.Tool, step.Action.ToolInput, step.Observation)
	// 	 	 }
	// 	 	 planInput["agent_scratchpad"] = scratchpad

	// 	 	 action, finish, err := e.Agent.Plan(ctx, planInput)

	// 	 	 if err != nil {
	// 	 	 	 if e.HandleParsingErrors {
	// 	 	 	 	 log.Printf("[AgentExecutor-Stream] Step %d: Agent planning/parsing error: %v. Handling as observation.", iterations, err)
	// 	 	 	 	 observation := e.handlePlanError(err)
	// 	 	 	 	 dummyAction := AgentAction{Log: fmt.Sprintf("Planning Error: %v", err)}
	// 	 	 	 	 stepOutput := AgentExecutorIntermediateStep{dummyAction, observation}
	// 	 	 	 	 intermediateSteps = append(intermediateSteps, stepOutput)
	// 	 	 	 	 // Stream the error step
	// 	 	 	 	 select {
	// 	 	 	 	 case resultChan <- StreamOutput{Step: &stepOutput}:
	// 	 	 	 	 case <-ctx.Done():
	// 	 	 	 	 	 resultChan <- StreamOutput{Err: ctx.Err()}
	// 	 	 	 	 	 return
	// 	 	 	 	 }
	// 	 	 	 	 continue
	// 	 	 	 } else {
	// 	 	 	 	 resultChan <- StreamOutput{Err: fmt.Errorf("agent planning failed on step %d: %w", iterations, err)}
	// 	 	 	 	 return
	// 	 	 	 }
	// 	 	 }

	// 	 	 if finish != nil {
	// 	 	 	 log.Printf("[AgentExecutor-Stream] Step %d: Agent Finish: %v", iterations, finish.ReturnValues)
	// 	 	 	 finalOutputs := finish.ReturnValues
	// 	 	 	 if err := e.saveContext(currentInputs, finalOutputs); err != nil {
	// 	 	 	 	 log.Printf("Warning: Failed to save final context to memory: %v", err)
	// 	 	 	 }
	// 	 	 	 // Stream final output
	// 	 	 	 select {
	// 	 	 	 case resultChan <- StreamOutput{FinalOutput: finalOutputs}:
	// 	 	 	 case <-ctx.Done():
	// 	 	 	 	 // Send context error if cancelled before final output
	// 	 	 	 	 resultChan <- StreamOutput{Err: ctx.Err()}
	// 	 	 	 }
	// 	 	 	 return // Finished successfully
	// 	 	 }

	// 	 	 if action == nil || action.Tool == "" {
	// 	 	 	 resultChan <- StreamOutput{Err: fmt.Errorf("agent plan on step %d returned neither valid action nor finish state", iterations)}
	// 	 	 	 return
	// 	 	 }

	// 	 	 log.Printf("[AgentExecutor-Stream] Step %d: Action: Tool=%s, Input=%v", iterations, action.Tool, action.ToolInput)

	// 	 	 // Execute Tool (similar logic as Invoke)
	// 	 	 var observation string
	// 	 	 toolToUse, exists := e.toolMap[action.Tool]
	// 	 	 if !exists {
	// 	 	 	 observation = fmt.Sprintf("Error: Tool \"%s\" not found. Available tools: %s", action.Tool, e.getToolNames())
	// 	 	 } else {
	// 	 	 	 toolInputForInvoke := action.ToolInput
	// 	 	 	 toolOutput, toolErr := toolToUse.Invoke(ctx, toolInputForInvoke)
	// 	 	 	 if toolErr != nil {
	// 	 	 	 	 log.Printf("[AgentExecutor-Stream] Step %d: Tool execution error for %s: %v", iterations, action.Tool, toolErr)
	// 	 	 	 	 observation = fmt.Sprintf("Error executing tool %s: %v", action.Tool, toolErr)
	// 	 	 	 } else {
	// 	 	 	 	 // Convert tool output to string observation
	// 	 	 	 	 obsStr, ok := toolOutput.(string)
	// 	 	 	 	 if !ok {
	// 	 	 	 	 	 jsonBytes, jsonErr := json.Marshal(toolOutput)
	// 	 	 	 	 	 if jsonErr != nil {
	// 	 	 	 	 	 	 observation = fmt.Sprintf("Error: Tool %s returned unserializable type %T: %v", action.Tool, toolOutput, jsonErr)
	// 	 	 	 	 	 } else {
	// 	 	 	 	 	 	 observation = string(jsonBytes)
	// 	 	 	 	 	 }
	// 	 	 	 	 } else {
	// 	 	 	 	 	 observation = obsStr
	// 	 	 	 	 }
	// 	 	 	 }
	// 	 	 }

	// 	 	 log.Printf("[AgentExecutor-Stream] Step %d: Observation: %s", iterations, observation)

	// 	 	 // Record and Stream the step
	// 	 	 stepOutput := AgentExecutorIntermediateStep{*action, observation}
	// 	 	 intermediateSteps = append(intermediateSteps, stepOutput)
	// 	 	 select {
	// 	 	 case resultChan <- StreamOutput{Step: &stepOutput}:
	// 	 	 case <-ctx.Done():
	// 	 	 	 resultChan <- StreamOutput{Err: ctx.Err()}
	// 	 	 	 return
	// 	 	 }
	// 	 }

	// 	 // If loop finishes, max iterations were reached
	// 	 resultChan <- StreamOutput{Err: fmt.Errorf("agent stopped after reaching max iterations (%d)", e.MaxIterations)}
	// }()

	// return resultChan, nil
	return nil, errors.New("AgentExecutor Stream needs completion") // Placeholder
}

// getToolNames is a helper function to get a comma-separated list of tool names.
func (e *StandardAgentExecutor) getToolNames() string {
	// names := make([]string, 0, len(e.toolMap))
	// for name := range e.toolMap {
	// 	 names = append(names, name)
	// }
	// return strings.Join(names, ", ")
	return "" // Placeholder
}

// Compile-time checks to ensure implementation satisfies interfaces.
var _ core.Runnable = (*StandardAgentExecutor)(nil)

