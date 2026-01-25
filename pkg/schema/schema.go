// Package schema provides core data structures and interfaces for the Beluga AI Framework.
// It defines types for messages, documents, configurations, and agent interactions.
//
// The package follows the Beluga AI Framework design patterns with:
// - Interface segregation principle (small, focused interfaces)
// - Dependency inversion principle (high-level modules depend on abstractions)
// - Factory pattern for creating instances
// - Functional options for configuration
// - OpenTelemetry integration for observability
package schema

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema/internal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Core types and interfaces

// Re-export types from iface and internal packages.
type (
	// Interfaces from iface.
	Message     = iface.Message
	ChatHistory = iface.ChatHistory

	// Types from internal.
	Document             = internal.Document
	AgentAction          = internal.AgentAction
	AgentObservation     = internal.AgentObservation
	Step                 = internal.Step
	FinalAnswer          = internal.FinalAnswer
	AgentFinish          = internal.AgentFinish
	AgentScratchPadEntry = internal.AgentScratchPadEntry
	ToolCall             = iface.ToolCall
	ToolCallChunk        = internal.ToolCallChunk
	FunctionCall         = iface.FunctionCall
	Generation           = internal.Generation
	LLMResponse          = internal.LLMResponse
	BaseChatHistory      = internal.BaseChatHistory
	ChatHistoryConfig    = internal.ChatHistoryConfig
	CallOptions          = internal.CallOptions
	LLMOption            = internal.LLMOption
	ChatMessage          = internal.ChatMessage
	AIMessage            = internal.AIMessage
	ToolMessage          = internal.ToolMessage

	// A2A Communication Types.
	AgentMessage     = internal.AgentMessage
	AgentMessageType = internal.AgentMessageType
	AgentRequest     = internal.AgentRequest
	AgentResponse    = internal.AgentResponse
	AgentError       = internal.AgentError

	// Event Types.
	Event                   = internal.Event
	AgentLifecycleEvent     = internal.AgentLifecycleEvent
	AgentLifecycleEventType = internal.AgentLifecycleEventType
	TaskEvent               = internal.TaskEvent
	TaskEventType           = internal.TaskEventType
	WorkflowEvent           = internal.WorkflowEvent
	WorkflowEventType       = internal.WorkflowEventType
)

// Re-export MessageType from iface.
type MessageType = iface.MessageType

// Re-export constants from iface.
const (
	RoleHuman     = iface.RoleHuman
	RoleAssistant = iface.RoleAssistant
	RoleSystem    = iface.RoleSystem
	RoleTool      = iface.RoleTool
	RoleFunction  = iface.RoleFunction
)

// Re-export A2A Communication constants.
const (
	AgentMessageRequest      = internal.AgentMessageRequest
	AgentMessageResponse     = internal.AgentMessageResponse
	AgentMessageNotification = internal.AgentMessageNotification
	AgentMessageBroadcast    = internal.AgentMessageBroadcast
	AgentMessageError        = internal.AgentMessageError
)

// Re-export Event constants.
const (
	AgentStarted       = internal.AgentStarted
	AgentStopped       = internal.AgentStopped
	AgentPaused        = internal.AgentPaused
	AgentResumed       = internal.AgentResumed
	AgentFailed        = internal.AgentFailed
	AgentConfigUpdated = internal.AgentConfigUpdated

	TaskStarted   = internal.TaskStarted
	TaskProgress  = internal.TaskProgress
	TaskCompleted = internal.TaskCompleted
	TaskFailed    = internal.TaskFailed
	TaskCancelled = internal.TaskCancelled

	WorkflowStarted       = internal.WorkflowStarted
	WorkflowStepCompleted = internal.WorkflowStepCompleted
	WorkflowCompleted     = internal.WorkflowCompleted
	WorkflowFailed        = internal.WorkflowFailed
	WorkflowCancelled     = internal.WorkflowCancelled
)

// Factory functions for creating messages

// NewHumanMessage creates a new human message.
// Human messages represent user input in a conversation.
//
// Parameters:
//   - content: The text content of the human message
//
// Returns:
//   - Message: A new human message instance
//
// Example:
//
//	msg := schema.NewHumanMessage("What is machine learning?")
//
// Example usage can be found in examples/schema/basic/main.go.
func NewHumanMessage(content string) Message {
	return &internal.ChatMessage{
		BaseMessage: internal.BaseMessage{Content: content},
		Role:        RoleHuman,
	}
}

// NewAIMessage creates a new AI message.
// AI messages represent assistant responses in a conversation.
//
// Parameters:
//   - content: The text content of the AI message
//
// Returns:
//   - Message: A new AI message instance
//
// Example:
//
//	msg := schema.NewAIMessage("Machine learning is a subset of AI.")
//
// Example usage can be found in examples/schema/basic/main.go.
func NewAIMessage(content string) Message {
	return &internal.AIMessage{
		BaseMessage: internal.BaseMessage{Content: content},
	}
}

// NewSystemMessage creates a new system message.
// System messages provide instructions or context to the AI model.
//
// Parameters:
//   - content: The text content of the system message
//
// Returns:
//   - Message: A new system message instance
//
// Example:
//
//	msg := schema.NewSystemMessage("You are a helpful assistant.")
//
// Example usage can be found in examples/schema/basic/main.go.
func NewSystemMessage(content string) Message {
	return &internal.ChatMessage{
		BaseMessage: internal.BaseMessage{Content: content},
		Role:        RoleSystem,
	}
}

// NewToolMessage creates a new tool message.
// Tool messages contain the results of tool execution for the AI model.
//
// Parameters:
//   - content: The text content of the tool message (tool execution result)
//   - toolCallID: The ID of the tool call this message corresponds to
//
// Returns:
//   - Message: A new tool message instance
//
// Example:
//
//	msg := schema.NewToolMessage("42", "call_123")
//
// Example usage can be found in examples/schema/basic/main.go.
func NewToolMessage(content, toolCallID string) Message {
	return &internal.ToolMessage{
		BaseMessage: internal.BaseMessage{Content: content},
		ToolCallID:  toolCallID,
	}
}

// NewFunctionMessage creates a new function message.
// Function messages represent function call results in function calling workflows.
//
// Parameters:
//   - name: The name of the function that was called
//   - content: The result content from the function execution
//
// Returns:
//   - Message: A new function message instance
//
// Example:
//
//	msg := schema.NewFunctionMessage("calculate", "42")
//
// Example usage can be found in examples/schema/basic/main.go.
func NewFunctionMessage(name, content string) Message {
	return &internal.FunctionMessage{
		BaseMessage: internal.BaseMessage{Content: content},
		Name:        name,
	}
}

// NewChatMessage creates a new chat message with specified role.
// This is a generic factory function that can create messages of any role type.
//
// Parameters:
//   - role: The message role (RoleHuman, RoleAssistant, RoleSystem, etc.)
//   - content: The text content of the message
//
// Returns:
//   - Message: A new chat message instance with the specified role
//
// Example:
//
//	msg := schema.NewChatMessage(schema.RoleHuman, "Hello")
//
// Example usage can be found in examples/schema/basic/main.go.
func NewChatMessage(role MessageType, content string) Message {
	return &internal.ChatMessage{
		BaseMessage: internal.BaseMessage{Content: content},
		Role:        role,
	}
}

// Factory functions for creating documents

// NewDocument creates a new Document.
// Documents are used for storing text content with metadata in RAG pipelines.
//
// Parameters:
//   - pageContent: The main text content of the document
//   - metadata: Optional metadata map (source, author, topic, etc.)
//
// Returns:
//   - Document: A new document instance
//
// Example:
//
//	doc := schema.NewDocument(
//	    "Machine learning is a subset of AI.",
//	    map[string]string{"source": "textbook", "topic": "AI"},
//	)
//
// Example usage can be found in examples/schema/basic/main.go.
func NewDocument(pageContent string, metadata map[string]string) Document {
	return internal.NewDocument(pageContent, metadata)
}

// NewDocumentWithID creates a new Document with an ID.
// This is useful when you need to track documents by a specific identifier.
//
// Parameters:
//   - id: Unique identifier for the document
//   - pageContent: The main text content of the document
//   - metadata: Optional metadata map
//
// Returns:
//   - Document: A new document instance with the specified ID
//
// Example:
//
//	doc := schema.NewDocumentWithID(
//	    "doc_123",
//	    "Machine learning is a subset of AI.",
//	    map[string]string{"source": "textbook"},
//	)
//
// Example usage can be found in examples/schema/basic/main.go.
func NewDocumentWithID(id, pageContent string, metadata map[string]string) Document {
	return internal.NewDocumentWithID(id, pageContent, metadata)
}

// NewDocumentWithEmbedding creates a new Document with an embedding.
// This is useful when you have pre-computed embeddings for the document.
//
// Parameters:
//   - pageContent: The main text content of the document
//   - metadata: Optional metadata map
//   - embedding: Pre-computed embedding vector for the document
//
// Returns:
//   - Document: A new document instance with the embedding
//
// Example:
//
//	embedding := []float32{0.1, 0.2, 0.3, ...} // 768-dimensional vector
//	doc := schema.NewDocumentWithEmbedding(
//	    "Machine learning is a subset of AI.",
//	    map[string]string{"source": "textbook"},
//	    embedding,
//	)
//
// Example usage can be found in examples/schema/basic/main.go.
func NewDocumentWithEmbedding(pageContent string, metadata map[string]string, embedding []float32) Document {
	doc := internal.NewDocument(pageContent, metadata)
	doc.Embedding = embedding
	return doc
}

// Factory functions for chat history

// NewBaseChatHistory creates a new BaseChatHistory.
func NewBaseChatHistory(opts ...ChatHistoryOption) (ChatHistory, error) {
	config, err := NewChatHistoryConfig(opts...)
	if err != nil {
		return nil, err
	}

	return internal.NewBaseChatHistory((*internal.ChatHistoryConfig)(config)), nil
}

// Agent I/O factory functions

// NewAgentAction creates a new AgentAction.
func NewAgentAction(tool string, toolInput any, log string) AgentAction {
	return internal.AgentAction{
		Tool:      tool,
		ToolInput: toolInput,
		Log:       log,
	}
}

// NewAgentObservation creates a new AgentObservation.
func NewAgentObservation(actionLog, output string, parsedOutput any) AgentObservation {
	return internal.AgentObservation{
		ActionLog:    actionLog,
		Output:       output,
		ParsedOutput: parsedOutput,
	}
}

// NewStep creates a new Step with an action and observation.
func NewStep(action AgentAction, observation AgentObservation) Step {
	return internal.Step{
		Action:      action,
		Observation: observation,
	}
}

// NewFinalAnswer creates a new FinalAnswer.
func NewFinalAnswer(output string, sourceDocuments []any, intermediateSteps []Step) FinalAnswer {
	return internal.FinalAnswer{
		Output:            output,
		SourceDocuments:   sourceDocuments,
		IntermediateSteps: intermediateSteps,
	}
}

// NewAgentFinish creates a new AgentFinish.
func NewAgentFinish(returnValues map[string]any, log string) AgentFinish {
	return internal.AgentFinish{
		ReturnValues: returnValues,
		Log:          log,
	}
}

// LLM call options factory functions

// NewCallOptions creates a new CallOptions with default values.
func NewCallOptions() *CallOptions {
	return &CallOptions{
		ProviderSpecificArgs: make(map[string]any),
	}
}

// WithTemperature sets the temperature for LLM calls.
func WithTemperature(temp float64) LLMOption {
	return func(o *CallOptions) {
		o.Temperature = &temp
	}
}

// WithMaxTokens sets the max tokens for LLM calls.
func WithMaxTokens(maxTokens int) LLMOption {
	return func(o *CallOptions) {
		o.MaxTokens = &maxTokens
	}
}

// WithTopP sets the TopP for LLM calls.
func WithTopP(topP float64) LLMOption {
	return func(o *CallOptions) {
		o.TopP = &topP
	}
}

// WithFrequencyPenalty sets the frequency penalty for LLM calls.
func WithFrequencyPenalty(penalty float64) LLMOption {
	return func(o *CallOptions) {
		o.FrequencyPenalty = &penalty
	}
}

// WithPresencePenalty sets the presence penalty for LLM calls.
func WithPresencePenalty(penalty float64) LLMOption {
	return func(o *CallOptions) {
		o.PresencePenalty = &penalty
	}
}

// WithStopSequences sets the stop sequences for LLM calls.
func WithStopSequences(stop []string) LLMOption {
	return func(o *CallOptions) {
		o.Stop = stop
	}
}

// WithStreaming enables or disables streaming for LLM calls.
func WithStreaming(streaming bool) LLMOption {
	return func(o *CallOptions) {
		o.Streaming = streaming
	}
}

// WithProviderSpecificArg adds a provider-specific argument.
func WithProviderSpecificArg(key string, value any) LLMOption {
	return func(o *CallOptions) {
		if o.ProviderSpecificArgs == nil {
			o.ProviderSpecificArgs = make(map[string]any)
		}
		o.ProviderSpecificArgs[key] = value
	}
}

// Generation and response factory functions

// NewGeneration creates a new Generation.
func NewGeneration(text string, message Message, generationInfo map[string]any) *Generation {
	return &internal.Generation{
		Text:           text,
		Message:        message,
		GenerationInfo: generationInfo,
	}
}

// NewLLMResponse creates a new LLMResponse.
func NewLLMResponse(generations [][]*Generation, llmOutput map[string]any) *LLMResponse {
	return &internal.LLMResponse{
		Generations: generations,
		LLMOutput:   llmOutput,
	}
}

// Factory functions for A2A communication

// NewAgentMessage creates a new AgentMessage.
func NewAgentMessage(fromAgentID, messageID string, messageType AgentMessageType, payload any) AgentMessage {
	return internal.AgentMessage{
		FromAgentID: fromAgentID,
		MessageID:   messageID,
		Timestamp:   time.Now().Unix(),
		MessageType: messageType,
		Payload:     payload,
		Metadata:    make(map[string]any),
	}
}

// NewAgentRequest creates a new AgentRequest.
func NewAgentRequest(action string, parameters map[string]any) AgentRequest {
	return internal.AgentRequest{
		Action:     action,
		Parameters: parameters,
	}
}

// NewAgentResponse creates a new AgentResponse.
func NewAgentResponse(requestID, status string, result any) AgentResponse {
	return internal.AgentResponse{
		RequestID: requestID,
		Status:    status,
		Result:    result,
	}
}

// NewAgentError creates a new AgentError.
func NewAgentError(code, message string, details map[string]any) *AgentError {
	return &internal.AgentError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Factory functions for events

// NewEvent creates a new Event.
func NewEvent(eventID, eventType, source string, payload any) Event {
	return internal.Event{
		EventID:   eventID,
		EventType: eventType,
		Source:    source,
		Timestamp: time.Now().Unix(),
		Version:   "1.0",
		Payload:   payload,
		Metadata:  make(map[string]any),
	}
}

// NewAgentLifecycleEvent creates a new AgentLifecycleEvent.
func NewAgentLifecycleEvent(agentID string, eventType AgentLifecycleEventType) AgentLifecycleEvent {
	return internal.AgentLifecycleEvent{
		AgentID:   agentID,
		EventType: eventType,
	}
}

// NewTaskEvent creates a new TaskEvent.
func NewTaskEvent(taskID, agentID string, eventType TaskEventType) TaskEvent {
	return internal.TaskEvent{
		TaskID:    taskID,
		AgentID:   agentID,
		EventType: eventType,
	}
}

// NewWorkflowEvent creates a new WorkflowEvent.
func NewWorkflowEvent(workflowID string, eventType WorkflowEventType) WorkflowEvent {
	return internal.WorkflowEvent{
		WorkflowID: workflowID,
		EventType:  eventType,
	}
}

// Context-aware factory functions (with tracing)

// NewHumanMessageWithContext creates a new human message with tracing context.
func NewHumanMessageWithContext(ctx context.Context, content string) Message {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/schema")
	ctx, span := tracer.Start(ctx, "schema.NewHumanMessage",
		trace.WithAttributes(
			attribute.String("message.type", "human"),
			attribute.Int("content.length", len(content)),
		))
	defer span.End()

	// Record metrics
	RecordMessageCreated(ctx, RoleHuman)

	// Structured logging with OTEL context
	logWithOTELContext(ctx, slog.LevelInfo, "Creating human message",
		"message_type", "human",
		"content_length", len(content))

	msg := NewHumanMessage(content)
	span.SetStatus(codes.Ok, "")
	return msg
}

// NewAIMessageWithContext creates a new AI message with tracing context.
func NewAIMessageWithContext(ctx context.Context, content string) Message {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/schema")
	ctx, span := tracer.Start(ctx, "schema.NewAIMessage",
		trace.WithAttributes(
			attribute.String("message.type", "ai"),
			attribute.Int("content.length", len(content)),
		))
	defer span.End()

	// Record metrics
	RecordMessageCreated(ctx, RoleAssistant)

	// Structured logging with OTEL context
	logWithOTELContext(ctx, slog.LevelInfo, "Creating AI message",
		"message_type", "ai",
		"content_length", len(content))

	msg := NewAIMessage(content)
	span.SetStatus(codes.Ok, "")
	return msg
}

// logWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}

// Validation helpers

// ValidateMessage validates a message implementation.
func ValidateMessage(msg Message) error {
	if msg == nil {
		return errors.New("message cannot be nil")
	}

	if msg.GetType() == "" {
		return errors.New("message type cannot be empty")
	}

	if msg.GetContent() == "" {
		return errors.New("message content cannot be empty")
	}

	return nil
}

// ValidateDocument validates a document.
func ValidateDocument(doc Document) error {
	if doc.PageContent == "" {
		return errors.New("document page content cannot be empty")
	}

	if doc.Metadata == nil {
		return errors.New("document metadata cannot be nil")
	}

	return nil
}

// Ensure interface compliance.
var (
	_ Message     = (*internal.ChatMessage)(nil)
	_ Message     = (*internal.ToolMessage)(nil)
	_ Message     = (*internal.FunctionMessage)(nil)
	_ Message     = (*internal.AIMessage)(nil)
	_ Message     = (*internal.Document)(nil)
	_ ChatHistory = (*internal.BaseChatHistory)(nil)
)
