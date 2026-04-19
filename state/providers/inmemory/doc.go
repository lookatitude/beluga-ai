// Package inmemory provides an in-memory implementation of the [state.Store]
// interface. It is intended for development and testing. Data is not persisted
// across process restarts.
//
// The store registers itself under the name "inmemory" via init(), so it can
// be created through the state registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/state/providers/inmemory"
//
//	store, err := state.New("inmemory", state.Config{})
//
// Or created directly:
//
//	store := inmemory.New()
//	defer store.Close()
//
//	err := store.Set(ctx, "key", "value")
//	val, err := store.Get(ctx, "key")
//
// # Watch Support
//
// The in-memory store fully supports [state.Store.Watch], returning an
// iter.Seq2 stream. Internal watcher buffers have capacity 16 to reduce
// blocking; notifications that arrive while the buffer is full are dropped.
// Iterators end when the store is closed or the watch context is cancelled.
//
// # Thread Safety
//
// All operations are protected by a sync.RWMutex and are safe for concurrent
// use from multiple goroutines.
package inmemory
