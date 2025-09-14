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

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	core "github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// --- Helper config struct for NewAnthropicChat ---
type anthropicChatConfig struct {
	APIKey               string
	BaseURL              string
	APIVersion           string
	ModelName            string // User-provided model name string
	DefaultRequest       anthropic.BetaMessageNewParams
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
func WithAnthropicDefaultRequest(req anthropic.BetaMessageNewParams) AnthropicOption {
	return func(cfg *anthropicChatConfig) {
		cfg.DefaultRequest = req
	}
}

// WithAnthropicMaxConcurrentBatches sets the concurrency limit for Batch.
func WithAnthropicMaxConcurrentBatches(n int) AnthropicOption {
	return func(cfg *anthropicChatConfig) {
		if n > 0 {
			cfg.MaxConcurrentBatches = n
		}
	}
}

// --- End Options ---

// AnthropicChat represents a chat model client for the Anthropic API.
type AnthropicChat struct {
	client               *anthropic.Client
	modelName            string
	defaultRequest       anthropic.BetaMessageNewParams
	boundTools           []anthropic.BetaToolUnionParam
	maxConcurrentBatches int
}

var _ llms.ChatModel = (*AnthropicChat)(nil)
var _ core.Runnable = (*AnthropicChat)(nil)

const (
	DefaultAnthropicModelName = "claude-3-haiku-20240307"
	roleUser                  = "user"
	roleAssistant             = "assistant"
	roleSystem               = "system"
)

// NewAnthropicChat creates a new AnthropicChat instance for interacting with the Anthropic Claude API.
// It accepts optional configuration options through AnthropicOption functions.
func NewAnthropicChat(options ...AnthropicOption) (*AnthropicChat, error) {
	cfg := &anthropicChatConfig{
		APIKey:               os.Getenv("ANTHROPIC_API_KEY"),
		BaseURL:              os.Getenv("ANTHROPIC_BASE_URL"),
		APIVersion:           os.Getenv("ANTHROPIC_API_VERSION"),
		ModelName:            DefaultAnthropicModelName,
		MaxConcurrentBatches: 5,
	}

	// Initialize DefaultRequest with a model and MaxTokens
	cfg.DefaultRequest = anthropic.BetaMessageNewParams{
		Model:     cfg.ModelName,
		MaxTokens: 1024,
	}

	for _, opt := range options {
		opt(cfg)
	}

	// Ensure model is set in DefaultRequest, potentially overriding from cfg.ModelName
	if cfg.ModelName != "" {
		cfg.DefaultRequest.Model = cfg.ModelName
	} else if cfg.DefaultRequest.Model == "" {
		// Fallback if ModelName was empty and DefaultRequest.Model wasn't set by WithAnthropicDefaultRequest
		cfg.DefaultRequest.Model = DefaultAnthropicModelName
	}

	opts := []option.RequestOption{}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	// Initialize the client first, then we can set additional headers
	// Set API version as a custom header if provided
	if cfg.APIVersion != "" {
		opts = append(opts, option.WithHeader("anthropic-version", cfg.APIVersion))
	}
	
	client := anthropic.NewClient(opts...)

	resolvedModelName := DefaultAnthropicModelName
	if cfg.DefaultRequest.Model != "" {
		resolvedModelName = cfg.DefaultRequest.Model
	}

	ac := &AnthropicChat{
		client:               &client,
		modelName:            resolvedModelName,
		defaultRequest:       cfg.DefaultRequest,
		maxConcurrentBatches: cfg.MaxConcurrentBatches,
	}

	return ac, nil
}

// mapMessagesAndExtractSystem converts Beluga schema.Message objects to Anthropic API message format,
// extracting the system message if present.
func mapMessagesAndExtractSystem(messages []schema.Message) (*string, []anthropic.BetaMessageParam, error) {
	var systemPromptText *string
	var anthropicMsgs []anthropic.BetaMessageParam
	processedMessages := messages

	if len(messages) > 0 {
		if sysMsg, ok := messages[0].(*schema.SystemMessage); ok {
			if sysMsg.GetContent() != "" {
				content := sysMsg.GetContent()
				systemPromptText = &content
			}
			processedMessages = messages[1:]
		}
	}

	anthropicMsgs = make([]anthropic.BetaMessageParam, 0, len(processedMessages))
	for _, msg := range processedMessages {
		var contentBlocks []anthropic.BetaContentBlockParamUnion
		var role anthropic.BetaMessageParamRole
		
		switch m := msg.(type) {
		case *schema.HumanMessage:
			role = anthropic.BetaMessageParamRole(roleUser)
			text := m.GetContent()
			contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestTextBlock(text))
		case *schema.AIMessage:
			role = anthropic.BetaMessageParamRole(roleAssistant)
			if m.GetContent() != "" {
				text := m.GetContent()
				contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestTextBlock(text))
			}
			for _, tc := range m.ToolCalls {
				var inputMap map[string]any
				if tc.Arguments != "" && tc.Arguments != "{}" && tc.Arguments != "null" {
					err := json.Unmarshal([]byte(tc.Arguments), &inputMap)
					if err != nil {
						log.Printf("Warning: Failed to unmarshal tool call arguments for %s: %v. Args: %s", tc.Name, err, tc.Arguments)
						// Continue with nil inputMap if unmarshalling fails, or handle as error
						inputMap = nil // Or some other default / error handling
					}
				}
				contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestToolUseBlock(tc.ID, inputMap, tc.Name))
			}
		case *schema.ToolMessage:
			role = anthropic.BetaMessageParamRole(roleUser) // Tool results are sent as user messages
			contentStr := m.GetContent()
			// For tool results, content can be text or JSON.
			// Tool result handling with correct structure
			var tempMap map[string]any
			if json.Unmarshal([]byte(contentStr), &tempMap) != nil {
				// Not valid JSON, log a warning
				log.Printf("Warning: Tool message content is not valid JSON: %s", contentStr)
			}
			// Create a tool result content block with correct structure
			contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestToolResultBlock(
				m.ToolCallID,
			))
		default:
			log.Printf("Warning: Skipping message of unknown type %T for Anthropic API call.\n", msg)
			continue
		}

		if len(contentBlocks) > 0 {
			// Add message with appropriate role
			anthropicMsgs = append(anthropicMsgs, anthropic.BetaMessageParam{
				Role:    role,
				Content: contentBlocks,
			})
		} else if role == anthropic.BetaMessageParamRole(roleAssistant) && len(msg.(*schema.AIMessage).ToolCalls) > 0 {
			// Assistant message might only contain tool calls, no text content
			anthropicMsgs = append(anthropicMsgs, anthropic.BetaMessageParam{
				Role:    role,
				Content: contentBlocks,
			})
		} else if role != anthropic.BetaMessageParamRole(roleAssistant) {
			log.Printf("Warning: Skipping empty message conversion for type %T with role %s", msg, role)
		}
	}

	if len(anthropicMsgs) == 0 && systemPromptText == nil {
		return nil, nil, errors.New("no valid messages provided for Anthropic conversion")
	}
	return systemPromptText, anthropicMsgs, nil
}

// applyAnthropicOptions applies Beluga core.Options to Anthropic API request parameters.
func applyAnthropicOptions(defaults anthropic.BetaMessageNewParams, options ...core.Option) anthropic.BetaMessageNewParams {
	req := defaults
	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}

	if model, ok := config["model_name"].(string); ok && model != "" {
		req.Model = model // Direct string assignment
	}
	if temp, ok := config["temperature"].(float64); ok {
		req.Temperature = param.NewOpt(temp)
	} else if temp, ok := config["temperature"].(float32); ok {
		req.Temperature = param.NewOpt(float64(temp))
	}

	if maxTokens, ok := config["max_tokens"].(int); ok {
		req.MaxTokens = int64(maxTokens) // Direct assignment
	}
	if stops, ok := config["stop_sequences"].([]string); ok {
		req.StopSequences = stops // Direct assignment
	}
	if topP, ok := config["top_p"].(float64); ok {
		req.TopP = param.NewOpt(topP)
	} else if topP, ok := config["top_p"].(float32); ok {
		req.TopP = param.NewOpt(float64(topP))
	}

	if topK, ok := config["top_k"].(int); ok {
		req.TopK = param.NewOpt(int64(topK))
	}

	// Handle tool choice parameter - simplified to support the current SDK
	if choice, ok := config["tool_choice"].(string); ok {
		switch choice {
		case "auto":
			// Auto mode - let the model decide whether to call tools
			// Currently using an empty parameter as auto is typically the default
		case "any":
			// Any mode - model can use any tool
			// Currently using an empty parameter as default behavior
		default:
			// If a specific tool name is provided, we'll handle it in a simplified way
			log.Printf("Note: Tool choice '%s' specified but simplified for SDK compatibility", choice)
		}
	} else if choiceMap, ok := config["tool_choice"].(map[string]any); ok {
		// Handle mapped tool choice in a simplified way
		if typeVal, ok := choiceMap["type"].(string); ok {
			log.Printf("Note: Tool choice type '%s' specified but simplified for SDK compatibility", typeVal)
		}
	}

	return req
}

// Function schema structure that matches Anthropic's requirements
type FunctionParameters struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// ToolParameters wraps function params to match Anthropic's API structure
type ToolParameters struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// mapAnthropicTool converts a Beluga tool definition to a tool function schema.
func mapAnthropicTool(toolDef tools.ToolDefinition) (string, error) {
	// Create a tool function schema definition
	toolSchema := map[string]interface{}{
		"name": toolDef.Name,
	}

	// Get schema string from the input schema
	var schemaStr string
	switch schema := toolDef.InputSchema.(type) {
	case string:
		schemaStr = schema
	case []byte:
		schemaStr = string(schema)
	default:
		return "", fmt.Errorf("unsupported InputSchema type: %T", toolDef.InputSchema)
	}

	// Parse the schema
	var schemaObj map[string]interface{}
	if err := json.Unmarshal([]byte(schemaStr), &schemaObj); err != nil {
		return "", fmt.Errorf("failed to unmarshal tool schema: %w", err)
	}
	
	// Add parameters to the tool schema
	toolSchema["parameters"] = schemaObj

	// Add description if present
	if toolDef.Description != "" {
		toolSchema["description"] = toolDef.Description
	}
	
	// Marshal the full tool schema to a JSON string
	schemaBytes, err := json.Marshal(toolSchema)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool schema: %w", err)
	}
	
	return string(schemaBytes), nil
}

// Generate creates a completion for the given messages using the Anthropic API.
// It returns a single message response without streaming.
func (ac *AnthropicChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	systemPromptText, anthropicMessages, err := mapMessagesAndExtractSystem(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for Anthropic: %w", err)
	}

	req := applyAnthropicOptions(ac.defaultRequest, options...)
	req.Messages = anthropicMessages
	if systemPromptText != nil {
		// Correct system prompt field name - use the System field for the latest API
		req.System = []anthropic.BetaTextBlockParam{
			{Text: *systemPromptText},
		}
	}

	if len(ac.boundTools) > 0 {
		req.Tools = ac.boundTools
	}

	if req.MaxTokens == 0 {
		return nil, errors.New("MaxTokens must be set and non-zero for Anthropic Generate")
	}

	resp, err := ac.client.Beta.Messages.New(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("anthropic chat completion failed: %w", err)
	}

	var responseText string
	var toolCalls []schema.ToolCall

	for _, blockUnion := range resp.Content {
		switch content := blockUnion.AsAny().(type) {
		case anthropic.BetaTextBlock:
			responseText += content.Text
		case anthropic.BetaToolUseBlock:
			argsBytes, _ := json.Marshal(content.Input)
			toolCall := schema.ToolCall{
				ID:        content.ID,
				Name:      content.Name,
				Arguments: string(argsBytes),
			}
			toolCalls = append(toolCalls, toolCall)
		case map[string]interface{}:
			// Handle potential map representation in newer API versions
			if contentType, ok := content["type"].(string); ok && contentType == "tool_use" {
				id, _ := content["id"].(string)
				name, _ := content["name"].(string)
				input, _ := content["input"].(map[string]interface{})
				argsBytes, _ := json.Marshal(input)
				toolCall := schema.ToolCall{
					ID:        id,
					Name:      name,
					Arguments: string(argsBytes),
				}
				toolCalls = append(toolCalls, toolCall)
			} else if contentType, ok := content["type"].(string); ok && contentType == "text" {
				if text, ok := content["text"].(string); ok {
					responseText += text
				}
			}
		}
	}

	aiMsg := schema.NewAIMessage(responseText)
	aiMsg.ToolCalls = toolCalls
	
	// Add metadata and usage information
	if resp.StopReason != "" {
		aiMsg.AdditionalArgs = map[string]any{"finish_reason": string(resp.StopReason)}
	} else {
		aiMsg.AdditionalArgs = map[string]any{}
	}
	
	// Add token usage information
	aiMsg.AdditionalArgs["usage"] = map[string]int{
		"input_tokens":  int(resp.Usage.InputTokens),
		"output_tokens": int(resp.Usage.OutputTokens),
		"total_tokens":  int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
	}

	return aiMsg, nil
}

// StreamChat creates a streaming completion for the given messages using the Anthropic API.
// It returns a channel that receives message chunks as they are generated.
func (ac *AnthropicChat) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llms.AIMessageChunk, error) {
	systemPromptText, anthropicMessages, err := mapMessagesAndExtractSystem(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for Anthropic streaming: %w", err)
	}

	req := applyAnthropicOptions(ac.defaultRequest, options...)
	req.Messages = anthropicMessages
	if systemPromptText != nil {
		// Correct system prompt field name
		req.System = []anthropic.BetaTextBlockParam{
			{Text: *systemPromptText},
		}
	}
	// Stream is set by NewStreaming method, not directly in params for that method.

	if len(ac.boundTools) > 0 {
		req.Tools = ac.boundTools
	}

	if req.MaxTokens == 0 {
		return nil, errors.New("MaxTokens must be set and non-zero for Anthropic StreamChat")
	}

	streamResp := ac.client.Beta.Messages.NewStreaming(ctx, req)
	if err := streamResp.Err(); err != nil {
		return nil, fmt.Errorf("anthropic stream creation failed: %w", err)
	}
	stream := streamResp
	
	outputCh := make(chan llms.AIMessageChunk)
	
	go func() {
		defer close(outputCh)
		defer stream.Close()
		
		currentToolCallChunks := make(map[int]*schema.ToolCallChunk)
		
		// Use the correct stream events method
		for {
			event := stream.Next()
			if stream.Err() != nil {
				if errors.Is(stream.Err(), io.EOF) {
					break
				}
				finalChunk := llms.AIMessageChunk{Err: fmt.Errorf("anthropic stream error: %w", stream.Err())}
				outputCh <- finalChunk
				return
			}
			
			chunk := llms.AIMessageChunk{AdditionalArgs: make(map[string]any)}

			// Since event types might be different in latest SDK, let's handle generically
			if bytes, err := json.Marshal(event); err == nil {
				var eventMap map[string]interface{}
				if err := json.Unmarshal(bytes, &eventMap); err == nil {
					// Extract type
					if eventType, ok := eventMap["type"].(string); ok {
						switch eventType {
						case "message_start":
							// Handle message start
							if message, ok := eventMap["message"].(map[string]interface{}); ok {
								if usage, ok := message["usage"].(map[string]interface{}); ok {
									if inputTokens, ok := usage["input_tokens"].(float64); ok {
										chunk.AdditionalArgs["usage_input_tokens"] = int(inputTokens)
									}
								}
							}
						case "content_block_start":
							// Handle content block start
							if contentBlock, ok := eventMap["content_block"].(map[string]interface{}); ok {
								if contentType, ok := contentBlock["type"].(string); ok && contentType == "tool_use" {
									tcc := schema.ToolCallChunk{
										ID: contentBlock["id"].(string),
									}
									if name, ok := contentBlock["name"].(string); ok {
										nameCopy := name
										tcc.Name = &nameCopy
									}
									if index, ok := eventMap["index"].(float64); ok {
										indexInt := int(index)
										tcc.Index = &indexInt
										currentToolCallChunks[indexInt] = &tcc
									}
									chunk.ToolCallChunks = []schema.ToolCallChunk{tcc}
								}
							}
						case "content_block_delta":
							// Handle content block delta
							if delta, ok := eventMap["delta"].(map[string]interface{}); ok {
								if deltaType, ok := delta["type"].(string); ok {
									switch deltaType {
									case "text_delta":
										if text, ok := delta["text"].(string); ok {
											chunk.Content = text
										}
									case "input_json_delta":
										if jsonData, ok := delta["partial_json"].(string); ok {
											if index, ok := eventMap["index"].(float64); ok {
												indexInt := int(index)
												if tc, exists := currentToolCallChunks[indexInt]; exists {
													tc.Arguments += jsonData
													chunk.ToolCallChunks = []schema.ToolCallChunk{*tc}
												}
											}
										}
									}
								}
							}
						case "error":
							if errorData, ok := eventMap["error"].(map[string]interface{}); ok {
								errorType, _ := errorData["type"].(string)
								errorMsg, _ := errorData["message"].(string)
								chunk.Err = fmt.Errorf("anthropic stream error event: %s - %s", errorType, errorMsg)
							} else {
								chunk.Err = fmt.Errorf("unknown anthropic stream error event")
							}
						default:
							log.Printf("Warning: Unhandled Anthropic stream event type: %s", eventType)
						}
					}
				}
			}

			// Only send if there's content, a tool call chunk, or an error
			if chunk.Content != "" || len(chunk.ToolCallChunks) > 0 || chunk.Err != nil || len(chunk.AdditionalArgs) > 0 {
				outputCh <- chunk
			}
		}
	}()

	return outputCh, nil
}

// Invoke implements the core.Runnable interface for AnthropicChat.
// It expects input to be a slice of schema.Message.
func (ac *AnthropicChat) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, ok := input.([]schema.Message)
	if !ok {
		return nil, errors.New("AnthropicChat Invoke expects input to be []schema.Message")
	}
	return ac.Generate(ctx, messages, options...)
}

// Batch processes multiple inputs in parallel, with a concurrency limit.
// Each input should be a slice of schema.Message.
func (ac *AnthropicChat) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	numJobs := len(inputs)
	results := make([]any, numJobs)
	errs := make([]error, numJobs)
	var wg sync.WaitGroup

	// Determine concurrency: use MaxConcurrentBatches or number of jobs if smaller
	concurrency := ac.maxConcurrentBatches
	if numJobs < concurrency {
		concurrency = numJobs
	}
	jobChan := make(chan int, numJobs)
	for i := 0; i < numJobs; i++ {
		jobChan <- i
	}
	close(jobChan)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jobIndex := range jobChan {
				messages, ok := inputs[jobIndex].([]schema.Message)
				if !ok {
					errs[jobIndex] = fmt.Errorf("input at index %d is not []schema.Message", jobIndex)
					continue
				}
				result, err := ac.Generate(ctx, messages, options...)
				results[jobIndex] = result
				errs[jobIndex] = err
			}
		}()
	}

	wg.Wait()

	// Consolidate errors
	var combinedErr error
	for i, err := range errs {
		if err != nil {
			if combinedErr == nil {
				combinedErr = fmt.Errorf("error in batch job %d: %w", i, err)
			} else {
				combinedErr = fmt.Errorf("%w; error in batch job %d: %w", combinedErr, i, err)
			}
		}
	}

	return results, combinedErr
}

// Stream implements the core.Runnable interface for streaming responses.
// It expects input to be a slice of schema.Message and returns a channel of any.
func (ac *AnthropicChat) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, ok := input.([]schema.Message)
	if !ok {
		return nil, errors.New("AnthropicChat Stream expects input to be []schema.Message")
	}

	aiChunkChan, err := ac.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	// Adapt llms.AIMessageChunk channel to chan any for core.Runnable
	outputChan := make(chan any)
	go func() {
		defer close(outputChan)
		for chunk := range aiChunkChan {
			outputChan <- chunk // Send the AIMessageChunk as is
		}
	}()
	return outputChan, nil
}

// BindTools attaches tools to the AnthropicChat instance for function calling capabilities.
// It returns a new AnthropicChat instance with the tools bound.
func (ac *AnthropicChat) BindTools(toolsToBind []tools.Tool) llms.ChatModel {
	// Create a new slice of tools to bind
	anthropicTools := make([]anthropic.BetaToolUnionParam, 0, len(toolsToBind))
	
	for _, t := range toolsToBind {
		// Get tool definition from the tool interface
		def := t.Definition()
		toolDef := tools.ToolDefinition{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		}
		
		// Convert to Anthropic tool format
		schemaJSON, err := mapAnthropicTool(toolDef)
		if err != nil {
			log.Printf("Warning: Failed to map tool %s for Anthropic: %v", def.Name, err)
			continue
		}
		
		// Create a simplified tool parameter
		// For compatibility with the SDK, use a map-based approach
		toolMap := map[string]interface{}{
			"type": "function",
		}
		
		// Parse the schema JSON
		var funcSchema map[string]interface{}
		if err := json.Unmarshal([]byte(schemaJSON), &funcSchema); err != nil {
			log.Printf("Warning: Failed to parse tool schema JSON for %s: %v", def.Name, err)
			continue
		}
		
		// Add the function to the tool map
		toolMap["function"] = funcSchema
		
		// Create a tool union parameter from the map
		toolBytes, _ := json.Marshal(toolMap)
		var toolUnion anthropic.BetaToolUnionParam
		if err := json.Unmarshal(toolBytes, &toolUnion); err != nil {
			log.Printf("Warning: Failed to create tool union param for %s: %v", def.Name, err)
			continue
		}
		
		// Add the tool to our list
		anthropicTools = append(anthropicTools, toolUnion)
	}

	newChat := *ac // Create a shallow copy
	newChat.boundTools = anthropicTools
	return &newChat
}

// GetModelName returns the model name used by this AnthropicChat instance.
func (ac *AnthropicChat) GetModelName() string {
	return ac.modelName
}

