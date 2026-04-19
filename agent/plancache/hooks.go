package plancache

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/internal/hookutil"
)

// Hooks provides optional callback functions invoked during plan cache
// operations. All fields are optional; nil hooks are skipped. Hooks are
// composable via ComposeHooks.
type Hooks struct {
	// OnCacheHit is called when a cached template matches the input.
	OnCacheHit func(ctx context.Context, tmpl *Template, score float64)

	// OnCacheMiss is called when no cached template matches the input.
	OnCacheMiss func(ctx context.Context, input string)

	// OnTemplateExtracted is called when a new template is extracted from
	// planner output and saved to the store.
	OnTemplateExtracted func(ctx context.Context, tmpl *Template)

	// OnTemplateEvicted is called when a template is evicted due to
	// exceeding the deviation threshold.
	OnTemplateEvicted func(ctx context.Context, tmpl *Template)
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnCacheHit: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, *Template, float64) {
			return hk.OnCacheHit
		}),
		OnCacheMiss: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, string) {
			return hk.OnCacheMiss
		}),
		OnTemplateExtracted: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, *Template) {
			return hk.OnTemplateExtracted
		}),
		OnTemplateEvicted: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, *Template) {
			return hk.OnTemplateEvicted
		}),
	}
}
