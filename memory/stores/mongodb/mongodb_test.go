package mongodb

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Compile-time interface check.
var _ memory.MessageStore = (*MessageStore)(nil)

// cursorCollection is an in-memory mock that returns real mongo.Cursor objects
// via NewCursorFromDocuments, enabling full testing of allDocs/All/Search.
type cursorCollection struct {
	mu   sync.Mutex
	docs []messageDoc
}

func (m *cursorCollection) InsertOne(_ context.Context, document any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, ok := document.(messageDoc)
	if !ok {
		return nil, nil
	}
	m.docs = append(m.docs, doc)
	return &mongo.InsertOneResult{}, nil
}

func (m *cursorCollection) Find(_ context.Context, _ any, _ ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Sort by sequence number.
	sorted := make([]messageDoc, len(m.docs))
	copy(sorted, m.docs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Seq < sorted[j].Seq
	})

	// Convert to BSON documents for the cursor.
	bsonDocs := make([]any, len(sorted))
	for i, doc := range sorted {
		raw, err := bson.Marshal(doc)
		if err != nil {
			return nil, err
		}
		bsonDocs[i] = bson.Raw(raw)
	}

	return mongo.NewCursorFromDocuments(bsonDocs, nil, nil)
}

func (m *cursorCollection) DeleteMany(_ context.Context, _ any, _ ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := int64(len(m.docs))
	m.docs = nil
	return &mongo.DeleteResult{DeletedCount: count}, nil
}

func newTestStore(t *testing.T) *MessageStore {
	t.Helper()
	mc := &cursorCollection{}
	store, err := New(Config{Collection: mc})
	require.NoError(t, err)
	return store
}

func TestNew(t *testing.T) {
	t.Run("nil collection returns error", func(t *testing.T) {
		_, err := New(Config{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection is required")
	})

	t.Run("valid config", func(t *testing.T) {
		store, err := New(Config{Collection: &cursorCollection{}})
		require.NoError(t, err)
		assert.NotNil(t, store)
	})
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

	err := store.Append(ctx, schema.NewSystemMessage("you are helpful"))
	require.NoError(t, err)

	err = store.Append(ctx, schema.NewHumanMessage("hello"))
	require.NoError(t, err)

	ai := &schema.AIMessage{
		Parts:   []schema.ContentPart{schema.TextPart{Text: "calling tool"}},
		ModelID: "gpt-4",
		ToolCalls: []schema.ToolCall{
			{ID: "tc1", Name: "search", Arguments: `{"q":"test"}`},
		},
	}
	err = store.Append(ctx, ai)
	require.NoError(t, err)

	err = store.Append(ctx, schema.NewToolMessage("tc1", "result"))
	require.NoError(t, err)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 4)

	assert.Equal(t, schema.RoleSystem, msgs[0].GetRole())
	assert.Equal(t, schema.RoleHuman, msgs[1].GetRole())
	assert.Equal(t, schema.RoleAI, msgs[2].GetRole())
	assert.Equal(t, schema.RoleTool, msgs[3].GetRole())

	aiMsg, ok := msgs[2].(*schema.AIMessage)
	require.True(t, ok)
	assert.Equal(t, "gpt-4", aiMsg.ModelID)
	require.Len(t, aiMsg.ToolCalls, 1)
	assert.Equal(t, "search", aiMsg.ToolCalls[0].Name)

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
		{"match multiple", "o", 10, 4},
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
	store := newTestStore(t)

	msgs, err := store.All(ctx)
	require.NoError(t, err)
	assert.Empty(t, msgs)

	_ = store.Append(ctx, schema.NewHumanMessage("msg1"))
	_ = store.Append(ctx, schema.NewAIMessage("msg2"))
	_ = store.Append(ctx, schema.NewHumanMessage("msg3"))

	msgs, err = store.All(ctx)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)

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

func TestSearchOrder(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	_ = store.Append(ctx, schema.NewHumanMessage("test one"))
	_ = store.Append(ctx, schema.NewAIMessage("test two"))
	_ = store.Append(ctx, schema.NewHumanMessage("test three"))

	results, err := store.Search(ctx, "test", 10)
	require.NoError(t, err)
	require.Len(t, results, 3)

	assert.Contains(t, textOf(results[0]), "one")
	assert.Contains(t, textOf(results[1]), "two")
	assert.Contains(t, textOf(results[2]), "three")
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	store := newTestStore(t)

	ai := &schema.AIMessage{
		Parts:   []schema.ContentPart{schema.TextPart{Text: "response"}},
		ModelID: "claude-3",
		ToolCalls: []schema.ToolCall{
			{ID: "tc1", Name: "search", Arguments: `{"q":"test"}`},
		},
		Metadata: map[string]any{"source": "test"},
	}

	doc := store.marshalMessage(ai)
	result := unmarshalDoc(doc)

	aiResult, ok := result.(*schema.AIMessage)
	require.True(t, ok)
	assert.Equal(t, "claude-3", aiResult.ModelID)
	require.Len(t, aiResult.ToolCalls, 1)
	assert.Equal(t, "search", aiResult.ToolCalls[0].Name)
	assert.Equal(t, "test", aiResult.Metadata["source"])
}

func TestToBsonM(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		result := toBsonM(nil)
		assert.Nil(t, result)
	})

	t.Run("non-nil map", func(t *testing.T) {
		m := map[string]any{"key": "value"}
		result := toBsonM(m)
		assert.Equal(t, bson.M{"key": "value"}, result)
	})
}

func TestFromBsonM(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		result := fromBsonM(nil)
		assert.Nil(t, result)
	})

	t.Run("non-nil map", func(t *testing.T) {
		m := bson.M{"key": "value"}
		result := fromBsonM(m)
		assert.Equal(t, map[string]any{"key": "value"}, result)
	})
}

// errorCollection is a mock that returns errors for all operations.
type errorCollection struct {
	insertErr  error
	findErr    error
	deleteErr  error
}

func (e *errorCollection) InsertOne(_ context.Context, _ any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	return nil, e.insertErr
}

func (e *errorCollection) Find(_ context.Context, _ any, _ ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	return nil, e.findErr
}

func (e *errorCollection) DeleteMany(_ context.Context, _ any, _ ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error) {
	return nil, e.deleteErr
}

func TestAppend_InsertError(t *testing.T) {
	ec := &errorCollection{insertErr: fmt.Errorf("connection refused")}
	store, err := New(Config{Collection: ec})
	require.NoError(t, err)

	err = store.Append(context.Background(), schema.NewHumanMessage("hello"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mongodb: insert:")
}

func TestAll_FindError(t *testing.T) {
	ec := &errorCollection{findErr: fmt.Errorf("find failed")}
	store, err := New(Config{Collection: ec})
	require.NoError(t, err)

	msgs, err := store.All(context.Background())
	assert.Error(t, err)
	assert.Nil(t, msgs)
	assert.Contains(t, err.Error(), "mongodb: find:")
}

func TestSearch_FindError(t *testing.T) {
	ec := &errorCollection{findErr: fmt.Errorf("find failed")}
	store, err := New(Config{Collection: ec})
	require.NoError(t, err)

	msgs, err := store.Search(context.Background(), "query", 10)
	assert.Error(t, err)
	assert.Nil(t, msgs)
	assert.Contains(t, err.Error(), "mongodb: find:")
}

func TestClear_DeleteError(t *testing.T) {
	ec := &errorCollection{deleteErr: fmt.Errorf("delete failed")}
	store, err := New(Config{Collection: ec})
	require.NoError(t, err)

	err = store.Clear(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mongodb: clear:")
}

func TestUnmarshalDoc_DefaultRole(t *testing.T) {
	doc := messageDoc{
		Role:  "observer",
		Parts: []partDoc{{Type: "text", Text: "hello"}},
	}
	msg := unmarshalDoc(doc)
	// Unknown role defaults to HumanMessage.
	assert.Equal(t, schema.RoleHuman, msg.GetRole())
	human, ok := msg.(*schema.HumanMessage)
	require.True(t, ok)
	require.Len(t, human.Parts, 1)
	tp, ok := human.Parts[0].(schema.TextPart)
	require.True(t, ok)
	assert.Equal(t, "hello", tp.Text)
}

func TestUnmarshalDoc_NilMetadata(t *testing.T) {
	doc := messageDoc{
		Role:  "human",
		Parts: []partDoc{{Type: "text", Text: "hello"}},
	}
	msg := unmarshalDoc(doc)
	assert.Nil(t, msg.GetMetadata())
}

func TestContainsText_NoMatch(t *testing.T) {
	msg := schema.NewHumanMessage("hello world")
	assert.False(t, containsText(msg, "xyz"))
}

func TestContainsText_Empty(t *testing.T) {
	msg := &schema.HumanMessage{Parts: []schema.ContentPart{}}
	assert.False(t, containsText(msg, "any"))
}

func TestMarshalMessage_SystemMessage(t *testing.T) {
	store := newTestStore(t)
	msg := schema.NewSystemMessage("be helpful")
	doc := store.marshalMessage(msg)
	assert.Equal(t, "system", doc.Role)
	assert.NotZero(t, doc.Seq)
}

func TestMarshalMessage_ToolMessage(t *testing.T) {
	store := newTestStore(t)
	msg := schema.NewToolMessage("tc1", "tool result")
	doc := store.marshalMessage(msg)
	assert.Equal(t, "tool", doc.Role)
	assert.Equal(t, "tc1", doc.ToolCallID)
}

func TestUnmarshalDoc_SystemMessage(t *testing.T) {
	doc := messageDoc{
		Role:     "system",
		Parts:    []partDoc{{Type: "text", Text: "be helpful"}},
		Metadata: bson.M{"key": "val"},
	}
	msg := unmarshalDoc(doc)
	assert.Equal(t, schema.RoleSystem, msg.GetRole())
	sys, ok := msg.(*schema.SystemMessage)
	require.True(t, ok)
	assert.Equal(t, "val", sys.Metadata["key"])
}

func TestUnmarshalDoc_ToolMessage(t *testing.T) {
	doc := messageDoc{
		Role:       "tool",
		Parts:      []partDoc{{Type: "text", Text: "result"}},
		ToolCallID: "tc1",
	}
	msg := unmarshalDoc(doc)
	assert.Equal(t, schema.RoleTool, msg.GetRole())
	toolMsg, ok := msg.(*schema.ToolMessage)
	require.True(t, ok)
	assert.Equal(t, "tc1", toolMsg.ToolCallID)
}

func TestSequenceIncrement(t *testing.T) {
	store := newTestStore(t)
	msg1 := store.marshalMessage(schema.NewHumanMessage("first"))
	msg2 := store.marshalMessage(schema.NewHumanMessage("second"))
	assert.Less(t, msg1.Seq, msg2.Seq)
}

func textOf(msg schema.Message) string {
	var parts []string
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			parts = append(parts, tp.Text)
		}
	}
	return strings.Join(parts, " ")
}
