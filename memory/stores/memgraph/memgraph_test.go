package memgraph

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/memory"
)

// mockRunner implements sessionRunner for testing.
type mockRunner struct {
	mu       sync.Mutex
	writes   []writeCall
	readData []record
	writeErr error
	readErr  error
	closeErr error
	closed   bool
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
