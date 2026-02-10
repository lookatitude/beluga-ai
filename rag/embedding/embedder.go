package embedding

import "context"

// Embedder converts text into dense vector embeddings. Implementations must
// be safe for concurrent use.
type Embedder interface {
	// Embed produces embeddings for a batch of texts. The returned slice has
	// the same length as texts, with each element being a float32 vector of
	// Dimensions() length.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedSingle is a convenience method that embeds a single text and
	// returns its vector.
	EmbedSingle(ctx context.Context, text string) ([]float32, error)

	// Dimensions returns the dimensionality of the embedding vectors
	// produced by this embedder.
	Dimensions() int
}

// Hooks provides optional callback functions invoked around embedding
// operations. All fields are optional; nil hooks are skipped.
type Hooks struct {
	// BeforeEmbed is called before each Embed or EmbedSingle call with the
	// input texts. Returning an error aborts the call.
	BeforeEmbed func(ctx context.Context, texts []string) error

	// AfterEmbed is called after Embed completes with the embeddings and
	// any error.
	AfterEmbed func(ctx context.Context, embeddings [][]float32, err error)
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For BeforeEmbed, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		BeforeEmbed: func(ctx context.Context, texts []string) error {
			for _, h := range hooks {
				if h.BeforeEmbed != nil {
					if err := h.BeforeEmbed(ctx, texts); err != nil {
						return err
					}
				}
			}
			return nil
		},
		AfterEmbed: func(ctx context.Context, embeddings [][]float32, err error) {
			for _, h := range hooks {
				if h.AfterEmbed != nil {
					h.AfterEmbed(ctx, embeddings, err)
				}
			}
		},
	}
}
