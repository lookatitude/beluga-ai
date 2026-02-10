// Package neo4j provides a Neo4j-backed [memory.GraphStore] implementation for the
// Beluga AI memory system. It uses Cypher queries for graph operations and
// supports the full [memory.GraphStore] interface including entity management,
// relationship creation, querying, and neighbor traversal.
//
// # Usage
//
//	import "github.com/lookatitude/beluga-ai/memory/stores/neo4j"
//
//	store, err := neo4j.New(neo4j.Config{
//	    URI:      "neo4j://localhost:7687",
//	    Username: "neo4j",
//	    Password: "password",
//	    Database: "",  // empty for default database
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close(ctx)
//
//	err = store.AddEntity(ctx, memory.Entity{
//	    ID:   "alice",
//	    Type: "person",
//	    Properties: map[string]any{"age": 30},
//	})
//	err = store.AddRelation(ctx, "alice", "bob", "knows", nil)
//	results, err := store.Query(ctx, "MATCH (n:Entity) RETURN n")
//	entities, relations, err := store.Neighbors(ctx, "alice", 2)
//
// # Graph Model
//
// Entities are stored as Neo4j nodes with the label "Entity" and properties
// set from the entity's Properties map. The entity ID and type are stored
// as node properties. Relations are stored as "RELATION" edges with a "type"
// property.
//
// # Testability
//
// The store uses an internal sessionRunner interface that abstracts Neo4j
// session operations, enabling mock-based testing without a live database.
//
// This implementation requires github.com/neo4j/neo4j-go-driver/v5.
package neo4j
