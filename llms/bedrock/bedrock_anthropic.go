// Package bedrock contains provider-specific logic for AWS Bedrock.
package bedrock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	// brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types" // Not directly needed if bedrockruntime is used for stream type

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
}

type anthropicMessagePart struct {
	Role    string                    `json:"role"`
	Content []anthropicMessageContent `json:"content"`
}

type anthropicMessageContent struct {
	Type               string         `json:"type"`
	Text               string         `json:"text,omitempty"`
	ToolUseID          string         `json:"tool_use_id,omitempty"` // For tool_use block
	Name               string         `json:"name,omitempty"`      // For tool_use block
	Input              map[string]any `json:"input,omitempty"`     // For tool_use block
	ToolUseIDForResult string         `json:"tool_use_id,omitempty"` // For tool_result block
	ContentForResult   any            `json:"content,omitempty"`   // For tool_result block (string or structured)
	IsError            *bool          `json:"is_error,omitempty"`  // For tool_result block
}

type anthropicMessagesResponseBody struct {
	ID           string                    `json:"id"`
	Type         string                    `json:"type"`
	Role         string                    `json:"role"`
	Content      []anthropicMessageContent `json:"content"`
	StopReason   string                    `json:"stop_reason"`
	StopSequence *string                   `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamChunk struct {
	Type  string `json:"type"`
	Index *int   `json:"index,omitempty"` // For content_block_start, content_block_delta
	Delta *struct {
		Type         string `json:"type"` // e.g., "text_delta", "input_json_delta"
		Text         string `json:"text,omitempty"`
		PartialJson  string `json:"partial_json,omitempty"` // For tool input streaming
		StopReason   string `json:"stop_reason,omitempty"`   // In message_delta
		StopSequence string `json:"stop_sequence,omitempty"` // In message_delta
	} `json:"delta,omitempty"`
	ContentBlock *struct { // For content_block_start
		Type  string `json:"type"` // e.g., "tool_use"
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Input string `json:"input,omitempty"` // This is usually an empty object string initially for tool_use
	} `json:"content_block,omitempty"`
	Message *struct { // For message_start, message_stop
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
	Error *struct { // For error type
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// --- Anthropic specific request/response mappers ---

func (bl *BedrockLLM) invokeAnthropicModel(ctx context.Context, _ []schema.Message, opts map[string]any, _ bool) (json.RawMessage, error) {
    messages, ok := opts["messages"].([]schema.Message)
    if !ok {
        return nil, errors.New("messages not found or not of correct type in options for Anthropic")
    }

	var anthropicTools []any
	if toolsVal, ok := opts["tools"]; ok {
		if t, ok := toolsVal.([]tools.Tool); ok {
			anthropicTools = mapToolsToAnthropic(t)
		} else if t, ok := toolsVal.([]any); ok {
		    anthropicTools = t
		}
	}
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

func (bl *BedrockLLM) invokeAnthropicModelStream(ctx context.Context, _ []schema.Message, opts map[string]any) (*bedrockruntime.InvokeModelWithResponseStreamEventStream, error) {
    messages, ok := opts["messages"].([]schema.Message)
    if !ok {
        return nil, errors.New("messages not found or not of correct type in options for Anthropic stream")
    }

	var anthropicTools []any
	if toolsVal, ok := opts["tools"]; ok {
		if t, ok := toolsVal.([]tools.Tool); ok {
			anthropicTools = mapToolsToAnthropic(t)
		} else if t, ok := toolsVal.([]any); ok {
		    anthropicTools = t
		}
	}
	requestBodyBytes, err := buildAnthropicChatRequest(messages, anthropicTools, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build Anthropic stream request for Bedrock: %w", err)
	}

	bedrockReq := bl.createInvokeModelWithResponseStreamInput(requestBodyBytes)
	output, err := bl.client.InvokeModelWithResponseStream(ctx, bedrockReq)
	if err != nil {
		return nil, fmt.Errorf("Bedrock InvokeModelWithResponseStream (Anthropic) failed: %w", err)
	}
	return output.GetStream(), nil
}

func buildAnthropicChatRequest(messages []schema.Message, mappedTools []any, callOpts map[string]any) ([]byte, error) {
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
				if len(contentParts) == 0 {
					log.Println("Warning: AIMessage has neither text content nor tool calls for Anthropic. This might be an issue.")
				}
			}

		case *schema.ToolMessage:
			role = "user"
			var toolContentResult any = m.GetContent()
			contentStr := m.GetContent()
			if (strings.HasPrefix(contentStr, "{") && strings.HasSuffix(contentStr, "}")) || (strings.HasPrefix(contentStr, "[") && strings.HasSuffix(contentStr, "]")) {
			    var jsonData any
			    if err := json.Unmarshal([]byte(contentStr), &jsonData); err == nil {
			        toolContentResult = jsonData
			    }
			}
			contentParts = append(contentParts, anthropicMessageContent{
				Type:               "tool_result",
				ToolUseIDForResult: m.ToolCallID,
				ContentForResult:   toolContentResult,
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

	maxTokens := 1024
	if mt, ok := callOpts["max_tokens"].(int); ok && mt > 0 {
		maxTokens = mt
	} else if mt, ok := callOpts["max_tokens_to_sample"].(int); ok && mt > 0 { 
	    maxTokens = mt
	}

	body := anthropicMessagesRequestBody{
		AnthropicVersion: "bedrock-2023-05-31",
		Messages:         anthropicMsgs,
		System:           systemPrompt,
		MaxTokens:        maxTokens,
	}

	if temp, ok := callOpts["temperature"].(float32); ok {
		body.Temperature = &temp
	} else if temp, ok := callOpts["temperature"].(float64); ok {
	    temp32 := float32(temp)
	    tbody.Temperature = &temp32
	}

	if stop, ok := callOpts["stop_words"].([]string); ok && len(stop) > 0 {
		body.StopSequences = stop
	} else if stop, ok := callOpts["stop_sequences"].([]string); ok && len(stop) > 0 {
	    tbody.StopSequences = stop
	}

	if topP, ok := callOpts["top_p"].(float32); ok {
		body.TopP = &topP
	} else if topP, ok := callOpts["top_p"].(float64); ok {
	    topP32 := float32(topP)
	    tbody.TopP = &topP32
	}

	if topK, ok := callOpts["top_k"].(int); ok {
		body.TopK = &topK
	}

	if len(mappedTools) > 0 {
		body.Tools = mappedTools
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
			argsCopy := ""
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
		isMeaningful = true
	case "message_delta":
		if streamEvent.Delta != nil {
			if streamEvent.Delta.StopReason != "" {
				chunk.AdditionalArgs["stop_reason"] = streamEvent.Delta.StopReason
				isMeaningful = true
			}
		}
	case "message_stop":
		if streamEvent.Message != nil && streamEvent.Message.Usage.OutputTokens > 0 {
			chunk.AdditionalArgs["usage_output_tokens"] = streamEvent.Message.Usage.OutputTokens
			isMeaningful = true
		}
		if streamEvent.Message != nil && streamEvent.Message.StopReason != "" {
		    if _, ok := chunk.AdditionalArgs["stop_reason"]; !ok {
		        chunk.AdditionalArgs["stop_reason"] = streamEvent.Message.StopReason
		    }
		    isMeaningful = true
		}
	case "ping":
		return nil, nil
	case "error":
		if streamEvent.Error != nil {
			chunk.Err = fmt.Errorf("anthropic stream error (%s): %s", streamEvent.Error.Type, streamEvent.Error.Message)
		} else {
			chunk.Err = errors.New("unknown anthropic stream error event")
		}
		isMeaningful = true
	default:
		log.Printf("Warning: Unhandled Anthropic stream event type: %s. Chunk: %s", streamEvent.Type, string(chunkBytes))
		return nil, nil
	}

	if !isMeaningful && chunk.Err == nil {
		return nil, nil
	}
	if len(chunk.AdditionalArgs) == 0 {
	    chunk.AdditionalArgs = nil
	}
	return &chunk, chunk.Err
}

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
			paramsSchema = make(map[string]any)
			paramsSchema["type"] = "object"
			paramsSchema["properties"] = make(map[string]any)
		}
		anthropicTools = append(anthropicTools, map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"input_schema": paramsSchema,
		})
	}
	return anthropicTools
}

