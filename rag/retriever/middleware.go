package retriever

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// Middleware wraps a Retriever to add cross-cutting behaviour such as
// caching, tracing, or logging. Middleware is applied in reverse order
// so the first middleware in the list is the outermost wrapper.
type Middleware func(Retriever) Retriever

// ApplyMiddleware wraps base with the given middleware. The first middleware
// in the list is the outermost wrapper (executed first on entry, last on exit).
func ApplyMiddleware(base Retriever, mws ...Middleware) Retriever {
	for i := len(mws) - 1; i >= 0; i-- {
		base = mws[i](base)
	}
	return base
}

// WithHooks returns a Middleware that invokes the given hooks around each
// Retrieve call.
func WithHooks(hooks Hooks) Middleware {
	return func(next Retriever) Retriever {
		return &hookedRetriever{next: next, hooks: hooks}
	}
}

// hookedRetriever wraps a Retriever with lifecycle hooks.
type hookedRetriever struct {
	next  Retriever
	hooks Hooks
}

func (h *hookedRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	if h.hooks.BeforeRetrieve != nil {
		if err := h.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	docs, err := h.next.Retrieve(ctx, query, opts...)

	if h.hooks.AfterRetrieve != nil {
		h.hooks.AfterRetrieve(ctx, docs, err)
	}

	return docs, err
}
