package azure

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
		"id": "chatcmpl-azure", "object": "chat.completion",
		"created": 1700000000, "model": "gpt-4o",
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
			"id": "chatcmpl-azure", "object": "chat.completion.chunk",
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
		"id": "chatcmpl-azure", "object": "chat.completion.chunk",
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
		if n == "azure" {
			found = true
			break
		}
	}
	if !found {
		t.Error("azure provider not registered")
	}
}

func TestNew_MissingBaseURL(t *testing.T) {
	_, err := New(config.ProviderConfig{APIKey: "test", Model: "gpt-4o"})
	if err == nil {
		t.Fatal("expected error for missing base_url")
	}
	if !strings.Contains(err.Error(), "base_url is required") {
		t.Errorf("error = %q, want base_url required message", err.Error())
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:   "gpt-4o",
		APIKey:  "test-key",
		BaseURL: "https://myresource.openai.azure.com/openai/deployments/my-gpt4o",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gpt-4o")
	}
}

func TestNew_DefaultModel(t *testing.T) {
	m, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "https://myresource.openai.azure.com/openai/deployments/my-gpt4o",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gpt-4o")
	}
}

func TestGenerate_APIKeyHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Azure uses api-key header, not Bearer token
		apiKey := r.Header.Get("api-key")
		if apiKey != "my-azure-key" {
			t.Errorf("api-key header = %q, want %q", apiKey, "my-azure-key")
		}
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("Authorization header should be empty for Azure, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("Hello from Azure!"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "gpt-4o", APIKey: "my-azure-key", BaseURL: ts.URL,
	})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Azure!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from Azure!")
	}
}

func TestGenerate_APIVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := r.URL.Query().Get("api-version")
		if version != "2024-10-21" {
			t.Errorf("api-version = %q, want %q", version, "2024-10-21")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("versioned"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test", BaseURL: ts.URL,
	})
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
}

func TestGenerate_CustomAPIVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := r.URL.Query().Get("api-version")
		if version != "2025-01-01" {
			t.Errorf("api-version = %q, want %q", version, "2025-01-01")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mockResponse("custom version"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test", BaseURL: ts.URL,
		Options: map[string]any{"api_version": "2025-01-01"},
	})
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, streamResponse([]string{"Azure", " rocks"}))
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
	if text.String() != "Azure rocks" {
		t.Errorf("text = %q, want %q", text.String(), "Azure rocks")
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test",
		BaseURL: "https://test.openai.azure.com/openai/deployments/gpt4o",
	})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "search", Description: "search"},
	})
	if bound.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("azure", config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test",
		BaseURL: "https://test.openai.azure.com/openai/deployments/gpt4o",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "gpt-4o" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestGenerate_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "gpt-4o", APIKey: "test", BaseURL: ts.URL,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := m.Generate(ctx, []schema.Message{schema.NewHumanMessage("Hi")})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
