package google

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
	"google.golang.org/genai"
)

// geminiResponse builds a mock Gemini API JSON response.
func geminiResponse(text string, functionCalls []geminiFC) string {
	parts := []map[string]any{}
	if text != "" {
		parts = append(parts, map[string]any{"text": text})
	}
	for _, fc := range functionCalls {
		parts = append(parts, map[string]any{
			"functionCall": map[string]any{
				"name": fc.Name,
				"args": fc.Args,
			},
		})
	}
	resp := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": parts,
					"role":  "model",
				},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     10,
			"candidatesTokenCount": 20,
			"totalTokenCount":      30,
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

type geminiFC struct {
	Name string
	Args map[string]any
}

// geminiStreamResponse builds a mock Gemini streaming response in SSE format.
// The GenAI SDK expects "data: {json}\n\n" per chunk.
func geminiStreamResponse(deltas []string, finishReason string) string {
	var sb strings.Builder
	for _, d := range deltas {
		chunk := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{{"text": d}},
						"role":  "model",
					},
				},
			},
		}
		b, _ := json.Marshal(chunk)
		sb.WriteString("data: ")
		sb.Write(b)
		sb.WriteString("\n\n")
	}
	// Final chunk with finish reason and usage.
	final := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{{"text": ""}},
					"role":  "model",
				},
				"finishReason": finishReason,
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     10,
			"candidatesTokenCount": 5,
			"totalTokenCount":      15,
		},
	}
	b, _ := json.Marshal(final)
	sb.WriteString("data: ")
	sb.Write(b)
	sb.WriteString("\n\n")
	return sb.String()
}

func newTestModel(handler http.HandlerFunc) (*httptest.Server, *Model) {
	ts := httptest.NewServer(handler)
	m, _ := NewWithHTTPClient(config.ProviderConfig{
		Model:   "gemini-2.5-flash",
		APIKey:  "test-key",
		BaseURL: ts.URL,
	}, ts.Client())
	return ts, m
}

func TestRegistration(t *testing.T) {
	names := llm.List()
	found := false
	for _, n := range names {
		if n == "google" {
			found = true
			break
		}
	}
	if !found {
		t.Error("google provider not registered")
	}
}

func TestNew(t *testing.T) {
	m, err := New(config.ProviderConfig{
		Model:  "gemini-2.5-flash",
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if m.ModelID() != "gemini-2.5-flash" {
		t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gemini-2.5-flash")
	}
}

func TestNew_MissingModel(t *testing.T) {
	_, err := New(config.ProviderConfig{APIKey: "test"})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestGenerate(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, geminiResponse("Hello from Gemini!", nil))
	})
	defer ts.Close()

	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "Hello from Gemini!" {
		t.Errorf("text = %q, want %q", resp.Text(), "Hello from Gemini!")
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want 20", resp.Usage.OutputTokens)
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d, want 30", resp.Usage.TotalTokens)
	}
}

func TestGenerateWithSystemMessage(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		// Check system_instruction is present.
		if _, ok := req["systemInstruction"]; !ok {
			t.Error("expected systemInstruction in request")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, geminiResponse("I'm helpful", nil))
	})
	defer ts.Close()

	resp, err := m.Generate(context.Background(), []schema.Message{
		schema.NewSystemMessage("You are helpful"),
		schema.NewHumanMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text() != "I'm helpful" {
		t.Errorf("text = %q", resp.Text())
	}
}

func TestGenerateWithTools(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		if _, ok := req["tools"]; !ok {
			t.Error("expected tools in request")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, geminiResponse("", []geminiFC{
			{Name: "get_weather", Args: map[string]any{"city": "NYC"}},
		}))
	})
	defer ts.Close()

	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "get_weather", Description: "Get weather", InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"city": map[string]any{"type": "string"}},
		}},
	})
	resp, err := bound.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Weather in NYC?"),
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

func TestStream(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, geminiStreamResponse([]string{"Hello", " world"}, "STOP"))
	})
	defer ts.Close()

	var text strings.Builder
	var gotFinish string
	for chunk, err := range m.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		text.WriteString(chunk.Delta)
		if chunk.FinishReason != "" {
			gotFinish = chunk.FinishReason
		}
	}
	if !strings.Contains(text.String(), "Hello") {
		t.Errorf("streamed text = %q, should contain 'Hello'", text.String())
	}
	if gotFinish != "stop" {
		t.Errorf("FinishReason = %q, want %q", gotFinish, "stop")
	}
}

func TestBindTools(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model:  "gemini-2.5-flash",
		APIKey: "test",
	})
	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "test", Description: "test"},
	})
	if bound.ModelID() != "gemini-2.5-flash" {
		t.Errorf("ModelID = %q", bound.ModelID())
	}
	// Original should not have tools.
	if len(m.tools) != 0 {
		t.Error("original should not have tools")
	}
}

func TestModelID(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model:  "gemini-2.5-pro",
		APIKey: "test",
	})
	if m.ModelID() != "gemini-2.5-pro" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestRegistryNew(t *testing.T) {
	m, err := llm.New("google", config.ProviderConfig{
		Model:  "gemini-2.5-flash",
		APIKey: "test",
	})
	if err != nil {
		t.Fatalf("llm.New() error: %v", err)
	}
	if m.ModelID() != "gemini-2.5-flash" {
		t.Errorf("ModelID = %q", m.ModelID())
	}
}

func TestGenerateOptions(t *testing.T) {
	var capturedBody map[string]any
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, geminiResponse("ok", nil))
	})
	defer ts.Close()

	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	},
		llm.WithTemperature(0.5),
		llm.WithMaxTokens(100),
		llm.WithTopP(0.9),
	)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	gc, ok := capturedBody["generationConfig"].(map[string]any)
	if !ok {
		t.Fatal("expected generationConfig in request")
	}
	if temp, ok := gc["temperature"].(float64); !ok || temp != 0.5 {
		t.Errorf("temperature = %v, want 0.5", gc["temperature"])
	}
	if maxT, ok := gc["maxOutputTokens"].(float64); !ok || int(maxT) != 100 {
		t.Errorf("maxOutputTokens = %v, want 100", gc["maxOutputTokens"])
	}
	if topP, ok := gc["topP"].(float64); !ok || topP != 0.9 {
		t.Errorf("topP = %v, want 0.9", gc["topP"])
	}
}

func TestErrorResponse(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"Invalid API key","code":401}}`)
	})
	defer ts.Close()

	_, err := m.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error from 401")
	}
}

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		input genai.FinishReason
		want  string
	}{
		{genai.FinishReasonStop, "stop"},
		{genai.FinishReasonMaxTokens, "length"},
		{genai.FinishReason("OTHER"), "OTHER"},
	}
	for _, tt := range tests {
		got := mapFinishReason(tt.input)
		if got != tt.want {
			t.Errorf("mapFinishReason(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestConvertMessages(t *testing.T) {
	msgs := []schema.Message{
		schema.NewSystemMessage("Be helpful"),
		schema.NewHumanMessage("Hello"),
		schema.NewAIMessage("Hi there"),
		schema.NewToolMessage("call_1", `{"result":"ok"}`),
	}
	contents, system := convertMessages(msgs)
	if system == nil {
		t.Fatal("expected system instruction")
	}
	// System message is extracted, so contents should have 3 messages.
	if len(contents) != 3 {
		t.Errorf("contents len = %d, want 3", len(contents))
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
	got := convertTools(tools)
	if len(got) != 1 {
		t.Fatalf("tools len = %d, want 1", len(got))
	}
	if len(got[0].FunctionDeclarations) != 1 {
		t.Fatalf("declarations len = %d, want 1", len(got[0].FunctionDeclarations))
	}
	if got[0].FunctionDeclarations[0].Name != "search" {
		t.Errorf("name = %q, want %q", got[0].FunctionDeclarations[0].Name, "search")
	}
}

func TestConvertResponseNil(t *testing.T) {
	ai := convertResponse(nil, "test-model")
	if ai.ModelID != "test-model" {
		t.Errorf("ModelID = %q", ai.ModelID)
	}
}

func TestContextCancellation(t *testing.T) {
	m, _ := New(config.ProviderConfig{
		Model:  "gemini-2.5-flash",
		APIKey: "test",
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := m.Generate(ctx, []schema.Message{
		schema.NewHumanMessage("Hi"),
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// --- Additional tests for coverage ---

func TestApplyToolChoice_Auto(t *testing.T) {
	cfg := &genai.GenerateContentConfig{}
	genOpts := llm.GenerateOptions{ToolChoice: llm.ToolChoiceAuto}
	applyToolChoice(cfg, genOpts)

	if cfg.ToolConfig == nil {
		t.Fatal("expected ToolConfig to be set")
	}
	if cfg.ToolConfig.FunctionCallingConfig == nil {
		t.Fatal("expected FunctionCallingConfig to be set")
	}
	if cfg.ToolConfig.FunctionCallingConfig.Mode != genai.FunctionCallingConfigModeAuto {
		t.Errorf("Mode = %v, want FunctionCallingConfigModeAuto", cfg.ToolConfig.FunctionCallingConfig.Mode)
	}
}

func TestApplyToolChoice_None(t *testing.T) {
	cfg := &genai.GenerateContentConfig{}
	genOpts := llm.GenerateOptions{ToolChoice: llm.ToolChoiceNone}
	applyToolChoice(cfg, genOpts)

	if cfg.ToolConfig == nil {
		t.Fatal("expected ToolConfig to be set")
	}
	if cfg.ToolConfig.FunctionCallingConfig == nil {
		t.Fatal("expected FunctionCallingConfig to be set")
	}
	if cfg.ToolConfig.FunctionCallingConfig.Mode != genai.FunctionCallingConfigModeNone {
		t.Errorf("Mode = %v, want FunctionCallingConfigModeNone", cfg.ToolConfig.FunctionCallingConfig.Mode)
	}
}

func TestApplyToolChoice_Required(t *testing.T) {
	cfg := &genai.GenerateContentConfig{}
	genOpts := llm.GenerateOptions{ToolChoice: llm.ToolChoiceRequired}
	applyToolChoice(cfg, genOpts)

	if cfg.ToolConfig == nil {
		t.Fatal("expected ToolConfig to be set")
	}
	if cfg.ToolConfig.FunctionCallingConfig == nil {
		t.Fatal("expected FunctionCallingConfig to be set")
	}
	if cfg.ToolConfig.FunctionCallingConfig.Mode != genai.FunctionCallingConfigModeAny {
		t.Errorf("Mode = %v, want FunctionCallingConfigModeAny", cfg.ToolConfig.FunctionCallingConfig.Mode)
	}
}

func TestApplyToolChoice_SpecificTool(t *testing.T) {
	cfg := &genai.GenerateContentConfig{}
	genOpts := llm.GenerateOptions{
		ToolChoice:   llm.ToolChoiceAuto, // should be overridden by SpecificTool
		SpecificTool: "get_weather",
	}
	applyToolChoice(cfg, genOpts)

	if cfg.ToolConfig == nil {
		t.Fatal("expected ToolConfig to be set")
	}
	if cfg.ToolConfig.FunctionCallingConfig == nil {
		t.Fatal("expected FunctionCallingConfig to be set")
	}
	if cfg.ToolConfig.FunctionCallingConfig.Mode != genai.FunctionCallingConfigModeAny {
		t.Errorf("Mode = %v, want FunctionCallingConfigModeAny", cfg.ToolConfig.FunctionCallingConfig.Mode)
	}
	if len(cfg.ToolConfig.FunctionCallingConfig.AllowedFunctionNames) != 1 {
		t.Fatalf("AllowedFunctionNames len = %d, want 1", len(cfg.ToolConfig.FunctionCallingConfig.AllowedFunctionNames))
	}
	if cfg.ToolConfig.FunctionCallingConfig.AllowedFunctionNames[0] != "get_weather" {
		t.Errorf("AllowedFunctionNames[0] = %q, want %q", cfg.ToolConfig.FunctionCallingConfig.AllowedFunctionNames[0], "get_weather")
	}
}

func TestConvertHumanParts_ImageData(t *testing.T) {
	parts := []schema.ContentPart{
		schema.TextPart{Text: "Look at this"},
		schema.ImagePart{Data: []byte("fake-image-data")}, // no MimeType, should default to png
	}
	result := convertHumanParts(parts)

	if len(result) != 2 {
		t.Fatalf("result len = %d, want 2", len(result))
	}
	if result[0].Text != "Look at this" {
		t.Errorf("result[0].Text = %q, want %q", result[0].Text, "Look at this")
	}
	if result[1].InlineData == nil {
		t.Fatal("expected InlineData to be set")
	}
	if result[1].InlineData.MIMEType != "image/png" {
		t.Errorf("MIMEType = %q, want %q", result[1].InlineData.MIMEType, "image/png")
	}
	if string(result[1].InlineData.Data) != "fake-image-data" {
		t.Errorf("Data = %q, want %q", result[1].InlineData.Data, "fake-image-data")
	}
}

func TestConvertHumanParts_ImageDataWithMime(t *testing.T) {
	parts := []schema.ContentPart{
		schema.ImagePart{Data: []byte("jpeg-data"), MimeType: "image/jpeg"},
	}
	result := convertHumanParts(parts)

	if len(result) != 1 {
		t.Fatalf("result len = %d, want 1", len(result))
	}
	if result[0].InlineData == nil {
		t.Fatal("expected InlineData to be set")
	}
	if result[0].InlineData.MIMEType != "image/jpeg" {
		t.Errorf("MIMEType = %q, want %q", result[0].InlineData.MIMEType, "image/jpeg")
	}
}

func TestConvertHumanParts_ImageURL(t *testing.T) {
	parts := []schema.ContentPart{
		schema.ImagePart{URL: "gs://bucket/image.png"},
	}
	result := convertHumanParts(parts)

	if len(result) != 1 {
		t.Fatalf("result len = %d, want 1", len(result))
	}
	if result[0].FileData == nil {
		t.Fatal("expected FileData to be set")
	}
	if result[0].FileData.FileURI != "gs://bucket/image.png" {
		t.Errorf("FileURI = %q, want %q", result[0].FileData.FileURI, "gs://bucket/image.png")
	}
	if result[0].FileData.MIMEType != "image/png" {
		t.Errorf("MIMEType = %q, want %q", result[0].FileData.MIMEType, "image/png")
	}
}

func TestConvertAIParts_WithToolCalls(t *testing.T) {
	msg := &schema.AIMessage{
		Parts: []schema.ContentPart{schema.TextPart{Text: "Let me check that"}},
		ToolCalls: []schema.ToolCall{
			{
				ID:        "call_1",
				Name:      "search",
				Arguments: `{"query":"test"}`,
			},
			{
				ID:        "call_2",
				Name:      "calculator",
				Arguments: `{"expression":"2+2"}`,
			},
		},
	}
	result := convertAIParts(msg)

	if len(result) != 3 {
		t.Fatalf("result len = %d, want 3 (text + 2 tool calls)", len(result))
	}
	if result[0].Text != "Let me check that" {
		t.Errorf("result[0].Text = %q", result[0].Text)
	}
	if result[1].FunctionCall == nil {
		t.Fatal("expected result[1].FunctionCall to be set")
	}
	if result[1].FunctionCall.Name != "search" {
		t.Errorf("FunctionCall.Name = %q, want %q", result[1].FunctionCall.Name, "search")
	}
	if result[1].FunctionCall.ID != "call_1" {
		t.Errorf("FunctionCall.ID = %q, want %q", result[1].FunctionCall.ID, "call_1")
	}
	if result[2].FunctionCall == nil {
		t.Fatal("expected result[2].FunctionCall to be set")
	}
	if result[2].FunctionCall.Name != "calculator" {
		t.Errorf("FunctionCall.Name = %q, want %q", result[2].FunctionCall.Name, "calculator")
	}
}

func TestConvertAIParts_Empty(t *testing.T) {
	msg := &schema.AIMessage{}
	result := convertAIParts(msg)

	if len(result) != 0 {
		t.Errorf("result len = %d, want 0 for empty message", len(result))
	}
}

func TestConvertStreamResponse_Nil(t *testing.T) {
	chunk := convertStreamResponse(nil, "test-model")
	if chunk.ModelID != "test-model" {
		t.Errorf("ModelID = %q, want %q", chunk.ModelID, "test-model")
	}
	if chunk.Delta != "" {
		t.Errorf("Delta = %q, want empty", chunk.Delta)
	}
}

func TestConvertStreamResponse_FunctionCall(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{
							FunctionCall: &genai.FunctionCall{
								ID:   "call_xyz",
								Name: "get_weather",
								Args: map[string]any{"city": "SF"},
							},
						},
					},
				},
			},
		},
	}
	chunk := convertStreamResponse(resp, "gemini-test")

	if len(chunk.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(chunk.ToolCalls))
	}
	if chunk.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", chunk.ToolCalls[0].Name, "get_weather")
	}
	if chunk.ToolCalls[0].ID != "call_xyz" {
		t.Errorf("ToolCall.ID = %q, want %q", chunk.ToolCalls[0].ID, "call_xyz")
	}
}

func TestConvertStreamResponse_Usage(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: "done"}},
				},
				FinishReason: genai.FinishReasonStop,
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     100,
			CandidatesTokenCount: 50,
		},
	}
	chunk := convertStreamResponse(resp, "gemini-test")

	if chunk.Usage == nil {
		t.Fatal("expected Usage to be set")
	}
	if chunk.Usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", chunk.Usage.InputTokens)
	}
	if chunk.Usage.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", chunk.Usage.OutputTokens)
	}
	if chunk.Usage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", chunk.Usage.TotalTokens)
	}
	if chunk.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", chunk.FinishReason, "stop")
	}
}

func TestConvertResponse_NoCandidates(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{},
	}
	ai := convertResponse(resp, "test-model")

	if ai.ModelID != "test-model" {
		t.Errorf("ModelID = %q", ai.ModelID)
	}
	if len(ai.Parts) != 0 {
		t.Errorf("Parts len = %d, want 0", len(ai.Parts))
	}
}

func TestConvertResponse_NilContent(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: nil},
		},
	}
	ai := convertResponse(resp, "test-model")

	if ai.ModelID != "test-model" {
		t.Errorf("ModelID = %q", ai.ModelID)
	}
	if len(ai.Parts) != 0 {
		t.Errorf("Parts len = %d, want 0", len(ai.Parts))
	}
}

func TestConvertMessages_ToolMessageNonJSON(t *testing.T) {
	msgs := []schema.Message{
		schema.NewToolMessage("call_1", "plain text result"),
	}
	contents, _ := convertMessages(msgs)

	if len(contents) != 1 {
		t.Fatalf("contents len = %d, want 1", len(contents))
	}
	if len(contents[0].Parts) != 1 {
		t.Fatalf("parts len = %d, want 1", len(contents[0].Parts))
	}
	if contents[0].Parts[0].FunctionResponse == nil {
		t.Fatal("expected FunctionResponse to be set")
	}
	resp := contents[0].Parts[0].FunctionResponse.Response
	if resp == nil {
		t.Fatal("expected Response to be non-nil")
	}
	// Should have fallback: {"result": "plain text result"}
	if result, ok := resp["result"]; !ok || result != "plain text result" {
		t.Errorf("Response = %v, want map with 'result' key", resp)
	}
}

func TestGenerateWithToolChoice(t *testing.T) {
	var capturedBody map[string]any
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, geminiResponse("ok", nil))
	})
	defer ts.Close()

	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "test_tool", Description: "test"},
	})

	_, err := bound.Generate(context.Background(), []schema.Message{
		schema.NewHumanMessage("Hi"),
	}, llm.WithToolChoice(llm.ToolChoiceRequired))

	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify toolConfig is in the request.
	if _, ok := capturedBody["toolConfig"]; !ok {
		t.Error("expected toolConfig in request")
	}
}

func TestStreamWithFunctionCalls(t *testing.T) {
	ts, m := newTestModel(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Stream response with function call.
		chunk := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{
								"functionCall": map[string]any{
									"name": "get_weather",
									"args": map[string]any{"city": "NYC"},
									"id":   "call_123",
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
		}
		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", b)
	})
	defer ts.Close()

	bound := m.BindTools([]schema.ToolDefinition{
		{Name: "get_weather", Description: "Get weather"},
	})

	var toolCalls []schema.ToolCall
	for chunk, err := range bound.Stream(context.Background(), []schema.Message{
		schema.NewHumanMessage("What's the weather in NYC?"),
	}) {
		if err != nil {
			t.Fatalf("Stream() error: %v", err)
		}
		toolCalls = append(toolCalls, chunk.ToolCalls...)
	}

	if len(toolCalls) != 1 {
		t.Fatalf("toolCalls len = %d, want 1", len(toolCalls))
	}
	if toolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", toolCalls[0].Name, "get_weather")
	}
}
