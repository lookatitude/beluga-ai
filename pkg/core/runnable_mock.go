package core

import (
	"github.com/stretchr/testify/mock"
)

// RunnableMockcomponent is a mock implementation of Interface
type RunnableMockcomponent struct {
	mock.Mock
}

// NewRunnableMockcomponent creates a new RunnableMockcomponent
func NewRunnableMockcomponent() *RunnableMockcomponent {
	return &RunnableMockcomponent{}
}
