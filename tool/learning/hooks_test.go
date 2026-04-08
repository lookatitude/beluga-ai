package learning

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComposeHooks(t *testing.T) {
	t.Run("composes OnToolCreated", func(t *testing.T) {
		var calls []string
		h1 := Hooks{OnToolCreated: func(name, code string) { calls = append(calls, "h1:"+name) }}
		h2 := Hooks{OnToolCreated: func(name, code string) { calls = append(calls, "h2:"+name) }}

		composed := ComposeHooks(h1, h2)
		composed.OnToolCreated("test", "code")

		assert.Equal(t, []string{"h1:test", "h2:test"}, calls)
	})

	t.Run("composes OnToolTested", func(t *testing.T) {
		var calls []bool
		h1 := Hooks{OnToolTested: func(name string, passed bool) { calls = append(calls, passed) }}
		h2 := Hooks{OnToolTested: func(name string, passed bool) { calls = append(calls, !passed) }}

		composed := ComposeHooks(h1, h2)
		composed.OnToolTested("t", true)

		assert.Equal(t, []bool{true, false}, calls)
	})

	t.Run("composes OnVersionActivated", func(t *testing.T) {
		var versions []int
		h1 := Hooks{OnVersionActivated: func(name string, v int) { versions = append(versions, v) }}
		h2 := Hooks{OnVersionActivated: func(name string, v int) { versions = append(versions, v*10) }}

		composed := ComposeHooks(h1, h2)
		composed.OnVersionActivated("t", 3)

		assert.Equal(t, []int{3, 30}, versions)
	})

	t.Run("nil hooks skipped", func(t *testing.T) {
		var calls int
		h1 := Hooks{} // All nil.
		h2 := Hooks{OnToolCreated: func(name, code string) { calls++ }}

		composed := ComposeHooks(h1, h2)
		composed.OnToolCreated("test", "code")

		assert.Equal(t, 1, calls)
	})

	t.Run("empty compose", func(t *testing.T) {
		composed := ComposeHooks()
		// Should not panic.
		composed.OnToolCreated("test", "code")
		composed.OnToolTested("test", true)
		composed.OnVersionActivated("test", 1)
	})
}
