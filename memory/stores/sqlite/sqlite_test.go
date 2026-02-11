package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// Compile-time interface check.
var _ memory.MessageStore = (*MessageStore)(nil)

func newTestStore(t *testing.T) *MessageStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	store, err := New(Config{DB: db})
	require.NoError(t, err)
	err = store.EnsureTable(context.Background())
	require.NoError(t, err)
	return store
}

func TestNew(t *testing.T) {
	t.Run("nil db returns error", func(t *testing.T) {
		_, err := New(Config{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db is required")
	})

	t.Run("default table", func(t *testing.T) {
		db, _ := sql.Open("sqlite", ":memory:")
		defer db.Close()
		store, err := New(Config{DB: db})
		require.NoError(t, err)
		assert.Equal(t, "messages", store.table)
	})

	t.Run("custom table", func(t *testing.T) {
		db, _ := sql.Open("sqlite", ":memory:")
		defer db.Close()
		store, err := New(Config{DB: db, Table: "custom_msgs"})
		require.NoError(t, err)
		assert.Equal(t, "custom_msgs", store.table)
	})
}

func TestEnsureTable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	store, err := New(Config{DB: db})
	require.NoError(t, err)

	// First call creates table.
	err = store.EnsureTable(context.Background())
	require.NoError(t, err)

	// Second call is idempotent.
	err = store.EnsureTable(context.Background())
	require.NoError(t, err)
}

func TestAppend(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

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
	store := newTestStore(t)

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
	assert.Equal(t, `{"q":"test"}`, aiMsg.ToolCalls[0].Arguments)

	// Check tool message preserved call ID
	toolMsg, ok := msgs[3].(*schema.ToolMessage)
	require.True(t, ok)
	assert.Equal(t, "tc1", toolMsg.ToolCallID)
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Search(ctx, tt.query, tt.k)
			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
		})
	}
}

func TestSearchOrder(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

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

func TestAll(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

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

	// Verify order is preserved.
	assert.Contains(t, textOf(msgs[0]), "msg1")
	assert.Contains(t, textOf(msgs[1]), "msg2")
	assert.Contains(t, textOf(msgs[2]), "msg3")
}

func TestClear(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

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
	store := newTestStore(t)

	// Clear an already empty store should not error.
	err := store.Clear(ctx)
	require.NoError(t, err)
}

func TestMetadataPreserved(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

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

func TestCustomTable(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	store, err := New(Config{DB: db, Table: "custom_msgs"})
	require.NoError(t, err)
	err = store.EnsureTable(ctx)
	require.NoError(t, err)

	err = store.Append(ctx, schema.NewHumanMessage("hello"))
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
}

func TestNilMetadata(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	msg := &schema.HumanMessage{
		Parts: []schema.ContentPart{schema.TextPart{Text: "hello"}},
		// No metadata
	}
	err := store.Append(ctx, msg)
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	// Should not crash on nil metadata.
}

func TestSearch_Error(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	store, err := New(Config{DB: db})
	require.NoError(t, err)
	// Don't call EnsureTable → table doesn't exist → query will fail.
	_, err = store.Search(ctx, "query", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite: search:")
}

func TestAll_Error(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	store, err := New(Config{DB: db})
	require.NoError(t, err)
	// Don't call EnsureTable → table doesn't exist → query will fail.
	_, err = store.All(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite: all:")
}

func TestClear_Error(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	store, err := New(Config{DB: db})
	require.NoError(t, err)
	// Don't call EnsureTable → table doesn't exist → delete will fail.
	err = store.Clear(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite: clear:")
}

func TestAppend_Error(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	store, err := New(Config{DB: db})
	require.NoError(t, err)
	// Don't call EnsureTable → table doesn't exist → insert will fail.
	err = store.Append(ctx, schema.NewHumanMessage("hello"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite: append:")
}

func TestScanMessages_UnknownRole(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	// Insert a row with unknown role directly.
	_, err := store.db.ExecContext(ctx,
		"INSERT INTO messages (role, content, metadata, created_at) VALUES (?, ?, ?, ?)",
		"observer", `{"parts":[{"type":"text","text":"hello"}]}`, `null`, "2024-01-01T00:00:00Z",
	)
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	// Unknown role defaults to HumanMessage.
	assert.Equal(t, schema.RoleHuman, msgs[0].GetRole())
}

func TestScanMessages_EmptyMetadata(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	// Insert a row with empty metadata string.
	_, err := store.db.ExecContext(ctx,
		"INSERT INTO messages (role, content, metadata, created_at) VALUES (?, ?, ?, ?)",
		"human", `{"parts":[{"type":"text","text":"hello"}]}`, "", "2024-01-01T00:00:00Z",
	)
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Nil(t, msgs[0].GetMetadata())
}

func TestScanMessages_NullMetadata(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	_, err := store.db.ExecContext(ctx,
		"INSERT INTO messages (role, content, metadata, created_at) VALUES (?, ?, ?, ?)",
		"human", `{"parts":[{"type":"text","text":"hello"}]}`, "null", "2024-01-01T00:00:00Z",
	)
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Nil(t, msgs[0].GetMetadata())
}

func TestScanMessages_BadContentJSON(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	_, err := store.db.ExecContext(ctx,
		"INSERT INTO messages (role, content, metadata, created_at) VALUES (?, ?, ?, ?)",
		"human", `not json`, `null`, "2024-01-01T00:00:00Z",
	)
	require.NoError(t, err)

	_, err = store.All(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite: unmarshal content:")
}

func TestScanMessages_BadMetadataJSON(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	_, err := store.db.ExecContext(ctx,
		"INSERT INTO messages (role, content, metadata, created_at) VALUES (?, ?, ?, ?)",
		"human", `{"parts":[{"type":"text","text":"hello"}]}`, `not json`, "2024-01-01T00:00:00Z",
	)
	require.NoError(t, err)

	_, err = store.All(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite: unmarshal metadata:")
}

func TestAppendAndRetrieveToolMessage(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	err := store.Append(ctx, schema.NewToolMessage("tc1", "result data"))
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	toolMsg, ok := msgs[0].(*schema.ToolMessage)
	require.True(t, ok)
	assert.Equal(t, "tc1", toolMsg.ToolCallID)
}

func TestAppendAndRetrieveSystemMessage(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	err := store.Append(ctx, schema.NewSystemMessage("you are helpful"))
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, schema.RoleSystem, msgs[0].GetRole())
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
