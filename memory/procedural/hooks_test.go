package procedural

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
)

func TestComposeHooks(t *testing.T) {
	t.Run("OnSkillSaved chains in order", func(t *testing.T) {
		var order []int
		h1 := Hooks{OnSkillSaved: func(_ context.Context, _ *schema.Skill) { order = append(order, 1) }}
		h2 := Hooks{OnSkillSaved: func(_ context.Context, _ *schema.Skill) { order = append(order, 2) }}

		composed := ComposeHooks(h1, h2)
		composed.OnSkillSaved(context.Background(), &schema.Skill{})
		assert.Equal(t, []int{1, 2}, order)
	})

	t.Run("OnSkillUpdated chains in order", func(t *testing.T) {
		var order []int
		h1 := Hooks{OnSkillUpdated: func(_ context.Context, _, _ *schema.Skill) { order = append(order, 1) }}
		h2 := Hooks{OnSkillUpdated: func(_ context.Context, _, _ *schema.Skill) { order = append(order, 2) }}

		composed := ComposeHooks(h1, h2)
		composed.OnSkillUpdated(context.Background(), &schema.Skill{}, &schema.Skill{})
		assert.Equal(t, []int{1, 2}, order)
	})

	t.Run("OnSkillRetrieved chains in order", func(t *testing.T) {
		var order []int
		h1 := Hooks{OnSkillRetrieved: func(_ context.Context, _ string, _ []*schema.Skill) { order = append(order, 1) }}
		h2 := Hooks{OnSkillRetrieved: func(_ context.Context, _ string, _ []*schema.Skill) { order = append(order, 2) }}

		composed := ComposeHooks(h1, h2)
		composed.OnSkillRetrieved(context.Background(), "query", nil)
		assert.Equal(t, []int{1, 2}, order)
	})

	t.Run("skips nil hooks", func(t *testing.T) {
		var called bool
		h1 := Hooks{} // all nil
		h2 := Hooks{OnSkillSaved: func(_ context.Context, _ *schema.Skill) { called = true }}

		composed := ComposeHooks(h1, h2)
		composed.OnSkillSaved(context.Background(), &schema.Skill{})
		assert.True(t, called)
	})

	t.Run("empty compose returns safe hooks", func(t *testing.T) {
		composed := ComposeHooks()
		// Should not panic.
		composed.OnSkillSaved(context.Background(), &schema.Skill{})
		composed.OnSkillUpdated(context.Background(), &schema.Skill{}, &schema.Skill{})
		composed.OnSkillRetrieved(context.Background(), "", nil)
	})
}
