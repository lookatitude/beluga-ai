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

	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
)

// MistralRequest represents the request payload for Mistral models on Bedrock.
// Based on: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-mistral-chat-completion.html
type MistralRequest struct {
	Prompt      string    `json:"prompt"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
	TopP        float32   `json:"top_p,omitempty"`
	TopK        int       `json:"top_k,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// MistralResponse represents the response payload from Mistral models on Bedrock.
type MistralResponse struct {
	Outputs []MistralOutput `json:"outputs"`
}

// MistralOutput represents a single output from the Mistral model.
type MistralOutput struct {
	Text         string               `json:"text"`
	StopReason   string               `json:"stop_reason"`
}

// MistralStreamResponseChunk represents a chunk in a streaming response from Mistral.
type MistralStreamResponseChunk struct {
	Chunk struct {
		Bytes string `json:"bytes"`
	} `json:"chunk"`
	Delta struct {
		Text string `json:"text"`
	} `json:"delta"`
	OutputTokenCount *int `json:"amazon-bedrock-outputTokenCount,omitempty"`
}

func (bl *BedrockLLM) invokeMistralModel(ctx context.Context, _ string, messages []schema.Message, options map[string]any) (json.RawMessage, error) {
	mistralPrompt := formatMessagesToMistralPrompt(messages)

	payload := MistralRequest{
		Prompt: mistralPrompt,
	}

	if mt, ok := options["max_tokens"].(int); ok && mt > 0 {
		payload.MaxTokens = mt
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

	if topK, ok := options["top_k"].(int); ok && topK > 0 {
		payload.TopK = topK
	}

	if stop, ok := options["stop_words"].([]string); ok && len(stop) > 0 {
		payload.Stop = stop
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Mistral request: %w", err)
	}

	resp, err := bl.client.InvokeModel(ctx, bl.createInvokeModelInput(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Mistral model: %w", err)
	}

	return resp.Body, nil
}

func (bl *BedrockLLM) invokeMistralModelStream(ctx context.Context, _ string, messages []schema.Message, options map[string]any) (*bedrockruntime.InvokeModelWithResponseStreamEventStream, error) {
	mistralPrompt := formatMessagesToMistralPrompt(messages)

	payload := MistralRequest{
		Prompt: mistralPrompt,
	}

	if mt, ok := options["max_tokens"].(int); ok && mt > 0 {
		payload.MaxTokens = mt
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

	if topK, ok := options["top_k"].(int); ok && topK > 0 {
		payload.TopK = topK
	}

	if stop, ok := options["stop_words"].([]string); ok && len(stop) > 0 {
		payload.Stop = stop
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Mistral stream request: %w", err)
	}

	output, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Mistral model with response stream: %w", err)
	}

	return output.GetStream(), nil
}

func (bl *BedrockLLM) mistralResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var mistralResp MistralResponse
	if err := json.Unmarshal(body, &mistralResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Mistral response: %w", err)
	}

	if len(mistralResp.Outputs) == 0 {
		return nil, fmt.Errorf("no output found in Mistral response")
	}

	output := mistralResp.Outputs[0]
	content := output.Text

	aiMsg := schema.NewAIMessage(content)
	aiMsg.AdditionalArgs = make(map[string]any)
	aiMsg.AdditionalArgs["stop_reason"] = output.StopReason

	return aiMsg, nil
}

func (bl *BedrockLLM) mistralStreamChunkToAIMessageChunk(chunkBytes []byte) (*llms.AIMessageChunk, error) {
	var streamResp MistralStreamResponseChunk
	if err := json.Unmarshal(chunkBytes, &streamResp); err != nil {
		log.Printf("Mistral stream chunk is not JSON, attempting to treat as raw text: %s. Error: %v", string(chunkBytes), err)
		return &llms.AIMessageChunk{Content: string(chunkBytes)}, nil
	}

	if streamResp.Delta.Text != "" {
		chunk := &llms.AIMessageChunk{
			Content: streamResp.Delta.Text,
			AdditionalArgs: make(map[string]any),
		}
		if streamResp.OutputTokenCount != nil {
			chunk.AdditionalArgs["output_token_count_so_far"] = *streamResp.OutputTokenCount
		}
		return chunk, nil
	}

	log.Printf("Mistral stream chunk did not contain expected content fields: %s", string(chunkBytes))
	return nil, nil
}

func formatMessagesToMistralPrompt(messages []schema.Message) string {
	var promptBuilder strings.Builder
	promptBuilder.WriteString("<s>")

	for _, msg := range messages {
		switch msg.GetType() {
		case schema.MessageTypeSystem:
			promptBuilder.WriteString(fmt.Sprintf("[INST] %s ", msg.GetContent()))
		case schema.MessageTypeHuman:
			if promptBuilder.Len() > 3 && !strings.HasSuffix(promptBuilder.String(), "[INST] ") && !strings.HasSuffix(promptBuilder.String(), "[/INST]") {
				promptBuilder.WriteString("[INST] ")
			} else if !strings.HasSuffix(promptBuilder.String(), "[INST] ") {
			    promptBuilder.WriteString("[INST] ")
            }
			promptBuilder.WriteString(msg.GetContent())
			promptBuilder.WriteString(" [/INST]")
		case schema.MessageTypeAI:
			promptBuilder.WriteString(msg.GetContent())
			promptBuilder.WriteString("</s>")
		case schema.MessageTypeTool:
			promptBuilder.WriteString("[INST] Tool Result: ")
			promptBuilder.WriteString(msg.GetContent())
			promptBuilder.WriteString(" [/INST]")
		default:
			log.Printf("Warning: Unknown message type in formatMessagesToMistralPrompt: %s", msg.GetType())
		}
	}

	finalPrompt := promptBuilder.String()
	log.Printf("Formatted Mistral Prompt: %s", finalPrompt)
	return finalPrompt
}

func (bl *BedrockLLM) mapToolsToMistralFormat(_ []tools.Tool, _ string) (any, string) {
	log.Println("mapToolsToMistralFormat is a placeholder and needs implementation based on Bedrock Mistral's tool support.")
	return nil, ""
}


