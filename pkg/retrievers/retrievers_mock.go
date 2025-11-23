package retrievers

import (
	"github.com/stretchr/testify/mock"
)

// RetrieversMockcomponent is a mock implementation of Interface
type RetrieversMockcomponent struct {
	mock.Mock
}

// NewRetrieversMockcomponent creates a new RetrieversMockcomponent
func NewRetrieversMockcomponent() *RetrieversMockcomponent {
	return &RetrieversMockcomponent{}
}
