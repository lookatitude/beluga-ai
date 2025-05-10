package embeddings

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/registry"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

const (
	MockProviderName = "mock"
)

func init() {
	registry.RegisterEmbedder(MockProviderName, func(generalConfig config.Provider, instanceConfig schema.EmbeddingProviderConfig) (iface.Embedder, error) {
		dimension := 128 // Default
		seed := int64(0)    // Default
		randomizeNil := false // Default

		if instanceConfig.ProviderSpecific != nil {
			if dimVal, ok := instanceConfig.ProviderSpecific["dimension"]; ok {
				if iDim, isInt := dimVal.(int); isInt {
					dimension = iDim
				} else if fDim, isFloat := dimVal.(float64); isFloat {
					dimension = int(fDim)
				} else {
					fmt.Printf("MockEmbedderCreator: Could not parse dimension 	%v	, using default 	%d	\n", dimVal, dimension)
				}
			}
			if seedVal, ok := instanceConfig.ProviderSpecific["seed"]; ok {
				if iSeed, isInt64 := seedVal.(int64); isInt64 {
					seed = iSeed
				} else if fSeed, isFloat64 := seedVal.(float64); isFloat64 {
					seed = int64(fSeed)
				} else if iSeed, isInt := seedVal.(int); isInt {
					seed = int64(iSeed)
				} else {
					fmt.Printf("MockEmbedderCreator: Could not parse seed 	%v	, using default 	%d	\n", seedVal, seed)
				}
			}
			if randNilVal, ok := instanceConfig.ProviderSpecific["randomize_nil"]; ok {
				if bRandNil, isBool := randNilVal.(bool); isBool {
					randomizeNil = bRandNil
				} else {
					fmt.Printf("MockEmbedderCreator: Could not parse randomize_nil 	%v	, using default 	%t	\n", randNilVal, randomizeNil)
				}
			}
		}
		fmt.Printf("MockEmbedderCreator: Creating MockEmbedder with Dimension=	%d	, Seed=	%d	, RandomizeNil=	%t	 for instance 	%s	\n", dimension, seed, randomizeNil, instanceConfig.Name)
		return NewMockEmbedder(dimension, seed, randomizeNil), nil
	})
}

// MockEmbedder is a mock implementation of the Embedder interface for testing.
type MockEmbedder struct {
	DimensionValue int
	SeedValue      int64
	RandomizeNil   bool
	mu             sync.Mutex
	rng            *rand.Rand
}

// NewMockEmbedder creates a new MockEmbedder.
func NewMockEmbedder(dimension int, seed int64, randomizeNil bool) *MockEmbedder {
	if dimension == 0 {
		dimension = 128 // Default dimension
	}
	src := rand.NewSource(seed)
	return &MockEmbedder{
		DimensionValue: dimension,
		SeedValue:      seed,
		RandomizeNil:   randomizeNil,
		rng:            rand.New(src),
	}
}

// EmbedDocuments mocks embedding multiple documents.
func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		if texts[i] == "" && !m.RandomizeNil {
			embeddings[i] = make([]float32, m.DimensionValue) // Zero vector for empty string
		} else {
			embedding := make([]float32, m.DimensionValue)
			for j := 0; j < m.DimensionValue; j++ {
				embedding[j] = m.rng.Float32()
			}
			embeddings[i] = embedding
		}
	}
	return embeddings, nil
}

// EmbedQuery mocks embedding a single query.
func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if text == "" && !m.RandomizeNil {
		return make([]float32, m.DimensionValue), nil // Zero vector
	}
	embedding := make([]float32, m.DimensionValue)
	for i := 0; i < m.DimensionValue; i++ {
		embedding[i] = m.rng.Float32()
	}
	return embedding, nil
}

// GetDimension returns the mock dimension.
func (m *MockEmbedder) GetDimension(ctx context.Context) (int, error) {
	return m.DimensionValue, nil
}

var _ iface.Embedder = (*MockEmbedder)(nil)
