package ollama

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
		"id": "chatcmpl-ollama", "object": "chat.completion",
		"created": 1700000000, "model": "llama3.2",
		"choices": []map[string]any{{
			"index":         0,
			"message":       map[string]any{"role": "assistant", "content": content},
			"finish_reason": "stop",
			"logprobs":      nil,
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
			"id": "chatcmpl-os", "object": "chat.completion.chunk",
			"created": 1700000000, "model": "llama3.2",
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
		"id": "chatcmpl-os", "object": "chat.completion.chunk",
		"created": 1700000000, "model": "llama3.2",
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
	found := false
	for _, n := range llm.List() {
		if n == "ollama" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ollama provider not registered")
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{Model: "llama3.2"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "llama3.2" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

func TestNew_MissingModel(t *testing.T) {
	_, err := New(config.ProviderConfig{})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("Hello from Ollama!"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{Model: "llama3.2", BaseURL: ts.URL})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Ollama!" {
		t.Errorf("text = %q", resp.Text())
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamResponse([]string{"Local", " LLM"}))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{Model: "llama3.2", BaseURL: ts.URL})
	var text strings.Builder
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		text.WriteString(chunk.Delta)
	}
	if text.String() != "Local LLM" {
		t.Errorf("text = %q", text.String())
	}
}

func TestDefaultAPIKey(t *testing.T) {
	m, err := New(config.ProviderConfig{Model: "llama3.2"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "llama3.2" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("ollama", config.ProviderConfig{Model: "llama3.2"})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "llama3.2" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{Model: "llama3.2"})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "test", Description: "test"},
	})
	if bound.ModelID() != "llama3.2" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
}
