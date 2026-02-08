package retriever_test

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// mockChatModel implements llm.ChatModel for testing.
type mockChatModel struct {
	generateFn func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error)
	calls      int
}

func (m *mockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	m.calls++
	if m.generateFn != nil {
		return m.generateFn(ctx, msgs, opts...)
	}
	return schema.NewAIMessage("mock response"), nil
}

func (m *mockChatModel) Stream(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {}
}

func (m *mockChatModel) BindTools(_ []schema.ToolDefinition) llm.ChatModel {
	return m
}

func (m *mockChatModel) ModelID() string {
	return "mock-model"
}

// mockEmbedder implements embedding.Embedder for testing.
type mockEmbedder struct {
	embedFn       func(ctx context.Context, texts []string) ([][]float32, error)
	embedSingleFn func(ctx context.Context, text string) ([]float32, error)
	dimensions    int
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}
	// Default: return fixed vector for each text.
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{1.0, 0.0, 0.0}
	}
	return result, nil
}

func (m *mockEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if m.embedSingleFn != nil {
		return m.embedSingleFn(ctx, text)
	}
	// Default: delegate to Embed.
	vecs, err := m.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (m *mockEmbedder) Dimensions() int {
	if m.dimensions > 0 {
		return m.dimensions
	}
	return 3
}

// mockVectorStore implements vectorstore.VectorStore for testing.
type mockVectorStore struct {
	addFn    func(ctx context.Context, docs []schema.Document, embeddings [][]float32) error
	searchFn func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error)
	deleteFn func(ctx context.Context, ids []string) error
}

func (m *mockVectorStore) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	if m.addFn != nil {
		return m.addFn(ctx, docs, embeddings)
	}
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, query, k, opts...)
	}
	// Default: return empty results.
	return []schema.Document{}, nil
}

func (m *mockVectorStore) Delete(ctx context.Context, ids []string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, ids)
	}
	return nil
}

// mockRetriever implements retriever.Retriever for testing.
type mockRetriever struct {
	retrieveFn func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error)
}

func (m *mockRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
	if m.retrieveFn != nil {
		return m.retrieveFn(ctx, query, opts...)
	}
	return []schema.Document{}, nil
}

// mockBM25Searcher implements retriever.BM25Searcher for testing.
type mockBM25Searcher struct {
	searchFn func(ctx context.Context, query string, k int) ([]schema.Document, error)
}

func (m *mockBM25Searcher) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, query, k)
	}
	return []schema.Document{}, nil
}

// mockWebSearcher implements retriever.WebSearcher for testing.
type mockWebSearcher struct {
	searchFn func(ctx context.Context, query string, k int) ([]schema.Document, error)
}

func (m *mockWebSearcher) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, query, k)
	}
	return []schema.Document{}, nil
}

// Helper to create test documents.
func makeDocs(ids ...string) []schema.Document {
	docs := make([]schema.Document, len(ids))
	for i, id := range ids {
		docs[i] = schema.Document{
			ID:      id,
			Content: "content for " + id,
			Score:   float64(len(ids) - i),
		}
	}
	return docs
}

// Compile-time interface checks.
var (
	_ llm.ChatModel            = (*mockChatModel)(nil)
	_ embedding.Embedder       = (*mockEmbedder)(nil)
	_ vectorstore.VectorStore  = (*mockVectorStore)(nil)
	_ retriever.Retriever      = (*mockRetriever)(nil)
)
