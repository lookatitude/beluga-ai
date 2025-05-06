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

// MistralRequest represents the request payload for Mistral models on Bedrock.
// Based on: https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-mistral-chat-completion.html
type MistralRequest struct {
	Prompt      string    `json:"prompt"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
	TopP        float32   `json:"top_p,omitempty"`
	TopK        int       `json:"top_k,omitempty"`
	Stop        []string  `json:"stop,omitempty"` // Mistral uses "stop"
	// TODO: Add tool_choice and tools if Mistral on Bedrock supports it directly in this format.
	// For now, we assume a simpler text-completion like interface or a specific tool format if documented.
	// The main bedrock.go file's BindTools will store tools, and this provider needs to map them.
}

// MistralResponse represents the response payload from Mistral models on Bedrock.
type MistralResponse struct {
	Outputs []MistralOutput `json:"outputs"`
	// TODO: Add token usage if available directly in the response, otherwise it might be in headers or a separate call.
	// For now, we'll estimate or rely on stream completion for output tokens.
}

// MistralOutput represents a single output from the Mistral model.
type MistralOutput struct {
	Text         string               `json:"text"`
	StopReason   string               `json:"stop_reason"` // e.g., "stop", "length", "tool_calls"
	// TODO: Add tool_calls if supported and structured this way.
}

// MistralStreamResponseChunk represents a chunk in a streaming response from Mistral.
type MistralStreamResponseChunk struct {
	Chunk struct {
		Bytes string `json:"bytes"` // The actual text chunk, often base64 encoded if not text/event-stream
	} `json:"chunk"` // This structure might vary based on actual Bedrock stream format for Mistral
	// Alternative simpler structure if Bedrock normalizes it:
	Delta struct {
		Text string `json:"text"`
	} `json:"delta"`
	OutputTokenCount *int `json:"amazon-bedrock-outputTokenCount,omitempty"` // Check if this header is present in stream events
	// TODO: Add tool_call_chunks if supported.
}

// invokeMistralModel sends a request to a Mistral model on Bedrock.
// Note: Mistral models on Bedrock typically use a chat-like prompt format, but the API might be text-completion style.
// We will adapt the schema.Message array to a single string prompt as per Mistral's typical API structure.
func (bl *BedrockLLM) invokeMistralModel(ctx context.Context, prompt string, options schema.CallOptions) (json.RawMessage, error) {
	// Construct the prompt string from messages, ensuring Mistral's expected format (e.g., <s>[INST] ... [/INST] ...</s>)
	// For simplicity, we'll use the passed `prompt` string which should be pre-formatted by the caller or a helper.
	// A more robust implementation would convert schema.Message to Mistral's chat template.

	mistralPrompt := formatMessagesToMistralPrompt(options.Messages) // Use the messages from options

	payload := MistralRequest{
		Prompt:      mistralPrompt,
		MaxTokens:   options.MaxTokens,
		Temperature: options.Temperature,
		TopP:        options.TopP,
		TopK:        options.TopK,
		Stop:        options.StopWords,
	}

	// TODO: Implement tool mapping if Mistral on Bedrock supports it.
	// if len(options.Tools) > 0 {
	// 	 mappedTools, toolChoice := bl.mapToolsToMistralFormat(options.Tools, options.ToolChoice)
	// 	 payload.Tools = mappedTools
	// 	 payload.ToolChoice = toolChoice
	// }

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

// invokeMistralModelStream sends a streaming request to a Mistral model on Bedrock.
func (bl *BedrockLLM) invokeMistralModelStream(ctx context.Context, prompt string, options schema.CallOptions) (*brtypes.ResponseStream, error) {
	mistralPrompt := formatMessagesToMistralPrompt(options.Messages)

	payload := MistralRequest{
		Prompt:      mistralPrompt,
		MaxTokens:   options.MaxTokens,
		Temperature: options.Temperature,
		TopP:        options.TopP,
		TopK:        options.TopK,
		Stop:        options.StopWords,
	}

	// TODO: Implement tool mapping for streaming if supported.

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Mistral stream request: %w", err)
	}

	resp, err := bl.client.InvokeModelWithResponseStream(ctx, bl.createInvokeModelWithResponseStreamInput(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Mistral model with response stream: %w", err)
	}

	return resp.GetStream(), nil
}

// mistralResponseToAIMessage converts a Mistral Bedrock response to a schema.AIMessage.
func (bl *BedrockLLM) mistralResponseToAIMessage(body json.RawMessage) (schema.Message, error) {
	var mistralResp MistralResponse
	if err := json.Unmarshal(body, &mistralResp);
 err != nil {
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

	// TODO: Extract and add token usage if available.
	// For now, input tokens would need to be calculated from the prompt.
	// Output tokens can be approximated by len(content) or from a specific field if present.
	// Example: aiMsg.AdditionalArgs["usage"] = schema.TokenUsage{...}

	// TODO: Parse tool calls if present in output.StopReason == "tool_calls" and output.ToolCalls is populated.
	// if output.StopReason == "tool_calls" && output.ToolCalls != nil {
	// 	 for _, tc := range output.ToolCalls {
	// 		 aiMsg.ToolCalls = append(aiMsg.ToolCalls, schema.ToolCall{...})
	// 	 }
	// }

	return aiMsg, nil
}

// mistralStreamChunkToAIMessageChunk converts a Mistral Bedrock stream chunk to an llms.AIMessageChunk.
func (bl *BedrockLLM) mistralStreamChunkToAIMessageChunk(chunkBytes []byte) (*llms.AIMessageChunk, error) {
	// The structure of chunkBytes for Mistral on Bedrock needs to be confirmed.
	// It might be a JSON object per event, or just raw text for text/event-stream.
	// Assuming it's a JSON object similar to other Bedrock stream events for now.
	var streamResp MistralStreamResponseChunk
	if err := json.Unmarshal(chunkBytes, &streamResp);
 err != nil {
		// If it's not JSON, it might be raw text. This part needs to be robust.
		log.Printf("Mistral stream chunk is not JSON, attempting to treat as raw text: %s. Error: %v", string(chunkBytes), err)
		// Fallback: if it's not a structured chunk, it might be just the text delta.
		// This is a common pattern for simpler streaming APIs.
		// However, Bedrock usually wraps chunks in some JSON structure.
		// For now, we'll assume the documented structure or a common Bedrock stream structure.
		// If the `MistralStreamResponseChunk` has a simple `Delta.Text` field, that's what we use.
		// If it has `Chunk.Bytes` (often base64), it needs decoding.
		return &llms.AIMessageChunk{Content: string(chunkBytes)}, nil // Simplistic fallback, likely incorrect for Bedrock
	}

	// Prefer Delta.Text if available, as it's usually the direct content.
	if streamResp.Delta.Text != "" {
		chunk := &llms.AIMessageChunk{
			Content: streamResp.Delta.Text,
			AdditionalArgs: make(map[string]any),
		}
		if streamResp.OutputTokenCount != nil {
			// This is usually an aggregate count, not per chunk. Handle accordingly.
			// For now, just log it or store it if it's the final chunk.
			chunk.AdditionalArgs["output_token_count_so_far"] = *streamResp.OutputTokenCount
		}
		return chunk, nil
	}

	// TODO: Handle `Chunk.Bytes` if it's the actual format (e.g., base64 encoded text).
	// TODO: Handle tool call chunks if Mistral on Bedrock supports streaming tool calls.

	// If no content found in expected fields, return nil or an empty chunk to signify no new text.
	// This can happen for control messages or metadata in the stream.
	log.Printf("Mistral stream chunk did not contain expected content fields: %s", string(chunkBytes))
	return nil, nil // Or return an error if this state is unexpected.
}

// formatMessagesToMistralPrompt converts a slice of schema.Message to a Mistral-compatible prompt string.
// Mistral models often use a format like: "<s>[INST] User Message [/INST] Assistant Response</s>[INST] Next User Message [/INST]"
func formatMessagesToMistralPrompt(messages []schema.Message) string {
	var promptBuilder strings.Builder
	promptBuilder.WriteString("<s>") // Start of sequence token, optional for some fine-tunes

	for _, msg := range messages {
		switch msg.GetType() {
		case schema.MessageTypeSystem:
			// Mistral doesn't have a distinct system prompt role in the same way as OpenAI.
			// It's often prepended to the first user message or handled as part of the initial instruction.
			// For now, we'll prepend it to the first user message if it exists.
			// This might need adjustment based on specific Mistral model fine-tuning on Bedrock.
			promptBuilder.WriteString(fmt.Sprintf("[INST] %s ", msg.GetContent())) // Treat as part of the first instruction
		case schema.MessageTypeHuman:
			if promptBuilder.Len() > 3 && !strings.HasSuffix(promptBuilder.String(), "[INST] ") && !strings.HasSuffix(promptBuilder.String(), "[/INST]") {
				promptBuilder.WriteString("[INST] ") // Ensure it's an instruction if not following an assistant response
			} else if !strings.HasSuffix(promptBuilder.String(), "[INST] ") {
			    promptBuilder.WriteString("[INST] ")
            }
			promptBuilder.WriteString(msg.GetContent())
			promptBuilder.WriteString(" [/INST]")
		case schema.MessageTypeAI:
			promptBuilder.WriteString(msg.GetContent())
			promptBuilder.WriteString("</s>") // End of assistant turn, then expect new [INST]
			// If next message is Human, it will add [INST]. If it's AI again (unlikely), this might need adjustment.
		case schema.MessageTypeTool: // Tool results are typically provided back to the model as a user message.
			// This requires specific formatting for Mistral's tool use, which is not fully detailed here yet.
			// For now, append as if it's a user observation.
			promptBuilder.WriteString("[INST] Tool Result: ")
			promptBuilder.WriteString(msg.GetContent()) // Content should be the JSON result of the tool call
			promptBuilder.WriteString(" [/INST]")
		default:
			log.Printf("Warning: Unknown message type in formatMessagesToMistralPrompt: %s", msg.GetType())
		}
	}

	// Ensure the prompt doesn't end with </s> if we expect the model to generate next.
	// If the last message was AI, it would have </s>. If Human, it ends with [/INST].
	// If the last message was Human (ending in [/INST]), the model will continue from there.
	// If the last message was AI (ending in </s>), and we want the model to speak again, this is unusual.
	// Typically, a human prompt follows an AI response.

	finalPrompt := promptBuilder.String()
    // If it ends with </s>[INST] ... [/INST], it's ready for model generation.
    // If it ends with </s>, it implies the conversation ended with AI. To prompt further, a new [INST] is needed.
    // This logic assumes the typical alternating conversation flow.

	log.Printf("Formatted Mistral Prompt: %s", finalPrompt)
	return finalPrompt
}

// mapToolsToMistralFormat converts generic tools.Tool to Mistral's specific tool format.
// This is a placeholder and needs to be implemented based on Mistral's actual tool specification on Bedrock.
func (bl *BedrockLLM) mapToolsToMistralFormat(toolsToMap []tools.Tool, toolChoice string) (any, string) {
	// Placeholder: Mistral's tool format on Bedrock needs to be determined from documentation.
	// It might be similar to OpenAI's function calling or Anthropic's tool use.
	// Example structure (hypothetical):
	// type MistralTool struct {
	// 	 Name        string `json:"name"`
	// 	 Description string `json:"description"`
	// 	 InputSchema any    `json:"input_schema"`
	// }
	// var mistralTools []MistralTool
	// for _, t := range toolsToMap {
	// 	 mistralTools = append(mistralTools, MistralTool{Name: t.Name(), Description: t.Description(), InputSchema: t.InputSchema()})
	// }
	// return mistralTools, toolChoice // toolChoice might be "auto", "none", or {"type": "function", "function": {"name": "my_function"}}
	log.Println("mapToolsToMistralFormat is a placeholder and needs implementation based on Bedrock Mistral's tool support.")
	return nil, ""
}

