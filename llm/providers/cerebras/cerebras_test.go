package cerebras

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func mockResponse(content string) string {
	resp := map[string]any{
		"id": "chatcmpl-cb", "object": "chat.completion",
		"created": 1700000000, "model": "llama-3.3-70b",
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

func streamResponse(deltas []string) string {
	var sb strings.Builder
	for _, d := range deltas {
		chunk := map[string]any{
			"id": "chatcmpl-cb", "object": "chat.completion.chunk",
			"created": 1700000000, "model": "llama-3.3-70b",
			"choices": []map[string]any{{
				"index": 0, "delta": map[string]any{"content": d}, "finish_reason": nil,
			}},
		}
		b, _ := json.Marshal(chunk)
		sb.WriteString("data: ")
		sb.Write(b)
		sb.WriteString("\n\n")
	}
	final := map[string]any{
		"id": "chatcmpl-cb", "object": "chat.completion.chunk",
		"created": 1700000000, "model": "llama-3.3-70b",
		"choices": []map[string]any{{
			"index": 0, "delta": map[string]any{}, "finish_reason": "stop",
		}},
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
		if n == "cerebras" {
			found = true
			break
		}
	}
	if !found {
		t.Error("cerebras provider not registered")
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{Model: "llama-3.3-70b", APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "llama-3.3-70b" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "llama-3.3-70b")
	}
}

func TestNew_MissingModel(t *testing.T) {
	_, err := New(config.ProviderConfig{APIKey: "test"})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("Hello from Cerebras!"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "llama-3.3-70b", APIKey: "test", BaseURL: ts.URL,
	})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Cerebras!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from Cerebras!")
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamResponse([]string{"Cere", "bras"}))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "llama-3.3-70b", APIKey: "test", BaseURL: ts.URL,
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
	if text.String() != "Cerebras" {
		t.Errorf("text = %q, want %q", text.String(), "Cerebras")
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{Model: "llama-3.3-70b", APIKey: "test"})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "search", Description: "search"},
	})
	if bound.ModelID() != "llama-3.3-70b" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("cerebras", config.ProviderConfig{
		Model: "llama-3.3-70b", APIKey: "test",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "llama-3.3-70b" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestGenerate_ToolCalls(t *testing.T) {
	resp := map[string]any{
		"id": "chatcmpl-cb", "object": "chat.completion",
		"created": 1700000000, "model": "llama-3.3-70b",
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role": "assistant", "content": "",
				"tool_calls": []map[string]any{{
					"id":       "call_1",
					"type":     "function",
					"function": map[string]any{"name": "search", "arguments": `{"q":"test"}`},
				}},
			},
			"finish_reason": "tool_calls",
		}},
		"usage": map[string]any{
			"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15,
		},
	}
	b, _ := json.Marshal(resp)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "llama-3.3-70b", APIKey: "test", BaseURL: ts.URL,
	})
	ai, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("search"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(ai.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(ai.ToolCalls))
	}
	if ai.ToolCalls[0].Name != "search" {
		t.Errorf("ToolCall name = %q, want %q", ai.ToolCalls[0].Name, "search")
	}
}

func TestGenerate_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "llama-3.3-70b", APIKey: "test", BaseURL: ts.URL,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := m.Generate(ctx, []schema.Message{schema.NewHumanMessage("Hi")})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDefaultBaseURL(t *testing.T) {
	if defaultBaseURL != "https://api.cerebras.ai/v1" {
		t.Errorf("defaultBaseURL = %q", defaultBaseURL)
	}
}
