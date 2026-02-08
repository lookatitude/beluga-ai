package llm

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestTruncateStrategy_WithinBudget(t *testing.T) {
	cm := NewContextManager(WithContextStrategy("truncate"))
	msgs := []schema.Message{
		schema.NewHumanMessage("hi"),
	}

	result, err := cm.Fit(context.Background(), msgs, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(msgs) {
		t.Errorf("expected %d messages, got %d", len(msgs), len(result))
	}
}

func TestTruncateStrategy_DropsOldestNonSystem(t *testing.T) {
	cm := NewContextManager(WithContextStrategy("truncate"))

	msgs := []schema.Message{
		schema.NewSystemMessage("system prompt"),
		schema.NewHumanMessage("old message one"),
		schema.NewHumanMessage("old message two"),
		schema.NewHumanMessage("recent message"),
	}

	// Use a tight budget that forces dropping old messages.
	// SimpleTokenizer counts ~4 chars per token + 4 overhead per message.
	result, err := cm.Fit(context.Background(), msgs, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// System message should always be preserved.
	if len(result) < 1 {
		t.Fatal("expected at least system message")
	}
	if result[0].GetRole() != schema.RoleSystem {
		t.Errorf("first message should be system, got %s", result[0].GetRole())
	}
	// Should have fewer non-system messages than original.
	if len(result) >= len(msgs) {
		t.Errorf("expected truncation to reduce messages from %d, got %d", len(msgs), len(result))
	}
}

func TestTruncateStrategy_InvalidBudget(t *testing.T) {
	cm := NewContextManager(WithContextStrategy("truncate"))
	_, err := cm.Fit(context.Background(), nil, 0)
	if err == nil {
		t.Fatal("expected error for zero budget")
	}

	_, err = cm.Fit(context.Background(), nil, -1)
	if err == nil {
		t.Fatal("expected error for negative budget")
	}
}

func TestTruncateStrategy_SystemOnlyExceedsBudget(t *testing.T) {
	cm := NewContextManager(WithContextStrategy("truncate"))
	msgs := []schema.Message{
		schema.NewSystemMessage("a very long system prompt that uses many tokens"),
		schema.NewHumanMessage("user input"),
	}

	// Very tight budget: only system message might fit.
	result, err := cm.Fit(context.Background(), msgs, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should at least return the system message.
	if len(result) == 0 {
		t.Fatal("expected at least system message returned")
	}
	if result[0].GetRole() != schema.RoleSystem {
		t.Errorf("first message should be system, got %s", result[0].GetRole())
	}
}

func TestSlidingStrategy_WithinBudget(t *testing.T) {
	cm := NewContextManager(WithContextStrategy("sliding"))
	msgs := []schema.Message{
		schema.NewHumanMessage("hi"),
	}

	result, err := cm.Fit(context.Background(), msgs, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(msgs) {
		t.Errorf("expected %d messages, got %d", len(msgs), len(result))
	}
}

func TestSlidingStrategy_KeepsRecentMessages(t *testing.T) {
	cm := NewContextManager(WithContextStrategy("sliding"))

	msgs := []schema.Message{
		schema.NewSystemMessage("system"),
		schema.NewHumanMessage("old1"),
		schema.NewHumanMessage("old2"),
		schema.NewHumanMessage("recent"),
	}

	result, err := cm.Fit(context.Background(), msgs, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// System should be preserved.
	if len(result) < 1 {
		t.Fatal("expected at least system message")
	}
	if result[0].GetRole() != schema.RoleSystem {
		t.Errorf("first message should be system, got %s", result[0].GetRole())
	}

	// Should have fewer messages than original due to sliding window.
	if len(result) >= len(msgs) {
		t.Errorf("expected sliding window to reduce messages from %d, got %d", len(msgs), len(result))
	}

	// Last non-system message should be the most recent.
	last := result[len(result)-1]
	if last.GetRole() != schema.RoleHuman {
		t.Errorf("last message should be human, got %s", last.GetRole())
	}
}

func TestSlidingStrategy_InvalidBudget(t *testing.T) {
	cm := NewContextManager(WithContextStrategy("sliding"))
	_, err := cm.Fit(context.Background(), nil, 0)
	if err == nil {
		t.Fatal("expected error for zero budget")
	}
}

func TestNewContextManager_DefaultStrategy(t *testing.T) {
	cm := NewContextManager()
	// Default is truncate; it should work without error.
	msgs := []schema.Message{schema.NewHumanMessage("test")}
	result, err := cm.Fit(context.Background(), msgs, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 message, got %d", len(result))
	}
}

func TestContextManager_WithKeepSystemFalse(t *testing.T) {
	cm := NewContextManager(
		WithContextStrategy("truncate"),
		WithKeepSystemMessages(false),
	)

	msgs := []schema.Message{
		schema.NewSystemMessage("system"),
		schema.NewHumanMessage("user"),
	}

	// With a very tight budget, system messages can be dropped if keepSystem=false.
	result, err := cm.Fit(context.Background(), msgs, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All messages go into the "rest" pool when keepSystem is false.
	_ = result // No crash is the minimum bar here.
}

func TestContextManager_EmptyMessages(t *testing.T) {
	cm := NewContextManager()
	result, err := cm.Fit(context.Background(), nil, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 messages for nil input, got %d", len(result))
	}
}
