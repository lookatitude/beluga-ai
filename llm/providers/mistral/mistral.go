package mistral

import (
	"context"
	"encoding/json"
	"iter"
	"regexp"
	"strconv"
	"time"

	"github.com/gage-technologies/mistral-go"
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/errclass"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
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
		return nil, core.Errorf(core.ErrInvalidInput, "mistral: api_key is required")
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
		return nil, classifyError("mistral: generate failed", err)
	}
	return convertResponse(resp), nil
}

// mistralStatusRe extracts the status from mistral-go's untyped
// "(HTTP Error 429) ..." errors.
var mistralStatusRe = regexp.MustCompile(`\(HTTP Error (\d{3})\)`)

// classifyError maps a mistral-go error onto a typed core.Error. mistral-go
// returns plain fmt errors, so the HTTP status is parsed from the message.
func classifyError(prefix string, err error) error {
	status := 0
	if err != nil {
		if m := mistralStatusRe.FindStringSubmatch(err.Error()); m != nil {
			status, _ = strconv.Atoi(m[1])
		}
	}
	return errclass.Classify(prefix, status, err)
}

// Stream sends messages and returns an iterator of response chunks.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	chatMsgs := convertMessages(msgs)
	params := m.buildParams(opts)

	return func(yield func(schema.StreamChunk, error) bool) {
		// Open the stream lazily so an un-iterated Seq starts no request and
		// no SDK goroutine.
		ch, err := m.client.ChatStream(m.model, chatMsgs, params)
		if err != nil {
			yield(schema.StreamChunk{}, classifyError("mistral: stream failed", err))
			return
		}
		consumeStreamChannel(ctx, ch, m.model, yield)
	}
}

// drain discards the rest of ch in the background. mistral-go's reader
// goroutine sends on an unbuffered channel and ignores context, so it would
// block forever (holding the HTTP body) if the consumer stopped early.
func drain(ch <-chan mistral.ChatCompletionStreamResponse) {
	go func() {
		for range ch { //nolint:revive // intentionally empty: discard remaining events
		}
	}()
}

// consumeStreamChannel reads from the Mistral stream channel and yields chunks.
func consumeStreamChannel(ctx context.Context, ch <-chan mistral.ChatCompletionStreamResponse, modelID string, yield func(schema.StreamChunk, error) bool) {
	for {
		select {
		case <-ctx.Done():
			drain(ch)
			yield(schema.StreamChunk{}, ctx.Err())
			return
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			if chunk.Error != nil {
				drain(ch)
				yield(schema.StreamChunk{}, classifyError("mistral: stream error", chunk.Error))
				return
			}
			sc := convertStreamChunk(chunk, modelID)
			if !yield(sc, nil) {
				drain(ch)
				return
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
