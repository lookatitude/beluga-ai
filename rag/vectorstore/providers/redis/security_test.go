package redis

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FI-03: tag values must be escaped and keys validated.
func TestBuildFilterExpr(t *testing.T) {
	tests := []struct {
		name   string
		filter map[string]any
		want   string
	}{
		{"empty", nil, "*"},
		{"plain", map[string]any{"cat": "A_1"}, "@cat:{A_1}"},
		{"breakout", map[string]any{"tenant": "a} | @tenant:{b"}, `@tenant:{a\}\ \|\ \@tenant\:\{b}`},
		{"sorted multi", map[string]any{"b": 2, "a": "x y"}, `@a:{x\ y} @b:{2}`},
		{"unicode", map[string]any{"c": "olá-mundo"}, `@c:{olá\-mundo}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildFilterExpr(tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFilterExpr_InvalidKey(t *testing.T) {
	_, err := buildFilterExpr(map[string]any{"a:{x} @b": "y"})
	require.Error(t, err)
}

func TestStore_Search_InvalidFilter(t *testing.T) {
	_, store := newTestStore(t)
	_, err := store.Search(context.Background(), []float32{1}, 1, vectorstore.WithFilter(map[string]any{"a b": "x"}))
	require.Error(t, err)
}
