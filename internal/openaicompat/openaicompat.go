package openaicompat

import (
	"context"
	"fmt"
	"iter"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

// Model implements llm.ChatModel using the OpenAI-compatible chat completions API.
type Model struct {
	client  openai.Client
	model   string
	tools   []schema.ToolDefinition
	options []option.RequestOption
}

// Compile-time interface check.
var _ llm.ChatModel = (*Model)(nil)

// New creates a new Model from a ProviderConfig.
// It configures the openai-go client with the provided API key and base URL.
func New(cfg config.ProviderConfig) (*Model, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("openaicompat: model is required")
	}
	opts := []option.RequestOption{}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, option.WithRequestTimeout(cfg.Timeout))
	}
	client := openai.NewClient(opts...)
	return &Model{
		client:  client,
		model:   cfg.Model,
		options: opts,
	}, nil
}

// NewWithOptions creates a new Model with additional openai-go request options.
// This allows providers to inject custom headers or middleware.
func NewWithOptions(cfg config.ProviderConfig, extraOpts ...option.RequestOption) (*Model, error) {
	m, err := New(cfg)
	if err != nil {
		return nil, err
	}
	m.options = append(m.options, extraOpts...)
	m.client = openai.NewClient(m.options...)
	return m, nil
}

// Generate sends messages and returns a complete AI response.
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	params, err := m.buildParams(msgs, opts)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: generate failed: %w", err)
	}
	return ConvertResponse(resp), nil
}

// Stream sends messages and returns an iterator of response chunks.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	params, err := m.buildParams(msgs, opts)
	if err != nil {
		return func(yield func(schema.StreamChunk, error) bool) {
			yield(schema.StreamChunk{}, err)
		}
	}
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}
	stream := m.client.Chat.Completions.NewStreaming(ctx, params)
	return StreamToSeq(stream, m.model)
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

func (m *Model) buildParams(msgs []schema.Message, opts []llm.GenerateOption) (openai.ChatCompletionNewParams, error) {
	converted, err := ConvertMessages(msgs)
	if err != nil {
		return openai.ChatCompletionNewParams{}, err
	}
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(m.model),
		Messages: converted,
	}
	if len(m.tools) > 0 {
		params.Tools = ConvertTools(m.tools)
	}
	genOpts := llm.ApplyOptions(opts...)
	applyGenerateOptions(&params, genOpts)
	return params, nil
}

func applyGenerateOptions(params *openai.ChatCompletionNewParams, opts llm.GenerateOptions) {
	if opts.Temperature != nil {
		params.Temperature = openai.Float(*opts.Temperature)
	}
	if opts.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(opts.MaxTokens))
	}
	if opts.TopP != nil {
		params.TopP = openai.Float(*opts.TopP)
	}
	if len(opts.StopSequences) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfStringArray: opts.StopSequences,
		}
	}
	applyToolChoice(params, opts)
	if opts.Format != nil {
		applyResponseFormat(params, opts.Format)
	}
}

// applyToolChoice sets the tool choice parameter from generate options.
func applyToolChoice(params *openai.ChatCompletionNewParams, opts llm.GenerateOptions) {
	if opts.SpecificTool != "" {
		params.ToolChoice = openai.ChatCompletionToolChoiceOptionParamOfChatCompletionNamedToolChoice(
			openai.ChatCompletionNamedToolChoiceFunctionParam{
				Name: opts.SpecificTool,
			},
		)
		return
	}
	switch opts.ToolChoice {
	case llm.ToolChoiceAuto:
		params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("auto"),
		}
	case llm.ToolChoiceNone:
		params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("none"),
		}
	case llm.ToolChoiceRequired:
		params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("required"),
		}
	}
}

// applyResponseFormat sets the response format parameter.
func applyResponseFormat(params *openai.ChatCompletionNewParams, format *llm.ResponseFormat) {
	switch format.Type {
	case "json_object":
		v := shared.NewResponseFormatJSONObjectParam()
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &v,
		}
	case "text":
		v := shared.NewResponseFormatTextParam()
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfText: &v,
		}
	case "json_schema":
		params.ResponseFormat = buildJSONSchemaFormat(format.Schema)
	}
}

// buildJSONSchemaFormat constructs a JSON Schema response format.
func buildJSONSchemaFormat(schema map[string]any) openai.ChatCompletionNewParamsResponseFormatUnion {
	name, _ := schema["name"].(string)
	if name == "" {
		name = "response_schema"
	}
	jsonSchemaParam := shared.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:   name,
		Schema: schema,
	}
	if desc, ok := schema["description"].(string); ok {
		jsonSchemaParam.Description = param.NewOpt(desc)
	}
	if strict, ok := schema["strict"].(bool); ok {
		jsonSchemaParam.Strict = param.NewOpt(strict)
	}
	v := shared.ResponseFormatJSONSchemaParam{
		JSONSchema: jsonSchemaParam,
	}
	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &v,
	}
}
