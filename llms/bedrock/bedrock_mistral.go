// Package bedrock contains provider-specific logic for Mistral AI models on AWS Bedrock.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
)

// MistralRequest represents the request payload for Mistral AI models (e.g., mistral.mistral-7b-instruct-v0:2, mistral.mixtral-8x7b-instruct-v0:1, mistral.mistral-large-2402-v1:0) on Bedrock.
// See: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-mistral.html
type MistralRequest struct {
	Prompt        string   `json:"prompt"` // Required
	MaxTokens     *int     `json:"max_tokens,omitempty"`
	Stop          []string `json:"stop,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"` // Changed to float64
	TopP          *float64 `json:"top_p,omitempty"`         // Changed to float64
	TopK          *int     `json:"top_k,omitempty"`
}

// MistralResponse represents the response payload from Mistral AI models on Bedrock (non-streaming).
type MistralResponse struct {
	Outputs []MistralOutput `json:"outputs"`
}

// MistralOutput represents a single output from a Mistral model.
type MistralOutput struct {
	Text         string `json:"text"`
	StopReason   string `json:"stop_reason"` // e.g., "stop", "length", "tool_calls"
	// Mistral on Bedrock (especially newer models like Large) might support tool calls.
	// If so, the structure would be: "tool_calls": [{"id": "...", "function": {"name": "...", "arguments": "..."}}]
	// This needs to be confirmed with specific model documentation if tool use is intended.
}

// MistralStreamResponse represents a chunk in the streaming response for Mistral AI models.
// This structure attempts to unify potential stream chunk formats.
// Actual format can vary: some chunks are deltas, some are full messages, some are metadata.
type MistralStreamResponse struct {
	Outputs           []MistralOutput `json:"outputs,omitempty"` // For text delta or full output in a chunk
	Type              *string         `json:"type,omitempty"`    // Some Mistral APIs use a type field (e.g. "message_delta", "message_stop")
	Delta             *MistralDelta   `json:"delta,omitempty"`  // For delta updates (common in some Mistral API direct integrations)
	Message           *MistralMessage `json:"message,omitempty"`// For full message updates (less common in Bedrock stream chunks, but possible)
	InvocationMetrics *struct {
		InputTokenCount  int `json:"inputTokenCount"`
		OutputTokenCount int `json:"outputTokenCount"
	} `json:"amazon-bedrock-invocationMetrics,omitempty"`
	// Mistral specific streaming fields (if they differ from the general `Outputs`)
	// For example, Mistral API might send `choices: [{ delta: { content: "..."}}]`
	// Bedrock might normalize this. The `Outputs` field is based on Bedrock_s typical InvokeModel response structure.
	// If Bedrock stream for Mistral is different, this needs adjustment.
}

// MistralDelta is used if the stream provides delta updates (more common with direct Mistral API).
// Bedrock might simplify this to just `outputText` in `MistralOutput` within `MistralStreamResponse`.
type MistralDelta struct {
	Content   *string `json:"content,omitempty"`
	Role      *string `json:"role,omitempty"` // Typically "assistant"
	ToolCalls []any   `json:"tool_calls,omitempty"` // Placeholder for tool call deltas if supported
}

// MistralMessage is used if the stream provides full message updates.
type MistralMessage struct {
	ID        *string `json:"id,omitempty"`
	Type      *string `json:"type,omitempty"` // e.g. "message"
	Role      *string `json:"role,omitempty"`
	Content   []any   `json:"content,omitempty"` // Can be complex with text and tool calls
	Model     *string `json:"model,omitempty"`
	StopReason *string `json:"stop_reason,omitempty"`
	Usage     *any    `json:"usage,omitempty"` // Placeholder for usage info if in message
}

// invokeMistralModel handles the invocation of Mistral AI models.
// Note: Mistral models on Bedrock (like Instruct models) are designed for conversational prompts.
// The prompt should be formatted accordingly (e.g., with `<s>[INST] ... [/INST]`).
func (bl *BedrockLLM) invokeMistralModel(ctx context.Context, _ string, messages []schema.Message, options schema.CallOptions) (json.RawMessage, error) {
	// Construct prompt for Mistral Instruct models
	// See: https://docs.aws.amazon.com/bedrock/latest/userguide/prompt-templates-mistral.html
	var mistralPrompt string
	if len(messages) == 1 && messages[0].GetType() == schema.HumanMessageType {
		// Single turn, simple case
		mistralPrompt = fmt.Sprintf("<s>[INST] %s [/INST]", messages[0].GetContent())
	} else {
		// Multi-turn. System messages are tricky with Mistral_s INST format if not first.
		// A common approach is to prepend system instructions before the first [INST]
		// or integrate them into the first user turn if possible.
		var builtPrompt string
		firstUserTurn := true
		for _, msg := range messages {
			switch m := msg.(type) {
			case *schema.SystemMessage:
				// Prepend system message if it_s at the beginning or before the first user turn.
				// Mistral doesn_t have a dedicated system role in its INST/regular message flow.
				builtPrompt = m.GetContent() + "\n" + builtPrompt // Prepend
			case *schema.HumanMessage:
				if firstUserTurn {
					builtPrompt += fmt.Sprintf("<s>[INST] %s [/INST]", m.GetContent())
					firstUserTurn = false
				} else {
					builtPrompt += fmt.Sprintf("[INST] %s [/INST]", m.GetContent()) // No <s> for subsequent turns
				}
			case *schema.AIMessage:
				// AIMessage content should not contain [INST] or [/INST]
				builtPrompt += fmt.Sprintf("%s</s>", m.GetContent()) // Add </s> after AI response
			// ToolMessages are not directly part of Mistral prompt format this way.
			// Tool calls/results would be handled via specific tool use schemas if supported by the model version on Bedrock.
			}
		}
		mistralPrompt = builtPrompt
	}

	requestPayload := MistralRequest{
		Prompt: mistralPrompt,
	}

	if options.MaxTokens > 0 {
		mt := options.MaxTokens
		requestPayload.MaxTokens = &mt
	}
	if options.Temperature > 0 {
		temp := float64(options.Temperature) // Use float64
		requestPayload.Temperature = &temp
	}
	if options.TopP > 0 {
		topP := float64(options.TopP) // Use float64
		requestPayload.TopP = &topP
	}
	if options.TopK > 0 {
		topK := options.TopK
		requestPayload.TopK = &topK
	}
	if len(options.StopWords) > 0 {
		requestPayload.Stop = options.StopWords
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Mistral request: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModel(ctx, bl.createInvokeModelInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Mistral model: %w", err)
	}
	return output.Body, nil
}

func (bl *BedrockLLM) mistralResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var resp MistralResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Mistral response: %w. Body: %s", err, string(body))
	}

	content := ""
	stopReason := ""
	if len(resp.Outputs) > 0 {
		content = resp.Outputs[0].Text
		stopReason = resp.Outputs[0].StopReason
	}

	aiMsg := schema.NewAIMessage(content)
	aiMsg.AdditionalArgs = make(map[string]any)
	aiMsg.AdditionalArgs["finish_reason"] = stopReason

	// For Mistral non-streaming on Bedrock, token counts are not part of the direct model response body.
	// They are available via CloudWatch metrics or Bedrock_s GetModelInvocationLoggingConfiguration.
	aiMsg.AdditionalArgs["usage_note"] = "Token usage for Mistral InvokeModel is typically available via Bedrock metrics, not in the direct response payload; use streaming for token counts if available in stream."

	// Mistral tool use on Bedrock (e.g., Mistral Large) would require parsing specific tool_calls structures if present in resp.Outputs[0].
	// For now, assuming no complex tool call objects are directly in this basic response structure.
	aiMsg.ToolCalls = nil

	return aiMsg, nil
}

func (bl *BedrockLLM) invokeMistralModelStream(ctx context.Context, _ string, messages []schema.Message, options schema.CallOptions) (*brtypes.ResponseStream, error) {
	var mistralPrompt string
	if len(messages) == 1 && messages[0].GetType() == schema.HumanMessageType {
		mistralPrompt = fmt.Sprintf("<s>[INST] %s [/INST]", messages[0].GetContent())
	} else {
		var builtPrompt string
		firstUserTurn := true
		for _, msg := range messages {
			switch m := msg.(type) {
			case *schema.SystemMessage:
				builtPrompt = m.GetContent() + "\n" + builtPrompt
			case *schema.HumanMessage:
				if firstUserTurn {
					builtPrompt += fmt.Sprintf("<s>[INST] %s [/INST]", m.GetContent())
					firstUserTurn = false
				} else {
					builtPrompt += fmt.Sprintf("[INST] %s [/INST]", m.GetContent())
				}
			case *schema.AIMessage:
				builtPrompt += fmt.Sprintf("%s</s>", m.GetContent())
			}
		}
		mistralPrompt = builtPrompt
	}

	requestPayload := MistralRequest{
		Prompt: mistralPrompt,
	}

	if options.MaxTokens > 0 {
		mt := options.MaxTokens
		requestPayload.MaxTokens = &mt
	}
	if options.Temperature > 0 {
		temp := float64(options.Temperature)
		requestPayload.Temperature = &temp
	}
	if options.TopP > 0 {
		topP := float64(options.TopP)
		requestPayload.TopP = &topP
	}
	if options.TopK > 0 {
		topK := options.TopK
		requestPayload.TopK = &topK
	}
	if len(options.StopWords) > 0 {
		requestPayload.Stop = options.StopWords
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Mistral request for streaming: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Mistral model with response stream: %w", err)
	}
	return output.Stream, nil
}

func (bl *BedrockLLM) mistralStreamChunkToAIMessageChunk(chunkBytes []byte) (*llms.AIMessageChunk, error) {
	var streamResp MistralStreamResponse
	if err := json.Unmarshal(chunkBytes, &streamResp); err != nil {
		log.Printf("Warning: failed to unmarshal Mistral stream chunk: %v. Chunk: %s", err, string(chunkBytes))
		return nil, nil
	}

	chunk := llms.NewAIMessageChunk("")
	chunk.AdditionalArgs = make(map[string]any)
	var isMeaningful bool

	// Check for text content in outputs (Bedrock_s typical way)
	if len(streamResp.Outputs) > 0 && streamResp.Outputs[0].Text != "" {
		chunk.Content = streamResp.Outputs[0].Text
		isMeaningful = true
	}

	// Check for stop reason in outputs
	if len(streamResp.Outputs) > 0 && streamResp.Outputs[0].StopReason != "" {
		chunk.AdditionalArgs["finish_reason"] = streamResp.Outputs[0].StopReason
		isMeaningful = true
	}

	// Check for Bedrock invocation metrics (usually in the last chunk)
	if streamResp.InvocationMetrics != nil {
		chunk.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  streamResp.InvocationMetrics.InputTokenCount,
			"output_tokens": streamResp.InvocationMetrics.OutputTokenCount,
			"total_tokens":  streamResp.InvocationMetrics.InputTokenCount + streamResp.InvocationMetrics.OutputTokenCount,
		}
		isMeaningful = true
	}

	// Mistral tool use on Bedrock (e.g., Mistral Large) might have tool_calls in streamResp.Outputs[0].ToolCalls
	// or via a delta structure if Bedrock passes Mistral_s native stream more directly.
	// For now, assuming no complex tool call objects are directly in this basic stream structure without further model-specific parsing.
	chunk.ToolCallChunks = nil

	if !isMeaningful {
	    return nil, nil
	}
    if len(chunk.AdditionalArgs) == 0 {
        chunk.AdditionalArgs = nil
    }
	return chunk, nil
}

