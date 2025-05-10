package schema

import (
	"context"
	"encoding/json"
	"fmt"
)

// MessageType defines the type of a message.
type MessageType string

const (
	// AIMessageType is a message from an AI.
	AIMessageType MessageType = "ai"
	// HumanMessageType is a message from a human.
	HumanMessageType MessageType = "human"
	// SystemMessageType is a message from the system.
	SystemMessageType MessageType = "system"
	// ChatMessageType is a message that can be stored in a chat history.
	ChatMessageType MessageType = "chat"
	// FunctionMessageType is a message that represents a function call.
	FunctionMessageType MessageType = "function"
	// ToolMessageType is a message that represents a tool call.
	ToolMessageType MessageType = "tool"
)

// Message is the interface that all message types must implement.
type Message interface {
	GetType() MessageType
	String() string
	GetContent() string
}

// BaseMessage is a struct that provides the base implementation for the Message interface.
// It includes the common fields that all messages should have.
type BaseMessage struct {
	Content string `json:"content"`
}

// NewMessage creates a new message with the given content and type.
// This is a generic constructor. Specific types might have their own dedicated constructors.
func NewMessage(content string, msgType MessageType) Message {
	switch msgType {
	case AIMessageType:
		return &AIMessage{BaseMessage: BaseMessage{Content: content}}
	case HumanMessageType:
		return NewHumanMessage(content)
	case SystemMessageType:
		return &SystemMessage{BaseMessage: BaseMessage{Content: content}}
	case ChatMessageType:
		// For ChatMessage, Role would typically be set separately or via a different constructor
		return &ChatMessage{BaseMessage: BaseMessage{Content: content}}
	case FunctionMessageType:
		// For FunctionMessage, Name would typically be set separately or via a different constructor
		return &FunctionMessage{BaseMessage: BaseMessage{Content: content}}
	case ToolMessageType:
		// For ToolMessage, ToolCallId would typically be set separately or via a different constructor
		// Use NewToolMessage for proper initialization with ToolCallID
		return NewToolMessage(content, "") // Default empty toolCallID if using generic constructor
	default:
		panic(fmt.Sprintf("Unknown message type: %s", msgType))
	}
}

// AIMessage is a message from an AI.
type AIMessage struct {
	BaseMessage
}

func (m *AIMessage) GetType() MessageType { return AIMessageType }
func (m *AIMessage) String() string       { return m.Content }
func (m *AIMessage) GetContent() string   { return m.Content }

// HumanMessage is a message from a human.
type HumanMessage struct {
	BaseMessage
}

// NewHumanMessage creates a new HumanMessage.
func NewHumanMessage(content string) *HumanMessage {
	return &HumanMessage{BaseMessage: BaseMessage{Content: content}}
}

func (m *HumanMessage) GetType() MessageType { return HumanMessageType }
func (m *HumanMessage) String() string       { return m.Content }
func (m *HumanMessage) GetContent() string   { return m.Content }

// SystemMessage is a message from the system.
type SystemMessage struct {
	BaseMessage
}

func (m *SystemMessage) GetType() MessageType { return SystemMessageType }
func (m *SystemMessage) String() string       { return m.Content }
func (m *SystemMessage) GetContent() string   { return m.Content }

// ChatMessage is a message that can be stored in a chat history.
type ChatMessage struct {
	BaseMessage
	Role string `json:"role"`
}

func (m *ChatMessage) GetType() MessageType { return ChatMessageType }
func (m *ChatMessage) String() string       { return fmt.Sprintf("%s: %s", m.Role, m.Content) }
func (m *ChatMessage) GetContent() string   { return m.Content }

// FunctionMessage is a message that represents a function call.
type FunctionMessage struct {
	BaseMessage
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"` // Added arguments field
}

func (m *FunctionMessage) GetType() MessageType { return FunctionMessageType }
func (m *FunctionMessage) String() string       { return fmt.Sprintf("Function call: %s, Args: %v", m.Name, m.Arguments) }
func (m *FunctionMessage) GetContent() string   { return m.Content } // Content might be observation from function call

// ToolMessage is a message that represents a tool call.
type ToolMessage struct {
	BaseMessage
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name,omitempty"` // Optional: name of the tool being called
}

// NewToolMessage creates a new ToolMessage.
func NewToolMessage(content string, toolCallID string) *ToolMessage {
	return &ToolMessage{
		BaseMessage: BaseMessage{Content: content},
		ToolCallID:  toolCallID,
	}
}

func (m *ToolMessage) GetType() MessageType { return ToolMessageType }
func (m *ToolMessage) String() string       { return fmt.Sprintf("Tool call ID: %s, Name: %s, Content: %s", m.ToolCallID, m.Name, m.Content) }
func (m *ToolMessage) GetContent() string   { return m.Content } // Content is often the result of the tool call

// StoredMessage is a message that can be stored in a database.
// It includes the type of the message and the message itself.
type StoredMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// NewStoredMessage creates a new StoredMessage from a Message.
func NewStoredMessage(message Message) (StoredMessage, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return StoredMessage{}, fmt.Errorf("failed to marshal message: %w", err)
	}
	return StoredMessage{
		Type: string(message.GetType()),
		Data: data,
	}, nil
}

// ToMessage converts a StoredMessage back to a Message.
func (sm *StoredMessage) ToMessage() (Message, error) {
	var msg Message
	switch MessageType(sm.Type) {
	case AIMessageType:
		// Check if it could be an AIChatMessage (which also has AIMessageType)
		// This requires a more sophisticated unmarshalling or a distinct type for AIChatMessage in storage.
		// For now, assume it could be either AIMessage or AIChatMessage.
		// A simple way is to try unmarshalling into AIChatMessage first if it has tool_calls.
		// However, sm.Data is already unmarshalled into msg, so this logic is tricky here.
		// Let's assume for now that AIChatMessage is handled by its own registration if needed.
		aiMsg := &AIChatMessage{} // Try AIChatMessage first as it has more fields
		if err := json.Unmarshal(sm.Data, aiMsg); err == nil && (len(aiMsg.ToolCalls) > 0 || aiMsg.Role != "") {
			msg = aiMsg
		} else {
			msg = &AIMessage{}
		}
	case HumanMessageType:
		msg = &HumanMessage{}
	case SystemMessageType:
		msg = &SystemMessage{}
	case ChatMessageType:
		msg = &ChatMessage{}
	case FunctionMessageType:
		msg = &FunctionMessage{}
	case ToolMessageType:
		msg = &ToolMessage{}
	default:
		return nil, fmt.Errorf("unknown message type: %s", sm.Type)
	}

	err := json.Unmarshal(sm.Data, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message data for type %s: %w", sm.Type, err)
	}
	return msg, nil
}

// LLMOption represents a functional option for LLM calls.
// This is the type used by the LLM interface.
type LLMOption func(*LLMCallOptions)

// --- Additional types for rich AI interaction (e.g., with tool usage) ---

// Role represents the role of a participant in a chat.
// These are often used within ChatMessage or similar structures.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	RoleFunction  = "function" // Often used for function call results
)

// ToolCall represents a tool call requested by an AI model.
type ToolCall struct {
	ID       string       `json:"id"`                 // A unique identifier for this tool call
	Type     string       `json:"type"`               // The type of the tool call, typically "function"
	Function ToolFunction `json:"function"`           // The function to be called
}

// ToolFunction specifies the function to be called by a tool.
type ToolFunction struct {
	Name      string `json:"name"`      // The name of the function to call
	Arguments string `json:"arguments"` // A JSON string representing the arguments to the function
}

// AIChatMessage represents a message from the AI, potentially including tool calls.
// This can be used when the AI's response is more structured than a simple string.
type AIChatMessage struct {
	BaseMessage
	Role      string     `json:"role"`                // Should be RoleAssistant
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // Optional: list of tool calls requested by the AI
	// Other AI-specific fields can be added here, e.g., usage metadata.
}

// GetType returns the message type for AIChatMessage.
func (m *AIChatMessage) GetType() MessageType { return AIMessageType } // Or a new MessageType if distinct handling is needed

// String provides a string representation of AIChatMessage.
func (m *AIChatMessage) String() string {
	content := m.Content
	if len(m.ToolCalls) > 0 {
		content += fmt.Sprintf(" (Tool Calls: %d)", len(m.ToolCalls))
	}
	return content
}

// GetContent returns the primary content of the AIChatMessage.
func (m *AIChatMessage) GetContent() string {
	return m.Content
}

// Ensure AIChatMessage implements the Message interface.
var _ Message = (*AIChatMessage)(nil)

// LLMCallOption is a more concrete definition for LLM call options.
// This was the previous name, now LLMOption is the primary type.
// type LLMCallOption func(*LLMCallOptions)

// LLMCallOptions holds common options for an LLM call.
// Providers can extend this or use their own specific option structs.
type LLMCallOptions struct {
	Model            string   `json:"model,omitempty"`             // Specific model to use for this call
	Temperature      *float64 `json:"temperature,omitempty"`      // Sampling temperature
	MaxTokens        *int     `json:"max_tokens,omitempty"`        // Max tokens to generate
	TopP             *float64 `json:"top_p,omitempty"`             // Top-p sampling
	StopSequences    []string `json:"stop_sequences,omitempty"`    // Sequences to stop generation at
	Streaming        bool     `json:"streaming,omitempty"`        // Flag to indicate if streaming is requested
	StreamingFunc    func(ctx context.Context, chunk []byte) error `json:"-"` // Function to handle streaming chunks
	ToolChoice       interface{} `json:"tool_choice,omitempty"`    // OpenAI specific: controls which tool is called. e.g. "auto", "none", {"type": "function", "function": {"name": "my_function"}}
	Tools            []Tool   `json:"tools,omitempty"`            // OpenAI specific: A list of tools the model may call.
	ResponseFormat   interface{} `json:"response_format,omitempty"` // OpenAI specific: e.g. { "type": "json_object" }
	// Add other common options here
}

// Tool is a schema for defining a tool that an LLM can use (OpenAI specific style).
type Tool struct {
	Type     string       `json:"type"`               // Typically "function"
	Function ToolFunctionSchema `json:"function"`
}

// ToolFunctionSchema defines the schema of a function that can be called by an LLM.
type ToolFunctionSchema struct {
	Name        string `json:"name"`                  // The name of the function to be called.
	Description string `json:"description,omitempty"`  // A description of what the function does, used by the model to choose when and how to call the function.
	Parameters  *JSONSchemaProps `json:"parameters,omitempty"` // The parameters the functions accepts, described as a JSON Schema object.
}

// JSONSchemaProps represents properties of a JSON schema.
// This is a simplified version. A full JSON schema definition can be quite complex.
type JSONSchemaProps struct {
	Type       string                     `json:"type"`                 // e.g., "object", "string", "number", "integer", "boolean", "array"
	Properties map[string]JSONSchemaProps `json:"properties,omitempty"` // For type "object"
	Required   []string                   `json:"required,omitempty"`   // For type "object"
	Enum       []interface{}              `json:"enum,omitempty"`       // For string, number, integer
	Description string                    `json:"description,omitempty"`
	// Add other JSON schema fields as needed (e.g., items for array, format for string, etc.)
	Items      *JSONSchemaProps           `json:"items,omitempty"`       // For type "array"
}

// Helper functions to create LLMOption instances

// WithModel sets the model for the LLM call.
func WithModel(model string) LLMOption {
	return func(o *LLMCallOptions) {
		o.Model = model
	}
}

// WithTemperature sets the temperature for the LLM call.
func WithTemperature(temp float64) LLMOption {
	return func(o *LLMCallOptions) {
		o.Temperature = &temp
	}
}

// WithMaxTokens sets the max tokens for the LLM call.
func WithMaxTokens(maxTokens int) LLMOption {
	return func(o *LLMCallOptions) {
		o.MaxTokens = &maxTokens
	}
}

// WithStreaming sets the streaming flag for the LLM call.
func WithStreaming(streaming bool) LLMOption {
	return func(o *LLMCallOptions) {
		o.Streaming = streaming
	}
}

// WithStreamingFunc sets the streaming function for the LLM call.
func WithStreamingFunc(fn func(ctx context.Context, chunk []byte) error) LLMOption {
	return func(o *LLMCallOptions) {
		o.StreamingFunc = fn
		o.Streaming = true // Implicitly enable streaming if a func is provided
	}
}

// WithTools sets the tools for the LLM call (OpenAI specific).
func WithTools(tools []Tool) LLMOption {
	return func(o *LLMCallOptions) {
		o.Tools = tools
	}
}

// WithToolChoice sets the tool_choice for the LLM call (OpenAI specific).
func WithToolChoice(toolChoice interface{}) LLMOption {
	return func(o *LLMCallOptions) {
		o.ToolChoice = toolChoice
	}
}

// WithResponseFormat sets the response_format for the LLM call (OpenAI specific).
func WithResponseFormat(responseFormat interface{}) LLMOption {
	return func(o *LLMCallOptions) {
		o.ResponseFormat = responseFormat
	}
}

// ApplyLLMOptions applies a list of LLMOption to a LLMCallOptions struct.
func ApplyLLMOptions(opts ...LLMOption) *LLMCallOptions {
	callOpts := &LLMCallOptions{}
	for _, opt := range opts {
		opt(callOpts)
	}
	return callOpts
}

