package mistral

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func chatResponse(content string) string {
	resp := map[string]any{
		"id": "cmpl-mistral", "object": "chat.completion",
		"created": 1700000000, "model": "mistral-large-latest",
		"choices": []map[string]any{{
			"index":         0,
			"message":       map[string]any{"role": "assistant", "content": content},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{
			"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15,
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func toolCallResponse() string {
	resp := map[string]any{
		"id": "cmpl-mistral", "object": "chat.completion",
		"created": 1700000000, "model": "mistral-large-latest",
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role": "assistant", "content": "",
				"tool_calls": []map[string]any{{
					"id":       "call_abc",
					"type":     "function",
					"function": map[string]any{"name": "search", "arguments": `{"q":"test"}`},
				}},
			},
			"finish_reason": "tool_calls",
		}},
		"usage": map[string]any{
			"prompt_tokens": 10, "completion_tokens": 8, "total_tokens": 18,
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func streamResponse(deltas []string) string {
	var sb strings.Builder
	for _, d := range deltas {
		chunk := map[string]any{
			"id": "cmpl-ms", "model": "mistral-large-latest",
			"choices": []map[string]any{{
				"index": 0, "delta": map[string]any{"role": "assistant", "content": d},
				"finish_reason": nil,
			}},
		}
		b, _ := json.Marshal(chunk)
		sb.WriteString("data: ")
		sb.Write(b)
		sb.WriteString("\n\n")
	}
	final := map[string]any{
		"id": "cmpl-ms", "model": "mistral-large-latest",
		"choices": []map[string]any{{
			"index": 0, "delta": map[string]any{}, "finish_reason": "stop",
		}},
		"usage": map[string]any{
			"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15,
		},
	}
	b, _ := json.Marshal(final)
	sb.WriteString("data: ")
	sb.Write(b)
	sb.WriteString("\n\n")
	sb.WriteString("data: [DONE]\n\n")
	return sb.String()
}

func TestRegistration(t *testing.T) {
	names := llm.List()
	found := false
	for _, n := range names {
		if n == "mistral" {
			found = true
			break
		}
	}
	if !found {
		t.Error("mistral provider not registered")
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "mistral-large-latest" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "mistral-large-latest")
	}
}

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New(config.ProviderConfig{})
	if err == nil {
		t.Fatal("expected error for missing api_key")
	}
}

func TestNew_CustomModel(t *testing.T) {
	m, err := New(config.ProviderConfig{Model: "mistral-small-latest", APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "mistral-small-latest" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

func TestGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatResponse("Bonjour from Mistral!"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Bonjour from Mistral!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Bonjour from Mistral!")
	}
}

func TestGenerate_Usage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatResponse("test"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("OutputTokens = %d, want 5", resp.Usage.OutputTokens)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestGenerate_ToolCalls(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, toolCallResponse())
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("search for test"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "search" {
		t.Errorf("ToolCall name = %q, want %q", resp.ToolCalls[0].Name, "search")
	}
	if resp.ToolCalls[0].ID != "call_abc" {
		t.Errorf("ToolCall ID = %q, want %q", resp.ToolCalls[0].ID, "call_abc")
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamResponse([]string{"Mistral", " says", " hi"}))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	var text strings.Builder
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		text.WriteString(chunk.Delta)
	}
	if text.String() != "Mistral says hi" {
		t.Errorf("text = %q, want %q", text.String(), "Mistral says hi")
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test",
	})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "search", Description: "search the web"},
	})
	if bound.ModelID() != "mistral-large-latest" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
	// Verify original is not modified
	if len(m.tools) != 0 {
		t.Errorf("original tools modified: len = %d", len(m.tools))
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("mistral", config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "mistral-large-latest" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestConvertMessages(t *testing.T) {
	msgs := []schema.Message{
		schema.NewSystemMessage("You are helpful"),
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi there"),
		schema.NewToolMessage("call_1", "result"),
	}
	converted := convertMessages(msgs)
	if len(converted) != 4 {
		t.Fatalf("len = %d, want 4", len(converted))
	}
	if converted[0].Role != "system" {
		t.Errorf("msg[0] role = %q, want system", converted[0].Role)
	}
	if converted[1].Role != "user" {
		t.Errorf("msg[1] role = %q, want user", converted[1].Role)
	}
	if converted[2].Role != "assistant" {
		t.Errorf("msg[2] role = %q, want assistant", converted[2].Role)
	}
	if converted[3].Role != "tool" {
		t.Errorf("msg[3] role = %q, want tool", converted[3].Role)
	}
}

func TestGenerate_WithOptions(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatResponse("ok"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}, llm.WithTemperature(0.5), llm.WithMaxTokens(100))
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if temp, ok := receivedBody["temperature"].(float64); ok {
		if temp != 0.5 {
			t.Errorf("temperature = %v, want 0.5", temp)
		}
	}
	if maxTok, ok := receivedBody["max_tokens"].(float64); ok {
		if int(maxTok) != 100 {
			t.Errorf("max_tokens = %v, want 100", maxTok)
		}
	}
}

func TestGenerate_ErrorHandling(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"message": "internal error"}`)
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
		Timeout: 5 * time.Second,
	})
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
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
			},
		},
	}
	converted := convertTools(tools)
	if len(converted) != 1 {
		t.Fatalf("len = %d, want 1", len(converted))
	}
	if converted[0].Function.Name != "search" {
		t.Errorf("name = %q, want search", converted[0].Function.Name)
	}
}

func TestConvertResponse_Nil(t *testing.T) {
	resp := convertResponse(nil)
	if resp == nil {
		t.Fatal("expected non-nil AIMessage for nil response")
	}
	if len(resp.Parts) != 0 {
		t.Errorf("expected no parts, got %d", len(resp.Parts))
	}
}

func TestConvertMessages_AIWithToolCalls(t *testing.T) {
	msgs := []schema.Message{
		&schema.AIMessage{
			Parts: []schema.ContentPart{schema.TextPart{Text: "I'll search for that"}},
			ToolCalls: []schema.ToolCall{
				{ID: "call_123", Name: "search", Arguments: `{"q":"test"}`},
			},
		},
	}
	converted := convertMessages(msgs)
	if len(converted) != 1 {
		t.Fatalf("len = %d, want 1", len(converted))
	}
	if converted[0].Role != "assistant" {
		t.Errorf("role = %q, want assistant", converted[0].Role)
	}
	if len(converted[0].ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(converted[0].ToolCalls))
	}
	if converted[0].ToolCalls[0].Id != "call_123" {
		t.Errorf("ToolCall ID = %q, want call_123", converted[0].ToolCalls[0].Id)
	}
	if converted[0].ToolCalls[0].Function.Name != "search" {
		t.Errorf("ToolCall name = %q, want search", converted[0].ToolCalls[0].Function.Name)
	}
}

func TestBuildParams_TopP(t *testing.T) {
	m, _ := New(config.ProviderConfig{APIKey: "test"})
	topP := 0.9
	params := m.buildParams([]llm.GenerateOption{llm.WithTopP(topP)})
	if params.TopP != 0.9 {
		t.Errorf("TopP = %v, want 0.9", params.TopP)
	}
}

func TestBuildParams_ToolChoiceAuto(t *testing.T) {
	m, _ := New(config.ProviderConfig{APIKey: "test"})
	params := m.buildParams([]llm.GenerateOption{llm.WithToolChoice(llm.ToolChoiceAuto)})
	if params.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %q, want auto", params.ToolChoice)
	}
}

func TestBuildParams_ToolChoiceNone(t *testing.T) {
	m, _ := New(config.ProviderConfig{APIKey: "test"})
	params := m.buildParams([]llm.GenerateOption{llm.WithToolChoice(llm.ToolChoiceNone)})
	if params.ToolChoice != "none" {
		t.Errorf("ToolChoice = %q, want none", params.ToolChoice)
	}
}

func TestBuildParams_ToolChoiceRequired(t *testing.T) {
	m, _ := New(config.ProviderConfig{APIKey: "test"})
	params := m.buildParams([]llm.GenerateOption{llm.WithToolChoice(llm.ToolChoiceRequired)})
	if params.ToolChoice != "any" {
		t.Errorf("ToolChoice = %q, want any", params.ToolChoice)
	}
}

func TestBuildParams_WithTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{APIKey: "test"})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "search", Description: "search"},
	}).(*Model)
	params := bound.buildParams(nil)
	if len(params.Tools) != 1 {
		t.Errorf("Tools len = %d, want 1", len(params.Tools))
	}
	if params.Tools[0].Function.Name != "search" {
		t.Errorf("Tool name = %q, want search", params.Tools[0].Function.Name)
	}
}

func TestBuildParams_JSONFormat(t *testing.T) {
	m, _ := New(config.ProviderConfig{APIKey: "test"})
	params := m.buildParams([]llm.GenerateOption{
		llm.WithResponseFormat(llm.ResponseFormat{Type: "json_object"}),
	})
	if params.ResponseFormat != "json_object" {
		t.Errorf("ResponseFormat = %q, want json_object", params.ResponseFormat)
	}
}

func streamToolCallResponse() string {
	var sb strings.Builder
	// First chunk with tool call
	chunk1 := map[string]any{
		"id": "cmpl-ms", "model": "mistral-large-latest",
		"choices": []map[string]any{{
			"index": 0,
			"delta": map[string]any{
				"role": "assistant",
				"tool_calls": []map[string]any{{
					"id":   "call_xyz",
					"type": "function",
					"function": map[string]any{
						"name":      "search",
						"arguments": `{"q":"test"}`,
					},
				}},
			},
			"finish_reason": nil,
		}},
	}
	b1, _ := json.Marshal(chunk1)
	sb.WriteString("data: ")
	sb.Write(b1)
	sb.WriteString("\n\n")
	// Final chunk
	final := map[string]any{
		"id": "cmpl-ms", "model": "mistral-large-latest",
		"choices": []map[string]any{{
			"index": 0, "delta": map[string]any{}, "finish_reason": "tool_calls",
		}},
	}
	b2, _ := json.Marshal(final)
	sb.WriteString("data: ")
	sb.Write(b2)
	sb.WriteString("\n\n")
	sb.WriteString("data: [DONE]\n\n")
	return sb.String()
}

func streamWithUsageResponse() string {
	var sb strings.Builder
	chunk := map[string]any{
		"id": "cmpl-ms", "model": "mistral-large-latest",
		"choices": []map[string]any{{
			"index": 0, "delta": map[string]any{"content": "test"},
			"finish_reason": nil,
		}},
		"usage": map[string]any{
			"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30,
		},
	}
	b, _ := json.Marshal(chunk)
	sb.WriteString("data: ")
	sb.Write(b)
	sb.WriteString("\n\n")
	sb.WriteString("data: [DONE]\n\n")
	return sb.String()
}

func TestStream_WithToolCalls(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamToolCallResponse())
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	var toolCalls []schema.ToolCall
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("search for test"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}
	}
	if len(toolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(toolCalls))
	}
	if toolCalls[0].Name != "search" {
		t.Errorf("ToolCall name = %q, want search", toolCalls[0].Name)
	}
	if toolCalls[0].ID != "call_xyz" {
		t.Errorf("ToolCall ID = %q, want call_xyz", toolCalls[0].ID)
	}
}

func TestStream_WithUsage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamWithUsageResponse())
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	var usage *schema.Usage
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}
	if usage == nil {
		t.Fatal("expected usage data")
	}
	if usage.InputTokens != 20 {
		t.Errorf("InputTokens = %d, want 20", usage.InputTokens)
	}
	if usage.OutputTokens != 10 {
		t.Errorf("OutputTokens = %d, want 10", usage.OutputTokens)
	}
	if usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d, want 30", usage.TotalTokens)
	}
}

func TestStream_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Send infinite stream
		for i := 0; i < 1000; i++ {
			chunk := map[string]any{
				"id": "cmpl-ms", "model": "mistral-large-latest",
				"choices": []map[string]any{{
					"index": 0, "delta": map[string]any{"content": "test "},
				}},
			}
			b, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", b)
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	count := 0
	for _, err := range m.Stream(ctx, []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		count++
		if count == 2 {
			cancel()
		}
		if err != nil {
			if err != context.Canceled {
				t.Errorf("expected context.Canceled, got %v", err)
			}
			break
		}
	}
	if count > 10 {
		t.Errorf("stream continued after cancel: count = %d", count)
	}
}

func TestNew_CustomEndpointAndTimeout(t *testing.T) {
	m, err := New(config.ProviderConfig{
		APIKey:  "test",
		BaseURL: "https://custom.mistral.ai",
		Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.client == nil {
		t.Error("client is nil")
	}
}

func TestStream_ChunkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Send an error chunk
		chunk := map[string]any{
			"error": map[string]any{
				"message": "rate limit exceeded",
				"type":    "rate_limit_error",
			},
		}
		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", b)
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "mistral-large-latest", APIKey: "test", BaseURL: ts.URL,
	})
	var gotError bool
	for _, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			gotError = true
			break
		}
	}
	if !gotError {
		t.Error("expected error from chunk with error field")
	}
}
