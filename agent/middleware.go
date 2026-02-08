package agent

// Middleware wraps an Agent to add cross-cutting concerns such as
// tracing, retry, logging, or metrics.
type Middleware func(Agent) Agent

// ApplyMiddleware applies a chain of middleware to an agent. Middleware is
// applied in reverse order so that the first middleware in the slice is the
// outermost wrapper.
func ApplyMiddleware(a Agent, mws ...Middleware) Agent {
	for i := len(mws) - 1; i >= 0; i-- {
		a = mws[i](a)
	}
	return a
}
