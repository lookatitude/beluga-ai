// Package storeutil provides shared types and helpers for memory store implementations.
package storeutil

import "github.com/lookatitude/beluga-ai/schema"

// StoredPart represents a serialised content part.
type StoredPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// StoredContent represents the serialised content of a message.
type StoredContent struct {
	Parts      []StoredPart      `json:"parts"`
	ToolCalls  []schema.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	ModelID    string            `json:"model_id,omitempty"`
}

// BuildMessage creates the correct schema.Message type for the given role.
func BuildMessage(role string, parts []schema.ContentPart, metadata map[string]any, sc StoredContent) schema.Message {
	switch schema.Role(role) {
	case schema.RoleSystem:
		return &schema.SystemMessage{Parts: parts, Metadata: metadata}
	case schema.RoleHuman:
		return &schema.HumanMessage{Parts: parts, Metadata: metadata}
	case schema.RoleAI:
		return &schema.AIMessage{Parts: parts, ToolCalls: sc.ToolCalls, ModelID: sc.ModelID, Metadata: metadata}
	case schema.RoleTool:
		return &schema.ToolMessage{ToolCallID: sc.ToolCallID, Parts: parts, Metadata: metadata}
	default:
		return &schema.HumanMessage{Parts: parts, Metadata: metadata}
	}
}

// EncodeContent converts a schema.Message into a StoredContent for serialisation.
func EncodeContent(msg schema.Message) StoredContent {
	sc := StoredContent{}
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			sc.Parts = append(sc.Parts, StoredPart{Type: "text", Text: tp.Text})
		}
	}
	if ai, ok := msg.(*schema.AIMessage); ok {
		sc.ToolCalls = ai.ToolCalls
		sc.ModelID = ai.ModelID
	}
	if tm, ok := msg.(*schema.ToolMessage); ok {
		sc.ToolCallID = tm.ToolCallID
	}
	return sc
}

// DecodeParts converts StoredParts back into schema.ContentParts.
func DecodeParts(stored []StoredPart) []schema.ContentPart {
	parts := make([]schema.ContentPart, 0, len(stored))
	for _, sp := range stored {
		parts = append(parts, schema.TextPart{Text: sp.Text})
	}
	return parts
}
