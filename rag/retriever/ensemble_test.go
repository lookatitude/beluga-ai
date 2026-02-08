package retriever

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/schema"
)

// --- Tests for RRFStrategy ---

func TestRRFStrategy_Fuse_OverlappingDocs(t *testing.T) {
	// Two result sets with overlapping documents
	sets := [][]schema.Document{
		{
			{ID: "a", Content: "doc a"},
			{ID: "b", Content: "doc b"},
			{ID: "c", Content: "doc c"},
		},
		{
			{ID: "b", Content: "doc b"},
			{ID: "d", Content: "doc d"},
			{ID: "a", Content: "doc a"},
		},
	}

	rrf := NewRRFStrategy(60)
	fused, err := rrf.Fuse(context.Background(), sets)
	require.NoError(t, err)
	assert.Len(t, fused, 4, "should have 4 unique documents")

	// Documents appearing in both sets should have higher scores
	// a: 1/(60+0+1) + 1/(60+2+1) = 1/61 + 1/63 ≈ 0.0164 + 0.0159 = 0.0323
	// b: 1/(60+1+1) + 1/(60+0+1) = 1/62 + 1/61 ≈ 0.0161 + 0.0164 = 0.0325
	// c: 1/(60+2+1) = 1/63 ≈ 0.0159
	// d: 1/(60+1+1) = 1/62 ≈ 0.0161

	// b and a should be top 2 (in either order, very close scores)
	topIDs := map[string]bool{fused[0].ID: true, fused[1].ID: true}
	assert.True(t, topIDs["a"], "a should be in top 2")
	assert.True(t, topIDs["b"], "b should be in top 2")

	// Verify scores are in descending order
	for i := 1; i < len(fused); i++ {
		assert.GreaterOrEqual(t, fused[i-1].Score, fused[i].Score,
			"documents should be sorted by score desc")
	}
}

func TestRRFStrategy_Fuse_SingleResultSet(t *testing.T) {
	sets := [][]schema.Document{
		{
			{ID: "a", Content: "doc a"},
			{ID: "b", Content: "doc b"},
			{ID: "c", Content: "doc c"},
		},
	}

	rrf := NewRRFStrategy(60)
	fused, err := rrf.Fuse(context.Background(), sets)
	require.NoError(t, err)
	assert.Len(t, fused, 3)

	// Verify scores: 1/(60+rank+1)
	expectedScores := []float64{
		1.0 / 61.0, // rank 0
		1.0 / 62.0, // rank 1
		1.0 / 63.0, // rank 2
	}

	for i, doc := range fused {
		assert.InDelta(t, expectedScores[i], doc.Score, 0.0001,
			"doc %s should have score %.4f", doc.ID, expectedScores[i])
	}
}

func TestRRFStrategy_Fuse_EmptyResultSets(t *testing.T) {
	tests := []struct {
		name string
		sets [][]schema.Document
	}{
		{
			name: "nil sets",
			sets: nil,
		},
		{
			name: "empty slice",
			sets: [][]schema.Document{},
		},
		{
			name: "sets with empty documents",
			sets: [][]schema.Document{
				{},
				{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rrf := NewRRFStrategy(60)
			fused, err := rrf.Fuse(context.Background(), tt.sets)
			require.NoError(t, err)
			assert.Empty(t, fused)
		})
	}
}

func TestRRFStrategy_DefaultK(t *testing.T) {
	tests := []struct {
		name     string
		k        int
		expected int
	}{
		{"zero uses default", 0, 60},
		{"negative uses default", -1, 60},
		{"positive keeps value", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rrf := NewRRFStrategy(tt.k)
			assert.Equal(t, tt.expected, rrf.K)
		})
	}
}

func TestRRFStrategy_DifferentKValues(t *testing.T) {
	sets := [][]schema.Document{
		{{ID: "a"}, {ID: "b"}},
		{{ID: "b"}, {ID: "a"}},
	}

	// Test with different K values
	tests := []struct {
		k int
	}{
		{k: 1},
		{k: 10},
		{k: 60},
		{k: 100},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			rrf := NewRRFStrategy(tt.k)
			fused, err := rrf.Fuse(context.Background(), sets)
			require.NoError(t, err)
			assert.Len(t, fused, 2)
			// Both docs appear in both sets, so they should have equal scores
			assert.InDelta(t, fused[0].Score, fused[1].Score, 0.0001)
		})
	}
}

// --- Tests for WeightedStrategy ---

func TestWeightedStrategy_Fuse_Success(t *testing.T) {
	sets := [][]schema.Document{
		{
			{ID: "a", Score: 0.9},
			{ID: "b", Score: 0.5},
		},
		{
			{ID: "b", Score: 0.8},
			{ID: "c", Score: 0.7},
		},
	}

	ws := NewWeightedStrategy([]float64{0.7, 0.3})
	fused, err := ws.Fuse(context.Background(), sets)
	require.NoError(t, err)
	assert.Len(t, fused, 3)

	// Find scores for each doc
	scoreMap := make(map[string]float64)
	for _, doc := range fused {
		scoreMap[doc.ID] = doc.Score
	}

	// a: 0.9 * 0.7 = 0.63
	// b: 0.5 * 0.7 + 0.8 * 0.3 = 0.35 + 0.24 = 0.59
	// c: 0.7 * 0.3 = 0.21

	assert.InDelta(t, 0.63, scoreMap["a"], 0.01)
	assert.InDelta(t, 0.59, scoreMap["b"], 0.01)
	assert.InDelta(t, 0.21, scoreMap["c"], 0.01)

	// Verify sorted by score desc
	assert.Equal(t, "a", fused[0].ID)
	assert.Equal(t, "b", fused[1].ID)
	assert.Equal(t, "c", fused[2].ID)
}

func TestWeightedStrategy_Fuse_MismatchedWeights(t *testing.T) {
	sets := [][]schema.Document{
		{{ID: "a", Score: 1.0}},
		{{ID: "b", Score: 1.0}},
	}

	ws := NewWeightedStrategy([]float64{0.5}) // Only 1 weight for 2 sets
	_, err := ws.Fuse(context.Background(), sets)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 weights for 2 result sets")
}

func TestWeightedStrategy_Fuse_ZeroWeights(t *testing.T) {
	sets := [][]schema.Document{
		{{ID: "a", Score: 1.0}},
		{{ID: "b", Score: 1.0}},
	}

	ws := NewWeightedStrategy([]float64{0.0, 0.0})
	_, err := ws.Fuse(context.Background(), sets)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "weights sum to zero")
}

func TestWeightedStrategy_Fuse_NormalizesWeights(t *testing.T) {
	// Weights that don't sum to 1 should be normalized
	sets := [][]schema.Document{
		{{ID: "a", Score: 1.0}},
		{{ID: "b", Score: 1.0}},
	}

	ws := NewWeightedStrategy([]float64{2.0, 2.0}) // Sum = 4.0
	fused, err := ws.Fuse(context.Background(), sets)
	require.NoError(t, err)
	assert.Len(t, fused, 2)

	// After normalization: each weight becomes 0.5
	// a: 1.0 * 0.5 = 0.5
	// b: 1.0 * 0.5 = 0.5
	assert.InDelta(t, 0.5, fused[0].Score, 0.01)
	assert.InDelta(t, 0.5, fused[1].Score, 0.01)
}

func TestWeightedStrategy_Fuse_EmptyResults(t *testing.T) {
	ws := NewWeightedStrategy([]float64{0.5, 0.5})
	fused, err := ws.Fuse(context.Background(), [][]schema.Document{{}, {}})
	require.NoError(t, err)
	assert.Empty(t, fused)
}

// --- Tests for EnsembleRetriever ---

func TestEnsembleRetriever_RRF_Detailed(t *testing.T) {
	r1 := &mockRetriever{docs: makeDocs("a", "b", "c")}
	r2 := &mockRetriever{docs: makeDocs("b", "d", "a")}

	ensemble := NewEnsembleRetriever([]Retriever{r1, r2}, NewRRFStrategy(60))
	docs, err := ensemble.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 4, "should have all unique docs")

	// a and b appear in both, so should have higher scores
	assert.Greater(t, docs[0].Score, 0.0)
}

func TestEnsembleRetriever_Weighted_Detailed(t *testing.T) {
	r1 := &mockRetriever{docs: []schema.Document{
		{ID: "a", Score: 0.9},
		{ID: "b", Score: 0.5},
	}}
	r2 := &mockRetriever{docs: []schema.Document{
		{ID: "b", Score: 0.8},
		{ID: "c", Score: 0.7},
	}}

	ensemble := NewEnsembleRetriever(
		[]Retriever{r1, r2},
		NewWeightedStrategy([]float64{0.6, 0.4}),
	)
	docs, err := ensemble.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 3)
	assert.Greater(t, docs[0].Score, 0.0)
}

func TestEnsembleRetriever_WithTopK(t *testing.T) {
	r1 := &mockRetriever{docs: makeDocs("a", "b", "c", "d", "e")}
	r2 := &mockRetriever{docs: makeDocs("f", "g", "h", "i", "j")}

	ensemble := NewEnsembleRetriever([]Retriever{r1, r2}, NewRRFStrategy(60))
	docs, err := ensemble.Retrieve(context.Background(), "query", WithTopK(3))
	require.NoError(t, err)
	assert.Len(t, docs, 3, "should limit to TopK")
}

func TestEnsembleRetriever_InnerRetrieverError(t *testing.T) {
	r1 := &mockRetriever{docs: makeDocs("a")}
	r2 := &mockRetriever{err: errors.New("retrieval failed")}

	ensemble := NewEnsembleRetriever([]Retriever{r1, r2}, nil)
	_, err := ensemble.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ensemble retriever 1")
}

func TestEnsembleRetriever_FusionError(t *testing.T) {
	// WeightedStrategy will fail with mismatched weights
	r1 := &mockRetriever{docs: makeDocs("a")}
	r2 := &mockRetriever{docs: makeDocs("b")}

	ensemble := NewEnsembleRetriever(
		[]Retriever{r1, r2},
		NewWeightedStrategy([]float64{1.0}), // Only 1 weight for 2 retrievers
	)
	_, err := ensemble.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ensemble fuse")
}

func TestEnsembleRetriever_NilStrategy(t *testing.T) {
	// Should default to RRF
	r := &mockRetriever{docs: makeDocs("a", "b")}
	ensemble := NewEnsembleRetriever([]Retriever{r}, nil)

	docs, err := ensemble.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestEnsembleRetriever_WithHooks(t *testing.T) {
	r1 := &mockRetriever{docs: makeDocs("a")}
	r2 := &mockRetriever{docs: makeDocs("b")}

	var beforeCalled, afterCalled bool

	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			beforeCalled = true
			return nil
		},
		AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
			afterCalled = true
		},
	}

	ensemble := NewEnsembleRetriever(
		[]Retriever{r1, r2},
		NewRRFStrategy(60),
		WithEnsembleHooks(hooks),
	)

	_, err := ensemble.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}

func TestEnsembleRetriever_BeforeHookError(t *testing.T) {
	r := &mockRetriever{docs: makeDocs("a")}
	hookErr := errors.New("hook rejected")

	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return hookErr
		},
	}

	ensemble := NewEnsembleRetriever(
		[]Retriever{r},
		nil,
		WithEnsembleHooks(hooks),
	)

	_, err := ensemble.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Equal(t, hookErr, err)
}

func TestEnsembleRetriever_EmptyRetrievers(t *testing.T) {
	ensemble := NewEnsembleRetriever([]Retriever{}, NewRRFStrategy(60))
	docs, err := ensemble.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestEnsembleRetriever_SingleRetriever(t *testing.T) {
	r := &mockRetriever{docs: makeDocs("a", "b", "c")}
	ensemble := NewEnsembleRetriever([]Retriever{r}, NewRRFStrategy(60))

	docs, err := ensemble.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 3)
}

// Compile-time interface checks
var (
	_ FusionStrategy = (*RRFStrategy)(nil)
	_ FusionStrategy = (*WeightedStrategy)(nil)
	_ Retriever      = (*EnsembleRetriever)(nil)
)
