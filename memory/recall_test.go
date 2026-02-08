package memory

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMessageStore is a test double for MessageStore.
type mockMessageStore struct {
	msgs []schema.Message
}

func (m *mockMessageStore) Append(_ context.Context, msg schema.Message) error {
	m.msgs = append(m.msgs, msg)
	return nil
}

func (m *mockMessageStore) Search(_ context.Context, query string, k int) ([]schema.Message, error) {
	var results []schema.Message
	for _, msg := range m.msgs {
		if matchesQuery(msg, query) {
			results = append(results, msg)
			if len(results) >= k {
				break
			}
		}
	}
	return results, nil
}

func (m *mockMessageStore) All(_ context.Context) ([]schema.Message, error) {
	cp := make([]schema.Message, len(m.msgs))
	copy(cp, m.msgs)
	return cp, nil
}

func (m *mockMessageStore) Clear(_ context.Context) error {
	m.msgs = nil
	return nil
}

func TestNewRecall(t *testing.T) {
	store := &mockMessageStore{}
	recall := NewRecall(store)

	assert.NotNil(t, recall)
	assert.Equal(t, store, recall.store)
}

func TestRecallSave(t *testing.T) {
	ctx := context.Background()
	store := &mockMessageStore{}
	recall := NewRecall(store)

	input := schema.NewHumanMessage("hello")
	output := schema.NewAIMessage("hi there")

	err := recall.Save(ctx, input, output)
	require.NoError(t, err)

	// Verify both messages were appended.
	assert.Len(t, store.msgs, 2)
	assert.Equal(t, schema.RoleHuman, store.msgs[0].GetRole())
	assert.Equal(t, schema.RoleAI, store.msgs[1].GetRole())
}

func TestRecallLoad(t *testing.T) {
	ctx := context.Background()
	store := &mockMessageStore{}
	recall := NewRecall(store)

	// Add some messages.
	msg1 := schema.NewHumanMessage("hello")
	msg2 := schema.NewAIMessage("hi there")
	msg3 := schema.NewHumanMessage("how are you?")
	msg4 := schema.NewAIMessage("I'm doing well, thanks!")

	require.NoError(t, recall.Save(ctx, msg1, msg2))
	require.NoError(t, recall.Save(ctx, msg3, msg4))

	t.Run("empty query returns all", func(t *testing.T) {
		msgs, err := recall.Load(ctx, "")
		require.NoError(t, err)
		assert.Len(t, msgs, 4)
	})

	t.Run("query searches messages", func(t *testing.T) {
		msgs, err := recall.Load(ctx, "hello")
		require.NoError(t, err)
		assert.Len(t, msgs, 1)
		assert.Contains(t, msgs[0].(*schema.HumanMessage).Text(), "hello")
	})

	t.Run("query with no matches", func(t *testing.T) {
		msgs, err := recall.Load(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})
}

func TestRecallSearch(t *testing.T) {
	// Recall memory does not support document search.
	ctx := context.Background()
	store := &mockMessageStore{}
	recall := NewRecall(store)

	docs, err := recall.Search(ctx, "any query", 10)
	require.NoError(t, err)
	assert.Nil(t, docs)
}

func TestRecallClear(t *testing.T) {
	ctx := context.Background()
	store := &mockMessageStore{}
	recall := NewRecall(store)

	// Add messages.
	require.NoError(t, recall.Save(ctx,
		schema.NewHumanMessage("hello"),
		schema.NewAIMessage("hi")))

	// Clear.
	err := recall.Clear(ctx)
	require.NoError(t, err)

	// Verify store is empty.
	msgs, err := recall.Load(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestRecallRegistryEntry(t *testing.T) {
	// Verify "recall" is registered via init().
	mem, err := New("recall", config.ProviderConfig{Provider: "recall"})
	require.NoError(t, err)

	recall, ok := mem.(*Recall)
	require.True(t, ok)
	assert.NotNil(t, recall.store)
}

func TestInlineMessageStore(t *testing.T) {
	ctx := context.Background()
	store := &inlineMessageStore{}

	t.Run("append", func(t *testing.T) {
		msg := schema.NewHumanMessage("hello")
		err := store.Append(ctx, msg)
		require.NoError(t, err)
		assert.Len(t, store.msgs, 1)
	})

	t.Run("all", func(t *testing.T) {
		msgs, err := store.All(ctx)
		require.NoError(t, err)
		assert.Len(t, msgs, 1)
	})

	t.Run("search", func(t *testing.T) {
		store.msgs = nil
		_ = store.Append(ctx, schema.NewHumanMessage("hello world"))
		_ = store.Append(ctx, schema.NewAIMessage("hi there"))
		_ = store.Append(ctx, schema.NewHumanMessage("goodbye"))

		// Search for "hello".
		results, err := store.Search(ctx, "hello", 10)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].(*schema.HumanMessage).Text(), "hello")

		// Search respects k limit.
		_ = store.Append(ctx, schema.NewHumanMessage("hello again"))
		results, err = store.Search(ctx, "hello", 1)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("clear", func(t *testing.T) {
		err := store.Clear(ctx)
		require.NoError(t, err)
		assert.Nil(t, store.msgs)
	})
}
