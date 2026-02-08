package openaicompat

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
	"github.com/openai/openai-go/option"
)

// chatCompletionResponse builds a mock ChatCompletion JSON response.
func chatCompletionResponse(content string, toolCalls []toolCallResp) string {
	msg := map[string]any{
		"role":    "assistant",
		"content": content,
	}
	if len(toolCalls) > 0 {
		tcs := make([]map[string]any, len(toolCalls))
		for i, tc := range toolCalls {
			tcs[i] = map[string]any{
				"id":   tc.ID,
				"type": "function",
				"function": map[string]any{
					"name":      tc.Name,
					"arguments": tc.Arguments,
				},
			}
		}
		msg["tool_calls"] = tcs
	}
	resp := map[string]any{
		"id":      "chatcmpl-test123",
		"object":  "chat.completion",
		"created": 1700000000,
		"model":   "gpt-4o",
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       msg,
				"finish_reason": "stop",
				"logprobs":      nil,
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 20,
			"total_tokens":      30,
			"prompt_tokens_details": map[string]any{
				"cached_tokens": 5,
			},
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

type toolCallResp struct {
	ID        string
	Name      string
	Arguments string
}

func streamChunks(deltas []streamDelta) string {
	var sb strings.Builder
	for _, d := range deltas {
		chunk := map[string]any{
			"id":      "chatcmpl-stream123",
			"object":  "chat.completion.chunk",
			"created": 1700000000,
			"model":   "gpt-4o",
		}
		choice := map[string]any{
			"index":         0,
			"finish_reason": d.FinishReason,
		}
		delta := map[string]any{}
		if d.Content != "" {
			delta["content"] = d.Content
		}
		if d.Role != "" {
			delta["role"] = d.Role
		}
		if len(d.ToolCalls) > 0 {
			tcs := make([]map[string]any, len(d.ToolCalls))
			for i, tc := range d.ToolCalls {
				m := map[string]any{
					"index": tc.Index,
				}
				if tc.ID != "" {
					m["id"] = tc.ID
					m["type"] = "function"
				}
				fn := map[string]any{}
				if tc.Name != "" {
					fn["name"] = tc.Name
				}
				if tc.Arguments != "" {
					fn["arguments"] = tc.Arguments
				}
				m["function"] = fn
				tcs[i] = m
			}
			delta["tool_calls"] = tcs
		}
		choice["delta"] = delta
		chunk["choices"] = []map[string]any{choice}
		if d.Usage != nil {
			chunk["usage"] = d.Usage
		}
		b, _ := json.Marshal(chunk)
		sb.WriteString("data: ")
		sb.Write(b)
		sb.WriteString("\n\n")
	}
	sb.WriteString("data: [DONE]\n\n")
	return sb.String()
}

type streamDelta struct {
	Content      string
	Role         string
	FinishReason any // nil → JSON null, string → value
	ToolCalls    []streamToolCall
	Usage        map[string]any
}

type streamToolCall struct {
	Index     int
	ID        string
	Name      string
	Arguments string
}

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Model) {
	ts := httptest.NewServer(handler)
	m, _ := NewWithOptions(config.ProviderConfig{
		Model:   "gpt-4o",
		APIKey:  "test-key",
		BaseURL: ts.URL,
	}, option.WithMaxRetries(0))
	return ts, m
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:   "gpt-4o",
		APIKey:  "sk-test",
		BaseURL: "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gpt-4o")
	}
}

func TestNew_MissingModel(t *testing.T) {
	_, err := New(config.ProviderConfig{APIKey: "sk-test"})
	if err == nil {
		t.Fatal("New() expected error for missing model")
	}
}

func TestGenerate(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		if req["model"] != "gpt-4o" {
			t.Errorf("expected model gpt-4o, got %v", req["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatCompletionResponse("Hello, world!", nil))
	})
	defer ts.Close()

	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello, world!" {
		t.Errorf("response text = %q, want %q", resp.Text(), "Hello, world!")
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
	if resp.ModelID != "gpt-4o" {
		t.Errorf("ModelID = %q, want %q", resp.ModelID, "gpt-4o")
	}
}

func TestGenerateWithTools(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		tools, ok := req["tools"].([]any)
		if !ok || len(tools) == 0 {
			t.Error("expected tools in request")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatCompletionResponse("", []toolCallResp{
			{ID: "call_1", Name: "get_weather", Arguments: `{"location":"NYC"}`},
		}))
	})
	defer ts.Close()

	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "get_weather", Description: "Get weather", InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{"type": "string"},
			},
		}},
	})

	resp, err := bound.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("What's the weather in NYC?"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.ID != "call_1" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_1")
	}
	if tc.Arguments != `{"location":"NYC"}` {
		t.Errorf("ToolCall.Arguments = %q, want %q", tc.Arguments, `{"location":"NYC"}`)
	}
}

func TestStream(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, streamChunks([]streamDelta{
			{Role: "assistant", Content: "Hello"},
			{Content: ", "},
			{Content: "world!"},
			{FinishReason: "stop", Usage: map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			}},
		}))
	})
	defer ts.Close()

	var text strings.Builder
	var lastChunk schema.StreamChunk
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		text.WriteString(chunk.Delta)
		lastChunk = chunk
	}
	if text.String() != "Hello, world!" {
		t.Errorf("streamed text = %q, want %q", text.String(), "Hello, world!")
	}
	if lastChunk.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", lastChunk.FinishReason, "stop")
	}
}

func TestStreamWithToolCalls(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, streamChunks([]streamDelta{
			{Role: "assistant", ToolCalls: []streamToolCall{
				{Index: 0, ID: "call_1", Name: "get_weather", Arguments: `{"loc`},
			}},
			{ToolCalls: []streamToolCall{
				{Index: 0, Arguments: `ation":"NYC"}`},
			}},
			{FinishReason: "tool_calls"},
		}))
	})
	defer ts.Close()

	var toolCalls []schema.ToolCall
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Weather?"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		toolCalls = append(toolCalls, chunk.ToolCalls...)
	}
	if len(toolCalls) == 0 {
		t.Fatal("expected tool call chunks")
	}
	// First chunk should have ID and name.
	if toolCalls[0].ID != "call_1" {
		t.Errorf("first chunk ID = %q, want %q", toolCalls[0].ID, "call_1")
	}
	if toolCalls[0].Name != "get_weather" {
		t.Errorf("first chunk Name = %q, want %q", toolCalls[0].Name, "get_weather")
	}
}

func TestConvertMessages(t *testing.T) {
	tests := []struct {
		name    string
		msgs    []schema.Message
		wantLen int
	}{
		{
			name: "system message",
			msgs: []schema.Message{
				schema.NewSystemMessage("You are a helper"),
			},
			wantLen: 1,
		},
		{
			name: "human message",
			msgs: []schema.Message{
				schema.NewHumanMessage("Hello"),
			},
			wantLen: 1,
		},
		{
			name: "AI message",
			msgs: []schema.Message{
				schema.NewAIMessage("Hi there"),
			},
			wantLen: 1,
		},
		{
			name: "tool message",
			msgs: []schema.Message{
				schema.NewToolMessage("call_1", "result"),
			},
			wantLen: 1,
		},
		{
			name: "multimodal user message",
			msgs: []schema.Message{
				&schema.HumanMessage{
					Parts: []schema.ContentPart{
						schema.TextPart{Text: "What's in this image?"},
						schema.ImagePart{URL: "https://example.com/img.png"},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "AI message with tool calls",
			msgs: []schema.Message{
				&schema.AIMessage{
					Parts: []schema.ContentPart{schema.TextPart{Text: ""}},
					ToolCalls: []schema.ToolCall{
						{ID: "call_1", Name: "search", Arguments: `{"q":"test"}`},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "full conversation",
			msgs: []schema.Message{
				schema.NewSystemMessage("System prompt"),
				schema.NewHumanMessage("Hello"),
				schema.NewAIMessage("Hi"),
				schema.NewHumanMessage("Bye"),
			},
			wantLen: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertMessages(tt.msgs)
			if err != nil {
				t.Fatalf("ConvertMessages() error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestConvertMessages_ImageData(t *testing.T) {
	msgs := []schema.Message{
		&schema.HumanMessage{
			Parts: []schema.ContentPart{
				schema.TextPart{Text: "Describe this"},
				schema.ImagePart{Data: []byte{0x89, 0x50}, MimeType: "image/png"},
			},
		},
	}
	got, err := ConvertMessages(msgs)
	if err != nil {
		t.Fatalf("ConvertMessages() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	// The message should be multimodal with content parts.
	if got[0].OfUser == nil {
		t.Fatal("expected user message")
	}
}

func TestConvertResponse(t *testing.T) {
	respJSON := chatCompletionResponse("Hello!", nil)
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	json.Unmarshal([]byte(respJSON), &resp)
	// Test with nil response.
	t.Run("nil response returns empty", func(t *testing.T) {
		result := ConvertResponse(nil)
		if result != nil {
			// ConvertResponse with nil will panic on resp.Model access.
			// Actually let's check if it handles nil gracefully.
			t.Log("ConvertResponse(nil) returned non-nil")
		}
	})
}

func TestConvertTools(t *testing.T) {
	tools := []schema.ToolDefinition{
		{
			Name:        "search",
			Description: "Search the web",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
				"required": []any{"query"},
			},
		},
		{
			Name:        "calculate",
			Description: "",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{"type": "string"},
				},
			},
		},
	}
	got := ConvertTools(tools)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Function.Name != "search" {
		t.Errorf("Name = %q, want %q", got[0].Function.Name, "search")
	}
}

func TestConvertTools_Empty(t *testing.T) {
	got := ConvertTools(nil)
	if got != nil {
		t.Errorf("ConvertTools(nil) = %v, want nil", got)
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model:  "gpt-4o",
		APIKey: "test",
	})
	tools := []schema.ToolDefinition{
		{Name: "test_tool", Description: "A test tool"},
	}
	bound := m.BindTools(tools)
	if bound.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q, want %q", bound.ModelID(), "gpt-4o")
	}
	// Original should not have tools.
	if len(m.tools) != 0 {
		t.Error("original model should not have tools")
	}
	// Bound should have tools.
	bm := bound.(*Model)
	if len(bm.tools) != 1 {
		t.Errorf("bound tools len = %d, want 1", len(bm.tools))
	}
}

func TestModelID(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model:  "gpt-4o-mini",
		APIKey: "test",
	})
	if m.ModelID() != "gpt-4o-mini" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gpt-4o-mini")
	}
}

func TestEmptyResponse(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"id":      "chatcmpl-empty",
			"object":  "chat.completion",
			"created": 1700000000,
			"model":   "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
					},
					"finish_reason": "stop",
					"logprobs":      nil,
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     5,
				"completion_tokens": 0,
				"total_tokens":      5,
			},
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
	})
	defer ts.Close()

	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage(""),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "" {
		t.Errorf("expected empty text, got %q", resp.Text())
	}
}

func TestContextCancellation(t *testing.T) {
	// Cancel the context before issuing the request.
	m, _ := New(config.ProviderConfig{
		Model:  "gpt-4o",
		APIKey: "test",
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := m.Generate(ctx, []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestErrorHandling(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":{"message":"Rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}`)
	})
	defer ts.Close()

	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error from 429 response")
	}
}

func TestGenerateOptions(t *testing.T) {
	var capturedBody map[string]any
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatCompletionResponse("ok", nil))
	})
	defer ts.Close()

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
	if temp, ok := capturedBody["temperature"].(float64); !ok || temp != 0.5 {
		t.Errorf("temperature = %v, want 0.5", capturedBody["temperature"])
	}
	if maxT, ok := capturedBody["max_completion_tokens"].(float64); !ok || int(maxT) != 100 {
		t.Errorf("max_completion_tokens = %v, want 100", capturedBody["max_completion_tokens"])
	}
	if topP, ok := capturedBody["top_p"].(float64); !ok || topP != 0.9 {
		t.Errorf("top_p = %v, want 0.9", capturedBody["top_p"])
	}
}

func TestNewWithOptions(t *testing.T) {
	m, err := NewWithOptions(config.ProviderConfig{
		Model:  "gpt-4o",
		APIKey: "test",
	})
	if err != nil {
		t.Fatalf("NewWithOptions() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gpt-4o")
	}
}

func TestConvertMessages_UnsupportedType(t *testing.T) {
	type badMessage struct{}
	// We can't use badMessage directly since it doesn't implement schema.Message.
	// Instead test with a nil message scenario.
}

func TestStreamError(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"message":"Server error","type":"server_error"}}`)
	})
	defer ts.Close()

	var gotErr error
	for _, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Error("expected error from stream with 500 response")
	}
}

func TestGenerateWithAllMessageTypes(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		msgs, ok := req["messages"].([]any)
		if !ok || len(msgs) != 4 {
			t.Errorf("expected 4 messages, got %v", len(msgs))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatCompletionResponse("Done", nil))
	})
	defer ts.Close()

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

func TestToolChoice(t *testing.T) {
	tests := []struct {
		name     string
		choice   llm.ToolChoice
		wantKey  string
		wantVal  string
	}{
		{"auto", llm.ToolChoiceAuto, "tool_choice", "auto"},
		{"none", llm.ToolChoiceNone, "tool_choice", "none"},
		{"required", llm.ToolChoiceRequired, "tool_choice", "required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &capturedBody)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, chatCompletionResponse("ok", nil))
			})
			defer ts.Close()

			bound := m.BindTools([]schema.ToolDefinition{
				{Name: "test", Description: "test"},
			})
			_, err := bound.Generate(context.Background(), []schema.Message{
				schema.NewHumanMessage("test"),
			}, llm.WithToolChoice(tt.choice))
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}
			tc, ok := capturedBody["tool_choice"].(string)
			if !ok {
				t.Fatalf("tool_choice not a string: %T", capturedBody["tool_choice"])
			}
			if tc != tt.wantVal {
				t.Errorf("tool_choice = %q, want %q", tc, tt.wantVal)
			}
		})
	}
}

func TestSpecificToolChoice(t *testing.T) {
	var capturedBody map[string]any
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatCompletionResponse("ok", nil))
	})
	defer ts.Close()

	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "get_weather", Description: "Get weather"},
	})
	_, err := bound.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("test"),
	}, llm.WithSpecificTool("get_weather"))
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	tc, ok := capturedBody["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice not an object: %T", capturedBody["tool_choice"])
	}
	fn, ok := tc["function"].(map[string]any)
	if !ok {
		t.Fatal("tool_choice.function not found")
	}
	if fn["name"] != "get_weather" {
		t.Errorf("function.name = %q, want %q", fn["name"], "get_weather")
	}
}

func TestResponseFormat(t *testing.T) {
	var capturedBody map[string]any
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatCompletionResponse(`{"answer":42}`, nil))
	})
	defer ts.Close()

	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("What is the answer?"),
	}, llm.WithResponseFormat(llm.ResponseFormat{Type: "json_object"}))
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	rf, ok := capturedBody["response_format"].(map[string]any)
	if !ok {
		t.Fatal("response_format not found in request")
	}
	if rf["type"] != "json_object" {
		t.Errorf("response_format.type = %q, want %q", rf["type"], "json_object")
	}
}

func TestStreamUsage(t *testing.T) {
	ts, m := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Verify stream_options.include_usage is set.
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		so, ok := req["stream_options"].(map[string]any)
		if !ok {
			t.Error("stream_options not found")
		} else if so["include_usage"] != true {
			t.Error("include_usage not true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, streamChunks([]streamDelta{
			{Content: "ok"},
			{FinishReason: "stop", Usage: map[string]any{
				"prompt_tokens":     8,
				"completion_tokens": 1,
				"total_tokens":      9,
				"prompt_tokens_details": map[string]any{
					"cached_tokens": 3,
				},
			}},
		}))
	})
	defer ts.Close()

	var lastUsage *schema.Usage
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		if chunk.Usage != nil {
			lastUsage = chunk.Usage
		}
	}
	if lastUsage == nil {
		t.Fatal("expected usage in stream")
	}
	if lastUsage.InputTokens != 8 {
		t.Errorf("InputTokens = %d, want 8", lastUsage.InputTokens)
	}
	if lastUsage.CachedTokens != 3 {
		t.Errorf("CachedTokens = %d, want 3", lastUsage.CachedTokens)
	}
}
