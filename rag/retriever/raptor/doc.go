// Package raptor implements RAPTOR (Recursive Abstractive Processing for
// Tree-Organized Retrieval), a hierarchical tree-based retrieval strategy.
//
// RAPTOR recursively clusters document chunks, summarizes each cluster using
// an LLM, embeds the summaries, and repeats the process to build a multi-level
// tree. At query time, the collapsed tree (all nodes flattened) is searched
// via cosine similarity, allowing retrieval across multiple abstraction levels.
//
// Usage:
//
//	builder := raptor.NewTreeBuilder(
//	    raptor.WithEmbedder(embedder),
//	    raptor.WithSummarizer(raptor.NewLLMSummarizer(model)),
//	    raptor.WithMaxLevels(3),
//	)
//	tree, err := builder.Build(ctx, docs)
//	r := raptor.NewRAPTORRetriever(raptor.WithTree(tree), raptor.WithRaptorTopK(5))
//	results, err := r.Retrieve(ctx, "query")
//
// The package registers itself as "raptor" in the retriever registry.
package raptor
