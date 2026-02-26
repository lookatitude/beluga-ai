package retriever

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// Retriever searches for documents relevant to a query. Implementations must
// be safe for concurrent use.
type Retriever interface {
	// Retrieve returns documents relevant to the query, ordered by
	// decreasing relevance. Options configure top-k, threshold, and
	// metadata filters.
	Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error)
}

// Config holds configuration for a Retrieve call.
type Config struct {
	// TopK is the maximum number of documents to return.
	TopK int
	// Threshold is the minimum relevance score for returned documents.
	// Documents with scores below this threshold are excluded.
	Threshold float64
	// Metadata restricts results to documents whose metadata matches
	// all key-value pairs.
	Metadata map[string]any
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		TopK: 10,
	}
}

// Option configures a Retrieve call.
type Option func(*Config)

// WithTopK sets the maximum number of documents to return.
func WithTopK(k int) Option {
	return func(c *Config) {
		c.TopK = k
	}
}

// WithThreshold sets the minimum relevance score for returned documents.
func WithThreshold(t float64) Option {
	return func(c *Config) {
		c.Threshold = t
	}
}

// WithMetadata restricts results to documents whose metadata matches all
// key-value pairs in the filter map.
func WithMetadata(m map[string]any) Option {
	return func(c *Config) {
		c.Metadata = m
	}
}

// ApplyOptions applies the given options to a default Config and returns it.
func ApplyOptions(opts ...Option) Config {
	cfg := DefaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// Hooks provides optional callback functions invoked around retriever
// operations. All fields are optional; nil hooks are skipped.
type Hooks struct {
	// BeforeRetrieve is called before each Retrieve call with the query.
	// Returning an error aborts the call.
	BeforeRetrieve func(ctx context.Context, query string) error

	// AfterRetrieve is called after Retrieve completes with the results
	// and any error.
	AfterRetrieve func(ctx context.Context, docs []schema.Document, err error)

	// OnRerank is called when a re-ranking step is applied to the results.
	OnRerank func(ctx context.Context, query string, before, after []schema.Document)
}

func composeBeforeRetrieve(hooks []Hooks) func(context.Context, string) error {
	return func(ctx context.Context, query string) error {
		for _, h := range hooks {
			if h.BeforeRetrieve != nil {
				if err := h.BeforeRetrieve(ctx, query); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterRetrieve(hooks []Hooks) func(context.Context, []schema.Document, error) {
	return func(ctx context.Context, docs []schema.Document, err error) {
		for _, h := range hooks {
			if h.AfterRetrieve != nil {
				h.AfterRetrieve(ctx, docs, err)
			}
		}
	}
}

func composeOnRerank(hooks []Hooks) func(context.Context, string, []schema.Document, []schema.Document) {
	return func(ctx context.Context, query string, before, after []schema.Document) {
		for _, h := range hooks {
			if h.OnRerank != nil {
				h.OnRerank(ctx, query, before, after)
			}
		}
	}
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For BeforeRetrieve, the first error returned short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeRetrieve: composeBeforeRetrieve(h),
		AfterRetrieve:  composeAfterRetrieve(h),
		OnRerank:       composeOnRerank(h),
	}
}
