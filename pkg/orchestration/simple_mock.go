package orchestration

import (
	"github.com/stretchr/testify/mock"
)

// SimpleMockcomponent is a mock implementation of Interface.
type SimpleMockcomponent struct {
	mock.Mock
}

// NewSimpleMockcomponent creates a new SimpleMockcomponent.
func NewSimpleMockcomponent() *SimpleMockcomponent {
	return &SimpleMockcomponent{}
}
