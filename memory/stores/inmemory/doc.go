// Package inmemory provides in-memory implementations of the memory store
// interfaces. These implementations are suitable for development, testing,
// and short-lived agent sessions. Data is not persisted across process
// restarts.
//
// # MessageStore
//
// [MessageStore] implements [memory.MessageStore] with a thread-safe slice.
// Messages are searched via case-insensitive substring matching on text
// content parts:
//
//	store := inmemory.NewMessageStore()
//	err := store.Append(ctx, msg)
//	results, err := store.Search(ctx, "hello", 10)
//	all, err := store.All(ctx)
//
// # GraphStore
//
// [GraphStore] implements [memory.GraphStore] with in-memory maps for entities
// and relations. It supports basic type-based queries ("type:person") and
// breadth-first neighbor traversal:
//
//	graph := inmemory.NewGraphStore()
//	err := graph.AddEntity(ctx, memory.Entity{ID: "alice", Type: "person"})
//	err = graph.AddRelation(ctx, "alice", "bob", "knows", nil)
//	entities, relations, err := graph.Neighbors(ctx, "alice", 2)
//
// For full Cypher query support, use the neo4j or memgraph store providers.
//
// Both stores are safe for concurrent use.
package inmemory
