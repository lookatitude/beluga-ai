package embedding

import "context"

// Middleware wraps an Embedder to add cross-cutting behaviour.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(Embedder) Embedder

// ApplyMiddleware wraps emb with the given middlewares in reverse order so
// that the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(emb Embedder, mws ...Middleware) Embedder {
	for i := len(mws) - 1; i >= 0; i-- {
		emb = mws[i](emb)
	}
	return emb
}

// WithHooks returns middleware that invokes the given Hooks around Embed
// and EmbedSingle calls.
func WithHooks(hooks Hooks) Middleware {
	return func(next Embedder) Embedder {
		return &hookedEmbedder{next: next, hooks: hooks}
	}
}

type hookedEmbedder struct {
	next  Embedder
	hooks Hooks
}

func (e *hookedEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if e.hooks.BeforeEmbed != nil {
		if err := e.hooks.BeforeEmbed(ctx, texts); err != nil {
			return nil, err
		}
	}

	embeddings, err := e.next.Embed(ctx, texts)

	if e.hooks.AfterEmbed != nil {
		e.hooks.AfterEmbed(ctx, embeddings, err)
	}

	return embeddings, err
}

func (e *hookedEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if e.hooks.BeforeEmbed != nil {
		if err := e.hooks.BeforeEmbed(ctx, []string{text}); err != nil {
			return nil, err
		}
	}

	vec, err := e.next.EmbedSingle(ctx, text)

	if e.hooks.AfterEmbed != nil {
		var embeddings [][]float32
		if vec != nil {
			embeddings = [][]float32{vec}
		}
		e.hooks.AfterEmbed(ctx, embeddings, err)
	}

	return vec, err
}

func (e *hookedEmbedder) Dimensions() int {
	return e.next.Dimensions()
}
