package server

import (
	"github.com/stretchr/testify/mock"
)

// MCPServerMockcomponent is a mock implementation of Interface.
type MCPServerMockcomponent struct {
	mock.Mock
}

// NewMCPServerMockcomponent creates a new MCPServerMockcomponent.
func NewMCPServerMockcomponent() *MCPServerMockcomponent {
	return &MCPServerMockcomponent{}
}
