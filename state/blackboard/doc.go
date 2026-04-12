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
//
// # Usage
//
//	store := inmemory.New()
//	bb := blackboard.New(store,
//	    blackboard.WithReducers(
//	        state.WithReducer("counter", func(old, new any) any {
//	            o, _ := old.(int)
//	            n, _ := new.(int)
//	            return o + n
//	        }),
//	    ),
//	)
//	defer bb.Close()
//
//	bb.Set(ctx, "agent-1", "counter", 5)
//	bb.Set(ctx, "agent-2", "counter", 3) // merged to 8
package blackboard
