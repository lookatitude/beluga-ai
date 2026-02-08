package litellm

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
		"id":      "chatcmpl-litellm",
		"object":  "chat.completion",
		"created": 1700000000,
		"model":   "gpt-4o",
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
			"id": "chatcmpl-ls", "object": "chat.completion.chunk",
			"created": 1700000000, "model": "gpt-4o",
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
		"id": "chatcmpl-ls", "object": "chat.completion.chunk",
		"created": 1700000000, "model": "gpt-4o",
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
		if n == "litellm" {
			found = true
			break
		}
	}
	if !found {
		t.Error("litellm provider not registered")
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:  "gpt-4o",
		APIKey: "sk-test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gpt-4o")
	}
}

func TestNew_DefaultModel(t *testing.T) {
	m, err := New(config.ProviderConfig{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gpt-4o")
	}
}

func TestGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("Hello from LiteLLM!"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test", BaseURL: ts.URL,
	})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from LiteLLM!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from LiteLLM!")
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamResponse([]string{"Proxied", " response"}))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test", BaseURL: ts.URL,
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
	if text.String() != "Proxied response" {
		t.Errorf("text = %q, want %q", text.String(), "Proxied response")
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("litellm", config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test",
	})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "search", Description: "search the web"},
	})
	if bound.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
}

func TestDefaultBaseURL(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:  "gpt-4o",
		APIKey: "test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}
