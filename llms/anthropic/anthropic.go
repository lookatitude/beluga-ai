// Package anthropic provides an implementation of the llms.ChatModel interface
// using the Anthropic API (Claude models).
package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
)

// --- Helper config struct for NewAnthropicChat ---
type anthropicChatConfig struct {
	APIKey               string
	BaseURL              string
	APIVersion           string // Note: Anthropic Go SDK v0.2.0-beta.3 does not directly use an API version header in the same way; this might be for older custom setups or future use.
	ModelName            string
	DefaultRequest       anthropic.MessageNewParams // Changed from MessagesRequest
	MaxConcurrentBatches int
}

// AnthropicOption is a function type for setting options on the AnthropicChat client configuration.
type AnthropicOption func(*anthropicChatConfig)

// WithAnthropicAPIKey sets the API key.
func WithAnthropicAPIKey(apiKey string) AnthropicOption {
	return func(cfg *anthropicChatConfig) {
		cfg.APIKey = apiKey
	}
}

// WithAnthropicBaseURL sets the base URL.
func WithAnthropicBaseURL(baseURL string) AnthropicOption {
	return func(cfg *anthropicChatConfig) {
		cfg.BaseURL = baseURL
	}
}

// WithAnthropicAPIVersion sets the API version header.
// Note: The current SDK might not use this directly for client construction in the same way.
// It uses anthropic.WithAPIVersion for specific request headers if needed, but not typically for client init.
func WithAnthropicAPIVersion(version string) AnthropicOption {
	return func(cfg *anthropicChatConfig) {
		cfg.APIVersion = version
	}
}

// WithAnthropicModel sets the default model name.
func WithAnthropicModel(modelName string) AnthropicOption {
	return func(cfg *anthropicChatConfig) {
		cfg.ModelName = modelName
	}
}

// WithAnthropicDefaultRequest sets the default request parameters.
// Note: This replaces the entire default request struct.
func WithAnthropicDefaultRequest(req anthropic.MessageNewParams) AnthropicOption {
	return func(cfg *anthropicChatConfig) {
		cfg.DefaultRequest = req
	}
}

// WithAnthropicMaxConcurrentBatches sets the concurrency limit for Batch.
func WithAnthropicMaxConcurrentBatches(n int) AnthropicOption {
	return func(cfg *anthropicChatConfig) {
		// TODO: Implement concurrency limiting in Batch method
		// if n > 0 {
		// 	 cfg.MaxConcurrentBatches = n
		// }
	}
}

// --- End Options ---

// AnthropicChat represents a chat model client for the Anthropic API.
type AnthropicChat struct {
	client               *anthropic.Client
	modelName            string // Default model name
	defaultRequest       anthropic.MessageNewParams
	boundTools           []anthropic.ToolParam // Changed from ToolDefinition to ToolParam
	maxConcurrentBatches int
}

// Compile-time check to ensure AnthropicChat implements llms.ChatModel
var _ llms.ChatModel = (*AnthropicChat)(nil)

// NewAnthropicChat creates a new Anthropic chat client.
// It requires an API key (read from ANTHROPIC_API_KEY env var by default) and accepts functional options.
func NewAnthropicChat(options ...AnthropicOption) (*AnthropicChat, error) {
	cfg := &anthropicChatConfig{
		APIKey:               os.Getenv("ANTHROPIC_API_KEY"),
		BaseURL:              os.Getenv("ANTHROPIC_BASE_URL"),
		APIVersion:           os.Getenv("ANTHROPIC_API_VERSION"),
		ModelName:            string(anthropic.ModelClaude3Haiku), // Default to Haiku
		MaxConcurrentBatches: 5,
		DefaultRequest: anthropic.MessageNewParams{
			MaxTokens: anthropic.Int(1024),
		},
	}

	for _, opt := range options {
		opt(cfg)
	}

	clientOpts := []anthropic.ClientOption{}
	if cfg.APIKey != "" {
		clientOpts = append(clientOpts, anthropic.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		clientOpts = append(clientOpts, anthropic.WithBaseURL(cfg.BaseURL))
	}
	if cfg.APIVersion != "" {
		// The SDK v0.2.0-beta.3 uses anthropic.WithDefaultHeader for arbitrary headers.
		// If APIVersion is meant to be the "anthropic-version" header:
		clientOpts = append(clientOpts, anthropic.WithDefaultHeader("anthropic-version", cfg.APIVersion))
	}

	client := anthropic.NewClient(clientOpts...)

	ac := &AnthropicChat{
		client:               client,
		modelName:            cfg.ModelName,
		defaultRequest:       cfg.DefaultRequest,
		maxConcurrentBatches: cfg.MaxConcurrentBatches,
	}

	if ac.defaultRequest.Model == "" || ac.defaultRequest.Model == anthropic.MessageNewParamsModel("") {
		ac.defaultRequest.Model = anthropic.MessageNewParamsModel(ac.modelName)
	}

	return ac, nil
}

// mapMessagesAndExtractSystem converts Beluga-ai messages to Anthropic format.
func mapMessagesAndExtractSystem(messages []schema.Message) (anthropic.MessageNewParamsSystem, []anthropic.MessageParam, error) {
	var systemPrompt anthropic.MessageNewParamsSystem
	var anthropicMsgs []anthropic.MessageParam
	processedMessages := messages

	if len(messages) > 0 {
		if sysMsg, ok := messages[0].(*schema.SystemMessage); ok {
			systemPrompt = anthropic.MessageNewParamsSystem(sysMsg.GetContent()) // Direct string conversion
			processedMessages = messages[1:]
		}
	}

	anthropicMsgs = make([]anthropic.MessageParam, 0, len(processedMessages))
	for _, msg := range processedMessages {
		var contentBlocks []anthropic.ContentBlockParamUnion
		var role anthropic.MessageParamRole

		switch m := msg.(type) {
		case *schema.HumanMessage:
			role = anthropic.MessageParamRoleUser
			contentBlocks = append(contentBlocks, anthropic.NewContentBlockParam(anthropic.NewTextBlock(m.GetContent())))
		case *schema.AIMessage:
			role = anthropic.MessageParamRoleAssistant
			if m.GetContent() != "" {
				contentBlocks = append(contentBlocks, anthropic.NewContentBlockParam(anthropic.NewTextBlock(m.GetContent())))
			}
			for _, tc := range m.ToolCalls {
				var inputMap map[string]any
				if tc.Arguments != "" && tc.Arguments != "{}" && tc.Arguments != "null" {
					err := json.Unmarshal([]byte(tc.Arguments), &inputMap)
					if err != nil {
						log.Printf("Warning: Failed to unmarshal tool call arguments for %s: %v. Args: %s", tc.Name, err, tc.Arguments)
						continue
					}
				}
				contentBlocks = append(contentBlocks, anthropic.NewContentBlockParam(anthropic.ToolUseBlockParam{
					ID:    anthropic.String(tc.ID),
					Name:  anthropic.String(tc.Name),
					Input: inputMap,
				}))
			}
		case *schema.ToolMessage:
			role = anthropic.MessageParamRoleUser // Tool results are sent as user role
			contentBlocks = append(contentBlocks, anthropic.NewContentBlockParam(anthropic.ToolResultBlockParam{
				ToolUseID: anthropic.String(m.ToolCallID),
				Content:   []anthropic.ToolResultBlockParamContentUnion{anthropic.NewToolResultBlockParamContent(anthropic.NewTextBlock(m.GetContent()))},
				// IsError: anthropic.Bool(m.IsError), // Assuming schema.ToolMessage has IsError field
			}))
		case *schema.SystemMessage:
			log.Println("Warning: System message encountered in unexpected position, ignoring.")
			continue
		default:
			log.Printf("Warning: Skipping message of unknown type %T for Anthropic API call.\n", msg)
			continue
		}

		if len(contentBlocks) > 0 {
			anthropicMsgs = append(anthropicMsgs, anthropic.MessageParam{
				Role:    role,
				Content: contentBlocks,
			})
		} else if role == anthropic.MessageParamRoleAssistant && len(m.(*schema.AIMessage).ToolCalls) > 0 {
			// Handle assistant message that ONLY contains tool calls (no text content)
			anthropicMsgs = append(anthropicMsgs, anthropic.MessageParam{
				Role:    role,
				Content: contentBlocks, // Will contain only ToolUseBlockParam
			})
		} else {
			log.Printf("Warning: Skipping empty message conversion for type %T with role %s", msg, role)
		}
	}

	if len(anthropicMsgs) == 0 && systemPrompt == "" {
		return "", nil, errors.New("no valid messages provided for Anthropic conversion")
	}
	return systemPrompt, anthropicMsgs, nil
}

// applyAnthropicOptions converts core.Option into Anthropic MessageNewParams fields.
func applyAnthropicOptions(defaults anthropic.MessageNewParams, options ...core.Option) anthropic.MessageNewParams {
	req := defaults
	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}

	if model, ok := config["model_name"].(string); ok {
		req.Model = anthropic.MessageNewParamsModel(model)
	}
	if temp, ok := config["temperature"].(float64); ok {
		req.Temperature = anthropic.Float(temp) // SDK uses *float64
	} else if temp, ok := config["temperature"].(float32); ok {
		req.Temperature = anthropic.Float(float64(temp))
	}

	if maxTokens, ok := config["max_tokens"].(int); ok {
		req.MaxTokens = anthropic.Int(maxTokens)
	}
	if stops, ok := config["stop_sequences"].([]string); ok {
		req.StopSequences = stops
	}
	if topP, ok := config["top_p"].(float64); ok {
		req.TopP = anthropic.Float(topP)
	} else if topP, ok := config["top_p"].(float32); ok {
		req.TopP = anthropic.Float(float64(topP))
	}

	if topK, ok := config["top_k"].(int); ok {
		req.TopK = anthropic.Int(topK)
	}

	if choice, ok := config["tool_choice"]; ok {
		switch tc := choice.(type) {
		case string:
			if tc == "auto" {
				req.ToolChoice = anthropic.NewToolChoiceAuto()
			} else if tc == "any" {
				req.ToolChoice = anthropic.NewToolChoiceAny()
			} else {
				req.ToolChoice = anthropic.NewToolChoiceTool(tc)
			}
		case map[string]any:
			if typeVal, ok := tc["type"].(string); ok && typeVal == "tool" {
				if nameVal, ok := tc["name"].(string); ok {
					req.ToolChoice = anthropic.NewToolChoiceTool(nameVal)
				}
			}
		default:
			log.Printf("Warning: Unsupported tool_choice format: %T", choice)
		}
	}

	return req
}

// mapAnthropicTool converts a Beluga-ai tool definition to an Anthropic tool definition.
func mapAnthropicTool(tool tools.Tool) (anthropic.ToolParam, error) {
	schemaStr := tool.SchemaDefinition()
	var paramsSchema anthropic.JSONSchemaDefinition
	err := json.Unmarshal([]byte(schemaStr), &paramsSchema)
	if err != nil {
		if schemaStr == "" || schemaStr == "{}" || schemaStr == "null" {
			paramsSchema = anthropic.JSONSchemaDefinition{Type: anthropic.JSONSchemaTypeObject, Properties: map[string]anthropic.JSONSchemaDefinition{}}
			log.Printf("Warning: Tool 		%s		 has empty or invalid schema, using empty object schema.", tool.Name())
		} else {
			return anthropic.ToolParam{}, fmt.Errorf("failed to unmarshal tool schema for 		%s		: %w. Schema was: %s", tool.Name(), err, schemaStr)
		}
	}

	if paramsSchema.Type == "" {
		paramsSchema.Type = anthropic.JSONSchemaTypeObject // Default to object if not specified
	}
	if paramsSchema.Type == anthropic.JSONSchemaTypeObject && paramsSchema.Properties == nil {
		paramsSchema.Properties = map[string]anthropic.JSONSchemaDefinition{}
	}

	return anthropic.ToolParam{
		Name:        anthropic.String(tool.Name()),
		Description: anthropic.String(tool.Description()),
		InputSchema: paramsSchema,
	}, nil
}

// Generate implements the llms.ChatModel interface.
func (ac *AnthropicChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	systemPrompt, anthropicMessages, err := mapMessagesAndExtractSystem(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for Anthropic: %w", err)
	}

	req := applyAnthropicOptions(ac.defaultRequest, options...)
	req.Messages = anthropicMessages
	req.System = systemPrompt

	if len(ac.boundTools) > 0 {
		var toolUnionParams []anthropic.ToolUnionParam
		for _, t := range ac.boundTools {
			toolUnionParams = append(toolUnionParams, anthropic.ToolUnionParam{OfTool: &t})
		}
		req.Tools = toolUnionParams
		if req.ToolChoice == nil {
			req.ToolChoice = anthropic.NewToolChoiceAuto()
		}
	}

	if req.MaxTokens == nil || *req.MaxTokens == 0 {
		return nil, errors.New("MaxTokens must be set and non-zero for Anthropic Generate")
	}

	resp, err := ac.client.Messages.New(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("anthropic chat completion failed: %w", err)
	}

	var responseText string
	var toolCalls []schema.ToolCall

	for _, block := range resp.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			responseText += b.Text
		case anthropic.ToolUseBlock:
			argsBytes, _ := json.Marshal(b.Input)
			toolCalls = append(toolCalls, schema.ToolCall{
				ID:        *b.ID,
				Name:      *b.Name,
				Arguments: string(argsBytes),
			})
		}
	}

	aiMsg := schema.NewAIMessage(responseText)
	aiMsg.ToolCalls = toolCalls
	aiMsg.StopReason = string(resp.StopReason)
	if resp.Usage != nil {
		aiMsg.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  int(resp.Usage.InputTokens),
			"output_tokens": int(resp.Usage.OutputTokens),
		}
	}

	return aiMsg, nil
}

// StreamChat implements the llms.ChatModel interface with streaming.
func (ac *AnthropicChat) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan schema.ChatResponseChunk, error) {
	systemPrompt, anthropicMessages, err := mapMessagesAndExtractSystem(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for Anthropic streaming: %w", err)
	}

	req := applyAnthropicOptions(ac.defaultRequest, options...)
	req.Messages = anthropicMessages
	req.System = systemPrompt

	if len(ac.boundTools) > 0 {
		var toolUnionParams []anthropic.ToolUnionParam
		for _, t := range ac.boundTools {
			toolUnionParams = append(toolUnionParams, anthropic.ToolUnionParam{OfTool: &t})
		}
		req.Tools = toolUnionParams
		if req.ToolChoice == nil {
			req.ToolChoice = anthropic.NewToolChoiceAuto()
		}
	}

	if req.MaxTokens == nil || *req.MaxTokens == 0 {
		return nil, errors.New("MaxTokens must be set and non-zero for Anthropic StreamChat")
	}

	stream, err := ac.client.Messages.NewStreaming(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("anthropic stream creation failed: %w", err)
	}

	outputCh := make(chan schema.ChatResponseChunk)

	go func() {
		defer close(outputCh)
		defer stream.Close()

		currentMessage := anthropic.Message{}	// Accumulator for the full message
		var accumulatedToolCalls []schema.ToolCall

		for stream.Next() {
			event := stream.Current()
			err := currentMessage.Accumulate(event) // Accumulate into the SDK's message struct
			if err != nil {
				outputCh <- schema.ChatResponseChunk{Error: fmt.Errorf("error accumulating stream event: %w", err)}
				return
			}

			chunk := schema.ChatResponseChunk{}

			switch ev := event.AsAny().(type) {
			case anthropic.MessageStartEvent:
				chunk.Delta = "" // No text delta for start event
				if ev.Message.Usage != nil {
					chunk.AdditionalArgs = map[string]any{
						"usage_input_tokens": int(ev.Message.Usage.InputTokens),
					}
				}
			case anthropic.ContentBlockDeltaEvent:
				if textDelta, ok := ev.Delta.AsTextDelta(); ok {
					chunk.Delta = textDelta.Text
				}
			case anthropic.ContentBlockStopEvent:
				// This event signals the end of a content block. We can inspect the accumulated currentMessage here.
				// For tool calls, they appear fully formed in ContentBlockStartEvent or within the accumulated message.
			case anthropic.MessageDeltaEvent:
				// Contains stop reason and usage for the delta
				chunk.StopReason = string(ev.Delta.StopReason)
				if ev.Usage != nil {
					chunk.AdditionalArgs = map[string]any{
						"usage_output_tokens_delta": int(ev.Usage.OutputTokens),
					}
				}
			case anthropic.MessageStopEvent:
				// Final event, contains overall stop reason and final usage.
				// The accumulated `currentMessage` should be complete here.
				chunk.StopReason = string(currentMessage.StopReason)
				if currentMessage.Usage != nil {
					chunk.AdditionalArgs = map[string]any{
						"usage_input_tokens":  int(currentMessage.Usage.InputTokens),
						"usage_output_tokens": int(currentMessage.Usage.OutputTokens),
					}
				}
				// Extract final tool calls from the accumulated message
				for _, block := range currentMessage.Content {
					if toolUseBlock, ok := block.AsToolUseBlock(); ok {
						argsBytes, _ := json.Marshal(toolUseBlock.Input)
						found := false
						for i, tc := range accumulatedToolCalls {
							if tc.ID == *toolUseBlock.ID {
								accumulatedToolCalls[i].Arguments = string(argsBytes) // Update if ID exists
								found = true
								break
							}
						}
						if !found {
							accumulatedToolCalls = append(accumulatedToolCalls, schema.ToolCall{
								ID:        *toolUseBlock.ID,
								Name:      *toolUseBlock.Name,
								Arguments: string(argsBytes),
							})
						}
					}
				}
				chunk.ToolCalls = accumulatedToolCalls

			case anthropic.ContentBlockStartEvent:
				if toolUseBlock, ok := ev.ContentBlock.AsToolUseBlock(); ok {
					// A new tool use block has started. Capture its ID and Name.
					// Arguments will come in ContentBlockDeltaEvents.
					// For streaming, we might send partial tool calls or accumulate.
					// Let's send a new tool call placeholder.
					partialToolCall := schema.ToolCall{
						ID:   *toolUseBlock.ID,
						Name: *toolUseBlock.Name,
						// Arguments will be filled by subsequent delta events if applicable
					}
					chunk.ToolCalls = []schema.ToolCall{partialToolCall}
					// Add to accumulated list for final message reconstruction
					found := false
					for _, tc := range accumulatedToolCalls {
						if tc.ID == partialToolCall.ID {
							found = true
							break
						}
					}
					if !found {
						accumulatedToolCalls = append(accumulatedToolCalls, partialToolCall)
					}
				}
			default:
				// Other event types like PingEvent can be ignored for chat response chunks
				// log.Printf("Anthropic stream: unhandled event type %T\n", ev)
				continue // Don't send an empty chunk for unhandled events
			}

			// Send the chunk if it has content or relevant info
			if chunk.Delta != "" || len(chunk.ToolCalls) > 0 || chunk.StopReason != "" || chunk.Error != nil || len(chunk.AdditionalArgs) > 0 {
				outputCh <- chunk
			}
		}

		if stream.Err() != nil {
			finalErr := stream.Err()
			if finalErr != io.EOF { // Don't send EOF as an error chunk
				outputCh <- schema.ChatResponseChunk{Error: fmt.Errorf("anthropic stream error: %w", finalErr)}
			}
		}
	}()

	return outputCh, nil
}

// BindTools implements the llms.ChatModel interface.
func (ac *AnthropicChat) BindTools(toolsToBind []tools.Tool) (llms.ChatModel, error) {
	newClient := *ac // Create a shallow copy
	newClient.boundTools = make([]anthropic.ToolParam, 0, len(toolsToBind))
	for _, t := range toolsToBind {
		anthropicTool, err := mapAnthropicTool(t)
		if err != nil {
			return nil, fmt.Errorf("failed to map tool 		%s		 for Anthropic: %w", t.Name(), err)
		}
		newClient.boundTools = append(newClient.boundTools, anthropicTool)
	}
	return &newClient, nil
}

// Invoke implements the llms.ChatModel interface (core.Runnable).
func (ac *AnthropicChat) Invoke(ctx context.Context, input schema.Message, options ...core.Option) (schema.Message, error) {
	return ac.Generate(ctx, []schema.Message{input}, options...)
}

// Batch implements the llms.ChatModel interface (core.Runnable).
func (ac *AnthropicChat) Batch(ctx context.Context, inputs [][]schema.Message, options ...core.Option) ([][]schema.Message, error) {
	// TODO: Implement proper batching with concurrency control (ac.maxConcurrentBatches)
	// For now, simple sequential execution.
	results := make([][]schema.Message, len(inputs))
	var wg sync.WaitGroup
	// Create a channel to limit concurrency
	sem := make(chan struct{}, ac.maxConcurrentBatches)

	for i, msgSet := range inputs {
		wg.Add(1)
		sem <- struct{}{} // Acquire a spot
		go func(idx int, currentMsgSet []schema.Message) {
			defer wg.Done()
			defer func() { <-sem }() // Release spot

			// Create a new set of options for each batch invocation to avoid interference
			// This is important if options contain things like specific tool_choice for that batch
			currentOptions := make([]core.Option, len(options))
			copy(currentOptions, options)

			// If options contain a specific "batch_index" or similar, it could be used here.
			// For now, we assume options apply globally to all batches or are handled by applyAnthropicOptions.

			resp, err := ac.Generate(ctx, currentMsgSet, currentOptions...)
			if err != nil {
				// Handle error, perhaps by returning an error message in the result set
				log.Printf("Error in batch %d: %v", idx, err)
				// Store error in a way that can be retrieved, or return error immediately if critical
				// For now, we'll store a generic error message or nil if successful.
				results[idx] = []schema.Message{schema.NewSystemMessage(fmt.Sprintf("Error processing batch %d: %v", idx, err))}
			} else {
				results[idx] = []schema.Message{resp}
			}
		}(i, msgSet)
	}
	wg.Wait()
	// Check if any of the results contain only an error message, and if so, potentially return a single aggregated error.
	// For now, returning the slice of results which might contain error messages.
	return results, nil // TODO: Proper error aggregation for batch failures
}

// Stream implements the llms.ChatModel interface (core.Runnable).
func (ac *AnthropicChat) Stream(ctx context.Context, input []schema.Message, options ...core.Option) (<-chan schema.ChatResponseChunk, error) {
	return ac.StreamChat(ctx, input, options...)
}

