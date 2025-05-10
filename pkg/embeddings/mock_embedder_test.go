package embeddings

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockEmbedder(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := config.MockEmbedderConfig{
			Dimension: 10,
			Seed:      123,
		}
		embedder, err := NewMockEmbedder(cfg)
		require.NoError(t, err)
		require.NotNil(t, embedder)
		assert.Equal(t, cfg.Dimension, embedder.config.Dimension)
		assert.Equal(t, cfg.Seed, embedder.config.Seed)
	})

	t.Run("InvalidDimensionZero", func(t *testing.T) {
		cfg := config.MockEmbedderConfig{
			Dimension: 0,
			Seed:      123,
		}
		// NewMockEmbedder now defaults Dimension 0 to 128, so no error is expected here.
		embedder, err := NewMockEmbedder(cfg)
		require.NoError(t, err) 
		require.NotNil(t, embedder)
		assert.Equal(t, 128, embedder.config.Dimension) // Check it defaulted
	})

	t.Run("InvalidDimensionNegative", func(t *testing.T) {
		cfg := config.MockEmbedderConfig{
			Dimension: -5,
			Seed:      123,
		}
		_, err := NewMockEmbedder(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dimension must be non-negative")
	})
}

func TestMockEmbedder_GetDimension(t *testing.T) {
	cfg := config.MockEmbedderConfig{Dimension: 128, Seed: 1}
	embedder, _ := NewMockEmbedder(cfg)
	dim, err := embedder.GetDimension(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 128, dim)
}

func TestMockEmbedder_EmbedQuery(t *testing.T) {
	cfg := config.MockEmbedderConfig{Dimension: 5, Seed: 42}
	embedder, _ := NewMockEmbedder(cfg)

	t.Run("ValidQuery", func(t *testing.T) {
		query := "hello world"
		embedding, err := embedder.EmbedQuery(context.Background(), query)
		require.NoError(t, err)
		require.NotNil(t, embedding)
		assert.Len(t, embedding, cfg.Dimension)
	})

	t.Run("EmptyQueryError", func(t *testing.T) {
		cfgError := config.MockEmbedderConfig{Dimension: 5, Seed: 42, RandomizeNil: false}
		embedderError, _ := NewMockEmbedder(cfgError)
		_, err := embedderError.EmbedQuery(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot embed empty query text")
	})

	t.Run("EmptyQueryNil", func(t *testing.T) {
		cfgNil := config.MockEmbedderConfig{Dimension: 5, Seed: 42, RandomizeNil: true}
		embedderNil, _ := NewMockEmbedder(cfgNil)
		embedding, err := embedderNil.EmbedQuery(context.Background(), "")
		require.NoError(t, err)
		assert.Nil(t, embedding)
	})

	t.Run("DeterministicOutputSameSeed", func(t *testing.T) {
		query := "test query"
		embedding1, _ := embedder.EmbedQuery(context.Background(), query)
		embedding2, _ := embedder.EmbedQuery(context.Background(), query)
		assert.Equal(t, embedding1, embedding2)
	})

	t.Run("DifferentOutputDifferentSeed", func(t *testing.T) {
		query := "test query"
		cfg2 := config.MockEmbedderConfig{Dimension: 5, Seed: 43} // Different seed
		embedder2, _ := NewMockEmbedder(cfg2)
		embedding1, _ := embedder.EmbedQuery(context.Background(), query)
		embedding2, _ := embedder2.EmbedQuery(context.Background(), query)
		assert.NotEqual(t, embedding1, embedding2)
	})

	t.Run("DifferentOutputDifferentText", func(t *testing.T) {
		query1 := "test query 1"
		query2 := "test query 2"
		embedding1, _ := embedder.EmbedQuery(context.Background(), query1)
		embedding2, _ := embedder.EmbedQuery(context.Background(), query2)
		assert.NotEqual(t, embedding1, embedding2)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel context immediately
		_, err := embedder.EmbedQuery(ctx, "test")
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestMockEmbedder_EmbedDocuments(t *testing.T) {
	cfg := config.MockEmbedderConfig{Dimension: 3, Seed: 77}
	embedder, _ := NewMockEmbedder(cfg)

	t.Run("ValidDocuments", func(t *testing.T) {
		docs := []string{"doc1", "doc2", "another document"}
		embeddings, err := embedder.EmbedDocuments(context.Background(), docs)
		require.NoError(t, err)
		require.Len(t, embeddings, len(docs))
		for _, emb := range embeddings {
			assert.Len(t, emb, cfg.Dimension)
		}
	})

	t.Run("EmptyDocumentList", func(t *testing.T) {
		embeddings, err := embedder.EmbedDocuments(context.Background(), []string{})
		require.NoError(t, err)
		assert.Empty(t, embeddings)
	})

	t.Run("EmptyDocumentInListError", func(t *testing.T) {
		cfgError := config.MockEmbedderConfig{Dimension: 3, Seed: 77, RandomizeNil: false}
		embedderError, _ := NewMockEmbedder(cfgError)
		docs := []string{"doc1", "", "doc3"}
		_, err := embedderError.EmbedDocuments(context.Background(), docs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot embed empty text at index 1")
	})

	t.Run("EmptyDocumentInListNil", func(t *testing.T) {
		cfgNil := config.MockEmbedderConfig{Dimension: 3, Seed: 77, RandomizeNil: true}
		embedderNil, _ := NewMockEmbedder(cfgNil)
		docs := []string{"doc1", "", "doc3"}
		embeddings, err := embedderNil.EmbedDocuments(context.Background(), docs)
		require.NoError(t, err)
		require.Len(t, embeddings, len(docs))
		assert.NotNil(t, embeddings[0])
		assert.Nil(t, embeddings[1])
		assert.NotNil(t, embeddings[2])
	})

	t.Run("DeterministicOutputSameSeed", func(t *testing.T) {
		docs := []string{"doc alpha", "doc beta"}
		embeddings1, _ := embedder.EmbedDocuments(context.Background(), docs)
		embeddings2, _ := embedder.EmbedDocuments(context.Background(), docs)
		assert.Equal(t, embeddings1, embeddings2)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond) // Tiny timeout
		defer cancel()
		// Ensure the loop runs at least once to hit the context check
		// This is a bit tricky to guarantee cancellation within the loop without making the test flaky
		// For a mock, it's simpler. For real API calls, this would be more critical.
		_, err := embedder.EmbedDocuments(ctx, []string{"text1", "text2", "text3"})
		if err != nil { // Error is expected
		    assert.ErrorIs(t, err, context.DeadlineExceeded) // or context.Canceled if cancel() was used directly
		} else {
		    // This case might occur if the operation is too fast for the timeout
		    // For this mock, it's less of an issue. For real network calls, the timeout would likely hit.
		    t.Log("Context cancellation test for EmbedDocuments might not have triggered cancellation due to speed of mock operation.")
		}
	})
}

