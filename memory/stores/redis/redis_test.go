package redis

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ memory.MessageStore = (*MessageStore)(nil)

func newTestStore(t *testing.T) (*MessageStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	store, err := New(Config{Client: client})
	require.NoError(t, err)
	return store, mr
}

func TestNew(t *testing.T) {
	t.Run("nil client returns error", func(t *testing.T) {
		_, err := New(Config{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client is required")
	})

	t.Run("default key", func(t *testing.T) {
		mr := miniredis.RunT(t)
		client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
		store, err := New(Config{Client: client})
		require.NoError(t, err)
		assert.Equal(t, "beluga:messages", store.key)
	})

	t.Run("custom key", func(t *testing.T) {
		mr := miniredis.RunT(t)
		client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
		store, err := New(Config{Client: client, Key: "custom:key"})
		require.NoError(t, err)
		assert.Equal(t, "custom:key", store.key)
	})
}

func TestAppend(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	err := store.Append(ctx, schema.NewHumanMessage("hello"))
	require.NoError(t, err)

	err = store.Append(ctx, schema.NewAIMessage("hi"))
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
	assert.Equal(t, schema.RoleHuman, msgs[0].GetRole())
	assert.Equal(t, schema.RoleAI, msgs[1].GetRole())
}

func TestAppendAllMessageTypes(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	// System message
	err := store.Append(ctx, schema.NewSystemMessage("you are helpful"))
	require.NoError(t, err)

	// Human message
	err = store.Append(ctx, schema.NewHumanMessage("hello"))
	require.NoError(t, err)

	// AI message with tool calls
	ai := &schema.AIMessage{
		Parts:   []schema.ContentPart{schema.TextPart{Text: "calling tool"}},
		ModelID: "gpt-4",
		ToolCalls: []schema.ToolCall{
			{ID: "tc1", Name: "search", Arguments: `{"q":"test"}`},
		},
	}
	err = store.Append(ctx, ai)
	require.NoError(t, err)

	// Tool message
	err = store.Append(ctx, schema.NewToolMessage("tc1", "result"))
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 4)

	assert.Equal(t, schema.RoleSystem, msgs[0].GetRole())
	assert.Equal(t, schema.RoleHuman, msgs[1].GetRole())
	assert.Equal(t, schema.RoleAI, msgs[2].GetRole())
	assert.Equal(t, schema.RoleTool, msgs[3].GetRole())

	// Check AI message preserved tool calls
	aiMsg, ok := msgs[2].(*schema.AIMessage)
	require.True(t, ok)
	assert.Equal(t, "gpt-4", aiMsg.ModelID)
	require.Len(t, aiMsg.ToolCalls, 1)
	assert.Equal(t, "search", aiMsg.ToolCalls[0].Name)

	// Check tool message preserved call ID
	toolMsg, ok := msgs[3].(*schema.ToolMessage)
	require.True(t, ok)
	assert.Equal(t, "tc1", toolMsg.ToolCallID)
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	_ = store.Append(ctx, schema.NewHumanMessage("hello world"))
	_ = store.Append(ctx, schema.NewAIMessage("hi there"))
	_ = store.Append(ctx, schema.NewHumanMessage("how are you"))
	_ = store.Append(ctx, schema.NewAIMessage("I'm doing well"))
	_ = store.Append(ctx, schema.NewHumanMessage("goodbye"))

	tests := []struct {
		name      string
		query     string
		k         int
		wantCount int
	}{
		{"match single", "hello", 10, 1},
		{"case insensitive", "HELLO", 10, 1},
		{"match multiple", "o", 10, 4}, // hello, how, doing, goodbye
		{"limit results", "o", 2, 2},
		{"no match", "xyz", 10, 0},
		{"empty query matches all", "", 10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Search(ctx, tt.query, tt.k)
			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
		})
	}
}

func TestAll(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	// Empty store.
	msgs, err := store.All(ctx)
	require.NoError(t, err)
	assert.Empty(t, msgs)

	// Add messages.
	_ = store.Append(ctx, schema.NewHumanMessage("msg1"))
	_ = store.Append(ctx, schema.NewAIMessage("msg2"))
	_ = store.Append(ctx, schema.NewHumanMessage("msg3"))

	msgs, err = store.All(ctx)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)

	// Verify chronological order.
	assert.Contains(t, textOf(msgs[0]), "msg1")
	assert.Contains(t, textOf(msgs[1]), "msg2")
	assert.Contains(t, textOf(msgs[2]), "msg3")
}

func TestClear(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	_ = store.Append(ctx, schema.NewHumanMessage("hello"))
	_ = store.Append(ctx, schema.NewAIMessage("hi"))

	err := store.Clear(ctx)
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestClearIdempotent(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	// Clear an already empty store should not error.
	err := store.Clear(ctx)
	require.NoError(t, err)
}

func TestMetadataPreserved(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	msg := &schema.HumanMessage{
		Parts:    []schema.ContentPart{schema.TextPart{Text: "hello"}},
		Metadata: map[string]any{"key": "value", "num": float64(42)},
	}
	err := store.Append(ctx, msg)
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "value", msgs[0].GetMetadata()["key"])
	assert.Equal(t, float64(42), msgs[0].GetMetadata()["num"])
}

func TestCustomKey(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	store1, err := New(Config{Client: client, Key: "session:1"})
	require.NoError(t, err)
	store2, err := New(Config{Client: client, Key: "session:2"})
	require.NoError(t, err)

	_ = store1.Append(ctx, schema.NewHumanMessage("store1 msg"))
	_ = store2.Append(ctx, schema.NewHumanMessage("store2 msg"))

	msgs1, _ := store1.All(ctx)
	msgs2, _ := store2.All(ctx)

	assert.Len(t, msgs1, 1)
	assert.Len(t, msgs2, 1)
	assert.Contains(t, textOf(msgs1[0]), "store1")
	assert.Contains(t, textOf(msgs2[0]), "store2")
}

func TestSearchOrder(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)

	// Add messages with a common word.
	_ = store.Append(ctx, schema.NewHumanMessage("test one"))
	_ = store.Append(ctx, schema.NewAIMessage("test two"))
	_ = store.Append(ctx, schema.NewHumanMessage("test three"))

	results, err := store.Search(ctx, "test", 10)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Verify chronological order.
	assert.Contains(t, textOf(results[0]), "one")
	assert.Contains(t, textOf(results[1]), "two")
	assert.Contains(t, textOf(results[2]), "three")
}

// textOf extracts concatenated text from a message.
func textOf(msg schema.Message) string {
	var parts []string
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			parts = append(parts, tp.Text)
		}
	}
	return strings.Join(parts, " ")
}
