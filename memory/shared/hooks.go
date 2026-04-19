package shared

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/internal/hookutil"
)

// Hooks provides optional callback functions for SharedMemory operations.
// All fields are optional; nil hooks are skipped. Multiple Hooks values
// can be merged with ComposeHooks.
type Hooks struct {
	// OnWrite is called after a successful write. The fragment reflects its
	// new state including the updated version.
	OnWrite func(ctx context.Context, frag *Fragment)

	// OnRead is called after a successful read.
	OnRead func(ctx context.Context, frag *Fragment)

	// OnGrant is called after permissions are granted on a fragment.
	OnGrant func(ctx context.Context, key string, agentID string, perm Permission)

	// OnDenied is called when an access check fails.
	OnDenied func(ctx context.Context, key string, agentID string, perm Permission)
}

// ComposeHooks merges multiple Hooks into a single Hooks value. Callbacks
// are called in the order the hooks were provided.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnWrite: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, *Fragment) {
			return hk.OnWrite
		}),
		OnRead: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, *Fragment) {
			return hk.OnRead
		}),
		OnGrant: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, string, string, Permission) {
			return hk.OnGrant
		}),
		OnDenied: hookutil.ComposeVoid3(h, func(hk Hooks) func(context.Context, string, string, Permission) {
			return hk.OnDenied
		}),
	}
}
