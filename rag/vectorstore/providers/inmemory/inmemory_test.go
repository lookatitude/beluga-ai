package inmemory

import (
	"context"
	"math"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	store := New()
	require.NotNil(t, store)
	assert.NotNil(t, store.entries)
}

func TestStore_Add(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "first document"},
		{ID: "doc2", Content: "second document"},
	}
	embeddings := [][]float32{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	// Verify documents were added.
	assert.Len(t, store.entries, 2)
	assert.Contains(t, store.entries, "doc1")
	assert.Contains(t, store.entries, "doc2")
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "first document"},
	}
	embeddings := [][]float32{
		{0.1, 0.2},
		{0.3, 0.4},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs length")
}

func TestStore_Add_Overwrite(t *testing.T) {
	store := New()

	// Add initial document.
	docs1 := []schema.Document{{ID: "doc1", Content: "original"}}
	embeddings1 := [][]float32{{0.1, 0.2, 0.3}}
	err := store.Add(context.Background(), docs1, embeddings1)
	require.NoError(t, err)

	// Overwrite with new content.
	docs2 := []schema.Document{{ID: "doc1", Content: "updated"}}
	embeddings2 := [][]float32{{0.9, 0.8, 0.7}}
	err = store.Add(context.Background(), docs2, embeddings2)
	require.NoError(t, err)

	assert.Len(t, store.entries, 1)
	assert.Equal(t, "updated", store.entries["doc1"].doc.Content)
}

func TestStore_Search_Empty(t *testing.T) {
	store := New()

	query := []float32{0.1, 0.2, 0.3}
	results, err := store.Search(context.Background(), query, 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestStore_Search_CosineSimilarity(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "first"},
		{ID: "doc2", Content: "second"},
		{ID: "doc3", Content: "third"},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0}, // Orthogonal to query
		{0.0, 1.0, 0.0}, // Identical to query
		{0.5, 0.5, 0.0}, // Partial match
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	query := []float32{0.0, 1.0, 0.0}
	results, err := store.Search(context.Background(), query, 3)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Results should be sorted by descending similarity.
	// doc2 (identical) should be first, doc3 (partial) second, doc1 (orthogonal) third.
	assert.Equal(t, "doc2", results[0].ID)
	assert.Equal(t, "doc3", results[1].ID)
	assert.Equal(t, "doc1", results[2].ID)

	// Check scores are populated.
	assert.Greater(t, results[0].Score, results[1].Score)
	assert.Greater(t, results[1].Score, results[2].Score)
}

func TestStore_Search_LimitK(t *testing.T) {
	store := New()

	docs := make([]schema.Document, 10)
	embeddings := make([][]float32, 10)
	for i := 0; i < 10; i++ {
		docs[i] = schema.Document{ID: "doc" + string(rune('0'+i)), Content: "document"}
		embeddings[i] = []float32{float32(i), 0.0, 0.0}
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	query := []float32{5.0, 0.0, 0.0}
	results, err := store.Search(context.Background(), query, 3)
	require.NoError(t, err)

	// Should return only top 3 results.
	assert.Len(t, results, 3)
}

func TestStore_Search_WithFilter(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "first", Metadata: map[string]any{"category": "A"}},
		{ID: "doc2", Content: "second", Metadata: map[string]any{"category": "B"}},
		{ID: "doc3", Content: "third", Metadata: map[string]any{"category": "A"}},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0},
		{0.9, 0.1, 0.0},
		{0.8, 0.2, 0.0},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	query := []float32{1.0, 0.0, 0.0}
	filter := map[string]any{"category": "A"}

	results, err := store.Search(context.Background(), query, 5, vectorstore.WithFilter(filter))
	require.NoError(t, err)

	// Should only return documents with category "A".
	assert.Len(t, results, 2)
	for _, doc := range results {
		assert.Equal(t, "A", doc.Metadata["category"])
	}
}

func TestStore_Search_WithThreshold(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "first"},
		{ID: "doc2", Content: "second"},
		{ID: "doc3", Content: "third"},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0}, // High similarity
		{0.5, 0.5, 0.0}, // Medium similarity
		{0.0, 0.0, 1.0}, // Low similarity
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	query := []float32{1.0, 0.0, 0.0}

	// Set a threshold to filter out low similarity results.
	results, err := store.Search(context.Background(), query, 5, vectorstore.WithThreshold(0.5))
	require.NoError(t, err)

	// Should only return documents with similarity >= 0.5.
	assert.LessOrEqual(t, len(results), 2)
	for _, doc := range results {
		assert.GreaterOrEqual(t, doc.Score, 0.5)
	}
}

func TestStore_Search_DotProduct(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "first"},
		{ID: "doc2", Content: "second"},
	}
	embeddings := [][]float32{
		{1.0, 2.0, 3.0},
		{-1.0, -2.0, -3.0},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	query := []float32{1.0, 2.0, 3.0}
	results, err := store.Search(context.Background(), query, 2, vectorstore.WithStrategy(vectorstore.DotProduct))
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Dot product of query with doc1 = 1*1 + 2*2 + 3*3 = 14 (positive)
	// Dot product of query with doc2 = 1*-1 + 2*-2 + 3*-3 = -14 (negative)
	// Results should be sorted descending, so doc1 first.
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "doc2", results[1].ID)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestStore_Search_Euclidean(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "near"},
		{ID: "doc2", Content: "far"},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0}, // Distance = 1.0 from query
		{10.0, 0.0, 0.0}, // Distance = 9.0 from query
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	query := []float32{0.0, 0.0, 0.0}
	results, err := store.Search(context.Background(), query, 2, vectorstore.WithStrategy(vectorstore.Euclidean))
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Euclidean strategy returns negative distance, so higher (less negative) is better.
	// doc1 (distance 1) should have score -1, doc2 (distance 9) should have score -9.
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "doc2", results[1].ID)
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestStore_Delete(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "first"},
		{ID: "doc2", Content: "second"},
		{ID: "doc3", Content: "third"},
	}
	embeddings := [][]float32{
		{0.1, 0.2},
		{0.3, 0.4},
		{0.5, 0.6},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)
	assert.Len(t, store.entries, 3)

	// Delete doc2.
	err = store.Delete(context.Background(), []string{"doc2"})
	require.NoError(t, err)

	assert.Len(t, store.entries, 2)
	assert.Contains(t, store.entries, "doc1")
	assert.NotContains(t, store.entries, "doc2")
	assert.Contains(t, store.entries, "doc3")
}

func TestStore_Delete_NonExistent(t *testing.T) {
	store := New()

	// Deleting non-existent IDs should not error.
	err := store.Delete(context.Background(), []string{"nonexistent"})
	require.NoError(t, err)
}

func TestStore_Delete_Multiple(t *testing.T) {
	store := New()

	docs := []schema.Document{
		{ID: "doc1", Content: "first"},
		{ID: "doc2", Content: "second"},
		{ID: "doc3", Content: "third"},
	}
	embeddings := [][]float32{
		{0.1, 0.2},
		{0.3, 0.4},
		{0.5, 0.6},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	// Delete multiple documents.
	err = store.Delete(context.Background(), []string{"doc1", "doc3"})
	require.NoError(t, err)

	assert.Len(t, store.entries, 1)
	assert.Contains(t, store.entries, "doc2")
}

func TestStore_InterfaceCompliance(t *testing.T) {
	// Compile-time check that Store implements vectorstore.VectorStore.
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestRegistry_Integration(t *testing.T) {
	// Test that the provider is registered.
	store, err := vectorstore.New("inmemory", config.ProviderConfig{})
	require.NoError(t, err)
	require.NotNil(t, store)

	// Test basic operations.
	docs := []schema.Document{{ID: "test", Content: "test doc"}}
	embeddings := [][]float32{{0.1, 0.2, 0.3}}

	err = store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 1)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{1.0, 2.0, 3.0}

	sim := cosineSimilarity(a, b)
	assert.InDelta(t, 1.0, sim, 0.0001, "identical vectors should have similarity 1.0")
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	a := []float32{1.0, 0.0, 0.0}
	b := []float32{0.0, 1.0, 0.0}

	sim := cosineSimilarity(a, b)
	assert.InDelta(t, 0.0, sim, 0.0001, "orthogonal vectors should have similarity 0.0")
}

func TestCosineSimilarity_OppositeVectors(t *testing.T) {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{-1.0, -2.0, -3.0}

	sim := cosineSimilarity(a, b)
	assert.InDelta(t, -1.0, sim, 0.0001, "opposite vectors should have similarity -1.0")
}

func TestDotProduct(t *testing.T) {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{4.0, 5.0, 6.0}

	expected := 1.0*4.0 + 2.0*5.0 + 3.0*6.0 // = 32.0
	result := dotProduct(a, b)
	assert.InDelta(t, expected, result, 0.0001)
}

func TestEuclideanDistance(t *testing.T) {
	a := []float32{0.0, 0.0, 0.0}
	b := []float32{3.0, 4.0, 0.0}

	expected := 5.0 // sqrt(3^2 + 4^2) = 5
	result := euclideanDistance(a, b)
	assert.InDelta(t, expected, result, 0.0001)
}

func TestMatchesFilter_NilFilter(t *testing.T) {
	doc := schema.Document{
		ID:       "doc1",
		Content:  "test",
		Metadata: map[string]any{"key": "value"},
	}

	// Nil filter should match everything.
	assert.True(t, matchesFilter(doc, nil))
}

func TestMatchesFilter_EmptyFilter(t *testing.T) {
	doc := schema.Document{
		ID:       "doc1",
		Content:  "test",
		Metadata: map[string]any{"key": "value"},
	}

	// Empty filter should match everything.
	assert.True(t, matchesFilter(doc, map[string]any{}))
}

func TestMatchesFilter_Match(t *testing.T) {
	doc := schema.Document{
		ID:      "doc1",
		Content: "test",
		Metadata: map[string]any{
			"category": "A",
			"priority": 1,
		},
	}

	filter := map[string]any{
		"category": "A",
	}

	assert.True(t, matchesFilter(doc, filter))
}

func TestMatchesFilter_NoMatch(t *testing.T) {
	doc := schema.Document{
		ID:      "doc1",
		Content: "test",
		Metadata: map[string]any{
			"category": "A",
		},
	}

	filter := map[string]any{
		"category": "B",
	}

	assert.False(t, matchesFilter(doc, filter))
}

func TestMatchesFilter_MissingKey(t *testing.T) {
	doc := schema.Document{
		ID:       "doc1",
		Content:  "test",
		Metadata: map[string]any{"category": "A"},
	}

	filter := map[string]any{
		"missing_key": "value",
	}

	assert.False(t, matchesFilter(doc, filter))
}

func TestMatchesFilter_NilMetadata(t *testing.T) {
	doc := schema.Document{
		ID:       "doc1",
		Content:  "test",
		Metadata: nil,
	}

	filter := map[string]any{
		"key": "value",
	}

	assert.False(t, matchesFilter(doc, filter))
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := New()

	// Add some initial documents.
	docs := []schema.Document{
		{ID: "doc1", Content: "first"},
		{ID: "doc2", Content: "second"},
	}
	embeddings := [][]float32{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	// Concurrent reads should work.
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 2)
			assert.NoError(t, err)
			done <- true
		}()
	}

	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestSimilarity_MismatchedDimensions(t *testing.T) {
	// These functions handle mismatched dimensions gracefully.
	a := []float32{1.0, 2.0}
	b := []float32{1.0, 2.0, 3.0}

	// Should return 0 for mismatched dimensions.
	assert.Equal(t, 0.0, cosineSimilarity(a, b))
	assert.Equal(t, 0.0, dotProduct(a, b))
	assert.Equal(t, 0.0, euclideanDistance(a, b))
}

func TestSimilarity_EmptyVectors(t *testing.T) {
	a := []float32{}
	b := []float32{}

	assert.Equal(t, 0.0, cosineSimilarity(a, b))
	assert.Equal(t, 0.0, dotProduct(a, b))
	assert.Equal(t, 0.0, euclideanDistance(a, b))
}

func TestSimilarity_ZeroVectors(t *testing.T) {
	a := []float32{0.0, 0.0, 0.0}
	b := []float32{1.0, 2.0, 3.0}

	// Cosine similarity with zero vector should be 0.
	assert.Equal(t, 0.0, cosineSimilarity(a, b))

	// Dot product with zero vector should be 0.
	assert.InDelta(t, 0.0, dotProduct(a, b), 0.0001)

	// Euclidean distance should be the magnitude of b.
	expected := math.Sqrt(1.0*1.0 + 2.0*2.0 + 3.0*3.0)
	assert.InDelta(t, expected, euclideanDistance(a, b), 0.0001)
}
