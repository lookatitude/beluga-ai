package google

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"google.golang.org/genai"
)

func init() {
	llm.Register("google", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// Model implements llm.ChatModel using the Google Gemini API.
type Model struct {
	client *genai.Client
	model  string
	tools  []schema.ToolDefinition
}

// Compile-time interface check.
var _ llm.ChatModel = (*Model)(nil)

// New creates a new Google Gemini ChatModel.
func New(cfg config.ProviderConfig) (*Model, error) {
	return NewWithHTTPClient(cfg, nil)
}

// NewWithHTTPClient creates a new Google Gemini ChatModel with a custom HTTP client.
// This is useful for testing with httptest.
func NewWithHTTPClient(cfg config.ProviderConfig, httpClient *http.Client) (*Model, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("google: model is required")
	}

	cc := &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	}
	if httpClient != nil {
		cc.HTTPClient = httpClient
	}
	if cfg.BaseURL != "" {
		cc.HTTPOptions = genai.HTTPOptions{
			BaseURL: cfg.BaseURL,
		}
	}

	client, err := genai.NewClient(context.Background(), cc)
	if err != nil {
		return nil, fmt.Errorf("google: failed to create client: %w", err)
	}

	return &Model{
		client: client,
		model:  cfg.Model,
	}, nil
}

// Generate sends messages and returns a complete AI response.
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	contents, gcConfig := m.buildRequest(msgs, opts)
	resp, err := m.client.Models.GenerateContent(ctx, m.model, contents, gcConfig)
	if err != nil {
		return nil, fmt.Errorf("google: generate failed: %w", err)
	}
	return convertResponse(resp, m.model), nil
}

// Stream sends messages and returns an iterator of response chunks.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	contents, gcConfig := m.buildRequest(msgs, opts)
	return func(yield func(schema.StreamChunk, error) bool) {
		for resp, err := range m.client.Models.GenerateContentStream(ctx, m.model, contents, gcConfig) {
			if err != nil {
				yield(schema.StreamChunk{}, fmt.Errorf("google: stream error: %w", err))
				return
			}
			chunk := convertStreamResponse(resp, m.model)
			if !yield(chunk, nil) {
				return
			}
		}
	}
}

// BindTools returns a new Model that includes the given tools in every request.
func (m *Model) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	cp := *m
	cp.tools = make([]schema.ToolDefinition, len(tools))
	copy(cp.tools, tools)
	return &cp
}

// ModelID returns the model identifier.
func (m *Model) ModelID() string {
	return m.model
}

func (m *Model) buildRequest(msgs []schema.Message, opts []llm.GenerateOption) ([]*genai.Content, *genai.GenerateContentConfig) {
	contents, systemInstruction := convertMessages(msgs)
	genOpts := llm.ApplyOptions(opts...)

	gcConfig := &genai.GenerateContentConfig{}
	if systemInstruction != nil {
		gcConfig.SystemInstruction = systemInstruction
	}

	if genOpts.Temperature != nil {
		t := float32(*genOpts.Temperature)
		gcConfig.Temperature = &t
	}
	if genOpts.TopP != nil {
		p := float32(*genOpts.TopP)
		gcConfig.TopP = &p
	}
	if genOpts.MaxTokens > 0 {
		gcConfig.MaxOutputTokens = int32(genOpts.MaxTokens)
	}
	if len(genOpts.StopSequences) > 0 {
		gcConfig.StopSequences = genOpts.StopSequences
	}

	if len(m.tools) > 0 {
		gcConfig.Tools = convertTools(m.tools)
	}

	if genOpts.ToolChoice != "" || genOpts.SpecificTool != "" {
		applyToolChoice(gcConfig, genOpts)
	}

	return contents, gcConfig
}

func applyToolChoice(cfg *genai.GenerateContentConfig, genOpts llm.GenerateOptions) {
	switch genOpts.ToolChoice {
	case llm.ToolChoiceAuto:
		cfg.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAuto,
			},
		}
	case llm.ToolChoiceNone:
		cfg.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeNone,
			},
		}
	case llm.ToolChoiceRequired:
		cfg.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAny,
			},
		}
	}
	if genOpts.SpecificTool != "" {
		cfg.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode:                 genai.FunctionCallingConfigModeAny,
				AllowedFunctionNames: []string{genOpts.SpecificTool},
			},
		}
	}
}

func convertMessages(msgs []schema.Message) ([]*genai.Content, *genai.Content) {
	var system *genai.Content
	contents := make([]*genai.Content, 0, len(msgs))

	for _, msg := range msgs {
		switch m := msg.(type) {
		case *schema.SystemMessage:
			system = &genai.Content{
				Parts: []*genai.Part{{Text: m.Text()}},
				Role:  "user",
			}
		case *schema.HumanMessage:
			parts := convertHumanParts(m.Parts)
			contents = append(contents, &genai.Content{
				Parts: parts,
				Role:  "user",
			})
		case *schema.AIMessage:
			parts := convertAIParts(m)
			contents = append(contents, &genai.Content{
				Parts: parts,
				Role:  "model",
			})
		case *schema.ToolMessage:
			var result map[string]any
			json.Unmarshal([]byte(m.Text()), &result)
			if result == nil {
				result = map[string]any{"result": m.Text()}
			}
			contents = append(contents, &genai.Content{
				Parts: []*genai.Part{{
					FunctionResponse: &genai.FunctionResponse{
						Name:     m.ToolCallID,
						Response: result,
					},
				}},
				Role: "user",
			})
		}
	}
	return contents, system
}

func convertHumanParts(parts []schema.ContentPart) []*genai.Part {
	result := make([]*genai.Part, 0, len(parts))
	for _, p := range parts {
		switch cp := p.(type) {
		case schema.TextPart:
			result = append(result, &genai.Part{Text: cp.Text})
		case schema.ImagePart:
			if len(cp.Data) > 0 {
				mime := cp.MimeType
				if mime == "" {
					mime = "image/png"
				}
				result = append(result, &genai.Part{
					InlineData: &genai.Blob{
						Data:     cp.Data,
						MIMEType: mime,
					},
				})
			} else if cp.URL != "" {
				result = append(result, &genai.Part{
					FileData: &genai.FileData{
						FileURI:  cp.URL,
						MIMEType: "image/png",
					},
				})
			}
		}
	}
	return result
}

func convertAIParts(m *schema.AIMessage) []*genai.Part {
	var parts []*genai.Part
	text := m.Text()
	if text != "" {
		parts = append(parts, &genai.Part{Text: text})
	}
	for _, tc := range m.ToolCalls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Arguments), &args)
		parts = append(parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				Name: tc.Name,
				Args: args,
				ID:   tc.ID,
			},
		})
	}
	return parts
}

func convertTools(tools []schema.ToolDefinition) []*genai.Tool {
	decls := make([]*genai.FunctionDeclaration, len(tools))
	for i, t := range tools {
		decls[i] = &genai.FunctionDeclaration{
			Name:                 t.Name,
			Description:          t.Description,
			ParametersJsonSchema: t.InputSchema,
		}
	}
	return []*genai.Tool{{FunctionDeclarations: decls}}
}

func convertResponse(resp *genai.GenerateContentResponse, modelID string) *schema.AIMessage {
	if resp == nil {
		return &schema.AIMessage{ModelID: modelID}
	}
	ai := &schema.AIMessage{ModelID: modelID}

	if resp.UsageMetadata != nil {
		ai.Usage = schema.Usage{
			InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
			OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:  int(resp.UsageMetadata.PromptTokenCount + resp.UsageMetadata.CandidatesTokenCount),
			CachedTokens: int(resp.UsageMetadata.CachedContentTokenCount),
		}
	}

	if len(resp.Candidates) == 0 {
		return ai
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil {
		return ai
	}

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			ai.Parts = append(ai.Parts, schema.TextPart{Text: part.Text})
		}
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			ai.ToolCalls = append(ai.ToolCalls, schema.ToolCall{
				ID:        part.FunctionCall.ID,
				Name:      part.FunctionCall.Name,
				Arguments: string(args),
			})
		}
	}

	return ai
}

func convertStreamResponse(resp *genai.GenerateContentResponse, modelID string) schema.StreamChunk {
	chunk := schema.StreamChunk{ModelID: modelID}
	if resp == nil {
		return chunk
	}

	if len(resp.Candidates) > 0 {
		extractCandidateContent(resp.Candidates[0], &chunk)
	}

	if resp.UsageMetadata != nil {
		chunk.Usage = convertStreamUsage(resp.UsageMetadata)
	}

	return chunk
}

// extractCandidateContent populates a StreamChunk with data from a response candidate.
func extractCandidateContent(candidate *genai.Candidate, chunk *schema.StreamChunk) {
	if candidate.FinishReason != "" {
		chunk.FinishReason = mapFinishReason(candidate.FinishReason)
	}
	if candidate.Content == nil {
		return
	}
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			chunk.Delta += part.Text
		}
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			chunk.ToolCalls = append(chunk.ToolCalls, schema.ToolCall{
				ID:        part.FunctionCall.ID,
				Name:      part.FunctionCall.Name,
				Arguments: string(args),
			})
		}
	}
}

// convertStreamUsage converts genai usage metadata to a schema.Usage pointer.
func convertStreamUsage(meta *genai.GenerateContentResponseUsageMetadata) *schema.Usage {
	return &schema.Usage{
		InputTokens:  int(meta.PromptTokenCount),
		OutputTokens: int(meta.CandidatesTokenCount),
		TotalTokens:  int(meta.PromptTokenCount + meta.CandidatesTokenCount),
	}
}

func mapFinishReason(reason genai.FinishReason) string {
	switch reason {
	case genai.FinishReasonStop:
		return "stop"
	case genai.FinishReasonMaxTokens:
		return "length"
	default:
		return string(reason)
	}
}
