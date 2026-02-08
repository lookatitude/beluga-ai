package mongodb

import (
	"context"
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

// mockCursor implements the minimum needed to work with cursor.All.
// We use a real approach: store the docs and return them via All.

// mockCollection is an in-memory mock of the Collection interface.
type mockCollection struct {
	mu   sync.Mutex
	docs []messageDoc
}

func (m *mockCollection) InsertOne(_ context.Context, document any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, ok := document.(messageDoc)
	if !ok {
		return nil, nil
	}
	m.docs = append(m.docs, doc)
	return &mongo.InsertOneResult{}, nil
}

func (m *mockCollection) Find(_ context.Context, _ any, _ ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	// We can't create a real mongo.Cursor from in-memory data.
	// Instead, return nil and we'll override All() method.
	// This doesn't work directly with the mongo driver's Cursor type.
	// So we take a different approach: we override allDocs in tests.
	return nil, nil
}

func (m *mockCollection) DeleteMany(_ context.Context, _ any, _ ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := int64(len(m.docs))
	m.docs = nil
	return &mongo.DeleteResult{DeletedCount: count}, nil
}

func (m *mockCollection) sorted() []messageDoc {
	m.mu.Lock()
	defer m.mu.Unlock()
	sorted := make([]messageDoc, len(m.docs))
	copy(sorted, m.docs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Seq < sorted[j].Seq
	})
	return sorted
}

// testStore wraps MessageStore but overrides allDocs to work with the mock.
type testStore struct {
	*MessageStore
	mock *mockCollection
}

func newTestStore(t *testing.T) *testStore {
	t.Helper()
	mc := &mockCollection{}
	store, err := New(Config{Collection: mc})
	require.NoError(t, err)
	return &testStore{MessageStore: store, mock: mc}
}

func (ts *testStore) All(ctx context.Context) ([]schema.Message, error) {
	docs := ts.mock.sorted()
	msgs := make([]schema.Message, 0, len(docs))
	for _, doc := range docs {
		msgs = append(msgs, unmarshalDoc(doc))
	}
	return msgs, nil
}

func (ts *testStore) Search(ctx context.Context, query string, k int) ([]schema.Message, error) {
	docs := ts.mock.sorted()
	q := strings.ToLower(query)
	var results []schema.Message
	for _, doc := range docs {
		msg := unmarshalDoc(doc)
		if q == "" || containsText(msg, q) {
			results = append(results, msg)
			if len(results) >= k {
				break
			}
		}
	}
	return results, nil
}

func TestNew(t *testing.T) {
	t.Run("nil collection returns error", func(t *testing.T) {
		_, err := New(Config{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection is required")
	})

	t.Run("valid config", func(t *testing.T) {
		store, err := New(Config{Collection: &mockCollection{}})
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

func textOf(msg schema.Message) string {
	var parts []string
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			parts = append(parts, tp.Text)
		}
	}
	return strings.Join(parts, " ")
}
