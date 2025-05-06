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
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
	core "github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
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
	modelName            string // Stores the resolved model name as a string
	defaultRequest       anthropic.BetaMessageNewParams
	boundTools           []anthropic.BetaToolParam // Changed from BetaToolUnionParam based on mapping
	maxConcurrentBatches int
}

var _ llms.ChatModel = (*AnthropicChat)(nil)
var _ core.Runnable = (*AnthropicChat)(nil)

const (
	DefaultAnthropicModelName = "claude-3-haiku-20240307"
)

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
		Model:     param.NewOpt(anthropic.BetaMessageNewParamsModelUnion{OfStr: anthropic.String(cfg.ModelName)}),
		MaxTokens: param.NewOpt[int64](1024),
	}

	for _, opt := range options {
		opt(cfg)
	}

	// Ensure model is set in DefaultRequest, potentially overriding from cfg.ModelName
	if cfg.ModelName != "" {
		cfg.DefaultRequest.Model = param.NewOpt(anthropic.BetaMessageNewParamsModelUnion{OfStr: anthropic.String(cfg.ModelName)})
	} else if !cfg.DefaultRequest.Model.IsPresent() {
		// Fallback if ModelName was empty and DefaultRequest.Model wasn't set by WithAnthropicDefaultRequest
		cfg.DefaultRequest.Model = param.NewOpt(anthropic.BetaMessageNewParamsModelUnion{OfStr: anthropic.String(DefaultAnthropicModelName)})
	}

	clientOpts := []option.RequestOption{}
	if cfg.APIKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.APIVersion != "" {
		clientOpts = append(clientOpts, option.WithDefaultHeader("anthropic-version", cfg.APIVersion))
	}

	client := anthropic.NewClient(clientOpts...)

	resolvedModelName := DefaultAnthropicModelName
	if cfg.DefaultRequest.Model.IsPresent() && cfg.DefaultRequest.Model.Get().OfStr != nil {
		resolvedModelName = *cfg.DefaultRequest.Model.Get().OfStr
	}

	ac := &AnthropicChat{
		client:               client,
		modelName:            resolvedModelName,
		defaultRequest:       cfg.DefaultRequest,
		maxConcurrentBatches: cfg.MaxConcurrentBatches,
	}

	return ac, nil
}

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
		var role constant.MessageRole // Use constant.MessageRole directly

		switch m := msg.(type) {
		case *schema.HumanMessage:
			role = constant.MessageRoleUser
			text := m.GetContent()
			contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestTextBlock(text))
		case *schema.AIMessage:
			role = constant.MessageRoleAssistant
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
			role = constant.MessageRoleUser // Tool results are sent as user messages
			toolUseIDStr := m.ToolCallID
			contentStr := m.GetContent()
			// For tool results, content can be text or JSON.
			// We'll try to marshal as JSON first, if not, then as text.
			var toolResultContent []anthropic.BetaToolResultBlockContentUnionParam
			var tempMap map[string]any
			if json.Unmarshal([]byte(contentStr), &tempMap) == nil {
				toolResultContent = append(toolResultContent, anthropic.BetaToolResultBlockContentUnionParam{
					OfRequestToolResultBlockContentJSONBlock: &anthropic.BetaToolResultBlockContentJSONBlockParam{
						JSON: tempMap,
						Type: constant.JSON,
					},
				})
			} else {
				toolResultContent = append(toolResultContent, anthropic.BetaToolResultBlockContentUnionParam{
					OfRequestToolResultBlockContentTextBlock: &anthropic.BetaToolResultBlockContentTextBlockParam{
						Text: contentStr,
						Type: constant.Text,
					},
				})
			}
			contentBlocks = append(contentBlocks, anthropic.BetaContentBlockParamOfRequestToolResultBlock(toolUseIDStr, anthropic.BetaToolResultBlockParam{
				ToolUseID: toolUseIDStr,
				Content:   param.NewOpt(toolResultContent),
			}))

		case *schema.SystemMessage:
			log.Println("Warning: System message encountered in unexpected position, ignoring.")
			continue
		default:
			log.Printf("Warning: Skipping message of unknown type %T for Anthropic API call.\n", msg)
			continue
		}

		if len(contentBlocks) > 0 {
			anthropicMsgs = append(anthropicMsgs, anthropic.BetaMessageParam{
				Role:    param.NewOpt(role),
				Content: param.NewOpt(contentBlocks),
			})
		} else if role == constant.MessageRoleAssistant && len(m.(*schema.AIMessage).ToolCalls) > 0 {
			// Assistant message might only contain tool calls, no text content
			anthropicMsgs = append(anthropicMsgs, anthropic.BetaMessageParam{Role: param.NewOpt(role), Content: param.NewOpt(contentBlocks)})
		} else if role != constant.MessageRoleAssistant {
			log.Printf("Warning: Skipping empty message conversion for type %T with role %s", msg, role)
		}
	}

	if len(anthropicMsgs) == 0 && systemPromptText == nil {
		return nil, nil, errors.New("no valid messages provided for Anthropic conversion")
	}
	return systemPromptText, anthropicMsgs, nil
}

func applyAnthropicOptions(defaults anthropic.BetaMessageNewParams, options ...core.Option) anthropic.BetaMessageNewParams {
	req := defaults
	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}

	if model, ok := config["model_name"].(string); ok && model != "" {
		req.Model = param.NewOpt(anthropic.BetaMessageNewParamsModelUnion{OfStr: anthropic.String(model)})
	}
	if temp, ok := config["temperature"].(float64); ok {
		req.Temperature = param.NewOpt(temp)
	} else if temp, ok := config["temperature"].(float32); ok {
		req.Temperature = param.NewOpt(float64(temp))
	}

	if maxTokens, ok := config["max_tokens"].(int); ok {
		req.MaxTokens = param.NewOpt(int64(maxTokens))
	}
	if stops, ok := config["stop_sequences"].([]string); ok {
		req.StopSequences = param.NewOpt(stops)
	}
	if topP, ok := config["top_p"].(float64); ok {
		req.TopP = param.NewOpt(topP)
	} else if topP, ok := config["top_p"].(float32); ok {
		req.TopP = param.NewOpt(float64(topP))
	}

	if topK, ok := config["top_k"].(int); ok {
		req.TopK = param.NewOpt(int64(topK))
	}

	if choice, ok := config["tool_choice"].(string); ok {			switch choice {
		case "auto":
			req.ToolChoice = param.NewOpt(anthropic.BetaToolChoiceParamOfRequestToolChoiceAuto(anthropic.RequestToolChoiceAutoParam{Type: constant.ToolChoiceTypeAuto}))
		case "any":
				req.ToolChoice = param.NewOpt(anthropic.BetaToolChoiceParamOfRequestToolChoiceAny(anthropic.RequestToolChoiceAnyParam{Type: constant.ToolChoiceTypeAny}))
			default: // Assumed to be a tool name
				req.ToolChoice = param.NewOpt(anthropic.BetaToolChoiceParamOfRequestToolChoiceTool(anthropic.RequestToolChoiceToolParam{Name: choice, Type: constant.ToolChoiceTypeTool}))
		}
	} else if choiceMap, ok := config["tool_choice"].(map[string]any); ok {
		if typeVal, ok := choiceMap["type"].(string); ok && typeVal == "tool" {
			if nameVal, ok := choiceMap["name"].(string); ok {
				req.ToolChoice = param.NewOpt(anthropic.BetaToolChoiceParamOfRequestToolChoiceTool(nameVal))
			}
		}
	}

	return req
}

func mapAnthropicTool(toolDef tools.ToolDefinition) (anthropic.BetaToolParam, error) {
	schemaStr := toolDef.InputSchemaJSON
	var paramsSchema anthropic.BetaSchemaParam

	if schemaStr == "" || schemaStr == "{}" || schemaStr == "null" {
		paramsSchema = anthropic.BetaSchemaParam{
			Type:       constant.Object,
			Properties: param.NewOpt(map[string]anthropic.BetaSchemaParamPropertiesUnion{}),
		}
		log.Printf("Warning: Tool %s has empty or invalid schema, using empty object schema.", toolDef.Name)
	} else {
		err := json.Unmarshal([]byte(schemaStr), &paramsSchema)
		if err != nil {
			return anthropic.BetaToolParam{}, fmt.Errorf("failed to unmarshal tool schema for %s: %w. Schema was: %s", toolDef.Name, err, schemaStr)
		}
	}

	// Ensure type is object if not set
	if paramsSchema.Type == "" {
		paramsSchema.Type = constant.Object
	}
	// Ensure properties is initialized if type is object and properties is nil
	if paramsSchema.Type == constant.Object && !paramsSchema.Properties.IsPresent() {
		paramsSchema.Properties = param.NewOpt(map[string]anthropic.BetaSchemaParamPropertiesUnion{})
	}

	var descOpt param.Opt[string]
	if toolDef.Description != "" {
		descOpt = param.NewOpt(toolDef.Description)
	}

	return anthropic.BetaToolParam{
		Name:        toolDef.Name,
		Description: descOpt,
		InputSchema: paramsSchema,
	}, nil
}

func (ac *AnthropicChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	systemPromptText, anthropicMessages, err := mapMessagesAndExtractSystem(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for Anthropic: %w", err)
	}

	req := applyAnthropicOptions(ac.defaultRequest, options...)
	req.Messages = anthropicMessages
	if systemPromptText != nil {
		req.System = param.NewOpt([]anthropic.BetaTextBlockParam{{Text: *systemPromptText, Type: constant.TextBlockTypeText}})
	}

	if len(ac.boundTools) > 0 {
		req.Tools = param.NewOpt(ac.boundTools)
			if !req.ToolChoice.IsPresent() { // Only set default ToolChoice if not already set by options
				req.ToolChoice = param.NewOpt(anthropic.BetaToolChoiceParamOfRequestToolChoiceAuto(anthropic.RequestToolChoiceAutoParam{Type: constant.ToolChoiceTypeAuto}))
			}

	if !req.MaxTokens.IsPresent() || req.MaxTokens.Get() == 0 {
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
		}
	}

	aiMsg := schema.NewAIMessage(responseText)
	aiMsg.ToolCalls = toolCalls
	if resp.StopReason.IsPresent() {
		aiMsg.AdditionalArgs["stop_reason"] = string(resp.StopReason.Get())
	}
	if resp.Usage.IsPresent() {
		usage := resp.Usage.Get()
		aiMsg.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  int(usage.InputTokens),
			"output_tokens": int(usage.OutputTokens),
		}
	}

	return aiMsg, nil
}

func (ac *AnthropicChat) StreamChat(firstParamForTest int, messages []schema.Message, options ...core.Option) (<-chan llms.AIMessageChunk, error) {
	systemPromptText, anthropicMessages, err := mapMessagesAndExtractSystem(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for Anthropic streaming: %w", err)
	}

	req := applyAnthropicOptions(ac.defaultRequest, options...)
	req.Messages = anthropicMessages
	if systemPromptText != nil {
		req.System = param.NewOpt([]anthropic.BetaTextBlockParam{{Text: *systemPromptText, Type: constant.TextBlockTypeText}})
	}
	// Stream is set by NewStreaming method, not directly in params for that method.

	if len(ac.boundTools) > 0 {
		req.Tools = param.NewOpt(ac.boundTools)
		if !req.ToolChoice.IsPresent() {
			req.ToolChoice = param.NewOpt(anthropic.BetaToolChoiceParamOfRequestToolChoiceAuto())
		}
	}

	if !req.MaxTokens.IsPresent() || req.MaxTokens.Get() == 0 {
		return nil, errors.New("MaxTokens must be set and non-zero for Anthropic StreamChat")
	}

	stream, err := ac.client.Beta.Messages.NewStreaming(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("anthropic stream creation failed: %w", err)
	}

	outputCh := make(chan llms.AIMessageChunk)

	go func() {
		defer close(outputCh)
		defer stream.Close()

		currentToolCallChunks := make(map[int]*schema.ToolCallChunk)

		for {
			eventUnion, err := stream.Recv() // eventUnion is anthropic.BetaRawMessageStreamEventUnion
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				finalChunk := llms.AIMessageChunk{Err: fmt.Errorf("anthropic stream error: %w", err)}
				outputCh <- finalChunk
				return
			}

			chunk := llms.AIMessageChunk{AdditionalArgs: make(map[string]any)}

			switch event := eventUnion.AsAny().(type) {
			case anthropic.BetaMessageStartEvent:
				chunk.Content = ""
				if event.Message.Usage.IsPresent() {
					usage := event.Message.Usage.Get()
					chunk.AdditionalArgs["usage_input_tokens"] = int(usage.InputTokens)
				}
			case anthropic.BetaContentBlockStartEvent:
				if toolUse := event.ContentBlock.AsResponseToolUseBlock(); toolUse.Type == constant.ToolUse { // Check type explicitly
					tcc := schema.ToolCallChunk{
						Index: int(event.Index),
						ID:    toolUse.ID,
						Name:  toolUse.Name,
					}
					currentToolCallChunks[int(event.Index)] = &tcc
					chunk.ToolCallChunks = []schema.ToolCallChunk{tcc} // Send initial chunk for the tool call
				}
			case anthropic.BetaContentBlockDeltaEvent:
				switch delta := event.Delta.AsAny().(type) {
				case anthropic.BetaTextDelta:
					chunk.Content = delta.Text
				case anthropic.BetaInputJSONDelta: // This is for tool arguments
					if tcc, exists := currentToolCallChunks[int(event.Index)]; exists {
						tcc.Arguments += delta.PartialJSON
						chunk.ToolCallChunks = []schema.ToolCallChunk{*tcc} // Send updated chunk
					}
				}
			case anthropic.BetaMessageDeltaEvent:
				if event.Delta.StopReason.IsPresent() {
					chunk.AdditionalArgs["stop_reason"] = string(event.Delta.StopReason.Get())
				}
				if event.Usage.OutputTokens.IsPresent() {
					chunk.AdditionalArgs["usage_output_tokens"] = int(event.Usage.OutputTokens.Get())
				}
			case anthropic.BetaMessageStopEvent:
				// Final event, can carry final usage if not already sent.
				// No specific content for AIMessageChunk here, but good to handle.
			case anthropic.BetaContentBlockStopEvent:
				// Marks the end of a content block, useful if tracking specific blocks.
			case anthropic.BetaPingEvent:
				// Keep-alive, ignore for AIMessageChunk.
				continue // Don't send an empty chunk
			case anthropic.BetaErrorEvent:
				chunk.Err = fmt.Errorf("anthropic stream error event: %s - %s", event.Error.Type, event.Error.Message)
			}

			// Only send if there's content, a tool call chunk, or an error
			if chunk.Content != "" || len(chunk.ToolCallChunks) > 0 || chunk.Err != nil || len(chunk.AdditionalArgs) > 0 {
				outputCh <- chunk
			}
		}
	}()

	return outputCh, nil
}

func (ac *AnthropicChat) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, ok := input.([]schema.Message)
	if !ok {
		return nil, errors.New("AnthropicChat Invoke expects input to be []schema.Message")
	}
	return ac.Generate(ctx, messages, options...)
}

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

// Stream implements the core.Runnable interface.
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

func (ac *AnthropicChat) BindTools(toolsToBind []tools.Tool) llms.ChatModel {
	anthropicTools := make([]anthropic.BetaToolParam, 0, len(toolsToBind))
	for _, t := range toolsToBind {
		// Assuming tools.Tool has a Definition() method that returns tools.ToolDefinition
		// or directly provides the necessary fields.
		// For this example, let's assume tools.Tool is directly usable or has a method to get definition.
		// This part needs to align with the actual structure of tools.Tool.
		// Let's assume tools.Tool is equivalent to tools.ToolDefinition for now.
		toolDef := tools.ToolDefinition{
			Name:            t.Name(),
			Description:     t.Description(),
			InputSchemaJSON: t.InputSchemaJSON(), // Assuming this method exists
		}
		mappedTool, err := mapAnthropicTool(toolDef)
		if err != nil {
			log.Printf("Warning: Failed to map tool %s for Anthropic: %v", t.Name(), err)
			continue
		}
		anthropicTools = append(anthropicTools, mappedTool)
	}

	newChat := *ac // Create a shallow copy
	newChat.boundTools = anthropicTools
	return &newChat
}

func (ac *AnthropicChat) GetModelName() string {
	return ac.modelName
}

