package neo4j

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/stretchr/testify/assert"
)

// mockRunner implements sessionRunner for testing.
type mockRunner struct {
	mu        sync.Mutex
	writes    []writeCall
	readData  []record
	writeErr  error
	readErr   error
	closeErr  error
	closed    bool
}

type writeCall struct {
	cypher string
	params map[string]any
}

func (r *mockRunner) executeWrite(_ context.Context, cypher string, params map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes = append(r.writes, writeCall{cypher: cypher, params: params})
	return r.writeErr
}

func (r *mockRunner) executeRead(_ context.Context, _ string, _ map[string]any) ([]record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.readErr != nil {
		return nil, r.readErr
	}
	return r.readData, nil
}

func (r *mockRunner) close(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return r.closeErr
}

func newMockStore() (*GraphStore, *mockRunner) {
	runner := &mockRunner{}
	store := newWithRunner(runner)
	return store, runner
}

func TestAddEntity(t *testing.T) {
	store, runner := newMockStore()
	ctx := context.Background()

	entity := memory.Entity{
		ID:         "alice",
		Type:       "person",
		Properties: map[string]any{"name": "Alice", "age": 30},
	}

	if err := store.AddEntity(ctx, entity); err != nil {
		t.Fatalf("AddEntity: %v", err)
	}

	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(runner.writes))
	}
	if runner.writes[0].params["id"] != "alice" {
		t.Errorf("expected id 'alice', got %v", runner.writes[0].params["id"])
	}
	if runner.writes[0].params["type"] != "person" {
		t.Errorf("expected type 'person', got %v", runner.writes[0].params["type"])
	}
}

func TestAddEntity_Error(t *testing.T) {
	store, runner := newMockStore()
	runner.writeErr = fmt.Errorf("connection refused")

	err := store.AddEntity(context.Background(), memory.Entity{ID: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddRelation(t *testing.T) {
	store, runner := newMockStore()
	ctx := context.Background()

	props := map[string]any{"since": "2024"}
	if err := store.AddRelation(ctx, "alice", "bob", "knows", props); err != nil {
		t.Fatalf("AddRelation: %v", err)
	}

	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(runner.writes))
	}
	if runner.writes[0].params["from"] != "alice" {
		t.Errorf("expected from 'alice', got %v", runner.writes[0].params["from"])
	}
	if runner.writes[0].params["to"] != "bob" {
		t.Errorf("expected to 'bob', got %v", runner.writes[0].params["to"])
	}
	if runner.writes[0].params["relType"] != "knows" {
		t.Errorf("expected relType 'knows', got %v", runner.writes[0].params["relType"])
	}
}

func TestAddRelation_Error(t *testing.T) {
	store, runner := newMockStore()
	runner.writeErr = fmt.Errorf("write error")

	err := store.AddRelation(context.Background(), "a", "b", "r", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddRelation_NilProps(t *testing.T) {
	store, runner := newMockStore()
	ctx := context.Background()

	if err := store.AddRelation(ctx, "a", "b", "rel", nil); err != nil {
		t.Fatalf("AddRelation: %v", err)
	}

	runner.mu.Lock()
	defer runner.mu.Unlock()
	props := runner.writes[0].params["props"].(map[string]any)
	if len(props) != 0 {
		t.Errorf("expected empty props, got %v", props)
	}
}

func TestQuery(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				nodeWrapper{
					elementID: "n1",
					props:     map[string]any{"id": "alice", "type": "person", "name": "Alice"},
				},
			}},
		},
	}
	store := newWithRunner(runner)

	results, err := store.Query(context.Background(), "MATCH (n) RETURN n")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(results[0].Entities))
	}
	if results[0].Entities[0].ID != "alice" {
		t.Errorf("expected id 'alice', got %q", results[0].Entities[0].ID)
	}
	if results[0].Entities[0].Properties["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", results[0].Entities[0].Properties["name"])
	}
}

func TestQuery_WithRelationships(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				nodeWrapper{
					elementID: "n1",
					props:     map[string]any{"id": "alice", "type": "person"},
				},
				relWrapper{
					elementID:      "r1",
					startElementID: "n1",
					endElementID:   "n2",
					props:          map[string]any{"type": "knows"},
				},
				nodeWrapper{
					elementID: "n2",
					props:     map[string]any{"id": "bob", "type": "person"},
				},
			}},
		},
	}
	store := newWithRunner(runner)

	results, err := store.Query(context.Background(), "MATCH (a)-[r]->(b) RETURN a,r,b")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results[0].Entities) != 2 {
		t.Errorf("expected 2 entities, got %d", len(results[0].Entities))
	}
	if len(results[0].Relations) != 1 {
		t.Errorf("expected 1 relation, got %d", len(results[0].Relations))
	}
	if results[0].Relations[0].Type != "knows" {
		t.Errorf("expected type 'knows', got %q", results[0].Relations[0].Type)
	}
}

func TestQuery_DuplicateDedup(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				nodeWrapper{elementID: "n1", props: map[string]any{"id": "alice", "type": "person"}},
			}},
			{values: []any{
				nodeWrapper{elementID: "n1", props: map[string]any{"id": "alice", "type": "person"}},
			}},
		},
	}
	store := newWithRunner(runner)

	results, err := store.Query(context.Background(), "MATCH (n) RETURN n")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results[0].Entities) != 1 {
		t.Errorf("expected 1 entity after dedup, got %d", len(results[0].Entities))
	}
}

func TestQuery_Error(t *testing.T) {
	runner := &mockRunner{readErr: fmt.Errorf("syntax error")}
	store := newWithRunner(runner)

	_, err := store.Query(context.Background(), "INVALID CYPHER")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestQuery_EmptyResult(t *testing.T) {
	store, _ := newMockStore()

	results, err := store.Query(context.Background(), "MATCH (n) RETURN n")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Entities) != 0 {
		t.Errorf("expected 0 entities, got %d", len(results[0].Entities))
	}
}

func TestNeighbors(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				nodeWrapper{
					elementID: "n2",
					props:     map[string]any{"id": "bob", "type": "person"},
				},
				[]any{
					relWrapper{
						elementID:      "r1",
						startElementID: "n1",
						endElementID:   "n2",
						props:          map[string]any{"type": "knows"},
					},
				},
			}},
		},
	}
	store := newWithRunner(runner)

	entities, relations, err := store.Neighbors(context.Background(), "alice", 1)
	if err != nil {
		t.Fatalf("Neighbors: %v", err)
	}

	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if entities[0].ID != "bob" {
		t.Errorf("expected id 'bob', got %q", entities[0].ID)
	}
	if len(relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(relations))
	}
}

func TestNeighbors_DefaultDepth(t *testing.T) {
	store, _ := newMockStore()

	// Depth 0 should default to 1; should not error.
	_, _, err := store.Neighbors(context.Background(), "alice", 0)
	if err != nil {
		t.Fatalf("Neighbors: %v", err)
	}
}

func TestNeighbors_Error(t *testing.T) {
	runner := &mockRunner{readErr: fmt.Errorf("read error")}
	store := newWithRunner(runner)

	_, _, err := store.Neighbors(context.Background(), "alice", 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClose(t *testing.T) {
	store, runner := newMockStore()
	if err := store.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !runner.closed {
		t.Error("expected runner to be closed")
	}
}

func TestClose_Error(t *testing.T) {
	runner := &mockRunner{closeErr: fmt.Errorf("close failed")}
	store := newWithRunner(runner)

	err := store.Close(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitizeProps(t *testing.T) {
	t.Run("nil props", func(t *testing.T) {
		result := sanitizeProps(nil)
		if result == nil || len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("supported types", func(t *testing.T) {
		props := map[string]any{
			"s": "hello",
			"i": 42,
			"f": 3.14,
			"b": true,
		}
		result := sanitizeProps(props)
		if len(result) != 4 {
			t.Errorf("expected 4 props, got %d", len(result))
		}
	})

	t.Run("unsupported types converted to string", func(t *testing.T) {
		props := map[string]any{
			"slice": []int{1, 2, 3},
		}
		result := sanitizeProps(props)
		if _, ok := result["slice"].(string); !ok {
			t.Errorf("expected string, got %T", result["slice"])
		}
	})
}

func TestNodeToEntity(t *testing.T) {
	node := nodeWrapper{
		props: map[string]any{
			"id":   "test",
			"type": "person",
			"name": "Test",
			"age":  42,
		},
	}
	entity := nodeToEntity(node)
	if entity.ID != "test" {
		t.Errorf("expected id 'test', got %q", entity.ID)
	}
	if entity.Type != "person" {
		t.Errorf("expected type 'person', got %q", entity.Type)
	}
	// id and type should not be in properties.
	if _, ok := entity.Properties["id"]; ok {
		t.Error("id should not be in properties")
	}
	if _, ok := entity.Properties["type"]; ok {
		t.Error("type should not be in properties")
	}
	if entity.Properties["name"] != "Test" {
		t.Errorf("expected name 'Test', got %v", entity.Properties["name"])
	}
}

func TestNodeID_WithProps(t *testing.T) {
	node := nodeWrapper{
		elementID: "elem-1",
		props:     map[string]any{"id": "my-id"},
	}
	if id := nodeID(node); id != "my-id" {
		t.Errorf("expected 'my-id', got %q", id)
	}
}

func TestNodeID_Fallback(t *testing.T) {
	node := nodeWrapper{
		elementID: "elem-1",
		props:     map[string]any{},
	}
	if id := nodeID(node); id != "elem-1" {
		t.Errorf("expected 'elem-1', got %q", id)
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ memory.GraphStore = (*GraphStore)(nil)
}

func TestGetString_NonStringValue(t *testing.T) {
	props := map[string]any{"id": 42}
	result := getString(props, "id")
	assert.Equal(t, "", result)
}

func TestGetString_MissingKey(t *testing.T) {
	props := map[string]any{"name": "Alice"}
	result := getString(props, "id")
	assert.Equal(t, "", result)
}

func TestRelToRelation_WithExtraProps(t *testing.T) {
	rel := relWrapper{
		elementID:      "r1",
		startElementID: "n1",
		endElementID:   "n2",
		props:          map[string]any{"type": "knows", "since": "2024", "weight": 0.9},
	}
	r := relToRelation(rel)
	assert.Equal(t, "knows", r.Type)
	assert.Equal(t, "n1", r.From)
	assert.Equal(t, "n2", r.To)
	assert.Equal(t, "2024", r.Properties["since"])
	assert.Equal(t, 0.9, r.Properties["weight"])
	_, hasType := r.Properties["type"]
	assert.False(t, hasType)
}

func TestRelToRelation_EmptyProps(t *testing.T) {
	rel := relWrapper{
		elementID:      "r1",
		startElementID: "n1",
		endElementID:   "n2",
		props:          map[string]any{},
	}
	r := relToRelation(rel)
	assert.Equal(t, "", r.Type)
	assert.Empty(t, r.Properties)
}

func TestExtractFromList_WithNodes(t *testing.T) {
	list := []any{
		nodeWrapper{
			elementID: "n1",
			props:     map[string]any{"id": "alice", "type": "person", "name": "Alice"},
		},
		nodeWrapper{
			elementID: "n2",
			props:     map[string]any{"id": "bob", "type": "person", "name": "Bob"},
		},
	}
	var entities []memory.Entity
	var relations []memory.Relation
	entitySeen := make(map[string]bool)
	relSeen := make(map[string]bool)

	extractFromList(list, &entities, &relations, entitySeen, relSeen)

	assert.Len(t, entities, 2)
	assert.Len(t, relations, 0)
	assert.Equal(t, "alice", entities[0].ID)
	assert.Equal(t, "bob", entities[1].ID)
}

func TestExtractFromList_Dedup(t *testing.T) {
	list := []any{
		nodeWrapper{elementID: "n1", props: map[string]any{"id": "alice", "type": "person"}},
		nodeWrapper{elementID: "n1", props: map[string]any{"id": "alice", "type": "person"}},
		relWrapper{elementID: "r1", startElementID: "n1", endElementID: "n2", props: map[string]any{"type": "knows"}},
		relWrapper{elementID: "r1", startElementID: "n1", endElementID: "n2", props: map[string]any{"type": "knows"}},
	}
	var entities []memory.Entity
	var relations []memory.Relation
	entitySeen := make(map[string]bool)
	relSeen := make(map[string]bool)

	extractFromList(list, &entities, &relations, entitySeen, relSeen)

	assert.Len(t, entities, 1)
	assert.Len(t, relations, 1)
}

func TestExtractFromList_MixedTypes(t *testing.T) {
	list := []any{
		nodeWrapper{elementID: "n1", props: map[string]any{"id": "alice", "type": "person"}},
		relWrapper{elementID: "r1", startElementID: "n1", endElementID: "n2", props: map[string]any{"type": "knows"}},
		"some other value",
		42,
	}
	var entities []memory.Entity
	var relations []memory.Relation
	entitySeen := make(map[string]bool)
	relSeen := make(map[string]bool)

	extractFromList(list, &entities, &relations, entitySeen, relSeen)

	assert.Len(t, entities, 1)
	assert.Len(t, relations, 1)
}

func TestQuery_WithListValues(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				[]any{
					nodeWrapper{elementID: "n1", props: map[string]any{"id": "alice", "type": "person"}},
					relWrapper{elementID: "r1", startElementID: "n1", endElementID: "n2", props: map[string]any{"type": "knows"}},
					nodeWrapper{elementID: "n2", props: map[string]any{"id": "bob", "type": "person"}},
				},
			}},
		},
	}
	store := newWithRunner(runner)

	results, err := store.Query(context.Background(), "MATCH p=(a)-[r]->(b) RETURN p")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Len(t, results[0].Entities, 2)
	assert.Len(t, results[0].Relations, 1)
}

func TestNeighbors_WithNodeInList(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				nodeWrapper{
					elementID: "n2",
					props:     map[string]any{"id": "bob", "type": "person"},
				},
				[]any{
					relWrapper{
						elementID:      "r1",
						startElementID: "n1",
						endElementID:   "n2",
						props:          map[string]any{"type": "knows"},
					},
					nodeWrapper{
						elementID: "n3",
						props:     map[string]any{"id": "carol", "type": "person"},
					},
				},
			}},
		},
	}
	store := newWithRunner(runner)

	entities, relations, err := store.Neighbors(context.Background(), "alice", 2)
	assert.NoError(t, err)
	assert.Len(t, entities, 2)
	assert.Len(t, relations, 1)
}

func TestNeighbors_NegativeDepth(t *testing.T) {
	store, _ := newMockStore()
	_, _, err := store.Neighbors(context.Background(), "alice", -5)
	assert.NoError(t, err)
}

func TestNeighbors_DuplicateDedup(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				nodeWrapper{elementID: "n2", props: map[string]any{"id": "bob", "type": "person"}},
				[]any{
					relWrapper{elementID: "r1", startElementID: "n1", endElementID: "n2", props: map[string]any{"type": "knows"}},
				},
			}},
			{values: []any{
				nodeWrapper{elementID: "n2", props: map[string]any{"id": "bob", "type": "person"}},
				[]any{
					relWrapper{elementID: "r1", startElementID: "n1", endElementID: "n2", props: map[string]any{"type": "knows"}},
				},
			}},
		},
	}
	store := newWithRunner(runner)

	entities, relations, err := store.Neighbors(context.Background(), "alice", 1)
	assert.NoError(t, err)
	assert.Len(t, entities, 1, "duplicate entities should be deduped")
	assert.Len(t, relations, 1, "duplicate relations should be deduped")
}

func TestQuery_DuplicateRelDedup(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				relWrapper{elementID: "r1", startElementID: "n1", endElementID: "n2", props: map[string]any{"type": "knows"}},
			}},
			{values: []any{
				relWrapper{elementID: "r1", startElementID: "n1", endElementID: "n2", props: map[string]any{"type": "knows"}},
			}},
		},
	}
	store := newWithRunner(runner)

	results, err := store.Query(context.Background(), "MATCH ()-[r]->() RETURN r")
	assert.NoError(t, err)
	assert.Len(t, results[0].Relations, 1, "duplicate relations should be deduped")
}

func TestNodeToEntity_EmptyProps(t *testing.T) {
	node := nodeWrapper{
		elementID: "n1",
		props:     map[string]any{},
	}
	entity := nodeToEntity(node)
	assert.Equal(t, "", entity.ID)
	assert.Equal(t, "", entity.Type)
	assert.Empty(t, entity.Properties)
}

func TestSanitizeProps_Int64(t *testing.T) {
	props := map[string]any{
		"count": int64(100),
	}
	result := sanitizeProps(props)
	assert.Equal(t, int64(100), result["count"])
}

func TestNodeID_NilProps(t *testing.T) {
	node := nodeWrapper{
		elementID: "elem-1",
		props:     nil,
	}
	assert.Equal(t, "elem-1", nodeID(node))
}

func TestGetString_NilProps(t *testing.T) {
	result := getString(nil, "key")
	assert.Equal(t, "", result)
}

func TestNew_CreatesStore(t *testing.T) {
	// NewDriverWithContext doesn't connect eagerly, so this tests the constructor.
	store, err := New(Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "test",
		Database: "neo4j",
	})
	assert.NoError(t, err)
	assert.NotNil(t, store)
	_ = store.Close(context.Background())
}

func TestNew_InvalidURI(t *testing.T) {
	_, err := New(Config{
		URI:      "://invalid",
		Username: "neo4j",
		Password: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "neo4j: create driver:")
}

func TestNeighbors_DirectRelWrapper(t *testing.T) {
	runner := &mockRunner{
		readData: []record{
			{values: []any{
				nodeWrapper{
					elementID: "n2",
					props:     map[string]any{"id": "bob", "type": "person"},
				},
				relWrapper{
					elementID:      "r1",
					startElementID: "n1",
					endElementID:   "n2",
					props:          map[string]any{"type": "knows"},
				},
			}},
		},
	}
	store := newWithRunner(runner)

	entities, relations, err := store.Neighbors(context.Background(), "alice", 1)
	assert.NoError(t, err)
	assert.Len(t, entities, 1)
	assert.Len(t, relations, 1)
	assert.Equal(t, "bob", entities[0].ID)
	assert.Equal(t, "knows", relations[0].Type)
}
