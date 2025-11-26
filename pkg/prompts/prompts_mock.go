package prompts

import (
	"github.com/stretchr/testify/mock"
)

// PromptsMockcomponent is a mock implementation of Interface.
type PromptsMockcomponent struct {
	mock.Mock
}

// NewPromptsMockcomponent creates a new PromptsMockcomponent.
func NewPromptsMockcomponent() *PromptsMockcomponent {
	return &PromptsMockcomponent{}
}
