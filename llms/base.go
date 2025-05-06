// Package llms defines interfaces for interacting with Large Language Models (LLMs)
// and specific chat models.
package llms

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
)

// ChatModel is the interface for chat models.
// It extends the Runnable interface for more complex interactions.
type ChatModel interface {
	core.Runnable
	// Generate takes a series of messages and returns an AI message.
	// It is a single call to the model.
	Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)

	// StreamChat takes a series of messages and returns a channel of AIMessageChunk.
	// This allows for streaming responses from the model.
	StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan AIMessageChunk, error)

	// BindTools binds a list of tools to the ChatModel. The returned ChatModel
	// will then be ableto call these tools.
	// The specific way tools are bound and used depends on the underlying model provider.
	BindTools(toolsToBind []tools.Tool) ChatModel
}

// AIMessageChunk represents a chunk of an AI message, typically used in streaming.
// It can contain content, tool calls, or an error.
type AIMessageChunk struct {
	Content        string // Text content of the chunk
	ToolCallChunks []schema.ToolCallChunk
	AdditionalArgs map[string]interface{} // Provider-specific arguments or metadata
	Err            error                  // Error encountered during streaming for this chunk
}

// EnsureMessages ensures the input is a slice of schema.Message.
// It attempts to convert common input types (like a single string or Message) into the required format.
func EnsureMessages(input any) ([]schema.Message, error) {
	switch v := input.(type) {
	case string:
		return []schema.Message{schema.NewHumanMessage(v)}, nil
	case schema.Message:
		return []schema.Message{v}, nil
	case []schema.Message:
		return v, nil
	default:
		return nil, fmt.Errorf("invalid input type for messages: %T", input)
	}
}

// GetSystemAndHumanPrompts extracts the system prompt and concatenates human messages.
// This is a utility function that might be useful for models that don_t support distinct system messages
// or require a single prompt string.
func GetSystemAndHumanPrompts(messages []schema.Message) (string, string) {
	var systemPrompt string
	var humanPrompts []string
	for _, msg := range messages {
		if msg.GetType() == schema.MessageTypeSystem { 
			systemPrompt = msg.GetContent()
		} else if msg.GetType() == schema.MessageTypeHuman { 
			humanPrompts = append(humanPrompts, msg.GetContent())
		}
	}
	fullHumanPrompt := ""
	for i, p := range humanPrompts {
		if i > 0 {
			fullHumanPrompt += "\n"
		}
		fullHumanPrompt += p
	}
	return systemPrompt, fullHumanPrompt
}

// Option is a function type for setting options for LLM calls.
// It is a wrapper around core.Option for convenience within the llms package.
// Deprecated: Use core.Option directly.
type Option func(map[string]interface{})

// Apply applies the option to the given map.
// Deprecated: Use core.Option directly.
func (o Option) Apply(m *map[string]interface{}) {
	o(*m)
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(tokens int) core.Option {
	return core.WithOption("max_tokens", tokens) 
}

// WithTemperature sets the sampling temperature.
func WithTemperature(temp float32) core.Option {
	return core.WithOption("temperature", temp) 
}

// WithTopP sets the nucleus sampling probability.
func WithTopP(topP float32) core.Option {
	return core.WithOption("top_p", topP) 
}

// WithTopK sets the top-k sampling parameter.
func WithTopK(topK int) core.Option {
	return core.WithOption("top_k", topK) 
}

// WithStopWords sets the stop sequences for generation.
func WithStopWords(stop []string) core.Option {
	return core.WithOption("stop_words", stop) 
}

// WithStreamingFunc sets a callback function for streaming responses.
// The function will be called with each AIMessageChunk.
// Deprecated: Streaming is handled via the StreamChat method and channels.
func WithStreamingFunc(fn func(context.Context, AIMessageChunk) error) core.Option {
	return core.WithOption("streaming_func", fn) 
}

// WithTools sets the tools that the model can call.
func WithTools(toolsToUse []tools.Tool) core.Option { 
    return core.WithOption("tools", toolsToUse) 
}

// WithToolChoice forces the model to call a specific tool or no tool.
// Use tool name to force a specific tool, "any" to allow any tool, "none" to prevent tool use.
func WithToolChoice(choice string) core.Option {
    return core.WithOption("tool_choice", choice) 
}

