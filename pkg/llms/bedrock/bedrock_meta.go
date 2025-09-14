// Package bedrock provides an implementation of the llms.ChatModel interface
// using AWS Bedrock Runtime, with support for multiple model providers.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	// brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types" // Not directly needed if bedrockruntime is used for stream type

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// MetaLlamaRequest represents the request payload for Meta Llama models on Bedrock.
// Based on: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-meta.html
type MetaLlamaRequest struct {
	Prompt      string    `json:"prompt"`
	Temperature float32   `json:"temperature,omitempty"`
	TopP        float32   `json:"top_p,omitempty"`
	MaxGenLen   int       `json:"max_gen_len,omitempty"` // Max tokens to generate
}

// MetaLlamaResponse represents the full response payload from Meta Llama models on Bedrock.
type MetaLlamaResponse struct {
	Generation          string `json:"generation"`
	PromptTokenCount    *int   `json:"prompt_token_count,omitempty"`
	GenerationTokenCount *int   `json:"generation_token_count,omitempty"`
	StopReason          string `json:"stop_reason,omitempty"`
}

// MetaLlamaStreamResponseChunk represents a chunk in a streaming response from Meta Llama.
type MetaLlamaStreamResponseChunk struct {
	Generation          string `json:"generation"`
	PromptTokenCount    *int   `json:"prompt_token_count,omitempty"`
	GenerationTokenCount *int   `json:"generation_token_count,omitempty"`
	StopReason          string `json:"stop_reason,omitempty"`
	AmazonBedrockInvocationMetrics *struct {
		InputTokenCount  *int `json:"inputTokenCount"`
		OutputTokenCount *int `json:"outputTokenCount"`
	} `json:"amazon-bedrock-invocationMetrics,omitempty"`
}

// invokeMetaLlamaModel sends a request to a Meta Llama model on Bedrock.
func (bl *BedrockLLM) invokeMetaLlamaModel(ctx context.Context, _ string, messages []schema.Message, options map[string]any) (json.RawMessage, error) {
	llamaPrompt := formatMessagesToLlamaPrompt(messages)

	payload := MetaLlamaRequest{
		Prompt: llamaPrompt,
	}

	if temp, ok := options["temperature"].(float32); ok {
		payload.Temperature = temp
	} else if temp, ok := options["temperature"].(float64); ok {
	    payload.Temperature = float32(temp)
	}

	if topP, ok := options["top_p"].(float32); ok {
		payload.TopP = topP
	} else if topP, ok := options["top_p"].(float64); ok {
	    payload.TopP = float32(topP)
	}

	if mt, ok := options["max_tokens"].(int); ok && mt > 0 {
		payload.MaxGenLen = mt
	} else if mt, ok := options["max_gen_len"].(int); ok && mt > 0 { // Llama specific
	    payload.MaxGenLen = mt
	}

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
func (bl *BedrockLLM) invokeMetaLlamaModelStream(ctx context.Context, _ string, messages []schema.Message, options map[string]any) (*bedrockruntime.InvokeModelWithResponseStreamEventStream, error) {
	llamaPrompt := formatMessagesToLlamaPrompt(messages)

	payload := MetaLlamaRequest{
		Prompt: llamaPrompt,
	}

	if temp, ok := options["temperature"].(float32); ok {
		payload.Temperature = temp
	} else if temp, ok := options["temperature"].(float64); ok {
	    payload.Temperature = float32(temp)
	}

	if topP, ok := options["top_p"].(float32); ok {
		payload.TopP = topP
	} else if topP, ok := options["top_p"].(float64); ok {
	    payload.TopP = float32(topP)
	}

	if mt, ok := options["max_tokens"].(int); ok && mt > 0 {
		payload.MaxGenLen = mt
	} else if mt, ok := options["max_gen_len"].(int); ok && mt > 0 {
	    payload.MaxGenLen = mt
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Meta Llama stream request: %w", err)
	}

	output, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Meta Llama model with response stream: %w", err)
	}

	return output.GetStream(), nil
}

// metaLlamaResponseToAIMessage converts a Meta Llama Bedrock response to a schema.AIMessage.
func (bl *BedrockLLM) metaLlamaResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var llamaResp MetaLlamaResponse
	if err := json.Unmarshal(body, &llamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Meta Llama response: %w", err)
	}

	aiMsg := schema.NewAIMessage(llamaResp.Generation)
	aiMsg.AdditionalArgs = make(map[string]any)
	if llamaResp.StopReason != "" {
		aiMsg.AdditionalArgs["stop_reason"] = llamaResp.StopReason
	}

	usage := map[string]int{
		"input_tokens":  0,
		"output_tokens": 0,
		"total_tokens":  0,
	}
	var inputTokensCalculated bool
	if llamaResp.PromptTokenCount != nil {
		usage["input_tokens"] = *llamaResp.PromptTokenCount
		inputTokensCalculated = true
	}
	if llamaResp.GenerationTokenCount != nil {
		usage["output_tokens"] = *llamaResp.GenerationTokenCount
	}

	if !inputTokensCalculated {
		log.Println("PromptTokenCount not available in Llama response, input tokens not set in usage.")
	}

	usage["total_tokens"] = usage["input_tokens"] + usage["output_tokens"]
	if usage["input_tokens"] > 0 || usage["output_tokens"] > 0 {
		aiMsg.AdditionalArgs["usage"] = usage
	}

	return aiMsg, nil
}

// metaLlamaStreamChunkToAIMessageChunk converts a Meta Llama Bedrock stream chunk to an llms.AIMessageChunk.
func (bl *BedrockLLM) metaLlamaStreamChunkToAIMessageChunk(chunkBytesPayload []byte) (*llms.AIMessageChunk, error) {
	var streamResp MetaLlamaStreamResponseChunk
	if err := json.Unmarshal(chunkBytesPayload, &streamResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Meta Llama stream chunk payload: %w. Payload: %s", err, string(chunkBytesPayload))
	}

	chunk := &llms.AIMessageChunk{
		Content: streamResp.Generation,
		AdditionalArgs: make(map[string]any),
	}

	if streamResp.StopReason != "" {
		chunk.AdditionalArgs["stop_reason"] = streamResp.StopReason
		// Note: IsLast is not a field of AIMessageChunk, but this would indicate it's the last chunk
	}

	usage := map[string]int{
		"input_tokens":  0,
		"output_tokens": 0,
		"total_tokens":  0,
	}
	updatedUsage := false
	if streamResp.PromptTokenCount != nil {
		usage["input_tokens"] = *streamResp.PromptTokenCount
		updatedUsage = true
	}
	if streamResp.GenerationTokenCount != nil { 
		usage["output_tokens"] = *streamResp.GenerationTokenCount
		updatedUsage = true
	}
	if streamResp.AmazonBedrockInvocationMetrics != nil {
		if streamResp.AmazonBedrockInvocationMetrics.InputTokenCount != nil {
			usage["input_tokens"] = *streamResp.AmazonBedrockInvocationMetrics.InputTokenCount
			updatedUsage = true
		}
		if streamResp.AmazonBedrockInvocationMetrics.OutputTokenCount != nil {
			usage["output_tokens"] = *streamResp.AmazonBedrockInvocationMetrics.OutputTokenCount
			updatedUsage = true
		}
	}

	if updatedUsage {
		usage["total_tokens"] = usage["input_tokens"] + usage["output_tokens"]
		chunk.AdditionalArgs["usage"] = usage
	}

	return chunk, nil
}

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
		case schema.MessageTypeTool:
			role = "user" // Llama 3 expects tool responses under the user role
			promptBuilder.WriteString(fmt.Sprintf("<|start_header_id|>%s<|end_header_id|>\n\nTool Response: %s<|eot_id|>", role, msg.GetContent()))
			continue
		default:
			log.Printf("Warning: Unknown message type in formatMessagesToLlamaPrompt: %s", msg.GetType())
			continue
		}

		promptBuilder.WriteString(fmt.Sprintf("<|start_header_id|>%s<|end_header_id|>\n\n%s<|eot_id|>", role, msg.GetContent()))
	}

	// If the last message was not from the assistant, add the assistant header to prompt it to speak.
	if len(messages) > 0 && messages[len(messages)-1].GetType() != schema.MessageTypeAI {
		promptBuilder.WriteString("<|start_header_id|>assistant<|end_header_id|>\n\n")
	}

	finalPrompt := promptBuilder.String()
	log.Printf("Formatted Llama Prompt: %s", finalPrompt)
	return finalPrompt
}

func (bl *BedrockLLM) mapToolsToLlamaFormat(_ []tools.Tool, _ string) (any, string) {
	// Llama 3 tool usage is typically handled via specific prompt formatting within the main prompt,
	// rather than a separate tools parameter in the API request for Bedrock InvokeModel.
	// This function is a placeholder; actual tool integration would involve modifying
	// formatMessagesToLlamaPrompt to include tool definitions and calls in the Llama 3 format.
	log.Println("mapToolsToLlamaFormat is a placeholder. Llama tool use via InvokeModel usually requires prompt engineering.")
	return nil, ""
}


