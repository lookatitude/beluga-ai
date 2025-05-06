// Package bedrock provides an implementation of the llms.ChatModel interface
// using AWS Bedrock Runtime, with support for multiple model providers.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
)

// MetaLlamaRequest represents the request payload for Meta Llama models on Bedrock.
// Based on: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-meta.html
type MetaLlamaRequest struct {
	Prompt      string    `json:"prompt"`
	Temperature float32   `json:"temperature,omitempty"`
	TopP        float32   `json:"top_p,omitempty"`
	MaxGenLen   int       `json:"max_gen_len,omitempty"` // Max tokens to generate
	// Note: Llama 3.2+ can include `images: Optional[List[str]]` for multimodal inputs.
	// This basic implementation focuses on text for now.
	// Tools are not directly part of this basic request structure for InvokeModel.
	// Tool use with Llama on Bedrock is often better handled via the Converse API or requires specific prompt engineering.
}

// MetaLlamaResponse represents the full response payload from Meta Llama models on Bedrock.
type MetaLlamaResponse struct {
	Generation          string `json:"generation"`
	PromptTokenCount    *int   `json:"prompt_token_count,omitempty"`    // Optional, may not always be present
	GenerationTokenCount *int   `json:"generation_token_count,omitempty"` // Optional
	StopReason          string `json:"stop_reason,omitempty"`          // e.g., "stop", "length", "content_filter"
	// Note: Llama 3 on Bedrock (InvokeModel) does not seem to return structured tool calls directly in this response.
	// It might output text that needs to be parsed for tool invocations if using prompt-based tool use.
}

// MetaLlamaStreamResponseChunk represents a chunk in a streaming response from Meta Llama.
// This is the structure of the JSON object within the `bytes` field of brtypes.PayloadPart.
type MetaLlamaStreamResponseChunk struct {
	Generation          string `json:"generation"`                     // The text chunk
	PromptTokenCount    *int   `json:"prompt_token_count,omitempty"`    // Usually in the first chunk or a separate metadata chunk
	GenerationTokenCount *int   `json:"generation_token_count,omitempty"` // Incremental or final count
	StopReason          string `json:"stop_reason,omitempty"`          // Present in the last chunk
	// amazon-bedrock-invocationMetrics might appear in some stream events for token counts
	AmazonBedrockInvocationMetrics *struct {
		InputTokenCount  *int `json:"inputTokenCount"`
		OutputTokenCount *int `json:"outputTokenCount"`
	} `json:"amazon-bedrock-invocationMetrics,omitempty"`
}

// invokeMetaLlamaModel sends a request to a Meta Llama model on Bedrock.
func (bl *BedrockLLM) invokeMetaLlamaModel(ctx context.Context, prompt string, options schema.CallOptions) (json.RawMessage, error) {
	llamaPrompt := formatMessagesToLlamaPrompt(options.Messages)

	payload := MetaLlamaRequest{
		Prompt:      llamaPrompt,
		Temperature: options.Temperature,
		TopP:        options.TopP,
		MaxGenLen:   options.MaxTokens, // Map MaxTokens to MaxGenLen
	}

	// Tool binding for Llama via InvokeModel is typically done through prompt engineering.
	// The tools.Tool definitions would need to be formatted into the prompt itself.
	// This is a complex topic and depends on the specific Llama model version and fine-tuning.
	// For now, we assume tools are handled by the prompt formatting if used.

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Meta Llama request: %w", err)
	}

	resp, err := bl.client.InvokeModel(ctx, bl.createInvokeModelInput(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Meta Llama model: %w", err)
	}

	return resp.Body, nil
}

// invokeMetaLlamaModelStream sends a streaming request to a Meta Llama model on Bedrock.
func (bl *BedrockLLM) invokeMetaLlamaModelStream(ctx context.Context, prompt string, options schema.CallOptions) (*brtypes.ResponseStream, error) {
	llamaPrompt := formatMessagesToLlamaPrompt(options.Messages)

	payload := MetaLlamaRequest{
		Prompt:      llamaPrompt,
		Temperature: options.Temperature,
		TopP:        options.TopP,
		MaxGenLen:   options.MaxTokens,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Meta Llama stream request: %w", err)
	}

	resp, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Meta Llama model with response stream: %w", err)
	}

	return resp.GetStream(), nil
}

// metaLlamaResponseToAIMessage converts a Meta Llama Bedrock response to a schema.AIMessage.
func (bl *BedrockLLM) metaLlamaResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var llamaResp MetaLlamaResponse
	if err := json.Unmarshal(body, &llamaResp);
 err != nil {
		return nil, fmt.Errorf("failed to unmarshal Meta Llama response: %w", err)
	}

	aiMsg := schema.NewAIMessage(llamaResp.Generation)
	aiMsg.AdditionalArgs = make(map[string]any)
	if llamaResp.StopReason != "" {
		aiMsg.AdditionalArgs["stop_reason"] = llamaResp.StopReason
	}

	usage := schema.TokenUsage{}
	var inputTokensCalculated bool
	if llamaResp.PromptTokenCount != nil {
		usage.InputTokens = *llamaResp.PromptTokenCount
		inputTokensCalculated = true
	}
	if llamaResp.GenerationTokenCount != nil {
		usage.OutputTokens = *llamaResp.GenerationTokenCount
	}

	// If prompt tokens not in response, try to calculate from input messages
	if !inputTokensCalculated {
		// This is a rough estimation. A proper tokenizer for Llama would be needed for accuracy.
		// For now, we can sum content length as a proxy or leave it to a more sophisticated token counter.
		// Alternatively, the calling layer could estimate it.
		// For simplicity here, we will not attempt a complex calculation.
		log.Println("PromptTokenCount not available in Llama response, input tokens not set in usage.")
	}

	usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	if usage.InputTokens > 0 || usage.OutputTokens > 0 {
		aiMsg.AdditionalArgs["usage"] = usage
	}

	// Tool Calls: Llama with InvokeModel typically requires parsing the `Generation` text for tool calls
	// if a tool-use prompting strategy was employed. This is complex and model-specific.
	// The Beluga framework might need a separate parsing layer or expect the Converse API for robust tool use.
	// For now, no automatic tool call parsing from text is implemented here.

	return aiMsg, nil
}

// metaLlamaStreamChunkToAIMessageChunk converts a Meta Llama Bedrock stream chunk to an llms.AIMessageChunk.
// The `chunkBytes` here is the actual JSON payload within the bedrockruntime.PayloadPart.Bytes.
func (bl *BedrockLLM) metaLlamaStreamChunkToAIMessageChunk(chunkBytesPayload []byte) (*llms.AIMessageChunk, error) {
	var streamResp MetaLlamaStreamResponseChunk
	if err := json.Unmarshal(chunkBytesPayload, &streamResp);
 err != nil {
		return nil, fmt.Errorf("failed to unmarshal Meta Llama stream chunk payload: %w. Payload: %s", err, string(chunkBytesPayload))
	}

	chunk := &llms.AIMessageChunk{
		Content: streamResp.Generation,
		AdditionalArgs: make(map[string]any),
	}

	if streamResp.StopReason != "" {
		chunk.AdditionalArgs["stop_reason"] = streamResp.StopReason
		chunk.IsLast = true // Assuming stop_reason indicates the final chunk
	}

	// Handle token usage if present in the chunk (often in the last chunk or specific metadata chunks)
	usage := schema.TokenUsage{}
	updatedUsage := false
	if streamResp.PromptTokenCount != nil {
		usage.InputTokens = *streamResp.PromptTokenCount
		updatedUsage = true
	}
	if streamResp.GenerationTokenCount != nil { // This is likely cumulative for Llama stream
		usage.OutputTokens = *streamResp.GenerationTokenCount
		updatedUsage = true
	}
	if streamResp.AmazonBedrockInvocationMetrics != nil {
		if streamResp.AmazonBedrockInvocationMetrics.InputTokenCount != nil {
			usage.InputTokens = *streamResp.AmazonBedrockInvocationMetrics.InputTokenCount
			updatedUsage = true
		}
		if streamResp.AmazonBedrockInvocationMetrics.OutputTokenCount != nil {
			usage.OutputTokens = *streamResp.AmazonBedrockInvocationMetrics.OutputTokenCount
			updatedUsage = true
		}
	}

	if updatedUsage {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
		chunk.AdditionalArgs["usage"] = usage
	}

	// As with non-streaming, tool calls from Llama via InvokeModel would require parsing `Generation` text.

	return chunk, nil
}

// formatMessagesToLlamaPrompt converts schema.Message to Llama 3/4 Instruct prompt format.
// Reference: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-meta.html
// Llama 3/4 format: <|begin_of_text|><|start_header_id|>system<|end_header_id|>
//
// You are a helpful AI assistant.<|eot_id|><|start_header_id|>user<|end_header_id|>
//
// What is the capital of France?<|eot_id|><|start_header_id|>assistant<|end_header_id|>
//
// The capital of France is Paris.<|eot_id|>
func formatMessagesToLlamaPrompt(messages []schema.Message) string {
	var promptBuilder strings.Builder
	promptBuilder.WriteString("<|begin_of_text|>")

	for _, msg := range messages {
		var role string
		switch msg.GetType() {
		case schema.MessageTypeSystem:
			role = "system"
		case schema.MessageTypeHuman:
			role = "user"
		case schema.MessageTypeAI:
			role = "assistant"
		case schema.MessageTypeTool: // Tool results are typically fed back as user messages containing observations.
			// Llama 3.1+ has more formal tool support, but via InvokeModel it might still be prompt based.
			// For now, we format it as a user message. This might need refinement for structured tool results.
			role = "user"
			promptBuilder.WriteString(fmt.Sprintf("<|start_header_id|>%s<|end_header_id|>\n\nTool Response: %s<|eot_id|>", role, msg.GetContent()))
			continue // Skip default formatting for tool messages
		default:
			log.Printf("Warning: Unknown message type in formatMessagesToLlamaPrompt: %s", msg.GetType())
			continue
		}

		promptBuilder.WriteString(fmt.Sprintf("<|start_header_id|>%s<|end_header_id|>\n\n%s<|eot_id|>", role, msg.GetContent()))
	}

	// If the last message was not an assistant, we need to prompt the assistant to speak.
	if len(messages) > 0 && messages[len(messages)-1].GetType() != schema.MessageTypeAI {
		promptBuilder.WriteString("<|start_header_id|>assistant<|end_header_id|>\n\n") // Ready for assistant to generate
	}

	finalPrompt := promptBuilder.String()
	log.Printf("Formatted Llama Prompt: %s", finalPrompt)
	return finalPrompt
}

// mapToolsToLlamaFormat is a placeholder. Tool use with Llama on Bedrock via InvokeModel
// is typically handled by formatting tool definitions and calls into the prompt itself.
// More structured tool use might be available via the Bedrock Converse API or specific Llama versions (e.g., Llama 3.1+).
func (bl *BedrockLLM) mapToolsToLlamaFormat(toolsToMap []tools.Tool, toolChoice string) (any, string) {
	log.Println("mapToolsToLlamaFormat is a placeholder. Llama tool use via InvokeModel usually requires prompt engineering.")
	// For RAG, the retrieved documents would be formatted into the prompt, likely as part of the user message or system prompt.
	return nil, ""
}

