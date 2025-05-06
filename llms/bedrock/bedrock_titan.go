// Package bedrock contains provider-specific logic for Amazon Titan Text models on AWS Bedrock.
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

// TitanTextRequest represents the request payload for Amazon Titan Text models (e.g., titan-text-express-v1, titan-text-lite-v1, titan-text-agile-v1).
// See: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-titan-text.html
type TitanTextRequest struct {
	InputText            string                  `json:"inputText"`
	TextGenerationConfig *TitanTextConfig        `json:"textGenerationConfig,omitempty"`
}

// TitanTextConfig holds the configuration parameters for Titan Text generation.
type TitanTextConfig struct {
	MaxTokenCount      int      `json:"maxTokenCount,omitempty"`
	Temperature        float64  `json:"temperature,omitempty"` // Changed to float64
	TopP               float64  `json:"topP,omitempty"`         // Changed to float64
	StopSequences      []string `json:"stopSequences,omitempty"`
}

// TitanTextResponse represents the response payload from Amazon Titan Text models.
type TitanTextResponse struct {
	InputTextTokenCount int                   `json:"inputTextTokenCount"`
	Results             []TitanTextResult     `json:"results"`
}

// TitanTextResult represents a single generation result from a Titan Text model.
type TitanTextResult struct {
	TokenCount       int    `json:"tokenCount"`
	OutputText       string `json:"outputText"`
	CompletionReason string `json:"completionReason"` // e.g., "FINISH", "LENGTH", "MAX_TOKENS"
}

// invokeTitanTextModel handles the invocation of Amazon Titan Text models.
// Note: Titan Text models are primarily text completion models and do not natively support chat history or tools in the same way as chat models.
// We will adapt the prompt for a basic chat-like interaction if multiple messages are provided.
func (bl *BedrockLLM) invokeTitanTextModel(ctx context.Context, _ string, messages []schema.Message, options schema.CallOptions) (json.RawMessage, error) {
	var combinedPrompt string
	if len(messages) > 0 {
		for _, msg := range messages {
			switch m := msg.(type) {
			case *schema.HumanMessage:
				combinedPrompt += fmt.Sprintf("\nHuman: %s", m.GetContent())
			case *schema.AIMessage:
				combinedPrompt += fmt.Sprintf("\nAssistant: %s", m.GetContent())
			case *schema.SystemMessage:
				combinedPrompt = fmt.Sprintf("%s%s", m.GetContent(), combinedPrompt) // Prepend system message
			}
		}
		if messages[len(messages)-1].GetType() == schema.HumanMessageType {
		    combinedPrompt += "\nAssistant:"
		}
	} else {
		return nil, fmt.Errorf("no messages provided for Titan Text invocation")
	}

	config := &TitanTextConfig{}
	populatedConfig := false

	if options.MaxTokens > 0 {
		config.MaxTokenCount = options.MaxTokens
		populatedConfig = true
	}
	if options.Temperature > 0 {
		config.Temperature = float64(options.Temperature) // Use float64 from schema.CallOptions
		populatedConfig = true
	}
	if options.TopP > 0 {
		config.TopP = float64(options.TopP) // Use float64 from schema.CallOptions
		populatedConfig = true
	}
	if len(options.StopWords) > 0 {
		config.StopSequences = options.StopWords
		populatedConfig = true
	}

	requestPayload := TitanTextRequest{
		InputText: combinedPrompt,
	}
	if populatedConfig {
		requestPayload.TextGenerationConfig = config
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Titan Text request: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModel(ctx, bl.createInvokeModelInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Titan Text model: %w", err)
	}
	return output.Body, nil
}

func (bl *BedrockLLM) titanTextResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var resp TitanTextResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Titan Text response: %w. Body: %s", err, string(body))
	}

	content := ""
	completionReason := ""
	completionTokens := 0

	if len(resp.Results) > 0 {
		content = resp.Results[0].OutputText
		completionReason = resp.Results[0].CompletionReason
		completionTokens = resp.Results[0].TokenCount
	}

	aiMsg := schema.NewAIMessage(content)
	aiMsg.AdditionalArgs = make(map[string]any)
	aiMsg.AdditionalArgs["finish_reason"] = completionReason
	aiMsg.AdditionalArgs["usage"] = map[string]int{
		"input_tokens":  resp.InputTextTokenCount,
		"output_tokens": completionTokens,
		"total_tokens":  resp.InputTextTokenCount + completionTokens,
	}

	// Titan Text models do not support tool calls via Bedrock API.
	aiMsg.ToolCalls = nil

	return aiMsg, nil
}

// TitanTextStreamResponse represents a chunk in the streaming response for Titan Text.
// Example: {"outputText": "...", "index": 0, "totalOutputTextTokenCount": null, "completionReason": null, "amazon-bedrock-invocationMetrics": {"inputTokenCount": X, "outputTokenCount": Y}}
type TitanTextStreamResponse struct {
	OutputText                string  `json:"outputText,omitempty"`
	Index                     *int    `json:"index,omitempty"` // Typically 0 for single generation
	TotalOutputTextTokenCount *int    `json:"totalOutputTextTokenCount,omitempty"` // Null until the end
	CompletionReason          *string `json:"completionReason,omitempty"`         // Null until the end
	InvocationMetrics *struct {
		InputTokenCount  int `json:"inputTokenCount"`
		OutputTokenCount int `json:"outputTokenCount"`
	} `json:"amazon-bedrock-invocationMetrics,omitempty"`
}

func (bl *BedrockLLM) invokeTitanTextModelStream(ctx context.Context, _ string, messages []schema.Message, options schema.CallOptions) (*brtypes.ResponseStream, error) {
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
		return nil, fmt.Errorf("no messages provided for Titan Text stream invocation")
	}

	config := &TitanTextConfig{}
	populatedConfig := false

	if options.MaxTokens > 0 {
		config.MaxTokenCount = options.MaxTokens
		populatedConfig = true
	}
	if options.Temperature > 0 {
		config.Temperature = float64(options.Temperature)
		populatedConfig = true
	}
	if options.TopP > 0 {
		config.TopP = float64(options.TopP)
		populatedConfig = true
	}
	if len(options.StopWords) > 0 {
		config.StopSequences = options.StopWords
		populatedConfig = true
	}

	requestPayload := TitanTextRequest{
		InputText: combinedPrompt,
	}
	if populatedConfig {
		requestPayload.TextGenerationConfig = config
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Titan Text request for streaming: %w. Payload: %+v", err, requestPayload)
	}

	output, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Titan Text model with response stream: %w", err)
	}
	return output.Stream, nil
}

func (bl *BedrockLLM) titanTextStreamChunkToAIMessageChunk(chunkBytes []byte) (*llms.AIMessageChunk, error) {
	var streamResp TitanTextStreamResponse
	if err := json.Unmarshal(chunkBytes, &streamResp); err != nil {
		log.Printf("Warning: failed to unmarshal Titan Text stream chunk: %v. Chunk: %s", err, string(chunkBytes))
		return nil, nil // Or an empty chunk if preferred
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

	if streamResp.InvocationMetrics != nil {
		chunk.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  streamResp.InvocationMetrics.InputTokenCount,
			"output_tokens": streamResp.InvocationMetrics.OutputTokenCount,
			"total_tokens":  streamResp.InvocationMetrics.InputTokenCount + streamResp.InvocationMetrics.OutputTokenCount,
		}
		isMeaningful = true
	}

    // Titan Text models do not support tool calls via Bedrock API.
    chunk.ToolCallChunks = nil

	if !isMeaningful {
	    return nil, nil
	}
    if len(chunk.AdditionalArgs) == 0 {
        chunk.AdditionalArgs = nil
    }
	return chunk, nil
}

