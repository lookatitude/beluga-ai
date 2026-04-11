package routing

// Middleware wraps a CostRouter to add cross-cutting behaviour such as
// tracing, logging, or metrics. Middlewares are composed via ApplyMiddleware
// and applied outside-in (the first middleware in the list is the outermost
// wrapper).
type Middleware func(CostRouter) CostRouter

// ApplyMiddleware wraps r with the given middlewares in reverse order so that
// the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(r CostRouter, mws ...Middleware) CostRouter {
	for i := len(mws) - 1; i >= 0; i-- {
		r = mws[i](r)
	}
	return r
}
