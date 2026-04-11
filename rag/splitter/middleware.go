package splitter

// Middleware wraps a TextSplitter to add cross-cutting behaviour.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(TextSplitter) TextSplitter

// ApplyMiddleware wraps splitter with the given middlewares in reverse order
// so that the first middleware in the list is the outermost (first to
// execute).
func ApplyMiddleware(splitter TextSplitter, mws ...Middleware) TextSplitter {
	for i := len(mws) - 1; i >= 0; i-- {
		splitter = mws[i](splitter)
	}
	return splitter
}
