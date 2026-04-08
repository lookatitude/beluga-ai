package contract

import (
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	names := List()
	assert.Contains(t, names, "text-in-text-out")
	assert.Contains(t, names, "json-object")
	// Verify sorted.
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i])
	}
}

func TestNewFromTemplate_TextInTextOut(t *testing.T) {
	c, err := NewFromTemplate("text-in-text-out", "my-text-contract")
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, "my-text-contract", c.Name)
	assert.Equal(t, map[string]any{"type": "string"}, c.InputSchema)
	assert.Equal(t, map[string]any{"type": "string"}, c.OutputSchema)
	assert.NotEmpty(t, c.Description)
}

func TestNewFromTemplate_JSONObject(t *testing.T) {
	c, err := NewFromTemplate("json-object", "my-json-contract",
		WithVersion("2.0.0"),
	)
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, "my-json-contract", c.Name)
	assert.Equal(t, map[string]any{"type": "object"}, c.InputSchema)
	assert.Equal(t, map[string]any{"type": "object"}, c.OutputSchema)
	assert.Equal(t, "2.0.0", c.Version)
}

func TestNewFromTemplate_Unknown(t *testing.T) {
	c, err := NewFromTemplate("nonexistent", "test")
	assert.Error(t, err)
	assert.Nil(t, c)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRegister_Custom(t *testing.T) {
	Register("test-custom-template", func(name string, opts ...Option) *schema.Contract {
		c := &schema.Contract{
			Name:         name,
			Description:  "Custom template",
			InputSchema:  map[string]any{"type": "string"},
			OutputSchema: map[string]any{"type": "boolean"},
		}
		for _, opt := range opts {
			opt(c)
		}
		return c
	})

	c, err := NewFromTemplate("test-custom-template", "my-custom")
	require.NoError(t, err)
	assert.Equal(t, "my-custom", c.Name)
	assert.Equal(t, map[string]any{"type": "boolean"}, c.OutputSchema)

	assert.Contains(t, List(), "test-custom-template")
}
