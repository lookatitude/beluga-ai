package mistral

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"time"

	"github.com/gage-technologies/mistral-go"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

const (
	defaultEndpoint = "https://api.mistral.ai"
	defaultModel    = "mistral-large-latest"
)

func init() {
	llm.Register("mistral", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// Model implements llm.ChatModel using the Mistral AI SDK.
type Model struct {
	client *mistral.MistralClient
	model  string
	tools  []schema.ToolDefinition
}

// Compile-time interface check.
var _ llm.ChatModel = (*Model)(nil)

// New creates a new Mistral ChatModel.
func New(cfg config.ProviderConfig) (*Model, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("mistral: api_key is required")
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	endpoint := cfg.BaseURL
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	client := mistral.NewMistralClient(cfg.APIKey, endpoint, 3, timeout)
	return &Model{
		client: client,
		model:  model,
	}, nil
}

// Generate sends messages and returns a complete AI response.
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	chatMsgs := convertMessages(msgs)
	params := m.buildParams(opts)

	resp, err := m.client.Chat(m.model, chatMsgs, params)
	if err != nil {
		return nil, fmt.Errorf("mistral: generate failed: %w", err)
	}
	return convertResponse(resp), nil
}

// Stream sends messages and returns an iterator of response chunks.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	chatMsgs := convertMessages(msgs)
	params := m.buildParams(opts)

	ch, err := m.client.ChatStream(m.model, chatMsgs, params)
	if err != nil {
		return func(yield func(schema.StreamChunk, error) bool) {
			yield(schema.StreamChunk{}, fmt.Errorf("mistral: stream failed: %w", err))
		}
	}

	return func(yield func(schema.StreamChunk, error) bool) {
		for {
			select {
			case <-ctx.Done():
				yield(schema.StreamChunk{}, ctx.Err())
				return
			case chunk, ok := <-ch:
				if !ok {
					return
				}
				if chunk.Error != nil {
					yield(schema.StreamChunk{}, chunk.Error)
					return
				}
				sc := convertStreamChunk(chunk, m.model)
				if !yield(sc, nil) {
					return
				}
			}
		}
	}
}

// BindTools returns a new Model with the given tool definitions.
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

func (m *Model) buildParams(opts []llm.GenerateOption) *mistral.ChatRequestParams {
	genOpts := llm.ApplyOptions(opts...)
	params := &mistral.ChatRequestParams{
		Temperature: 0.7,
		TopP:        1.0,
	}
	if genOpts.Temperature != nil {
		params.Temperature = *genOpts.Temperature
	}
	if genOpts.TopP != nil {
		params.TopP = *genOpts.TopP
	}
	if genOpts.MaxTokens > 0 {
		params.MaxTokens = genOpts.MaxTokens
	}
	if len(m.tools) > 0 {
		params.Tools = convertTools(m.tools)
	}
	switch genOpts.ToolChoice {
	case llm.ToolChoiceAuto:
		params.ToolChoice = mistral.ToolChoiceAuto
	case llm.ToolChoiceNone:
		params.ToolChoice = mistral.ToolChoiceNone
	case llm.ToolChoiceRequired:
		params.ToolChoice = mistral.ToolChoiceAny
	}
	if genOpts.Format != nil && genOpts.Format.Type == "json_object" {
		params.ResponseFormat = mistral.ResponseFormatJsonObject
	}
	return params
}

// convertStreamChunk converts a Mistral stream chunk to a Beluga StreamChunk.
func convertStreamChunk(chunk mistral.ChatCompletionStreamResponse, modelID string) schema.StreamChunk {
	sc := schema.StreamChunk{ModelID: modelID}
	if len(chunk.Choices) > 0 {
		delta := chunk.Choices[0].Delta
		sc.Delta = delta.Content
		sc.FinishReason = string(chunk.Choices[0].FinishReason)
		if len(delta.ToolCalls) > 0 {
			sc.ToolCalls = convertMistralToolCalls(delta.ToolCalls)
		}
	}
	if chunk.Usage.TotalTokens > 0 {
		sc.Usage = &schema.Usage{
			InputTokens:  chunk.Usage.PromptTokens,
			OutputTokens: chunk.Usage.CompletionTokens,
			TotalTokens:  chunk.Usage.TotalTokens,
		}
	}
	return sc
}

// convertMistralToolCalls converts Mistral tool calls to Beluga tool calls.
func convertMistralToolCalls(calls []mistral.ToolCall) []schema.ToolCall {
	out := make([]schema.ToolCall, len(calls))
	for i, tc := range calls {
		out[i] = schema.ToolCall{
			ID:        tc.Id,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
	}
	return out
}

func convertMessages(msgs []schema.Message) []mistral.ChatMessage {
	out := make([]mistral.ChatMessage, 0, len(msgs))
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *schema.SystemMessage:
			out = append(out, mistral.ChatMessage{
				Role:    mistral.RoleSystem,
				Content: m.Text(),
			})
		case *schema.HumanMessage:
			out = append(out, mistral.ChatMessage{
				Role:    mistral.RoleUser,
				Content: m.Text(),
			})
		case *schema.AIMessage:
			cm := mistral.ChatMessage{
				Role:    mistral.RoleAssistant,
				Content: m.Text(),
			}
			if len(m.ToolCalls) > 0 {
				cm.ToolCalls = make([]mistral.ToolCall, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					cm.ToolCalls[i] = mistral.ToolCall{
						Id:   tc.ID,
						Type: mistral.ToolTypeFunction,
						Function: mistral.FunctionCall{
							Name:      tc.Name,
							Arguments: tc.Arguments,
						},
					}
				}
			}
			out = append(out, cm)
		case *schema.ToolMessage:
			out = append(out, mistral.ChatMessage{
				Role:    mistral.RoleTool,
				Content: m.Text(),
			})
		}
	}
	return out
}

func convertTools(tools []schema.ToolDefinition) []mistral.Tool {
	out := make([]mistral.Tool, len(tools))
	for i, t := range tools {
		var params any
		if t.InputSchema != nil {
			params = t.InputSchema
		}
		out[i] = mistral.Tool{
			Type: mistral.ToolTypeFunction,
			Function: mistral.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		}
	}
	return out
}

func convertResponse(resp *mistral.ChatCompletionResponse) *schema.AIMessage {
	if resp == nil {
		return &schema.AIMessage{}
	}
	ai := &schema.AIMessage{
		ModelID: resp.Model,
		Usage: schema.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message.Content != "" {
			ai.Parts = []schema.ContentPart{schema.TextPart{Text: choice.Message.Content}}
		}
		if len(choice.Message.ToolCalls) > 0 {
			ai.ToolCalls = convertResponseToolCalls(choice.Message.ToolCalls)
		}
	}
	return ai
}

// convertResponseToolCalls converts response tool calls, validating JSON arguments.
func convertResponseToolCalls(calls []mistral.ToolCall) []schema.ToolCall {
	out := make([]schema.ToolCall, len(calls))
	for i, tc := range calls {
		args := tc.Function.Arguments
		if _, err := json.Marshal(json.RawMessage(args)); err != nil {
			args = "{}"
		}
		out[i] = schema.ToolCall{
			ID:        tc.Id,
			Name:      tc.Function.Name,
			Arguments: args,
		}
	}
	return out
}
