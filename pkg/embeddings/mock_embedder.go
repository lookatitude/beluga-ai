package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
)

// MockEmbedderConfig holds configuration for the MockEmbedder.
type MockEmbedderConfig struct {
	Dimension    int
	Seed         int64 // Seed for deterministic random number generation
	RandomizeNil bool  // If true, returns nil for empty texts, otherwise error
}

// MockEmbedder is an implementation of Embedder that returns mock embeddings.
// Useful for testing and development purposes.
type MockEmbedder struct {
	config MockEmbedderConfig
	rand   *rand.Rand
}

// NewMockEmbedder creates a new MockEmbedder with the given configuration.
func NewMockEmbedder(config MockEmbedderConfig) (*MockEmbedder, error) {
	if config.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive, got %d", config.Dimension)
	}
	return &MockEmbedder{
		config: config,
		rand:   rand.New(rand.NewSource(config.Seed)),
	}, nil
}

// EmbedDocuments generates mock embeddings for a batch of documents.
func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if text == "" {
			if m.config.RandomizeNil {
				embeddings[i] = nil
				continue
			}
			return nil, fmt.Errorf("cannot embed empty text at index %d", i)
		}
		embeddings[i] = m.generateMockEmbedding(text)
	}
	return embeddings, nil
}

// EmbedQuery generates a mock embedding for a single query text.
func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if text == "" {
		if m.config.RandomizeNil {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot embed empty query text")
	}
	return m.generateMockEmbedding(text), nil
}

// GetDimension returns the configured dimensionality of the mock embeddings.
func (m *MockEmbedder) GetDimension(_ context.Context) (int, error) {
	return m.config.Dimension, nil
}

// generateMockEmbedding creates a deterministic embedding based on the text content and seed.
// This ensures that the same text always produces the same mock embedding for a given seed.
func (m *MockEmbedder) generateMockEmbedding(text string) []float32 {
	embedding := make([]float32, m.config.Dimension)
	
	// Use a hash of the text to seed a local random generator for this specific text
	// This makes the embedding deterministic based on text content, but still pseudo-random looking.
	h := sha256.New()
	h.Write([]byte(text))
	hashBytes := h.Sum(nil)

	var seed int64
	if len(hashBytes) >= 8 {
		seed = int64(binary.BigEndian.Uint64(hashBytes[:8]))
	} else {
		// Fallback for very short hashes (should not happen with SHA256)
		var tempBytes [8]byte
		copy(tempBytes[:], hashBytes)
		seed = int64(binary.BigEndian.Uint64(tempBytes[:]))
	}

	// Add the global seed to ensure different MockEmbedder instances can produce different results for the same text if desired.
	localRand := rand.New(rand.NewSource(m.config.Seed + seed))

	for i := 0; i < m.config.Dimension; i++ {
		// Generate a float between -1.0 and 1.0
		embedding[i] = localRand.Float32()*2 - 1
	}
	return embedding
}

// Ensure MockEmbedder implements the Embedder interface.
var _ Embedder = (*MockEmbedder)(nil)

