// Package gemini provides an implementation of the llms.ChatModel interface
// using the Google Generative AI SDK for Gemini models.
package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	// "io" // Unused, removing
	"log"
	// "strings" // Unused, removing
	"sync"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	belugaConfig "github.com/lookatitude/beluga-ai/pkg/config" // Alias config
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// GeminiChat represents a chat model client for Google Gemini.
type GeminiChat struct {
	client               *genai.Client
	modelName            string
	apiKey               string // Store API key for potential re-use or validation
	maxConcurrentBatches int
	defaultGenConfig     *genai.GenerationConfig
	defaultSafety        []*genai.SafetySetting
	boundTools           []*genai.Tool
}

// Compile-time check to ensure GeminiChat implements interfaces.
var _ llms.ChatModel = (*GeminiChat)(nil)
var _ core.Runnable = (*GeminiChat)(nil)

// GeminiOption is a function type for setting options on the GeminiChat client.
type GeminiOption func(*GeminiChat)

// WithGeminiMaxConcurrentBatches sets the concurrency limit for Batch.
func WithGeminiMaxConcurrentBatches(n int) GeminiOption {
	return func(gc *GeminiChat) {
		 if n > 0 {
		 	 gc.maxConcurrentBatches = n
		 }
	}
}

// WithGeminiDefaultGenerationConfig sets the default generation configuration.
func WithGeminiDefaultGenerationConfig(config *genai.GenerationConfig) GeminiOption {
	return func(gc *GeminiChat) {
		 gc.defaultGenConfig = config
	}
}

// WithGeminiDefaultSafetySettings sets the default safety settings.
func WithGeminiDefaultSafetySettings(settings []*genai.SafetySetting) GeminiOption {
	return func(gc *GeminiChat) {
		 gc.defaultSafety = settings
	}
}

// NewGeminiChat creates a new Gemini chat client.
// It requires an API key and model name (e.g., "gemini-1.5-flash-latest").
func NewGeminiChat(ctx context.Context, options ...GeminiOption) (*GeminiChat, error) {
	 apiKey := belugaConfig.Cfg.LLMs.Gemini.APIKey
	 modelName := belugaConfig.Cfg.LLMs.Gemini.Model

	 if apiKey == "" {
	 	 return nil, errors.New("Google Gemini API key not found in configuration (BELUGA_LLMS_GEMINI_APIKEY)")
	 }
	 if modelName == "" {
	 	 return nil, errors.New("Google Gemini model name not found in configuration (BELUGA_LLMS_GEMINI_MODEL)")
	 }

	 client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	 if err != nil {
	 	 return nil, fmt.Errorf("failed to create Google GenAI client: %w", err)
	 }

	 gc := &GeminiChat{
	 	 client:               client,
	 	 modelName:            modelName,
	 	 apiKey:               apiKey,
	 	 maxConcurrentBatches: 5, // Default concurrency
	 	 // Initialize empty defaults, they can be set via options
	 	 defaultGenConfig: &genai.GenerationConfig{},
	 	 defaultSafety:    []*genai.SafetySetting{},
	 }

	 // Apply functional options
	 for _, opt := range options {
	 	 opt(gc)
	 }

	 return gc, nil
}

// --- Message & Tool Mapping ---

// mapMessagesToGeminiContent converts Beluga messages to Gemini content format.
// It handles system instructions, user/model roles, and tool calls/responses.
func mapMessagesToGeminiContent(messages []schema.Message) ([]*genai.Content, *genai.Content, error) {
	 var history []*genai.Content
	 var systemInstruction *genai.Content
	 var currentContent *genai.Content

	 for i, msg := range messages {
	 	 role := ""
	 	 var parts []genai.Part

	 	 switch m := msg.(type) {
	 	 case *schema.SystemMessage:
	 	 	 if i == 0 {
	 	 	 	 systemInstruction = &genai.Content{Parts: []genai.Part{genai.Text(m.GetContent())}}
	 	 	 	 continue // System instruction handled separately
	 	 	 } else {
	 	 	 	 log.Println("Warning: System message found after the first message, treating as user message for Gemini.")
	 	 	 	 role = "user"
	 	 	 	 parts = append(parts, genai.Text(m.GetContent()))
	 	 	 }
	 	 case *schema.HumanMessage:
	 	 	 role = "user"
	 	 	 // TODO: Handle multimodal input (images) if schema supports it
	 	 	 parts = append(parts, genai.Text(m.GetContent()))
	 	 case *schema.AIMessage:
	 	 	 role = "model" // Gemini uses "model" for assistant role
	 	 	 hasContent := m.GetContent() != ""
	 	 	 hasToolCalls := len(m.ToolCalls) > 0

	 	 	 if hasToolCalls {
	 	 	 	 for _, tc := range m.ToolCalls {
	 	 	 	 	 // Convert arguments string back to map[string]any if possible
	 	 	 	 	 // Gemini FunctionCall expects structured data, but SDK might handle string?
	 	 	 	 	 // Let's assume SDK handles string for now, needs verification.
	 	 	 	 	 // If not, unmarshal tc.Arguments here.
	 	 	 	 	 parts = append(parts, &genai.FunctionCall{Name: tc.Name, Args: map[string]any{"_beluga_raw_args": tc.Arguments}}) // Wrap raw args
	 	 	 	 }
	 	 	 }
	 	 	 if hasContent {
	 	 	 	 parts = append(parts, genai.Text(m.GetContent()))
	 	 	 }
	 	 	 if !hasContent && !hasToolCalls {
	 	 	 	 log.Println("Warning: AIMessage has neither content nor tool calls.")
	 	 	 	 // Gemini requires non-empty parts, add placeholder?
	 	 	 	 parts = append(parts, genai.Text(""))
	 	 	 }
	 	 case *schema.ToolMessage:
	 	 	 role = "function" // Gemini uses "function" role for tool results
	 	 	 parts = append(parts, &genai.FunctionResponse{
	 	 	 	 Name: m.ToolCallID, // Gemini FunctionResponse uses Name field for the *original call name* (or ID? Needs verification)
	 	 	 	 Response: map[string]any{
	 	 	 	 	 "content": m.GetContent(),
	 	 	 	 	 // TODO: Add error indication if schema supports it
	 	 	 	 },
	 	 	 })
	 	 default:
	 	 	 log.Printf("Warning: Skipping message of unknown type %T for Gemini conversion.\n", msg)
	 	 	 continue
	 	 }

	 	 // Combine parts into content, handling role changes
	 	 if currentContent == nil || currentContent.Role != role {
	 	 	 // If previous content exists, add it to history
	 	 	 if currentContent != nil {
	 	 	 	 history = append(history, currentContent)
	 	 	 }
	 	 	 // Start new content block
	 	 	 currentContent = &genai.Content{Role: role, Parts: parts}
	 	 } else {
	 	 	 // Append parts to existing content block of the same role
	 	 	 currentContent.Parts = append(currentContent.Parts, parts...)
	 	 }
	 }

	 // Add the last content block to history
	 if currentContent != nil {
	 	 history = append(history, currentContent)
	 }

	 if len(history) == 0 && systemInstruction == nil {
	 	 return nil, nil, errors.New("no valid messages provided for Gemini conversion")
	 }

	 return history, systemInstruction, nil
}

// mapGeminiResponseToAIMessage converts a Gemini response to a Beluga AI message.
func mapGeminiResponseToAIMessage(resp *genai.GenerateContentResponse) (schema.Message, error) {
	 if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
	 	 // Check for safety blocks
	 	 if resp != nil && len(resp.Candidates) > 0 && resp.Candidates[0].FinishReason == genai.FinishReasonSafety {
	 	 	 blockReason := "blocked due to safety settings"
	 	 	 if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != genai.BlockReasonUnspecified {
	 	 	 	 blockReason = resp.PromptFeedback.BlockReason.String() // Use BlockReason.String()
	 	 	 }
	 	 	 return schema.NewAIMessage(fmt.Sprintf("Content blocked: %s", blockReason)), nil
	 	 }
	 	 return nil, errors.New("received empty or invalid response from Gemini")
	 }

	 candidate := resp.Candidates[0]
	 content := candidate.Content
	 responseText := ""
	 toolCalls := []schema.ToolCall{}

	 for _, part := range content.Parts {
	 	 switch p := part.(type) {
	 	 case genai.Text:
	 	 	 responseText += string(p)
	 	 case *genai.FunctionCall:
	 	 	 // Extract raw arguments if we wrapped them
	 	 	 argsStr := "{}"
	 	 	 if rawArgs, ok := p.Args["_beluga_raw_args"].(string); ok {
	 	 	 	 argsStr = rawArgs
	 	 	 } else {
	 	 	 	 // Attempt to marshal the args map back to JSON
	 	 	 	 argsBytes, err := json.Marshal(p.Args)
	 	 	 	 if err == nil {
	 	 	 	 	 argsStr = string(argsBytes)
	 	 	 	 } else {
	 	 	 	 	 log.Printf("Warning: Failed to marshal Gemini FunctionCall args: %v", err)
	 	 	 	 }
	 	 	 }
	 	 	 toolCalls = append(toolCalls, schema.ToolCall{
	 	 	 	 // Gemini doesn't seem to provide a unique ID for the call itself in the response?
	 	 	 	 // We might need to generate one or use the function name + index.
	 	 	 	 // Using function name for now, might need adjustment.
	 	 	 	 ID:        p.Name, // Placeholder ID
	 	 	 	 Name:      p.Name,
	 	 	 	 Arguments: argsStr,
	 	 	 })
	 	 default:
	 	 	 log.Printf("Warning: Unhandled Gemini response part type: %T", p)
	 	 }
	 }

	 aiMsg := schema.NewAIMessage(responseText)
	 if len(toolCalls) > 0 {
	 	 aiMsg.ToolCalls = toolCalls
	 }

	// Add usage and finish reason
	usageMap := make(map[string]int)
	hasUsageData := false

	if resp.UsageMetadata != nil {
		inputTokens := resp.UsageMetadata.PromptTokenCount
		totalTokens := resp.UsageMetadata.TotalTokenCount
		outputTokens := 0
		if totalTokens >= inputTokens {
			outputTokens = int(totalTokens - inputTokens) // Cast to int
		} else {
			log.Printf("Warning: Gemini UsageMetadata.PromptTokenCount (%d) > TotalTokenCount (%d). Setting output_tokens to 0.", inputTokens, totalTokens)
			// If prompt tokens are reported higher than total, output is effectively 0 for this calculation.
			// Total tokens remains as reported by API, even if inconsistent.
		}

		usageMap["input_tokens"] = int(inputTokens) // Cast to int
		usageMap["output_tokens"] = outputTokens
		usageMap["total_tokens"] = int(totalTokens) // Cast to int
		hasUsageData = true
	} else if candidate.TokenCount > 0 { // Fallback if UsageMetadata is not available
		// This is less accurate as we only have output tokens from candidate.
		usageMap["input_tokens"] = 0 // Input tokens unknown from candidate data
		usageMap["output_tokens"] = int(candidate.TokenCount)
		usageMap["total_tokens"] = int(candidate.TokenCount) // Total is approximated as output, as input is unknown
		hasUsageData = true
	}

	if hasUsageData {
		aiMsg.AdditionalArgs["usage"] = usageMap
	}
	aiMsg.AdditionalArgs["finish_reason"] = candidate.FinishReason.String() // Corrected line

	 return aiMsg, nil
}

// mapGeminiStreamChunkToAIMessageChunk converts a Gemini stream response chunk.
func mapGeminiStreamChunkToAIMessageChunk(resp *genai.GenerateContentResponse) (llms.AIMessageChunk, error) {
	 chunk := llms.AIMessageChunk{}

	 // Check for prompt feedback / blocking first
	 if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != genai.BlockReasonUnspecified {
	 	 blockReason := resp.PromptFeedback.BlockReason.String()
	 	 // Removed check for BlockReasonMessage as it does not exist
	 	 chunk.Err = fmt.Errorf("prompt blocked by safety settings: %s", blockReason)
	 	 return chunk, chunk.Err
	 }

	 if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
	 	 // Check for safety finish reason
	 	 if len(resp.Candidates) > 0 && resp.Candidates[0].FinishReason == genai.FinishReasonSafety {
	 	 	 chunk.Err = fmt.Errorf("content generation stopped due to safety settings")
	 	 	 return chunk, chunk.Err
	 	 }
	 	 // May be an empty chunk or end of stream, not necessarily an error
	 	 return chunk, nil
	 }

	 candidate := resp.Candidates[0]
	 content := candidate.Content
	 toolCallChunks := []schema.ToolCallChunk{}

	 for _, part := range content.Parts {
	 	 switch p := part.(type) {
	 	 case genai.Text:
	 	 	 chunk.Content += string(p)
	 	 case *genai.FunctionCall:
	 	 	 // Gemini stream might send FunctionCall parts incrementally or whole?
	 	 	 // Assuming whole for now, might need adjustment based on observed behavior.
	 	 	 argsStr := "{}"
	 	 	 if rawArgs, ok := p.Args["_beluga_raw_args"].(string); ok {
	 	 	 	 argsStr = rawArgs
	 	 	 } else {
	 	 	 	 argsBytes, err := json.Marshal(p.Args)
	 	 	 	 if err == nil {
	 	 	 	 	 argsStr = string(argsBytes)
	 	 	 	 } else {
	 	 	 	 	 log.Printf("Warning: Failed to marshal streaming Gemini FunctionCall args: %v", err)
	 	 	 	 }
	 	 	 }
	 	 	 nameCopy := p.Name
	 	 	 idCopy := p.Name // Placeholder ID
	 	 	 toolCallChunks = append(toolCallChunks, schema.ToolCallChunk{
	 	 	 	 ID:        idCopy,      // Corrected: remove & if field is string
	 	 	 	 Name:      &nameCopy,   // Assuming Name is *string as no error was reported for it
	 	 	 	 Arguments: argsStr,     // Corrected: remove & if field is string
	 	 	 	 // Index might be needed if calls are chunked
	 	 	 })
	 	 default:
	 	 	 log.Printf("Warning: Unhandled Gemini stream part type: %T", p)
	 	 }
	 }

	 if len(toolCallChunks) > 0 {
	 	 chunk.ToolCallChunks = toolCallChunks
	 }

	 // Add finish reason if present in the chunk
	 if candidate.FinishReason != genai.FinishReasonUnspecified {
	 	 chunk.AdditionalArgs = map[string]any{
	 	 	 "finish_reason": candidate.FinishReason.String(),
	 	 }
	 }
	 // TODO: Add usage metadata if available in stream chunks

	 return chunk, nil
}

// mapToolsToGemini converts Beluga tools to Gemini tool format.
func mapToolsToGemini(toolsToBind []tools.Tool) ([]*genai.Tool, error) {
	 geminiTools := make([]*genai.Tool, 0, len(toolsToBind))
	 for _, t := range toolsToBind {
		toolDef := t.Definition() // Use Definition() to get ToolDefinition
	 	 decl := &genai.FunctionDeclaration{
	 	 	 Name:        toolDef.Name, // Access Name from ToolDefinition
	 	 	 Description: toolDef.Description, // Access Description from ToolDefinition
	 	 }

		// Access InputSchema from ToolDefinition, which is map[string]any
		if toolDef.InputSchema != nil {
			// Convert map[string]any to genai.Schema
			// This requires careful mapping or assuming a compatible structure.
			// For simplicity, let's assume direct marshalling/unmarshalling if compatible
			// or specific field mapping if needed.
			schemaBytes, err := json.Marshal(toolDef.InputSchema)
			if err != nil {
				log.Printf("Warning: Failed to marshal schema for tool 	%s	 for Gemini binding: %v. Skipping parameters.", toolDef.Name, err)
			} else {
				var openAPISchema genai.Schema
				err = json.Unmarshal(schemaBytes, &openAPISchema)
				if err != nil {
					log.Printf("Warning: Failed to unmarshal schema into genai.Schema for tool 	%s	: %v. Skipping parameters.", toolDef.Name, err)
				} else {
					// Basic validation: Ensure type is object if properties exist
					if openAPISchema.Type == genai.TypeUnspecified && len(openAPISchema.Properties) > 0 {
						openAPISchema.Type = genai.TypeObject
					}
					decl.Parameters = &openAPISchema
				}
			}
		}

	 	 geminiTools = append(geminiTools, &genai.Tool{
	 	 	 FunctionDeclarations: []*genai.FunctionDeclaration{decl},
	 	 })
	 }
	 return geminiTools, nil
}

// --- llms.ChatModel Implementation ---

// Generate implements the llms.ChatModel interface.
func (gc *GeminiChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	 history, systemInstruction, err := mapMessagesToGeminiContent(messages)
	 if err != nil {
	 	 return nil, fmt.Errorf("failed to map messages for Gemini: %w", err)
	 }

	 model := gc.client.GenerativeModel(gc.modelName)
	 model.Tools = gc.boundTools
	 model.SystemInstruction = systemInstruction
	 model.GenerationConfig = *gc.defaultGenConfig // Copy defaults
	 model.SafetySettings = gc.defaultSafety

	 // Apply call-specific options to GenerationConfig
	 configMap := make(map[string]any)
	 for _, opt := range options {
	 	 opt.Apply(&configMap)
	 }
	 if temp, ok := configMap["temperature"].(float32); ok {
	 	 model.GenerationConfig.Temperature = &temp
	 }
	 if topP, ok := configMap["top_p"].(float32); ok {
	 	 model.GenerationConfig.TopP = &topP
	 }
	 if topK, ok := configMap["top_k"].(int32); ok {
	 	 model.GenerationConfig.TopK = &topK
	 }
	 if maxTokens, ok := configMap["max_tokens"].(int32); ok {
	 	 model.GenerationConfig.MaxOutputTokens = &maxTokens
	 }
	 if stops, ok := configMap["stop_sequences"].([]string); ok {
	 	 model.GenerationConfig.StopSequences = stops
	 }
	 // TODO: Apply safety settings overrides from options if needed

	 cs := model.StartChat()
	 cs.History = history // Set the mapped history

	 // Send the last message (which is implicitly the user prompt)
	 // The history already contains all but the last message if mapped correctly.
	 // Gemini SDK expects the prompt in SendMessage, not just history.
	 if len(history) == 0 {
	 	 return nil, errors.New("cannot send empty message history to Gemini")
	 }
	 lastMessageContent := history[len(history)-1]

	 resp, err := cs.SendMessage(ctx, lastMessageContent.Parts...)
	 if err != nil {
	 	 return nil, fmt.Errorf("gemini SendMessage failed: %w", err)
	 }

	 return mapGeminiResponseToAIMessage(resp)
}

// StreamChat implements the llms.ChatModel interface.
func (gc *GeminiChat) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llms.AIMessageChunk, error) {
	 history, systemInstruction, err := mapMessagesToGeminiContent(messages)
	 if err != nil {
	 	 return nil, fmt.Errorf("failed to map messages for Gemini stream: %w", err)
	 }

	 model := gc.client.GenerativeModel(gc.modelName)
	 model.Tools = gc.boundTools
	 model.SystemInstruction = systemInstruction
	 model.GenerationConfig = *gc.defaultGenConfig // Copy defaults
	 model.SafetySettings = gc.defaultSafety

	 // Apply call-specific options (similar to Generate)
	 configMap := make(map[string]any)
	 for _, opt := range options {
	 	 opt.Apply(&configMap)
	 }
	 if temp, ok := configMap["temperature"].(float32); ok {
	 	 model.GenerationConfig.Temperature = &temp
	 }
	 if topP, ok := configMap["top_p"].(float32); ok {
	 	 model.GenerationConfig.TopP = &topP
	 }
	 if topK, ok := configMap["top_k"].(int32); ok {
	 	 model.GenerationConfig.TopK = &topK
	 }
	 if maxTokens, ok := configMap["max_tokens"].(int32); ok {
	 	 model.GenerationConfig.MaxOutputTokens = &maxTokens
	 }
	 if stops, ok := configMap["stop_sequences"].([]string); ok {
	 	 model.GenerationConfig.StopSequences = stops
	 }

	 cs := model.StartChat()
	 cs.History = history // Set history

	 if len(history) == 0 {
	 	 return nil, errors.New("cannot send empty message history to Gemini stream")
	 }
	 lastMessageContent := history[len(history)-1]

	 iter := cs.SendMessageStream(ctx, lastMessageContent.Parts...)
	 chunkChan := make(chan llms.AIMessageChunk, 1)

	 go func() {
	 	 defer close(chunkChan)
	 	 for {
	 	 	 resp, err := iter.Next()
	 	 	 if err == iterator.Done {
	 	 	 	 return
	 	 	 }
	 	 	 if err != nil {
	 	 	 	 log.Printf("Gemini stream iteration error: %v", err)
	 	 	 	 chunkChan <- llms.AIMessageChunk{Err: fmt.Errorf("gemini stream error: %w", err)}
	 	 	 	 return
	 	 	 }

	 	 	 chunk, mapErr := mapGeminiStreamChunkToAIMessageChunk(resp)
	 	 	 if mapErr != nil {
	 	 	 	 // Send error as a chunk
	 	 	 	 chunkChan <- llms.AIMessageChunk{Err: mapErr}
	 	 	 	 // Decide whether to stop stream on mapping error? For now, yes.
	 	 	 	 return
	 	 	 }

	 	 	 select {
	 	 	 case chunkChan <- chunk:
	 	 	 case <-ctx.Done():
	 	 	 	 log.Println("Gemini stream cancelled by context")
	 	 	 	 chunkChan <- llms.AIMessageChunk{Err: ctx.Err()}
	 	 	 	 return
	 	 	 }
	 	 }
	 }()

	 return chunkChan, nil
}

// BindTools implements the llms.ChatModel interface.
func (gc *GeminiChat) BindTools(toolsToBind []tools.Tool) llms.ChatModel {
	 boundClient := *gc // Create a shallow copy
	 geminiTools, err := mapToolsToGemini(toolsToBind)
	 if err != nil {
	 	 // Log the error but maybe don't fail? Or return original?
	 	 log.Printf("Error mapping tools for Gemini binding: %v. Tools will not be bound.", err)
	 	 boundClient.boundTools = nil
	 } else {
	 	 boundClient.boundTools = geminiTools
	 }
	 return &boundClient
}

// --- core.Runnable Implementation ---

// Invoke implements the core.Runnable interface.
func (gc *GeminiChat) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	 messages, err := llms.EnsureMessages(input)
	 if err != nil {
	 	 return nil, err
	 }
	 return gc.Generate(ctx, messages, options...)
}

// Batch implements the core.Runnable interface.
func (gc *GeminiChat) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	 results := make([]any, len(inputs))
	 errors := make([]error, len(inputs))
	 var wg sync.WaitGroup
	 sem := make(chan struct{}, gc.maxConcurrentBatches)

	 for i, input := range inputs {
	 	 wg.Add(1)
	 	 sem <- struct{}{} // Acquire semaphore
	 	 go func(index int, currentInput any) {
	 	 	 defer wg.Done()
	 	 	 defer func() { <-sem }() // Release semaphore

	 	 	 currentOptions := options // TODO: Handle per-request options if needed
	 	 	 result, err := gc.Invoke(ctx, currentInput, currentOptions...)
	 	 	 results[index] = result
	 	 	 errors[index] = err
	 	 }(i, input)
	 }

	 wg.Wait()

	 // Combine errors
	 var combinedError error
	 for _, err := range errors {
	 	 if err != nil {
	 	 	 if combinedError == nil {
	 	 	 	 combinedError = fmt.Errorf("batch errors: %w", err)
	 	 	 } else {
	 	 	 	 combinedError = fmt.Errorf("%w; %w", combinedError, err)
	 	 	 }
	 	 }
	 }
	 return results, combinedError
}

// Stream implements the core.Runnable interface.
func (gc *GeminiChat) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	 messages, err := llms.EnsureMessages(input)
	 if err != nil {
	 	 return nil, err
	 }

	 chunkChan, err := gc.StreamChat(ctx, messages, options...)
	 if err != nil {
	 	 return nil, err
	 }

	 // Adapt the llms.AIMessageChunk channel to an any channel
	 outputChan := make(chan any, 1)
	 go func() {
	 	 defer close(outputChan)
	 	 for chunk := range chunkChan {
	 	 	 if chunk.Err != nil {
	 	 	 	 select {
	 	 	 	 case outputChan <- chunk.Err:
	 	 	 	 case <-ctx.Done():
	 	 	 	 }
	 	 	 	 return // Stop streaming on error
	 	 	 }
	 	 	 select {
	 	 	 case outputChan <- chunk:
	 	 	 case <-ctx.Done():
	 	 	 	 return
	 	 	 }
	 	 }
	 }()

	 return outputChan, nil
}

