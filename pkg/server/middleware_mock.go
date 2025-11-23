package server

import (
	"github.com/stretchr/testify/mock"
)

// MiddlewareMockcomponent is a mock implementation of Interface
type MiddlewareMockcomponent struct {
	mock.Mock
}

// NewMiddlewareMockcomponent creates a new MiddlewareMockcomponent
func NewMiddlewareMockcomponent() *MiddlewareMockcomponent {
	return &MiddlewareMockcomponent{}
}
