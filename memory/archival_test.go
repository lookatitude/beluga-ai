package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbedder is a test double for embedding.Embedder.
type mockEmbedder struct {
	dim         int
	embedErr    error
	embedSingle error
}

func (m *mockEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, m.dim)
		// Fill with incrementing values for test verification.
		for j := 0; j < m.dim; j++ {
			result[i][j] = float32(i*m.dim + j)
		}
	}
	return result, nil
}

func (m *mockEmbedder) EmbedSingle(_ context.Context, text string) ([]float32, error) {
	if m.embedSingle != nil {
		return nil, m.embedSingle
	}
	vec := make([]float32, m.dim)
	for i := 0; i < m.dim; i++ {
		vec[i] = float32(i)
	}
	return vec, nil
}

func (m *mockEmbedder) Dimensions() int {
	return m.dim
}

// mockVectorStore is a test double for vectorstore.VectorStore.
type mockVectorStore struct {
	docs      []schema.Document
	addErr    error
	searchErr error
}

func (m *mockVectorStore) Add(_ context.Context, docs []schema.Document, embeddings [][]float32) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.docs = append(m.docs, docs...)
	return nil
}

func (m *mockVectorStore) Search(_ context.Context, vec []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if k > len(m.docs) {
		k = len(m.docs)
	}
	return m.docs[:k], nil
}

func (m *mockVectorStore) Delete(_ context.Context, ids []string) error {
	return nil
}

func TestNewArchival(t *testing.T) {
	tests := []struct {
		name      string
		cfg       ArchivalConfig
		wantError string
	}{
		{
			name: "valid config",
			cfg: ArchivalConfig{
				VectorStore: &mockVectorStore{},
				Embedder:    &mockEmbedder{dim: 128},
			},
		},
		{
			name: "nil vector store",
			cfg: ArchivalConfig{
				Embedder: &mockEmbedder{dim: 128},
			},
			wantError: "VectorStore is required",
		},
		{
			name: "nil embedder",
			cfg: ArchivalConfig{
				VectorStore: &mockVectorStore{},
			},
			wantError: "Embedder is required",
		},
		{
			name:      "both nil",
			cfg:       ArchivalConfig{},
			wantError: "VectorStore is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arch, err := NewArchival(tt.cfg)
			if tt.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
				assert.Nil(t, arch)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, arch)
			}
		})
	}
}

func TestArchivalSave(t *testing.T) {
	ctx := context.Background()

	t.Run("saves and embeds messages", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    emb,
		})
		require.NoError(t, err)

		input := schema.NewHumanMessage("hello world")
		output := schema.NewAIMessage("hi there")

		err = arch.Save(ctx, input, output)
		require.NoError(t, err)

		// Verify both messages were stored.
		assert.Len(t, vs.docs, 2)
		assert.Equal(t, "archival-1", vs.docs[0].ID)
		assert.Equal(t, "hello world", vs.docs[0].Content)
		assert.Equal(t, "human", vs.docs[0].Metadata["role"])

		assert.Equal(t, "archival-2", vs.docs[1].ID)
		assert.Equal(t, "hi there", vs.docs[1].Content)
		assert.Equal(t, "ai", vs.docs[1].Metadata["role"])
	})

	t.Run("skips empty messages", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    emb,
		})
		require.NoError(t, err)

		// Messages with no text content.
		input := &schema.HumanMessage{}
		output := schema.NewAIMessage("hi")

		err = arch.Save(ctx, input, output)
		require.NoError(t, err)

		// Only the non-empty message was stored.
		assert.Len(t, vs.docs, 1)
		assert.Equal(t, "hi", vs.docs[0].Content)
	})

	t.Run("embed error", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4, embedErr: errors.New("embed failed")}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    emb,
		})
		require.NoError(t, err)

		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err = arch.Save(ctx, input, output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embed")
	})

	t.Run("vector store add error", func(t *testing.T) {
		vs := &mockVectorStore{addErr: errors.New("add failed")}
		emb := &mockEmbedder{dim: 4}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    emb,
		})
		require.NoError(t, err)

		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err = arch.Save(ctx, input, output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "add failed")
	})
}

func TestArchivalLoad(t *testing.T) {
	// Archival does not support Load (returns nil).
	ctx := context.Background()
	vs := &mockVectorStore{}
	emb := &mockEmbedder{dim: 4}

	arch, err := NewArchival(ArchivalConfig{
		VectorStore: vs,
		Embedder:    emb,
	})
	require.NoError(t, err)

	msgs, err := arch.Load(ctx, "any query")
	require.NoError(t, err)
	assert.Nil(t, msgs)
}

func TestArchivalSearch(t *testing.T) {
	ctx := context.Background()

	t.Run("search returns documents", func(t *testing.T) {
		doc1 := schema.Document{ID: "1", Content: "hello"}
		doc2 := schema.Document{ID: "2", Content: "world"}
		vs := &mockVectorStore{docs: []schema.Document{doc1, doc2}}
		emb := &mockEmbedder{dim: 4}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    emb,
		})
		require.NoError(t, err)

		docs, err := arch.Search(ctx, "test query", 2)
		require.NoError(t, err)
		assert.Len(t, docs, 2)
		assert.Equal(t, "1", docs[0].ID)
		assert.Equal(t, "2", docs[1].ID)
	})

	t.Run("default k value", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    emb,
		})
		require.NoError(t, err)

		// k <= 0 defaults to 10.
		_, err = arch.Search(ctx, "query", 0)
		require.NoError(t, err)

		_, err = arch.Search(ctx, "query", -5)
		require.NoError(t, err)
	})

	t.Run("embed error", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4, embedSingle: errors.New("embed failed")}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    emb,
		})
		require.NoError(t, err)

		docs, err := arch.Search(ctx, "query", 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embed query")
		assert.Nil(t, docs)
	})

	t.Run("search error", func(t *testing.T) {
		vs := &mockVectorStore{searchErr: errors.New("search failed")}
		emb := &mockEmbedder{dim: 4}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    emb,
		})
		require.NoError(t, err)

		docs, err := arch.Search(ctx, "query", 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "search failed")
		assert.Nil(t, docs)
	})
}

func TestArchivalClear(t *testing.T) {
	// Clear is a no-op for archival memory.
	ctx := context.Background()
	vs := &mockVectorStore{}
	emb := &mockEmbedder{dim: 4}

	arch, err := NewArchival(ArchivalConfig{
		VectorStore: vs,
		Embedder:    emb,
	})
	require.NoError(t, err)

	err = arch.Clear(ctx)
	assert.NoError(t, err)
}

func TestArchivalRegistryEntry(t *testing.T) {
	// The archival registry entry returns an error because it requires
	// VectorStore and Embedder to be provided via NewArchival directly.
	mem, err := New("archival", config.ProviderConfig{Provider: "archival"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use NewArchival directly")
	assert.Nil(t, mem)
}

func TestArchivalSequenceID(t *testing.T) {
	// Verify that Save increments the sequence ID for each document.
	ctx := context.Background()
	vs := &mockVectorStore{}
	emb := &mockEmbedder{dim: 4}

	arch, err := NewArchival(ArchivalConfig{
		VectorStore: vs,
		Embedder:    emb,
	})
	require.NoError(t, err)

	// Save multiple times.
	for i := 1; i <= 3; i++ {
		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")
		err = arch.Save(ctx, input, output)
		require.NoError(t, err)
	}

	// Verify IDs are sequential.
	assert.Len(t, vs.docs, 6)
	for i := 0; i < 6; i++ {
		expectedID := "archival-" + string(rune('1'+i))
		assert.Equal(t, expectedID, vs.docs[i].ID)
	}
}
