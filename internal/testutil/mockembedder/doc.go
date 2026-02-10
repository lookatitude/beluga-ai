// Package mockembedder provides a mock implementation of the
// embedding.Embedder interface for testing.
//
// This is an internal package and is not part of the public API. It is used
// by RAG pipeline tests, retriever tests, and any test that requires an
// Embedder without calling a real embedding provider.
//
// # MockEmbedder
//
// [MockEmbedder] implements the embedding.Embedder interface with configurable
// behavior. It supports canned embeddings, error injection, custom embed
// functions, and call tracking for assertions.
//
// Create a mock with functional options:
//
//	embedder := mockembedder.New(
//	    mockembedder.WithEmbeddings([][]float32{{0.1, 0.2}, {0.3, 0.4}}),
//	    mockembedder.WithDimensions(2),
//	)
//
// Configure error injection:
//
//	embedder := mockembedder.New(
//	    mockembedder.WithError(errors.New("embed failed")),
//	)
//
// Use a custom function for dynamic behavior:
//
//	embedder := mockembedder.New(
//	    mockembedder.WithEmbedFunc(func(ctx context.Context, texts []string) ([][]float32, error) {
//	        // custom embedding logic
//	    }),
//	)
//
// Inspect call history:
//
//	_ = embedder.Embed(ctx, []string{"hello"})
//	fmt.Println(embedder.EmbedCalls()) // 1
//	fmt.Println(embedder.LastTexts())  // ["hello"]
//
// The mock is safe for concurrent use. Call [MockEmbedder.Reset] to clear
// all state between test cases.
package mockembedder
