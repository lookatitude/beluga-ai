// Package bedrock contains provider-specific logic for AI21 Labs Jurassic-2 models on AWS Bedrock.
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

// AI21Jurassic2Request represents the request payload for AI21 Jurassic-2 models on Bedrock.
// See: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-jurassic2.html
type AI21Jurassic2Request struct {
	Prompt           string   `json:"prompt"`
	MaxTokens        int      `json:"maxTokens,omitempty"`
	Temperature      float64  `json:"temperature,omitempty"` // Changed to float64 for consistency with schema.CallOptions
	TopP             float64  `json:"topP,omitempty"`         // Changed to float64
	StopSequences    []string `json:"stopSequences,omitempty"`
	CountPenalty     *AI21Penalty `json:"countPenalty,omitempty"`
	PresencePenalty  *AI21Penalty `json:"presencePenalty,omitempty"`
	FrequencyPenalty *AI21Penalty `json:"frequencyPenalty,omitempty"`
	NumResults       int      `json:"numResults,omitempty"` // Default 1
}

// AI21Penalty defines penalty structures for AI21 models.
type AI21Penalty struct {
	Scale            float64 `json:"scale"` // Changed to float64
	ApplyToNumbers   bool    `json:"applyToNumbers,omitempty"`
	ApplyToPunctuation bool  `json:"applyToPunctuation,omitempty"`
	ApplyToStopwords bool    `json:"applyToStopwords,omitempty"`
	ApplyToWhitespaces bool  `json:"applyToWhitespaces,omitempty"`
	ApplyToEmojis    bool    `json:"applyToEmojis,omitempty"`
}

// AI21Jurassic2Response represents the response payload from AI21 Jurassic-2 models on Bedrock.
type AI21Jurassic2Response struct {
	ID          string                `json:"id"`
	Prompt      AI21PromptDetails     `json:"prompt"`
	Completions []AI21Completion      `json:"completions"`
}

// AI21PromptDetails contains details about the prompt in the response.
type AI21PromptDetails struct {
	Text         string        `json:"text"`
	Tokens       []AI21Token   `json:"tokens,omitempty"` // If requested
}

// AI21Completion represents a single completion from the AI21 model.
type AI21Completion struct {
	Data         AI21CompletionData `json:"data"`
	FinishReason AI21FinishReason   `json:"finishReason"`
}

// AI21CompletionData contains the actual generated text and tokens.
type AI21CompletionData struct {
	Text   string      `json:"text"`
	Tokens []AI21Token `json:"tokens,omitempty"` // If requested
}

// AI21FinishReason indicates why the generation stopped.
type AI21FinishReason struct {
	Reason string `json:"reason"` // e.g., "length", "endoftext", "stop"
}

// AI21Token represents a token with its text and log probability (if requested).
type AI21Token struct {
	GeneratedToken struct {
		Token      string  `json:"token"`
		Logprob    float64 `json:"logprob"`
		RawLogprob float64 `json:"raw_logprob"`
	} `json:"generatedToken"`
	TopTokens []any `json:"topTokens,omitempty"` // Can be complex, using any
	TextRange struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"textRange"`
}

// invokeAI21Jurassic2Model handles the invocation of AI21 Jurassic-2 models.
// Note: AI21 Jurassic-2 models are primarily text completion models and do not natively support chat history or tools in the same way as chat models.
// We will adapt the prompt for a basic chat-like interaction if multiple messages are provided.
func (bl *BedrockLLM) invokeAI21Jurassic2Model(ctx context.Context, _ string, messages []schema.Message, options schema.CallOptions) (json.RawMessage, error) {
	var combinedPrompt string
	if len(messages) > 0 {
		// For AI21, combine messages into a single prompt string, trying to mimic a conversation.
		// This is a simplification as it_s not a true chat model.
		for _, msg := range messages {
			switch m := msg.(type) {
			case *schema.HumanMessage:
				combinedPrompt += fmt.Sprintf("\nHuman: %s", m.GetContent())
			case *schema.AIMessage:
				combinedPrompt += fmt.Sprintf("\nAssistant: %s", m.GetContent())
			case *schema.SystemMessage:
				combinedPrompt = fmt.Sprintf("%s%s", m.GetContent(), combinedPrompt) // Prepend system message
			// ToolMessages are not directly applicable here in the prompt construction for AI21
			}
		}
		// Ensure the prompt ends with an Assistant marker if the last message was Human, to guide completion.
		if messages[len(messages)-1].GetType() == schema.HumanMessageType {
		    combinedPrompt += "\nAssistant:"
		}
	} else {
		return nil, fmt.Errorf("no messages provided for AI21 Jurassic-2 invocation")
	}

	requestPayload := AI21Jurassic2Request{
		Prompt: combinedPrompt,
	}

	if options.MaxTokens > 0 {
		requestPayload.MaxTokens = options.MaxTokens
	}
	if options.Temperature > 0 {
		requestPayload.Temperature = float64(options.Temperature) // Cast from schema_s float32
	}
	if options.TopP > 0 {
		requestPayload.TopP = float64(options.TopP) // Cast from schema_s float32
	}
	if len(options.StopWords) > 0 {
		requestPayload.StopSequences = options.StopWords
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AI21 Jurassic-2 request: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModel(ctx, bl.createInvokeModelInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke AI21 Jurassic-2 model: %w", err)
	}
	return output.Body, nil
}

func (bl *BedrockLLM) ai21Jurassic2ResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var resp AI21Jurassic2Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI21 Jurassic-2 response: %w. Body: %s", err, string(body))
	}

	content := ""
	finishReason := ""
	if len(resp.Completions) > 0 {
		content = resp.Completions[0].Data.Text
		finishReason = resp.Completions[0].FinishReason.Reason
	}

	aiMsg := schema.NewAIMessage(content)
	aiMsg.AdditionalArgs = make(map[string]any)
	aiMsg.AdditionalArgs["finish_reason"] = finishReason
	aiMsg.AdditionalArgs["id"] = resp.ID

	// AI21 Jurassic-2 API on Bedrock does not directly return token counts in the main response body for InvokeModel.
	// Token counts are typically available in the output stream for InvokeModelWithResponseStream.
	aiMsg.AdditionalArgs["usage_note"] = "Token usage for AI21 InvokeModel is not directly available in the response payload; use streaming for token counts."

	// AI21 Jurassic-2 does not support tool calls in the Bedrock API.
	aiMsg.ToolCalls = nil

	return aiMsg, nil
}

// AI21Jurassic2StreamResponse represents a chunk in the streaming response.
// Based on Bedrock documentation, the stream for AI21 provides `outputText` and `amazon-bedrock-invocationMetrics`.
type AI21Jurassic2StreamResponse struct {
	OutputText      string  `json:"outputText,omitempty"`
	CompletionReason *string `json:"completionReason,omitempty"` // Appears in the last chunk
	// Bedrock adds these metrics, typically in the last chunk or a separate metadata chunk.
	InputTokenCount  *int `json:"amazon-bedrock-invocationMetrics_inputTokenCount,omitempty"`
	OutputTokenCount *int `json:"amazon-bedrock-invocationMetrics_outputTokenCount,omitempty"`
}

func (bl *BedrockLLM) invokeAI21Jurassic2ModelStream(ctx context.Context, _ string, messages []schema.Message, options schema.CallOptions) (*brtypes.ResponseStream, error) {
	var combinedPrompt string
	if len(messages) > 0 {
		for _, msg := range messages {
			switch m := msg.(type) {
			case *schema.HumanMessage:
				combinedPrompt += fmt.Sprintf("\nHuman: %s", m.GetContent())
			case *schema.AIMessage:
				combinedPrompt += fmt.Sprintf("\nAssistant: %s", m.GetContent())
			case *schema.SystemMessage:
				combinedPrompt = fmt.Sprintf("%s%s", m.GetContent(), combinedPrompt)
			}
		}
		if messages[len(messages)-1].GetType() == schema.HumanMessageType {
		    combinedPrompt += "\nAssistant:"
		}
	} else {
		return nil, fmt.Errorf("no messages provided for AI21 Jurassic-2 stream invocation")
	}

	requestPayload := AI21Jurassic2Request{
		Prompt: combinedPrompt,
	}

	if options.MaxTokens > 0 {
		requestPayload.MaxTokens = options.MaxTokens
	}
	if options.Temperature > 0 {
		requestPayload.Temperature = float64(options.Temperature)
	}
	if options.TopP > 0 {
		requestPayload.TopP = float64(options.TopP)
	}
	if len(options.StopWords) > 0 {
		requestPayload.StopSequences = options.StopWords
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AI21 Jurassic-2 request for streaming: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke AI21 Jurassic-2 model with response stream: %w", err)
	}
	return output.Stream, nil
}

func (bl *BedrockLLM) ai21Jurassic2StreamChunkToAIMessageChunk(chunkBytes []byte) (*llms.AIMessageChunk, error) {
	var streamResp AI21Jurassic2StreamResponse
	if err := json.Unmarshal(chunkBytes, &streamResp); err != nil {
		// Log the error and the problematic chunk for debugging, but don_t necessarily stop the stream if it_s a non-fatal issue.
		log.Printf("Warning: failed to unmarshal AI21 stream chunk: %v. Chunk: %s", err, string(chunkBytes))
		// Return an empty chunk or nil if the chunk is not processable as main content.
		return nil, nil
	}

	chunk := llms.NewAIMessageChunk(streamResp.OutputText)
	chunk.AdditionalArgs = make(map[string]any)
	var isMeaningful bool

	if streamResp.OutputText != "" {
	    isMeaningful = true
	}

	if streamResp.CompletionReason != nil {
		chunk.AdditionalArgs["finish_reason"] = *streamResp.CompletionReason
		isMeaningful = true
	}

	inputTokens := 0
	outputTokens := 0
	usageFound := false

	if streamResp.InputTokenCount != nil {
		inputTokens = *streamResp.InputTokenCount
		usageFound = true
	}
	if streamResp.OutputTokenCount != nil {
		outputTokens = *streamResp.OutputTokenCount
		usageFound = true
	}

	if usageFound {
		chunk.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_tokens":  inputTokens + outputTokens,
		}
		isMeaningful = true
	}

    // AI21 Jurassic-2 does not support tool calls via Bedrock API.
    chunk.ToolCallChunks = nil

	if !isMeaningful {
	    return nil, nil // Not a chunk we process into AIMessageChunk directly
	}
    if len(chunk.AdditionalArgs) == 0 {
        chunk.AdditionalArgs = nil
    }
	return chunk, nil
}

