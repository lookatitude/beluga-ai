package learning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeDynTool(name string, version int) *DynamicTool {
	return NewDynamicTool(
		name,
		"description for "+name,
		map[string]any{"type": "object"},
		"code v"+string(rune('0'+version)),
		&NoopExecutor{Response: "v" + string(rune('0'+version))},
		WithVersion(version),
	)
}

func TestVersionedRegistry_Upsert(t *testing.T) {
	vr := NewVersionedRegistry(nil)

	t.Run("first upsert creates version 1", func(t *testing.T) {
		tool := makeDynTool("calc", 1)
		v, err := vr.Upsert(tool)
		require.NoError(t, err)
		assert.Equal(t, 1, v)
	})

	t.Run("second upsert creates version 2", func(t *testing.T) {
		tool := makeDynTool("calc", 2)
		v, err := vr.Upsert(tool)
		require.NoError(t, err)
		assert.Equal(t, 2, v)
	})

	t.Run("get returns latest version", func(t *testing.T) {
		got, err := vr.Get("calc")
		require.NoError(t, err)
		assert.Equal(t, "calc", got.Name())
	})

	t.Run("list returns tool names", func(t *testing.T) {
		names := vr.List()
		assert.Equal(t, []string{"calc"}, names)
	})
}

func TestVersionedRegistry_Activate(t *testing.T) {
	vr := NewVersionedRegistry(nil)

	// Create 3 versions.
	for i := 1; i <= 3; i++ {
		_, err := vr.Upsert(makeDynTool("search", i))
		require.NoError(t, err)
	}

	t.Run("activate version 1", func(t *testing.T) {
		err := vr.Activate("search", 1)
		require.NoError(t, err)

		history, err := vr.History("search")
		require.NoError(t, err)
		assert.True(t, history[0].Active)
		assert.False(t, history[1].Active)
		assert.False(t, history[2].Active)
	})

	t.Run("activate nonexistent tool", func(t *testing.T) {
		err := vr.Activate("nonexistent", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("activate nonexistent version", func(t *testing.T) {
		err := vr.Activate("search", 99)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version 99 not found")
	})
}

func TestVersionedRegistry_Rollback(t *testing.T) {
	vr := NewVersionedRegistry(nil)

	t.Run("rollback with no previous version", func(t *testing.T) {
		_, err := vr.Upsert(makeDynTool("single", 1))
		require.NoError(t, err)

		_, err = vr.Rollback("single")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no previous version")
	})

	t.Run("rollback to previous version", func(t *testing.T) {
		_, err := vr.Upsert(makeDynTool("multi", 1))
		require.NoError(t, err)
		_, err = vr.Upsert(makeDynTool("multi", 2))
		require.NoError(t, err)

		v, err := vr.Rollback("multi")
		require.NoError(t, err)
		assert.Equal(t, 1, v)

		history, err := vr.History("multi")
		require.NoError(t, err)
		assert.True(t, history[0].Active)
		assert.False(t, history[1].Active)
	})

	t.Run("rollback nonexistent tool", func(t *testing.T) {
		_, err := vr.Rollback("ghost")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestVersionedRegistry_History(t *testing.T) {
	vr := NewVersionedRegistry(nil)

	t.Run("history of nonexistent tool", func(t *testing.T) {
		_, err := vr.History("nope")
		assert.Error(t, err)
	})

	t.Run("history returns all versions", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			_, err := vr.Upsert(makeDynTool("hist", i))
			require.NoError(t, err)
		}

		records, err := vr.History("hist")
		require.NoError(t, err)
		require.Len(t, records, 3)

		assert.Equal(t, 1, records[0].Version)
		assert.Equal(t, 2, records[1].Version)
		assert.Equal(t, 3, records[2].Version)

		// Only last should be active.
		assert.False(t, records[0].Active)
		assert.False(t, records[1].Active)
		assert.True(t, records[2].Active)
	})

	t.Run("history is a copy", func(t *testing.T) {
		records, err := vr.History("hist")
		require.NoError(t, err)
		records[0].Active = true // Mutate the copy.

		original, err := vr.History("hist")
		require.NoError(t, err)
		assert.False(t, original[0].Active) // Original unchanged.
	})
}

func TestVersionedRegistry_All(t *testing.T) {
	vr := NewVersionedRegistry(nil)

	_, err := vr.Upsert(makeDynTool("a", 1))
	require.NoError(t, err)
	_, err = vr.Upsert(makeDynTool("b", 1))
	require.NoError(t, err)

	all := vr.All()
	assert.Len(t, all, 2)
}

func TestVersionedRegistry_Hooks(t *testing.T) {
	var activated []string

	hooks := Hooks{
		OnVersionActivated: func(name string, version int) {
			activated = append(activated, name)
		},
	}

	vr := NewVersionedRegistry(nil, WithRegistryHooks(hooks))
	_, err := vr.Upsert(makeDynTool("hooked", 1))
	require.NoError(t, err)

	assert.Equal(t, []string{"hooked"}, activated)

	// Activate fires hook too.
	_, err = vr.Upsert(makeDynTool("hooked", 2))
	require.NoError(t, err)
	err = vr.Activate("hooked", 1)
	require.NoError(t, err)

	assert.Equal(t, []string{"hooked", "hooked", "hooked"}, activated)
}
