package vectorstores

import (
	"github.com/stretchr/testify/mock"
)

// VectorStoresMockcomponent is a mock implementation of Interface
type VectorStoresMockcomponent struct {
	mock.Mock
}

// NewVectorStoresMockcomponent creates a new VectorStoresMockcomponent
func NewVectorStoresMockcomponent() *VectorStoresMockcomponent {
	return &VectorStoresMockcomponent{}
}
