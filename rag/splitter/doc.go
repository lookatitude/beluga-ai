// Package splitter provides text splitting capabilities for the RAG pipeline.
// It defines the [TextSplitter] interface for dividing text content into smaller
// chunks suitable for embedding and retrieval.
//
// # Interface
//
// The core interface is [TextSplitter]:
//
//	type TextSplitter interface {
//	    Split(ctx context.Context, text string) ([]string, error)
//	    SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error)
//	}
//
// SplitDocuments preserves and augments original metadata with chunk_index,
// chunk_total, and parent_id fields.
//
// # Registry
//
// The package follows Beluga's registry pattern. Implementations register via
// init() and are instantiated with [New]:
//
//	s, err := splitter.New("recursive", config.ProviderConfig{
//	    Options: map[string]any{"chunk_size": 1000, "chunk_overlap": 200},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	chunks, err := s.Split(ctx, longText)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Use [List] to discover all registered splitter names.
//
// # Built-in Splitters
//
//   - "recursive" — recursive character splitter with configurable separators
//   - "markdown" — Markdown-aware splitter that respects heading hierarchy
//   - "token" — token-based splitter using an [llm.Tokenizer]
//
// # Custom Splitter
//
// To add a custom text splitter:
//
//	func init() {
//	    splitter.Register("custom", func(cfg config.ProviderConfig) (splitter.TextSplitter, error) {
//	        return &mySplitter{chunkSize: 500}, nil
//	    })
//	}
package splitter
