// Package bedrock contains provider-specific logic for AWS Bedrock.
package bedrock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools" // Keep for potential future tool mapping
)

// --- Anthropic specific types ---
type anthropicMessagesRequestBody struct {
	AnthropicVersion string                 `json:"anthropic_version"`
	Messages         []anthropicMessagePart `json:"messages"`
	System           string                 `json:"system,omitempty"`
	MaxTokens        int                    `json:"max_tokens"`
	Temperature      *float32               `json:"temperature,omitempty"`
	TopP             *float32               `json:"top_p,omitempty"`
	TopK             *int                   `json:"top_k,omitempty"`
	StopSequences    []string               `json:"stop_sequences,omitempty"`
	Tools            []any                  `json:"tools,omitempty"` 
	// ToolChoice could be added here if needed, mirroring Anthropic API
}

type anthropicMessagePart struct {
	Role    string                    `json:"role"`
	Content []anthropicMessageContent `json:"content"`
}

type anthropicMessageContent struct {
	Type               string         `json:"type"`
	Text               string         `json:"text,omitempty"`
	ToolUseID          string         `json:"tool_use_id,omitempty"` // Corrected tag based on Anthropic Bedrock docs
	Name               string         `json:"name,omitempty"`
	Input              map[string]any `json:"input,omitempty"`
	ToolUseIDForResult string         `json:"tool_use_id,omitempty"` // Used in tool_result content
	ContentForResult   any            `json:"content,omitempty"`   // Changed to `any` for flexibility, often string or list of blocks
	IsError            *bool          `json:"is_error,omitempty"`  // Pointer for optional field
}

type anthropicMessagesResponseBody struct {
	ID           string                    `json:"id"`
	Type         string                    `json:"type"`
	Role         string                    `json:"role"`
	Content      []anthropicMessageContent `json:"content"`
	StopReason   string                    `json:"stop_reason"`
	StopSequence *string                   `json:"stop_sequence,omitempty"` // Pointer for optional field
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamChunk struct {
	Type  string `json:"type"`
	Index *int   `json:"index,omitempty"`
	Delta *struct {
		Type         string `json:"type"`
		Text         string `json:"text,omitempty"`
		PartialJson  string `json:"partial_json,omitempty"`
		StopReason   string `json:"stop_reason,omitempty"`
		StopSequence string `json:"stop_sequence,omitempty"`
	} `json:"delta,omitempty"`
	ContentBlock *struct {
		Type  string `json:"type"`
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Input string `json:"input,omitempty"` // Input for tool_use is a JSON string here
	} `json:"content_block,omitempty"`
	Message *struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		StopReason   string `json:"stop_reason,omitempty"`
		StopSequence string `json:"stop_sequence,omitempty"`
		Usage        struct {
			InputTokens  int `json:"input_tokens,omitempty"`
			OutputTokens int `json:"output_tokens,omitempty"`
		} `json:"usage,omitempty"`
	} `json:"message,omitempty"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// --- Anthropic specific request/response mappers ---

func (bl *BedrockLLM) invokeAnthropicModel(ctx context.Context, messages []schema.Message, opts schema.CallOptions, stream bool) (json.RawMessage, error) {
	anthropicTools := mapToolsToAnthropic(opts.Tools)
	requestBodyBytes, err := buildAnthropicChatRequest(messages, anthropicTools, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build Anthropic request for Bedrock: %w", err)
	}

	bedrockReq := bl.createInvokeModelInput(requestBodyBytes)
	resp, err := bl.client.InvokeModel(ctx, bedrockReq)
	if err != nil {
		return nil, fmt.Errorf("Bedrock InvokeModel (Anthropic) failed: %w", err)
	}
	return resp.Body, nil
}

func (bl *BedrockLLM) invokeAnthropicModelStream(ctx context.Context, messages []schema.Message, opts schema.CallOptions) (*brtypes.ResponseStream, error) {
	anthropicTools := mapToolsToAnthropic(opts.Tools)
	requestBodyBytes, err := buildAnthropicChatRequest(messages, anthropicTools, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build Anthropic stream request for Bedrock: %w", err)
	}

	bedrockReq := bl.createInvokeModelWithResponseStreamInput(requestBodyBytes)
	resp, err := bl.client.InvokeModelWithResponseStream(ctx, bedrockReq)
	if err != nil {
		return nil, fmt.Errorf("Bedrock InvokeModelWithResponseStream (Anthropic) failed: %w", err)
	}
	return resp.Stream, nil
}

func buildAnthropicChatRequest(messages []schema.Message, mappedTools []any, callOpts schema.CallOptions) ([]byte, error) {
	systemPrompt := ""
	anthropicMsgs := make([]anthropicMessagePart, 0, len(messages))
	processedMessages := messages

	if len(messages) > 0 {
		if sysMsg, ok := messages[0].(*schema.SystemMessage); ok {
			systemPrompt = sysMsg.GetContent()
			processedMessages = messages[1:]
		}
	}

	for _, msg := range processedMessages {
		var role string
		contentParts := []anthropicMessageContent{}

		switch m := msg.(type) {
		case *schema.HumanMessage:
			role = "user"
			contentParts = append(contentParts, anthropicMessageContent{Type: "text", Text: m.GetContent()})
		case *schema.AIMessage:
			role = "assistant"
			hasContent := m.GetContent() != ""
			hasToolCalls := len(m.ToolCalls) > 0

			if hasToolCalls {
				for _, tc := range m.ToolCalls {
					var inputMap map[string]any
					if tc.Arguments != "" && tc.Arguments != "{}" && tc.Arguments != "null" {
						err := json.Unmarshal([]byte(tc.Arguments), &inputMap)
						if err != nil {
							log.Printf("Warning: Failed to unmarshal tool call arguments for %s: %v. Using raw string.", tc.Name, err)
							inputMap = map[string]any{"_beluga_raw_args": tc.Arguments}
						}
					} else {
						inputMap = make(map[string]any)
					}
					contentParts = append(contentParts, anthropicMessageContent{
						Type:      "tool_use",
						ToolUseID: tc.ID,
						Name:      tc.Name,
						Input:     inputMap,
					})
				}
			}
			if hasContent {
				contentParts = append(contentParts, anthropicMessageContent{Type: "text", Text: m.GetContent()})
			}
			if !hasContent && !hasToolCalls {
				// AIMessage can be empty if it only signals the end of a turn after tool_calls
				// Or if it_s just a container for tool_calls without preceding text.
				// Anthropic expects content for assistant messages, even if it_s just to hold tool_use.
				// If contentParts is still empty, it means no tool_calls were successfully mapped either.
				if len(contentParts) == 0 {
					log.Println("Warning: AIMessage has neither text content nor tool calls for Anthropic. This might be an issue.")
				}
			}

		case *schema.ToolMessage:
			role = "user"
			contentParts = append(contentParts, anthropicMessageContent{
				Type:               "tool_result",
				ToolUseIDForResult: m.ToolCallID,
				ContentForResult:   m.GetContent(), // Anthropic expects string or list of blocks for tool_result content
				// IsError: &m.IsError, // Assuming schema.ToolMessage has IsError field
			})

		default:
			log.Printf("Warning: Skipping message of type %T for Anthropic Bedrock conversion.", msg)
			continue
		}

		if len(contentParts) > 0 {
			anthropicMsgs = append(anthropicMsgs, anthropicMessagePart{Role: role, Content: contentParts})
		} else {
			log.Printf("Warning: No content parts generated for message type %T in Anthropic conversion.", msg)
		}
	}

	if len(anthropicMsgs) == 0 && systemPrompt == "" {
		return nil, errors.New("no valid messages or system prompt for Anthropic Bedrock request")
	}

	body := anthropicMessagesRequestBody{
		AnthropicVersion: "bedrock-2023-05-31",
		Messages:         anthropicMsgs,
		System:           systemPrompt,
		MaxTokens:        callOpts.MaxTokens,
	}

	if callOpts.Temperature != 0 {
		temp32 := float32(callOpts.Temperature)
		body.Temperature = &temp32
	}
	if len(callOpts.StopWords) > 0 {
		body.StopSequences = callOpts.StopWords
	}
	if callOpts.TopP != 0 {
		topP32 := float32(callOpts.TopP)
		body.TopP = &topP32
	}
	if callOpts.TopK != 0 {
		body.TopK = &callOpts.TopK
	}
	if len(mappedTools) > 0 {
		body.Tools = mappedTools
		// TODO: Add tool_choice logic from callOpts if needed
		// if callOpts.ToolChoice != "" { ... }
	}

	return json.Marshal(body)
}

func (bl *BedrockLLM) anthropicResponseToAIMessage(responseBody []byte) (schema.Message, error) {
	var resp anthropicMessagesResponseBody
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Anthropic Bedrock response: %w. Body: %s", err, string(responseBody))
	}

	responseText := ""
	toolCalls := []schema.ToolCall{}

	for _, content := range resp.Content {
		switch content.Type {
		case "text":
			responseText += content.Text
		case "tool_use":
			inputBytes, jsonErr := json.Marshal(content.Input)
			argsStr := "{}"
			if jsonErr == nil {
				argsStr = string(inputBytes)
			} else {
				log.Printf("Warning: Failed to marshal tool_use input for %s: %v", content.Name, jsonErr)
			}
			toolCalls = append(toolCalls, schema.ToolCall{
				ID:        content.ToolUseID,
				Name:      content.Name,
				Arguments: argsStr,
			})
		}
	}

	aiMsg := schema.NewAIMessage(responseText)
	if len(toolCalls) > 0 {
		aiMsg.ToolCalls = toolCalls
	}
	aiMsg.AdditionalArgs = make(map[string]any)
	aiMsg.AdditionalArgs["usage"] = map[string]int{
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
		"total_tokens":  resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}
	aiMsg.AdditionalArgs["stop_reason"] = resp.StopReason
	if resp.StopSequence != nil {
		aiMsg.AdditionalArgs["stop_sequence"] = *resp.StopSequence
	}

	return aiMsg, nil
}

func (bl *BedrockLLM) anthropicStreamChunkToAIMessageChunk(chunkBytes []byte) (*llms.AIMessageChunk, error) {
	var streamEvent anthropicStreamChunk
	err := json.Unmarshal(chunkBytes, &streamEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Anthropic stream chunk: %w. Chunk: %s", err, string(chunkBytes))
	}

	chunk := llms.AIMessageChunk{AdditionalArgs: make(map[string]any)}
	var isMeaningful bool

	switch streamEvent.Type {
	case "message_start":
		if streamEvent.Message != nil && streamEvent.Message.Usage.InputTokens > 0 {
			chunk.AdditionalArgs["usage_input_tokens"] = streamEvent.Message.Usage.InputTokens
			isMeaningful = true
		}
	case "content_block_start":
		if streamEvent.ContentBlock != nil && streamEvent.ContentBlock.Type == "tool_use" {
			idx := streamEvent.Index
			idCopy := streamEvent.ContentBlock.ID
			nameCopy := streamEvent.ContentBlock.Name
			argsCopy := "" // Placeholder, actual args come in delta
			chunk.ToolCallChunks = []schema.ToolCallChunk{{
				ID:        &idCopy,
				Name:      &nameCopy,
				Arguments: &argsCopy,
				Index:     idx,
			}}
			isMeaningful = true
		}
	case "content_block_delta":
		if streamEvent.Delta != nil {
			if streamEvent.Delta.Type == "text_delta" && streamEvent.Delta.Text != "" {
				chunk.Content = streamEvent.Delta.Text
				isMeaningful = true
			} else if streamEvent.Delta.Type == "input_json_delta" && streamEvent.Delta.PartialJson != "" {
				idx := streamEvent.Index
				argsDelta := streamEvent.Delta.PartialJson
				chunk.ToolCallChunks = []schema.ToolCallChunk{{
					Arguments: &argsDelta,
					Index:     idx,
				}}
				isMeaningful = true
			}
		}
	case "content_block_stop":
		// Can be meaningful if it_s the only signal for a tool call ending.
		isMeaningful = true // Consider it meaningful to ensure stream processing continues.
	case "message_delta":
		if streamEvent.Delta != nil {
			if streamEvent.Delta.StopReason != "" {
				chunk.AdditionalArgs["stop_reason"] = streamEvent.Delta.StopReason
				isMeaningful = true
			}
			// Usage in message_delta is usually output tokens for the current delta, not final.
		}
	case "message_stop":
		if streamEvent.Message != nil && streamEvent.Message.Usage.OutputTokens > 0 {
			chunk.AdditionalArgs["usage_output_tokens"] = streamEvent.Message.Usage.OutputTokens
			isMeaningful = true
		}
		if streamEvent.Message != nil && streamEvent.Message.StopReason != "" {
		    if chunk.AdditionalArgs["stop_reason"] == nil {
		        chunk.AdditionalArgs["stop_reason"] = streamEvent.Message.StopReason
		    }
		    isMeaningful = true
		}
	case "ping":
		// Meaningless for content, but keeps stream alive.
	case "error":
		if streamEvent.Error != nil {
			chunk.Err = fmt.Errorf("anthropic stream error (%s): %s", streamEvent.Error.Type, streamEvent.Error.Message)
		} else {
			chunk.Err = errors.New("unknown anthropic stream error event")
		}
		isMeaningful = true // Error is always meaningful
	default:
		log.Printf("Warning: Unhandled Anthropic stream event type: %s", streamEvent.Type)
	}

	if !isMeaningful && chunk.Err == nil {
		return nil, nil // Return nil if not a meaningful chunk and no error
	}
	if len(chunk.AdditionalArgs) == 0 {
	    chunk.AdditionalArgs = nil
	}
	return &chunk, chunk.Err
}

// mapToolsToAnthropic converts Beluga tools to the format expected by Anthropic on Bedrock.
func mapToolsToAnthropic(toolsToBind []tools.Tool) []any {
	if len(toolsToBind) == 0 {
		return nil
	}
	anthropicTools := make([]any, 0, len(toolsToBind))
	for _, t := range toolsToBind {
		schemaStr := t.Schema()
		var paramsSchema map[string]any

		if schemaStr != "" && schemaStr != "{}" && schemaStr != "null" {
			err := json.Unmarshal([]byte(schemaStr), &paramsSchema)
			if err != nil {
				log.Printf("ERROR: Failed to unmarshal schema for tool %s for Anthropic binding: %v. Schema was: %s. Skipping tool.", t.Name(), err, schemaStr)
				continue
			}
		} else {
			paramsSchema = map[string]any{"type": "object", "properties": map[string]any{}}
			log.Printf("Warning: Tool %s has empty or null schema for Anthropic binding, using empty object schema.", t.Name())
		}

		if _, typeOk := paramsSchema["type"]; !typeOk {
			if _, propsOk := paramsSchema["properties"]; propsOk {
				paramsSchema["type"] = "object"
			} else {
				log.Printf("Warning: Tool %s schema lacks 'type' and 'properties', ensuring empty object schema for Anthropic.", t.Name())
				paramsSchema = map[string]any{"type": "object", "properties": map[string]any{}}
			}
		} else if paramsSchema["type"] != "object" {
		    log.Printf("Warning: Tool %s schema type is '%s', not 'object'. Anthropic expects an object. Attempting to wrap.", t.Name(), paramsSchema["type"])
		    // This case is tricky. If it_s a scalar, Anthropic might not support it directly.
		    // For now, we proceed, but this might lead to errors from Anthropic.
		}

		anthropicToolSpec := map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"input_schema": map[string]any{ // Corrected from inputSchema.json to input_schema
				"type": "object", // Ensure the input_schema itself is an object containing the JSON schema
				"properties": paramsSchema["properties"], // Pass the properties of the tool_s schema
				// "required": paramsSchema["required"], // Pass required if present
			},
		}
		if req, ok := paramsSchema["required"]; ok {
		    if inputSchema, ok2 := anthropicToolSpec["input_schema"].(map[string]any); ok2 {
		        inputSchema["required"] = req
		    }
		}

		anthropicTools = append(anthropicTools, anthropicToolSpec) // Directly append the spec, not wrapped in "toolSpec"
	}
	return anthropicTools
}

