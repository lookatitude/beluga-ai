package schema

// MessageType defines the type of a message.
type MessageType string

// Constants for MessageType
const (
	RoleHuman     MessageType = "human"
	RoleAssistant MessageType = "ai"
	RoleSystem    MessageType = "system"
	RoleTool      MessageType = "tool"
	RoleFunction  MessageType = "function"

	// Deprecated, use Role* constants where specific roles are intended.
	// These are kept for broader type categorization if ever needed but might be removed.
	AIMessageType       MessageType = "ai"
	HumanMessageType    MessageType = "human"
	SystemMessageType   MessageType = "system"
	ChatMessageType     MessageType = "chat" // Generic chat, role specified in ChatMessage
	FunctionMessageType MessageType = "function"
	ToolMessageType     MessageType = "tool"
)

// Message is the interface that all message types must implement.
type Message interface {
	GetType() MessageType
	GetContent() string
	// String() string // Stringer interface, often GetContent or a formatted version
}

// BaseMessage provides common fields for messages.
type BaseMessage struct {
	Content string `json:"content"`
}

// GetContent returns the content of the base message.
func (bm *BaseMessage) GetContent() string {
	return bm.Content
}

// ToolCall represents a call to a tool by the LLM.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // Typically "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function to be called.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string of arguments
}

// ChatMessage represents a message in a chat sequence.
type ChatMessage struct {
	BaseMessage
	Role      MessageType `json:"role"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"` // Used by AIMessage for proposing tool calls
}

// GetType returns the role of the ChatMessage.
func (m *ChatMessage) GetType() MessageType {
	return m.Role
}

// ToolMessage represents the result of a tool invocation.
type ToolMessage struct {
	BaseMessage
	ToolCallID string `json:"tool_call_id"`
}

// GetType returns RoleTool.
func (m *ToolMessage) GetType() MessageType {
	return RoleTool
}

// FunctionMessage represents a message related to a function call.
type FunctionMessage struct {
	BaseMessage
	Name string `json:"name"` // Name of the function that was called
}

// GetType returns RoleFunction.
func (m *FunctionMessage) GetType() MessageType {
	return RoleFunction
}

// AIMessage represents a message from the AI.
// It can include content and a list of tool calls the AI wants to make.
type AIMessage struct {
	BaseMessage
	// Content is inherited from BaseMessage.
	// Role is implicitly RoleAssistant, returned by GetType().
	ToolCalls []ToolCall `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"`
}

// GetType returns the message type, which is always RoleAssistant for AIMessage.
func (m *AIMessage) GetType() MessageType {
	return RoleAssistant
}

// Constructor functions

// NewChatMessage creates a new ChatMessage.
func NewChatMessage(role MessageType, content string) Message {
	return &ChatMessage{
		BaseMessage: BaseMessage{Content: content},
		Role:        role,
	}
}

// NewHumanMessage creates a new human message.
func NewHumanMessage(content string) Message {
	return &ChatMessage{
		BaseMessage: BaseMessage{Content: content},
		Role:        RoleHuman,
	}
}

// NewAIMessage creates a new AI message.
func NewAIMessage(content string) Message {
	return &AIMessage{ // MODIFIED HERE
		BaseMessage: BaseMessage{Content: content},
		// ToolCalls is omitted, so it will be nil (default for a slice)
	}
}

// NewSystemMessage creates a new system message.
func NewSystemMessage(content string) Message {
	return &ChatMessage{
		BaseMessage: BaseMessage{Content: content},
		Role:        RoleSystem,
	}
}

// NewToolMessage creates a new tool message.
func NewToolMessage(content string, toolCallID string) Message {
	return &ToolMessage{
		BaseMessage: BaseMessage{Content: content},
		ToolCallID:  toolCallID,
	}
}

// NewFunctionMessage creates a new function message (primarily for function results).
func NewFunctionMessage(name string, content string) Message {
    return &FunctionMessage{
        BaseMessage: BaseMessage{Content: content},
        Name:        name,
    }
}


// Ensure all message types implement the Message interface.
var _ Message = (*ChatMessage)(nil)
var _ Message = (*ToolMessage)(nil)
var _ Message = (*FunctionMessage)(nil)
var _ Message = (*AIMessage)(nil) // ADDED HERE


// Generation represents a single generation from an LLM.
type Generation struct {
	Text           string                 `json:"text"`
	Message        Message                `json:"message"` // The actual message object, e.g., a ChatMessage
	GenerationInfo map[string]interface{} `json:"generation_info,omitempty"`
}

// LLMResponse represents the response from an LLM.
type LLMResponse struct {
	Generations [][]*Generation        `json:"generations"`
	LLMOutput   map[string]interface{} `json:"llm_output,omitempty"` // Provider-specific output
}

