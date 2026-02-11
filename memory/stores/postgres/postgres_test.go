package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ memory.MessageStore = (*MessageStore)(nil)

func newTestStore(t *testing.T) (*MessageStore, pgxmock.PgxConnIface) {
	t.Helper()
	mock, err := pgxmock.NewConn()
	require.NoError(t, err)
	store, err := New(Config{DB: mock})
	require.NoError(t, err)
	return store, mock
}

func makeContentJSON(text string) []byte {
	sc := storedContent{
		Parts: []storedPart{{Type: "text", Text: text}},
	}
	b, _ := json.Marshal(sc)
	return b
}

func makeAIContentJSON(text, modelID string, toolCalls []schema.ToolCall) []byte {
	sc := storedContent{
		Parts:     []storedPart{{Type: "text", Text: text}},
		ToolCalls: toolCalls,
		ModelID:   modelID,
	}
	b, _ := json.Marshal(sc)
	return b
}

func makeToolContentJSON(text, toolCallID string) []byte {
	sc := storedContent{
		Parts:      []storedPart{{Type: "text", Text: text}},
		ToolCallID: toolCallID,
	}
	b, _ := json.Marshal(sc)
	return b
}

func makeMetadataJSON(m map[string]any) []byte {
	b, _ := json.Marshal(m)
	return b
}

func TestNew(t *testing.T) {
	t.Run("nil db returns error", func(t *testing.T) {
		_, err := New(Config{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db is required")
	})

	t.Run("default table", func(t *testing.T) {
		mock, _ := pgxmock.NewConn()
		store, err := New(Config{DB: mock})
		require.NoError(t, err)
		assert.Equal(t, "messages", store.table)
	})

	t.Run("custom table", func(t *testing.T) {
		mock, _ := pgxmock.NewConn()
		store, err := New(Config{DB: mock, Table: "custom_messages"})
		require.NoError(t, err)
		assert.Equal(t, "custom_messages", store.table)
	})
}

func TestEnsureTable(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS messages").
		WillReturnResult(pgconn.NewCommandTag("CREATE TABLE"))

	err := store.EnsureTable(ctx)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAppend(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("INSERT INTO messages").
		WithArgs("human", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	err := store.Append(ctx, schema.NewHumanMessage("hello"))
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAppendAIMessage(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("INSERT INTO messages").
		WithArgs("ai", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	ai := &schema.AIMessage{
		Parts:   []schema.ContentPart{schema.TextPart{Text: "calling"}},
		ModelID: "gpt-4",
		ToolCalls: []schema.ToolCall{
			{ID: "tc1", Name: "search", Arguments: `{"q":"test"}`},
		},
	}
	err := store.Append(ctx, ai)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAppendToolMessage(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("INSERT INTO messages").
		WithArgs("tool", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	err := store.Append(ctx, schema.NewToolMessage("tc1", "result"))
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAppendSystemMessage(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("INSERT INTO messages").
		WithArgs("system", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	err := store.Append(ctx, schema.NewSystemMessage("you are helpful"))
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("human", makeContentJSON("hello world"), makeMetadataJSON(nil))

	mock.ExpectQuery("SELECT role, content, metadata FROM messages WHERE").
		WithArgs("%hello%", 10).
		WillReturnRows(rows)

	results, err := store.Search(ctx, "hello", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, schema.RoleHuman, results[0].GetRole())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchMultipleResults(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("human", makeContentJSON("test one"), makeMetadataJSON(nil)).
		AddRow("ai", makeContentJSON("test two"), makeMetadataJSON(nil))

	mock.ExpectQuery("SELECT role, content, metadata FROM messages WHERE").
		WithArgs("%test%", 10).
		WillReturnRows(rows)

	results, err := store.Search(ctx, "test", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAll(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("human", makeContentJSON("msg1"), makeMetadataJSON(nil)).
		AddRow("ai", makeContentJSON("msg2"), makeMetadataJSON(nil)).
		AddRow("human", makeContentJSON("msg3"), makeMetadataJSON(nil))

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnRows(rows)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 3)
	assert.Equal(t, schema.RoleHuman, msgs[0].GetRole())
	assert.Equal(t, schema.RoleAI, msgs[1].GetRole())
	assert.Equal(t, schema.RoleHuman, msgs[2].GetRole())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAllEmpty(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"})

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnRows(rows)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	assert.Empty(t, msgs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClear(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("DELETE FROM messages").
		WillReturnResult(pgconn.NewCommandTag("DELETE 2"))

	err := store.Clear(ctx)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAllMessageTypes(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	toolCalls := []schema.ToolCall{{ID: "tc1", Name: "search", Arguments: `{"q":"test"}`}}

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("system", makeContentJSON("be helpful"), makeMetadataJSON(nil)).
		AddRow("human", makeContentJSON("hello"), makeMetadataJSON(nil)).
		AddRow("ai", makeAIContentJSON("calling tool", "gpt-4", toolCalls), makeMetadataJSON(nil)).
		AddRow("tool", makeToolContentJSON("result", "tc1"), makeMetadataJSON(nil))

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnRows(rows)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 4)

	assert.Equal(t, schema.RoleSystem, msgs[0].GetRole())
	assert.Equal(t, schema.RoleHuman, msgs[1].GetRole())
	assert.Equal(t, schema.RoleAI, msgs[2].GetRole())
	assert.Equal(t, schema.RoleTool, msgs[3].GetRole())

	// Check AI message preserved fields.
	aiMsg, ok := msgs[2].(*schema.AIMessage)
	require.True(t, ok)
	assert.Equal(t, "gpt-4", aiMsg.ModelID)
	require.Len(t, aiMsg.ToolCalls, 1)
	assert.Equal(t, "search", aiMsg.ToolCalls[0].Name)

	// Check tool message preserved call ID.
	toolMsg, ok := msgs[3].(*schema.ToolMessage)
	require.True(t, ok)
	assert.Equal(t, "tc1", toolMsg.ToolCallID)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMetadataPreserved(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	meta := map[string]any{"key": "value", "num": float64(42)}

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("human", makeContentJSON("hello"), makeMetadataJSON(meta))

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnRows(rows)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "value", msgs[0].GetMetadata()["key"])
	assert.Equal(t, float64(42), msgs[0].GetMetadata()["num"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCustomTable(t *testing.T) {
	ctx := context.Background()
	mock, _ := pgxmock.NewConn()
	store, err := New(Config{DB: mock, Table: "custom_msgs"})
	require.NoError(t, err)

	mock.ExpectExec("INSERT INTO custom_msgs").
		WithArgs("human", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	err = store.Append(ctx, schema.NewHumanMessage("hello"))
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAppend_Error(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("INSERT INTO messages").
		WithArgs("human", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("connection refused"))

	err := store.Append(ctx, schema.NewHumanMessage("hello"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres: append:")
}

func TestSearch_Error(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectQuery("SELECT role, content, metadata FROM messages WHERE").
		WithArgs("%hello%", 10).
		WillReturnError(fmt.Errorf("query failed"))

	_, err := store.Search(ctx, "hello", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres: search:")
}

func TestAll_Error(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnError(fmt.Errorf("query failed"))

	_, err := store.All(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres: all:")
}

func TestClear_Error(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("DELETE FROM messages").
		WillReturnError(fmt.Errorf("delete failed"))

	err := store.Clear(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postgres: clear:")
}

func TestScanMessages_UnknownRole(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("observer", makeContentJSON("hello"), makeMetadataJSON(nil))

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnRows(rows)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	// Unknown role defaults to HumanMessage.
	assert.Equal(t, schema.RoleHuman, msgs[0].GetRole())
}

func TestScanMessages_NilMetadata(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("human", makeContentJSON("hello"), []byte("null"))

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnRows(rows)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Nil(t, msgs[0].GetMetadata())
}

func TestScanMessages_EmptyMetadata(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("human", makeContentJSON("hello"), []byte{})

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnRows(rows)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Nil(t, msgs[0].GetMetadata())
}

func TestEnsureTable_Error(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS messages").
		WillReturnError(fmt.Errorf("permission denied"))

	err := store.EnsureTable(ctx)
	assert.Error(t, err)
}

func TestAppendToolMessage_Roundtrip(t *testing.T) {
	ctx := context.Background()
	store, mock := newTestStore(t)

	mock.ExpectExec("INSERT INTO messages").
		WithArgs("tool", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	err := store.Append(ctx, schema.NewToolMessage("tc1", "result data"))
	require.NoError(t, err)

	rows := pgxmock.NewRows([]string{"role", "content", "metadata"}).
		AddRow("tool", makeToolContentJSON("result data", "tc1"), makeMetadataJSON(nil))

	mock.ExpectQuery("SELECT role, content, metadata FROM messages ORDER BY").
		WillReturnRows(rows)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	toolMsg, ok := msgs[0].(*schema.ToolMessage)
	require.True(t, ok)
	assert.Equal(t, "tc1", toolMsg.ToolCallID)
	require.NoError(t, mock.ExpectationsWereMet())
}
