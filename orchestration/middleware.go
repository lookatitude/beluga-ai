package orchestration

import "github.com/lookatitude/beluga-ai/core"

// Middleware wraps a core.Runnable to add cross-cutting behaviour.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(core.Runnable) core.Runnable

// ApplyMiddleware wraps r with the given middlewares in reverse order so that
// the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(r core.Runnable, mws ...Middleware) core.Runnable {
	for i := len(mws) - 1; i >= 0; i-- {
		r = mws[i](r)
	}
	return r
}
