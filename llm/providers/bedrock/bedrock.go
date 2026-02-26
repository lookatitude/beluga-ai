package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brdocument "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	cfgpkg "github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	llm.Register("bedrock", func(cfg cfgpkg.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// ConverseAPI defines the subset of bedrockruntime.Client methods we need.
// This allows injection of mock clients for testing.
type ConverseAPI interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
	ConverseStream(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error)
}

// Model implements llm.ChatModel using the AWS Bedrock Converse API.
type Model struct {
	client  ConverseAPI
	modelID string
	tools   []schema.ToolDefinition
}

// Compile-time interface check.
var _ llm.ChatModel = (*Model)(nil)

// New creates a new Bedrock ChatModel.
func New(cfg cfgpkg.ProviderConfig) (*Model, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("bedrock: model is required")
	}

	region, _ := cfgpkg.GetOption[string](cfg, "region")
	if region == "" {
		region = "us-east-1"
	}

	var awsOpts []func(*awsconfig.LoadOptions) error
	awsOpts = append(awsOpts, awsconfig.WithRegion(region))

	if cfg.APIKey != "" {
		secretKey, _ := cfgpkg.GetOption[string](cfg, "secret_key")
		awsOpts = append(awsOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.APIKey, secretKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsOpts...)
	if err != nil {
		return nil, fmt.Errorf("bedrock: failed to load AWS config: %w", err)
	}

	var brOpts []func(*bedrockruntime.Options)
	if cfg.BaseURL != "" {
		brOpts = append(brOpts, func(o *bedrockruntime.Options) {
			o.BaseEndpoint = aws.String(cfg.BaseURL)
		})
	}

	client := bedrockruntime.NewFromConfig(awsCfg, brOpts...)

	return &Model{
		client:  client,
		modelID: cfg.Model,
	}, nil
}

// NewWithClient creates a Model with a custom ConverseAPI implementation.
// This is useful for testing.
func NewWithClient(client ConverseAPI, modelID string) *Model {
	return &Model{
		client:  client,
		modelID: modelID,
	}
}

// Generate sends messages and returns a complete AI response.
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	input, err := m.buildInput(msgs, opts)
	if err != nil {
		return nil, err
	}
	output, err := m.client.Converse(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("bedrock: converse failed: %w", err)
	}
	return convertOutput(output, m.modelID), nil
}

// Stream sends messages and returns an iterator of response chunks.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	input, err := m.buildStreamInput(msgs, opts)
	if err != nil {
		return func(yield func(schema.StreamChunk, error) bool) {
			yield(schema.StreamChunk{}, err)
		}
	}

	return func(yield func(schema.StreamChunk, error) bool) {
		consumeBedrockStream(ctx, m.client, input, m.modelID, yield)
	}
}

// consumeBedrockStream opens a Bedrock converse stream and yields chunks to the caller.
func consumeBedrockStream(ctx context.Context, client ConverseAPI, input *bedrockruntime.ConverseStreamInput, modelID string, yield func(schema.StreamChunk, error) bool) {
	output, err := client.ConverseStream(ctx, input)
	if err != nil {
		yield(schema.StreamChunk{}, fmt.Errorf("bedrock: stream failed: %w", err))
		return
	}
	stream := output.GetStream()
	if stream == nil {
		return
	}
	defer stream.Close()

	for event := range stream.Events() {
		chunk := convertStreamEvent(event, modelID)
		if chunk == nil {
			continue
		}
		if !yield(*chunk, nil) {
			return
		}
	}
	if err := stream.Err(); err != nil {
		yield(schema.StreamChunk{}, fmt.Errorf("bedrock: stream error: %w", err))
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
	return m.modelID
}

func (m *Model) buildInput(msgs []schema.Message, opts []llm.GenerateOption) (*bedrockruntime.ConverseInput, error) {
	converted, system, err := convertMessages(msgs)
	if err != nil {
		return nil, err
	}
	genOpts := llm.ApplyOptions(opts...)
	input := &bedrockruntime.ConverseInput{
		ModelId:  aws.String(m.modelID),
		Messages: converted,
	}
	if len(system) > 0 {
		input.System = system
	}
	input.InferenceConfig = buildInferenceConfig(genOpts)
	if len(m.tools) > 0 {
		input.ToolConfig = convertToolConfig(m.tools, genOpts)
	}
	return input, nil
}

func (m *Model) buildStreamInput(msgs []schema.Message, opts []llm.GenerateOption) (*bedrockruntime.ConverseStreamInput, error) {
	converted, system, err := convertMessages(msgs)
	if err != nil {
		return nil, err
	}
	genOpts := llm.ApplyOptions(opts...)
	input := &bedrockruntime.ConverseStreamInput{
		ModelId:  aws.String(m.modelID),
		Messages: converted,
	}
	if len(system) > 0 {
		input.System = system
	}
	input.InferenceConfig = buildInferenceConfig(genOpts)
	if len(m.tools) > 0 {
		input.ToolConfig = convertToolConfig(m.tools, genOpts)
	}
	return input, nil
}

func buildInferenceConfig(opts llm.GenerateOptions) *brtypes.InferenceConfiguration {
	cfg := &brtypes.InferenceConfiguration{}
	hasValue := false
	if opts.Temperature != nil {
		t := float32(*opts.Temperature)
		cfg.Temperature = &t
		hasValue = true
	}
	if opts.TopP != nil {
		p := float32(*opts.TopP)
		cfg.TopP = &p
		hasValue = true
	}
	if opts.MaxTokens > 0 {
		n := int32(opts.MaxTokens)
		cfg.MaxTokens = &n
		hasValue = true
	}
	if len(opts.StopSequences) > 0 {
		cfg.StopSequences = opts.StopSequences
		hasValue = true
	}
	if !hasValue {
		return nil
	}
	return cfg
}

func convertMessages(msgs []schema.Message) ([]brtypes.Message, []brtypes.SystemContentBlock, error) {
	var system []brtypes.SystemContentBlock
	var out []brtypes.Message
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *schema.SystemMessage:
			system = append(system, &brtypes.SystemContentBlockMemberText{
				Value: m.Text(),
			})
		case *schema.HumanMessage:
			blocks := convertHumanParts(m.Parts)
			out = append(out, brtypes.Message{
				Role:    brtypes.ConversationRoleUser,
				Content: blocks,
			})
		case *schema.AIMessage:
			blocks := convertAIBlocks(m)
			out = append(out, brtypes.Message{
				Role:    brtypes.ConversationRoleAssistant,
				Content: blocks,
			})
		case *schema.ToolMessage:
			out = append(out, brtypes.Message{
				Role: brtypes.ConversationRoleUser,
				Content: []brtypes.ContentBlock{
					&brtypes.ContentBlockMemberToolResult{
						Value: brtypes.ToolResultBlock{
							ToolUseId: aws.String(m.ToolCallID),
							Content: []brtypes.ToolResultContentBlock{
								&brtypes.ToolResultContentBlockMemberText{
									Value: m.Text(),
								},
							},
						},
					},
				},
			})
		default:
			return nil, nil, fmt.Errorf("bedrock: unsupported message type %T", msg)
		}
	}
	return out, system, nil
}

func convertHumanParts(parts []schema.ContentPart) []brtypes.ContentBlock {
	var blocks []brtypes.ContentBlock
	for _, p := range parts {
		switch cp := p.(type) {
		case schema.TextPart:
			blocks = append(blocks, &brtypes.ContentBlockMemberText{Value: cp.Text})
		case schema.ImagePart:
			if len(cp.Data) > 0 {
				format := mimeToFormat(cp.MimeType)
				blocks = append(blocks, &brtypes.ContentBlockMemberImage{
					Value: brtypes.ImageBlock{
						Format: format,
						Source: &brtypes.ImageSourceMemberBytes{
							Value: cp.Data,
						},
					},
				})
			}
		}
	}
	return blocks
}

func mimeToFormat(mime string) brtypes.ImageFormat {
	switch mime {
	case "image/jpeg":
		return brtypes.ImageFormatJpeg
	case "image/gif":
		return brtypes.ImageFormatGif
	case "image/webp":
		return brtypes.ImageFormatWebp
	default:
		return brtypes.ImageFormatPng
	}
}

func convertAIBlocks(m *schema.AIMessage) []brtypes.ContentBlock {
	var blocks []brtypes.ContentBlock
	text := m.Text()
	if text != "" {
		blocks = append(blocks, &brtypes.ContentBlockMemberText{Value: text})
	}
	for _, tc := range m.ToolCalls {
		var input any
		json.Unmarshal([]byte(tc.Arguments), &input)
		blocks = append(blocks, &brtypes.ContentBlockMemberToolUse{
			Value: brtypes.ToolUseBlock{
				ToolUseId: aws.String(tc.ID),
				Name:      aws.String(tc.Name),
				Input:     brdocument.NewLazyDocument(input),
			},
		})
	}
	return blocks
}

func convertToolConfig(tools []schema.ToolDefinition, opts llm.GenerateOptions) *brtypes.ToolConfiguration {
	brTools := make([]brtypes.Tool, len(tools))
	for i, t := range tools {
		spec := brtypes.ToolSpecification{
			Name:        aws.String(t.Name),
			InputSchema: &brtypes.ToolInputSchemaMemberJson{Value: brdocument.NewLazyDocument(t.InputSchema)},
		}
		if t.Description != "" {
			spec.Description = aws.String(t.Description)
		}
		brTools[i] = &brtypes.ToolMemberToolSpec{Value: spec}
	}
	cfg := &brtypes.ToolConfiguration{
		Tools: brTools,
	}
	switch opts.ToolChoice {
	case llm.ToolChoiceAuto:
		cfg.ToolChoice = &brtypes.ToolChoiceMemberAuto{Value: brtypes.AutoToolChoice{}}
	case llm.ToolChoiceRequired:
		cfg.ToolChoice = &brtypes.ToolChoiceMemberAny{Value: brtypes.AnyToolChoice{}}
	case llm.ToolChoiceNone:
		// Bedrock doesn't have "none", just omit tool config to suppress.
	}
	if opts.SpecificTool != "" {
		cfg.ToolChoice = &brtypes.ToolChoiceMemberTool{
			Value: brtypes.SpecificToolChoice{
				Name: aws.String(opts.SpecificTool),
			},
		}
	}
	return cfg
}

func convertOutput(output *bedrockruntime.ConverseOutput, modelID string) *schema.AIMessage {
	ai := &schema.AIMessage{ModelID: modelID}
	if output.Usage != nil {
		ai.Usage = schema.Usage{
			InputTokens:  int(aws.ToInt32(output.Usage.InputTokens)),
			OutputTokens: int(aws.ToInt32(output.Usage.OutputTokens)),
			TotalTokens:  int(aws.ToInt32(output.Usage.TotalTokens)),
		}
		if output.Usage.CacheReadInputTokens != nil {
			ai.Usage.CachedTokens = int(aws.ToInt32(output.Usage.CacheReadInputTokens))
		}
	}
	ai.Metadata = map[string]any{
		"stop_reason": string(output.StopReason),
	}

	if msg, ok := output.Output.(*brtypes.ConverseOutputMemberMessage); ok {
		for _, block := range msg.Value.Content {
			switch b := block.(type) {
			case *brtypes.ContentBlockMemberText:
				ai.Parts = append(ai.Parts, schema.TextPart{Text: b.Value})
			case *brtypes.ContentBlockMemberToolUse:
				args := documentToJSON(b.Value.Input)
				ai.ToolCalls = append(ai.ToolCalls, schema.ToolCall{
					ID:        aws.ToString(b.Value.ToolUseId),
					Name:      aws.ToString(b.Value.Name),
					Arguments: args,
				})
			}
		}
	}
	return ai
}

func convertStreamEvent(event brtypes.ConverseStreamOutput, modelID string) *schema.StreamChunk {
	switch e := event.(type) {
	case *brtypes.ConverseStreamOutputMemberContentBlockDelta:
		chunk := &schema.StreamChunk{ModelID: modelID}
		switch d := e.Value.Delta.(type) {
		case *brtypes.ContentBlockDeltaMemberText:
			chunk.Delta = d.Value
		case *brtypes.ContentBlockDeltaMemberToolUse:
			chunk.ToolCalls = []schema.ToolCall{{
				Arguments: aws.ToString(d.Value.Input),
			}}
		}
		return chunk
	case *brtypes.ConverseStreamOutputMemberContentBlockStart:
		if tu, ok := e.Value.Start.(*brtypes.ContentBlockStartMemberToolUse); ok {
			return &schema.StreamChunk{
				ModelID: modelID,
				ToolCalls: []schema.ToolCall{{
					ID:   aws.ToString(tu.Value.ToolUseId),
					Name: aws.ToString(tu.Value.Name),
				}},
			}
		}
		return nil
	case *brtypes.ConverseStreamOutputMemberMessageStop:
		return &schema.StreamChunk{
			ModelID:      modelID,
			FinishReason: mapStopReason(e.Value.StopReason),
		}
	case *brtypes.ConverseStreamOutputMemberMetadata:
		if e.Value.Usage != nil {
			return &schema.StreamChunk{
				ModelID: modelID,
				Usage: &schema.Usage{
					InputTokens:  int(aws.ToInt32(e.Value.Usage.InputTokens)),
					OutputTokens: int(aws.ToInt32(e.Value.Usage.OutputTokens)),
					TotalTokens:  int(aws.ToInt32(e.Value.Usage.TotalTokens)),
				},
			}
		}
		return nil
	default:
		return nil
	}
}

func mapStopReason(reason brtypes.StopReason) string {
	switch reason {
	case brtypes.StopReasonEndTurn:
		return "stop"
	case brtypes.StopReasonToolUse:
		return "tool_calls"
	case brtypes.StopReasonMaxTokens:
		return "length"
	case brtypes.StopReasonStopSequence:
		return "stop_sequence"
	case brtypes.StopReasonContentFiltered:
		return "content_filter"
	default:
		return string(reason)
	}
}

func documentToJSON(doc brdocument.Interface) string {
	if doc == nil {
		return "{}"
	}
	b, err := doc.MarshalSmithyDocument()
	if err != nil {
		return "{}"
	}
	return string(b)
}
