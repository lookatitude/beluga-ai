package prompt

import (
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestBuilder_Build_FullOrder(t *testing.T) {
	b := NewBuilder(
		WithSystemPrompt("You are an assistant."),
		WithToolDefinitions([]schema.ToolDefinition{
			{Name: "search", Description: "Search the web"},
		}),
		WithStaticContext([]string{"Reference doc 1", "Reference doc 2"}),
		WithCacheBreakpoint(),
		WithDynamicContext([]schema.Message{
			schema.NewHumanMessage("previous question"),
			schema.NewAIMessage("previous answer"),
		}),
		WithUserInput(schema.NewHumanMessage("current question")),
	)

	msgs := b.Build()

	// Expected order:
	// 0: system prompt
	// 1: tool definitions (system)
	// 2: static context 1 (system)
	// 3: static context 2 (system)
	// 4: cache breakpoint (system with metadata)
	// 5: dynamic context - human
	// 6: dynamic context - ai
	// 7: user input - human

	if len(msgs) != 8 {
		t.Fatalf("expected 8 messages, got %d", len(msgs))
	}

	// Slot 1: System prompt.
	if msgs[0].GetRole() != schema.RoleSystem {
		t.Errorf("msg[0] role = %s, want system", msgs[0].GetRole())
	}
	assertTextContains(t, msgs[0], "You are an assistant.")

	// Slot 2: Tool definitions.
	if msgs[1].GetRole() != schema.RoleSystem {
		t.Errorf("msg[1] role = %s, want system", msgs[1].GetRole())
	}
	assertTextContains(t, msgs[1], "search")

	// Slot 3: Static context.
	if msgs[2].GetRole() != schema.RoleSystem {
		t.Errorf("msg[2] role = %s, want system", msgs[2].GetRole())
	}
	assertTextContains(t, msgs[2], "Reference doc 1")

	if msgs[3].GetRole() != schema.RoleSystem {
		t.Errorf("msg[3] role = %s, want system", msgs[3].GetRole())
	}
	assertTextContains(t, msgs[3], "Reference doc 2")

	// Slot 4: Cache breakpoint.
	if msgs[4].GetRole() != schema.RoleSystem {
		t.Errorf("msg[4] role = %s, want system", msgs[4].GetRole())
	}
	meta := msgs[4].GetMetadata()
	if meta == nil || meta["cache_breakpoint"] != true {
		t.Error("expected cache_breakpoint metadata on breakpoint message")
	}

	// Slot 5: Dynamic context.
	if msgs[5].GetRole() != schema.RoleHuman {
		t.Errorf("msg[5] role = %s, want human", msgs[5].GetRole())
	}
	if msgs[6].GetRole() != schema.RoleAI {
		t.Errorf("msg[6] role = %s, want ai", msgs[6].GetRole())
	}

	// Slot 6: User input.
	if msgs[7].GetRole() != schema.RoleHuman {
		t.Errorf("msg[7] role = %s, want human", msgs[7].GetRole())
	}
	assertTextContains(t, msgs[7], "current question")
}

func TestBuilder_Build_EmptyBuilder(t *testing.T) {
	b := NewBuilder()
	msgs := b.Build()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for empty builder, got %d", len(msgs))
	}
}

func TestBuilder_Build_SystemPromptOnly(t *testing.T) {
	b := NewBuilder(WithSystemPrompt("system only"))
	msgs := b.Build()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].GetRole() != schema.RoleSystem {
		t.Errorf("expected system role, got %s", msgs[0].GetRole())
	}
	assertTextContains(t, msgs[0], "system only")
}

func TestBuilder_Build_UserInputOnly(t *testing.T) {
	b := NewBuilder(WithUserInput(schema.NewHumanMessage("just user")))
	msgs := b.Build()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].GetRole() != schema.RoleHuman {
		t.Errorf("expected human role, got %s", msgs[0].GetRole())
	}
}

func TestBuilder_Build_ToolDefinitions(t *testing.T) {
	tools := []schema.ToolDefinition{
		{Name: "search", Description: "Search the web"},
		{Name: "calculate", Description: "Do math"},
	}
	b := NewBuilder(WithToolDefinitions(tools))
	msgs := b.Build()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	assertTextContains(t, msgs[0], "search")
	assertTextContains(t, msgs[0], "calculate")
	assertTextContains(t, msgs[0], "Available tools:")
}

func TestBuilder_Build_StaticContextSkipsEmpty(t *testing.T) {
	b := NewBuilder(WithStaticContext([]string{"doc1", "", "doc2"}))
	msgs := b.Build()
	// Empty strings should be skipped.
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (skipping empty), got %d", len(msgs))
	}
}

func TestBuilder_Build_CacheBreakpointOnly(t *testing.T) {
	b := NewBuilder(WithCacheBreakpoint())
	msgs := b.Build()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	meta := msgs[0].GetMetadata()
	if meta == nil || meta["cache_breakpoint"] != true {
		t.Error("expected cache_breakpoint metadata")
	}
}

func TestBuilder_Build_DynamicContextOnly(t *testing.T) {
	dynamic := []schema.Message{
		schema.NewHumanMessage("q1"),
		schema.NewAIMessage("a1"),
		schema.NewHumanMessage("q2"),
	}
	b := NewBuilder(WithDynamicContext(dynamic))
	msgs := b.Build()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].GetRole() != schema.RoleHuman {
		t.Errorf("msg[0] role = %s, want human", msgs[0].GetRole())
	}
	if msgs[1].GetRole() != schema.RoleAI {
		t.Errorf("msg[1] role = %s, want ai", msgs[1].GetRole())
	}
}

func TestBuilder_Build_StaticBeforeDynamic(t *testing.T) {
	b := NewBuilder(
		WithStaticContext([]string{"static doc"}),
		WithDynamicContext([]schema.Message{schema.NewHumanMessage("dynamic msg")}),
	)
	msgs := b.Build()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	// Static should come before dynamic.
	if msgs[0].GetRole() != schema.RoleSystem {
		t.Errorf("msg[0] should be system (static), got %s", msgs[0].GetRole())
	}
	if msgs[1].GetRole() != schema.RoleHuman {
		t.Errorf("msg[1] should be human (dynamic), got %s", msgs[1].GetRole())
	}
}

func TestBuilder_Build_Ordering_SystemBeforeTools(t *testing.T) {
	b := NewBuilder(
		WithToolDefinitions([]schema.ToolDefinition{{Name: "t1"}}),
		WithSystemPrompt("sys"),
	)
	msgs := b.Build()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	// System prompt should be first regardless of option order.
	assertTextContains(t, msgs[0], "sys")
	assertTextContains(t, msgs[1], "t1")
}

// assertTextContains checks that at least one TextPart in the message contains sub.
func assertTextContains(t *testing.T, msg schema.Message, sub string) {
	t.Helper()
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			if contains(tp.Text, sub) {
				return
			}
		}
	}
	t.Errorf("message does not contain %q", sub)
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
