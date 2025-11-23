package embeddings

import (
	"github.com/stretchr/testify/mock"
)

// EmbeddingsMockcomponent is a mock implementation of Interface
type EmbeddingsMockcomponent struct {
	mock.Mock
}

// NewEmbeddingsMockcomponent creates a new EmbeddingsMockcomponent
func NewEmbeddingsMockcomponent() *EmbeddingsMockcomponent {
	return &EmbeddingsMockcomponent{}
}
