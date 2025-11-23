package server

import (
	"github.com/stretchr/testify/mock"
)

// ObservabilityMockcomponent is a mock implementation of Interface
type ObservabilityMockcomponent struct {
	mock.Mock
}

// NewObservabilityMockcomponent creates a new ObservabilityMockcomponent
func NewObservabilityMockcomponent() *ObservabilityMockcomponent {
	return &ObservabilityMockcomponent{}
}
