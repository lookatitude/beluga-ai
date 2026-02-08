package schema

import "testing"

func TestNewSystemMessage(t *testing.T) {
	msg := NewSystemMessage("You are a helpful assistant.")

	if msg.GetRole() != RoleSystem {
		t.Errorf("GetRole() = %q, want %q", msg.GetRole(), RoleSystem)
	}
	if len(msg.GetContent()) != 1 {
		t.Fatalf("GetContent() len = %d, want 1", len(msg.GetContent()))
	}
	if tp, ok := msg.GetContent()[0].(TextPart); !ok || tp.Text != "You are a helpful assistant." {
		t.Errorf("content = %v, want TextPart with %q", msg.GetContent()[0], "You are a helpful assistant.")
	}
	if msg.Text() != "You are a helpful assistant." {
		t.Errorf("Text() = %q, want %q", msg.Text(), "You are a helpful assistant.")
	}
}

func TestNewHumanMessage(t *testing.T) {
	msg := NewHumanMessage("Hello, world!")

	if msg.GetRole() != RoleHuman {
		t.Errorf("GetRole() = %q, want %q", msg.GetRole(), RoleHuman)
	}
	if msg.Text() != "Hello, world!" {
		t.Errorf("Text() = %q, want %q", msg.Text(), "Hello, world!")
	}
}

func TestNewAIMessage(t *testing.T) {
	msg := NewAIMessage("I can help with that.")

	if msg.GetRole() != RoleAI {
		t.Errorf("GetRole() = %q, want %q", msg.GetRole(), RoleAI)
	}
	if msg.Text() != "I can help with that." {
		t.Errorf("Text() = %q, want %q", msg.Text(), "I can help with that.")
	}
}

func TestNewToolMessage(t *testing.T) {
	msg := NewToolMessage("call-123", `{"result": 42}`)

	if msg.GetRole() != RoleTool {
		t.Errorf("GetRole() = %q, want %q", msg.GetRole(), RoleTool)
	}
	if msg.ToolCallID != "call-123" {
		t.Errorf("ToolCallID = %q, want %q", msg.ToolCallID, "call-123")
	}
	if msg.Text() != `{"result": 42}` {
		t.Errorf("Text() = %q, want %q", msg.Text(), `{"result": 42}`)
	}
}

func TestMessageInterface(t *testing.T) {
	// Verify all message types implement the Message interface.
	tests := []struct {
		name string
		msg  Message
		role Role
	}{
		{"system", NewSystemMessage("sys"), RoleSystem},
		{"human", NewHumanMessage("hi"), RoleHuman},
		{"ai", NewAIMessage("hello"), RoleAI},
		{"tool", NewToolMessage("id", "result"), RoleTool},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.GetRole(); got != tt.role {
				t.Errorf("GetRole() = %q, want %q", got, tt.role)
			}
			if parts := tt.msg.GetContent(); len(parts) == 0 {
				t.Error("GetContent() returned empty slice")
			}
		})
	}
}

func TestMessage_GetMetadata(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{"system_nil_metadata", NewSystemMessage("s")},
		{"human_nil_metadata", NewHumanMessage("h")},
		{"ai_nil_metadata", NewAIMessage("a")},
		{"tool_nil_metadata", NewToolMessage("id", "t")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Default constructors produce nil metadata.
			meta := tt.msg.GetMetadata()
			if meta != nil {
				t.Errorf("GetMetadata() = %v, want nil", meta)
			}
		})
	}
}

func TestMessage_GetMetadata_WithData(t *testing.T) {
	msg := &SystemMessage{
		Parts:    []ContentPart{TextPart{Text: "test"}},
		Metadata: map[string]any{"key": "value"},
	}
	meta := msg.GetMetadata()
	if meta == nil || meta["key"] != "value" {
		t.Errorf("GetMetadata() = %v, want map with key=value", meta)
	}
}

func TestMessage_Text_MultipleParts(t *testing.T) {
	msg := &HumanMessage{
		Parts: []ContentPart{
			TextPart{Text: "Hello"},
			ImagePart{URL: "http://example.com/img.png"},
			TextPart{Text: "world"},
		},
	}

	got := msg.Text()
	want := "Hello\nworld"
	if got != want {
		t.Errorf("Text() = %q, want %q", got, want)
	}
}

func TestMessage_Text_NoTextParts(t *testing.T) {
	msg := &HumanMessage{
		Parts: []ContentPart{
			ImagePart{URL: "http://example.com/img.png"},
			AudioPart{Format: "mp3"},
		},
	}

	if got := msg.Text(); got != "" {
		t.Errorf("Text() = %q, want empty string", got)
	}
}

func TestMessage_Text_Empty(t *testing.T) {
	msg := &HumanMessage{Parts: nil}
	if got := msg.Text(); got != "" {
		t.Errorf("Text() = %q, want empty string", got)
	}
}

func TestAIMessage_ToolCalls(t *testing.T) {
	msg := &AIMessage{
		Parts: []ContentPart{TextPart{Text: "Let me look that up."}},
		ToolCalls: []ToolCall{
			{ID: "tc1", Name: "search", Arguments: `{"query":"test"}`},
			{ID: "tc2", Name: "calculate", Arguments: `{"expr":"1+1"}`},
		},
		Usage: Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
	}

	if len(msg.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Name != "search" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", msg.ToolCalls[0].Name, "search")
	}
	if msg.Usage.TotalTokens != 30 {
		t.Errorf("Usage.TotalTokens = %d, want 30", msg.Usage.TotalTokens)
	}
}

func TestAIMessage_ModelID(t *testing.T) {
	msg := &AIMessage{
		Parts:   []ContentPart{TextPart{Text: "hi"}},
		ModelID: "gpt-4o",
	}
	if msg.ModelID != "gpt-4o" {
		t.Errorf("ModelID = %q, want %q", msg.ModelID, "gpt-4o")
	}
}

func TestRole_Values(t *testing.T) {
	roles := map[Role]string{
		RoleSystem: "system",
		RoleHuman:  "human",
		RoleAI:     "ai",
		RoleTool:   "tool",
	}
	for role, want := range roles {
		if string(role) != want {
			t.Errorf("Role %v = %q, want %q", role, string(role), want)
		}
	}
}

func TestUsage_Fields(t *testing.T) {
	u := Usage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		CachedTokens: 20,
	}
	if u.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", u.InputTokens)
	}
	if u.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", u.OutputTokens)
	}
	if u.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", u.TotalTokens)
	}
	if u.CachedTokens != 20 {
		t.Errorf("CachedTokens = %d, want 20", u.CachedTokens)
	}
}
