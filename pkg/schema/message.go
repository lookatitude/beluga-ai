package schema

import (
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
func NewMessage(content string, msgType MessageType) Message {
	switch msgType {
	case AIMessageType:
		return &AIMessage{BaseMessage: BaseMessage{Content: content}}
	case HumanMessageType:
		return &HumanMessage{BaseMessage: BaseMessage{Content: content}}
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
		return &ToolMessage{BaseMessage: BaseMessage{Content: content}}
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
		msg = &AIMessage{}
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

