package inmemory

import (
	"context"
	"math"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_DefaultDimensions(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)
	require.NotNil(t, emb)

	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestNew_CustomDimensions(t *testing.T) {
	cfg := config.ProviderConfig{
		Options: map[string]any{
			"dimensions": float64(256),
		},
	}

	emb, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, emb)

	assert.Equal(t, 256, emb.Dimensions())
}

func TestNew_InvalidDimensions(t *testing.T) {
	// Zero or negative dimensions should fall back to default.
	cfg := config.ProviderConfig{
		Options: map[string]any{
			"dimensions": float64(0),
		},
	}

	emb, err := New(cfg)
	require.NoError(t, err)
	assert.Equal(t, defaultDimensions, emb.Dimensions())
}

func TestEmbedder_Embed(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	texts := []string{"hello", "world", "test"}
	embeddings, err := emb.Embed(context.Background(), texts)
	require.NoError(t, err)

	require.Len(t, embeddings, 3)
	for i, vec := range embeddings {
		assert.Len(t, vec, defaultDimensions, "embedding %d has wrong dimension", i)
		assertNormalized(t, vec, "embedding %d should be normalized", i)
	}
}

func TestEmbedder_EmbedSingle(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	vec, err := emb.EmbedSingle(context.Background(), "hello world")
	require.NoError(t, err)

	assert.Len(t, vec, defaultDimensions)
	assertNormalized(t, vec, "single embedding should be normalized")
}

func TestEmbedder_Deterministic(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	text := "deterministic test"

	// Embed the same text multiple times.
	vec1, err := emb.EmbedSingle(context.Background(), text)
	require.NoError(t, err)

	vec2, err := emb.EmbedSingle(context.Background(), text)
	require.NoError(t, err)

	vec3, err := emb.EmbedSingle(context.Background(), text)
	require.NoError(t, err)

	// All embeddings should be identical.
	assert.Equal(t, vec1, vec2, "embeddings should be deterministic")
	assert.Equal(t, vec2, vec3, "embeddings should be deterministic")
}

func TestEmbedder_DifferentTexts(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	vec1, err := emb.EmbedSingle(context.Background(), "hello")
	require.NoError(t, err)

	vec2, err := emb.EmbedSingle(context.Background(), "world")
	require.NoError(t, err)

	// Different texts should produce different embeddings.
	assert.NotEqual(t, vec1, vec2, "different texts should have different embeddings")
}

func TestEmbedder_EmptyText(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	vec, err := emb.EmbedSingle(context.Background(), "")
	require.NoError(t, err)

	assert.Len(t, vec, defaultDimensions)
	assertNormalized(t, vec, "empty text embedding should be normalized")
}

func TestEmbedder_Embed_EmptyList(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	embeddings, err := emb.Embed(context.Background(), []string{})
	require.NoError(t, err)
	assert.Len(t, embeddings, 0)
}

func TestEmbedder_Embed_SingleItem(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	embeddings, err := emb.Embed(context.Background(), []string{"single"})
	require.NoError(t, err)

	require.Len(t, embeddings, 1)
	assert.Len(t, embeddings[0], defaultDimensions)
	assertNormalized(t, embeddings[0], "single item embedding should be normalized")
}

func TestEmbedder_Dimensions(t *testing.T) {
	tests := []struct {
		name string
		dims int
	}{
		{"default", defaultDimensions},
		{"small", 64},
		{"large", 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ProviderConfig{}
			if tt.dims != defaultDimensions {
				cfg.Options = map[string]any{
					"dimensions": float64(tt.dims),
				}
			}

			emb, err := New(cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.dims, emb.Dimensions())

			vec, err := emb.EmbedSingle(context.Background(), "test")
			require.NoError(t, err)
			assert.Len(t, vec, tt.dims)
		})
	}
}

func TestEmbedder_InterfaceCompliance(t *testing.T) {
	// Compile-time check that Embedder implements embedding.Embedder.
	var _ embedding.Embedder = (*Embedder)(nil)
}

func TestRegistry_Integration(t *testing.T) {
	// Test that the provider is registered.
	emb, err := embedding.New("inmemory", config.ProviderConfig{})
	require.NoError(t, err)
	require.NotNil(t, emb)

	assert.Equal(t, defaultDimensions, emb.Dimensions())

	vec, err := emb.EmbedSingle(context.Background(), "registry test")
	require.NoError(t, err)
	assert.Len(t, vec, defaultDimensions)
}

func TestEmbedder_ConsistencyAcrossBatches(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	text := "consistency test"

	// Get embedding via batch.
	batch, err := emb.Embed(context.Background(), []string{text})
	require.NoError(t, err)
	require.Len(t, batch, 1)
	batchVec := batch[0]

	// Get embedding via single.
	singleVec, err := emb.EmbedSingle(context.Background(), text)
	require.NoError(t, err)

	// They should be identical.
	assert.Equal(t, batchVec, singleVec, "batch and single embeddings should match")
}

func TestEmbedder_VectorProperties(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	vec, err := emb.EmbedSingle(context.Background(), "test vector properties")
	require.NoError(t, err)

	// Check that all values are in reasonable range [-1, 1].
	for i, v := range vec {
		assert.GreaterOrEqual(t, v, float32(-1.0), "value at index %d out of range", i)
		assert.LessOrEqual(t, v, float32(1.0), "value at index %d out of range", i)
	}

	// Check that not all values are zero.
	hasNonZero := false
	for _, v := range vec {
		if v != 0 {
			hasNonZero = true
			break
		}
	}
	assert.True(t, hasNonZero, "vector should have at least one non-zero value")
}

// assertNormalized checks that a vector has unit length (L2 norm = 1).
func assertNormalized(t *testing.T, vec []float32, msgAndArgs ...interface{}) {
	t.Helper()

	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)

	assert.InDelta(t, 1.0, norm, 0.0001, msgAndArgs...)
}

func TestEmbedder_LargeText(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	// Create a large text (10KB).
	largeText := ""
	for i := 0; i < 1000; i++ {
		largeText += "large text "
	}

	vec, err := emb.EmbedSingle(context.Background(), largeText)
	require.NoError(t, err)

	assert.Len(t, vec, defaultDimensions)
	assertNormalized(t, vec, "large text embedding should be normalized")
}

func TestEmbedder_SpecialCharacters(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	texts := []string{
		"hello!@#$%^&*()",
		"unicode: \u4e2d\u6587",
		"newline\ntest",
		"tab\ttest",
		"",
	}

	embeddings, err := emb.Embed(context.Background(), texts)
	require.NoError(t, err)
	require.Len(t, embeddings, len(texts))

	for i, vec := range embeddings {
		assert.Len(t, vec, defaultDimensions, "embedding %d has wrong dimension", i)
		assertNormalized(t, vec, "embedding %d should be normalized", i)
	}
}

func TestEmbedder_ContextCancellation(t *testing.T) {
	emb, err := New(config.ProviderConfig{})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	// The in-memory embedder doesn't actually check context, but this tests the interface.
	vec, err := emb.EmbedSingle(ctx, "test")
	// Should still work since it's synchronous and fast.
	require.NoError(t, err)
	assert.Len(t, vec, defaultDimensions)
}
