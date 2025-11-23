package core

import (
	"github.com/stretchr/testify/mock"
)

// IntegrationMockcomponent is a mock implementation of Interface
type IntegrationMockcomponent struct {
	mock.Mock
}

// NewIntegrationMockcomponent creates a new IntegrationMockcomponent
func NewIntegrationMockcomponent() *IntegrationMockcomponent {
	return &IntegrationMockcomponent{}
}
