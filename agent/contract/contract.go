package contract

import (
	"github.com/lookatitude/beluga-ai/v2/internal/jsonutil"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// New creates a new Contract with the given name and options.
func New(name string, opts ...Option) *schema.Contract {
	c := &schema.Contract{Name: name}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFor creates a new Contract with input and output schemas derived from
// the Go types I and O using reflection-based schema generation. Additional
// options can override or augment the derived schemas.
func NewFor[I, O any](name string, opts ...Option) *schema.Contract {
	var zeroI I
	var zeroO O
	c := &schema.Contract{
		Name:         name,
		InputSchema:  jsonutil.GenerateSchema(zeroI),
		OutputSchema: jsonutil.GenerateSchema(zeroO),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
