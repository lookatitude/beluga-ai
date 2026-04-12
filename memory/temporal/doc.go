// Package temporal provides bi-temporal knowledge graph memory for agents.
//
// It implements the Graphiti-inspired 2-condition conflict resolution algorithm
// and supports querying the knowledge graph as it existed at any point in time.
// The package lives in Layer 3 (Capability) and depends on the memory and core
// packages.
//
// Key components include ConflictResolver for temporal conflict detection,
// InMemoryStore for testing, and hooks for lifecycle interception.
package temporal
