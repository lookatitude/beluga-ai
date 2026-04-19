// Package colbert implements a ColBERT-style late interaction retriever for
// the Beluga AI RAG pipeline.
//
// Late interaction models like ColBERT and ColPali produce per-token embeddings
// and compute relevance via MaxSim: for each query token, the maximum cosine
// similarity across all document tokens is found, then these per-token
// maximums are summed to produce the final score. This preserves fine-grained
// token-level matching while remaining efficient through pre-computed document
// embeddings.
//
// # Architecture
//
// The package provides three components:
//
//   - [MaxSimScorer] — computes the MaxSim score between query and document
//     token embeddings.
//   - [ColBERTIndex] — stores pre-computed per-token document embeddings and
//     supports search by MaxSim scoring. [NewInMemoryIndex] provides a
//     brute-force thread-safe implementation.
//   - [ColBERTRetriever] — implements [retriever.Retriever] by encoding queries
//     with a [embedding.MultiVectorEmbedder] and searching a [ColBERTIndex].
//
// # Registry
//
// The retriever is registered as "colbert" with the retriever registry. Import
// this package with a blank import to make it available:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/retriever/colbert"
//
// # Usage
//
//	idx := colbert.NewInMemoryIndex()
//	_ = idx.Add(ctx, "doc1", doc1TokenVecs)
//	_ = idx.Add(ctx, "doc2", doc2TokenVecs)
//
//	r, err := colbert.NewColBERTRetriever(
//	    colbert.WithEmbedder(multiVecEmbedder),
//	    colbert.WithIndex(idx),
//	    colbert.WithTopK(5),
//	)
//	if err != nil {
//	    // handle error
//	}
//	docs, err := r.Retrieve(ctx, "what is late interaction?")
package colbert
