package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	anthropicSDK "github.com/anthropics/anthropic-sdk-go"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// unsupportedTestMessage is a custom message type for testing unsupported message handling.
type unsupportedTestMessage struct{}

func (m *unsupportedTestMessage) GetRole() schema.Role              { return "custom" }
func (m *unsupportedTestMessage) GetContent() []schema.ContentPart  { return nil }
func (m *unsupportedTestMessage) GetMetadata() map[string]any       { return nil }

func mockAnthropicResponse(content string) string {
	resp := map[string]any{
		"id":            "msg_test",
		"type":          "message",
		"role":          "assistant",
		"model":         "claude-sonnet-4-5-20250929",
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"content": []map[string]any{
			{"type": "text", "text": content},
		},
		"usage": map[string]any{
			"input_tokens":               10,
			"output_tokens":              20,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     5,
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func mockAnthropicToolResponse() string {
	resp := map[string]any{
		"id":            "msg_tool",
		"type":          "message",
		"role":          "assistant",
		"model":         "claude-sonnet-4-5-20250929",
		"stop_reason":   "tool_use",
		"stop_sequence": nil,
		"content": []map[string]any{
			{"type": "text", "text": "I'll look up the weather."},
			{
				"type":  "tool_use",
				"id":    "toolu_01",
				"name":  "get_weather",
				"input": map[string]any{"city": "NYC"},
			},
		},
		"usage": map[string]any{
			"input_tokens":               15,
			"output_tokens":              25,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     0,
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func streamAnthropicResponse(text string) string {
	var sb strings.Builder
	// message_start event
	msgStart := map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            "msg_stream",
			"type":          "message",
			"role":          "assistant",
			"model":         "claude-sonnet-4-5-20250929",
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage":         map[string]any{"input_tokens": 10, "output_tokens": 0},
		},
	}
	b, _ := json.Marshal(msgStart)
	sb.WriteString("event: message_start\ndata: ")
	sb.Write(b)
	sb.WriteString("\n\n")

	// content_block_start
	blockStart := map[string]any{
		"type":          "content_block_start",
		"index":         0,
		"content_block": map[string]any{"type": "text", "text": ""},
	}
	b, _ = json.Marshal(blockStart)
	sb.WriteString("event: content_block_start\ndata: ")
	sb.Write(b)
	sb.WriteString("\n\n")

	// Split text into chunks for content_block_delta events.
	for _, ch := range strings.Split(text, "") {
		delta := map[string]any{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]any{"type": "text_delta", "text": ch},
		}
		b, _ = json.Marshal(delta)
		sb.WriteString("event: content_block_delta\ndata: ")
		sb.Write(b)
		sb.WriteString("\n\n")
	}

	// content_block_stop
	blockStop := map[string]any{"type": "content_block_stop", "index": 0}
	b, _ = json.Marshal(blockStop)
	sb.WriteString("event: content_block_stop\ndata: ")
	sb.Write(b)
	sb.WriteString("\n\n")

	// message_delta
	msgDelta := map[string]any{
		"type":  "message_delta",
		"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil},
		"usage": map[string]any{"output_tokens": 5},
	}
	b, _ = json.Marshal(msgDelta)
	sb.WriteString("event: message_delta\ndata: ")
	sb.Write(b)
	sb.WriteString("\n\n")

	// message_stop
	msgStop := map[string]any{"type": "message_stop"}
	b, _ = json.Marshal(msgStop)
	sb.WriteString("event: message_stop\ndata: ")
	sb.Write(b)
	sb.WriteString("\n\n")

	return sb.String()
}

func newTestModel(handler http.HandlerFunc) (*httptest.Server, *Model) {
	ts := httptest.NewServer(handler)
	m, _ := New(config.ProviderConfig{
		Model:   "claude-sonnet-4-5-20250929",
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	return ts, m
}

func TestRegistration(t *testing.T) {
	names := llm.List()
	found := false
	for _, n := range names {
		if n == "anthropic" {
			found = true
			break
		}
	}
	if !found {
		t.Error("anthropic provider not registered")
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:  "claude-sonnet-4-5-20250929",
		APIKey: "sk-ant-test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "claude-sonnet-4-5-20250929" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

func TestNew_MissingModel(t *testing.T) {
	_, err := New(config.ProviderConfig{APIKey: "test"})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestGenerate(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockAnthropicResponse("Hello from Claude!"))
	})
	defer ts.Close()

	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Claude!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from Claude!")
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want 20", resp.Usage.OutputTokens)
	}
	if resp.Usage.CachedTokens != 5 {
		t.Errorf("CachedTokens = %d, want 5", resp.Usage.CachedTokens)
	}
}

func TestGenerateWithSystemMessage(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		// Check system parameter is present.
		if _, ok := req["system"]; !ok {
			t.Error("expected system parameter")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockAnthropicResponse("I'm a helpful assistant"))
	})
	defer ts.Close()

	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewSystemMessage("You are helpful"),
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "I'm a helpful assistant" {
		t.Errorf("text = %q", resp.Text())
	}
}

func TestGenerateWithTools(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		if _, ok := req["tools"]; !ok {
			t.Error("expected tools in request")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockAnthropicToolResponse())
	})
	defer ts.Close()

	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "get_weather", Description: "Get weather"},
	})
	resp, err := bound.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Weather in NYC?"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("Name = %q, want %q", resp.ToolCalls[0].Name, "get_weather")
	}
	if resp.ToolCalls[0].ID != "toolu_01" {
		t.Errorf("ID = %q, want %q", resp.ToolCalls[0].ID, "toolu_01")
	}
}

func TestStream(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, streamAnthropicResponse("Hi"))
	})
	defer ts.Close()

	var text strings.Builder
	var gotFinish string
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hello"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		text.WriteString(chunk.Delta)
		if chunk.FinishReason != "" {
			gotFinish = chunk.FinishReason
		}
	}
	if text.String() != "Hi" {
		t.Errorf("streamed text = %q, want %q", text.String(), "Hi")
	}
	if gotFinish != "stop" {
		t.Errorf("FinishReason = %q, want %q", gotFinish, "stop")
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "test", Description: "test"},
	})
	if bound.ModelID() != "claude-sonnet-4-5-20250929" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
	// Original should not have tools.
	if len(m.tools) != 0 {
		t.Error("original should not have tools")
	}
}

func TestModelID(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-opus-4-6", APIKey: "test",
	})
	if m.ModelID() != "claude-opus-4-6" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestErrorHandling(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`)
	})
	defer ts.Close()

	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error from 401")
	}
}

func TestMapStopReason(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"end_turn", "stop"},
		{"tool_use", "tool_calls"},
		{"max_tokens", "length"},
		{"stop_sequence", "stop_sequence"},
	}
	for _, tt := range tests {
		got := mapStopReason(tt.input)
		if got != tt.want {
			t.Errorf("mapStopReason(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("anthropic", config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "claude-sonnet-4-5-20250929" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestGenerateOptions(t *testing.T) {
	var capturedBody map[string]any
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockAnthropicResponse("ok"))
	})
	defer ts.Close()

	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	},
		llm.WithTemperature(0.5),
		llm.WithMaxTokens(100),
		llm.WithTopP(0.9),
	)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if temp, ok := capturedBody["temperature"].(float64); !ok || temp != 0.5 {
		t.Errorf("temperature = %v", capturedBody["temperature"])
	}
	if maxT, ok := capturedBody["max_tokens"].(float64); !ok || int(maxT) != 100 {
		t.Errorf("max_tokens = %v", capturedBody["max_tokens"])
	}
	if topP, ok := capturedBody["top_p"].(float64); !ok || topP != 0.9 {
		t.Errorf("top_p = %v", capturedBody["top_p"])
	}
}

func TestContextCancellation(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := m.Generate(ctx, []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// TestNew_WithTimeout tests timeout configuration.
func TestNew_WithTimeout(t *testing.T) {
	cfg := config.ProviderConfig{
		Model:   "claude-sonnet-4-5-20250929",
		APIKey:  "test-key",
		Timeout: 5000000000, // 5 seconds in nanoseconds
	}
	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "claude-sonnet-4-5-20250929" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

// TestConvertAIContentParts tests conversion of AI message to content blocks.
func TestConvertAIContentParts(t *testing.T) {
	aiMsg := &schema.AIMessage{
		Parts: []schema.ContentPart{
			schema.TextPart{Text: "I'll help with that."},
		},
		ToolCalls: []schema.ToolCall{
			{
				ID:        "toolu_123",
				Name:      "get_weather",
				Arguments: `{"city":"NYC"}`,
			},
		},
	}
	blocks := convertAIContentParts(aiMsg)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	// First block should be text.
	if blocks[0].OfText == nil || blocks[0].OfText.Text != "I'll help with that." {
		t.Errorf("first block text = %v", blocks[0])
	}
	// Second block should be tool_use.
	if blocks[1].OfToolUse == nil || blocks[1].OfToolUse.Name != "get_weather" {
		t.Errorf("second block = %v", blocks[1])
	}
}

// TestConvertAIContentParts_Empty tests empty AI message.
func TestConvertAIContentParts_Empty(t *testing.T) {
	aiMsg := &schema.AIMessage{}
	blocks := convertAIContentParts(aiMsg)
	if len(blocks) != 0 {
		t.Errorf("got %d blocks, want 0", len(blocks))
	}
}

// TestConvertContentParts_ImageURL tests ImagePart with URL.
func TestConvertContentParts_ImageURL(t *testing.T) {
	parts := []schema.ContentPart{
		schema.ImagePart{URL: "https://example.com/image.png"},
	}
	blocks := convertContentParts(parts)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].OfImage == nil {
		t.Fatal("expected image block")
	}
	if blocks[0].OfImage.Source.OfURL == nil {
		t.Fatal("expected URL source")
	}
	if blocks[0].OfImage.Source.OfURL.URL != "https://example.com/image.png" {
		t.Errorf("URL = %q", blocks[0].OfImage.Source.OfURL.URL)
	}
}

// TestConvertContentParts_ImageData tests ImagePart with Data bytes and default MIME.
func TestConvertContentParts_ImageData(t *testing.T) {
	parts := []schema.ContentPart{
		schema.ImagePart{Data: []byte{0x89, 0x50, 0x4E, 0x47}}, // PNG header
	}
	blocks := convertContentParts(parts)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].OfImage == nil {
		t.Fatal("expected image block")
	}
	if blocks[0].OfImage.Source.OfBase64 == nil {
		t.Fatal("expected base64 source")
	}
	// Should default to "image/png".
	if blocks[0].OfImage.Source.OfBase64.MediaType != "image/png" {
		t.Errorf("MediaType = %q, want image/png", blocks[0].OfImage.Source.OfBase64.MediaType)
	}
}

// TestConvertContentParts_ImageDataWithMime tests ImagePart with custom MimeType.
func TestConvertContentParts_ImageDataWithMime(t *testing.T) {
	parts := []schema.ContentPart{
		schema.ImagePart{Data: []byte{0xFF, 0xD8}, MimeType: "image/jpeg"},
	}
	blocks := convertContentParts(parts)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].OfImage == nil {
		t.Fatal("expected image block")
	}
	if blocks[0].OfImage.Source.OfBase64 == nil {
		t.Fatal("expected base64 source")
	}
	if blocks[0].OfImage.Source.OfBase64.MediaType != "image/jpeg" {
		t.Errorf("MediaType = %q, want image/jpeg", blocks[0].OfImage.Source.OfBase64.MediaType)
	}
}

// TestConvertMessages_ToolMessage tests ToolMessage conversion.
func TestConvertMessages_ToolMessage(t *testing.T) {
	msgs := []schema.Message{
		schema.NewToolMessage("toolu_xyz", "42 degrees"),
	}
	converted, system, err := convertMessages(msgs)
	if err != nil {
		t.Fatalf("convertMessages() error: %v", err)
	}
	if len(converted) != 1 {
		t.Fatalf("got %d messages, want 1", len(converted))
	}
	if len(system) != 0 {
		t.Errorf("got %d system blocks, want 0", len(system))
	}
	// ToolMessage becomes user message with tool_result block.
	if converted[0].Role != "user" {
		t.Errorf("role = %q, want user", converted[0].Role)
	}
}

// TestConvertMessages_UnsupportedType tests unsupported message type returns error.
func TestConvertMessages_UnsupportedType(t *testing.T) {
	msgs := []schema.Message{
		&unsupportedTestMessage{},
	}
	_, _, err := convertMessages(msgs)
	if err == nil {
		t.Fatal("expected error for unsupported message type")
	}
	if !strings.Contains(err.Error(), "unsupported message type") {
		t.Errorf("error message = %q", err.Error())
	}
}

// TestBuildParams_ToolChoiceAuto tests ToolChoice auto.
func TestBuildParams_ToolChoiceAuto(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	params, err := m.buildParams([]schema.Message{
		schema.NewHumanMessage("test"),
	}, []llm.GenerateOption{
		llm.WithToolChoice(llm.ToolChoiceAuto),
	})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if params.ToolChoice.OfAuto == nil {
		t.Error("ToolChoice.OfAuto is nil")
	}
}

// TestBuildParams_ToolChoiceNone tests ToolChoice none.
func TestBuildParams_ToolChoiceNone(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	params, err := m.buildParams([]schema.Message{
		schema.NewHumanMessage("test"),
	}, []llm.GenerateOption{
		llm.WithToolChoice(llm.ToolChoiceNone),
	})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if params.ToolChoice.OfNone == nil {
		t.Error("ToolChoice.OfNone is nil")
	}
}

// TestBuildParams_ToolChoiceRequired tests ToolChoice required.
func TestBuildParams_ToolChoiceRequired(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	params, err := m.buildParams([]schema.Message{
		schema.NewHumanMessage("test"),
	}, []llm.GenerateOption{
		llm.WithToolChoice(llm.ToolChoiceRequired),
	})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if params.ToolChoice.OfAny == nil {
		t.Error("ToolChoice.OfAny is nil")
	}
}

// TestBuildParams_SpecificTool tests SpecificTool.
func TestBuildParams_SpecificTool(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	params, err := m.buildParams([]schema.Message{
		schema.NewHumanMessage("test"),
	}, []llm.GenerateOption{
		llm.WithSpecificTool("get_weather"),
	})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if params.ToolChoice.OfTool == nil {
		t.Fatal("ToolChoice.OfTool is nil")
	}
	if params.ToolChoice.OfTool.Name != "get_weather" {
		t.Errorf("tool name = %q, want get_weather", params.ToolChoice.OfTool.Name)
	}
}

// TestBuildParams_StopSequences tests StopSequences.
func TestBuildParams_StopSequences(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	params, err := m.buildParams([]schema.Message{
		schema.NewHumanMessage("test"),
	}, []llm.GenerateOption{
		llm.WithStopSequences("STOP", "END"),
	})
	if err != nil {
		t.Fatalf("buildParams() error: %v", err)
	}
	if len(params.StopSequences) != 2 {
		t.Fatalf("got %d stop sequences, want 2", len(params.StopSequences))
	}
	if params.StopSequences[0] != "STOP" || params.StopSequences[1] != "END" {
		t.Errorf("StopSequences = %v", params.StopSequences)
	}
}

// TestStreamError_BuildParamsError tests stream with messages that fail convertMessages.
func TestStreamError_BuildParamsError(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "claude-sonnet-4-5-20250929", APIKey: "test",
	})
	// Use the unsupported message type defined above.
	msgs := []schema.Message{
		&unsupportedTestMessage{},
	}

	var gotErr error
	for _, err := range m.Stream(context.Background(), msgs) {
		gotErr = err
		break
	}
	if gotErr == nil {
		t.Fatal("expected error from invalid message")
	}
	if !strings.Contains(gotErr.Error(), "unsupported message type") {
		t.Errorf("error = %q", gotErr.Error())
	}
}

// TestConvertStreamEvent_ToolUseStart tests content_block_start with tool_use type.
func TestConvertStreamEvent_ToolUseStart(t *testing.T) {
	event := anthropicSDK.MessageStreamEventUnion{
		Type: "content_block_start",
		ContentBlock: anthropicSDK.ContentBlockStartEventContentBlockUnion{
			Type: "tool_use",
			ID:   "toolu_abc",
			Name: "search",
		},
	}
	chunk := convertStreamEvent(event, "claude-sonnet-4-5-20250929")
	if chunk == nil {
		t.Fatal("expected non-nil chunk")
	}
	if len(chunk.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(chunk.ToolCalls))
	}
	if chunk.ToolCalls[0].ID != "toolu_abc" {
		t.Errorf("ID = %q, want toolu_abc", chunk.ToolCalls[0].ID)
	}
	if chunk.ToolCalls[0].Name != "search" {
		t.Errorf("Name = %q, want search", chunk.ToolCalls[0].Name)
	}
}

// TestConvertStreamEvent_InputJsonDelta tests content_block_delta with input_json_delta type.
func TestConvertStreamEvent_InputJsonDelta(t *testing.T) {
	event := anthropicSDK.MessageStreamEventUnion{
		Type: "content_block_delta",
		Delta: anthropicSDK.MessageStreamEventUnionDelta{
			Type:        "input_json_delta",
			PartialJSON: `{"city":"SF"}`,
		},
	}
	chunk := convertStreamEvent(event, "claude-sonnet-4-5-20250929")
	if chunk == nil {
		t.Fatal("expected non-nil chunk")
	}
	if len(chunk.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(chunk.ToolCalls))
	}
	if chunk.ToolCalls[0].Arguments != `{"city":"SF"}` {
		t.Errorf("Arguments = %q", chunk.ToolCalls[0].Arguments)
	}
}

// TestConvertStreamEvent_MessageDelta tests message_delta event.
func TestConvertStreamEvent_MessageDelta(t *testing.T) {
	event := anthropicSDK.MessageStreamEventUnion{
		Type: "message_delta",
		Delta: anthropicSDK.MessageStreamEventUnionDelta{
			StopReason: "end_turn",
		},
		Usage: anthropicSDK.MessageDeltaUsage{
			OutputTokens: 42,
		},
	}
	chunk := convertStreamEvent(event, "claude-sonnet-4-5-20250929")
	if chunk == nil {
		t.Fatal("expected non-nil chunk")
	}
	if chunk.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", chunk.FinishReason)
	}
	if chunk.Usage == nil || chunk.Usage.OutputTokens != 42 {
		t.Errorf("Usage.OutputTokens = %v, want 42", chunk.Usage)
	}
}

// TestConvertStreamEvent_UnknownType tests unknown event type returns nil.
func TestConvertStreamEvent_UnknownType(t *testing.T) {
	event := anthropicSDK.MessageStreamEventUnion{
		Type: "unknown_event_type",
	}
	chunk := convertStreamEvent(event, "claude-sonnet-4-5-20250929")
	if chunk != nil {
		t.Errorf("expected nil chunk for unknown type, got %v", chunk)
	}
}

// TestConvertResponse_Nil tests nil response.
func TestConvertResponse_Nil(t *testing.T) {
	ai := convertResponse(nil)
	if ai == nil {
		t.Fatal("expected non-nil AIMessage")
	}
	if len(ai.Parts) != 0 {
		t.Errorf("expected empty Parts, got %d", len(ai.Parts))
	}
	if len(ai.ToolCalls) != 0 {
		t.Errorf("expected empty ToolCalls, got %d", len(ai.ToolCalls))
	}
}

// TestGenerateWithToolMessage tests full end-to-end with tool message in conversation.
func TestGenerateWithToolMessage(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)

		// Verify we received messages including the tool result.
		msgs, ok := req["messages"].([]any)
		if !ok || len(msgs) < 2 {
			t.Error("expected multiple messages in request")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockAnthropicResponse("The weather is sunny and 42 degrees."))
	})
	defer ts.Close()

	// Simulate conversation: Human -> AI with tool call -> Tool result -> AI response.
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("What's the weather in NYC?"),
		&schema.AIMessage{
			Parts: []schema.ContentPart{
				schema.TextPart{Text: "I'll check the weather."},
			},
			ToolCalls: []schema.ToolCall{
				{ID: "toolu_01", Name: "get_weather", Arguments: `{"city":"NYC"}`},
			},
		},
		schema.NewToolMessage("toolu_01", "42 degrees, sunny"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "The weather is sunny and 42 degrees." {
		t.Errorf("text = %q", resp.Text())
	}
}
