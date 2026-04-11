package prompt

// Middleware wraps a PromptManager to add cross-cutting behaviour.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(PromptManager) PromptManager

// ApplyMiddleware wraps manager with the given middlewares in reverse order
// so that the first middleware in the list is the outermost (first to
// execute).
func ApplyMiddleware(manager PromptManager, mws ...Middleware) PromptManager {
	for i := len(mws) - 1; i >= 0; i-- {
		manager = mws[i](manager)
	}
	return manager
}
