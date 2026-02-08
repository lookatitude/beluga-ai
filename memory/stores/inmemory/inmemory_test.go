package inmemory

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface checks.
var (
	_ memory.MessageStore = (*MessageStore)(nil)
	_ memory.GraphStore   = (*GraphStore)(nil)
)

func TestNewMessageStore(t *testing.T) {
	store := NewMessageStore()
	assert.NotNil(t, store)
	assert.Empty(t, store.msgs)
}

func TestMessageStoreAppend(t *testing.T) {
	ctx := context.Background()
	store := NewMessageStore()

	msg1 := schema.NewHumanMessage("hello")
	msg2 := schema.NewAIMessage("hi")

	err := store.Append(ctx, msg1)
	require.NoError(t, err)
	assert.Len(t, store.msgs, 1)

	err = store.Append(ctx, msg2)
	require.NoError(t, err)
	assert.Len(t, store.msgs, 2)

	assert.Equal(t, schema.RoleHuman, store.msgs[0].GetRole())
	assert.Equal(t, schema.RoleAI, store.msgs[1].GetRole())
}

func TestMessageStoreSearch(t *testing.T) {
	ctx := context.Background()
	store := NewMessageStore()

	// Add test messages.
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
		wantText  string
	}{
		{
			name:      "match single",
			query:     "hello",
			k:         10,
			wantCount: 1,
			wantText:  "hello",
		},
		{
			name:      "case insensitive",
			query:     "HELLO",
			k:         10,
			wantCount: 1,
			wantText:  "hello",
		},
		{
			name:      "match multiple",
			query:     "o", // matches "hello", "how", "doing", "goodbye"
			k:         10,
			wantCount: 4,
		},
		{
			name:      "limit results",
			query:     "o",
			k:         2,
			wantCount: 2,
		},
		{
			name:      "no match",
			query:     "xyz",
			k:         10,
			wantCount: 0,
		},
		{
			name:      "empty query matches all",
			query:     "",
			k:         10,
			wantCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Search(ctx, tt.query, tt.k)
			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)

			if tt.wantText != "" && len(results) > 0 {
				text := extractText(results[0])
				assert.Contains(t, text, tt.wantText)
			}
		})
	}
}

func TestMessageStoreAll(t *testing.T) {
	ctx := context.Background()
	store := NewMessageStore()

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
	assert.Contains(t, extractText(msgs[0]), "msg1")
	assert.Contains(t, extractText(msgs[1]), "msg2")
	assert.Contains(t, extractText(msgs[2]), "msg3")

	// Verify copy semantics (modifying returned slice doesn't affect store).
	msgs[0] = schema.NewHumanMessage("modified")
	msgs2, _ := store.All(ctx)
	assert.Contains(t, extractText(msgs2[0]), "msg1")
}

func TestMessageStoreClear(t *testing.T) {
	ctx := context.Background()
	store := NewMessageStore()

	// Add messages.
	_ = store.Append(ctx, schema.NewHumanMessage("hello"))
	_ = store.Append(ctx, schema.NewAIMessage("hi"))

	// Clear.
	err := store.Clear(ctx)
	require.NoError(t, err)

	// Verify empty.
	msgs, err := store.All(ctx)
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestMessageStoreConcurrency(t *testing.T) {
	// Test concurrent reads and writes.
	ctx := context.Background()
	store := NewMessageStore()

	done := make(chan bool)

	// Concurrent writes.
	for i := 0; i < 10; i++ {
		go func(n int) {
			msg := schema.NewHumanMessage("message")
			_ = store.Append(ctx, msg)
			done <- true
		}(i)
	}

	// Concurrent reads.
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = store.All(ctx)
			_, _ = store.Search(ctx, "message", 5)
			done <- true
		}()
	}

	// Wait for all goroutines.
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify store has all messages.
	msgs, err := store.All(ctx)
	require.NoError(t, err)
	assert.Len(t, msgs, 10)
}

func TestNewGraphStore(t *testing.T) {
	store := NewGraphStore()
	assert.NotNil(t, store)
	assert.NotNil(t, store.entities)
	assert.Empty(t, store.relations)
}

func TestGraphStoreAddEntity(t *testing.T) {
	ctx := context.Background()
	store := NewGraphStore()

	entity1 := memory.Entity{
		ID:   "person-1",
		Type: "person",
		Properties: map[string]any{
			"name": "Alice",
		},
	}

	err := store.AddEntity(ctx, entity1)
	require.NoError(t, err)
	assert.Len(t, store.entities, 1)
	assert.Equal(t, "Alice", store.entities["person-1"].Properties["name"])

	// Update existing entity.
	entity1Updated := memory.Entity{
		ID:   "person-1",
		Type: "person",
		Properties: map[string]any{
			"name": "Alice Smith",
		},
	}

	err = store.AddEntity(ctx, entity1Updated)
	require.NoError(t, err)
	assert.Len(t, store.entities, 1)
	assert.Equal(t, "Alice Smith", store.entities["person-1"].Properties["name"])
}

func TestGraphStoreAddRelation(t *testing.T) {
	ctx := context.Background()
	store := NewGraphStore()

	err := store.AddRelation(ctx, "person-1", "company-1", "works_at", map[string]any{
		"since": "2020",
	})
	require.NoError(t, err)
	assert.Len(t, store.relations, 1)
	assert.Equal(t, "person-1", store.relations[0].From)
	assert.Equal(t, "company-1", store.relations[0].To)
	assert.Equal(t, "works_at", store.relations[0].Type)
	assert.Equal(t, "2020", store.relations[0].Properties["since"])
}

func TestGraphStoreQuery(t *testing.T) {
	ctx := context.Background()
	store := NewGraphStore()

	// Add test data.
	_ = store.AddEntity(ctx, memory.Entity{ID: "1", Type: "person"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "2", Type: "person"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "3", Type: "company"})
	_ = store.AddRelation(ctx, "1", "3", "works_at", nil)

	tests := []struct {
		name          string
		query         string
		wantEntities  int
		wantRelations int
	}{
		{
			name:          "type query person",
			query:         "type:person",
			wantEntities:  2,
			wantRelations: 0,
		},
		{
			name:          "type query company",
			query:         "type:company",
			wantEntities:  1,
			wantRelations: 0,
		},
		{
			name:          "case insensitive type",
			query:         "type:PERSON",
			wantEntities:  2,
			wantRelations: 0,
		},
		{
			name:          "default query returns all",
			query:         "anything",
			wantEntities:  3,
			wantRelations: 1,
		},
		{
			name:          "empty query returns all",
			query:         "",
			wantEntities:  3,
			wantRelations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Query(ctx, tt.query)
			require.NoError(t, err)
			assert.Len(t, results, 1) // Always returns one GraphResult
			assert.Len(t, results[0].Entities, tt.wantEntities)
			assert.Len(t, results[0].Relations, tt.wantRelations)
		})
	}
}

func TestGraphStoreNeighbors(t *testing.T) {
	ctx := context.Background()
	store := NewGraphStore()

	// Build a graph: A -> B -> C -> D
	//                A -> E
	_ = store.AddEntity(ctx, memory.Entity{ID: "A", Type: "node"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "B", Type: "node"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "C", Type: "node"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "D", Type: "node"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "E", Type: "node"})

	_ = store.AddRelation(ctx, "A", "B", "connects", nil)
	_ = store.AddRelation(ctx, "B", "C", "connects", nil)
	_ = store.AddRelation(ctx, "C", "D", "connects", nil)
	_ = store.AddRelation(ctx, "A", "E", "connects", nil)

	tests := []struct {
		name          string
		entityID      string
		depth         int
		wantEntities  int
		wantRelations int
	}{
		{
			name:          "depth 1 from A",
			entityID:      "A",
			depth:         1,
			wantEntities:  2, // B, E
			wantRelations: 2, // A->B, A->E
		},
		{
			name:          "depth 2 from A",
			entityID:      "A",
			depth:         2,
			wantEntities:  3, // B, E, C
			wantRelations: 5, // A->B, A->E (depth 1), then A->B again, A->E again, B->C (depth 2)
		},
		{
			name:          "depth 0 uses default 1",
			entityID:      "A",
			depth:         0,
			wantEntities:  2, // B, E
			wantRelations: 2,
		},
		{
			name:          "negative depth uses default 1",
			entityID:      "A",
			depth:         -1,
			wantEntities:  2,
			wantRelations: 2,
		},
		{
			name:          "leaf node has no neighbors",
			entityID:      "D",
			depth:         1,
			wantEntities:  1, // C (reverse direction)
			wantRelations: 1, // C->D
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, relations, err := store.Neighbors(ctx, tt.entityID, tt.depth)
			require.NoError(t, err)
			assert.Len(t, entities, tt.wantEntities)
			assert.Len(t, relations, tt.wantRelations)
		})
	}
}

func TestGraphStoreNeighborsBidirectional(t *testing.T) {
	// Test that Neighbors traverses in both directions.
	ctx := context.Background()
	store := NewGraphStore()

	_ = store.AddEntity(ctx, memory.Entity{ID: "A", Type: "node"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "B", Type: "node"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "C", Type: "node"})

	// A -> B <- C (B is in the middle)
	_ = store.AddRelation(ctx, "A", "B", "connects", nil)
	_ = store.AddRelation(ctx, "C", "B", "connects", nil)

	// From B, should find both A and C.
	entities, relations, err := store.Neighbors(ctx, "B", 1)
	require.NoError(t, err)
	assert.Len(t, entities, 2) // A, C
	assert.Len(t, relations, 2)

	// Verify we found both A and C.
	ids := make(map[string]bool)
	for _, e := range entities {
		ids[e.ID] = true
	}
	assert.True(t, ids["A"])
	assert.True(t, ids["C"])
}

func TestGraphStoreNeighborsNoDuplicates(t *testing.T) {
	// Test that duplicate entities are not returned.
	ctx := context.Background()
	store := NewGraphStore()

	_ = store.AddEntity(ctx, memory.Entity{ID: "A", Type: "node"})
	_ = store.AddEntity(ctx, memory.Entity{ID: "B", Type: "node"})

	// Multiple relations between A and B.
	_ = store.AddRelation(ctx, "A", "B", "connects", nil)
	_ = store.AddRelation(ctx, "A", "B", "also_connects", nil)

	entities, relations, err := store.Neighbors(ctx, "A", 1)
	require.NoError(t, err)
	assert.Len(t, entities, 1) // B only once
	assert.Len(t, relations, 2) // Both relations
}

func TestGraphStoreConcurrency(t *testing.T) {
	// Test concurrent operations.
	ctx := context.Background()
	store := NewGraphStore()

	done := make(chan bool)

	// Concurrent entity additions.
	for i := 0; i < 10; i++ {
		go func(n int) {
			entity := memory.Entity{
				ID:   string(rune('A' + n)),
				Type: "node",
			}
			_ = store.AddEntity(ctx, entity)
			done <- true
		}(i)
	}

	// Concurrent relation additions.
	for i := 0; i < 10; i++ {
		go func(n int) {
			_ = store.AddRelation(ctx, "A", "B", "connects", nil)
			done <- true
		}(i)
	}

	// Concurrent reads.
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = store.Query(ctx, "")
			_, _, _ = store.Neighbors(ctx, "A", 1)
			done <- true
		}()
	}

	// Wait for all goroutines.
	for i := 0; i < 30; i++ {
		<-done
	}

	// Verify store has entities.
	assert.NotEmpty(t, store.entities)
}

// Helper function to extract text from a message.
func extractText(msg schema.Message) string {
	switch m := msg.(type) {
	case *schema.HumanMessage:
		return m.Text()
	case *schema.AIMessage:
		return m.Text()
	case *schema.SystemMessage:
		return m.Text()
	case *schema.ToolMessage:
		return m.Text()
	default:
		return ""
	}
}
