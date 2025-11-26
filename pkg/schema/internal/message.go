package internal

import (
	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
)

// Re-export types from iface for internal use.
type (
	MessageType  = iface.MessageType
	ToolCall     = iface.ToolCall
	FunctionCall = iface.FunctionCall
)

// Re-export constants from iface.
const (
	RoleHuman     = iface.RoleHuman
	RoleAssistant = iface.RoleAssistant
	RoleSystem    = iface.RoleSystem
	RoleTool      = iface.RoleTool
	RoleFunction  = iface.RoleFunction
)

// BaseMessage provides common fields for messages.
type BaseMessage struct {
	Content string `json:"content"`
}

// GetContent returns the content of the base message.
func (bm *BaseMessage) GetContent() string {
	return bm.Content
}

// ToolCalls returns an empty slice by default.
func (bm *BaseMessage) ToolCalls() []ToolCall {
	return []ToolCall{}
}

// AdditionalArgs returns an empty map by default.
func (bm *BaseMessage) AdditionalArgs() map[string]any {
	return make(map[string]any)
}

// GetAdditionalArgs is an alias for AdditionalArgs for backward compatibility.
func (bm *BaseMessage) GetAdditionalArgs() map[string]any {
	return bm.AdditionalArgs()
}

// ToolCallChunk represents a chunk of a tool call, useful for streaming responses.
type ToolCallChunk struct {
	Function  FunctionCall `json:"function,omitempty"`
	ID        string       `json:"id,omitempty"`
	Type      string       `json:"type,omitempty"`
	Name      string       `json:"name,omitempty"`
	Arguments string       `json:"arguments,omitempty"`
	Index     int          `json:"index,omitempty"`
}

// ChatMessage represents a message in a chat sequence.
type ChatMessage struct {
	BaseMessage
	Role       MessageType `json:"role"`
	ToolCalls_ []ToolCall  `json:"tool_calls,omitempty"` // Used by AIMessage for proposing tool calls
}

// GetType returns the role of the ChatMessage.
func (m *ChatMessage) GetType() MessageType {
	return m.Role
}

// ToolCalls returns the tool calls in this message.
func (m *ChatMessage) ToolCalls() []ToolCall {
	if m.ToolCalls_ == nil {
		return []ToolCall{}
	}
	return m.ToolCalls_
}

// AdditionalArgs returns an empty map for ChatMessage.
func (m *ChatMessage) AdditionalArgs() map[string]any {
	return make(map[string]any)
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

// ToolCalls returns an empty slice for ToolMessage.
func (m *ToolMessage) ToolCalls() []ToolCall {
	return []ToolCall{}
}

// AdditionalArgs returns an empty map for ToolMessage.
func (m *ToolMessage) AdditionalArgs() map[string]any {
	return make(map[string]any)
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

// ToolCalls returns an empty slice for FunctionMessage.
func (m *FunctionMessage) ToolCalls() []ToolCall {
	return []ToolCall{}
}

// AdditionalArgs returns an empty map for FunctionMessage.
func (m *FunctionMessage) AdditionalArgs() map[string]any {
	return make(map[string]any)
}

// AIMessage represents a message from the AI.
// It can include content and a list of tool calls the AI wants to make.
type AIMessage struct {
	AdditionalArgs_ map[string]any `json:"additional_kwargs,omitempty"`
	BaseMessage
	ToolCalls_ []ToolCall `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"`
}

// GetType returns the message type, which is always RoleAssistant for AIMessage.
func (m *AIMessage) GetType() MessageType {
	return RoleAssistant
}

// ToolCalls returns the tool calls in this message.
func (m *AIMessage) ToolCalls() []ToolCall {
	if m.ToolCalls_ == nil {
		return []ToolCall{}
	}
	return m.ToolCalls_
}

// AdditionalArgs returns additional arguments for this message.
func (m *AIMessage) AdditionalArgs() map[string]any {
	if m.AdditionalArgs_ == nil {
		m.AdditionalArgs_ = make(map[string]any)
	}
	return m.AdditionalArgs_
}

// Generation represents a single generation from an LLM.
type Generation struct {
	Message        iface.Message  `json:"message"`
	GenerationInfo map[string]any `json:"generation_info,omitempty"`
	Text           string         `json:"text"`
}

// CallOptions holds parameters for an LLM call.
type CallOptions struct {
	Temperature          *float64
	MaxTokens            *int
	TopP                 *float64
	FrequencyPenalty     *float64
	PresencePenalty      *float64
	ProviderSpecificArgs map[string]any
	Stop                 []string
	Streaming            bool
}

// LLMOption defines a function type for LLM call options.
type LLMOption func(options *CallOptions)

// LLMResponse represents the response from an LLM.
type LLMResponse struct {
	LLMOutput   map[string]any  `json:"llm_output,omitempty"`
	Generations [][]*Generation `json:"generations"`
}
