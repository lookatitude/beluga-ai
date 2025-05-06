// Package bedrock contains provider-specific logic for Meta Llama models on AWS Bedrock.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"

	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
)

// MetaLlamaRequest represents the request payload for Meta Llama models on Bedrock.
// See: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-meta.html
type MetaLlamaRequest struct {
	Prompt      string  `json:"prompt"`
	MaxGenLen   int     `json:"max_gen_len,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	TopP        float32 `json:"top_p,omitempty"`
}

// MetaLlamaResponse represents the response payload from Meta Llama models on Bedrock.
type MetaLlamaResponse struct {
	Generation           string `json:"generation"`
	PromptTokenCount     int    `json:"prompt_token_count"`
	GenerationTokenCount int    `json:"generation_token_count"`
	StopReason           string `json:"stop_reason"` // e.g., "stop", "length"
}

func (bl *BedrockLLM) invokeMetaLlamaModel(ctx context.Context, prompt string, options schema.CallOptions) (json.RawMessage, error) {
	requestPayload := MetaLlamaRequest{
		Prompt: prompt,
	}

	if options.MaxTokens > 0 {
		requestPayload.MaxGenLen = options.MaxTokens
	}
	if options.Temperature > 0 {
		requestPayload.Temperature = float32(options.Temperature) // Ensure float32
	}
	if options.TopP > 0 {
		requestPayload.TopP = float32(options.TopP) // Ensure float32
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Meta Llama request: %w", err)
	}

	output, err := bl.client.InvokeModel(ctx, bl.createInvokeModelInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Meta Llama model: %w", err)
	}
	return output.Body, nil
}

func (bl *BedrockLLM) metaLlamaResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var resp MetaLlamaResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Meta Llama response: %w. Body: %s", err, string(body))
	}

	aiMsg := schema.NewAIMessage(resp.Generation)
	aiMsg.AdditionalArgs = make(map[string]any)
	aiMsg.AdditionalArgs["stop_reason"] = resp.StopReason
	aiMsg.AdditionalArgs["usage"] = map[string]int{
		"input_tokens":  resp.PromptTokenCount, // Bedrock uses input_tokens for Meta
		"output_tokens": resp.GenerationTokenCount,
		"total_tokens":  resp.PromptTokenCount + resp.GenerationTokenCount,
	}

	return aiMsg, nil
}

func (bl *BedrockLLM) invokeMetaLlamaModelStream(ctx context.Context, prompt string, options schema.CallOptions) (*brtypes.ResponseStream, error) {
	requestPayload := MetaLlamaRequest{
		Prompt: prompt,
	}

	if options.MaxTokens > 0 {
		requestPayload.MaxGenLen = options.MaxTokens
	}
	if options.Temperature > 0 {
		requestPayload.Temperature = float32(options.Temperature)
	}
	if options.TopP > 0 {
		requestPayload.TopP = float32(options.TopP)
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Meta Llama request for streaming: %w", err)
	}

	output, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(body))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Meta Llama model with response stream: %w", err)
	}
	return output.Stream, nil // output.Stream is already *brtypes.ResponseStream
}

func (bl *BedrockLLM) metaLlamaStreamChunkToAIMessageChunk(chunkPayload brtypes.PayloadPart) (*llms.AIMessageChunk, error) {
	var resp MetaLlamaResponse
	if err := json.Unmarshal(chunkPayload.Bytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Meta Llama stream chunk: %w. Chunk: %s", err, string(chunkPayload.Bytes))
	}

	// Meta Llama streams the full response object in each chunk, with `generation` being the delta.
	// The final chunk contains the full generation and metadata.
	// We only care about the `generation` field for content delta.

	chunk := llms.NewAIMessageChunk(resp.Generation)
	chunk.AdditionalArgs = make(map[string]any)
	var isMeaningful bool = (resp.Generation != "")

	if resp.StopReason != "" {
		chunk.AdditionalArgs["stop_reason"] = resp.StopReason
		isMeaningful = true
	}
	// Usage is typically in the last chunk for Meta Llama on Bedrock
	if resp.PromptTokenCount > 0 || resp.GenerationTokenCount > 0 {
		chunk.AdditionalArgs["usage"] = map[string]int{
			"input_tokens":  resp.PromptTokenCount,
			"output_tokens": resp.GenerationTokenCount,
			"total_tokens":  resp.PromptTokenCount + resp.GenerationTokenCount,
		}
		isMeaningful = true
	}

	if !isMeaningful {
	    // If only stop_reason is present without new generation content, it might be the final metadata chunk.
	    // If it has usage, it's also meaningful.
	    // If it has neither new generation, nor stop_reason, nor usage, it might be an empty chunk we can ignore.
	    // However, an empty generation string with a stop_reason IS meaningful.
	    // Let's return nil only if truly empty and no metadata of interest.
	    if resp.Generation == "" && resp.StopReason == "" && resp.PromptTokenCount == 0 && resp.GenerationTokenCount == 0 {
	        return nil, nil
	    }
	}
    if len(chunk.AdditionalArgs) == 0 {
        chunk.AdditionalArgs = nil
    }

	return chunk, nil
}

