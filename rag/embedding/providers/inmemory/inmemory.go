// Package inmemory provides a deterministic hash-based Embedder for testing.
// It generates reproducible embeddings by hashing the input text, making it
// suitable for unit tests and local development without external API calls.
//
// Registration:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/inmemory"
//
//	emb, err := embedding.New("inmemory", config.ProviderConfig{})
package inmemory

import (
	"context"
	"encoding/binary"
	"hash/fnv"
	"math"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
)

func init() {
	embedding.Register("inmemory", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
		return New(cfg)
	})
}

const defaultDimensions = 128

// Embedder is a deterministic hash-based embedder for testing. It produces
// reproducible float32 vectors by hashing input text with FNV-1a.
type Embedder struct {
	dims int
}

// Compile-time interface check.
var _ embedding.Embedder = (*Embedder)(nil)

// New creates a new in-memory Embedder. The "dimensions" option in
// cfg.Options controls the vector size (default 128).
func New(cfg config.ProviderConfig) (*Embedder, error) {
	dims := defaultDimensions
	if d, ok := config.GetOption[float64](cfg, "dimensions"); ok && d > 0 {
		dims = int(d)
	}
	return &Embedder{dims: dims}, nil
}

// Embed produces deterministic embeddings for a batch of texts.
func (e *Embedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		result[i] = e.hashToVector(text)
	}
	return result, nil
}

// EmbedSingle produces a deterministic embedding for a single text.
func (e *Embedder) EmbedSingle(_ context.Context, text string) ([]float32, error) {
	return e.hashToVector(text), nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *Embedder) Dimensions() int {
	return e.dims
}

// hashToVector creates a deterministic float32 vector from text using FNV-1a.
// The vector is normalized to unit length.
func (e *Embedder) hashToVector(text string) []float32 {
	vec := make([]float32, e.dims)

	h := fnv.New64a()
	h.Write([]byte(text))
	seed := h.Sum64()

	// Use the seed to generate deterministic float values.
	for i := range vec {
		// Mix the seed with the dimension index for each component.
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, seed^uint64(i)*2654435761)
		h2 := fnv.New32a()
		h2.Write(b)
		bits := h2.Sum32()
		// Map to [-1, 1] range.
		vec[i] = float32(bits)/float32(math.MaxUint32)*2 - 1
	}

	// Normalize to unit length.
	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec
}
