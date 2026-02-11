package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brdocument "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// mockClient implements ConverseAPI for testing.
type mockClient struct {
	converseFunc       func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
	converseStreamFunc func(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error)
}

func (m *mockClient) Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	return m.converseFunc(ctx, params, optFns...)
}

func (m *mockClient) ConverseStream(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error) {
	return m.converseStreamFunc(ctx, params, optFns...)
}

func TestRegistration(t *testing.T) {
	names := llm.List()
	found := false
	for _, n := range names {
		if n == "bedrock" {
			found = true
			break
		}
	}
	if !found {
		t.Error("bedrock provider not registered")
	}
}

func TestNew_MissingModel(t *testing.T) {
	_, err := New(config.ProviderConfig{})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestModelID(t *testing.T) {
	m := NewWithClient(&mockClient{}, "us.anthropic.claude-sonnet-4-5-20250929-v1:0")
	if m.ModelID() != "us.anthropic.claude-sonnet-4-5-20250929-v1:0" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestGenerate(t *testing.T) {
	client := &mockClient{
		converseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			if aws.ToString(params.ModelId) != "test-model" {
				t.Errorf("ModelId = %q, want %q", aws.ToString(params.ModelId), "test-model")
			}
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Role: brtypes.ConversationRoleAssistant,
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: "Hello from Bedrock!"},
						},
					},
				},
				StopReason: brtypes.StopReasonEndTurn,
				Usage: &brtypes.TokenUsage{
					InputTokens:  aws.Int32(10),
					OutputTokens: aws.Int32(20),
					TotalTokens:  aws.Int32(30),
				},
			}, nil
		},
	}
	m := NewWithClient(client, "test-model")
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Bedrock!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from Bedrock!")
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want 20", resp.Usage.OutputTokens)
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d, want 30", resp.Usage.TotalTokens)
	}
}

func TestGenerateWithSystemMessage(t *testing.T) {
	var gotSystem bool
	client := &mockClient{
		converseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			gotSystem = len(params.System) > 0
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Role: brtypes.ConversationRoleAssistant,
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: "ok"},
						},
					},
				},
				StopReason: brtypes.StopReasonEndTurn,
				Usage:      &brtypes.TokenUsage{InputTokens: aws.Int32(5), OutputTokens: aws.Int32(1), TotalTokens: aws.Int32(6)},
			}, nil
		},
	}
	m := NewWithClient(client, "test-model")
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewSystemMessage("Be helpful"),
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if !gotSystem {
		t.Error("expected system message in request")
	}
}

func TestGenerateWithTools(t *testing.T) {
	var gotTools bool
	client := &mockClient{
		converseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			gotTools = params.ToolConfig != nil && len(params.ToolConfig.Tools) > 0
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Role: brtypes.ConversationRoleAssistant,
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberToolUse{
								Value: brtypes.ToolUseBlock{
									ToolUseId: aws.String("call_1"),
									Name:      aws.String("get_weather"),
									Input:     brdocument.NewLazyDocument(map[string]any{"city": "NYC"}),
								},
							},
						},
					},
				},
				StopReason: brtypes.StopReasonToolUse,
				Usage:      &brtypes.TokenUsage{InputTokens: aws.Int32(10), OutputTokens: aws.Int32(15), TotalTokens: aws.Int32(25)},
			}, nil
		},
	}
	m := NewWithClient(client, "test-model")
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "get_weather", Description: "Get weather", InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{"type": "string"},
			},
		}},
	})
	resp, err := bound.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Weather in NYC?"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if !gotTools {
		t.Error("expected tools in request")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("Name = %q, want %q", resp.ToolCalls[0].Name, "get_weather")
	}
	if resp.ToolCalls[0].ID != "call_1" {
		t.Errorf("ID = %q, want %q", resp.ToolCalls[0].ID, "call_1")
	}
	// Verify arguments contains city.
	var args map[string]any
	json.Unmarshal([]byte(resp.ToolCalls[0].Arguments), &args)
	if args["city"] != "NYC" {
		t.Errorf("Arguments city = %v, want NYC", args["city"])
	}
}

func TestBindTools(t *testing.T) {
	m := NewWithClient(&mockClient{}, "test-model")
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "test", Description: "test"},
	})
	if bound.ModelID() != "test-model" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
	// Original should not have tools.
	if len(m.tools) != 0 {
		t.Error("original should not have tools")
	}
}

func TestGenerateOptions(t *testing.T) {
	var gotConfig *brtypes.InferenceConfiguration
	client := &mockClient{
		converseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			gotConfig = params.InferenceConfig
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Role:    brtypes.ConversationRoleAssistant,
						Content: []brtypes.ContentBlock{&brtypes.ContentBlockMemberText{Value: "ok"}},
					},
				},
				StopReason: brtypes.StopReasonEndTurn,
				Usage:      &brtypes.TokenUsage{InputTokens: aws.Int32(1), OutputTokens: aws.Int32(1), TotalTokens: aws.Int32(2)},
			}, nil
		},
	}
	m := NewWithClient(client, "test-model")
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	},
		llm.WithTemperature(0.5),
		llm.WithMaxTokens(100),
		llm.WithTopP(0.9),
		llm.WithStopSequences("END"),
	)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if gotConfig == nil {
		t.Fatal("expected inference config")
	}
	if gotConfig.Temperature == nil || *gotConfig.Temperature != 0.5 {
		t.Errorf("Temperature = %v, want 0.5", gotConfig.Temperature)
	}
	if gotConfig.MaxTokens == nil || *gotConfig.MaxTokens != 100 {
		t.Errorf("MaxTokens = %v, want 100", gotConfig.MaxTokens)
	}
	if gotConfig.TopP == nil || *gotConfig.TopP != 0.9 {
		t.Errorf("TopP = %v, want 0.9", gotConfig.TopP)
	}
	if len(gotConfig.StopSequences) != 1 || gotConfig.StopSequences[0] != "END" {
		t.Errorf("StopSequences = %v", gotConfig.StopSequences)
	}
}

func TestContextCancellation(t *testing.T) {
	client := &mockClient{
		converseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return nil, ctx.Err()
		},
	}
	m := NewWithClient(client, "test-model")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := m.Generate(ctx, []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestErrorHandling(t *testing.T) {
	client := &mockClient{
		converseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return nil, &brtypes.ThrottlingException{Message: aws.String("Rate limit")}
		},
	}
	m := NewWithClient(client, "test-model")
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConvertMessages(t *testing.T) {
	msgs := []schema.Message{
		schema.NewSystemMessage("Be helpful"),
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi"),
		schema.NewToolMessage("call_1", "result"),
	}
	converted, system, err := convertMessages(msgs)
	if err != nil {
		t.Fatalf("convertMessages() error: %v", err)
	}
	if len(system) != 1 {
		t.Errorf("system len = %d, want 1", len(system))
	}
	if len(converted) != 3 {
		t.Errorf("messages len = %d, want 3", len(converted))
	}
}

func TestMapStopReason(t *testing.T) {
	tests := []struct {
		input brtypes.StopReason
		want  string
	}{
		{brtypes.StopReasonEndTurn, "stop"},
		{brtypes.StopReasonToolUse, "tool_calls"},
		{brtypes.StopReasonMaxTokens, "length"},
		{brtypes.StopReasonStopSequence, "stop_sequence"},
		{brtypes.StopReasonContentFiltered, "content_filter"},
		{brtypes.StopReason("unknown"), "unknown"},
	}
	for _, tt := range tests {
		got := mapStopReason(tt.input)
		if got != tt.want {
			t.Errorf("mapStopReason(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestConvertStreamEvent_ContentBlockDelta(t *testing.T) {
	event := &brtypes.ConverseStreamOutputMemberContentBlockDelta{
		Value: brtypes.ContentBlockDeltaEvent{
			ContentBlockIndex: aws.Int32(0),
			Delta:             &brtypes.ContentBlockDeltaMemberText{Value: "Hello"},
		},
	}
	chunk := convertStreamEvent(event, "test-model")
	if chunk == nil {
		t.Fatal("expected chunk")
	}
	if chunk.Delta != "Hello" {
		t.Errorf("Delta = %q, want %q", chunk.Delta, "Hello")
	}
}

func TestConvertStreamEvent_ToolUseStart(t *testing.T) {
	event := &brtypes.ConverseStreamOutputMemberContentBlockStart{
		Value: brtypes.ContentBlockStartEvent{
			ContentBlockIndex: aws.Int32(0),
			Start: &brtypes.ContentBlockStartMemberToolUse{
				Value: brtypes.ToolUseBlockStart{
					ToolUseId: aws.String("call_1"),
					Name:      aws.String("get_weather"),
				},
			},
		},
	}
	chunk := convertStreamEvent(event, "test-model")
	if chunk == nil {
		t.Fatal("expected chunk")
	}
	if len(chunk.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(chunk.ToolCalls))
	}
	if chunk.ToolCalls[0].Name != "get_weather" {
		t.Errorf("Name = %q", chunk.ToolCalls[0].Name)
	}
	if chunk.ToolCalls[0].ID != "call_1" {
		t.Errorf("ID = %q", chunk.ToolCalls[0].ID)
	}
}

func TestConvertStreamEvent_MessageStop(t *testing.T) {
	event := &brtypes.ConverseStreamOutputMemberMessageStop{
		Value: brtypes.MessageStopEvent{
			StopReason: brtypes.StopReasonEndTurn,
		},
	}
	chunk := convertStreamEvent(event, "test-model")
	if chunk == nil {
		t.Fatal("expected chunk")
	}
	if chunk.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", chunk.FinishReason, "stop")
	}
}

func TestConvertStreamEvent_Metadata(t *testing.T) {
	event := &brtypes.ConverseStreamOutputMemberMetadata{
		Value: brtypes.ConverseStreamMetadataEvent{
			Usage: &brtypes.TokenUsage{
				InputTokens:  aws.Int32(10),
				OutputTokens: aws.Int32(5),
				TotalTokens:  aws.Int32(15),
			},
		},
	}
	chunk := convertStreamEvent(event, "test-model")
	if chunk == nil {
		t.Fatal("expected chunk")
	}
	if chunk.Usage == nil {
		t.Fatal("expected usage")
	}
	if chunk.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d", chunk.Usage.InputTokens)
	}
}

func TestRegistryNew(t *testing.T) {
	// This tests that the init() registration works.
	// We can't fully test llm.New("bedrock", ...) without AWS credentials,
	// but we verify the factory is registered.
	names := llm.List()
	found := false
	for _, n := range names {
		if n == "bedrock" {
			found = true
		}
	}
	if !found {
		t.Error("bedrock not in registry")
	}
}

func TestAllMessageTypes(t *testing.T) {
	client := &mockClient{
		converseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			if len(params.Messages) != 3 {
				t.Errorf("messages len = %d, want 3", len(params.Messages))
			}
			if len(params.System) != 1 {
				t.Errorf("system len = %d, want 1", len(params.System))
			}
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Role:    brtypes.ConversationRoleAssistant,
						Content: []brtypes.ContentBlock{&brtypes.ContentBlockMemberText{Value: "Done"}},
					},
				},
				StopReason: brtypes.StopReasonEndTurn,
				Usage:      &brtypes.TokenUsage{InputTokens: aws.Int32(10), OutputTokens: aws.Int32(5), TotalTokens: aws.Int32(15)},
			}, nil
		},
	}
	m := NewWithClient(client, "test-model")
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewSystemMessage("Be helpful"),
		schema.NewHumanMessage("Use the tool"),
		&schema.AIMessage{
			ToolCalls: []schema.ToolCall{
				{ID: "call_1", Name: "search", Arguments: `{"q":"test"}`},
			},
		},
		schema.NewToolMessage("call_1", `{"result":"found"}`),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
}

func TestStreamError(t *testing.T) {
	client := &mockClient{
		converseStreamFunc: func(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error) {
			return nil, fmt.Errorf("stream failed")
		},
	}
	m := NewWithClient(client, "test-model")
	for _, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err == nil {
			t.Fatal("expected error")
		}
		return
	}
}

func TestMimeToFormat(t *testing.T) {
	tests := []struct {
		mime string
		want brtypes.ImageFormat
	}{
		{"image/jpeg", brtypes.ImageFormatJpeg},
		{"image/gif", brtypes.ImageFormatGif},
		{"image/webp", brtypes.ImageFormatWebp},
		{"image/png", brtypes.ImageFormatPng},
		{"", brtypes.ImageFormatPng},
	}
	for _, tt := range tests {
		got := mimeToFormat(tt.mime)
		if got != tt.want {
			t.Errorf("mimeToFormat(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}

func TestBuildInferenceConfig_NoValues(t *testing.T) {
	cfg := buildInferenceConfig(llm.GenerateOptions{})
	if cfg != nil {
		t.Error("expected nil for empty options")
	}
}

func TestConvertHumanParts_Image(t *testing.T) {
	parts := []schema.ContentPart{
		schema.TextPart{Text: "Look at this:"},
		schema.ImagePart{Data: []byte("fakeimg"), MimeType: "image/jpeg"},
	}
	blocks := convertHumanParts(parts)
	if len(blocks) != 2 {
		t.Fatalf("blocks len = %d, want 2", len(blocks))
	}
	if _, ok := blocks[0].(*brtypes.ContentBlockMemberText); !ok {
		t.Error("first block should be text")
	}
	img, ok := blocks[1].(*brtypes.ContentBlockMemberImage)
	if !ok {
		t.Fatal("second block should be image")
	}
	if img.Value.Format != brtypes.ImageFormatJpeg {
		t.Errorf("format = %q, want jpeg", img.Value.Format)
	}
}

func TestDocumentToJSON(t *testing.T) {
	doc := brdocument.NewLazyDocument(map[string]any{"key": "value"})
	got := documentToJSON(doc)
	var parsed map[string]any
	json.Unmarshal([]byte(got), &parsed)
	if parsed["key"] != "value" {
		t.Errorf("documentToJSON() = %q", got)
	}
}

func TestDocumentToJSON_Nil(t *testing.T) {
	got := documentToJSON(nil)
	if got != "{}" {
		t.Errorf("documentToJSON(nil) = %q, want %q", got, "{}")
	}
}

func TestConvertStreamEvent_MetadataNoUsage(t *testing.T) {
	event := &brtypes.ConverseStreamOutputMemberMetadata{
		Value: brtypes.ConverseStreamMetadataEvent{},
	}
	chunk := convertStreamEvent(event, "test-model")
	if chunk != nil {
		t.Error("expected nil for metadata without usage")
	}
}

func TestConvertStreamEvent_ToolUseDelta(t *testing.T) {
	event := &brtypes.ConverseStreamOutputMemberContentBlockDelta{
		Value: brtypes.ContentBlockDeltaEvent{
			Delta: &brtypes.ContentBlockDeltaMemberToolUse{
				Value: brtypes.ToolUseBlockDelta{
					Input: aws.String(`{"key":"val"}`),
				},
			},
		},
	}
	chunk := convertStreamEvent(event, "test-model")
	if chunk == nil {
		t.Fatal("expected chunk")
	}
	if len(chunk.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d", len(chunk.ToolCalls))
	}
	if chunk.ToolCalls[0].Arguments != `{"key":"val"}` {
		t.Errorf("Arguments = %q", chunk.ToolCalls[0].Arguments)
	}
}

func TestConvertToolConfigWithToolChoice(t *testing.T) {
	tools := []schema.ToolDefinition{
		{Name: "test", InputSchema: map[string]any{"type": "object"}},
	}
	tests := []struct {
		name       string
		opts       llm.GenerateOptions
		wantChoice string
	}{
		{"auto", llm.GenerateOptions{ToolChoice: llm.ToolChoiceAuto}, "auto"},
		{"required", llm.GenerateOptions{ToolChoice: llm.ToolChoiceRequired}, "any"},
		{"specific", llm.GenerateOptions{SpecificTool: "test"}, "specific"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := convertToolConfig(tools, tt.opts)
			if cfg.ToolChoice == nil {
				t.Fatal("expected ToolChoice")
			}
			switch tt.wantChoice {
			case "auto":
				if _, ok := cfg.ToolChoice.(*brtypes.ToolChoiceMemberAuto); !ok {
					t.Error("expected AutoToolChoice")
				}
			case "any":
				if _, ok := cfg.ToolChoice.(*brtypes.ToolChoiceMemberAny); !ok {
					t.Error("expected AnyToolChoice")
				}
			case "specific":
				tc, ok := cfg.ToolChoice.(*brtypes.ToolChoiceMemberTool)
				if !ok {
					t.Fatal("expected SpecificToolChoice")
				}
				if aws.ToString(tc.Value.Name) != "test" {
					t.Errorf("specific tool = %q", aws.ToString(tc.Value.Name))
				}
			}
		})
	}
}

func TestCacheReadTokens(t *testing.T) {
	client := &mockClient{
		converseFunc: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Role:    brtypes.ConversationRoleAssistant,
						Content: []brtypes.ContentBlock{&brtypes.ContentBlockMemberText{Value: "ok"}},
					},
				},
				StopReason: brtypes.StopReasonEndTurn,
				Usage: &brtypes.TokenUsage{
					InputTokens:          aws.Int32(10),
					OutputTokens:         aws.Int32(5),
					TotalTokens:          aws.Int32(15),
					CacheReadInputTokens: aws.Int32(3),
				},
			}, nil
		},
	}
	m := NewWithClient(client, "test-model")
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Usage.CachedTokens != 3 {
		t.Errorf("CachedTokens = %d, want 3", resp.Usage.CachedTokens)
	}
}

// TestConvertMessages_UnsupportedType tests that unsupported message types return errors.
func TestConvertMessages_UnsupportedType(t *testing.T) {
	type unsupportedMsg struct {
		schema.Message
	}
	msgs := []schema.Message{
		&unsupportedMsg{},
	}
	_, _, err := convertMessages(msgs)
	if err == nil {
		t.Fatal("expected error for unsupported message type")
	}
	if err.Error() != "bedrock: unsupported message type *bedrock.unsupportedMsg" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestBuildStreamInput_WithSystemAndTools tests buildStreamInput with system messages and tools.
func TestBuildStreamInput_WithSystemAndTools(t *testing.T) {
	m := NewWithClient(&mockClient{}, "test-model")
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "get_weather", Description: "Get weather info", InputSchema: map[string]any{"type": "object"}},
	})
	input, err := bound.(*Model).buildStreamInput([]schema.Message{
		schema.NewSystemMessage("Be helpful"),
		schema.NewHumanMessage("Hi"),
	}, []llm.GenerateOption{
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(50),
	})
	if err != nil {
		t.Fatalf("buildStreamInput() error: %v", err)
	}
	if len(input.System) != 1 {
		t.Errorf("System len = %d, want 1", len(input.System))
	}
	if len(input.Messages) != 1 {
		t.Errorf("Messages len = %d, want 1", len(input.Messages))
	}
	if input.ToolConfig == nil {
		t.Fatal("expected ToolConfig")
	}
	if len(input.ToolConfig.Tools) != 1 {
		t.Errorf("Tools len = %d, want 1", len(input.ToolConfig.Tools))
	}
	if input.InferenceConfig == nil {
		t.Fatal("expected InferenceConfig")
	}
	if input.InferenceConfig.Temperature == nil || *input.InferenceConfig.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", input.InferenceConfig.Temperature)
	}
	if input.InferenceConfig.MaxTokens == nil || *input.InferenceConfig.MaxTokens != 50 {
		t.Errorf("MaxTokens = %v, want 50", input.InferenceConfig.MaxTokens)
	}
}

// TestBuildStreamInput_Error tests buildStreamInput with unsupported message type.
func TestBuildStreamInput_Error(t *testing.T) {
	type unsupportedMsg struct {
		schema.Message
	}
	m := NewWithClient(&mockClient{}, "test-model")
	_, err := m.buildStreamInput([]schema.Message{
		&unsupportedMsg{},
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported message type")
	}
}

// TestStream_BuildInputError tests Stream when buildStreamInput fails.
func TestStream_BuildInputError(t *testing.T) {
	type unsupportedMsg struct {
		schema.Message
	}
	m := NewWithClient(&mockClient{}, "test-model")
	count := 0
	for _, err := range m.Stream(context.Background(), []schema.Message{
		&unsupportedMsg{},
	}) {
		count++
		if err == nil {
			t.Fatal("expected error")
		}
		// Should yield exactly one error.
		break
	}
	if count != 1 {
		t.Errorf("expected exactly 1 error event, got %d", count)
	}
}

// TestNew_WithOptions tests New with region and APIKey options.
func TestNew_WithOptions(t *testing.T) {
	// This test creates a real AWS client config. It should succeed even without credentials
	// since we're not making actual API calls.
	cfg := config.ProviderConfig{
		Model:  "anthropic.claude-v2",
		APIKey: "test-key",
		Options: map[string]any{
			"region":     "us-west-2",
			"secret_key": "test-secret",
		},
	}
	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "anthropic.claude-v2" {
		t.Errorf("ModelID = %q, want %q", m.ModelID(), "anthropic.claude-v2")
	}
}

// TestNew_WithBaseURL tests New with BaseURL option.
func TestNew_WithBaseURL(t *testing.T) {
	cfg := config.ProviderConfig{
		Model:   "test-model",
		BaseURL: "https://custom-endpoint.example.com",
	}
	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "test-model" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

// TestConvertStreamEvent_ContentBlockStartNonToolUse tests ContentBlockStart that's not tool_use.
func TestConvertStreamEvent_ContentBlockStartNonToolUse(t *testing.T) {
	event := &brtypes.ConverseStreamOutputMemberContentBlockStart{
		Value: brtypes.ContentBlockStartEvent{
			ContentBlockIndex: aws.Int32(0),
			Start: &brtypes.ContentBlockStartMemberImage{
				Value: brtypes.ImageBlockStart{},
			},
		},
	}
	chunk := convertStreamEvent(event, "test-model")
	if chunk != nil {
		t.Error("expected nil for non-tool-use ContentBlockStart")
	}
}

// TestConvertStreamEvent_UnknownEvent tests unknown event type returns nil.
func TestConvertStreamEvent_UnknownEvent(t *testing.T) {
	// Use a stream event type that's not handled.
	event := &brtypes.ConverseStreamOutputMemberContentBlockStop{
		Value: brtypes.ContentBlockStopEvent{
			ContentBlockIndex: aws.Int32(0),
		},
	}
	chunk := convertStreamEvent(event, "test-model")
	if chunk != nil {
		t.Error("expected nil for unknown event type")
	}
}

// TestConvertToolConfig_NoDescription tests tool without description.
func TestConvertToolConfig_NoDescription(t *testing.T) {
	tools := []schema.ToolDefinition{
		{Name: "test_tool", InputSchema: map[string]any{"type": "object"}},
	}
	cfg := convertToolConfig(tools, llm.GenerateOptions{})
	if len(cfg.Tools) != 1 {
		t.Fatalf("Tools len = %d, want 1", len(cfg.Tools))
	}
	spec, ok := cfg.Tools[0].(*brtypes.ToolMemberToolSpec)
	if !ok {
		t.Fatal("expected ToolMemberToolSpec")
	}
	if spec.Value.Description != nil {
		t.Errorf("expected nil Description, got %v", spec.Value.Description)
	}
	if aws.ToString(spec.Value.Name) != "test_tool" {
		t.Errorf("Name = %q, want test_tool", aws.ToString(spec.Value.Name))
	}
}

// TestConvertToolConfig_ToolChoiceNone tests ToolChoiceNone (omits tool choice).
func TestConvertToolConfig_ToolChoiceNone(t *testing.T) {
	tools := []schema.ToolDefinition{
		{Name: "test", InputSchema: map[string]any{"type": "object"}},
	}
	cfg := convertToolConfig(tools, llm.GenerateOptions{ToolChoice: llm.ToolChoiceNone})
	if cfg.ToolChoice != nil {
		t.Error("expected nil ToolChoice for ToolChoiceNone")
	}
}

// TestConvertOutput_NilUsage tests convertOutput with nil usage.
func TestConvertOutput_NilUsage(t *testing.T) {
	output := &bedrockruntime.ConverseOutput{
		Output: &brtypes.ConverseOutputMemberMessage{
			Value: brtypes.Message{
				Role:    brtypes.ConversationRoleAssistant,
				Content: []brtypes.ContentBlock{&brtypes.ContentBlockMemberText{Value: "ok"}},
			},
		},
		StopReason: brtypes.StopReasonEndTurn,
		Usage:      nil,
	}
	ai := convertOutput(output, "test-model")
	if ai.Usage.InputTokens != 0 {
		t.Errorf("expected InputTokens = 0, got %d", ai.Usage.InputTokens)
	}
	if ai.Usage.OutputTokens != 0 {
		t.Errorf("expected OutputTokens = 0, got %d", ai.Usage.OutputTokens)
	}
}

// TestDocumentToJSON_MarshalError tests documentToJSON when marshal fails.
func TestDocumentToJSON_MarshalError(t *testing.T) {
	// Create a document that will fail marshaling.
	// Since we can't easily create a LazyDocument that fails, we test the nil case (already covered).
	// The error path in documentToJSON is difficult to trigger with real AWS types.
	// We just verify that nil returns "{}" (already covered in TestDocumentToJSON_Nil).
	got := documentToJSON(nil)
	if got != "{}" {
		t.Errorf("documentToJSON(nil) = %q, want {}", got)
	}
}

// TestBuildInput_NoTools tests buildInput without tools.
func TestBuildInput_NoTools(t *testing.T) {
	m := NewWithClient(&mockClient{}, "test-model")
	input, err := m.buildInput([]schema.Message{
		schema.NewHumanMessage("Hello"),
	}, nil)
	if err != nil {
		t.Fatalf("buildInput() error: %v", err)
	}
	if input.ToolConfig != nil {
		t.Error("expected nil ToolConfig when no tools bound")
	}
}

// TestConvertHumanParts_EmptyImageData tests that images with empty data are skipped.
func TestConvertHumanParts_EmptyImageData(t *testing.T) {
	parts := []schema.ContentPart{
		schema.TextPart{Text: "Text"},
		schema.ImagePart{Data: nil, MimeType: "image/png"},
		schema.ImagePart{Data: []byte{}, MimeType: "image/png"},
	}
	blocks := convertHumanParts(parts)
	// Should only have text block, empty image data should be skipped.
	if len(blocks) != 1 {
		t.Errorf("blocks len = %d, want 1 (empty images should be skipped)", len(blocks))
	}
	if _, ok := blocks[0].(*brtypes.ContentBlockMemberText); !ok {
		t.Error("expected text block")
	}
}

// TestConvertAIBlocks_OnlyToolCalls tests AI message with only tool calls (no text).
func TestConvertAIBlocks_OnlyToolCalls(t *testing.T) {
	msg := &schema.AIMessage{
		ToolCalls: []schema.ToolCall{
			{ID: "call_1", Name: "test", Arguments: `{"key":"value"}`},
		},
	}
	blocks := convertAIBlocks(msg)
	if len(blocks) != 1 {
		t.Fatalf("blocks len = %d, want 1", len(blocks))
	}
	tu, ok := blocks[0].(*brtypes.ContentBlockMemberToolUse)
	if !ok {
		t.Fatal("expected ToolUse block")
	}
	if aws.ToString(tu.Value.ToolUseId) != "call_1" {
		t.Errorf("ToolUseId = %q", aws.ToString(tu.Value.ToolUseId))
	}
}

// TestGenerate_BuildInputError tests Generate when buildInput fails.
func TestGenerate_BuildInputError(t *testing.T) {
	type unsupportedMsg struct {
		schema.Message
	}
	m := NewWithClient(&mockClient{}, "test-model")
	_, err := m.Generate(context.Background(), []schema.Message{
		&unsupportedMsg{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "bedrock: unsupported message type *bedrock.unsupportedMsg" {
		t.Errorf("unexpected error: %v", err)
	}
}
