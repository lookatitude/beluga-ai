package memory

import (
	"context"
	"time"
)

// Entity represents a node in the knowledge graph.
type Entity struct {
	// ID is the unique identifier for this entity.
	ID string
	// Type classifies the entity (e.g., "person", "organization", "concept").
	Type string
	// Properties holds arbitrary key-value attributes of the entity.
	Properties map[string]any
	// CreatedAt records when this entity was first observed. Zero value means unset.
	CreatedAt time.Time
	// Summary is a human-readable summary of this entity.
	Summary string
}

// Relation represents a directed edge between two entities in the knowledge graph.
// Relations support bi-temporal modeling: system time (CreatedAt/ExpiredAt) tracks
// when the fact was ingested and invalidated in the system, while valid time
// (ValidAt/InvalidAt) tracks when the fact was true in the real world.
type Relation struct {
	// From is the source entity ID.
	From string
	// To is the target entity ID.
	To string
	// Type classifies the relationship (e.g., "works_at", "knows", "part_of").
	Type string
	// Properties holds arbitrary key-value attributes of the relation.
	Properties map[string]any
	// CreatedAt is the system time when this relation was ingested.
	CreatedAt time.Time
	// ExpiredAt is the system time when this relation was invalidated.
	// Nil means the relation is still active in the system.
	ExpiredAt *time.Time
	// ValidAt is the valid time when this fact became true in the real world.
	ValidAt time.Time
	// InvalidAt is the valid time when this fact ceased to be true in the real world.
	// Nil means the fact is still considered true.
	InvalidAt *time.Time
	// Episodes contains the UUIDs of episodes that support this relation.
	Episodes []string
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

// QueryOption configures temporal query behavior.
type QueryOption func(*QueryOptions)

// QueryOptions holds the resolved configuration for a temporal query.
type QueryOptions struct {
	// Limit is the maximum number of results to return.
	Limit int
}

// WithQueryLimit sets the maximum number of results for a temporal query.
func WithQueryLimit(limit int) QueryOption {
	return func(o *QueryOptions) {
		if limit > 0 {
			o.Limit = limit
		}
	}
}

// ApplyQueryOptions resolves the given QueryOption functions into a QueryOptions value.
func ApplyQueryOptions(opts []QueryOption) QueryOptions {
	o := QueryOptions{Limit: 1000} // default limit to prevent unbounded results
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// TemporalGraphStore extends GraphStore with bi-temporal query capabilities.
// It supports querying the knowledge graph as it was at a specific point in time,
// invalidating relations, and retrieving the full history of relations between entities.
type TemporalGraphStore interface {
	GraphStore

	// QueryAsOf returns entities and relations valid at a specific point in time.
	// Only relations where ValidAt <= validTime and (InvalidAt is nil or InvalidAt > validTime)
	// are included. The opts parameter can be used to limit results.
	QueryAsOf(ctx context.Context, query string, validTime time.Time, opts ...QueryOption) ([]Entity, []Relation, error)

	// InvalidateRelation marks a relation as no longer valid at the given time.
	// It sets the InvalidAt field to invalidAt and ExpiredAt to the current system time.
	InvalidateRelation(ctx context.Context, relationID string, invalidAt time.Time) error

	// History returns all versions of relations between two entities, including
	// invalidated ones, ordered by ValidAt ascending.
	History(ctx context.Context, fromID, toID string) ([]Relation, error)
}
