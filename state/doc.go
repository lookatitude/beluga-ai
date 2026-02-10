// Package state provides shared agent state with key-value storage,
// change notifications via Watch, and scoping by agent, session, or global
// visibility.
//
// # Store Interface
//
// The primary interface is [Store]:
//
//	type Store interface {
//	    Get(ctx context.Context, key string) (any, error)
//	    Set(ctx context.Context, key string, value any) error
//	    Delete(ctx context.Context, key string) error
//	    Watch(ctx context.Context, key string) (<-chan StateChange, error)
//	    Close() error
//	}
//
// # Registry Pattern
//
// The package follows Beluga's standard registry pattern. Providers register
// via init() and are created with [New]:
//
//	store, err := state.New("inmemory", state.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
//
// Use [List] to discover all registered providers.
//
// # Scoped Keys
//
// State keys can be scoped to control visibility. Use [ScopedKey] with one
// of the predefined scopes:
//
//   - [ScopeAgent] — visible only to a single agent instance
//   - [ScopeSession] — visible within the current session
//   - [ScopeGlobal] — visible across all agents and sessions
//
// Example:
//
//	key := state.ScopedKey(state.ScopeAgent, "counter")
//	err := store.Set(ctx, key, 42)
//	val, err := store.Get(ctx, key)
//
// # Watch for Changes
//
// The [Store.Watch] method returns a channel that receives [StateChange]
// notifications whenever a key is modified or deleted:
//
//	ch, err := store.Watch(ctx, "mykey")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for change := range ch {
//	    fmt.Printf("op=%s old=%v new=%v\n", change.Op, change.OldValue, change.Value)
//	}
//
// Each [StateChange] includes the key, old value, new value, and the operation
// type ([OpSet] or [OpDelete]).
//
// # Middleware and Hooks
//
// Store operations can be wrapped with [Middleware] for cross-cutting concerns
// and observed via [Hooks] callbacks:
//
//	hooked := state.ApplyMiddleware(store, state.WithHooks(state.Hooks{
//	    BeforeSet: func(ctx context.Context, key string, value any) error {
//	        log.Printf("setting %s = %v", key, value)
//	        return nil
//	    },
//	}))
//
// Multiple hooks are merged with [ComposeHooks]. For Before* hooks, OnDelete,
// OnWatch, and OnError, the first error returned short-circuits the chain.
//
// # Providers
//
// Store backends are in sub-packages under state/providers/:
//
//   - state/providers/inmemory — in-memory (development/testing)
package state
