// Package schema defines the shared types used throughout the Beluga AI framework.
//
// It contains message types for conversations, multimodal content parts,
// tool call and result types, document types for RAG, streaming event types,
// and session types. This package has zero external dependencies beyond the
// standard library and contains no business logic — only type definitions
// and constructors.
//
// # Messages
//
// The [Message] interface represents a single message in a conversation.
// Concrete implementations cover all conversation roles:
//
//   - [SystemMessage] — system-level instructions that set model behavior
//   - [HumanMessage] — messages from human users
//   - [AIMessage] — model-generated responses with optional tool calls and usage stats
//   - [ToolMessage] — results from tool executions, correlated by tool call ID
//
// Convenience constructors create messages with a single text part:
//
//	sys := schema.NewSystemMessage("You are a helpful assistant.")
//	msg := schema.NewHumanMessage("What is the weather today?")
//	ai  := schema.NewAIMessage("The weather is sunny.")
//	tool := schema.NewToolMessage(callID, `{"temp": 72}`)
//
// # Content Parts
//
// Messages carry multimodal content via the [ContentPart] interface.
// Implementations support text, images, audio, video, and files:
//
//   - [TextPart] — plain text content
//   - [ImagePart] — image data (inline bytes or URL) with MIME type
//   - [AudioPart] — audio data with format and sample rate
//   - [VideoPart] — video data (inline bytes or URL) with MIME type
//   - [FilePart] — generic file attachments with name and MIME type
//
// # Tool Types
//
// [ToolCall] represents a model's request to invoke a tool, carrying the
// tool name and JSON-encoded arguments. [ToolResult] carries the tool's
// output back to the model. [ToolDefinition] describes a tool's interface
// with name, description, and JSON Schema for the model to use.
//
// # Documents
//
// [Document] is the primary unit for RAG (Retrieval-Augmented Generation)
// pipelines, carrying text content, metadata for filtering, optional
// relevance scores from retrieval, and optional embedding vectors.
//
// # Streaming Events
//
// [StreamChunk] represents incremental pieces of a streaming model response
// with text deltas, tool call updates, finish reasons, and token usage.
// [AgentEvent] represents discrete events emitted during agent execution
// such as thoughts, tool calls, and handoffs.
//
// # Sessions
//
// [Session] tracks a full conversation as an ordered sequence of [Turn]
// values, each containing an input message, output message, timestamp,
// and metadata. Sessions maintain arbitrary state across turns.
//
// # Token Usage
//
// [Usage] tracks token consumption for model responses, including input
// tokens, output tokens, total tokens, and cached tokens.
package schema
