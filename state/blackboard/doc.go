// Package blackboard provides an enhanced shared state layer for multi-agent
// coordination.
//
// It wraps a state.VersionedStore with ownership enforcement, per-key reducers,
// and iter.Seq2 watch streams. The package lives in Layer 2 (Cross-cutting) of
// the Beluga architecture and is used by the blackboard orchestration pattern
// in Layer 5.
//
// This package follows the Beluga 4-ring extension contract --
// see docs/architecture/03-extensibility-patterns.md.
package blackboard
