// Package bedrock contains provider-specific logic for Cohere models on AWS Bedrock.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	// brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types" // Not directly needed if bedrockruntime is used for stream type
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
)

// CohereCommandRequest represents the request payload for Cohere Command models on Bedrock.
// See: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-cohere-command.html
type CohereCommandRequest struct {
	Prompt            string               `json:"prompt"`
	MaxTokens         int                  `json:"max_tokens,omitempty"`
	Temperature       float64              `json:"temperature,omitempty"`
	P                 float64              `json:"p,omitempty"` // Nucleus sampling (top-p)
	K                 int                  `json:"k,omitempty"`   // Top-k sampling
	StopSequences     []string             `json:"stop_sequences,omitempty"`
	ReturnLikelihoods string               `json:"return_likelihoods,omitempty"` // NONE, GENERATION, ALL
	Stream            bool                 `json:"stream,omitempty"`            // Required for streaming
	NumGenerations    int                  `json:"num_generations,omitempty"`    // Not typically used in chat, but available
	LogitBias         map[string]float32   `json:"logit_bias,omitempty"`        // Example: {"234": -5.0}
	Truncate          string               `json:"truncate,omitempty"`          // NONE, START, END
	ChatHistory       []CohereChatMessage  `json:"chat_history,omitempty"`
	Tools             []CohereTool         `json:"tools,omitempty"`
	ToolResults       []CohereToolResult   `json:"tool_results,omitempty"`
	ForceSingleStep   bool                 `json:"force_single_step,omitempty"` // If true, model makes one tool call and waits for Tool Results
	PromptTruncation  string               `json:"prompt_truncation,omitempty"` // AUTO, AUTO_PRESERVE_ORDER, OFF
}

// CohereChatMessage represents a message in the chat history for Cohere models.
type CohereChatMessage struct {
	Role    string `json:"role"` // USER, CHATBOT, SYSTEM, TOOL
	Message string `json:"message"`
}

// CohereTool represents a tool definition for Cohere models.
type CohereTool struct {
	Name                 string                        `json:"name"`
	Description          string                        `json:"description"`
	ParameterDefinitions map[string]CohereToolParameter `json:"parameter_definitions,omitempty"`
}

// CohereToolParameter defines a parameter for a Cohere tool.
type CohereToolParameter struct {
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // string, number, boolean, integer, array, object
	Required    bool   `json:"required,omitempty"`
}

// CohereToolResult represents the result of a tool call to be sent to the model.
type CohereToolResult struct {
	Call    *CohereToolCallResponse `json:"call"`
	Outputs []map[string]any      `json:"outputs"`
}

// CohereToolCallResponse is used within CohereToolResult to identify the call.
type CohereToolCallResponse struct {
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters"`
}

// CohereCommandResponse represents the response payload from Cohere Command models on Bedrock.
type CohereCommandResponse struct {
	ID          string             `json:"id,omitempty"`
	Generations []CohereGeneration `json:"generations,omitempty"`
	Prompt      string             `json:"prompt,omitempty"`
	IsFinished   bool               `json:"is_finished,omitempty"`
	FinishReason string             `json:"finish_reason,omitempty"`
	Text         string             `json:"text,omitempty"`
	ToolCalls []CohereToolCall `json:"tool_calls,omitempty"`
	ChatHistory []CohereChatMessage `json:"chat_history,omitempty"`
	Meta *CohereResponseMeta `json:"meta,omitempty"`
	EventType string `json:"event_type,omitempty"` // For streaming
}

// CohereGeneration represents a single generation in a non-streaming response.
type CohereGeneration struct {
	ID           string             `json:"id"`
	Text         string             `json:"text"`
	FinishReason string             `json:"finish_reason"`
	ToolCalls    []CohereToolCall   `json:"tool_calls,omitempty"`
}

// CohereToolCall represents a tool call made by the Cohere model (part of response).
type CohereToolCall struct {
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters"`
}

// CohereResponseMeta contains metadata about the response, including token usage.
type CohereResponseMeta struct {
	APIVersion struct {
		Version string `json:"version"`
	} `json:"api_version"`
	BilledUnits struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"billed_units"`
	Tokens *struct { // This field is sometimes present in stream-end events
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"tokens,omitempty"`
}

func mapSchemaMessagesToCohereChat(messages []schema.Message, currentToolCalls []schema.ToolCall) ([]CohereChatMessage, []CohereToolResult) {
	cohereMessages := make([]CohereChatMessage, 0, len(messages))
	cohereToolResults := []CohereToolResult{}

	for _, msg := range messages {
		var role string
		switch m := msg.(type) {
		case *schema.HumanMessage:
			role = "USER"
			cohereMessages = append(cohereMessages, CohereChatMessage{Role: role, Message: m.GetContent()})
		case *schema.AIMessage:
			role = "CHATBOT"
			if m.GetContent() != "" {
				cohereMessages = append(cohereMessages, CohereChatMessage{Role: role, Message: m.GetContent()})
			}
		case *schema.SystemMessage:
			role = "SYSTEM"
			cohereMessages = append(cohereMessages, CohereChatMessage{Role: role, Message: m.GetContent()})
		case *schema.ToolMessage:
			var matchingCall *CohereToolCallResponse
			for _, prevCall := range currentToolCalls { 
				if prevCall.ID == m.ToolCallID {
					var params map[string]any
					if err := json.Unmarshal([]byte(prevCall.Arguments), &params); err != nil {
						log.Printf("Warning: Could not unmarshal params for tool call ID %s: %v", prevCall.ID, err)
						params = make(map[string]any) 
					}
					matchingCall = &CohereToolCallResponse{Name: prevCall.Name, Parameters: params}
					break
				}
			}
			if matchingCall == nil {
				log.Printf("Warning: No matching tool call found for ToolMessage ID %s. Skipping tool result.", m.ToolCallID)
				continue
			}

			var outputData map[string]any
			if err := json.Unmarshal([]byte(m.GetContent()), &outputData); err != nil {
				outputData = map[string]any{"output": m.GetContent()}
				log.Printf("Warning: Tool result content for %s is not valid JSON, wrapping as simple output: %v", m.ToolCallID, err)
			}
			cohereToolResults = append(cohereToolResults, CohereToolResult{
				Call:    matchingCall,
				Outputs: []map[string]any{outputData},
			})
		default:
			log.Printf("Warning: Skipping message of unknown type %T for Cohere chat history.", msg)
		}
	}
	return cohereMessages, cohereToolResults
}

func (bl *BedrockLLM) invokeCohereModel(ctx context.Context, _ string, messages []schema.Message, options map[string]any) (json.RawMessage, error) {
	requestPayload := CohereCommandRequest{
		Stream: false, 
	}

	var lastHumanMessageContent string
	var previousToolCalls []schema.ToolCall 

	if len(messages) > 0 {
		for i := len(messages) - 1; i >= 0; i-- {
			if hm, ok := messages[i].(*schema.HumanMessage); ok && lastHumanMessageContent == "" {
				lastHumanMessageContent = hm.GetContent()
			}
			if aim, ok := messages[i].(*schema.AIMessage); ok && len(previousToolCalls) == 0 {
			    if i < len(messages)-1 { 
			        if _, isToolMsg := messages[i+1].(*schema.ToolMessage); isToolMsg {
			             previousToolCalls = aim.ToolCalls 
			        }
			    }
			}
		}
	}

	if lastHumanMessageContent != "" {
		requestPayload.Prompt = lastHumanMessageContent
	} else if len(messages) > 0 && messages[len(messages)-1].GetType() == schema.MessageTypeTool {
	    requestPayload.Prompt = " " 
	} else {
	    return nil, fmt.Errorf("Cohere request requires a prompt (from last human message) or tool results with context")
	}

	requestPayload.ChatHistory, requestPayload.ToolResults = mapSchemaMessagesToCohereChat(messages, previousToolCalls)

	if mt, ok := options["max_tokens"].(int); ok && mt > 0 {
		requestPayload.MaxTokens = mt
	}
	if temp, ok := options["temperature"].(float64); ok && temp > 0 {
		requestPayload.Temperature = temp
	} else if temp, ok := options["temperature"].(float32); ok && temp > 0 {
		requestPayload.Temperature = float64(temp)
	}

	if topP, ok := options["top_p"].(float64); ok && topP > 0 {
		requestPayload.P = topP
	} else if topP, ok := options["top_p"].(float32); ok && topP > 0 {
		requestPayload.P = float64(topP)
	}

	if topK, ok := options["top_k"].(int); ok && topK > 0 {
		requestPayload.K = topK
	}

	if stop, ok := options["stop_words"].([]string); ok && len(stop) > 0 {
		requestPayload.StopSequences = stop
	}

	if toolsVal, ok := options["tools"].([]tools.Tool); ok && len(toolsVal) > 0 {
		requestPayload.Tools = mapBelugaToolsToCohere(toolsVal)
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Cohere Command request: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModel(ctx, bl.createInvokeModelInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Cohere Command model: %w", err)
	}
	return output.Body, nil
}

func (bl *BedrockLLM) cohereResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var resp CohereCommandResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Cohere Command response: %w. Body: %s", err, string(body))
	}

	content := ""
	if len(resp.Generations) > 0 {
		content = resp.Generations[0].Text 
	}

	aiMsg := schema.NewAIMessage(content)
	aiMsg.AdditionalArgs = make(map[string]any)

	if resp.Meta != nil && resp.Meta.Tokens != nil {
		aiMsg.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  resp.Meta.Tokens.InputTokens,
			"output_tokens": resp.Meta.Tokens.OutputTokens,
			"total_tokens":  resp.Meta.Tokens.InputTokens + resp.Meta.Tokens.OutputTokens,
		}
	} else if resp.Meta != nil && resp.Meta.BilledUnits.InputTokens > 0 { 
		aiMsg.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  resp.Meta.BilledUnits.InputTokens,
			"output_tokens": resp.Meta.BilledUnits.OutputTokens,
			"total_tokens":  resp.Meta.BilledUnits.InputTokens + resp.Meta.BilledUnits.OutputTokens,
		}
	}

	var finalToolCalls []schema.ToolCall
	responseToolCalls := resp.ToolCalls 
	if len(resp.Generations) > 0 && len(resp.Generations[0].ToolCalls) > 0 {
		responseToolCalls = resp.Generations[0].ToolCalls 
	}

	if len(responseToolCalls) > 0 {
		finalToolCalls = make([]schema.ToolCall, len(responseToolCalls))
		for i, tc := range responseToolCalls {
			argsBytes, _ := json.Marshal(tc.Parameters)
			finalToolCalls[i] = schema.ToolCall{
				ID:        fmt.Sprintf("%s-%d-%s", tc.Name, i, bl.modelID), 
				Name:      tc.Name,
				Arguments: string(argsBytes),
			}
		}
		aiMsg.ToolCalls = finalToolCalls
	}

	if len(resp.Generations) > 0 {
	    aiMsg.AdditionalArgs["finish_reason"] = resp.Generations[0].FinishReason
	} else if resp.FinishReason != "" {
	    aiMsg.AdditionalArgs["finish_reason"] = resp.FinishReason 
	}

	return aiMsg, nil
}

// For streaming
func (bl *BedrockLLM) invokeCohereModelStream(ctx context.Context, _ string, messages []schema.Message, options map[string]any) (*bedrockruntime.InvokeModelWithResponseStreamEventStream, error) {
	requestPayload := CohereCommandRequest{
		Stream: true, 
	}
    var lastHumanMessageContent string
    var previousToolCalls []schema.ToolCall

    if len(messages) > 0 {
        for i := len(messages) - 1; i >= 0; i-- {
            if hm, ok := messages[i].(*schema.HumanMessage); ok && lastHumanMessageContent == "" {
                lastHumanMessageContent = hm.GetContent()
            }
            if aim, ok := messages[i].(*schema.AIMessage); ok && len(previousToolCalls) == 0 {
                 if i < len(messages)-1 {
                    if _, isToolMsg := messages[i+1].(*schema.ToolMessage); isToolMsg {
                         previousToolCalls = aim.ToolCalls
                    }
                }
            }
        }
    }

    if lastHumanMessageContent != "" {
        requestPayload.Prompt = lastHumanMessageContent
    } else if len(messages) > 0 && messages[len(messages)-1].GetType() == schema.MessageTypeTool {
        requestPayload.Prompt = " "
    } else {
        if len(messages) == 0 { 
             return nil, fmt.Errorf("Cohere stream request requires a prompt or chat history")
        }
        requestPayload.Prompt = " "
    }

    requestPayload.ChatHistory, requestPayload.ToolResults = mapSchemaMessagesToCohereChat(messages, previousToolCalls)

	if mt, ok := options["max_tokens"].(int); ok && mt > 0 {
		requestPayload.MaxTokens = mt
	}
	if temp, ok := options["temperature"].(float64); ok && temp > 0 {
		requestPayload.Temperature = temp
	} else if temp, ok := options["temperature"].(float32); ok && temp > 0 {
		requestPayload.Temperature = float64(temp)
	}

	if topP, ok := options["top_p"].(float64); ok && topP > 0 {
		requestPayload.P = topP
	} else if topP, ok := options["top_p"].(float32); ok && topP > 0 {
		requestPayload.P = float64(topP)
	}

	if topK, ok := options["top_k"].(int); ok && topK > 0 {
		requestPayload.K = topK
	}

	if stop, ok := options["stop_words"].([]string); ok && len(stop) > 0 {
		requestPayload.StopSequences = stop
	}

	if toolsVal, ok := options["tools"].([]tools.Tool); ok && len(toolsVal) > 0 {
		requestPayload.Tools = mapBelugaToolsToCohere(toolsVal)
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Cohere Command stream request: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Cohere Command model with response stream: %w", err)
	}
	return output.GetStream(), nil
}

func (bl *BedrockLLM) cohereStreamChunkToAIMessageChunk(chunkBytes []byte) (*llms.AIMessageChunk, error) {
	var streamResp CohereCommandResponse
	if err := json.Unmarshal(chunkBytes, &streamResp); err != nil {
		// log.Printf("Warning: failed to unmarshal Cohere stream chunk: %v. Chunk: %s", err, string(chunkBytes))
		return nil, nil 
	}

	chunk := &llms.AIMessageChunk{Content: streamResp.Text}
	chunk.AdditionalArgs = make(map[string]any)
	var isMeaningful bool

	switch streamResp.EventType {
	case "text-generation":
		if streamResp.Text != "" {
			chunk.Content = streamResp.Text
			isMeaningful = true
		}
	case "tool-calls-generation":
	    if len(streamResp.ToolCalls) > 0 {
	        chunk.ToolCallChunks = make([]schema.ToolCallChunk, len(streamResp.ToolCalls))
	        for i, tc := range streamResp.ToolCalls {
	            nameCopy := tc.Name
	            argsBytes, _ := json.Marshal(tc.Parameters)
	            argsStr := string(argsBytes)
	            // Cohere tool calls in stream don_t have IDs, generate one or leave nil if not strictly needed for chunking
	            // For now, we don_t assign an ID to the chunk, as it_s about the delta.
					chunk.ToolCallChunks[i] = schema.ToolCallChunk{
	                Name:      &nameCopy,
	                Arguments: argsStr,
	                // Index might be relevant if multiple tool calls are streamed piecewise, but Cohere seems to send them whole in this event type.
	            }
	        }
	        isMeaningful = true
	    }
	case "stream-end":
		if streamResp.FinishReason != "" {
			chunk.AdditionalArgs["finish_reason"] = streamResp.FinishReason
			isMeaningful = true
		}
		if streamResp.Meta != nil && streamResp.Meta.Tokens != nil {
			chunk.AdditionalArgs["usage"] = map[string]int{
				"input_tokens":  streamResp.Meta.Tokens.InputTokens,
				"output_tokens": streamResp.Meta.Tokens.OutputTokens,
				"total_tokens":  streamResp.Meta.Tokens.InputTokens + streamResp.Meta.Tokens.OutputTokens,
			}
			isMeaningful = true
		} else if streamResp.Meta != nil && streamResp.Meta.BilledUnits.InputTokens > 0 { // Fallback for stream-end
		    chunk.AdditionalArgs["usage"] = map[string]int{
				"input_tokens":  streamResp.Meta.BilledUnits.InputTokens,
				"output_tokens": streamResp.Meta.BilledUnits.OutputTokens,
				"total_tokens":  streamResp.Meta.BilledUnits.InputTokens + streamResp.Meta.BilledUnits.OutputTokens,
			}
		    isMeaningful = true
		}

	case "stream-start", "search-queries-generation", "search-results", "citation-generation":
		// These are intermediate events, might not always translate to an AIMessageChunk directly
		// but could be used for logging or more complex state management if needed.
		// For now, we don_t create a chunk from them unless they carry specific data we need.
		return nil, nil
	default:
		log.Printf("Warning: Unhandled Cohere stream event type: %s. Chunk: %s", streamResp.EventType, string(chunkBytes))
		return nil, nil
	}

	if !isMeaningful {
		return nil, nil
	}
    if len(chunk.AdditionalArgs) == 0 {
        chunk.AdditionalArgs = nil
    }
	return chunk, nil
}

func mapBelugaToolsToCohere(belugaTools []tools.Tool) []CohereTool {
	cohereTools := make([]CohereTool, len(belugaTools))
	for i, tool := range belugaTools {
		var paramDefs map[string]CohereToolParameter
		toolDef := tool.Definition()
		schemaStr, ok := toolDef.InputSchema.(string)
		if ok && schemaStr != "" && schemaStr != "{}" && schemaStr != "null" {
			// For initial unmarshal of properties
			tempSchema := struct {
				Type       string                            `json:"type"`
				Properties map[string]json.RawMessage      `json:"properties"`
				Required   []string                          `json:"required"`
			}{}

			if err := json.Unmarshal([]byte(schemaStr), &tempSchema); err == nil && tempSchema.Type == "object" {
				paramDefs = make(map[string]CohereToolParameter)
				for name, propRaw := range tempSchema.Properties {
					var propDef struct {
						Type        string `json:"type"`
						Description string `json:"description"`
					}
					if err := json.Unmarshal(propRaw, &propDef); err == nil {
						isRequired := false
						for _, reqName := range tempSchema.Required {
							if reqName == name {
								isRequired = true
								break
							}
						}
						paramDefs[name] = CohereToolParameter{
							Description: propDef.Description,
							Type:        propDef.Type,
							Required:    isRequired,
						}
					} else {
						log.Printf("Warning: Could not unmarshal property definition for %s in tool %s: %v", name, tool.Name(), err)
					}
				}
			} else if err != nil {
			    log.Printf("Warning: Could not unmarshal tool schema for %s: %v. Schema: %s", tool.Name(), err, schemaStr)
			}
		}

		cohereTools[i] = CohereTool{
			Name:                 tool.Name(),
			Description:          tool.Description(),
			ParameterDefinitions: paramDefs,
		}
	}
	return cohereTools
}

