package cohere

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

func chatResponse(text string) string {
	resp := map[string]any{
		"text":          text,
		"generation_id": "gen-123",
		"response_id":   "resp-123",
		"finish_reason": "COMPLETE",
		"meta": map[string]any{
			"api_version": map[string]any{"version": "1"},
			"tokens": map[string]any{
				"input_tokens":  10.0,
				"output_tokens": 5.0,
			},
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func toolCallChatResponse() string {
	resp := map[string]any{
		"text": "",
		"tool_calls": []map[string]any{{
			"name":       "search",
			"parameters": map[string]any{"q": "test"},
		}},
		"finish_reason": "COMPLETE",
		"meta": map[string]any{
			"tokens": map[string]any{
				"input_tokens":  10.0,
				"output_tokens": 8.0,
			},
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func streamResponse(texts []string) string {
	var sb strings.Builder
	// stream-start
	start, _ := json.Marshal(map[string]any{
		"event_type":    "stream-start",
		"generation_id": "gen-123",
		"is_finished":   false,
	})
	sb.Write(start)
	sb.WriteString("\n")

	// text-generation events
	for _, text := range texts {
		ev, _ := json.Marshal(map[string]any{
			"event_type":  "text-generation",
			"text":        text,
			"is_finished": false,
		})
		sb.Write(ev)
		sb.WriteString("\n")
	}

	// stream-end
	end, _ := json.Marshal(map[string]any{
		"event_type":    "stream-end",
		"finish_reason": "COMPLETE",
		"response": map[string]any{
			"text":          strings.Join(texts, ""),
			"generation_id": "gen-123",
			"meta": map[string]any{
				"tokens": map[string]any{
					"input_tokens":  10.0,
					"output_tokens": 5.0,
				},
			},
		},
		"is_finished": true,
	})
	sb.Write(end)
	sb.WriteString("\n")
	return sb.String()
}

func TestRegistration(t *testing.T) {
	names := llm.List()
	found := false
	for _, n := range names {
		if n == "cohere" {
			found = true
			break
		}
	}
	if !found {
		t.Error("cohere provider not registered")
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "command-r-plus" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "command-r-plus")
	}
}

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New(config.ProviderConfig{})
	if err == nil {
		t.Fatal("expected error for missing api_key")
	}
}

func TestNew_CustomModel(t *testing.T) {
	m, err := New(config.ProviderConfig{Model: "command-r", APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "command-r" {
		t.Errorf("ModelID() = %q", m.ModelID())
	}
}

func TestGenerate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatResponse("Hello from Cohere!"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "command-r-plus", APIKey: "test", BaseURL: ts.URL,
	})
	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Cohere!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from Cohere!")
	}
}

func TestGenerate_Usage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatResponse("test"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "command-r-plus", APIKey: "test", BaseURL: ts.URL,
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
}

func TestGenerate_ToolCalls(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, toolCallChatResponse())
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "command-r-plus", APIKey: "test", BaseURL: ts.URL,
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
}

func TestGenerate_SystemMessage(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatResponse("ok"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "command-r-plus", APIKey: "test", BaseURL: ts.URL,
	})
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewSystemMessage("You are helpful"),
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if preamble, ok := receivedBody["preamble"].(string); ok {
		if preamble != "You are helpful" {
			t.Errorf("preamble = %q, want %q", preamble, "You are helpful")
		}
	}
}

func TestGenerate_ChatHistory(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatResponse("ok"))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "command-r-plus", APIKey: "test", BaseURL: ts.URL,
	})
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi there!"),
		schema.NewHumanMessage("How are you?"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if msg, ok := receivedBody["message"].(string); ok {
		if msg != "How are you?" {
			t.Errorf("message = %q, want %q", msg, "How are you?")
		}
	}
	if history, ok := receivedBody["chat_history"].([]any); ok {
		if len(history) != 2 {
			t.Errorf("chat_history len = %d, want 2", len(history))
		}
	}
}

func TestStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, streamResponse([]string{"Cohere", " AI"}))
	}))
	defer ts.Close()

	m, _ := New(config.ProviderConfig{
		Model: "command-r-plus", APIKey: "test", BaseURL: ts.URL,
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
	if text.String() != "Cohere AI" {
		t.Errorf("text = %q, want %q", text.String(), "Cohere AI")
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model: "command-r-plus", APIKey: "test",
	})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "search", Description: "search the web"},
	})
	if bound.ModelID() != "command-r-plus" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
	if len(m.tools) != 0 {
		t.Errorf("original tools modified: len = %d", len(m.tools))
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("cohere", config.ProviderConfig{
		Model: "command-r-plus", APIKey: "test",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "command-r-plus" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestSplitMessages(t *testing.T) {
	msgs := []schema.Message{
		schema.NewSystemMessage("You are helpful"),
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi there"),
		schema.NewHumanMessage("How are you?"),
	}
	preamble, history, message := splitMessages(msgs)
	if preamble != "You are helpful" {
		t.Errorf("preamble = %q, want %q", preamble, "You are helpful")
	}
	if len(history) != 2 {
		t.Errorf("history len = %d, want 2", len(history))
	}
	if message != "How are you?" {
		t.Errorf("message = %q, want %q", message, "How are you?")
	}
}

func TestSplitMessages_Empty(t *testing.T) {
	preamble, history, message := splitMessages(nil)
	if preamble != "" || message != "" || history != nil {
		t.Errorf("expected empty results for nil messages")
	}
}

func TestSplitMessages_OnlySystem(t *testing.T) {
	msgs := []schema.Message{
		schema.NewSystemMessage("You are helpful"),
	}
	preamble, history, message := splitMessages(msgs)
	if preamble != "You are helpful" {
		t.Errorf("preamble = %q, want %q", preamble, "You are helpful")
	}
	if history != nil {
		t.Errorf("history should be nil for system-only")
	}
	if message != "" {
		t.Errorf("message should be empty for system-only")
	}
}

func TestConvertToolDefs(t *testing.T) {
	tools := []schema.ToolDefinition{
		{
			Name:        "search",
			Description: "Search the web",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
				},
			},
		},
	}
	converted := convertToolDefs(tools)
	if len(converted) != 1 {
		t.Fatalf("len = %d, want 1", len(converted))
	}
	if converted[0].Name != "search" {
		t.Errorf("name = %q, want search", converted[0].Name)
	}
	if converted[0].ParameterDefinitions == nil {
		t.Fatal("parameter_definitions should not be nil")
	}
	if _, ok := converted[0].ParameterDefinitions["query"]; !ok {
		t.Error("expected 'query' parameter definition")
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
		Model: "command-r-plus", APIKey: "test", BaseURL: ts.URL,
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
		Model: "command-r-plus", APIKey: "test", BaseURL: ts.URL,
	})
	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
