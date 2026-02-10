// Package openaicompat provides a shared ChatModel implementation for providers
// that use OpenAI-compatible APIs. This includes OpenAI itself, as well as providers
// like Groq, Together, Fireworks, xAI, DeepSeek, and others that expose the same
// REST endpoint format.
//
// This is an internal package and is not part of the public API. It is the shared
// foundation used by 12+ thin wrapper LLM provider packages, eliminating duplicated
// conversion and streaming logic.
//
// # Model
//
// The [Model] type implements the llm.ChatModel interface using the openai-go SDK.
// Providers create a Model by calling [New] or [NewWithOptions] with their specific
// base URL and API key, then register it in the llm registry:
//
//	func init() {
//	    llm.Register("groq", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
//	        cfg.BaseURL = "https://api.groq.com/openai/v1"
//	        return openaicompat.New(cfg)
//	    })
//	}
//
// # Message Conversion
//
// [ConvertMessages] translates Beluga schema.Message types (SystemMessage,
// HumanMessage, AIMessage, ToolMessage) into OpenAI API format. It supports
// multimodal content including text and image parts.
//
// [ConvertResponse] translates OpenAI ChatCompletion responses back into
// Beluga schema.AIMessage, including tool calls and usage statistics.
//
// # Tool Conversion
//
// [ConvertTools] translates Beluga schema.ToolDefinition slices into OpenAI
// tool parameters for function calling.
//
// # Streaming
//
// [StreamToSeq] converts an openai-go SSE stream into a Beluga
// iter.Seq2[schema.StreamChunk, error] iterator, handling text deltas,
// tool call accumulation, finish reasons, and token usage.
package openaicompat
