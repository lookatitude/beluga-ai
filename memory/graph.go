package memory

import "context"

// Entity represents a node in the knowledge graph.
type Entity struct {
	// ID is the unique identifier for this entity.
	ID string
	// Type classifies the entity (e.g., "person", "organization", "concept").
	Type string
	// Properties holds arbitrary key-value attributes of the entity.
	Properties map[string]any
}

// Relation represents a directed edge between two entities in the knowledge graph.
type Relation struct {
	// From is the source entity ID.
	From string
	// To is the target entity ID.
	To string
	// Type classifies the relationship (e.g., "works_at", "knows", "part_of").
	Type string
	// Properties holds arbitrary key-value attributes of the relation.
	Properties map[string]any
}

// GraphResult holds the result of a graph query, containing matched entities
// and relations.
type GraphResult struct {
	// Entities are the nodes matched by the query.
	Entities []Entity
	// Relations are the edges matched by the query.
	Relations []Relation
}

// GraphStore is the interface for graph-based memory storage. Implementations
// provide entity-relationship storage for structured knowledge representation.
// All methods must be safe for concurrent use.
type GraphStore interface {
	// AddEntity adds or updates an entity in the graph.
	AddEntity(ctx context.Context, entity Entity) error

	// AddRelation creates a directed relationship between two entities.
	AddRelation(ctx context.Context, from, to, relation string, props map[string]any) error

	// Query executes a query string (e.g., Cypher) against the graph and
	// returns matching entities and relations.
	Query(ctx context.Context, query string) ([]GraphResult, error)

	// Neighbors returns all entities and relations within the given depth
	// from the specified entity.
	Neighbors(ctx context.Context, entityID string, depth int) ([]Entity, []Relation, error)
}
