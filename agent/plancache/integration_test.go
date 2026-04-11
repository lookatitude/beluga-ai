package plancache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapPlanner(t *testing.T) {
	inner := &mockPlanner{}

	cp, err := WrapPlanner(inner, WithMinScore(0.7), WithMaxTemplates(50))
	require.NoError(t, err)
	require.NotNil(t, cp)

	assert.Equal(t, 0.7, cp.opts.minScore)
	assert.Equal(t, 50, cp.opts.maxTemplates)

	// Verify it works.
	actions, err := cp.Plan(context.Background(), newTestState("search database"))
	require.NoError(t, err)
	assert.NotEmpty(t, actions)
}

func TestWrapPlanner_DefaultOptions(t *testing.T) {
	inner := &mockPlanner{}

	cp, err := WrapPlanner(inner)
	require.NoError(t, err)
	require.NotNil(t, cp)

	assert.Equal(t, 0.6, cp.opts.minScore)
	assert.Equal(t, 100, cp.opts.maxTemplates)
	assert.Equal(t, 0.5, cp.opts.evictionThreshold)
}
