package together

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
		"id": "chatcmpl-tog", "object": "chat.completion",
		"created": 1700000000, "model": "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
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
			"id": "chatcmpl-tog", "object": "chat.completion.chunk",
			"created": 1700000000, "model": "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
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
		"id": "chatcmpl-tog", "object": "chat.completion.chunk",
		"created": 1700000000, "model": "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
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
		if n == "together" {
			found = true
			break
		}
	}
	if !found {
		t.Error("together provider not registered")
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo" {
		t.Errorf("ModelID() = %q, want default", m.ModelID())
	}
}

func TestNew_CustomModel(t *testing.T) {
	m, err := New(config.ProviderConfig{Model: "mistralai/Mixtral-8x7B-Instruct-v0.1", APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "mistralai/Mixtral-8x7B-Instruct-v0.1" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

func TestGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("Hello from Together!"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", APIKey: "test", BaseURL: ts.URL,
	})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Together!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from Together!")
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamResponse([]string{"Together", " AI"}))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", APIKey: "test", BaseURL: ts.URL,
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
	if text.String() != "Together AI" {
		t.Errorf("text = %q, want %q", text.String(), "Together AI")
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", APIKey: "test",
	})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "search", Description: "search"},
	})
	if bound.ModelID() != "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("together", config.ProviderConfig{
		Model: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", APIKey: "test",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestGenerate_ToolCalls(t *testing.T) {
	resp := map[string]any{
		"id": "chatcmpl-tog", "object": "chat.completion",
		"created": 1700000000, "model": "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
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
		Model: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", APIKey: "test", BaseURL: ts.URL,
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
}

func TestGenerate_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", APIKey: "test", BaseURL: ts.URL,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := m.Generate(ctx, []schema.Message{schema.NewHumanMessage("Hi")})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDefaultBaseURL(t *testing.T) {
	m, err := New(config.ProviderConfig{APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}
