package orchestration

import (
	"github.com/stretchr/testify/mock"
)

// AdvancedMockcomponent is a mock implementation of Interface.
type AdvancedMockcomponent struct {
	mock.Mock
}

// NewAdvancedMockcomponent creates a new AdvancedMockcomponent.
func NewAdvancedMockcomponent() *AdvancedMockcomponent {
	return &AdvancedMockcomponent{}
}
