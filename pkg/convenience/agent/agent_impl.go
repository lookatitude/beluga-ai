package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// convenienceAgent implements the Agent interface.
type convenienceAgent struct {
	name         string
	systemPrompt string
	llm          llmsiface.LLM
	chatModel    llmsiface.ChatModel
	memory       memoryiface.Memory
	tools        []core.Tool
	maxTurns     int
	timeout      time.Duration
	verbose      bool
	agentType    string
	metrics      *Metrics
}

// Run executes the agent with a simple string input.
func (a *convenienceAgent) Run(ctx context.Context, input string) (string, error) {
	const op = "Run"

	// Apply timeout if configured
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		defer cancel()
	}

	// Start tracing
	start := time.Now()
	ctx, span := a.metrics.StartRunSpan(ctx, a.name, "run")
	if span != nil {
		defer span.End()
	}

	// Load memory if available
	var memoryVars map[string]any
	if a.memory != nil {
		var err error
		memoryVars, err = a.memory.LoadMemoryVariables(ctx, map[string]any{"input": input})
		if err != nil {
			a.metrics.RecordRun(ctx, a.name, time.Since(start), false)
			return "", NewError(op, ErrCodeExecution, fmt.Errorf("failed to load memory: %w", err))
		}
	}

	// Build messages
	messages := a.buildMessages(input, memoryVars)

	// Execute based on what we have available
	var response string
	var err error

	if a.chatModel != nil {
		// Use ChatModel for message-based interaction
		respMsg, genErr := a.chatModel.Generate(ctx, messages)
		if genErr != nil {
			a.metrics.RecordRun(ctx, a.name, time.Since(start), false)
			return "", NewError(op, ErrCodeExecution, genErr)
		}
		response = respMsg.GetContent()
	} else if a.llm != nil {
		// Use LLM for simple text-based interaction
		prompt := a.formatPromptFromMessages(messages)
		result, invokeErr := a.llm.Invoke(ctx, prompt)
		if invokeErr != nil {
			a.metrics.RecordRun(ctx, a.name, time.Since(start), false)
			return "", NewError(op, ErrCodeExecution, invokeErr)
		}
		// Handle various return types
		switch v := result.(type) {
		case string:
			response = v
		case schema.Message:
			response = v.GetContent()
		default:
			response = fmt.Sprintf("%v", result)
		}
	} else {
		a.metrics.RecordRun(ctx, a.name, time.Since(start), false)
		return "", NewError(op, ErrCodeMissingLLM, ErrMissingLLM)
	}

	// Save to memory if available
	if a.memory != nil {
		if err = a.memory.SaveContext(ctx,
			map[string]any{"input": input},
			map[string]any{"output": response},
		); err != nil {
			// Log but don't fail on memory save errors
			if a.verbose {
				fmt.Printf("Warning: failed to save to memory: %v\n", err)
			}
		}
	}

	a.metrics.RecordRun(ctx, a.name, time.Since(start), true)
	return response, nil
}

// RunWithInputs executes the agent with a map of inputs.
func (a *convenienceAgent) RunWithInputs(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	const op = "RunWithInputs"

	// Apply timeout if configured
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		defer cancel()
	}

	// Start tracing
	start := time.Now()
	ctx, span := a.metrics.StartRunSpan(ctx, a.name, "run_with_inputs")
	if span != nil {
		defer span.End()
	}

	// Extract input string from inputs
	var inputStr string
	if input, ok := inputs["input"].(string); ok {
		inputStr = input
	} else if query, ok := inputs["query"].(string); ok {
		inputStr = query
	} else {
		// Try to get first string value
		for _, v := range inputs {
			if s, ok := v.(string); ok {
				inputStr = s
				break
			}
		}
	}

	// Load memory if available
	var memoryVars map[string]any
	if a.memory != nil {
		var err error
		memoryVars, err = a.memory.LoadMemoryVariables(ctx, inputs)
		if err != nil {
			a.metrics.RecordRun(ctx, a.name, time.Since(start), false)
			return nil, NewError(op, ErrCodeExecution, fmt.Errorf("failed to load memory: %w", err))
		}
	}

	// Merge memory variables with inputs
	mergedInputs := make(map[string]any)
	for k, v := range inputs {
		mergedInputs[k] = v
	}
	for k, v := range memoryVars {
		mergedInputs[k] = v
	}

	// Build messages
	messages := a.buildMessages(inputStr, memoryVars)

	// Execute based on what we have available
	var response string
	var err error

	if a.chatModel != nil {
		respMsg, genErr := a.chatModel.Generate(ctx, messages)
		if genErr != nil {
			a.metrics.RecordRun(ctx, a.name, time.Since(start), false)
			return nil, NewError(op, ErrCodeExecution, genErr)
		}
		response = respMsg.GetContent()
	} else if a.llm != nil {
		prompt := a.formatPromptFromMessages(messages)
		result, invokeErr := a.llm.Invoke(ctx, prompt)
		if invokeErr != nil {
			a.metrics.RecordRun(ctx, a.name, time.Since(start), false)
			return nil, NewError(op, ErrCodeExecution, invokeErr)
		}
		switch v := result.(type) {
		case string:
			response = v
		case schema.Message:
			response = v.GetContent()
		default:
			response = fmt.Sprintf("%v", result)
		}
	} else {
		a.metrics.RecordRun(ctx, a.name, time.Since(start), false)
		return nil, NewError(op, ErrCodeMissingLLM, ErrMissingLLM)
	}

	// Build output
	outputs := map[string]any{
		"output": response,
	}

	// Save to memory if available
	if a.memory != nil {
		if err = a.memory.SaveContext(ctx, inputs, outputs); err != nil {
			if a.verbose {
				fmt.Printf("Warning: failed to save to memory: %v\n", err)
			}
		}
	}

	a.metrics.RecordRun(ctx, a.name, time.Since(start), true)
	return outputs, nil
}

// Stream executes the agent and returns a channel that streams response chunks.
func (a *convenienceAgent) Stream(ctx context.Context, input string) (<-chan StreamChunk, error) {
	const op = "Stream"

	// Apply timeout if configured
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		// Note: cancel will be called when the channel is closed
		defer cancel()
	}

	// Start tracing
	ctx, span := a.metrics.StartRunSpan(ctx, a.name, "stream")
	if span != nil {
		defer span.End()
	}

	// Load memory if available
	var memoryVars map[string]any
	if a.memory != nil {
		var err error
		memoryVars, err = a.memory.LoadMemoryVariables(ctx, map[string]any{"input": input})
		if err != nil {
			return nil, NewError(op, ErrCodeExecution, fmt.Errorf("failed to load memory: %w", err))
		}
	}

	// Build messages
	messages := a.buildMessages(input, memoryVars)

	// Create output channel
	out := make(chan StreamChunk)

	// Check if ChatModel supports streaming
	if a.chatModel != nil {
		// Try to stream
		streamCh, err := a.chatModel.StreamChat(ctx, messages)
		if err != nil {
			close(out)
			return nil, NewError(op, ErrCodeStreaming, err)
		}

		// Forward chunks
		go func() {
			defer close(out)

			var fullResponse string
			var fullResponseSb266 strings.Builder
			for chunk := range streamCh {
				if chunk.Err != nil {
					out <- StreamChunk{Error: chunk.Err}
					return
				}
				fullResponseSb266.WriteString(chunk.Content)
				a.metrics.RecordStreamChunk(ctx, a.name)
				out <- StreamChunk{Content: chunk.Content}
			}
			fullResponse += fullResponseSb266.String()

			// Send done marker
			out <- StreamChunk{Done: true}

			// Save to memory if available
			if a.memory != nil {
				if err := a.memory.SaveContext(ctx,
					map[string]any{"input": input},
					map[string]any{"output": fullResponse},
				); err != nil && a.verbose {
					fmt.Printf("Warning: failed to save to memory: %v\n", err)
				}
			}
		}()

		return out, nil
	}

	// Fallback: non-streaming execution, return single chunk
	go func() {
		defer close(out)

		response, err := a.Run(ctx, input)
		if err != nil {
			out <- StreamChunk{Error: err}
			return
		}

		out <- StreamChunk{Content: response, Done: true}
	}()

	return out, nil
}

// GetName returns the name of the agent.
func (a *convenienceAgent) GetName() string {
	return a.name
}

// GetTools returns the tools available to the agent.
func (a *convenienceAgent) GetTools() []core.Tool {
	return a.tools
}

// GetMemory returns the memory instance if configured.
func (a *convenienceAgent) GetMemory() memoryiface.Memory {
	return a.memory
}

// Shutdown gracefully stops the agent and releases resources.
func (a *convenienceAgent) Shutdown() error {
	// Clear memory if present
	if a.memory != nil {
		if err := a.memory.Clear(context.Background()); err != nil {
			return NewError("Shutdown", ErrCodeShutdown, err)
		}
	}
	return nil
}

// buildMessages creates the message list for the LLM call.
func (a *convenienceAgent) buildMessages(input string, memoryVars map[string]any) []schema.Message {
	var messages []schema.Message

	// Add system prompt if configured
	if a.systemPrompt != "" {
		messages = append(messages, schema.NewSystemMessage(a.systemPrompt))
	}

	// Add memory messages if available
	if memoryVars != nil {
		if history, ok := memoryVars["history"].([]schema.Message); ok {
			messages = append(messages, history...)
		}
	}

	// Add the user input
	messages = append(messages, schema.NewHumanMessage(input))

	return messages
}

// formatPromptFromMessages creates a text prompt from messages for non-chat LLMs.
func (a *convenienceAgent) formatPromptFromMessages(messages []schema.Message) string {
	var prompt string
	for _, msg := range messages {
		switch msg.GetType() {
		case schema.RoleSystem:
			prompt += fmt.Sprintf("System: %s\n\n", msg.GetContent())
		case schema.RoleHuman:
			prompt += fmt.Sprintf("Human: %s\n\n", msg.GetContent())
		case schema.RoleAssistant:
			prompt += fmt.Sprintf("Assistant: %s\n\n", msg.GetContent())
		default:
			prompt += fmt.Sprintf("%s: %s\n\n", msg.GetType(), msg.GetContent())
		}
	}
	prompt += "Assistant: "
	return prompt
}

// Ensure convenienceAgent implements Agent interface.
var _ Agent = (*convenienceAgent)(nil)
