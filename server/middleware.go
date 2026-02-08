package server

// Middleware wraps a ServerAdapter to add cross-cutting behaviour.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(ServerAdapter) ServerAdapter

// ApplyMiddleware wraps adapter with the given middlewares in reverse order so
// that the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(s ServerAdapter, mws ...Middleware) ServerAdapter {
	for i := len(mws) - 1; i >= 0; i-- {
		s = mws[i](s)
	}
	return s
}
