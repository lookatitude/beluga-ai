package associative

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
	"github.com/lookatitude/beluga-ai/schema"
)

// Hooks provides optional callback functions for associative memory lifecycle
// events. All fields are optional; nil hooks are skipped. Compose multiple
// Hooks values with ComposeHooks.
type Hooks struct {
	// OnNoteCreated is called after a note has been fully created (enriched,
	// embedded, and stored). The note parameter includes all enrichment data.
	OnNoteCreated func(ctx context.Context, note *schema.Note)

	// OnNoteLinked is called after bidirectional links have been established
	// between the new note and its neighbors. linkedIDs contains the IDs of
	// all notes that were linked.
	OnNoteLinked func(ctx context.Context, note *schema.Note, linkedIDs []string)

	// OnNoteRefined is called after retroactive refinement completes for
	// neighbor notes. refinedIDs contains the IDs of neighbors that were
	// successfully updated.
	OnNoteRefined func(ctx context.Context, note *schema.Note, refinedIDs []string)
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnNoteCreated: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, *schema.Note) {
			return hk.OnNoteCreated
		}),
		OnNoteLinked: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, *schema.Note, []string) {
			return hk.OnNoteLinked
		}),
		OnNoteRefined: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, *schema.Note, []string) {
			return hk.OnNoteRefined
		}),
	}
}
