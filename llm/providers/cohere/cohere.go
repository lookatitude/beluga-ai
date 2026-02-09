// Package cohere provides the Cohere LLM provider for the Beluga AI framework.
// It uses the official Cohere Go SDK (v2) to communicate with the Cohere API.
//
// Cohere uses a different message format than OpenAI: the last user message is
// the "message" field, system messages go into "preamble", and prior messages
// become "chat_history". This provider handles all the mapping transparently.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/llm/providers/cohere"
//
//	model, err := llm.New("cohere", config.ProviderConfig{
//	    Model:  "command-r-plus",
//	    APIKey: "...",
//	})
package cohere

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"

	coherego "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	"github.com/cohere-ai/cohere-go/v2/option"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

const defaultModel = "command-r-plus"

func init() {
	llm.Register("cohere", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
		return New(cfg)
	})
}

// Model implements llm.ChatModel using the Cohere SDK.
type Model struct {
	client *cohereclient.Client
	model  string
	tools  []schema.ToolDefinition
}

// Compile-time interface check.
var _ llm.ChatModel = (*Model)(nil)

// New creates a new Cohere ChatModel.
func New(cfg config.ProviderConfig) (*Model, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("cohere: api_key is required")
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	opts := []option.RequestOption{
		cohereclient.WithToken(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, cohereclient.WithBaseURL(cfg.BaseURL))
	}
	client := cohereclient.NewClient(opts...)
	return &Model{
		client: client,
		model:  model,
	}, nil
}

// Generate sends messages and returns a complete AI response.
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	req := m.buildRequest(msgs, opts)
	resp, err := m.client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cohere: generate failed: %w", err)
	}
	return convertResponse(resp, m.model), nil
}

// Stream sends messages and returns an iterator of response chunks.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	streamReq := m.buildStreamRequest(msgs, opts)

	stream, err := m.client.ChatStream(ctx, streamReq)
	if err != nil {
		return func(yield func(schema.StreamChunk, error) bool) {
			yield(schema.StreamChunk{}, fmt.Errorf("cohere: stream failed: %w", err))
		}
	}

	return func(yield func(schema.StreamChunk, error) bool) {
		defer stream.Close()
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				yield(schema.StreamChunk{}, err)
				return
			}

			sc := schema.StreamChunk{ModelID: m.model}
			switch {
			case event.TextGeneration != nil:
				sc.Delta = event.TextGeneration.Text
			case event.ToolCallsGeneration != nil:
				sc.ToolCalls = convertToolCalls(event.ToolCallsGeneration.ToolCalls)
			case event.StreamEnd != nil:
				sc.FinishReason = string(event.StreamEnd.FinishReason)
				if event.StreamEnd.Response != nil && event.StreamEnd.Response.Meta != nil {
					sc.Usage = convertUsage(event.StreamEnd.Response.Meta)
				}
			default:
				continue
			}
			if !yield(sc, nil) {
				return
			}
		}
	}
}

// BindTools returns a new Model with the given tool definitions.
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

// buildRequest constructs a Cohere ChatRequest from Beluga messages.
// Cohere's v1 chat API splits messages into: preamble (system), message (last user),
// and chat_history (everything in between).
func (m *Model) buildRequest(msgs []schema.Message, opts []llm.GenerateOption) *coherego.ChatRequest {
	preamble, chatHistory, message := splitMessages(msgs)
	genOpts := llm.ApplyOptions(opts...)

	req := &coherego.ChatRequest{
		Message: message,
		Model:   &m.model,
	}
	if preamble != "" {
		req.Preamble = &preamble
	}
	if len(chatHistory) > 0 {
		req.ChatHistory = chatHistory
	}
	applyOptions(req, genOpts)
	if len(m.tools) > 0 {
		req.Tools = convertToolDefs(m.tools)
	}
	return req
}

func (m *Model) buildStreamRequest(msgs []schema.Message, opts []llm.GenerateOption) *coherego.ChatStreamRequest {
	preamble, chatHistory, message := splitMessages(msgs)
	genOpts := llm.ApplyOptions(opts...)

	req := &coherego.ChatStreamRequest{
		Message: message,
		Model:   &m.model,
	}
	if preamble != "" {
		req.Preamble = &preamble
	}
	if len(chatHistory) > 0 {
		req.ChatHistory = chatHistory
	}
	if genOpts.Temperature != nil {
		req.Temperature = genOpts.Temperature
	}
	if genOpts.MaxTokens > 0 {
		req.MaxTokens = &genOpts.MaxTokens
	}
	if genOpts.TopP != nil {
		req.P = genOpts.TopP
	}
	if len(genOpts.StopSequences) > 0 {
		req.StopSequences = genOpts.StopSequences
	}
	if len(m.tools) > 0 {
		req.Tools = convertToolDefs(m.tools)
	}
	return req
}

func applyOptions(req *coherego.ChatRequest, opts llm.GenerateOptions) {
	if opts.Temperature != nil {
		req.Temperature = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		req.MaxTokens = &opts.MaxTokens
	}
	if opts.TopP != nil {
		req.P = opts.TopP
	}
	if len(opts.StopSequences) > 0 {
		req.StopSequences = opts.StopSequences
	}
}

// splitMessages separates a flat list of Beluga messages into Cohere's
// preamble (system), chat_history (prior turns), and message (last user input).
func splitMessages(msgs []schema.Message) (preamble string, history []*coherego.Message, message string) {
	if len(msgs) == 0 {
		return "", nil, ""
	}

	// Extract system messages into preamble
	var systemParts []string
	var nonSystem []schema.Message
	for _, msg := range msgs {
		if sm, ok := msg.(*schema.SystemMessage); ok {
			systemParts = append(systemParts, sm.Text())
		} else {
			nonSystem = append(nonSystem, msg)
		}
	}
	if len(systemParts) > 0 {
		for i, s := range systemParts {
			if i > 0 {
				preamble += "\n"
			}
			preamble += s
		}
	}

	if len(nonSystem) == 0 {
		return preamble, nil, ""
	}

	// Last message becomes the "message" field.
	// All prior non-system messages become chat_history.
	last := nonSystem[len(nonSystem)-1]
	if hm, ok := last.(*schema.HumanMessage); ok {
		message = hm.Text()
	} else if am, ok := last.(*schema.AIMessage); ok {
		message = am.Text()
	} else if tm, ok := last.(*schema.ToolMessage); ok {
		message = tm.Text()
	}

	if len(nonSystem) > 1 {
		for _, msg := range nonSystem[:len(nonSystem)-1] {
			cm := convertToCohereMessage(msg)
			if cm != nil {
				history = append(history, cm)
			}
		}
	}

	return preamble, history, message
}

func convertToCohereMessage(msg schema.Message) *coherego.Message {
	switch m := msg.(type) {
	case *schema.HumanMessage:
		return &coherego.Message{
			Role: "USER",
			User: &coherego.ChatMessage{Message: m.Text()},
		}
	case *schema.AIMessage:
		return &coherego.Message{
			Role:    "CHATBOT",
			Chatbot: &coherego.ChatMessage{Message: m.Text()},
		}
	case *schema.ToolMessage:
		return &coherego.Message{
			Role: "USER",
			User: &coherego.ChatMessage{Message: m.Text()},
		}
	default:
		return nil
	}
}

func convertToolDefs(tools []schema.ToolDefinition) []*coherego.Tool {
	out := make([]*coherego.Tool, len(tools))
	for i, t := range tools {
		ct := &coherego.Tool{
			Name:        t.Name,
			Description: t.Description,
		}
		if t.InputSchema != nil {
			if props, ok := t.InputSchema["properties"].(map[string]any); ok {
				paramDefs := make(map[string]*coherego.ToolParameterDefinitionsValue)
				for k, v := range props {
					pv := &coherego.ToolParameterDefinitionsValue{}
					if vm, ok := v.(map[string]any); ok {
						if desc, ok := vm["description"].(string); ok {
							pv.Description = &desc
						}
						if typ, ok := vm["type"].(string); ok {
							pv.Type = typ
						}
					}
					paramDefs[k] = pv
				}
				ct.ParameterDefinitions = paramDefs
			}
		}
		out[i] = ct
	}
	return out
}

func convertResponse(resp *coherego.NonStreamedChatResponse, modelID string) *schema.AIMessage {
	if resp == nil {
		return &schema.AIMessage{ModelID: modelID}
	}
	ai := &schema.AIMessage{ModelID: modelID}
	if resp.Text != "" {
		ai.Parts = []schema.ContentPart{schema.TextPart{Text: resp.Text}}
	}
	if len(resp.ToolCalls) > 0 {
		ai.ToolCalls = convertToolCalls(resp.ToolCalls)
	}
	if resp.Meta != nil {
		ai.Usage = *convertUsage(resp.Meta)
	}
	return ai
}

func convertToolCalls(calls []*coherego.ToolCall) []schema.ToolCall {
	out := make([]schema.ToolCall, len(calls))
	for i, tc := range calls {
		args, _ := json.Marshal(tc.Parameters)
		out[i] = schema.ToolCall{
			ID:        fmt.Sprintf("call_%d", i),
			Name:      tc.Name,
			Arguments: string(args),
		}
	}
	return out
}

func convertUsage(meta *coherego.ApiMeta) *schema.Usage {
	usage := &schema.Usage{}
	if meta.Tokens != nil {
		if meta.Tokens.InputTokens != nil {
			usage.InputTokens = int(*meta.Tokens.InputTokens)
		}
		if meta.Tokens.OutputTokens != nil {
			usage.OutputTokens = int(*meta.Tokens.OutputTokens)
		}
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	return usage
}
