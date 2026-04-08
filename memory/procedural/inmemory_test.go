package procedural

import (
	"context"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_Save(t *testing.T) {
	ctx := context.Background()

	t.Run("saves skill", func(t *testing.T) {
		store := NewInMemoryStore()
		skill := &schema.Skill{ID: "sk-1", Name: "deploy", Confidence: 0.9}

		err := store.Save(ctx, skill)
		require.NoError(t, err)

		all := store.All()
		assert.Len(t, all, 1)
		assert.Equal(t, "sk-1", all[0].ID)
	})

	t.Run("overwrites existing skill", func(t *testing.T) {
		store := NewInMemoryStore()
		skill := &schema.Skill{ID: "sk-1", Name: "v1"}
		err := store.Save(ctx, skill)
		require.NoError(t, err)

		skill2 := &schema.Skill{ID: "sk-1", Name: "v2"}
		err = store.Save(ctx, skill2)
		require.NoError(t, err)

		all := store.All()
		assert.Len(t, all, 1)
		assert.Equal(t, "v2", all[0].Name)
	})

	t.Run("empty ID errors", func(t *testing.T) {
		store := NewInMemoryStore()
		err := store.Save(ctx, &schema.Skill{Name: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "skill ID is required")
	})

	t.Run("stores copy not reference", func(t *testing.T) {
		store := NewInMemoryStore()
		skill := &schema.Skill{ID: "sk-1", Name: "original"}
		err := store.Save(ctx, skill)
		require.NoError(t, err)

		skill.Name = "mutated"
		got, err := store.Get(ctx, "sk-1")
		require.NoError(t, err)
		assert.Equal(t, "original", got.Name)
	})
}

func TestInMemoryStore_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("gets existing skill", func(t *testing.T) {
		store := NewInMemoryStore()
		err := store.Save(ctx, &schema.Skill{ID: "sk-1", Name: "deploy"})
		require.NoError(t, err)

		sk, err := store.Get(ctx, "sk-1")
		require.NoError(t, err)
		assert.Equal(t, "deploy", sk.Name)
	})

	t.Run("returns nil for missing", func(t *testing.T) {
		store := NewInMemoryStore()
		sk, err := store.Get(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, sk)
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		store := NewInMemoryStore()
		err := store.Save(ctx, &schema.Skill{ID: "sk-1", Name: "original"})
		require.NoError(t, err)

		got, err := store.Get(ctx, "sk-1")
		require.NoError(t, err)
		got.Name = "mutated"

		got2, err := store.Get(ctx, "sk-1")
		require.NoError(t, err)
		assert.Equal(t, "original", got2.Name)
	})
}

func TestInMemoryStore_Search(t *testing.T) {
	ctx := context.Background()

	store := NewInMemoryStore()
	_ = store.Save(ctx, &schema.Skill{
		ID: "sk-1", Name: "deploy-service",
		Description: "Deploy a microservice to Kubernetes",
		Triggers:    []string{"deploy", "release"},
		Tags:        []string{"devops"},
	})
	_ = store.Save(ctx, &schema.Skill{
		ID: "sk-2", Name: "run-tests",
		Description: "Execute test suite",
		Triggers:    []string{"test", "validate"},
		Tags:        []string{"ci"},
	})

	tests := []struct {
		name      string
		query     string
		k         int
		wantCount int
	}{
		{name: "match by name", query: "deploy", k: 10, wantCount: 1},
		{name: "match by description", query: "kubernetes", k: 10, wantCount: 1},
		{name: "match by trigger", query: "release", k: 10, wantCount: 1},
		{name: "match by tag", query: "devops", k: 10, wantCount: 1},
		{name: "no match", query: "database", k: 10, wantCount: 0},
		{name: "case insensitive", query: "DEPLOY", k: 10, wantCount: 1},
		{name: "default k", query: "deploy", k: 0, wantCount: 1},
		{name: "limit k", query: "e", k: 1, wantCount: 1}, // both match "e"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Search(ctx, tt.query, tt.k)
			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
		})
	}
}

func TestInMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes existing skill", func(t *testing.T) {
		store := NewInMemoryStore()
		_ = store.Save(ctx, &schema.Skill{ID: "sk-1", Name: "test"})

		err := store.Delete(ctx, "sk-1")
		require.NoError(t, err)
		assert.Empty(t, store.All())
	})

	t.Run("no error for missing skill", func(t *testing.T) {
		store := NewInMemoryStore()
		err := store.Delete(ctx, "nonexistent")
		require.NoError(t, err)
	})
}

func TestInMemoryStore_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "sk-" + string(rune('a'+i%10))
			_ = store.Save(ctx, &schema.Skill{ID: id, Name: "skill"})
			_, _ = store.Get(ctx, id)
			_, _ = store.Search(ctx, "skill", 5)
			_ = store.Delete(ctx, id)
		}(i)
	}
	wg.Wait()
}
