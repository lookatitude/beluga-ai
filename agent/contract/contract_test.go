package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	c := New("test-contract",
		WithDescription("A test contract"),
		WithInputSchema(map[string]any{"type": "string"}),
		WithOutputSchema(map[string]any{"type": "object"}),
		WithStrict(true),
		WithVersion("1.0.0"),
	)

	assert.Equal(t, "test-contract", c.Name)
	assert.Equal(t, "A test contract", c.Description)
	assert.Equal(t, map[string]any{"type": "string"}, c.InputSchema)
	assert.Equal(t, map[string]any{"type": "object"}, c.OutputSchema)
	assert.True(t, c.Strict)
	assert.Equal(t, "1.0.0", c.Version)
}

func TestNew_Defaults(t *testing.T) {
	c := New("minimal")

	assert.Equal(t, "minimal", c.Name)
	assert.Empty(t, c.Description)
	assert.Nil(t, c.InputSchema)
	assert.Nil(t, c.OutputSchema)
	assert.False(t, c.Strict)
	assert.Empty(t, c.Version)
}

type testInput struct {
	Query string `json:"query" required:"true"`
	Limit int    `json:"limit"`
}

type testOutput struct {
	Results []string `json:"results"`
	Total   int      `json:"total"`
}

func TestNewFor(t *testing.T) {
	c := NewFor[testInput, testOutput]("typed-contract",
		WithDescription("A typed contract"),
	)

	require.NotNil(t, c)
	assert.Equal(t, "typed-contract", c.Name)
	assert.Equal(t, "A typed contract", c.Description)

	// Input schema should be derived from testInput.
	require.NotNil(t, c.InputSchema)
	assert.Equal(t, "object", c.InputSchema["type"])
	props, ok := c.InputSchema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, props, "query")
	assert.Contains(t, props, "limit")

	// Output schema should be derived from testOutput.
	require.NotNil(t, c.OutputSchema)
	assert.Equal(t, "object", c.OutputSchema["type"])
	outProps, ok := c.OutputSchema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, outProps, "results")
	assert.Contains(t, outProps, "total")
}

func TestNewFor_OptionsOverride(t *testing.T) {
	customOutput := map[string]any{"type": "string"}
	c := NewFor[testInput, testOutput]("override",
		WithOutputSchema(customOutput),
	)

	assert.Equal(t, customOutput, c.OutputSchema)
	// Input schema should still be derived.
	assert.NotNil(t, c.InputSchema)
}
