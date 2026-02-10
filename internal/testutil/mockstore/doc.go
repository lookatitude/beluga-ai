// Package mockstore provides a mock implementation of the
// vectorstore.VectorStore interface for testing.
//
// This is an internal package and is not part of the public API. It is used
// by RAG pipeline tests, retriever tests, and any test that requires a
// vector store without an external database.
//
// # MockVectorStore
//
// [MockVectorStore] implements the vectorstore.VectorStore interface with
// in-memory document storage and configurable behavior. It supports canned
// search results, error injection, custom functions for each operation,
// and call tracking for assertions.
//
// Create a mock with functional options:
//
//	store := mockstore.New(
//	    mockstore.WithDocuments([]schema.Document{
//	        {ID: "1", Content: "first document", Score: 0.95},
//	        {ID: "2", Content: "second document", Score: 0.80},
//	    }),
//	)
//
// Configure error injection:
//
//	store := mockstore.New(
//	    mockstore.WithError(errors.New("connection failed")),
//	)
//
// Use custom functions for dynamic behavior:
//
//	store := mockstore.New(
//	    mockstore.WithSearchFunc(func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
//	        // custom search logic
//	    }),
//	)
//
// Inspect call history:
//
//	_ = store.Add(ctx, docs, embeddings)
//	fmt.Println(store.AddCalls())  // 1
//	fmt.Println(store.LastDocs())  // documents passed to Add
//
// The mock is safe for concurrent use. Call [MockVectorStore.Reset] to clear
// all state between test cases.
package mockstore
