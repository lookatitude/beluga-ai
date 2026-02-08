// Package neo4j provides a Neo4j-backed GraphStore implementation for the
// Beluga AI memory system. It uses Cypher queries for graph operations and
// supports the full GraphStore interface including entity management,
// relationship creation, querying, and neighbor traversal.
//
// Usage:
//
//	import "github.com/lookatitude/beluga-ai/memory/stores/neo4j"
//
//	store, err := neo4j.New(neo4j.Config{
//	    URI:      "neo4j://localhost:7687",
//	    Username: "neo4j",
//	    Password: "password",
//	})
//	defer store.Close(ctx)
package neo4j

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/memory"
	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Config holds configuration for the Neo4j GraphStore.
type Config struct {
	// URI is the Neo4j connection URI (e.g., "neo4j://localhost:7687").
	URI string
	// Username is the authentication username.
	Username string
	// Password is the authentication password.
	Password string
	// Database is the target database name. Empty means the default database.
	Database string
}

// sessionRunner abstracts Neo4j session operations for testability.
// Neo4j interfaces have unexported methods, so we use this thin wrapper.
type sessionRunner interface {
	executeWrite(ctx context.Context, cypher string, params map[string]any) error
	executeRead(ctx context.Context, cypher string, params map[string]any) ([]record, error)
	close(ctx context.Context) error
}

// record represents a single row from a query result.
type record struct {
	values []any
}

// neo4jRunner wraps a real Neo4j driver.
type neo4jRunner struct {
	drv      driver.DriverWithContext
	database string
}

func (r *neo4jRunner) executeWrite(ctx context.Context, cypher string, params map[string]any) error {
	session := r.drv.NewSession(ctx, driver.SessionConfig{DatabaseName: r.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx driver.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, cypher, params)
		return nil, err
	})
	return err
}

func (r *neo4jRunner) executeRead(ctx context.Context, cypher string, params map[string]any) ([]record, error) {
	session := r.drv.NewSession(ctx, driver.SessionConfig{
		DatabaseName: r.database,
		AccessMode:   driver.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx driver.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}

		var records []record
		for res.Next(ctx) {
			rec := res.Record()
			values := make([]any, len(rec.Values))
			for i, v := range rec.Values {
				switch typed := v.(type) {
				case driver.Node:
					values[i] = nodeWrapper{
						elementID: typed.ElementId,
						props:     typed.Props,
					}
				case driver.Relationship:
					values[i] = relWrapper{
						elementID:      typed.ElementId,
						startElementID: typed.StartElementId,
						endElementID:   typed.EndElementId,
						props:          typed.Props,
					}
				default:
					values[i] = v
				}
			}
			records = append(records, record{values: values})
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return records, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]record), nil
}

func (r *neo4jRunner) close(ctx context.Context) error {
	return r.drv.Close(ctx)
}

// nodeWrapper holds data extracted from a Neo4j node.
type nodeWrapper struct {
	elementID string
	props     map[string]any
}

// relWrapper holds data extracted from a Neo4j relationship.
type relWrapper struct {
	elementID      string
	startElementID string
	endElementID   string
	props          map[string]any
}

// GraphStore is a Neo4j-backed implementation of memory.GraphStore.
// It uses Cypher queries for all graph operations.
type GraphStore struct {
	runner sessionRunner
}

// New creates a new Neo4j GraphStore with the given configuration.
func New(cfg Config) (*GraphStore, error) {
	drv, err := driver.NewDriverWithContext(cfg.URI, driver.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j: create driver: %w", err)
	}
	return &GraphStore{
		runner: &neo4jRunner{drv: drv, database: cfg.Database},
	}, nil
}

// newWithRunner creates a GraphStore with a custom session runner (for testing).
func newWithRunner(r sessionRunner) *GraphStore {
	return &GraphStore{runner: r}
}

// Close closes the underlying Neo4j driver.
func (g *GraphStore) Close(ctx context.Context) error {
	return g.runner.close(ctx)
}

// AddEntity adds or updates an entity in the graph. Entities are stored as
// Neo4j nodes with a label matching the entity type and properties set from
// the entity's Properties map.
func (g *GraphStore) AddEntity(ctx context.Context, entity memory.Entity) error {
	cypher := "MERGE (e:Entity {id: $id}) SET e.type = $type, e += $props"
	params := map[string]any{
		"id":    entity.ID,
		"type":  entity.Type,
		"props": sanitizeProps(entity.Properties),
	}
	if err := g.runner.executeWrite(ctx, cypher, params); err != nil {
		return fmt.Errorf("neo4j/add_entity: %w", err)
	}
	return nil
}

// AddRelation creates a directed relationship between two entities.
func (g *GraphStore) AddRelation(ctx context.Context, from, to, relation string, props map[string]any) error {
	cypher := `MATCH (a:Entity {id: $from})
MATCH (b:Entity {id: $to})
CREATE (a)-[r:RELATION {type: $relType}]->(b)
SET r += $props`
	params := map[string]any{
		"from":    from,
		"to":      to,
		"relType": relation,
		"props":   sanitizeProps(props),
	}
	if err := g.runner.executeWrite(ctx, cypher, params); err != nil {
		return fmt.Errorf("neo4j/add_relation: %w", err)
	}
	return nil
}

// Query executes a Cypher query against the graph and returns matching entities
// and relations. The query string is executed as-is as a Cypher statement.
func (g *GraphStore) Query(ctx context.Context, query string) ([]memory.GraphResult, error) {
	records, err := g.runner.executeRead(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("neo4j/query: %w", err)
	}

	var entities []memory.Entity
	var relations []memory.Relation
	entitySeen := make(map[string]bool)
	relSeen := make(map[string]bool)

	for _, rec := range records {
		for _, val := range rec.values {
			switch v := val.(type) {
			case nodeWrapper:
				id := nodeID(v)
				if !entitySeen[id] {
					entitySeen[id] = true
					entities = append(entities, nodeToEntity(v))
				}
			case relWrapper:
				if !relSeen[v.elementID] {
					relSeen[v.elementID] = true
					relations = append(relations, relToRelation(v))
				}
			case []any:
				extractFromList(v, &entities, &relations, entitySeen, relSeen)
			}
		}
	}

	return []memory.GraphResult{{
		Entities:  entities,
		Relations: relations,
	}}, nil
}

// Neighbors returns all entities and relations within the given depth from
// the specified entity using variable-length path matching.
func (g *GraphStore) Neighbors(ctx context.Context, entityID string, depth int) ([]memory.Entity, []memory.Relation, error) {
	if depth <= 0 {
		depth = 1
	}

	cypher := fmt.Sprintf(
		"MATCH (start:Entity {id: $id})-[r*1..%d]-(neighbor:Entity) "+
			"RETURN neighbor, r", depth)
	records, err := g.runner.executeRead(ctx, cypher, map[string]any{"id": entityID})
	if err != nil {
		return nil, nil, fmt.Errorf("neo4j/neighbors: %w", err)
	}

	entitySeen := make(map[string]bool)
	relSeen := make(map[string]bool)
	var entities []memory.Entity
	var relations []memory.Relation

	for _, rec := range records {
		for _, val := range rec.values {
			switch v := val.(type) {
			case nodeWrapper:
				id := nodeID(v)
				if !entitySeen[id] {
					entitySeen[id] = true
					entities = append(entities, nodeToEntity(v))
				}
			case relWrapper:
				if !relSeen[v.elementID] {
					relSeen[v.elementID] = true
					relations = append(relations, relToRelation(v))
				}
			case []any:
				extractFromList(v, &entities, &relations, entitySeen, relSeen)
			}
		}
	}

	return entities, relations, nil
}

// extractFromList processes list values (e.g., variable-length paths).
func extractFromList(list []any, entities *[]memory.Entity, relations *[]memory.Relation, entitySeen, relSeen map[string]bool) {
	for _, item := range list {
		switch v := item.(type) {
		case nodeWrapper:
			id := nodeID(v)
			if !entitySeen[id] {
				entitySeen[id] = true
				*entities = append(*entities, nodeToEntity(v))
			}
		case relWrapper:
			if !relSeen[v.elementID] {
				relSeen[v.elementID] = true
				*relations = append(*relations, relToRelation(v))
			}
		}
	}
}

// nodeToEntity converts a node wrapper to a memory.Entity.
func nodeToEntity(node nodeWrapper) memory.Entity {
	props := make(map[string]any)
	for k, v := range node.props {
		if k != "id" && k != "type" {
			props[k] = v
		}
	}
	return memory.Entity{
		ID:         getString(node.props, "id"),
		Type:       getString(node.props, "type"),
		Properties: props,
	}
}

// relToRelation converts a relationship wrapper to a memory.Relation.
func relToRelation(rel relWrapper) memory.Relation {
	props := make(map[string]any)
	for k, v := range rel.props {
		if k != "type" {
			props[k] = v
		}
	}
	return memory.Relation{
		From:       rel.startElementID,
		To:         rel.endElementID,
		Type:       getString(rel.props, "type"),
		Properties: props,
	}
}

// nodeID returns a string identifier for a node wrapper.
func nodeID(node nodeWrapper) string {
	if id, ok := node.props["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	return node.elementID
}

// getString safely extracts a string value from a properties map.
func getString(props map[string]any, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// sanitizeProps returns a copy of the properties map that only contains
// Neo4j-compatible values (strings, numbers, booleans). Nil maps are
// returned as empty maps to avoid Cypher SET errors.
func sanitizeProps(props map[string]any) map[string]any {
	if props == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(props))
	for k, v := range props {
		switch v.(type) {
		case string, int, int64, float64, bool:
			result[k] = v
		default:
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// Compile-time check.
var _ memory.GraphStore = (*GraphStore)(nil)
