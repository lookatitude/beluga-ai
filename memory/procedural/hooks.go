package procedural

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/schema"
)

// Hooks provides optional callback functions for procedural memory operations.
// All fields are optional — nil hooks are skipped. Hooks can be composed
// via ComposeHooks.
type Hooks struct {
	// OnSkillSaved is called after a skill has been successfully saved.
	OnSkillSaved func(ctx context.Context, skill *schema.Skill)

	// OnSkillUpdated is called after a skill has been successfully updated,
	// with both the old and new versions.
	OnSkillUpdated func(ctx context.Context, old, updated *schema.Skill)

	// OnSkillRetrieved is called after skills have been retrieved via search,
	// with the query and the results.
	OnSkillRetrieved func(ctx context.Context, query string, skills []*schema.Skill)
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnSkillSaved: func(ctx context.Context, skill *schema.Skill) {
			for _, h := range hooks {
				if h.OnSkillSaved != nil {
					h.OnSkillSaved(ctx, skill)
				}
			}
		},
		OnSkillUpdated: func(ctx context.Context, old, updated *schema.Skill) {
			for _, h := range hooks {
				if h.OnSkillUpdated != nil {
					h.OnSkillUpdated(ctx, old, updated)
				}
			}
		},
		OnSkillRetrieved: func(ctx context.Context, query string, skills []*schema.Skill) {
			for _, h := range hooks {
				if h.OnSkillRetrieved != nil {
					h.OnSkillRetrieved(ctx, query, skills)
				}
			}
		},
	}
}
