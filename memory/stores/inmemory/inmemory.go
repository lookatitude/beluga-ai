package inmemory

import (
	"context"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// MessageStore is an in-memory implementation of memory.MessageStore.
// Messages are stored in a slice and searched via case-insensitive substring
// matching. Safe for concurrent use.
type MessageStore struct {
	mu   sync.RWMutex
	msgs []schema.Message
}

// NewMessageStore creates an empty in-memory MessageStore.
func NewMessageStore() *MessageStore {
	return &MessageStore{}
}

// Append adds a message to the store.
func (s *MessageStore) Append(_ context.Context, msg schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
	return nil
}

// Search finds messages whose text content contains the query as a
// case-insensitive substring, returning at most k results.
func (s *MessageStore) Search(_ context.Context, query string, k int) ([]schema.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	var results []schema.Message
	for _, msg := range s.msgs {
		if containsText(msg, q) {
			results = append(results, msg)
			if len(results) >= k {
				break
			}
		}
	}
	return results, nil
}

// All returns all stored messages in chronological order.
func (s *MessageStore) All(_ context.Context) ([]schema.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]schema.Message, len(s.msgs))
	copy(cp, s.msgs)
	return cp, nil
}

// Clear removes all messages.
func (s *MessageStore) Clear(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = nil
	return nil
}

// containsText checks if a message's text content contains the query.
func containsText(msg schema.Message, lowerQuery string) bool {
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			if strings.Contains(strings.ToLower(tp.Text), lowerQuery) {
				return true
			}
		}
	}
	return false
}

// Verify interface compliance.
var _ memory.MessageStore = (*MessageStore)(nil)

// GraphStore is an in-memory implementation of memory.GraphStore.
// Entities and relations are stored in maps. Safe for concurrent use.
type GraphStore struct {
	mu        sync.RWMutex
	entities  map[string]memory.Entity
	relations []memory.Relation
}

// NewGraphStore creates an empty in-memory GraphStore.
func NewGraphStore() *GraphStore {
	return &GraphStore{
		entities: make(map[string]memory.Entity),
	}
}

// AddEntity adds or updates an entity in the graph.
func (g *GraphStore) AddEntity(_ context.Context, entity memory.Entity) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entities[entity.ID] = entity
	return nil
}

// AddRelation creates a directed relationship between two entities.
func (g *GraphStore) AddRelation(_ context.Context, from, to, relation string, props map[string]any) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.relations = append(g.relations, memory.Relation{
		From:       from,
		To:         to,
		Type:       relation,
		Properties: props,
	})
	return nil
}

// Query executes a simple query against the graph. The in-memory
// implementation supports basic entity type queries in the form
// "type:<TYPE>". For full Cypher support, use a Neo4j-backed store.
func (g *GraphStore) Query(_ context.Context, query string) ([]memory.GraphResult, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Simple type-based query: "type:person"
	if strings.HasPrefix(query, "type:") {
		entityType := strings.TrimPrefix(query, "type:")
		var entities []memory.Entity
		for _, e := range g.entities {
			if strings.EqualFold(e.Type, entityType) {
				entities = append(entities, e)
			}
		}
		return []memory.GraphResult{{Entities: entities}}, nil
	}

	// Default: return all entities and relations
	entities := make([]memory.Entity, 0, len(g.entities))
	for _, e := range g.entities {
		entities = append(entities, e)
	}
	return []memory.GraphResult{{
		Entities:  entities,
		Relations: g.relations,
	}}, nil
}

// Neighbors returns all entities and relations within the given depth from
// the specified entity using breadth-first traversal.
func (g *GraphStore) Neighbors(_ context.Context, entityID string, depth int) ([]memory.Entity, []memory.Relation, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if depth <= 0 {
		depth = 1
	}

	visited := map[string]bool{entityID: true}
	frontier := []string{entityID}
	var resultEntities []memory.Entity
	var resultRelations []memory.Relation

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var nextFrontier []string
		for _, nodeID := range frontier {
			neighbors, rels := g.findNeighbors(nodeID, visited)
			resultRelations = append(resultRelations, rels...)
			for _, nID := range neighbors {
				if e, ok := g.entities[nID]; ok {
					resultEntities = append(resultEntities, e)
				}
			}
			nextFrontier = append(nextFrontier, neighbors...)
		}
		frontier = nextFrontier
	}
	return resultEntities, resultRelations, nil
}

// findNeighbors returns unvisited neighbor IDs and their relations for a node.
// It marks discovered neighbors as visited.
func (g *GraphStore) findNeighbors(nodeID string, visited map[string]bool) ([]string, []memory.Relation) {
	var neighbors []string
	var rels []memory.Relation
	for _, rel := range g.relations {
		neighborID := ""
		if rel.From == nodeID {
			neighborID = rel.To
		} else if rel.To == nodeID {
			neighborID = rel.From
		} else {
			continue
		}
		rels = append(rels, rel)
		if !visited[neighborID] {
			visited[neighborID] = true
			neighbors = append(neighbors, neighborID)
		}
	}
	return neighbors, rels
}

// Verify interface compliance.
var _ memory.GraphStore = (*GraphStore)(nil)
