package server

import (
	"github.com/stretchr/testify/mock"
)

// ServerMockcomponent is a mock implementation of Interface.
type ServerMockcomponent struct {
	mock.Mock
}

// NewServerMockcomponent creates a new ServerMockcomponent.
func NewServerMockcomponent() *ServerMockcomponent {
	return &ServerMockcomponent{}
}
