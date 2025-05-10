package schema

import (
"encoding/json"
"fmt"
)

// MessageType defines the type of a message.
// @enum MessageType
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

// BaseMessage is a struct that provides a base implementation for the Message interface.
// It includes the common fields that all messages should have.
type BaseMessage struct {
Content string `json:"content"`
Type    string `json:"type"`
}

// NewMessage creates a new message with the given content and type.
func NewMessage(content string, msgType MessageType) Message {
switch msgType {
case AIMessageType:
return &AIMessage{MessageContent: content}
case HumanMessageType:
return &HumanMessage{MessageContent: content}
case SystemMessageType:
return &SystemMessage{MessageContent: content}
case ChatMessageType:
return &ChatMessage{MessageContent: content}
case FunctionMessageType:
return &FunctionMessage{MessageContent: content}
case ToolMessageType:
return &ToolMessage{MessageContent: content}
default:
panic("Unknown message type")
}
}

// AIMessage is a message from an AI.
// It includes the content of the message.
type AIMessage struct {
MessageContent string `json:"content"`
}

func (m *AIMessage) GetType() MessageType { return AIMessageType }
func (m *AIMessage) String() string       { return m.MessageContent }
func (m *AIMessage) GetContent() string   { return m.MessageContent }

// HumanMessage is a message from a human.
// It includes the content of the message.
type HumanMessage struct {
MessageContent string `json:"content"`
}

func (m *HumanMessage) GetType() MessageType { return HumanMessageType }
func (m *HumanMessage) String() string       { return m.MessageContent }
func (m *HumanMessage) GetContent() string   { return m.MessageContent }

// SystemMessage is a message from the system.
// It includes the content of the message.
type SystemMessage struct {
MessageContent string `json:"content"`
}

func (m *SystemMessage) GetType() MessageType { return SystemMessageType }
func (m *SystemMessage) String() string       { return m.MessageContent }
func (m *SystemMessage) GetContent() string   { return m.MessageContent }

// ChatMessage is a message that can be stored in a chat history.
// It includes the role of the speaker (e.g., 
