// Package retriever provides the Retriever interface and implementations for
// the RAG pipeline. Retrievers search for relevant documents given a query,
// supporting strategies ranging from simple vector search to advanced
// multi-step retrieval with LLM-guided evaluation.
//
// # Interface
//
// The core interface is [Retriever]:
//
//	type Retriever interface {
//	    Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error)
//	}
//
// Implementations must be safe for concurrent use.
//
// # Registry
//
// The package follows Beluga's registry pattern. Implementations register via
// init() and are instantiated with [New]:
//
//	r, err := retriever.New("vector", cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	docs, err := r.Retrieve(ctx, "what is Go?", retriever.WithTopK(10))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Retrieval Options
//
// Retrieval behaviour is configurable via functional options:
//   - [WithTopK] — maximum number of documents to return (default: 10)
//   - [WithThreshold] — minimum relevance score
//   - [WithMetadata] — restrict results by metadata key-value pairs
//
// # Built-in Retrievers
//
// Vector search:
//   - [NewVectorStoreRetriever] — embeds query, searches a [vectorstore.VectorStore]
//
// Hybrid and ensemble:
//   - [NewHybridRetriever] — combines dense vector + sparse BM25 with RRF fusion
//   - [NewEnsembleRetriever] — merges results from multiple retrievers using [FusionStrategy]
//
// LLM-augmented:
//   - [NewHyDERetriever] — Hypothetical Document Embeddings (generates hypothetical
//     answer, embeds it, searches for similar real documents)
//   - [NewCRAGRetriever] — Corrective RAG (evaluates retrieval quality with an LLM,
//     falls back to web search when relevance is low)
//   - [NewMultiQueryRetriever] — generates query variations using an LLM for improved recall
//   - [NewAdaptiveRetriever] — classifies query complexity and routes to appropriate strategy
//
// Re-ranking:
//   - [NewRerankRetriever] — wraps a retriever with a [Reranker] for two-stage
//     retrieve-then-rerank
//
// # Fusion Strategies
//
//   - [NewRRFStrategy] — Reciprocal Rank Fusion (default, k=60)
//   - [NewWeightedStrategy] — weighted score fusion
//
// # Middleware and Hooks
//
// Cross-cutting concerns are layered via [Middleware] and [Hooks]:
//
//	r = retriever.ApplyMiddleware(r,
//	    retriever.WithHooks(retriever.Hooks{
//	        BeforeRetrieve: func(ctx context.Context, query string) error {
//	            log.Printf("retrieving for: %s", query)
//	            return nil
//	        },
//	    }),
//	)
//
// [Hooks] also supports OnRerank for observing re-ranking operations.
package retriever
