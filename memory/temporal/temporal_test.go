package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMemory(t *testing.T) (*TemporalMemory, *InMemoryStore) {
	t.Helper()
	store := NewInMemoryStore()
	tm := New(store)
	return tm, store
}

func TestTemporalMemory_Save(t *testing.T) {
	tm, store := newTestMemory(t)
	ctx := context.Background()

	input := schema.NewHumanMessage("hello")
	output := schema.NewAIMessage("hi there")

	err := tm.Save(ctx, input, output)
	require.NoError(t, err)

	// Verify entities were created.
	store.mu.RLock()
	assert.Len(t, store.entities, 2)
	assert.Len(t, store.relations, 1)
	store.mu.RUnlock()
}

func TestTemporalMemory_Save_ContextCancelled(t *testing.T) {
	tm, _ := newTestMemory(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := tm.Save(ctx, schema.NewHumanMessage("hello"), schema.NewAIMessage("hi"))
	assert.ErrorIs(t, err, context.Canceled)
}

func TestTemporalMemory_Load(t *testing.T) {
	tm, _ := newTestMemory(t)
	ctx := context.Background()

	// Save a conversation turn.
	err := tm.Save(ctx, schema.NewHumanMessage("hello"), schema.NewAIMessage("hi there"))
	require.NoError(t, err)

	// Load messages matching "message" (the entity type).
	msgs, err := tm.Load(ctx, "message")
	require.NoError(t, err)
	assert.Len(t, msgs, 2) // input + output entities
}

func TestTemporalMemory_Load_Empty(t *testing.T) {
	tm, _ := newTestMemory(t)
	ctx := context.Background()

	msgs, err := tm.Load(ctx, "anything")
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestTemporalMemory_LoadAt(t *testing.T) {
	store := NewInMemoryStore()
	tm := New(store)
	ctx := context.Background()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Add message entities directly for controlled temporal testing.
	require.NoError(t, store.AddEntity(ctx, memory.Entity{
		ID:         "msg-1",
		Type:       "message",
		Properties: map[string]any{"role": "human", "text": "hello"},
		CreatedAt:  t1,
	}))

	// Add a temporal relation valid from t1.
	require.NoError(t, store.AddTemporalRelation(ctx, memory.Relation{
		From:       "msg-1",
		To:         "msg-1",
		Type:       "message",
		ValidAt:    t1,
		Properties: map[string]any{"id": "rel-msg-1"},
	}))

	t.Run("load at valid time", func(t *testing.T) {
		msgs, err := tm.LoadAt(ctx, "message", t1.Add(time.Hour))
		require.NoError(t, err)
		assert.NotEmpty(t, msgs)
	})

	t.Run("zero time errors", func(t *testing.T) {
		_, err := tm.LoadAt(ctx, "message", time.Time{})
		require.Error(t, err)
	})

	t.Run("context cancelled", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err := tm.LoadAt(cancelCtx, "message", t1)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestTemporalMemory_Search(t *testing.T) {
	tm, _ := newTestMemory(t)
	ctx := context.Background()

	// Save some data.
	require.NoError(t, tm.Save(ctx, schema.NewHumanMessage("what is Go?"), schema.NewAIMessage("Go is a programming language")))

	t.Run("search with results", func(t *testing.T) {
		docs, err := tm.Search(ctx, "message", 10)
		require.NoError(t, err)
		assert.NotEmpty(t, docs)
		for _, doc := range docs {
			assert.NotEmpty(t, doc.ID)
		}
	})

	t.Run("search with k=0 returns nil", func(t *testing.T) {
		docs, err := tm.Search(ctx, "message", 0)
		require.NoError(t, err)
		assert.Nil(t, docs)
	})

	t.Run("search with no match", func(t *testing.T) {
		docs, err := tm.Search(ctx, "zzzzz", 10)
		require.NoError(t, err)
		assert.Empty(t, docs)
	})
}

func TestTemporalMemory_Clear(t *testing.T) {
	tm, store := newTestMemory(t)
	ctx := context.Background()

	require.NoError(t, tm.Save(ctx, schema.NewHumanMessage("hello"), schema.NewAIMessage("hi")))

	err := tm.Clear(ctx)
	require.NoError(t, err)

	store.mu.RLock()
	assert.Empty(t, store.entities)
	assert.Empty(t, store.relations)
	store.mu.RUnlock()
}

func TestTemporalMemory_ResolveConflicts(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	var hookCalled bool
	tm := New(store, WithHooks(Hooks{
		OnConflictResolved: func(_ context.Context, invalidated []memory.Relation, _ memory.Relation) {
			hookCalled = true
			assert.NotEmpty(t, invalidated)
		},
	}))

	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "alice", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "acme", Type: "company"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "globex", Type: "company"}))

	t1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Alice worked at ACME from t1.
	require.NoError(t, store.AddTemporalRelation(ctx, memory.Relation{
		From:       "alice",
		To:         "acme",
		Type:       "works_at",
		ValidAt:    t1,
		Properties: map[string]any{"id": "rel-1"},
	}))

	// Now Alice works at Globex from t2. Resolve conflicts.
	newRel := &memory.Relation{
		From:    "alice",
		To:      "acme",
		Type:    "works_at",
		ValidAt: t2,
	}

	invalidated, err := tm.ResolveConflicts(ctx, newRel)
	require.NoError(t, err)
	assert.Len(t, invalidated, 1)
	assert.True(t, hookCalled, "OnConflictResolved hook should have been called")
}

func TestTemporalMemory_ResolveConflicts_NilRelation(t *testing.T) {
	tm, _ := newTestMemory(t)
	_, err := tm.ResolveConflicts(context.Background(), nil)
	require.Error(t, err)
}

func TestTemporalMemory_Store(t *testing.T) {
	store := NewInMemoryStore()
	tm := New(store)
	assert.Equal(t, store, tm.Store())
}

func TestTemporalMemory_CompileTimeCheck(t *testing.T) {
	var _ memory.Memory = (*TemporalMemory)(nil)
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{name: "short string unchanged", input: "hello", maxLen: 10, want: "hello"},
		{name: "exact length unchanged", input: "hello", maxLen: 5, want: "hello"},
		{name: "truncated with ellipsis", input: "hello world", maxLen: 8, want: "hello..."},
		{name: "very short maxLen", input: "hello", maxLen: 2, want: "he"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractText(t *testing.T) {
	msg := schema.NewHumanMessage("hello world")
	assert.Equal(t, "hello world", extractText(msg))
}

func TestEntityToMessage(t *testing.T) {
	tests := []struct {
		name     string
		entity   memory.Entity
		wantRole schema.Role
		wantNil  bool
	}{
		{
			name: "human message",
			entity: memory.Entity{
				Type:       "message",
				Properties: map[string]any{"role": "human", "text": "hello"},
			},
			wantRole: schema.RoleHuman,
		},
		{
			name: "ai message",
			entity: memory.Entity{
				Type:       "message",
				Properties: map[string]any{"role": "ai", "text": "hi"},
			},
			wantRole: schema.RoleAI,
		},
		{
			name: "system message",
			entity: memory.Entity{
				Type:       "message",
				Properties: map[string]any{"role": "system", "text": "you are helpful"},
			},
			wantRole: schema.RoleSystem,
		},
		{
			name: "unknown role with text defaults to human",
			entity: memory.Entity{
				Type:       "message",
				Properties: map[string]any{"role": "unknown", "text": "data"},
			},
			wantRole: schema.RoleHuman,
		},
		{
			name: "unknown role without text returns nil",
			entity: memory.Entity{
				Type:       "message",
				Properties: map[string]any{"role": "unknown"},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := entityToMessage(tt.entity)
			if tt.wantNil {
				assert.Nil(t, msg)
				return
			}
			require.NotNil(t, msg)
			assert.Equal(t, tt.wantRole, msg.GetRole())
		})
	}
}
