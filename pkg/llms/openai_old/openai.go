// Package openai provides an implementation of the llms.ChatModel interface
// using the OpenAI API (including compatible APIs like Azure OpenAI).
package openai

import (
	"context"
	// "encoding/json" // Removed unused import
	"errors"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClientOption is a function type for setting options on the OpenAIChat client.
type OpenAIClientOption func(*OpenAIChat)

// WithBaseURL sets a custom base URL for the OpenAI client (for Azure, proxies, etc.).
func WithBaseURL(baseURL string) OpenAIClientOption {
	return func(o *OpenAIChat) {
		o.baseURL = baseURL
	}
}

// WithModel sets the default model name for the client.
func WithModel(modelName string) OpenAIClientOption {
	return func(o *OpenAIChat) {
		o.modelName = modelName
	}
}

// WithDefaultRequest sets default parameters for ChatCompletionRequest.
func WithDefaultRequest(req openai.ChatCompletionRequest) OpenAIClientOption {
	return func(o *OpenAIChat) {
		o.defaultRequest = req
	}
}

// WithMaxConcurrentBatches sets the maximum number of concurrent requests in Batch.
func WithMaxConcurrentBatches(n int) OpenAIClientOption {
	return func(o *OpenAIChat) {
		if n > 0 {
			o.maxConcurrentBatches = n
		}
	}
}

// OpenAIChat represents a chat model client for OpenAI compatible APIs.
type OpenAIChat struct {
	client               *openai.Client
	modelName            string
	baseURL              string // Optional custom base URL
	defaultRequest       openai.ChatCompletionRequest
	boundTools           []openai.Tool
	maxConcurrentBatches int
}

// NewOpenAIChat creates a new OpenAI chat client.
// It requires an API key and accepts functional options for customization.
func NewOpenAIChat(apiKey string, options ...OpenAIClientOption) (*OpenAIChat, error) {
	if apiKey == "" {
		return nil, errors.New("OpenAI API key cannot be empty")
	}

	o := &OpenAIChat{
		modelName:            openai.GPT3Dot5Turbo, // Default model
		maxConcurrentBatches: 5,                    // Default concurrency
		// Initialize defaultRequest with model, other fields are zero/nil
		defaultRequest: openai.ChatCompletionRequest{},
	}

	// Apply functional options
	for _, opt := range options {
		opt(o)
	}

	// Set model in default request if not already set by WithDefaultRequest
	if o.defaultRequest.Model == "" {
		o.defaultRequest.Model = o.modelName
	}

	// Configure OpenAI client
	config := openai.DefaultConfig(apiKey)
	if o.baseURL != "" {
		config.BaseURL = o.baseURL
	}
	o.client = openai.NewClientWithConfig(config)

	return o, nil
}

// mapMessages converts Beluga-ai schema messages to OpenAI chat messages.
func mapMessages(messages []schema.Message) ([]openai.ChatCompletionMessage, error) {
	openAIMessages := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		chatMsg := openai.ChatCompletionMessage{
			Content: msg.GetContent(),
		}

		switch m := msg.(type) {
		case *schema.SystemMessage:
			chatMsg.Role = openai.ChatMessageRoleSystem
		case *schema.HumanMessage:
			chatMsg.Role = openai.ChatMessageRoleUser
		case *schema.AIMessage:
			chatMsg.Role = openai.ChatMessageRoleAssistant
			if len(m.ToolCalls) > 0 {
				chatMsg.ToolCalls = make([]openai.ToolCall, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					chatMsg.ToolCalls[i] = openai.ToolCall{
						ID:   tc.ID,
						Type: openai.ToolTypeFunction, // Assuming only function tools for now
						Function: openai.FunctionCall{
							Name:      tc.Name,
							Arguments: tc.Arguments,
						},
					}
				}
				// Per OpenAI API: If tool_calls are present, content is optional.
				// We keep the content if it exists, but clear it if it's empty and there are tool calls.
				if chatMsg.Content == "" {
					chatMsg.Content = "" // Explicitly empty or nil? API expects string.
				}
			}
		case *schema.ToolMessage:
			chatMsg.Role = openai.ChatMessageRoleTool
			chatMsg.ToolCallID = m.ToolCallID
		default:
			// Skip generic or unknown message types
			log.Printf("Warning: Skipping message of unknown type %T for OpenAI API call.\n", msg)
			continue
		}
		openAIMessages = append(openAIMessages, chatMsg)
	}
	if len(openAIMessages) == 0 {
		return nil, errors.New("no valid messages provided for OpenAI conversion")
	}
	return openAIMessages, nil
}

// applyOptions converts core.Option into OpenAI ChatCompletionRequest fields,
// layering them over the default request settings.
func applyOptions(defaults openai.ChatCompletionRequest, options ...core.Option) openai.ChatCompletionRequest {
	// Start with default request
	req := defaults

	// Parse core.Option into a map
	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}

	// Override defaults with call-specific options from the config map
	if model, ok := config["model_name"].(string); ok {
		req.Model = model
	}
	if temp, ok := config["temperature"].(float32); ok {
		req.Temperature = temp
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		req.MaxTokens = maxTokens
	}
	if stops, ok := config["stop_sequences"].([]string); ok {
		req.Stop = stops
	}
	if topP, ok := config["top_p"].(float32); ok {
		req.TopP = topP
	}
	if presPenalty, ok := config["presence_penalty"].(float32); ok {
		req.PresencePenalty = presPenalty
	}
	if freqPenalty, ok := config["frequency_penalty"].(float32); ok {
		req.FrequencyPenalty = freqPenalty
	}
	if logitBias, ok := config["logit_bias"].(map[string]int); ok {
		req.LogitBias = logitBias
	}
	if user, ok := config["user"].(string); ok {
		req.User = user
	}
	if n, ok := config["n"].(int); ok {
		req.N = n
	}
	if seed, ok := config["seed"].(int); ok {
		// Seed needs to be pointer in openai client
		seedPtr := seed
		req.Seed = &seedPtr
	}
	if respFormat, ok := config["response_format"].(string); ok {
		if respFormat == "json_object" {
			req.ResponseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
		} else {
			req.ResponseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeText}
		}
	}

	// Note: TopK is not directly supported by the OpenAI API.
	if _, ok := config["top_k"]; ok {
		log.Println("Warning: 'top_k' option is not directly supported by the OpenAI API and will be ignored.")
	}

	return req
}

// mapTool converts a Beluga-ai tool definition to an OpenAI tool definition.
func mapTool(tool tools.Tool) (openai.Tool, error) {
	toolDef := tool.Definition()
	paramsSchema := toolDef.InputSchema // OpenAI expects JSON schema object for Parameters

	// Ensure paramsSchema is a map[string]any
	schemaMap, ok := paramsSchema.(map[string]any)
	if !ok || paramsSchema == nil {
		// If schema is nil or not a map, create a minimal valid schema (empty object)
		schemaMap = map[string]any{"type": "object", "properties": map[string]any{}}
		log.Printf("Warning: Tool %s has invalid InputSchema, using empty object schema.", toolDef.Name)
	}

	// Ensure the schema has a top-level type: object if not present and properties exist
	if typeVal, typeOk := schemaMap["type"]; !typeOk || typeVal == nil {
		// Check if properties exist
		if _, propsOk := schemaMap["properties"].(map[string]any); propsOk {
			schemaMap["type"] = "object"
		} else {
			// If no type and no properties, it might be an invalid schema or truly empty.
			// Defaulting to an empty object schema is a safe bet.
			log.Printf("Warning: Tool %s schema lacks 'type' and 'properties', ensuring empty object schema.", toolDef.Name)
			schemaMap = map[string]any{"type": "object", "properties": map[string]any{}}
		}
	}

	funcDef := openai.FunctionDefinition{
		Name:        toolDef.Name,
		Description: toolDef.Description,
		Parameters:  schemaMap, // Use the validated schemaMap
	}

	return openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &funcDef,
	}, nil
}

// Generate implements the llms.ChatModel interface.
func (o *OpenAIChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	openAIMessages, err := mapMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for OpenAI: %w", err)
	}

	req := applyOptions(o.defaultRequest, options...)
	req.Messages = openAIMessages
	req.Stream = false // Ensure stream is false for Generate

	// Add bound tools if any
	if len(o.boundTools) > 0 {
		req.Tools = o.boundTools
		// TODO: Add ToolChoice option if needed (e.g., force a specific tool)
	}

	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("openai returned no choices")
	}

	choice := resp.Choices[0].Message
	aiMsg := schema.NewAIMessage(choice.Content)

	// Handle tool calls in response
	if len(choice.ToolCalls) > 0 {
		aiMsg.ToolCalls = make([]schema.ToolCall, len(choice.ToolCalls))
		for i, tc := range choice.ToolCalls {
			if tc.Type != openai.ToolTypeFunction {
				log.Printf("Warning: Skipping unsupported tool type 	%s	 in OpenAI response\n", tc.Type)
				continue
			}
			aiMsg.ToolCalls[i] = schema.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}
	}

	// Add usage information
	if resp.Usage.PromptTokens > 0 || resp.Usage.CompletionTokens > 0 || resp.Usage.TotalTokens > 0 {
		aiMsg.AdditionalArgs["usage"] = map[string]int{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		}
	}

	return aiMsg, nil
}

// StreamChat implements the llms.ChatModel interface.
func (o *OpenAIChat) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llms.AIMessageChunk, error) {
	openAIMessages, err := mapMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for OpenAI stream: %w", err)
	}

	req := applyOptions(o.defaultRequest, options...)
	req.Messages = openAIMessages
	req.Stream = true

	// Add bound tools if any
	if len(o.boundTools) > 0 {
		req.Tools = o.boundTools
		// TODO: Add ToolChoice option if needed
	}

	stream, err := o.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion stream creation failed: %w", err)
	}

	chunkChan := make(chan llms.AIMessageChunk)

	go func() {
		defer close(chunkChan)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return // Stream finished successfully
			}
			if err != nil {
				errChunk := llms.AIMessageChunk{Err: fmt.Errorf("openai stream error: %w", err)}
				select {
				case chunkChan <- errChunk:
				case <-ctx.Done(): // Avoid blocking if context is cancelled
				}
				return
			}

			if len(response.Choices) == 0 {
				continue // Should not happen in practice, but good to guard
			}

			delta := response.Choices[0].Delta
			chunk := llms.AIMessageChunk{Content: delta.Content}

			// Handle tool call chunks
			if len(delta.ToolCalls) > 0 {
				chunk.ToolCallChunks = make([]schema.ToolCallChunk, len(delta.ToolCalls))
				for i, tcChunk := range delta.ToolCalls {
					if tcChunk.Type != openai.ToolTypeFunction {
						log.Printf("Warning: Skipping unsupported tool type 	%s	 in OpenAI stream chunk\n", tcChunk.Type)
						continue
					}
					// Corrected pointer assignments based on compiler errors
					choiceIndex := response.Choices[0].Index
					funcName := tcChunk.Function.Name
					chunk.ToolCallChunks[i] = schema.ToolCallChunk{
						Index:        &choiceIndex, 
						ID:           tcChunk.ID,
						Name:         &funcName,    
						Arguments:    tcChunk.Function.Arguments,
					}
				}
			}

			// TODO: Add usage info if available in stream completion response?
			// chunk.Usage = response.Usage

			select {
			case chunkChan <- chunk:
			case <-ctx.Done():
				return // Stop streaming if context is cancelled
			}
		}
	}()

	return chunkChan, nil
}

// BindTools implements the llms.ChatModel interface.
// It creates a *new* OpenAIChat instance with the tools bound.
func (o *OpenAIChat) BindTools(toolsToBind []tools.Tool) llms.ChatModel {
	newTools := make([]openai.Tool, 0, len(toolsToBind)) // Initialize with 0 length, but capacity
	for _, t := range toolsToBind {
		mappedTool, err := mapTool(t)
		if err != nil {
			// Log the error and skip this tool
			log.Printf("Error mapping tool 	%s	 for OpenAI binding: %v. Skipping tool.\n", t.Definition().Name, err)
			continue // Or return an error? Returning a modified model seems more aligned
		}
		newTools = append(newTools, mappedTool) // Append successfully mapped tools
	}

	// Create a shallow copy and update the tools
	newO := *o
	newO.boundTools = newTools

	// Update default request to include tools
	// Note: This modifies the default request for the *new* instance.
	// If the original defaultRequest already had tools, they are replaced.
	newO.defaultRequest.Tools = newTools

	return &newO
}

// --- core.Runnable Implementation ---

// Invoke implements the core.Runnable interface.
func (o *OpenAIChat) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, fmt.Errorf("invalid input type for OpenAIChat invoke: %w", err)
	}
	return o.Generate(ctx, messages, options...)
}

// Batch implements the core.Runnable interface.
// Executes requests concurrently up to maxConcurrentBatches.
func (o *OpenAIChat) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	var firstErr error
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, o.maxConcurrentBatches)
	var mu sync.Mutex // To safely update firstErr and results slice

	for i, input := range inputs {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore slot

		go func(index int, currentInput any) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore slot

			// TODO: Consider adding core.WithCallback option propagation here if callbacks are implemented
			output, err := o.Invoke(ctx, currentInput, options...)

			mu.Lock()
			if err != nil {
				log.Printf("Error in OpenAI batch item %d: %v\n", index, err)
				if firstErr == nil {
					firstErr = fmt.Errorf("error processing batch item %d: %w", index, err)
				}
				results[index] = err // Store the error in the result slice
			} else {
				results[index] = output
			}
			mu.Unlock()
		}(i, input)
	}

	wg.Wait()
	// Return the first error encountered, but the results slice contains all outcomes.
	return results, firstErr
}

// Stream implements the core.Runnable interface.
func (o *OpenAIChat) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, fmt.Errorf("invalid input type for OpenAIChat stream: %w", err)
	}

	aiChunkChan, err := o.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	outChan := make(chan any, 1)
	go func() {
		defer close(outChan)
		for chunk := range aiChunkChan {
			select {
			case outChan <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()
	return outChan, nil
}

// Compile-time check to ensure OpenAIChat implements interfaces.
var _ llms.ChatModel = (*OpenAIChat)(nil)
var _ core.Runnable = (*OpenAIChat)(nil)

