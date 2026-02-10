// Package memgraph provides a Memgraph-backed [memory.GraphStore] implementation for the
// Beluga AI memory system. Memgraph uses the Bolt protocol (same as Neo4j),
// so this implementation uses the Neo4j Go driver with Cypher queries.
//
// # Usage
//
//	import "github.com/lookatitude/beluga-ai/memory/stores/memgraph"
//
//	store, err := memgraph.New(memgraph.Config{
//	    URI:      "bolt://localhost:7687",
//	    Username: "",
//	    Password: "",
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
// # Memgraph vs Neo4j
//
// Memgraph is an in-memory graph database optimized for real-time analytics
// and streaming workloads. It is Bolt-compatible with Neo4j and uses the
// same Cypher query language. This store uses the same graph model as the
// neo4j store: entities as "Entity" nodes and relations as "RELATION" edges.
//
// # Testability
//
// The store uses an internal sessionRunner interface that abstracts session
// operations, enabling mock-based testing without a live database.
//
// This implementation requires github.com/neo4j/neo4j-go-driver/v5.
package memgraph
