package openai

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

func mockResponse(content string) string {
	resp := map[string]any{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": 1700000000,
		"model":   "gpt-4o",
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
			"id": "chatcmpl-s", "object": "chat.completion.chunk",
			"created": 1700000000, "model": "gpt-4o",
			"choices": []map[string]any{{
				"index":         0,
				"delta":         map[string]any{"content": d},
				"finish_reason": nil,
			}},
		}
		b, _ := json.Marshal(chunk)
		sb.WriteString("data: ")
		sb.Write(b)
		sb.WriteString("\n\n")
	}
	// Final chunk with finish_reason.
	final := map[string]any{
		"id": "chatcmpl-s", "object": "chat.completion.chunk",
		"created": 1700000000, "model": "gpt-4o",
		"choices": []map[string]any{{
			"index":         0,
			"delta":         map[string]any{},
			"finish_reason": "stop",
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
	for _, name := range names {
		if name == "openai" {
			found = true
			break
		}
	}
	if !found {
		t.Error("openai provider not registered")
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

func TestNew_MissingModel(t *testing.T) {
	_, err := New(config.ProviderConfig{APIKey: "sk-test"})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("Hello from OpenAI!"))
	}))
	defer ts.Close()

	m, err := New(config.ProviderConfig{
		Model:   "gpt-4o",
		APIKey:  "test",
		BaseURL: ts.URL,
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
	if resp.Text() != "Hello from OpenAI!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from OpenAI!")
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamResponse([]string{"Hello", " world"}))
	}))
	defer ts.Close()

	m, err := New(config.ProviderConfig{
		Model:   "gpt-4o",
		APIKey:  "test",
		BaseURL: ts.URL,
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
	if text.String() != "Hello world" {
		t.Errorf("text = %q, want %q", text.String(), "Hello world")
	}
}

func TestGenerateWithTools(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		if _, ok := req["tools"]; !ok {
			t.Error("expected tools in request")
		}
		resp := map[string]any{
			"id":      "chatcmpl-tc",
			"object":  "chat.completion",
			"created": 1700000000,
			"model":   "gpt-4o",
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []map[string]any{{
						"id": "call_1", "type": "function",
						"function": map[string]any{
							"name":      "get_weather",
							"arguments": `{"city":"SF"}`,
						},
					}},
				},
				"finish_reason": "tool_calls",
				"logprobs":      nil,
			}},
			"usage": map[string]any{
				"prompt_tokens": 10, "completion_tokens": 10, "total_tokens": 20,
			},
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	defer ts.Close()

	m, err := New(config.ProviderConfig{
		Model:   "gpt-4o",
		APIKey:  "test",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "get_weather", Description: "Get weather info"},
	})
	resp, err := bound.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Weather in SF?"),
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
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model:  "gpt-4o",
		APIKey: "test",
	})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "test", Description: "test"},
	})
	if bound.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q, want %q", bound.ModelID(), "gpt-4o")
	}
}

func TestDefaultBaseURL(t *testing.T) {
	// With empty BaseURL, should default to OpenAI.
	m, err := New(config.ProviderConfig{
		Model:  "gpt-4o",
		APIKey: "test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q, want %q", m.ModelID(), "gpt-4o")
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("openai", config.ProviderConfig{
		Model:  "gpt-4o",
		APIKey: "test",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q, want %q", m.ModelID(), "gpt-4o")
	}
}

func TestErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"Invalid API key","type":"invalid_request_error"}}`)
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model:   "gpt-4o",
		APIKey:  "bad-key",
		BaseURL: ts.URL,
	})
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error from 401 response")
	}
}
