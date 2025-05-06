// Package cohere provides an implementation of the llms.ChatModel interface
// using the Cohere API.
package cohere

import (
	"context"
	"encoding/json" // Added for unmarshalling tool arguments and parameters
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	coherecore "github.com/cohere-ai/cohere-go/v2/core"
	cohere "github.com/cohere-ai/cohere-go/v2" // Main Cohere types

	belugaConfig "github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
)

// CohereChat represents a chat model client for Cohere.
type CohereChat struct {
	client               *cohereclient.Client
	modelName            string
	apiKey               string
	maxConcurrentBatches int
	defaultTemperature *float64
	defaultMaxTokens   *int
	defaultP           *float64
	defaultK           *int
	boundTools []*cohere.Tool // Changed coheretypes.Tool to cohere.Tool
}

// Compile-time check to ensure CohereChat implements interfaces.
var _ llms.ChatModel = (*CohereChat)(nil)
var _ core.Runnable = (*CohereChat)(nil)

// CohereOption is a function type for setting options on the CohereChat client.
type CohereOption func(*CohereChat)

// WithCohereMaxConcurrentBatches sets the concurrency limit for Batch.
func WithCohereMaxConcurrentBatches(n int) CohereOption {
	return func(cc *CohereChat) {
		if n > 0 {
			cc.maxConcurrentBatches = n
		}
	}
}

// WithCohereDefaultTemperature sets the default temperature.
func WithCohereDefaultTemperature(temp float64) CohereOption {
	return func(cc *CohereChat) {
		cc.defaultTemperature = &temp
	}
}

// WithCohereDefaultMaxTokens sets the default max tokens.
func WithCohereDefaultMaxTokens(maxTokens int) CohereOption {
	return func(cc *CohereChat) {
		cc.defaultMaxTokens = &maxTokens
	}
}

// NewCohereChat creates a new Cohere chat client.
func NewCohereChat(options ...CohereOption) (*CohereChat, error) {
	apiKey := belugaConfig.Cfg.LLMs.Cohere.APIKey
	modelName := belugaConfig.Cfg.LLMs.Cohere.Model

	if apiKey == "" {
		return nil, errors.New("Cohere API key not found in configuration (BELUGA_LLMS_COHERE_APIKEY)")
	}
	if modelName == "" {
		modelName = "command-r" // Default model
		log.Printf("Cohere model name not found in configuration, defaulting to %s", modelName)
	}

	client := cohereclient.NewClient(cohereclient.WithToken(apiKey))

	cc := &CohereChat{
		client:               client,
		modelName:            modelName,
		apiKey:               apiKey,
		maxConcurrentBatches: 5, // Default concurrency
	}

	for _, opt := range options {
		opt(cc)
	}

	return cc, nil
}

// --- Message & Tool Mapping ---

func mapMessagesToCohereChatRequest(messages []schema.Message, toolsToBind []*cohere.Tool, stream bool, options ...core.Option) (*cohere.ChatRequest, error) {
	chatHistory := []*cohere.Message{}
	var currentMessage string
	var systemPreamble *string

	for i, msg := range messages {
		switch m := msg.(type) {
		case *schema.SystemMessage:
			if i == 0 {
				pre := m.GetContent()
				systemPreamble = &pre
			} else {
				log.Println("Warning: System message found after the first message for Cohere, prepending to next user message content.")
				currentMessage = m.GetContent() + "\n" + currentMessage
			}
		case *schema.HumanMessage:
			currentMessage += m.GetContent()
		case *schema.AIMessage:
			if currentMessage != "" {
				// Create a copy of currentMessage for the closure
				msgCopy := currentMessage
				chatHistory = append(chatHistory, &cohere.Message{Role: cohere.MessageRoleUser, Message: &msgCopy})
				currentMessage = ""
			}
			aiContent := m.GetContent()
			aiResponse := &cohere.Message{Role: cohere.MessageRoleChatbot, Message: &aiContent}
			if len(m.ToolCalls) > 0 {
				aiResponse.ToolCalls = []*cohere.ToolCall{}
				for _, tc := range m.ToolCalls {
					var params map[string]any
					if tc.Arguments != "" && tc.Arguments != "{}" && tc.Arguments != "null" {
						err := json.Unmarshal([]byte(tc.Arguments), &params)
						if err != nil {
							log.Printf("Warning: Failed to unmarshal tool call arguments for Cohere AI message: %v. Arguments: %s. Using raw string.", err, tc.Arguments)
							params = map[string]any{"_beluga_raw_args": tc.Arguments} // Fallback
						}
					} else {
						params = make(map[string]any) // Empty params if arguments are empty/null
					}
					aiResponse.ToolCalls = append(aiResponse.ToolCalls, &cohere.ToolCall{Name: tc.Name, Parameters: params})
				}
			}
			chatHistory = append(chatHistory, aiResponse)

		case *schema.ToolMessage:
			if currentMessage != "" {
				msgCopy := currentMessage
				chatHistory = append(chatHistory, &cohere.Message{Role: cohere.MessageRoleUser, Message: &msgCopy})
				currentMessage = ""
			}
			// IMPORTANT: schema.ToolMessage lacks the original tool name. Using ToolCallID as placeholder for Name.
			// This is likely incorrect for Cohere_s expectation of ToolResult.Call.Name.
			// A more robust solution requires schema.ToolMessage to carry the original tool_s name.
			log.Printf("Warning: Mapping schema.ToolMessage to Cohere ToolResult. Using ToolCallID (_%s_) as placeholder for the called tool_s Name due to schema limitations.", m.ToolCallID)
			toolResultMsg := &cohere.Message{
				Role: cohere.MessageRoleTool,
				ToolResults: []*cohere.ToolResult{{
					Call:    &cohere.ToolCall{Name: m.ToolCallID /* Placeholder */},
					Outputs: []map[string]any{{"content": m.GetContent()}},
				}},
			}
			chatHistory = append(chatHistory, toolResultMsg)

		default:
			log.Printf("Warning: Skipping message of unknown type %T for Cohere conversion.\n", msg)
		}
	}

	var finalUserMessage string
	if currentMessage != "" {
		finalUserMessage = currentMessage
	} else if len(chatHistory) == 0 && systemPreamble == nil {
		return nil, errors.New("no valid message content to send to Cohere")
	}

	req := &cohere.ChatRequest{
		ChatHistory: chatHistory,
		Message:     finalUserMessage,
		Preamble:    systemPreamble,
		Tools:       toolsToBind,
	}

	configMap := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&configMap)
	}

	if temp, ok := configMap["temperature"].(float64); ok { req.Temperature = &temp }
	if maxTokens, ok := configMap["max_tokens"].(int); ok { req.MaxTokens = &maxTokens }
	if p, ok := configMap["top_p"].(float64); ok { req.P = &p } 
	if k, ok := configMap["top_k"].(int); ok { req.K = &k } 
	if stops, ok := configMap["stop_sequences"].([]string); ok { req.StopSequences = &stops }
    if model, ok := configMap["model_name"].(string); ok && model != "" {
        req.Model = &model 
    }

	return req, nil
}


func mapCohereResponseToAIMessage(resp *cohere.NonStreamedChatResponse) (schema.Message, error) {
	if resp == nil {
		return nil, errors.New("received nil response from Cohere")
	}

	aiMsg := schema.NewAIMessage(resp.Text) 

	if len(resp.ToolCalls) > 0 {
		aiMsg.ToolCalls = []schema.ToolCall{}
		for i, tc := range resp.ToolCalls {
			argsBytes, err := json.Marshal(tc.Parameters)
			argsStr := "{}" 
			if err == nil {
				argsStr = string(argsBytes)
			} else {
				log.Printf("Warning: Failed to marshal Cohere tool call parameters for tool _%s_: %v. Using empty JSON.", tc.Name, err)
			}
			toolCallID := fmt.Sprintf("%s-%s-%d", tc.Name, resp.GenerationId, i) 
			aiMsg.ToolCalls = append(aiMsg.ToolCalls, schema.ToolCall{
				ID:        toolCallID,
				Name:      tc.Name,
				Arguments: argsStr,
			})
		}
	}

	if aiMsg.AdditionalArgs == nil { 
		aiMsg.AdditionalArgs = make(map[string]any)
	}
	usageMap := map[string]int{}
	if resp.Meta != nil && resp.Meta.Tokens != nil {
		if resp.Meta.Tokens.InputTokens != nil {
			usageMap["input_tokens"] = int(*resp.Meta.Tokens.InputTokens)
		}
		if resp.Meta.Tokens.OutputTokens != nil {
			usageMap["output_tokens"] = int(*resp.Meta.Tokens.OutputTokens)
		}
		usageMap["total_tokens"] = usageMap["input_tokens"] + usageMap["output_tokens"]
	}
	if len(usageMap) > 0 {
		aiMsg.AdditionalArgs["usage"] = usageMap
	}
	if resp.FinishReason != nil {
		aiMsg.AdditionalArgs["finish_reason"] = resp.FinishReason.String()
	}

	return aiMsg, nil
}

func mapCohereStreamChunkToAIMessageChunk(event cohere.StreamEvent) (llms.AIMessageChunk, error) {
	chunk := llms.AIMessageChunk{AdditionalArgs: make(map[string]any)} 

	switch e := event.(type) {
	case *cohere.StreamEventTextGeneration:
		chunk.Content = e.Text
	case *cohere.StreamEventStreamEnd:
		if e.FinishReason != nil {
			chunk.AdditionalArgs["finish_reason"] = e.FinishReason.String()
		}
		if e.Response != nil && e.Response.Meta != nil && e.Response.Meta.Tokens != nil {
			usageMap := map[string]int{}
			if e.Response.Meta.Tokens.InputTokens != nil {
				usageMap["input_tokens"] = int(*e.Response.Meta.Tokens.InputTokens)
			}
			if e.Response.Meta.Tokens.OutputTokens != nil {
				usageMap["output_tokens"] = int(*e.Response.Meta.Tokens.OutputTokens)
			}
			usageMap["total_tokens"] = usageMap["input_tokens"] + usageMap["output_tokens"]
			if len(usageMap) > 0 {
				chunk.AdditionalArgs["usage"] = usageMap
			}
		}
	case *cohere.StreamEventToolCallsGeneration:
		log.Println("Cohere stream: Tool calls generation started.")
	case *cohere.StreamEventToolCall: 
		toolCallChunks := []schema.ToolCallChunk{}
		argsStr := "{}"
		if e.Parameters != nil { 
			argsBytes, err := json.Marshal(e.Parameters)
			if err == nil {
				argsStr = string(argsBytes)
			} else {
				log.Printf("Warning: Failed to marshal Cohere streaming tool call parameters: %v", err)
			}
		}
		nameCopy := e.Name
		idCopy := fmt.Sprintf("%s-stream-%s", e.Name, "placeholderStreamID") 
		
		toolCallChunks = append(toolCallChunks, schema.ToolCallChunk{
			ID:        &idCopy, 
			Name:      &nameCopy,
			Arguments: &argsStr,
		})
		if len(toolCallChunks) > 0 {
			chunk.ToolCallChunks = toolCallChunks
		}

	default:
		log.Printf("Warning: Unhandled Cohere stream event type: %T\n", e)
	}
	if len(chunk.AdditionalArgs) == 0 { 
		chunk.AdditionalArgs = nil
	}
	return chunk, nil
}


func mapToolsToCohere(toolsToBind []tools.Tool) ([]*cohere.Tool, error) {
	cohereTools := make([]*cohere.Tool, 0, len(toolsToBind))
	for _, t := range toolsToBind {
		paramDefs := make(map[string]*cohere.ToolParameterDefinitionsValue)
		schemaStr := t.Schema()
		if schemaStr != "" && schemaStr != "{}" && schemaStr != "null" {
			var openAPISchema struct {
				Type       string                            `json:"type"`
				Properties map[string]map[string]interface{} `json:"properties"`
				Required   []string                          `json:"required"`
			}
			err := json.Unmarshal([]byte(schemaStr), &openAPISchema)
			if err != nil {
				log.Printf("Warning: Failed to unmarshal schema for tool %s for Cohere binding: %v. Skipping parameters.", t.Name(), err)
			} else {
				if strings.ToLower(openAPISchema.Type) == "object" && openAPISchema.Properties != nil {
					for propName, propDetails := range openAPISchema.Properties {
						desc, _ := propDetails["description"].(string)
						propTypeStr, _ := propDetails["type"].(string)
						paramDefs[propName] = &cohere.ToolParameterDefinitionsValue{
							Description: &desc,
							Type:        propTypeStr, 
							Required:    coherecore.Bool(isStringInSlice(propName, openAPISchema.Required)),
						}
					}
				}
			}
		}

		cohereTools = append(cohereTools, &cohere.Tool{
			Name:                  t.Name(),
			Description:           t.Description(),
			ParameterDefinitions:  paramDefs,
		})
	}
	return cohereTools, nil
}

func isStringInSlice(str string, list []string) bool {
    for _, item := range list {
        if item == str {
            return true
        }
    }
    return false
}

// --- llms.ChatModel Implementation ---

// Generate implements the llms.ChatModel interface.
func (cc *CohereChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	req, err := mapMessagesToCohereChatRequest(messages, cc.boundTools, false, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for Cohere: %w", err)
	}
    if req.Model == nil || *req.Model == "" { 
        req.Model = &cc.modelName
    }

	resp, err := cc.client.Chat(ctx, req)
	if err != nil {
		var cohereErr *coherecore.APIError
		if errors.As(err, &cohereErr) {
			bodyBytes, _ := io.ReadAll(cohereErr.Response().Body)
			return nil, fmt.Errorf("cohere Chat API error: %s, status: %d, body: %s", cohereErr.Message(), cohereErr.StatusCode(), string(bodyBytes))
		}
		return nil, fmt.Errorf("cohere Chat failed: %w", err)
	}

	return mapCohereResponseToAIMessage(resp)
}

// StreamChat implements the llms.ChatModel interface.
func (cc *CohereChat) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llms.AIMessageChunk, error) {
	req, err := mapMessagesToCohereChatRequest(messages, cc.boundTools, true, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to map messages for Cohere stream: %w", err)
	}
    if req.Model == nil || *req.Model == "" { 
        req.Model = &cc.modelName
    }

	stream, err := cc.client.ChatStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cohere ChatStream failed: %w", err)
	}

	chunkChan := make(chan llms.AIMessageChunk, 1) 
	go func() {
		defer close(chunkChan)
		defer stream.Close()

		for {
			event, err := stream.Recv()
			if err != nil { 
				if err == io.EOF {
					return 
				}
				log.Printf("Cohere stream Recv error: %v", err)
				select {
				case chunkChan <- llms.AIMessageChunk{Err: fmt.Errorf("cohere stream error: %w", err)}:
				case <-ctx.Done():
				}
				return
			}

			chunk, mapErr := mapCohereStreamChunkToAIMessageChunk(event)
			if mapErr != nil {
				log.Printf("Cohere stream mapping error: %v", mapErr)
				select {
				case chunkChan <- llms.AIMessageChunk{Err: mapErr}:
				case <-ctx.Done():
				}
				return 
			}

			isMeaningfulChunk := chunk.Content != "" ||
				len(chunk.ToolCallChunks) > 0 ||
				(chunk.AdditionalArgs != nil && chunk.AdditionalArgs["finish_reason"] != nil)

			if isMeaningfulChunk {
				select {
				case chunkChan <- chunk:
				case <-ctx.Done():
					log.Println("Cohere stream cancelled by context during send")
					return
				}
			}
		}
	}()

	return chunkChan, nil
}

// BindTools implements the llms.ChatModel interface.
func (cc *CohereChat) BindTools(toolsToBind []tools.Tool) llms.ChatModel {
	boundClient := *cc 
	cohereTools, err := mapToolsToCohere(toolsToBind)
	if err != nil {
		log.Printf("Error mapping tools for Cohere binding: %v. Tools will not be bound.", err)
		boundClient.boundTools = nil
	} else {
		boundClient.boundTools = cohereTools
	}
	return &boundClient
}

// --- core.Runnable Implementation ---

// Invoke implements the core.Runnable interface.
func (cc *CohereChat) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return cc.Generate(ctx, messages, options...)
}

// Batch implements the core.Runnable interface.
func (cc *CohereChat) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
    var firstErr error
	var wg sync.WaitGroup
	sem := make(chan struct{}, cc.maxConcurrentBatches)
    var mu sync.Mutex

	for i, input := range inputs {
		wg.Add(1)
		sem <- struct{}{} 
		go func(index int, currentInput any) {
			defer wg.Done()
			defer func() { <-sem }() 

			result, err := cc.Invoke(ctx, currentInput, options...) 

            mu.Lock()
			if err != nil {
                results[index] = err 
                if firstErr == nil {
                    firstErr = fmt.Errorf("error in batch item %d: %w", index, err)
                }
            } else {
                results[index] = result
            }
            mu.Unlock()
		}(i, input)
	}

	wg.Wait()
	return results, firstErr 
}

// Stream implements the core.Runnable interface.
func (cc *CohereChat) Stream(ctx context.Context, input any, options ...core.Option) (<-chan core.Chunk, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	aiChunkChan, err := cc.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	outputChan := make(chan core.Chunk)
	go func() {
		defer close(outputChan)
		for aiChunk := range aiChunkChan {
			outputChan <- aiChunk // llms.AIMessageChunk is already a core.Chunk
		}
	}()
	return outputChan, nil
}

// GetModelName returns the model name used by the client.
func (cc *CohereChat) GetModelName() string {
	return cc.modelName
}

// GetBoundTools returns the tools bound to the client.
func (cc *CohereChat) GetBoundTools() []*cohere.Tool {
	return cc.boundTools
}

