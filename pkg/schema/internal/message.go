package internal

import (
	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
)

// Re-export types from iface for internal use
type (
	MessageType  = iface.MessageType
	ToolCall     = iface.ToolCall
	FunctionCall = iface.FunctionCall
)

// Re-export constants from iface
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
	return nil
}

// AdditionalArgs returns an empty map by default.
func (bm *BaseMessage) AdditionalArgs() map[string]interface{} {
	return nil
}

// GetAdditionalArgs is an alias for AdditionalArgs for backward compatibility.
func (bm *BaseMessage) GetAdditionalArgs() map[string]interface{} {
	return bm.AdditionalArgs()
}

// ToolCallChunk represents a chunk of a tool call, useful for streaming responses.
type ToolCallChunk struct {
	ID        string       `json:"id,omitempty"`
	Type      string       `json:"type,omitempty"` // Typically "function"
	Function  FunctionCall `json:"function,omitempty"`
	Index     int          `json:"index,omitempty"`     // Index in a sequence of chunks
	Name      string       `json:"name,omitempty"`      // For direct access
	Arguments string       `json:"arguments,omitempty"` // For direct access
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
	return m.ToolCalls_
}

// AdditionalArgs returns an empty map for ChatMessage.
func (m *ChatMessage) AdditionalArgs() map[string]interface{} {
	return make(map[string]interface{})
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
	return nil
}

// AdditionalArgs returns an empty map for ToolMessage.
func (m *ToolMessage) AdditionalArgs() map[string]interface{} {
	return make(map[string]interface{})
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
	return nil
}

// AdditionalArgs returns an empty map for FunctionMessage.
func (m *FunctionMessage) AdditionalArgs() map[string]interface{} {
	return make(map[string]interface{})
}

// AIMessage represents a message from the AI.
// It can include content and a list of tool calls the AI wants to make.
type AIMessage struct {
	BaseMessage
	// Content is inherited from BaseMessage.
	// Role is implicitly RoleAssistant, returned by GetType().
	ToolCalls_      []ToolCall             `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"`
	AdditionalArgs_ map[string]interface{} `json:"additional_kwargs,omitempty"`
}

// GetType returns the message type, which is always RoleAssistant for AIMessage.
func (m *AIMessage) GetType() MessageType {
	return RoleAssistant
}

// ToolCalls returns the tool calls in this message.
func (m *AIMessage) ToolCalls() []ToolCall {
	return m.ToolCalls_
}

// AdditionalArgs returns additional arguments for this message.
func (m *AIMessage) AdditionalArgs() map[string]interface{} {
	if m.AdditionalArgs_ == nil {
		m.AdditionalArgs_ = make(map[string]interface{})
	}
	return m.AdditionalArgs_
}

// Generation represents a single generation from an LLM.
type Generation struct {
	Text           string                 `json:"text"`
	Message        iface.Message          `json:"message"` // The actual message object, e.g., a ChatMessage
	GenerationInfo map[string]interface{} `json:"generation_info,omitempty"`
}

// CallOptions holds parameters for an LLM call.
type CallOptions struct {
	Temperature      *float64
	MaxTokens        *int
	TopP             *float64
	FrequencyPenalty *float64
	PresencePenalty  *float64
	Stop             []string
	Streaming        bool
	// ProviderSpecificArgs allows for passing through any other provider-specific options.
	ProviderSpecificArgs map[string]interface{}
}

// LLMOption defines a function type for LLM call options.
type LLMOption func(options *CallOptions)

// LLMResponse represents the response from an LLM.
type LLMResponse struct {
	Generations [][]*Generation        `json:"generations"`
	LLMOutput   map[string]interface{} `json:"llm_output,omitempty"` // Provider-specific output
}
