package openaicompat

import (
	"encoding/base64"
	"fmt"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/openai/openai-go"
)

// ConvertMessages converts a slice of Beluga messages to OpenAI API format.
// It supports SystemMessage, HumanMessage (with text and image parts),
// AIMessage (with text and tool calls), and ToolMessage.
func ConvertMessages(msgs []schema.Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, msg := range msgs {
		converted, err := convertMessage(msg)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func convertMessage(msg schema.Message) (openai.ChatCompletionMessageParamUnion, error) {
	switch m := msg.(type) {
	case *schema.SystemMessage:
		return openai.SystemMessage(m.Text()), nil
	case *schema.HumanMessage:
		return convertHumanMessage(m)
	case *schema.AIMessage:
		return convertAIMessage(m), nil
	case *schema.ToolMessage:
		return openai.ToolMessage(m.Text(), m.ToolCallID), nil
	default:
		return openai.ChatCompletionMessageParamUnion{}, fmt.Errorf("openaicompat: unsupported message type %T", msg)
	}
}

func convertHumanMessage(m *schema.HumanMessage) (openai.ChatCompletionMessageParamUnion, error) {
	if !hasMultimodalParts(m.Parts) {
		return openai.UserMessage(m.Text()), nil
	}

	parts := make([]openai.ChatCompletionContentPartUnionParam, 0, len(m.Parts))
	for _, p := range m.Parts {
		if part, ok := convertContentPart(p); ok {
			parts = append(parts, part)
		}
	}
	return openai.UserMessage(parts), nil
}

// hasMultimodalParts returns true if any content part is not text.
func hasMultimodalParts(parts []schema.ContentPart) bool {
	for _, p := range parts {
		if p.PartType() != schema.ContentText {
			return true
		}
	}
	return false
}

// convertContentPart converts a single content part to the OpenAI format.
// Returns the converted part and true, or zero value and false if unsupported.
func convertContentPart(p schema.ContentPart) (openai.ChatCompletionContentPartUnionParam, bool) {
	switch cp := p.(type) {
	case schema.TextPart:
		return openai.TextContentPart(cp.Text), true
	case schema.ImagePart:
		url := resolveImageURL(cp)
		if url == "" {
			return openai.ChatCompletionContentPartUnionParam{}, false
		}
		return openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
			URL: url,
		}), true
	default:
		// Skip unsupported content types (AudioPart, VideoPart, FilePart).
		return openai.ChatCompletionContentPartUnionParam{}, false
	}
}

// resolveImageURL returns the URL for an image part, encoding inline data
// as a data URI if no URL is provided.
func resolveImageURL(cp schema.ImagePart) string {
	if cp.URL != "" {
		return cp.URL
	}
	if len(cp.Data) == 0 {
		return ""
	}
	mime := cp.MimeType
	if mime == "" {
		mime = "image/png"
	}
	return fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(cp.Data))
}

func convertAIMessage(m *schema.AIMessage) openai.ChatCompletionMessageParamUnion {
	msg := openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{},
	}
	text := m.Text()
	if text != "" {
		msg.OfAssistant.Content.OfString = openai.String(text)
	}
	if len(m.ToolCalls) > 0 {
		calls := make([]openai.ChatCompletionMessageToolCallParam, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			calls[i] = openai.ChatCompletionMessageToolCallParam{
				ID: tc.ID,
				Function: openai.ChatCompletionMessageToolCallFunctionParam{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			}
		}
		msg.OfAssistant.ToolCalls = calls
	}
	return msg
}

// ConvertResponse converts an OpenAI ChatCompletion response to a Beluga AIMessage.
func ConvertResponse(resp *openai.ChatCompletion) *schema.AIMessage {
	if resp == nil {
		return &schema.AIMessage{}
	}
	if len(resp.Choices) == 0 {
		return &schema.AIMessage{ModelID: resp.Model}
	}
	choice := resp.Choices[0]
	ai := &schema.AIMessage{
		ModelID: resp.Model,
		Usage: schema.Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:  int(resp.Usage.TotalTokens),
			CachedTokens: int(resp.Usage.PromptTokensDetails.CachedTokens),
		},
	}
	if choice.Message.Content != "" {
		ai.Parts = []schema.ContentPart{schema.TextPart{Text: choice.Message.Content}}
	}
	if len(choice.Message.ToolCalls) > 0 {
		ai.ToolCalls = make([]schema.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			ai.ToolCalls[i] = schema.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}
	}
	return ai
}
