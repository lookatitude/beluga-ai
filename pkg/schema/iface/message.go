package iface

// MessageType defines the type of a message.
type MessageType string

// Constants for MessageType.
const (
	RoleHuman     MessageType = "human"
	RoleAssistant MessageType = "ai"
	RoleSystem    MessageType = "system"
	RoleTool      MessageType = "tool"
	RoleFunction  MessageType = "function"
)

// ToolCall represents a call to a tool by the LLM.
type ToolCall struct {
	ID        string       `json:"id"`
	Type      string       `json:"type"` // Typically "function"
	Function  FunctionCall `json:"function"`
	Name      string       `json:"name,omitempty"`      // For direct access
	Arguments string       `json:"arguments,omitempty"` // For direct access
}

// FunctionCall represents the function to be called.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string of arguments
}

// Message is the interface that all message types must implement.
// It provides methods to access message content, type, tool calls, and additional arguments.
type Message interface {
	// GetType returns the type/role of the message (e.g., human, ai, system, tool)
	GetType() MessageType

	// GetContent returns the textual content of the message
	GetContent() string

	// ToolCalls returns any tool calls associated with this message
	ToolCalls() []ToolCall

	// AdditionalArgs returns additional provider-specific arguments
	AdditionalArgs() map[string]any
}

// ChatHistory defines the interface for storing and retrieving chat messages.
// It provides methods for managing conversation history.
type ChatHistory interface {
	// AddMessage adds a message to the history
	AddMessage(message Message) error

	// AddUserMessage adds a user message to the history
	AddUserMessage(message string) error

	// AddAIMessage adds an AI message to the history
	AddAIMessage(message string) error

	// Messages returns all messages in the history
	Messages() ([]Message, error)

	// Clear removes all messages from the history
	Clear() error
}
