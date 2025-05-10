package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

func init() {
	RegisterEmbedderProvider(ProviderMock, func(appConfig *config.ViperProvider) (iface.Embedder, error) {
		var mockCfg config.MockEmbedderConfig

		// Populate mockCfg field by field using ViperProvider methods
		// NewMockEmbedder handles defaulting for Dimension if it's 0.
		mockCfg.Dimension = appConfig.GetInt("embeddings.mock.dimension")
		mockCfg.Seed = int64(appConfig.GetInt("embeddings.mock.seed")) // Viper GetInt returns int
		mockCfg.RandomizeNil = appConfig.GetBool("embeddings.mock.randomize_nil")

		// If specific keys are absolutely required and not defaulted by NewMockEmbedder,
		// you might add checks here using appConfig.IsSet(), for example:
		// if !appConfig.IsSet("embeddings.mock.dimension") {
		// 	 fmt.Println("Warning: embeddings.mock.dimension not explicitly set, relying on NewMockEmbedder defaults or zero value.")
		// }

		return NewMockEmbedder(mockCfg)
	})
}

// MockEmbedder is an implementation of Embedder that returns mock embeddings.
// Useful for testing and development purposes.
type MockEmbedder struct {
	config config.MockEmbedderConfig
	rand   *rand.Rand
}

// NewMockEmbedder creates a new MockEmbedder with the given configuration.
func NewMockEmbedder(cfg config.MockEmbedderConfig) (*MockEmbedder, error) {
	if cfg.Dimension < 0 {
		return nil, fmt.Errorf("dimension must be non-negative, got %d", cfg.Dimension)
	}
	// Default dimension if not set or set to 0 by config
	if cfg.Dimension == 0 {
		fmt.Println("Warning: MockEmbedder.Dimension is 0 from config, defaulting to 128.")
		cfg.Dimension = 128
	}

	return &MockEmbedder{
		config: cfg,
		rand:   rand.New(rand.NewSource(cfg.Seed)),
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
func (m *MockEmbedder) generateMockEmbedding(text string) []float32 {
	embedding := make([]float32, m.config.Dimension)

	h := sha256.New()
	h.Write([]byte(text))
	hashBytes := h.Sum(nil)

	var seed int64
	if len(hashBytes) >= 8 {
		seed = int64(binary.BigEndian.Uint64(hashBytes[:8]))
	} else {
		var tempBytes [8]byte
		copy(tempBytes[:], hashBytes)
		seed = int64(binary.BigEndian.Uint64(tempBytes[:]))
	}

	localRand := rand.New(rand.NewSource(m.config.Seed + seed))

	for i := 0; i < m.config.Dimension; i++ {
		embedding[i] = localRand.Float32()*2 - 1
	}
	return embedding
}

var _ iface.Embedder = (*MockEmbedder)(nil)

