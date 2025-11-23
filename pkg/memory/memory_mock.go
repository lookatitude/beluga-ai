package memory

import (
	"github.com/stretchr/testify/mock"
)

// MemoryMockcomponent is a mock implementation of Interface
type MemoryMockcomponent struct {
	mock.Mock
}

// NewMemoryMockcomponent creates a new MemoryMockcomponent
func NewMemoryMockcomponent() *MemoryMockcomponent {
	return &MemoryMockcomponent{}
}
