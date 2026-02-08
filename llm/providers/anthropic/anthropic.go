// Package anthropic provides the Anthropic (Claude) LLM provider for the Beluga AI framework.
// It implements the llm.ChatModel interface using the anthropic-sdk-go SDK.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
//
//	model, err := llm.New("anthropic", config.ProviderConfig{
//	    Model:  "claude-sonnet-4-5-20250929",
//	    APIKey: "sk-ant-...",
//	})
package anthropic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"iter"

	anthropicSDK "github.com/anthropics/anthropic-sdk-go"
	anthropicOption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

const defaultMaxTokens = 4096

func init() {
	llm.Register("anthropic", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// Model implements llm.ChatModel using the Anthropic Messages API.
type Model struct {
	client anthropicSDK.Client
	model  string
	tools  []schema.ToolDefinition
}

// New creates a new Anthropic ChatModel.
func New(cfg config.ProviderConfig) (*Model, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("anthropic: model is required")
	}
	opts := []anthropicOption.RequestOption{}
	if cfg.APIKey != "" {
		opts = append(opts, anthropicOption.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, anthropicOption.WithBaseURL(cfg.BaseURL))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, anthropicOption.WithRequestTimeout(cfg.Timeout))
	}
	opts = append(opts, anthropicOption.WithMaxRetries(0))
	client := anthropicSDK.NewClient(opts...)
	return &Model{
		client: client,
		model:  cfg.Model,
	}, nil
}

// Generate sends messages and returns a complete AI response.
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	params, err := m.buildParams(msgs, opts)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic: generate failed: %w", err)
	}
	return convertResponse(resp), nil
}

// Stream sends messages and returns an iterator of response chunks.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	params, err := m.buildParams(msgs, opts)
	if err != nil {
		return func(yield func(schema.StreamChunk, error) bool) {
			yield(schema.StreamChunk{}, err)
		}
	}
	stream := m.client.Messages.NewStreaming(ctx, params)
	return func(yield func(schema.StreamChunk, error) bool) {
		defer stream.Close()
		for stream.Next() {
			event := stream.Current()
			chunk := convertStreamEvent(event, m.model)
			if chunk == nil {
				continue
			}
			if !yield(*chunk, nil) {
				return
			}
		}
		if err := stream.Err(); err != nil {
			yield(schema.StreamChunk{}, err)
		}
	}
}

// BindTools returns a new Model that includes the given tools in every request.
func (m *Model) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	cp := *m
	cp.tools = make([]schema.ToolDefinition, len(tools))
	copy(cp.tools, tools)
	return &cp
}

// ModelID returns the model identifier.
func (m *Model) ModelID() string {
	return m.model
}

func (m *Model) buildParams(msgs []schema.Message, opts []llm.GenerateOption) (anthropicSDK.MessageNewParams, error) {
	genOpts := llm.ApplyOptions(opts...)
	maxTokens := int64(defaultMaxTokens)
	if genOpts.MaxTokens > 0 {
		maxTokens = int64(genOpts.MaxTokens)
	}

	converted, system, err := convertMessages(msgs)
	if err != nil {
		return anthropicSDK.MessageNewParams{}, err
	}

	params := anthropicSDK.MessageNewParams{
		Model:     anthropicSDK.Model(m.model),
		MaxTokens: maxTokens,
		Messages:  converted,
	}

	if len(system) > 0 {
		params.System = system
	}

	if len(m.tools) > 0 {
		params.Tools = convertTools(m.tools)
	}

	if genOpts.Temperature != nil {
		params.Temperature = anthropicSDK.Float(*genOpts.Temperature)
	}
	if genOpts.TopP != nil {
		params.TopP = anthropicSDK.Float(*genOpts.TopP)
	}
	if len(genOpts.StopSequences) > 0 {
		params.StopSequences = genOpts.StopSequences
	}

	switch genOpts.ToolChoice {
	case llm.ToolChoiceAuto:
		params.ToolChoice = anthropicSDK.ToolChoiceUnionParam{
			OfAuto: &anthropicSDK.ToolChoiceAutoParam{},
		}
	case llm.ToolChoiceNone:
		params.ToolChoice = anthropicSDK.ToolChoiceUnionParam{
			OfNone: &anthropicSDK.ToolChoiceNoneParam{},
		}
	case llm.ToolChoiceRequired:
		params.ToolChoice = anthropicSDK.ToolChoiceUnionParam{
			OfAny: &anthropicSDK.ToolChoiceAnyParam{},
		}
	}
	if genOpts.SpecificTool != "" {
		params.ToolChoice = anthropicSDK.ToolChoiceUnionParam{
			OfTool: &anthropicSDK.ToolChoiceToolParam{
				Name: genOpts.SpecificTool,
			},
		}
	}

	return params, nil
}

func convertMessages(msgs []schema.Message) ([]anthropicSDK.MessageParam, []anthropicSDK.TextBlockParam, error) {
	var system []anthropicSDK.TextBlockParam
	out := make([]anthropicSDK.MessageParam, 0, len(msgs))
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *schema.SystemMessage:
			system = append(system, anthropicSDK.TextBlockParam{Text: m.Text()})
		case *schema.HumanMessage:
			blocks := convertContentParts(m.Parts)
			out = append(out, anthropicSDK.NewUserMessage(blocks...))
		case *schema.AIMessage:
			blocks := convertAIContentParts(m)
			out = append(out, anthropicSDK.NewAssistantMessage(blocks...))
		case *schema.ToolMessage:
			out = append(out, anthropicSDK.NewUserMessage(
				anthropicSDK.NewToolResultBlock(m.ToolCallID, m.Text(), false),
			))
		default:
			return nil, nil, fmt.Errorf("anthropic: unsupported message type %T", msg)
		}
	}
	return out, system, nil
}

func convertContentParts(parts []schema.ContentPart) []anthropicSDK.ContentBlockParamUnion {
	blocks := make([]anthropicSDK.ContentBlockParamUnion, 0, len(parts))
	for _, p := range parts {
		switch cp := p.(type) {
		case schema.TextPart:
			blocks = append(blocks, anthropicSDK.NewTextBlock(cp.Text))
		case schema.ImagePart:
			if cp.URL != "" {
				blocks = append(blocks, anthropicSDK.NewImageBlock(anthropicSDK.URLImageSourceParam{
					URL: cp.URL,
				}))
			} else if len(cp.Data) > 0 {
				mime := cp.MimeType
				if mime == "" {
					mime = "image/png"
				}
				encoded := base64.StdEncoding.EncodeToString(cp.Data)
				blocks = append(blocks, anthropicSDK.NewImageBlockBase64(mime, encoded))
			}
		}
	}
	return blocks
}

func convertAIContentParts(m *schema.AIMessage) []anthropicSDK.ContentBlockParamUnion {
	var blocks []anthropicSDK.ContentBlockParamUnion
	text := m.Text()
	if text != "" {
		blocks = append(blocks, anthropicSDK.NewTextBlock(text))
	}
	for _, tc := range m.ToolCalls {
		var input any
		json.Unmarshal([]byte(tc.Arguments), &input)
		blocks = append(blocks, anthropicSDK.NewToolUseBlock(tc.ID, input, tc.Name))
	}
	return blocks
}

func convertTools(tools []schema.ToolDefinition) []anthropicSDK.ToolUnionParam {
	out := make([]anthropicSDK.ToolUnionParam, len(tools))
	for i, t := range tools {
		tp := anthropicSDK.ToolParam{
			Name: t.Name,
			InputSchema: anthropicSDK.ToolInputSchemaParam{
				Properties: t.InputSchema["properties"],
			},
		}
		if t.Description != "" {
			tp.Description = anthropicSDK.String(t.Description)
		}
		if req, ok := t.InputSchema["required"].([]any); ok {
			for _, r := range req {
				if s, ok := r.(string); ok {
					tp.InputSchema.Required = append(tp.InputSchema.Required, s)
				}
			}
		}
		out[i] = anthropicSDK.ToolUnionParam{OfTool: &tp}
	}
	return out
}

func convertResponse(resp *anthropicSDK.Message) *schema.AIMessage {
	if resp == nil {
		return &schema.AIMessage{}
	}
	ai := &schema.AIMessage{
		ModelID: string(resp.Model),
		Usage: schema.Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
			TotalTokens:  int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
			CachedTokens: int(resp.Usage.CacheReadInputTokens),
		},
	}
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			ai.Parts = append(ai.Parts, schema.TextPart{Text: block.Text})
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			ai.ToolCalls = append(ai.ToolCalls, schema.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(args),
			})
		}
	}
	return ai
}

func convertStreamEvent(event anthropicSDK.MessageStreamEventUnion, modelID string) *schema.StreamChunk {
	switch event.Type {
	case "content_block_delta":
		chunk := &schema.StreamChunk{ModelID: modelID}
		if event.Delta.Type == "text_delta" {
			chunk.Delta = event.Delta.Text
		} else if event.Delta.Type == "input_json_delta" {
			chunk.ToolCalls = []schema.ToolCall{{
				Arguments: event.Delta.PartialJSON,
			}}
		}
		return chunk
	case "content_block_start":
		if event.ContentBlock.Type == "tool_use" {
			return &schema.StreamChunk{
				ModelID: modelID,
				ToolCalls: []schema.ToolCall{{
					ID:   event.ContentBlock.ID,
					Name: event.ContentBlock.Name,
				}},
			}
		}
		return nil
	case "message_delta":
		return &schema.StreamChunk{
			ModelID:      modelID,
			FinishReason: mapStopReason(string(event.Delta.StopReason)),
			Usage: &schema.Usage{
				OutputTokens: int(event.Usage.OutputTokens),
			},
		}
	default:
		return nil
	}
}

func mapStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	default:
		return reason
	}
}
