package vectorstore

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// Middleware wraps a VectorStore to add cross-cutting behaviour.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(VectorStore) VectorStore

// ApplyMiddleware wraps store with the given middlewares in reverse order so
// that the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(store VectorStore, mws ...Middleware) VectorStore {
	for i := len(mws) - 1; i >= 0; i-- {
		store = mws[i](store)
	}
	return store
}

// WithHooks returns middleware that invokes the given Hooks around Add and
// Search calls.
func WithHooks(hooks Hooks) Middleware {
	return func(next VectorStore) VectorStore {
		return &hookedStore{next: next, hooks: hooks}
	}
}

type hookedStore struct {
	next  VectorStore
	hooks Hooks
}

func (s *hookedStore) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if s.hooks.BeforeAdd != nil {
		if err := s.hooks.BeforeAdd(ctx, docs); err != nil {
			return err
		}
	}
	return s.next.Add(ctx, docs, embeddings)
}

func (s *hookedStore) Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error) {
	results, err := s.next.Search(ctx, query, k, opts...)

	if s.hooks.AfterSearch != nil {
		s.hooks.AfterSearch(ctx, results, err)
	}

	return results, err
}

func (s *hookedStore) Delete(ctx context.Context, ids []string) error {
	return s.next.Delete(ctx, ids)
}
