package inmemory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vsfactory "github.com/lookatitude/beluga-ai/pkg/vectorstores/factory"
)

func init() {
	vsfactory.Register("inmemory", func(config map[string]interface{}) (vectorstores.VectorStore, error) {
		// For InMemoryVectorStore, the config might specify a default embedder name
		// or other parameters. For now, we assume it might need an embedder, but
		// the NewInMemoryVectorStore constructor takes an iface.Embedder directly.
		// This factory function would need to resolve that embedder from a config name if specified.
		// For simplicity in this example, we pass nil, meaning an embedder must be provided
		// during AddDocuments or SimilaritySearchByQuery if not set at construction.
		// A more robust implementation would fetch an embedder from an embedder factory if a name is in config.
		return NewInMemoryVectorStore(nil), nil // Pass nil for embedder, to be set later or per-call
	})
}

// InMemoryVectorStore is a simple in-memory implementation of the VectorStore interface.
// It is not recommended for production use with large datasets due to performance and memory limitations.
type InMemoryVectorStore struct {
	mu         sync.RWMutex
	documents  []schema.Document
	embeddings [][]float32
	embedder   iface.Embedder // Store the embedder used if documents are added without pre-computed embeddings
	name       string
}

// NewInMemoryVectorStore creates a new InMemoryVectorStore.
// Optionally, an embedder can be provided if documents will be added without pre-computed embeddings directly.
func NewInMemoryVectorStore(embedder iface.Embedder) *InMemoryVectorStore {
	return &InMemoryVectorStore{
		documents:  make([]schema.Document, 0),
		embeddings: make([][]float32, 0),
		embedder:   embedder,
		name:       "inmemory",
	}
}

// AddDocuments adds documents to the store. If an embedder was provided at construction or here,
// it will be used to generate embeddings. Otherwise, embeddings must be pre-computed and present in docs.
func (s *InMemoryVectorStore) AddDocuments(ctx context.Context, docs []schema.Document, embedder iface.Embedder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentEmbedder := s.embedder
	if embedder != nil {
		currentEmbedder = embedder // Override with provided embedder for this call
	}

	if currentEmbedder == nil && len(docs) > 0 && (docs[0].Embedding == nil || len(docs[0].Embedding) == 0) {
		return fmt.Errorf("InMemoryVectorStore: embedder is required if documents do not have pre-computed embeddings")
	}

	for _, doc := range docs {
		var docEmbedding []float32
		if doc.Embedding != nil && len(doc.Embedding) > 0 {
			docEmbedding = doc.Embedding
		} else if currentEmbedder != nil {
			embeds, err := currentEmbedder.EmbedDocuments(ctx, []string{doc.PageContent})
			if err != nil {
				return fmt.Errorf("failed to embed document 	%s	: %w", doc.ID, err)
			}
			if len(embeds) == 0 {
				return fmt.Errorf("embedder returned no embeddings for document 	%s", doc.ID)
			}
			docEmbedding = embeds[0]
		} else {
			return fmt.Errorf("document 	%s	 has no embedding and no embedder is available", doc.ID)
		}

		s.documents = append(s.documents, doc)
		s.embeddings = append(s.embeddings, docEmbedding)
	}
	return nil
}

// SimilaritySearch performs a similarity search using a pre-computed query vector.
func (s *InMemoryVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int) ([]schema.Document, []float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.documents) == 0 {
		return []schema.Document{}, []float32{}, nil
	}
	if k <= 0 {
		return []schema.Document{}, []float32{}, fmt.Errorf("k must be greater than 0")
	}

	type scoreDoc struct {
		doc   schema.Document
		score float32
		index int
	}

	scores := make([]scoreDoc, len(s.documents))
	for i, emb := range s.embeddings {
		score, err := cosineSimilarity(queryVector, emb)
		if err != nil {
			return nil, nil, fmt.Errorf("error calculating similarity for doc index %d: %w", i, err)
		}
		scores[i] = scoreDoc{doc: s.documents[i], score: score, index: i}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	numResults := k
	if k > len(scores) {
		numResults = len(scores)
	}

	resultDocs := make([]schema.Document, numResults)
	resultScores := make([]float32, numResults)
	for i := 0; i < numResults; i++ {
		resultDocs[i] = scores[i].doc
		resultScores[i] = scores[i].score
	}

	return resultDocs, resultScores, nil
}

// SimilaritySearchByQuery generates an embedding for the query and then performs a similarity search.
func (s *InMemoryVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder iface.Embedder) ([]schema.Document, []float32, error) {
	currentEmbedder := s.embedder
	if embedder != nil {
		currentEmbedder = embedder
	}
	if currentEmbedder == nil {
		return nil, nil, fmt.Errorf("InMemoryVectorStore: embedder is required for SimilaritySearchByQuery")
	}

	queryEmbedding, err := currentEmbedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to embed query: %w", err)
	}

	return s.SimilaritySearch(ctx, queryEmbedding, k)
}

// GetName returns the name of the vector store.
func (s *InMemoryVectorStore) GetName() string {
	return s.name
}

func cosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors have different lengths: %d vs %d", len(a), len(b))
	}
	if len(a) == 0 {
		return 0, fmt.Errorf("vectors are empty")
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("cannot compute cosine similarity with zero vector")
	}

	return dotProduct / (sqrt(normA) * sqrt(normB)), nil
}

func sqrt(n float32) float32 {
	if n < 0 {
		return 0
	}
	x := n
	y := float32(1.0)
	e := float32(0.000001)
	for (x - y) > e {
		x = (x + y) / 2
		y = n / x
	}
	return x
}

var _ vectorstores.VectorStore = (*InMemoryVectorStore)(nil)

