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

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

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
