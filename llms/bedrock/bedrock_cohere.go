// Package bedrock contains provider-specific logic for Cohere models on AWS Bedrock.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
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
	// For TOOL role, message is the tool output (JSON string)
	// For CHATBOT role with tool_calls, message is text preceding tool_calls, tool_calls are separate.
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
	Call    *CohereToolCallResponse `json:"call"` // Pointer to the tool call that this result corresponds to
	Outputs []map[string]any      `json:"outputs"` // List of outputs, each output is a JSON object
}

// CohereToolCallResponse is used within CohereToolResult to identify the call.
// This mirrors the structure of a tool_call received from the model.
type CohereToolCallResponse struct {
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters"`
}

// CohereCommandResponse represents the response payload from Cohere Command models on Bedrock.
type CohereCommandResponse struct {
	ID          string             `json:"id,omitempty"` // Not always present in stream
	Generations []CohereGeneration `json:"generations,omitempty"` // For non-streaming
	Prompt      string             `json:"prompt,omitempty"`      // Echoed prompt
	// Streaming specific fields
	IsFinished   bool               `json:"is_finished,omitempty"`
	FinishReason string             `json:"finish_reason,omitempty"` // COMPLETE, ERROR, ERROR_TOXIC, ERROR_LIMIT, USER_CANCEL, MAX_TOKENS, TOOL_CALLS
	Text         string             `json:"text,omitempty"`         // For text generation stream event
	// Tool call related fields (can appear in non-streaming or streaming final response)
	ToolCalls []CohereToolCall `json:"tool_calls,omitempty"`
	ChatHistory []CohereChatMessage `json:"chat_history,omitempty"` // Updated chat history
	// Meta information, including token counts
	Meta *CohereResponseMeta `json:"meta,omitempty"`
	// Stream event type for differentiating chunks
	EventType string `json:"event_type,omitempty"` // e.g., "text-generation", "tool-calls", "stream-end"
}

// CohereGeneration represents a single generation in a non-streaming response.
type CohereGeneration struct {
	ID           string             `json:"id"`
	Text         string             `json:"text"`
	FinishReason string             `json:"finish_reason"`
	ToolCalls    []CohereToolCall   `json:"tool_calls,omitempty"`
	ChatHistory  []CohereChatMessage `json:"chat_history,omitempty"` // Bedrock Cohere doesn_t seem to return this in generations
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
	Tokens *struct { // Pointer as it might not always be present
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"tokens,omitempty"`
}

// Helper to map schema.Messages to CohereChatMessages
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
			// Cohere expects AI message text and tool_calls separately in history.
			// If AI message has text, add it.
			if m.GetContent() != "" {
				cohereMessages = append(cohereMessages, CohereChatMessage{Role: role, Message: m.GetContent()})
			}
			// If AI message has tool calls, they are part of the CHATBOT turn but not directly in CohereChatMessage.Message.
			// They are implicitly part of the turn that leads to subsequent TOOL messages.
			// For constructing the *next* request, currentToolCalls from the AIMessage are used to form ToolResults.
		case *schema.SystemMessage:
			role = "SYSTEM"
			cohereMessages = append(cohereMessages, CohereChatMessage{Role: role, Message: m.GetContent()})
		case *schema.ToolMessage:
			// This ToolMessage is a *result* to be sent back to Cohere.
			// Find the corresponding call from `currentToolCalls` (which should be from the *previous* AIMessage)
			var matchingCall *CohereToolCallResponse
			for _, prevCall := range currentToolCalls {
				if prevCall.ID == m.ToolCallID {
					var params map[string]any
					if err := json.Unmarshal([]byte(prevCall.Arguments), &params); err != nil {
						log.Printf("Warning: Could not unmarshal params for tool call ID %s: %v", prevCall.ID, err)
						params = make(map[string]any) // or skip this tool result
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
				// If not valid JSON, treat as a simple string output under a default key
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

func (bl *BedrockLLM) invokeCohereModel(ctx context.Context, _ string, messages []schema.Message, options schema.CallOptions) (json.RawMessage, error) {
	requestPayload := CohereCommandRequest{
		Stream: false,
	}

	var lastHumanMessageContent string
	var previousToolCalls []schema.ToolCall // Tool calls from the *last* AI message, to match with current ToolMessages

	if len(messages) > 0 {
		// Find the last human message for the `prompt` field
		// and the tool calls from the immediately preceding AI message for `tool_results`
		for i := len(messages) - 1; i >= 0; i-- {
			if hm, ok := messages[i].(*schema.HumanMessage); ok && lastHumanMessageContent == "" {
				lastHumanMessageContent = hm.GetContent()
			}
			if aim, ok := messages[i].(*schema.AIMessage); ok && len(previousToolCalls) == 0 {
			    if i < len(messages)-1 { // only if it's not the absolute last message
			        previousToolCalls = aim.ToolCalls
			    }
			}
		}
	}

	if lastHumanMessageContent != "" {
		requestPayload.Prompt = lastHumanMessageContent
	} else if len(messages) > 0 && messages[len(messages)-1].GetType() == schema.ToolMessageType {
	    // If the last message is a tool result, there might not be a new human prompt.
	    // Cohere might need an empty prompt or a specific instruction.
	    // For now, let's assume the chat history and tool_results are sufficient.
	    requestPayload.Prompt = " " // Or some placeholder if required
	} else {
	    return nil, fmt.Errorf("Cohere request requires a prompt (last human message) or tool results")
	}

	requestPayload.ChatHistory, requestPayload.ToolResults = mapSchemaMessagesToCohereChat(messages, previousToolCalls)


	if options.MaxTokens > 0 {
		requestPayload.MaxTokens = options.MaxTokens
	}
	if options.Temperature > 0 {
		requestPayload.Temperature = float64(options.Temperature)
	}
	if options.TopP > 0 {
		requestPayload.P = float64(options.TopP)
	}
	if options.TopK > 0 {
		requestPayload.K = options.TopK
	}
	if len(options.StopWords) > 0 {
		requestPayload.StopSequences = options.StopWords
	}

	if len(options.Tools) > 0 {
		requestPayload.Tools = mapBelugaToolsToCohere(options.Tools)
		// Handle tool_choice if Cohere Bedrock supports it via a specific param
		// For now, if tools are present, Cohere decides when to use them.
		// If `force_single_step` is needed based on ToolChoice, set it.
		if options.ToolChoice == "any" || (options.ToolChoice != "" && options.ToolChoice != "none") {
		    // Assuming if a specific tool is chosen or 'any', we might want to force a step.
		    // This needs more nuanced handling based on Cohere_s exact API for tool_choice.
		    // requestPayload.ForceSingleStep = true // Example, might not be correct for all ToolChoice values
		}
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
		content = resp.Generations[0].Text // Assuming single generation for chat
	}

	aiMsg := schema.NewAIMessage(content)
	aiMsg.AdditionalArgs = make(map[string]any)

	if resp.Meta != nil && resp.Meta.Tokens != nil {
		aiMsg.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  resp.Meta.Tokens.InputTokens,
			"output_tokens": resp.Meta.Tokens.OutputTokens,
			"total_tokens":  resp.Meta.Tokens.InputTokens + resp.Meta.Tokens.OutputTokens,
		}
	} else if resp.Meta != nil && resp.Meta.BilledUnits.InputTokens > 0 { // Fallback
		aiMsg.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  resp.Meta.BilledUnits.InputTokens,
			"output_tokens": resp.Meta.BilledUnits.OutputTokens,
			"total_tokens":  resp.Meta.BilledUnits.InputTokens + resp.Meta.BilledUnits.OutputTokens,
		}
	}

	var finalToolCalls []schema.ToolCall
	// Tool calls can be in Generations or at the top level of the response
	responseToolCalls := resp.ToolCalls
	if len(resp.Generations) > 0 && len(resp.Generations[0].ToolCalls) > 0 {
		responseToolCalls = resp.Generations[0].ToolCalls
	}

	if len(responseToolCalls) > 0 {
		finalToolCalls = make([]schema.ToolCall, len(responseToolCalls))
		for i, tc := range responseToolCalls {
			argsBytes, _ := json.Marshal(tc.Parameters)
			finalToolCalls[i] = schema.ToolCall{
				ID:        fmt.Sprintf("%s-%d-%s", tc.Name, i, bl.modelID), // Attempt more unique ID
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

func (bl *BedrockLLM) invokeCohereModelStream(ctx context.Context, _ string, messages []schema.Message, options schema.CallOptions) (*brtypes.ResponseStream, error) {
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
                    previousToolCalls = aim.ToolCalls
                }
            }
        }
    }

    if lastHumanMessageContent != "" {
        requestPayload.Prompt = lastHumanMessageContent
    } else if len(messages) > 0 && messages[len(messages)-1].GetType() == schema.ToolMessageType {
        requestPayload.Prompt = " " 
    } else {
        // If no human message and not a tool result continuation, this might be an issue.
        // However, Cohere might allow starting a stream with just chat history if it_s rich enough.
        // For now, we allow it and rely on Cohere to error if the prompt is insufficient.
        if len(messages) == 0 { // Truly empty, this is likely an error for Cohere
             return nil, fmt.Errorf("Cohere stream request requires a prompt or chat history")
        }
        requestPayload.Prompt = " " // Placeholder if only history is present
    }

    requestPayload.ChatHistory, requestPayload.ToolResults = mapSchemaMessagesToCohereChat(messages, previousToolCalls)

	if options.MaxTokens > 0 {
		requestPayload.MaxTokens = options.MaxTokens
	}
	if options.Temperature > 0 {
		requestPayload.Temperature = float64(options.Temperature)
	}
	if options.TopP > 0 {
		requestPayload.P = float64(options.TopP)
	}
	if options.TopK > 0 {
		requestPayload.K = options.TopK
	}
	if len(options.StopWords) > 0 {
		requestPayload.StopSequences = options.StopWords
	}

	if len(options.Tools) > 0 {
		requestPayload.Tools = mapBelugaToolsToCohere(options.Tools)
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Cohere Command stream request: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Cohere Command model with response stream: %w", err)
	}
	return output.Stream, nil
}

func (bl *BedrockLLM) cohereStreamChunkToAIMessageChunk(chunkBytes []byte) (*llms.AIMessageChunk, error) {
	// Cohere Bedrock stream events are directly the CohereCommandResponse structure for each event type.
	var streamEvent CohereCommandResponse
	if err := json.Unmarshal(chunkBytes, &streamEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Cohere stream event: %w, chunk: %s", err, string(chunkBytes))
	}

	chunk := llms.NewAIMessageChunk("")
	chunk.AdditionalArgs = make(map[string]any)
	var isMeaningful bool

	switch streamEvent.EventType {
	case "text-generation":
		if streamEvent.Text != "" {
			chunk.Content = streamEvent.Text
			isMeaningful = true
		}
	case "tool-calls":
		if len(streamEvent.ToolCalls) > 0 {
			schemaToolCallChunks := make([]schema.ToolCallChunk, len(streamEvent.ToolCalls))
			for i, tc := range streamEvent.ToolCalls {
				argsBytes, _ := json.Marshal(tc.Parameters)
				argsStr := string(argsBytes)
				nameStr := tc.Name
				// Cohere doesn_t provide an explicit ID for tool calls in the stream's tool-calls event.
				// We can generate one or rely on the consumer to match by name/index if needed.
				// For now, we don_t set an ID here, but the schema.ToolCallChunk has an optional ID field.
				schemaToolCallChunks[i] = schema.ToolCallChunk{
					Name:      &nameStr,
					Arguments: &argsStr,
					// Index might be relevant if Cohere provides it for multi-tool calls in one event
				}
			}
			chunk.ToolCallChunks = schemaToolCallChunks
			isMeaningful = true
		}
	case "stream-end":
		if streamEvent.FinishReason != "" {
			chunk.AdditionalArgs["finish_reason"] = streamEvent.FinishReason
			isMeaningful = true
		}
		if streamEvent.Meta != nil && streamEvent.Meta.Tokens != nil {
			chunk.AdditionalArgs["usage"] = map[string]int{
				"input_tokens":  streamEvent.Meta.Tokens.InputTokens,
				"output_tokens": streamEvent.Meta.Tokens.OutputTokens,
				"total_tokens":  streamEvent.Meta.Tokens.InputTokens + streamEvent.Meta.Tokens.OutputTokens,
			}
			isMeaningful = true
		} else if streamEvent.Meta != nil && streamEvent.Meta.BilledUnits.InputTokens > 0 {
		    chunk.AdditionalArgs["usage"] = map[string]int{
				"input_tokens":  streamEvent.Meta.BilledUnits.InputTokens,
				"output_tokens": streamEvent.Meta.BilledUnits.OutputTokens,
				"total_tokens":  streamEvent.Meta.BilledUnits.InputTokens + streamEvent.Meta.BilledUnits.OutputTokens,
			}
		    isMeaningful = true
		}
		if streamEvent.IsFinished != nil && *streamEvent.IsFinished {
			// This confirms the end. Already captured finish_reason and usage.
			isMeaningful = true
		}
	default:
		// Other event types like "search-queries-generation", "search-results", "citation-generation"
		// are not directly mapped to AIMessageChunk content/tool_calls for now.
		// log.Printf("Cohere stream: Unhandled event type 	%s	", streamEvent.EventType)
		return nil, nil // Not a chunk we process into AIMessageChunk directly
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
	if len(belugaTools) == 0 {
		return nil
	}
	cohereTools := make([]CohereTool, len(belugaTools))
	for i, t := range belugaTools {
		params := make(map[string]CohereToolParameter)
		schemaStr := t.Schema()
		var toolSchema struct {
			Type       string                              `json:"type"`
			Properties map[string]tools.JSONSchemaProperty `json:"properties"`
			Required   []string                            `json:"required"`
		}

		if schemaStr != "" && schemaStr != "{}" && schemaStr != "null" {
			if err := json.Unmarshal([]byte(schemaStr), &toolSchema); err != nil {
				log.Printf("Error unmarshalling tool schema for %s: %v. Schema: %s", t.Name(), err, schemaStr)
				// Fallback to empty properties if unmarshal fails but schema was provided
				toolSchema.Properties = make(map[string]tools.JSONSchemaProperty)
			} 
		} else {
		    // If schema is empty, still create an empty properties map
		    toolSchema.Properties = make(map[string]tools.JSONSchemaProperty)
		}

		for pk, pv := range toolSchema.Properties {
			cohereParamType := pv.Type
			// Basic type mapping, Cohere might have more specific types (e.g. integer vs number)
			switch pv.Type {
			case "integer":
				cohereParamType = "int" // Or number, check Cohere docs
			case "number":
				cohereParamType = "float" // Or number
			case "boolean":
				cohereParamType = "bool"
			case "string":
				cohereParamType = "str"
			// array and object might need more specific handling or pass through if Cohere supports them directly
			}
			params[pk] = CohereToolParameter{
				Description: pv.Description,
				Type:        cohereParamType,
				Required:    contains(toolSchema.Required, pk),
			}
		}
		cohereTools[i] = CohereTool{
			Name:                 t.Name(),
			Description:          t.Description(),
			ParameterDefinitions: params,
		}
	}
	return cohereTools
}

// Helper function to check if a slice contains a string (already defined, but keep local if needed)
// func contains(slice []string, item string) bool {
// 	for _, s := range slice {
// 		if s == item {
// 			return true
// 		}
// 	}
// 	return false
// }

