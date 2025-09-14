package schema

import (
	"testing"
)

func TestNewHumanMessage(t *testing.T) {
	content := "Hello, world!"
	msg := NewHumanMessage(content)

	if msg.GetType() != RoleHuman {
		t.Errorf("Expected message type to be RoleHuman, got %v", msg.GetType())
	}

	if msg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, msg.GetContent())
	}
}

func TestNewAIMessage(t *testing.T) {
	content := "Hello from AI!"
	msg := NewAIMessage(content)

	if msg.GetType() != RoleAssistant {
		t.Errorf("Expected message type to be RoleAssistant, got %v", msg.GetType())
	}

	if msg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, msg.GetContent())
	}
}

func TestNewSystemMessage(t *testing.T) {
	content := "You are a helpful assistant."
	msg := NewSystemMessage(content)

	if msg.GetType() != RoleSystem {
		t.Errorf("Expected message type to be RoleSystem, got %v", msg.GetType())
	}

	if msg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, msg.GetContent())
	}
}

func TestNewToolMessage(t *testing.T) {
	content := "Tool execution result"
	toolCallID := "call_123"
	msg := NewToolMessage(content, toolCallID)

	if msg.GetType() != RoleTool {
		t.Errorf("Expected message type to be RoleTool, got %v", msg.GetType())
	}

	if msg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, msg.GetContent())
	}

	if msg.ToolCallID != toolCallID {
		t.Errorf("Expected ToolCallID to be %q, got %q", toolCallID, msg.ToolCallID)
	}
}

func TestAIMessage_GetAdditionalArgs(t *testing.T) {
	msg := NewAIMessage("Hello")

	args := msg.GetAdditionalArgs()
	if args == nil {
		t.Error("Expected non-nil additional args")
		return
	}

	// AIMessage should have additional args initialized
	if len(args) != 0 {
		// This is expected for a basic AIMessage without tool calls
	}
}

func TestToolMessage_GetAdditionalArgs(t *testing.T) {
	toolCallID := "call_456"
	msg := NewToolMessage("Result", toolCallID)

	args := msg.GetAdditionalArgs()
	if args == nil {
		t.Error("Expected non-nil additional args")
		return
	}

	id, ok := args["tool_call_id"]
	if !ok {
		t.Error("Expected tool_call_id in additional args")
		return
	}

	if id != toolCallID {
		t.Errorf("Expected tool_call_id to be %q, got %q", toolCallID, id)
	}
}

func TestBaseMessage_GetType(t *testing.T) {
	baseMsg := BaseMessage{
		Type:    RoleSystem,
		Content: "Test content",
	}

	if baseMsg.GetType() != RoleSystem {
		t.Errorf("Expected type to be RoleSystem, got %v", baseMsg.GetType())
	}
}

func TestBaseMessage_GetContent(t *testing.T) {
	content := "Test message content"
	baseMsg := BaseMessage{
		Type:    RoleHuman,
		Content: content,
	}

	if baseMsg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, baseMsg.GetContent())
	}
}

func TestBaseMessage_GetAdditionalArgs(t *testing.T) {
	baseMsg := BaseMessage{
		Type:    RoleHuman,
		Content: "Test",
	}

	args := baseMsg.GetAdditionalArgs()
	if args != nil {
		t.Error("Expected nil additional args for BaseMessage")
	}
}
