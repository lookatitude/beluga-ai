package llama

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

	// Import backend providers so llm.New("together", ...) etc. work in tests.
	_ "github.com/lookatitude/beluga-ai/llm/providers/fireworks"
	_ "github.com/lookatitude/beluga-ai/llm/providers/together"
)

func mockResponse(content string) string {
	resp := map[string]any{
		"id": "chatcmpl-llama", "object": "chat.completion",
		"created": 1700000000, "model": "llama3.1",
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
			"id": "chatcmpl-llama", "object": "chat.completion.chunk",
			"created": 1700000000, "model": "llama3.1",
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
		"id": "chatcmpl-llama", "object": "chat.completion.chunk",
		"created": 1700000000, "model": "llama3.1",
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
		if n == "llama" {
			found = true
			break
		}
	}
	if !found {
		t.Error("llama provider not registered")
	}
}

func TestNew_MissingModel(t *testing.T) {
	_, err := New(config.ProviderConfig{APIKey: "test"})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestNew_TogetherBackend(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:   "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
		APIKey:  "test",
		Options: map[string]any{"backend": "together"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

func TestNew_FireworksBackend(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:   "accounts/fireworks/models/llama-v3p1-70b-instruct",
		APIKey:  "test",
		Options: map[string]any{"backend": "fireworks"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "accounts/fireworks/models/llama-v3p1-70b-instruct" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

func TestNew_CustomBaseURL(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:   "llama3.1",
		APIKey:  "test",
		BaseURL: "http://custom:8080/v1",
		Options: map[string]any{"backend": "together"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "llama3.1" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

func TestNew_UnsupportedBackend(t *testing.T) {
	_, err := New(config.ProviderConfig{
		Model:   "test-model",
		APIKey:  "test",
		Options: map[string]any{"backend": "unknown"},
	})
	if err == nil {
		t.Fatal("expected error for unsupported backend")
	}
}

func TestGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("Hello from Llama!"))
	}))
	defer ts.Close()

	m, err := New(config.ProviderConfig{
		Model:   "llama3.1",
		APIKey:  "test",
		BaseURL: ts.URL,
		Options: map[string]any{"backend": "together"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Llama!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from Llama!")
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamResponse([]string{"Lla", "ma"}))
	}))
	defer ts.Close()

	m, err := New(config.ProviderConfig{
		Model:   "llama3.1",
		APIKey:  "test",
		BaseURL: ts.URL,
		Options: map[string]any{"backend": "together"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	var text strings.Builder
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		text.WriteString(chunk.Delta)
	}
	if text.String() != "Llama" {
		t.Errorf("text = %q, want %q", text.String(), "Llama")
	}
}

func TestBindTools(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:   "llama3.1",
		APIKey:  "test",
		Options: map[string]any{"backend": "together"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "search", Description: "search"},
	})
	if bound.ModelID() != "llama3.1" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("llama", config.ProviderConfig{
		Model:   "llama3.1",
		APIKey:  "test",
		Options: map[string]any{"backend": "together"},
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "llama3.1" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestGenerate_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer ts.Close()

	m, err := New(config.ProviderConfig{
		Model:   "llama3.1",
		APIKey:  "test",
		BaseURL: ts.URL,
		Options: map[string]any{"backend": "together"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = m.Generate(ctx, []schema.Message{schema.NewHumanMessage("Hi")})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDefaultBackend(t *testing.T) {
	if defaultBackend != "together" {
		t.Errorf("defaultBackend = %q, want %q", defaultBackend, "together")
	}
}

func TestSupportedBackends(t *testing.T) {
	expected := []string{"together", "fireworks", "groq", "sambanova", "cerebras", "ollama"}
	for _, b := range expected {
		if _, ok := backends[b]; !ok {
			t.Errorf("backend %q not in backends map", b)
		}
	}
}
